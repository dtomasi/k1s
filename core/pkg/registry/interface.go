package registry

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Registry manages dynamic resource type registration and provides
// metadata management for Kubernetes-style resources.
type Registry interface {
	// RegisterResource registers a new resource with its configuration
	RegisterResource(gvr schema.GroupVersionResource, config ResourceConfig) error

	// GetResourceConfig retrieves the configuration for a registered resource
	GetResourceConfig(gvr schema.GroupVersionResource) (ResourceConfig, error)

	// ListResources returns all registered resources
	ListResources() []schema.GroupVersionResource

	// GetGVRForShortName resolves a short name to its full GVR
	GetGVRForShortName(shortName string) (schema.GroupVersionResource, error)

	// GetGVRsForCategory returns all resources in a given category
	GetGVRsForCategory(category string) []schema.GroupVersionResource

	// IsResourceRegistered checks if a resource is registered
	IsResourceRegistered(gvr schema.GroupVersionResource) bool

	// UnregisterResource removes a resource from the registry
	UnregisterResource(gvr schema.GroupVersionResource) error

	// GetGVKForGVR converts a GVR to its corresponding GVK
	GetGVKForGVR(gvr schema.GroupVersionResource) (schema.GroupVersionKind, error)

	// GetGVRForGVK converts a GVK to its corresponding GVR
	GetGVRForGVK(gvk schema.GroupVersionKind) (schema.GroupVersionResource, error)
}

// ResourceConfig contains metadata and configuration for a registered resource.
type ResourceConfig struct {
	// PrintColumns defines how this resource should be displayed in table format
	PrintColumns []metav1.TableColumnDefinition `json:"printColumns,omitempty"`

	// ShortNames provides alternative short names for this resource
	ShortNames []string `json:"shortNames,omitempty"`

	// Categories groups resources by logical categories (e.g., "all")
	Categories []string `json:"categories,omitempty"`

	// Singular is the singular form of the resource name
	Singular string `json:"singular"`

	// Plural is the plural form of the resource name (usually the resource name)
	Plural string `json:"plural"`

	// Kind is the kind name for this resource
	Kind string `json:"kind"`

	// ListKind is the kind name for list operations
	ListKind string `json:"listKind"`

	// Namespaced indicates if this resource is namespace-scoped
	Namespaced bool `json:"namespaced"`

	// Description provides human-readable description of the resource
	Description string `json:"description,omitempty"`
}

// RegistryOption allows for functional configuration of the registry.
type RegistryOption func(*registryConfig)

// registryConfig holds configuration options for the registry.
type registryConfig struct {
	// DefaultCategories are categories that all resources should belong to
	DefaultCategories []string

	// EnableShortNameValidation ensures short names don't conflict
	EnableShortNameValidation bool

	// CaseSensitiveShortNames controls case sensitivity for short name lookups
	CaseSensitiveShortNames bool
}

// WithDefaultCategories sets default categories that all resources inherit.
func WithDefaultCategories(categories ...string) RegistryOption {
	return func(config *registryConfig) {
		config.DefaultCategories = categories
	}
}

// WithShortNameValidation enables validation to prevent short name conflicts.
func WithShortNameValidation(enabled bool) RegistryOption {
	return func(config *registryConfig) {
		config.EnableShortNameValidation = enabled
	}
}

// WithCaseSensitiveShortNames controls whether short name lookups are case sensitive.
func WithCaseSensitiveShortNames(enabled bool) RegistryOption {
	return func(config *registryConfig) {
		config.CaseSensitiveShortNames = enabled
	}
}