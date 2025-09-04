package events

import (
	"fmt"
	"sync"
	"sync/atomic"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
)

// eventRecorder implements the EventRecorder interface
type eventRecorder struct {
	scheme         *runtime.Scheme
	source         corev1.EventSource
	broadcaster    EventBroadcaster
	eventNamespace string
	clock          Clock
	metrics        *EventMetrics
	mu             sync.RWMutex
}

// NewEventRecorder creates a new EventRecorder instance
func NewEventRecorder(broadcaster EventBroadcaster, options EventRecorderOptions) EventRecorder {
	if options.Clock == nil {
		options.Clock = RealClock{}
	}

	return &eventRecorder{
		scheme:         options.Scheme,
		source:         options.Source,
		broadcaster:    broadcaster,
		eventNamespace: options.EventNamespace,
		clock:          options.Clock,
		metrics:        &EventMetrics{},
	}
}

// Event constructs an event from the given information and records it
func (r *eventRecorder) Event(object runtime.Object, eventtype, reason, message string) {
	r.recordEvent(object, eventtype, reason, message, nil)
}

// Eventf is just like Event, but with Sprintf for the message field
func (r *eventRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	message := fmt.Sprintf(messageFmt, args...)
	r.recordEvent(object, eventtype, reason, message, nil)
}

// AnnotatedEventf is just like eventf, but with annotations attached
func (r *eventRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
	message := fmt.Sprintf(messageFmt, args...)
	r.recordEvent(object, eventtype, reason, message, annotations)
}

// recordEvent is the internal method that handles event creation and recording
func (r *eventRecorder) recordEvent(object runtime.Object, eventtype, reason, message string, annotations map[string]string) {
	// Create object reference
	objRef, err := CreateObjectReference(r.scheme, object)
	if err != nil {
		atomic.AddInt64(&r.metrics.EventsDropped, 1)
		return
	}

	// Determine event namespace
	eventNamespace := r.eventNamespace
	if eventNamespace == "" {
		eventNamespace = objRef.Namespace
	}
	if eventNamespace == "" {
		eventNamespace = metav1.NamespaceDefault
	}

	// Create the event
	event := r.createEvent(eventNamespace, objRef, eventtype, reason, message, annotations)

	// Send to broadcaster
	if r.broadcaster != nil {
		r.sendEventToBroadcaster(event)
	}

	atomic.AddInt64(&r.metrics.EventsRecorded, 1)
}

// createEvent creates a new Event object with the given parameters
func (r *eventRecorder) createEvent(namespace string, objRef corev1.ObjectReference, eventtype, reason, message string, annotations map[string]string) *corev1.Event {
	now := r.clock.Now()

	// Generate event name - using a combination of involved object and timestamp for uniqueness
	eventName := fmt.Sprintf("%s.%s", objRef.Name, generateEventSuffix())

	event := &corev1.Event{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Event",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        eventName,
			Namespace:   namespace,
			Annotations: annotations,
		},
		InvolvedObject:      objRef,
		Reason:              reason,
		Message:             message,
		Type:                eventtype,
		FirstTimestamp:      now,
		LastTimestamp:       now,
		Count:               1,
		Source:              r.source,
		ReportingController: r.source.Component,
		ReportingInstance:   r.source.Host,
	}

	return event
}

// sendEventToBroadcaster sends the event to the broadcaster for distribution
func (r *eventRecorder) sendEventToBroadcaster(event *corev1.Event) {
	// This is a simplified implementation - in a real system you might want
	// to use a channel or queue here for async processing
	if br, ok := r.broadcaster.(*eventBroadcaster); ok {
		br.recordEvent(event)
	}
}

// GetMetrics returns current metrics for this event recorder
func (r *eventRecorder) GetMetrics() EventMetrics {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return EventMetrics{
		EventsRecorded: atomic.LoadInt64(&r.metrics.EventsRecorded),
		EventsDropped:  atomic.LoadInt64(&r.metrics.EventsDropped),
		SinksActive:    atomic.LoadInt32(&r.metrics.SinksActive),
		WatchersActive: atomic.LoadInt32(&r.metrics.WatchersActive),
	}
}

// generateEventSuffix generates a unique suffix for event names
func generateEventSuffix() string {
	// Generate a short unique identifier based on UUID
	uid := uuid.NewUUID()
	// Take first 8 characters for brevity
	return string(uid)[:8]
}

// Helper functions for common event patterns

// CreateNormalEvent creates a normal event with the given parameters
func CreateNormalEvent(recorder EventRecorder, object runtime.Object, reason, message string) {
	recorder.Event(object, EventTypeNormal, reason, message)
}

// CreateWarningEvent creates a warning event with the given parameters
func CreateWarningEvent(recorder EventRecorder, object runtime.Object, reason, message string) {
	recorder.Event(object, EventTypeWarning, reason, message)
}

// CreateNormalEventf creates a normal event with formatted message
func CreateNormalEventf(recorder EventRecorder, object runtime.Object, reason, messageFmt string, args ...interface{}) {
	recorder.Eventf(object, EventTypeNormal, reason, messageFmt, args...)
}

// CreateWarningEventf creates a warning event with formatted message
func CreateWarningEventf(recorder EventRecorder, object runtime.Object, reason, messageFmt string, args ...interface{}) {
	recorder.Eventf(object, EventTypeWarning, reason, messageFmt, args...)
}

// Event creation patterns for common operations

// RecordSuccessfulCreate records a successful resource creation event
func RecordSuccessfulCreate(recorder EventRecorder, object runtime.Object) {
	CreateNormalEvent(recorder, object, ReasonSuccessfulCreate, "Successfully created resource")
}

// RecordSuccessfulUpdate records a successful resource update event
func RecordSuccessfulUpdate(recorder EventRecorder, object runtime.Object) {
	CreateNormalEvent(recorder, object, ReasonSuccessfulUpdate, "Successfully updated resource")
}

// RecordSuccessfulDelete records a successful resource deletion event
func RecordSuccessfulDelete(recorder EventRecorder, object runtime.Object) {
	CreateNormalEvent(recorder, object, ReasonSuccessfulDelete, "Successfully deleted resource")
}

// RecordFailedCreate records a failed resource creation event
func RecordFailedCreate(recorder EventRecorder, object runtime.Object, err error) {
	CreateWarningEventf(recorder, object, ReasonFailedCreate, "Failed to create resource: %v", err)
}

// RecordFailedUpdate records a failed resource update event
func RecordFailedUpdate(recorder EventRecorder, object runtime.Object, err error) {
	CreateWarningEventf(recorder, object, ReasonFailedUpdate, "Failed to update resource: %v", err)
}

// RecordFailedDelete records a failed resource deletion event
func RecordFailedDelete(recorder EventRecorder, object runtime.Object, err error) {
	CreateWarningEventf(recorder, object, ReasonFailedDelete, "Failed to delete resource: %v", err)
}

// RecordValidationError records a validation error event
func RecordValidationError(recorder EventRecorder, object runtime.Object, err error) {
	CreateWarningEventf(recorder, object, ReasonFailedValidation, "Validation failed: %v", err)
}
