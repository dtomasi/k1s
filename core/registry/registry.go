package registry

import (
	"fmt"
	"strings"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
)

// resourceRegistry is the default implementation of the Registry interface.
type resourceRegistry struct {
	mu sync.RWMutex

	// resources maps GVR to its configuration
	resources map[schema.GroupVersionResource]ResourceConfig

	// shortNames maps short names to GVRs for quick lookup
	shortNames map[string]schema.GroupVersionResource

	// categories maps category names to sets of GVRs
	categories map[string]sets.Set[schema.GroupVersionResource]

	// gvrToGVK maps GVR to GVK for conversion
	gvrToGVK map[schema.GroupVersionResource]schema.GroupVersionKind

	// gvkToGVR maps GVK to GVR for conversion
	gvkToGVR map[schema.GroupVersionKind]schema.GroupVersionResource

	// Configuration
	config registryConfig
}

// NewRegistry creates a new resource registry with optional configuration.
func NewRegistry(opts ...RegistryOption) Registry {
	config := registryConfig{
		DefaultCategories:         []string{"all"},
		EnableShortNameValidation: true,
		CaseSensitiveShortNames:   false,
	}

	for _, opt := range opts {
		opt(&config)
	}

	return &resourceRegistry{
		resources:  make(map[schema.GroupVersionResource]ResourceConfig),
		shortNames: make(map[string]schema.GroupVersionResource),
		categories: make(map[string]sets.Set[schema.GroupVersionResource]),
		gvrToGVK:   make(map[schema.GroupVersionResource]schema.GroupVersionKind),
		gvkToGVR:   make(map[schema.GroupVersionKind]schema.GroupVersionResource),
		config:     config,
	}
}

// RegisterResource registers a new resource with its configuration.
func (r *resourceRegistry) RegisterResource(gvr schema.GroupVersionResource, config ResourceConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate required fields
	if config.Singular == "" {
		return fmt.Errorf("resource configuration must specify singular name")
	}
	if config.Plural == "" {
		return fmt.Errorf("resource configuration must specify plural name")
	}
	if config.Kind == "" {
		return fmt.Errorf("resource configuration must specify kind")
	}

	// Check if resource is already registered
	if _, exists := r.resources[gvr]; exists {
		return fmt.Errorf("resource %s is already registered", gvr)
	}

	// Validate short names don't conflict if validation is enabled
	if r.config.EnableShortNameValidation {
		for _, shortName := range config.ShortNames {
			normalizedShortName := r.normalizeShortName(shortName)
			if existingGVR, exists := r.shortNames[normalizedShortName]; exists {
				return fmt.Errorf("short name %q conflicts with existing resource %s", shortName, existingGVR)
			}
		}
	}

	// Set default categories if none specified
	categories := config.Categories
	if len(categories) == 0 {
		categories = r.config.DefaultCategories
	} else {
		// Add default categories to user-specified ones
		categorySet := sets.New(categories...)
		for _, defaultCat := range r.config.DefaultCategories {
			categorySet.Insert(defaultCat)
		}
		categories = categorySet.UnsortedList()
	}
	config.Categories = categories

	// Set default ListKind if not specified
	if config.ListKind == "" {
		config.ListKind = config.Kind + "List"
	}

	// Register the resource
	r.resources[gvr] = config

	// Register short names
	for _, shortName := range config.ShortNames {
		normalizedShortName := r.normalizeShortName(shortName)
		r.shortNames[normalizedShortName] = gvr
	}

	// Register categories
	for _, category := range config.Categories {
		if r.categories[category] == nil {
			r.categories[category] = sets.New[schema.GroupVersionResource]()
		}
		r.categories[category].Insert(gvr)
	}

	// Register GVK <-> GVR mappings
	gvk := schema.GroupVersionKind{
		Group:   gvr.Group,
		Version: gvr.Version,
		Kind:    config.Kind,
	}
	r.gvrToGVK[gvr] = gvk
	r.gvkToGVR[gvk] = gvr

	return nil
}

// GetResourceConfig retrieves the configuration for a registered resource.
func (r *resourceRegistry) GetResourceConfig(gvr schema.GroupVersionResource) (ResourceConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	config, exists := r.resources[gvr]
	if !exists {
		return ResourceConfig{}, fmt.Errorf("resource %s is not registered", gvr)
	}

	// Return a copy to prevent external modification
	return config, nil
}

