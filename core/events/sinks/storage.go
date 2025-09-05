package sinks

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/dtomasi/k1s/core/client"
	"github.com/dtomasi/k1s/core/events"
)

// StorageSink implements EventSink by storing events as Kubernetes Event resources
// using the k1s client. This provides persistent storage of events that can be
// queried using standard CLI operations.
type StorageSink struct {
	client client.Client
	ctx    context.Context
}

// StorageSinkOptions provides configuration options for creating a StorageSink
type StorageSinkOptions struct {
	// Client is the k1s client used for storage operations
	Client client.Client

	// Context is the context used for storage operations
	Context context.Context
}

// NewStorageSink creates a new StorageSink instance
func NewStorageSink(options StorageSinkOptions) events.EventSink {
	if options.Context == nil {
		options.Context = context.Background()
	}

	return &StorageSink{
		client: options.Client,
		ctx:    options.Context,
	}
}

// Create creates a new event in storage
func (s *StorageSink) Create(event *corev1.Event) (*corev1.Event, error) {
	if event == nil {
		return nil, fmt.Errorf("event cannot be nil")
	}

	// Ensure required fields are set
	if event.Name == "" {
		return nil, fmt.Errorf("event name is required")
	}
	if event.Namespace == "" {
		event.Namespace = metav1.NamespaceDefault
	}

	// Set creation timestamp if not already set
	if event.CreationTimestamp.IsZero() {
		now := metav1.Now()
		event.CreationTimestamp = now
		if event.FirstTimestamp.IsZero() {
			event.FirstTimestamp = now
		}
		if event.LastTimestamp.IsZero() {
			event.LastTimestamp = now
		}
	}

	// Try to create the event
	err := s.client.Create(s.ctx, event)
	if err != nil {
		// If the event already exists, try to update it (for event aggregation)
		if apierrors.IsAlreadyExists(err) {
			return s.Update(event)
		}
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	return event, nil
}

// Update updates an existing event in storage (typically for count aggregation)
func (s *StorageSink) Update(event *corev1.Event) (*corev1.Event, error) {
	if event == nil {
		return nil, fmt.Errorf("event cannot be nil")
	}

	// Try to get the existing event first
	objectKey := client.ObjectKey{
		Namespace: event.Namespace,
		Name:      event.Name,
	}

	existingEvent := &corev1.Event{}
	err := s.client.Get(s.ctx, objectKey, existingEvent)
	if err != nil {
		// If event doesn't exist, create it
		if apierrors.IsNotFound(err) {
			return s.Create(event)
		}
		return nil, fmt.Errorf("failed to get existing event: %w", err)
	}

	// Aggregate the events - combine count and update timestamp
	updated := s.aggregateEvents(existingEvent, event)

	// Update the event
	err = s.client.Update(s.ctx, updated)
	if err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	return updated, nil
}

// Patch patches an existing event in storage
func (s *StorageSink) Patch(event *corev1.Event, data []byte) (*corev1.Event, error) {
	if event == nil {
		return nil, fmt.Errorf("event cannot be nil")
	}
	if data == nil {
		return nil, fmt.Errorf("patch data cannot be nil")
	}

	// Create a strategic merge patch
	patch := client.RawPatch{
		PatchType: types.StrategicMergePatchType,
		PatchData: data,
	}

	err := s.client.Patch(s.ctx, event, patch)
	if err != nil {
		return nil, fmt.Errorf("failed to patch event: %w", err)
	}

	return event, nil
}

// aggregateEvents combines information from two events for event aggregation
func (s *StorageSink) aggregateEvents(existing, new *corev1.Event) *corev1.Event {
	// Create a copy of the existing event to update
	updated := existing.DeepCopy()

	// Update count
	if new.Count > 0 {
		updated.Count += new.Count
	} else {
		updated.Count++
	}

	// Update last timestamp
	if !new.LastTimestamp.IsZero() {
		updated.LastTimestamp = new.LastTimestamp
	} else {
		updated.LastTimestamp = metav1.Now()
	}

	// Keep the first timestamp from the existing event (it should be earlier)
	// If the existing event doesn't have FirstTimestamp, use the new one
	if updated.FirstTimestamp.IsZero() && !new.FirstTimestamp.IsZero() {
		updated.FirstTimestamp = new.FirstTimestamp
	}

	// Update the message if it's different (could indicate new information)
	if new.Message != "" && new.Message != existing.Message {
		// For now, just use the new message. In a more sophisticated implementation,
		// you might want to combine messages or track message history
		updated.Message = new.Message
	}

	// Update other fields if they're different
	if new.Reason != "" && new.Reason != existing.Reason {
		updated.Reason = new.Reason
	}
	if new.Type != "" && new.Type != existing.Type {
		updated.Type = new.Type
	}

	return updated
}

// Helper functions for common storage sink operations

// CreateStorageSinkFromClient creates a StorageSink using the provided client
func CreateStorageSinkFromClient(client client.Client) events.EventSink {
	return NewStorageSink(StorageSinkOptions{
		Client:  client,
		Context: context.Background(),
	})
}

// CreateStorageSinkWithContext creates a StorageSink with a specific context
func CreateStorageSinkWithContext(client client.Client, ctx context.Context) events.EventSink {
	return NewStorageSink(StorageSinkOptions{
		Client:  client,
		Context: ctx,
	})
}

// EventStorageFactory provides a factory interface for creating storage sinks
type EventStorageFactory interface {
	// CreateEventStorageSink creates a new event storage sink
	CreateEventStorageSink(options StorageSinkOptions) events.EventSink
}

// DefaultEventStorageFactory implements EventStorageFactory
type DefaultEventStorageFactory struct{}

// CreateEventStorageSink creates a new storage sink using the default implementation
func (f *DefaultEventStorageFactory) CreateEventStorageSink(options StorageSinkOptions) events.EventSink {
	return NewStorageSink(options)
}

// Ensure StorageSink implements EventSink interface
var _ events.EventSink = (*StorageSink)(nil)

// Ensure DefaultEventStorageFactory implements EventStorageFactory interface
var _ EventStorageFactory = (*DefaultEventStorageFactory)(nil)
