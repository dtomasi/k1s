package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Event represents a single event in the system and is used for observability.
// It directly uses the standard Kubernetes corev1.Event for full compatibility.
type Event = corev1.Event

// EventList represents a list of Event objects.
type EventList = corev1.EventList

var (
	// EventGVK is the GroupVersionKind for Event.
	EventGVK = schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Event",
	}

	// EventGVR is the GroupVersionResource for Event.
	EventGVR = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "events",
	}
)

// GetEventGVK returns the GroupVersionKind for Event.
func GetEventGVK() schema.GroupVersionKind {
	return EventGVK
}

// GetEventGVR returns the GroupVersionResource for Event.
func GetEventGVR() schema.GroupVersionResource {
	return EventGVR
}

// NewEvent creates a new Event with the given parameters.
func NewEvent(namespace, name, reason, message string, involvedObject corev1.ObjectReference, eventType string) *Event {
	now := metav1.Now()
	return &Event{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Event",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		InvolvedObject:      involvedObject,
		Reason:              reason,
		Message:             message,
		Type:                eventType,
		FirstTimestamp:      now,
		LastTimestamp:       now,
		Count:               1,
		ReportingController: "k1s",
		ReportingInstance:   "k1s-runtime",
	}
}

// NewNormalEvent creates a new Normal type Event.
func NewNormalEvent(namespace, name, reason, message string, involvedObject corev1.ObjectReference) *Event {
	return NewEvent(namespace, name, reason, message, involvedObject, corev1.EventTypeNormal)
}

// NewWarningEvent creates a new Warning type Event.
func NewWarningEvent(namespace, name, reason, message string, involvedObject corev1.ObjectReference) *Event {
	return NewEvent(namespace, name, reason, message, involvedObject, corev1.EventTypeWarning)
}

// IsEventNamespaceScoped returns true as Event is a namespace-scoped resource.
func IsEventNamespaceScoped() bool {
	return true
}

// GetEventShortNames returns short names for Event resource.
func GetEventShortNames() []string {
	return []string{"ev"}
}

// GetEventCategories returns categories for Event resource.
func GetEventCategories() []string {
	return []string{"all"}
}

// GetEventPrintColumns returns table columns for Event display.
func GetEventPrintColumns() []metav1.TableColumnDefinition {
	return []metav1.TableColumnDefinition{
		{
			Name:        "Last Seen",
			Type:        "string",
			Format:      "",
			Description: "Time when this Event was last observed",
			Priority:    0,
		},
		{
			Name:        "Type",
			Type:        "string",
			Format:      "",
			Description: "Type of this event (Normal, Warning)",
			Priority:    0,
		},
		{
			Name:        "Reason",
			Type:        "string",
			Format:      "",
			Description: "Short, machine-readable reason for this event",
			Priority:    0,
		},
		{
			Name:        "Object",
			Type:        "string",
			Format:      "",
			Description: "Object this event is about",
			Priority:    0,
		},
		{
			Name:        "Message",
			Type:        "string",
			Format:      "",
			Description: "Human-readable description of this event",
			Priority:    1,
		},
	}
}

// GetEventPrintColumnsWithNamespace returns table columns for Event display including namespace.
func GetEventPrintColumnsWithNamespace() []metav1.TableColumnDefinition {
	return []metav1.TableColumnDefinition{
		{
			Name:        "Namespace",
			Type:        "string",
			Format:      "",
			Description: "Namespace of the event",
			Priority:    0,
		},
		{
			Name:        "Last Seen",
			Type:        "string",
			Format:      "",
			Description: "Time when this Event was last observed",
			Priority:    0,
		},
		{
			Name:        "Type",
			Type:        "string",
			Format:      "",
			Description: "Type of this event (Normal, Warning)",
			Priority:    0,
		},
		{
			Name:        "Reason",
			Type:        "string",
			Format:      "",
			Description: "Short, machine-readable reason for this event",
			Priority:    0,
		},
		{
			Name:        "Object",
			Type:        "string",
			Format:      "",
			Description: "Object this event is about",
			Priority:    0,
		},
		{
			Name:        "Message",
			Type:        "string",
			Format:      "",
			Description: "Human-readable description of this event",
			Priority:    1,
		},
	}
}

// AddEventToScheme adds Event types to the given scheme.
func AddEventToScheme(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(schema.GroupVersion{Group: "", Version: "v1"},
		&Event{},
		&EventList{},
	)
	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Group: "", Version: "v1"})
	return nil
}