// ListResources returns all registered resources.
func (r *resourceRegistry) ListResources() []schema.GroupVersionResource {
	r.mu.RLock()
	defer r.mu.RUnlock()

	resources := make([]schema.GroupVersionResource, 0, len(r.resources))
	for gvr := range r.resources {
		resources = append(resources, gvr)
	}

	return resources
}

// GetGVRForShortName resolves a short name to its full GVR.
func (r *resourceRegistry) GetGVRForShortName(shortName string) (schema.GroupVersionResource, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	normalizedShortName := r.normalizeShortName(shortName)
	gvr, exists := r.shortNames[normalizedShortName]
	if !exists {
		return schema.GroupVersionResource{}, fmt.Errorf("short name %q is not registered", shortName)
	}

	return gvr, nil
}

// GetGVRsForCategory returns all resources in a given category.
func (r *resourceRegistry) GetGVRsForCategory(category string) []schema.GroupVersionResource {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if gvrSet, exists := r.categories[category]; exists {
		return gvrSet.UnsortedList()
	}

	return nil
}

// IsResourceRegistered checks if a resource is registered.
func (r *resourceRegistry) IsResourceRegistered(gvr schema.GroupVersionResource) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.resources[gvr]
	return exists
}

// UnregisterResource removes a resource from the registry.
func (r *resourceRegistry) UnregisterResource(gvr schema.GroupVersionResource) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	config, exists := r.resources[gvr]
	if !exists {
		return fmt.Errorf("resource %s is not registered", gvr)
	}

	// Remove from resources
	delete(r.resources, gvr)

	// Remove short names
	for _, shortName := range config.ShortNames {
		normalizedShortName := r.normalizeShortName(shortName)
		delete(r.shortNames, normalizedShortName)
	}

	// Remove from categories
	for _, category := range config.Categories {
		if gvrSet, exists := r.categories[category]; exists {
			gvrSet.Delete(gvr)
			// Clean up empty category sets
			if gvrSet.Len() == 0 {
				delete(r.categories, category)
			}
		}
	}

	// Remove GVK mappings
	if gvk, exists := r.gvrToGVK[gvr]; exists {
		delete(r.gvrToGVK, gvr)
		delete(r.gvkToGVR, gvk)
	}

	return nil
}

// GetGVKForGVR converts a GVR to its corresponding GVK.
func (r *resourceRegistry) GetGVKForGVR(gvr schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	gvk, exists := r.gvrToGVK[gvr]
	if !exists {
		return schema.GroupVersionKind{}, fmt.Errorf("GVK mapping not found for GVR %s", gvr)
	}

	return gvk, nil
}

// GetGVRForGVK converts a GVK to its corresponding GVR.
func (r *resourceRegistry) GetGVRForGVK(gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	gvr, exists := r.gvkToGVR[gvk]
	if !exists {
		return schema.GroupVersionResource{}, fmt.Errorf("GVR mapping not found for GVK %s", gvk)
	}

	return gvr, nil
}

// normalizeShortName normalizes short names for comparison based on configuration.
func (r *resourceRegistry) normalizeShortName(shortName string) string {
	if r.config.CaseSensitiveShortNames {
		return shortName
	}
	return strings.ToLower(shortName)
}

// GetDefaultPrintColumns returns default print columns for basic resource display.
func GetDefaultPrintColumns() []metav1.TableColumnDefinition {
	return []metav1.TableColumnDefinition{
		{
			Name:        "Name",
			Type:        "string",
			Format:      "name",
			Description: "Name of the resource",
			Priority:    0,
		},
		{
			Name:        "Age",
			Type:        "string",
			Format:      "",
			Description: "Age of the resource",
			Priority:    0,
		},
	}
}

// GetDefaultPrintColumnsWithNamespace returns default print columns including namespace.
func GetDefaultPrintColumnsWithNamespace() []metav1.TableColumnDefinition {
	return []metav1.TableColumnDefinition{
		{
			Name:        "Namespace",
			Type:        "string",
			Format:      "",
			Description: "Namespace of the resource",
			Priority:    0,
		},
		{
			Name:        "Name",
			Type:        "string",
			Format:      "name",
			Description: "Name of the resource",
			Priority:    0,
		},
		{
			Name:        "Age",
			Type:        "string",
			Format:      "",
			Description: "Age of the resource",
			Priority:    0,
		},
	}
}
