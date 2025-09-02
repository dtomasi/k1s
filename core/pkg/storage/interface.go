package storage

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
)

// Interface defines the Kubernetes-compatible storage interface for k1s.
// This interface must match k8s.io/apiserver/pkg/storage.Interface exactly
// to ensure compatibility with existing Kubernetes storage implementations.
type Interface interface {
	// Versioner returns a storage.Versioner for managing resource versions
	Versioner() storage.Versioner

	// Create adds a new object at a key unless it already exists.
	// 'ttl' is time-to-live in seconds (0 means no TTL).
	// If no error is returned and out is not nil, out will be set to the created object.
	Create(ctx context.Context, key string, obj, out runtime.Object, ttl uint64) error

	// Delete removes the specified key and returns the value that existed at that key.
	// If key didn't exist, it will return NotFound storage error.
	// If 'cachedExistingObject' is non-nil, it can be used as a suggestion about the
	// current version of the object to avoid read operation from storage to get it.
	Delete(ctx context.Context, key string, out runtime.Object, preconditions *storage.Preconditions,
		validateDeletion storage.ValidateObjectFunc, cachedExistingObject runtime.Object) error

	// Watch begins watching a specific key or key prefix for changes.
	// The returned watch.Interface can be used to receive events.
	Watch(ctx context.Context, key string, opts storage.ListOptions) (watch.Interface, error)

	// Get unmarshals object found at key into objPtr. On a not found error,
	// will either return a zero object of the requested type, or an error,
	// depending on 'opts.ignoreNotFound'. Treats empty responses and nil response nodes exactly like a not found error.
	Get(ctx context.Context, key string, opts storage.GetOptions, objPtr runtime.Object) error

	// List unmarshalls objects found at key into a *List api object (an object
	// that satisfies runtime.IsList definition).
	// If 'opts.Recursive' is false, 'key' is used as an exact match. If `opts.Recursive`
	// is true, 'key' is used as a prefix.
	// The returned contents may be delayed, but it is guaranteed that they will
	// match 'opts.ResourceVersion' according to database consistency rules (e.g. linearizability).
	List(ctx context.Context, key string, opts storage.ListOptions, listObj runtime.Object) error
}

// Backend defines additional methods that storage backends may implement
// for enhanced functionality beyond the basic Kubernetes storage.Interface.
type Backend interface {
	Interface

	// Name returns the name of this storage backend
	Name() string

	// Close closes the storage backend and cleans up resources
	Close() error

	// Compact performs storage compaction if supported
	Compact(ctx context.Context) error

	// Count returns the number of objects stored under the given key prefix
	Count(ctx context.Context, key string) (int64, error)
}

// Factory creates storage instances with specific configurations
type Factory interface {
	// Create creates a new storage instance with the given configuration
	Create(config Config) (Interface, error)

	// CreateBackend creates a new backend storage instance
	CreateBackend(config Config) (Backend, error)

	// SupportedBackends returns the list of supported backend types
	SupportedBackends() []string
}
