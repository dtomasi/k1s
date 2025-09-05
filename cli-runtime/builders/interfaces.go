// Package builders provides fluent API builders for resource operations.
// These builders implement kubectl-style resource selection and filtering
// patterns for building CLI applications.
package builders

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/dtomasi/k1s/core/client"
)

// ResourceBuilder provides a fluent API for building resource queries.
type ResourceBuilder interface {
	// WithClient sets the client to use for operations
	WithClient(client client.Client) ResourceBuilder

	// WithResourceType sets the resource type to operate on
	WithResourceType(gvk schema.GroupVersionKind) ResourceBuilder

	// WithName sets a specific resource name
	WithName(name string) ResourceBuilder

	// WithNamespace sets a specific namespace
	WithNamespace(namespace string) ResourceBuilder

	// WithLabelSelector sets a label selector
	WithLabelSelector(selector map[string]string) ResourceBuilder

	// WithFieldSelector sets a field selector
	WithFieldSelector(selector map[string]string) ResourceBuilder

	// AllNamespaces indicates the operation should span all namespaces
	AllNamespaces() ResourceBuilder

	// Do executes the built query and returns a result
	Do(ctx context.Context) *Result
}

// Result contains the result of a resource builder operation.
type Result struct {
	err     error
	objects []client.Object
	single  client.Object
}

// Error returns any error that occurred during the operation.
func (r *Result) Error() error {
	return r.err
}

// Objects returns the list of objects from the operation.
func (r *Result) Objects() ([]client.Object, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.single != nil {
		return []client.Object{r.single}, nil
	}
	return r.objects, nil
}

// Object returns a single object from the operation.
func (r *Result) Object() (client.Object, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.single != nil {
		return r.single, nil
	}
	if len(r.objects) > 0 {
		return r.objects[0], nil
	}
	return nil, nil
}

// NewResourceBuilder creates a new resource builder.
func NewResourceBuilder() ResourceBuilder {
	return &resourceBuilder{}
}
