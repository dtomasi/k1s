package builders

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/dtomasi/k1s/core/client"
)

// resourceBuilder implements ResourceBuilder.
type resourceBuilder struct {
	client        client.Client
	gvk           schema.GroupVersionKind
	name          string
	namespace     string
	labelSelector map[string]string
	fieldSelector map[string]string
	allNamespaces bool
}

// WithClient sets the client to use for operations.
func (b *resourceBuilder) WithClient(client client.Client) ResourceBuilder {
	b.client = client
	return b
}

// WithResourceType sets the resource type to operate on.
func (b *resourceBuilder) WithResourceType(gvk schema.GroupVersionKind) ResourceBuilder {
	b.gvk = gvk
	return b
}

// WithName sets a specific resource name.
func (b *resourceBuilder) WithName(name string) ResourceBuilder {
	b.name = name
	return b
}

// WithNamespace sets a specific namespace.
func (b *resourceBuilder) WithNamespace(namespace string) ResourceBuilder {
	b.namespace = namespace
	return b
}

// WithLabelSelector sets a label selector.
func (b *resourceBuilder) WithLabelSelector(selector map[string]string) ResourceBuilder {
	b.labelSelector = selector
	return b
}

// WithFieldSelector sets a field selector.
func (b *resourceBuilder) WithFieldSelector(selector map[string]string) ResourceBuilder {
	b.fieldSelector = selector
	return b
}

// AllNamespaces indicates the operation should span all namespaces.
func (b *resourceBuilder) AllNamespaces() ResourceBuilder {
	b.allNamespaces = true
	b.namespace = "" // Clear specific namespace
	return b
}

// Do executes the built query and returns a result.
func (b *resourceBuilder) Do(ctx context.Context) *Result {
	if b.client == nil {
		return &Result{err: fmt.Errorf("client not set")}
	}

	if b.gvk.Empty() {
		return &Result{err: fmt.Errorf("resource type not set")}
	}

	// If a specific name is provided, get a single resource
	if b.name != "" {
		return b.getSingle(ctx)
	}

	// Otherwise, list resources
	return b.getList(ctx)
}

// getSingle retrieves a single resource.
func (b *resourceBuilder) getSingle(ctx context.Context) *Result {
	obj, err := b.createObjectForGVK()
	if err != nil {
		return &Result{err: err}
	}

	key := client.ObjectKey{
		Name:      b.name,
		Namespace: b.namespace,
	}

	err = b.client.Get(ctx, key, obj)
	if err != nil {
		return &Result{err: err}
	}

	return &Result{single: obj}
}

// getList retrieves multiple resources.
func (b *resourceBuilder) getList(ctx context.Context) *Result {
	list, err := b.createListForGVK()
	if err != nil {
		return &Result{err: err}
	}

	// Build list options
	var opts []client.ListOption

	// Add label selector
	if len(b.labelSelector) > 0 {
		opts = append(opts, client.MatchingLabels(b.labelSelector))
	}

	// Add field selector
	if len(b.fieldSelector) > 0 {
		opts = append(opts, client.MatchingFields(b.fieldSelector))
	}

	// Add namespace selector
	if b.namespace != "" && !b.allNamespaces {
		opts = append(opts, client.InNamespace(b.namespace))
	}

	err = b.client.List(ctx, list, opts...)
	if err != nil {
		return &Result{err: err}
	}

	// Extract objects from list
	objects, err := b.extractObjects(list)
	if err != nil {
		return &Result{err: err}
	}

	return &Result{objects: objects}
}

// createObjectForGVK creates an empty object instance for the given GVK.
func (b *resourceBuilder) createObjectForGVK() (client.Object, error) {
	scheme := b.client.Scheme()
	obj, err := scheme.New(b.gvk)
	if err != nil {
		return nil, fmt.Errorf("failed to create new object for GVK %s: %w", b.gvk, err)
	}

	clientObj, ok := obj.(client.Object)
	if !ok {
		return nil, fmt.Errorf("object for GVK %s does not implement client.Object", b.gvk)
	}

	return clientObj, nil
}

// createListForGVK creates an empty list instance for the given GVK.
func (b *resourceBuilder) createListForGVK() (client.ObjectList, error) {
	scheme := b.client.Scheme()

	// Convert the GVK to list GVK
	listGVK := b.gvk
	listGVK.Kind += "List"

	obj, err := scheme.New(listGVK)
	if err != nil {
		return nil, fmt.Errorf("failed to create new list for GVK %s: %w", listGVK, err)
	}

	list, ok := obj.(client.ObjectList)
	if !ok {
		return nil, fmt.Errorf("object for GVK %s does not implement client.ObjectList", listGVK)
	}

	return list, nil
}

// extractObjects extracts individual objects from a list.
func (b *resourceBuilder) extractObjects(list client.ObjectList) ([]client.Object, error) {
	// This is a simplified implementation - in a real scenario,
	// we would need more sophisticated reflection or interface methods
	// to extract items from different list types

	// For now, return empty list - this will be enhanced in future iterations
	return []client.Object{}, nil
}
