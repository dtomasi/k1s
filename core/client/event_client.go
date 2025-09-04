package client

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/dtomasi/k1s/core/events"
)

// EventAwareClient extends the Client interface with event recording capabilities
type EventAwareClient interface {
	Client

	// GetEventRecorder returns the event recorder for this client
	GetEventRecorder() events.EventRecorder

	// SetEventRecorder sets the event recorder for this client
	SetEventRecorder(recorder events.EventRecorder)
}

// eventAwareClient wraps a standard client with event recording functionality
type eventAwareClient struct {
	Client
	eventRecorder events.EventRecorder
	enableEvents  bool
}

// EventAwareClientOptions contains options for creating an event-aware client
type EventAwareClientOptions struct {
	// Client is the underlying client to wrap
	Client Client

	// EventRecorder is the event recorder to use (optional)
	EventRecorder events.EventRecorder

	// EnableEvents controls whether events are recorded automatically
	EnableEvents bool
}

// NewEventAwareClient creates a new event-aware client that wraps an existing client
func NewEventAwareClient(options EventAwareClientOptions) EventAwareClient {
	return &eventAwareClient{
		Client:        options.Client,
		eventRecorder: options.EventRecorder,
		enableEvents:  options.EnableEvents && options.EventRecorder != nil,
	}
}

// GetEventRecorder returns the event recorder for this client
func (c *eventAwareClient) GetEventRecorder() events.EventRecorder {
	return c.eventRecorder
}

// SetEventRecorder sets the event recorder for this client
func (c *eventAwareClient) SetEventRecorder(recorder events.EventRecorder) {
	c.eventRecorder = recorder
	c.enableEvents = recorder != nil
}

// Create saves the object obj in the k1s storage and records appropriate events
func (c *eventAwareClient) Create(ctx context.Context, obj Object, opts ...CreateOption) error {
	// Record a creation attempt event if events are enabled
	if c.enableEvents && c.eventRecorder != nil {
		events.RecordSuccessfulCreate(c.eventRecorder, obj)
	}

	// Perform the actual create operation
	err := c.Client.Create(ctx, obj, opts...)

	if err != nil {
		// Record failure event if events are enabled
		if c.enableEvents && c.eventRecorder != nil {
			events.RecordFailedCreate(c.eventRecorder, obj, err)
		}
		return err
	}

	// The success event was already recorded above
	return nil
}

// Update updates the given obj in the k1s storage and records appropriate events
func (c *eventAwareClient) Update(ctx context.Context, obj Object, opts ...UpdateOption) error {
	// Perform the actual update operation
	err := c.Client.Update(ctx, obj, opts...)

	if err != nil {
		// Record failure event if events are enabled
		if c.enableEvents && c.eventRecorder != nil {
			events.RecordFailedUpdate(c.eventRecorder, obj, err)
		}
		return err
	}

	// Record success event if events are enabled
	if c.enableEvents && c.eventRecorder != nil {
		events.RecordSuccessfulUpdate(c.eventRecorder, obj)
	}

	return nil
}

// Delete deletes the given obj from the k1s storage and records appropriate events
func (c *eventAwareClient) Delete(ctx context.Context, obj Object, opts ...DeleteOption) error {
	// Perform the actual delete operation
	err := c.Client.Delete(ctx, obj, opts...)

	if err != nil {
		// Record failure event if events are enabled
		if c.enableEvents && c.eventRecorder != nil {
			events.RecordFailedDelete(c.eventRecorder, obj, err)
		}
		return err
	}

	// Record success event if events are enabled
	if c.enableEvents && c.eventRecorder != nil {
		events.RecordSuccessfulDelete(c.eventRecorder, obj)
	}

	return nil
}

// WithEventRecording creates a new EventAwareClient from an existing client with event recording enabled
func WithEventRecording(client Client, eventRecorder events.EventRecorder) EventAwareClient {
	return NewEventAwareClient(EventAwareClientOptions{
		Client:        client,
		EventRecorder: eventRecorder,
		EnableEvents:  true,
	})
}

// WithOptionalEventRecording creates a new EventAwareClient from an existing client with optional event recording
func WithOptionalEventRecording(client Client, eventRecorder events.EventRecorder, enableEvents bool) EventAwareClient {
	return NewEventAwareClient(EventAwareClientOptions{
		Client:        client,
		EventRecorder: eventRecorder,
		EnableEvents:  enableEvents,
	})
}

// EventClientFactory provides a factory interface for creating event-aware clients
type EventClientFactory interface {
	// CreateEventAwareClient creates a new event-aware client
	CreateEventAwareClient(client Client, eventRecorder events.EventRecorder) EventAwareClient
}

// DefaultEventClientFactory implements EventClientFactory
type DefaultEventClientFactory struct{}

// CreateEventAwareClient creates a new event-aware client using the default implementation
func (f *DefaultEventClientFactory) CreateEventAwareClient(client Client, eventRecorder events.EventRecorder) EventAwareClient {
	return WithEventRecording(client, eventRecorder)
}

// Helper functions for creating event sources

// CreateEventSource creates an event source for client operations
func CreateEventSource() corev1.EventSource {
	return events.NewEventSource("k1s-client")
}

// CreateEventRecorderForClient creates an event recorder configured for client operations
func CreateEventRecorderForClient(broadcaster events.EventBroadcaster, scheme *runtime.Scheme) events.EventRecorder {
	if broadcaster == nil || scheme == nil {
		return nil
	}

	source := CreateEventSource()
	return broadcaster.NewRecorder(scheme, source)
}

// Ensure eventAwareClient implements EventAwareClient interface
var _ EventAwareClient = (*eventAwareClient)(nil)

// Ensure DefaultEventClientFactory implements EventClientFactory interface
var _ EventClientFactory = (*DefaultEventClientFactory)(nil)
