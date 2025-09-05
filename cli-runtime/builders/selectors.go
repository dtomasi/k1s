package builders

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/dtomasi/k1s/core/client"
)

// ResourceSelector provides methods for selecting and filtering resources.
type ResourceSelector struct {
	gvk           schema.GroupVersionKind
	names         []string
	namespace     string
	allNamespaces bool
	labelSelector map[string]string
	fieldSelector map[string]string
}

// NewResourceSelector creates a new resource selector.
func NewResourceSelector() *ResourceSelector {
	return &ResourceSelector{}
}

// ForType sets the resource type to select.
func (s *ResourceSelector) ForType(gvk schema.GroupVersionKind) *ResourceSelector {
	s.gvk = gvk
	return s
}

// WithNames sets specific resource names to select.
func (s *ResourceSelector) WithNames(names ...string) *ResourceSelector {
	s.names = names
	return s
}

// InNamespace sets the namespace to select from.
func (s *ResourceSelector) InNamespace(namespace string) *ResourceSelector {
	s.namespace = namespace
	s.allNamespaces = false
	return s
}

// InAllNamespaces sets the selector to span all namespaces.
func (s *ResourceSelector) InAllNamespaces() *ResourceSelector {
	s.allNamespaces = true
	s.namespace = ""
	return s
}

// WithLabels sets label selectors.
func (s *ResourceSelector) WithLabels(labels map[string]string) *ResourceSelector {
	s.labelSelector = labels
	return s
}

// WithFields sets field selectors.
func (s *ResourceSelector) WithFields(fields map[string]string) *ResourceSelector {
	s.fieldSelector = fields
	return s
}

// ToListOptions converts the selector to client.ListOption slice.
func (s *ResourceSelector) ToListOptions() []client.ListOption {
	var opts []client.ListOption

	// Add label selector
	if len(s.labelSelector) > 0 {
		opts = append(opts, client.MatchingLabels(s.labelSelector))
	}

	// Add field selector
	if len(s.fieldSelector) > 0 {
		opts = append(opts, client.MatchingFields(s.fieldSelector))
	}

	// Add namespace selector
	if s.namespace != "" && !s.allNamespaces {
		opts = append(opts, client.InNamespace(s.namespace))
	}

	return opts
}

// ToResourceBuilder converts the selector to a ResourceBuilder.
func (s *ResourceSelector) ToResourceBuilder(client client.Client) ResourceBuilder {
	builder := NewResourceBuilder().WithClient(client).WithResourceType(s.gvk)

	// Apply selections
	if len(s.labelSelector) > 0 {
		builder = builder.WithLabelSelector(s.labelSelector)
	}

	if len(s.fieldSelector) > 0 {
		builder = builder.WithFieldSelector(s.fieldSelector)
	}

	if s.allNamespaces {
		builder = builder.AllNamespaces()
	} else if s.namespace != "" {
		builder = builder.WithNamespace(s.namespace)
	}

	// If specific names are provided, handle them specially
	// For now, we just use the first name
	if len(s.names) > 0 {
		builder = builder.WithName(s.names[0])
	}

	return builder
}

// ResourceFilter provides filtering capabilities for resource lists.
type ResourceFilter struct {
	includeFunc func(client.Object) bool
	excludeFunc func(client.Object) bool
}

// NewResourceFilter creates a new resource filter.
func NewResourceFilter() *ResourceFilter {
	return &ResourceFilter{}
}

// Include sets a function to determine which objects to include.
func (f *ResourceFilter) Include(fn func(client.Object) bool) *ResourceFilter {
	f.includeFunc = fn
	return f
}

// Exclude sets a function to determine which objects to exclude.
func (f *ResourceFilter) Exclude(fn func(client.Object) bool) *ResourceFilter {
	f.excludeFunc = fn
	return f
}

// Filter filters a slice of objects based on the configured criteria.
func (f *ResourceFilter) Filter(objects []client.Object) []client.Object {
	if f.includeFunc == nil && f.excludeFunc == nil {
		return objects
	}

	var filtered []client.Object
	for _, obj := range objects {
		// Check include criteria
		if f.includeFunc != nil && !f.includeFunc(obj) {
			continue
		}

		// Check exclude criteria
		if f.excludeFunc != nil && f.excludeFunc(obj) {
			continue
		}

		filtered = append(filtered, obj)
	}

	return filtered
}

// CommonFilters provides common filtering functions.
type CommonFilters struct{}

// ByLabel creates a filter that matches objects with specific labels.
func (CommonFilters) ByLabel(key, value string) func(client.Object) bool {
	return func(obj client.Object) bool {
		labels := obj.GetLabels()
		return labels != nil && labels[key] == value
	}
}

// ByName creates a filter that matches objects with specific names.
func (CommonFilters) ByName(names ...string) func(client.Object) bool {
	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}

	return func(obj client.Object) bool {
		return nameSet[obj.GetName()]
	}
}

// ByNamespace creates a filter that matches objects in specific namespaces.
func (CommonFilters) ByNamespace(namespaces ...string) func(client.Object) bool {
	namespaceSet := make(map[string]bool)
	for _, ns := range namespaces {
		namespaceSet[ns] = true
	}

	return func(obj client.Object) bool {
		return namespaceSet[obj.GetNamespace()]
	}
}

// ByPrefix creates a filter that matches objects with names starting with a prefix.
func (CommonFilters) ByPrefix(prefix string) func(client.Object) bool {
	return func(obj client.Object) bool {
		return strings.HasPrefix(obj.GetName(), prefix)
	}
}

// ParseLabelSelector parses a kubectl-style label selector string.
func ParseLabelSelector(selector string) (map[string]string, error) {
	if selector == "" {
		return nil, nil
	}

	labelMap := make(map[string]string)

	// Split by comma
	pairs := strings.Split(selector, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		// Handle different operators (simplified implementation)
		if strings.Contains(pair, "=") {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				labelMap[key] = value
			}
		} else {
			// Simple key-only selector (existence check)
			key := strings.TrimSpace(pair)
			labelMap[key] = ""
		}
	}

	return labelMap, nil
}

// ParseFieldSelector parses a kubectl-style field selector string.
func ParseFieldSelector(selector string) (map[string]string, error) {
	if selector == "" {
		return nil, nil
	}

	fieldMap := make(map[string]string)

	// Split by comma
	pairs := strings.Split(selector, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		// Split by equals
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid field selector format: %s", pair)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		fieldMap[key] = value
	}

	return fieldMap, nil
}
