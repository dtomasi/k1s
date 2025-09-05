package events

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

// EventRecorder provides a kubernetes-compatible interface for recording events.
// This interface matches the client-go EventRecorder interface for maximum compatibility.
type EventRecorder interface {
	// Event constructs an event from the given information and puts it in the queue for sending.
	// 'object' is the object this event is about. Event will make a reference-- or you may also
	// pass a reference to the object directly.
	// 'eventtype' of this event, and can be one of Normal, Warning. New types could be added in future
	// 'reason' is the reason this event is generated. 'reason' should be short and unique; it
	// should be in UpperCamelCase format (starting with a capital letter). "reason" will be used
	// to automate handling of events, so imagine people writing switch statements to handle them.
	// You want to make that easy.
	// 'message' is intended to be consumed by humans.
	Event(object runtime.Object, eventtype, reason, message string)

	// Eventf is just like Event, but with Sprintf for the message field.
	Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{})

	// AnnotatedEventf is just like eventf, but with annotations attached
	AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{})
}

// EventBroadcaster provides event broadcasting functionality similar to client-go.
// It manages multiple event sinks and distributes events to all registered sinks.
type EventBroadcaster interface {
	// StartRecordingToSink begins recording events to the given sink and returns
	// a watch.Interface that can be used to stop recording.
	StartRecordingToSink(sink EventSink) watch.Interface

	// StartEventWatcher begins watching events and calling the provided handler
	// function for each event. Returns a watch.Interface that can be used to stop watching.
	StartEventWatcher(eventHandler func(*corev1.Event)) watch.Interface

	// NewRecorder returns an EventRecorder that records to this broadcaster.
	NewRecorder(scheme *runtime.Scheme, source corev1.EventSource) EventRecorder

	// Shutdown gracefully shuts down the broadcaster and all its watchers.
	Shutdown()
}

// EventSink represents a destination for events. This interface allows
// events to be sent to different backends (storage, logging, etc.).
type EventSink interface {
	// Create creates a new event in the sink.
	Create(event *corev1.Event) (*corev1.Event, error)

	// Update updates an existing event in the sink (typically for count aggregation).
	Update(event *corev1.Event) (*corev1.Event, error)

	// Patch patches an existing event in the sink.
	Patch(event *corev1.Event, data []byte) (*corev1.Event, error)
}

// Event types and reasons that mirror Kubernetes standard events
const (
	// EventTypeNormal represents normal, informational events
	EventTypeNormal = corev1.EventTypeNormal

	// EventTypeWarning represents events that indicate problems or issues
	EventTypeWarning = corev1.EventTypeWarning
)

// Standard event reasons following Kubernetes conventions
const (
	// Normal event reasons
	ReasonSuccessfulCreate = "SuccessfulCreate"
	ReasonSuccessfulUpdate = "SuccessfulUpdate"
	ReasonSuccessfulDelete = "SuccessfulDelete"
	ReasonCreated          = "Created"
	ReasonUpdated          = "Updated"
	ReasonDeleted          = "Deleted"
	ReasonStarted          = "Started"
	ReasonStopped          = "Stopped"
	ReasonCompleted        = "Completed"

	// Warning event reasons
	ReasonFailed           = "Failed"
	ReasonFailedCreate     = "FailedCreate"
	ReasonFailedUpdate     = "FailedUpdate"
	ReasonFailedDelete     = "FailedDelete"
	ReasonFailedValidation = "FailedValidation"
	ReasonUnhealthy        = "Unhealthy"
	ReasonError            = "Error"
	ReasonTimeout          = "Timeout"
)

// EventSource creates a standard event source for k1s components
func NewEventSource(component string) corev1.EventSource {
	return corev1.EventSource{
		Component: component,
		Host:      "k1s-runtime",
	}
}

// CreateObjectReference creates an ObjectReference from a runtime.Object
func CreateObjectReference(scheme *runtime.Scheme, obj runtime.Object) (corev1.ObjectReference, error) {
	if obj == nil {
		return corev1.ObjectReference{}, fmt.Errorf("cannot create reference for nil object")
	}

	// Get the object's metadata
	metaObj, ok := obj.(metav1.Object)
	if !ok {
		return corev1.ObjectReference{}, fmt.Errorf("object does not implement metav1.Object")
	}

	// Get the object's GVK
	gvks, _, err := scheme.ObjectKinds(obj)
	if err != nil {
		return corev1.ObjectReference{}, fmt.Errorf("failed to get object kind: %w", err)
	}
	if len(gvks) == 0 {
		return corev1.ObjectReference{}, fmt.Errorf("no GroupVersionKind found for object")
	}

	gvk := gvks[0]
	return corev1.ObjectReference{
		Kind:            gvk.Kind,
		Namespace:       metaObj.GetNamespace(),
		Name:            metaObj.GetName(),
		UID:             metaObj.GetUID(),
		APIVersion:      gvk.GroupVersion().String(),
		ResourceVersion: metaObj.GetResourceVersion(),
	}, nil
}

// EventMetrics provides metrics about event recording and broadcasting
type EventMetrics struct {
	// EventsRecorded counts the total number of events recorded
	EventsRecorded int64

	// EventsDropped counts the number of events that were dropped due to errors
	EventsDropped int64

	// SinksActive counts the number of active event sinks
	SinksActive int32

	// WatchersActive counts the number of active event watchers
	WatchersActive int32
}

// EventRecorderOptions provides configuration options for creating an EventRecorder
type EventRecorderOptions struct {
	// Scheme is the runtime scheme used for object kind resolution
	Scheme *runtime.Scheme

	// Source identifies the component recording events
	Source corev1.EventSource

	// EventNamespace is the namespace where events will be created.
	// If empty, events will be created in the same namespace as the object.
	EventNamespace string

	// Clock allows injection of a custom clock for testing
	Clock Clock
}

// Clock provides time-related functionality that can be mocked for testing
type Clock interface {
	Now() metav1.Time
}

// RealClock implements Clock using real time
type RealClock struct{}

// Now returns the current time
func (RealClock) Now() metav1.Time {
	return metav1.Now()
}

// EventBroadcasterOptions provides configuration options for creating an EventBroadcaster
type EventBroadcasterOptions struct {
	// QueueSize is the size of the internal event queue
	QueueSize int

	// MetricsEnabled enables metrics collection
	MetricsEnabled bool

	// Clock allows injection of a custom clock for testing
	Clock Clock
}

// Default values for event system configuration
const (
	DefaultEventQueueSize = 1000
	DefaultEventNamespace = ""
	DefaultComponent      = "k1s-runtime"
)
