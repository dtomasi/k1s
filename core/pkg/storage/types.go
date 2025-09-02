package storage

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage/value"
)

// Config holds storage configuration options including multi-tenancy support
type Config struct {
	// TenantID provides tenant isolation for multi-tenant deployments
	TenantID string

	// Namespace provides additional key scoping within a tenant
	Namespace string

	// KeyPrefix is a custom prefix for all keys in this storage instance
	KeyPrefix string

	// Transformer handles encryption/decryption of stored objects
	Transformer value.Transformer
}

// TenantConfig provides tenant-specific configuration
type TenantConfig struct {
	// ID is the unique identifier for the tenant
	ID string

	// Prefix is the storage key prefix for this tenant
	Prefix string

	// Namespace provides default namespace for tenant operations
	Namespace string
}

// KeyOptions provides configuration for key generation
type KeyOptions struct {
	// Tenant specifies the tenant for multi-tenant key generation
	Tenant *TenantConfig

	// Namespace overrides the default namespace
	Namespace string

	// Resource is the resource type name
	Resource string

	// Name is the object name
	Name string
}

// SimpleVersioner implements storage.Versioner for resource version management
type SimpleVersioner struct{}

// UpdateObject updates the resource version of an object
func (v SimpleVersioner) UpdateObject(obj runtime.Object, resourceVersion uint64) error {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	versionString := EncodeResourceVersion(resourceVersion)
	accessor.SetResourceVersion(versionString)
	return nil
}

// UpdateList updates the resource version of a list object
func (v SimpleVersioner) UpdateList(obj runtime.Object, resourceVersion uint64, continueValue string, remainingItemCount *int64) error {
	listAccessor, err := meta.ListAccessor(obj)
	if err != nil {
		return err
	}
	versionString := EncodeResourceVersion(resourceVersion)
	listAccessor.SetResourceVersion(versionString)
	listAccessor.SetContinue(continueValue)
	listAccessor.SetRemainingItemCount(remainingItemCount)
	return nil
}

// PrepareObjectForStorage prepares an object for storage operations
func (v SimpleVersioner) PrepareObjectForStorage(obj runtime.Object) error {
	// Set creation timestamp if not already set
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	
	if accessor.GetCreationTimestamp().Time.IsZero() {
		now := metav1.Time{Time: time.Now()}
		accessor.SetCreationTimestamp(now)
	}

	return nil
}

// ObjectResourceVersion extracts the resource version from an object
func (v SimpleVersioner) ObjectResourceVersion(obj runtime.Object) (uint64, error) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return 0, err
	}
	return ParseResourceVersion(accessor.GetResourceVersion())
}

// ParseWatchResourceVersion parses a watch resource version string
func (v SimpleVersioner) ParseWatchResourceVersion(resourceVersion string) (uint64, error) {
	return ParseResourceVersion(resourceVersion)
}

// WatchEvent represents a single watch event
type WatchEvent struct {
	Type   watch.EventType
	Object runtime.Object
}

// WatchChan represents a channel of watch events
type WatchChan <-chan *WatchEvent

// SimpleWatch implements watch.Interface for storage watch operations
type SimpleWatch struct {
	result chan *WatchEvent
	done   chan struct{}
	closed bool
}

// NewSimpleWatch creates a new SimpleWatch instance
func NewSimpleWatch() *SimpleWatch {
	return &SimpleWatch{
		result: make(chan *WatchEvent, 100),
		done:   make(chan struct{}),
	}
}

// Stop implements watch.Interface
func (w *SimpleWatch) Stop() {
	if !w.closed {
		w.closed = true
		close(w.done)
	}
}

// ResultChan implements watch.Interface
func (w *SimpleWatch) ResultChan() <-chan watch.Event {
	eventChan := make(chan watch.Event, 100)
	
	go func() {
		defer close(eventChan)
		for {
			select {
			case event := <-w.result:
				if event != nil {
					eventChan <- watch.Event{
						Type:   event.Type,
						Object: event.Object,
					}
				}
			case <-w.done:
				return
			}
		}
	}()
	
	return eventChan
}

// Send sends a watch event
func (w *SimpleWatch) Send(eventType watch.EventType, obj runtime.Object) {
	select {
	case w.result <- &WatchEvent{Type: eventType, Object: obj}:
	case <-w.done:
	}
}

// ContextCancelledError represents an error when context is cancelled
type ContextCancelledError struct {
	Err error
}

// Error implements error interface
func (e ContextCancelledError) Error() string {
	return "context cancelled: " + e.Err.Error()
}

// IsContextCancelled checks if an error is a context cancellation error
func IsContextCancelled(err error) bool {
	_, ok := err.(ContextCancelledError)
	return ok
}

// NewContextCancelledError creates a new context cancellation error
func NewContextCancelledError(ctx context.Context) error {
	return ContextCancelledError{Err: ctx.Err()}
}