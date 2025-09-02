package runtime

import (
	"fmt"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GVKMapper provides utilities for mapping between GroupVersionKind (GVK) and
// GroupVersionResource (GVR) values. This is essential for translating between
// the kind-based type system and the resource-based REST API.
type GVKMapper struct {
	mu sync.RWMutex

	// Bidirectional mappings between GVK and GVR
	gvkToGVR map[schema.GroupVersionKind]schema.GroupVersionResource
	gvrToGVK map[schema.GroupVersionResource]schema.GroupVersionKind

	// Kind name to GVK mappings for efficient lookup
	kindToGVKs map[string][]schema.GroupVersionKind
}

// DefaultGVKMapper is the global GVK mapper instance
var DefaultGVKMapper = NewGVKMapper()

// NewGVKMapper creates a new GVKMapper instance.
func NewGVKMapper() *GVKMapper {
	return &GVKMapper{
		gvkToGVR:   make(map[schema.GroupVersionKind]schema.GroupVersionResource),
		gvrToGVK:   make(map[schema.GroupVersionResource]schema.GroupVersionKind),
		kindToGVKs: make(map[string][]schema.GroupVersionKind),
	}
}

// RegisterMapping registers a bidirectional mapping between GVK and GVR.
// This is typically called during scheme registration to establish the
// relationship between kinds and their corresponding REST resources.
func (m *GVKMapper) RegisterMapping(gvk schema.GroupVersionKind, gvr schema.GroupVersionResource) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Store bidirectional mappings
	m.gvkToGVR[gvk] = gvr
	m.gvrToGVK[gvr] = gvk

	// Index by kind name for efficient lookup
	kindKey := strings.ToLower(gvk.Kind)
	m.kindToGVKs[kindKey] = append(m.kindToGVKs[kindKey], gvk)
}

// KindFor returns the GVK for a given GVR.
func (m *GVKMapper) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	gvk, exists := m.gvrToGVK[resource]
	if !exists {
		return schema.GroupVersionKind{}, fmt.Errorf("no kind registered for resource %v", resource)
	}
	return gvk, nil
}

// ResourceFor returns the GVR for a given GVK.
func (m *GVKMapper) ResourceFor(kind schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	gvr, exists := m.gvkToGVR[kind]
	if !exists {
		return schema.GroupVersionResource{}, fmt.Errorf("no resource registered for kind %v", kind)
	}
	return gvr, nil
}

// KindsFor returns all GVKs registered for a given GVR.
// This typically returns a single GVK, but could return multiple for resources
// that map to different kinds across versions.
func (m *GVKMapper) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	gvk, err := m.KindFor(resource)
	if err != nil {
		return nil, err
	}
	return []schema.GroupVersionKind{gvk}, nil
}

// ResourcesFor returns all GVRs registered for a given GVK.
// This typically returns a single GVR, but could return multiple in advanced scenarios.
func (m *GVKMapper) ResourcesFor(kind schema.GroupVersionKind) ([]schema.GroupVersionResource, error) {
	gvr, err := m.ResourceFor(kind)
	if err != nil {
		return nil, err
	}
	return []schema.GroupVersionResource{gvr}, nil
}

// KindsByName returns all GVKs that match the given kind name (case-insensitive).
func (m *GVKMapper) KindsByName(kindName string) []schema.GroupVersionKind {
	m.mu.RLock()
	defer m.mu.RUnlock()

	kindKey := strings.ToLower(kindName)
	gvks, exists := m.kindToGVKs[kindKey]
	if !exists {
		return nil
	}

	// Return a copy to prevent external modification
	result := make([]schema.GroupVersionKind, len(gvks))
	copy(result, gvks)
	return result
}

// GetAllMappings returns all registered GVK->GVR mappings.
func (m *GVKMapper) GetAllMappings() map[schema.GroupVersionKind]schema.GroupVersionResource {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[schema.GroupVersionKind]schema.GroupVersionResource)
	for gvk, gvr := range m.gvkToGVR {
		result[gvk] = gvr
	}
	return result
}

// GetGVKForObject returns the GroupVersionKind for a given runtime.Object.
// This is a convenience function that uses the global scheme.
func GetGVKForObject(obj runtime.Object, scheme *runtime.Scheme) (schema.GroupVersionKind, error) {
	if obj == nil {
		return schema.GroupVersionKind{}, fmt.Errorf("cannot determine GVK for nil object")
	}

	if scheme == nil {
		scheme = Scheme
	}

	gvks, _, err := scheme.ObjectKinds(obj)
	if err != nil {
		return schema.GroupVersionKind{}, fmt.Errorf("failed to get kinds for object: %w", err)
	}

	if len(gvks) == 0 {
		return schema.GroupVersionKind{}, fmt.Errorf("no kinds found for object type %T", obj)
	}

	// Return the first GVK if multiple exist
	return gvks[0], nil
}

// GetGVRForGVK returns the GroupVersionResource for a given GroupVersionKind
// using the global GVK mapper.
func GetGVRForGVK(gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	return DefaultGVKMapper.ResourceFor(gvk)
}

// GetGVKForGVR returns the GroupVersionKind for a given GroupVersionResource
// using the global GVK mapper.
func GetGVKForGVR(gvr schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	return DefaultGVKMapper.KindFor(gvr)
}

// RegisterGlobalMapping registers a GVK->GVR mapping in the global mapper.
// This is typically called during package initialization.
func RegisterGlobalMapping(gvk schema.GroupVersionKind, gvr schema.GroupVersionResource) {
	DefaultGVKMapper.RegisterMapping(gvk, gvr)
}

// PluralizationHelper provides utilities for converting between singular and plural forms
// of resource names. This is used when GVR mappings are not explicitly registered.
type PluralizationHelper struct {
	// Custom plural mappings for irregular cases
	singularToPlural map[string]string
	pluralToSingular map[string]string
}

// NewPluralizationHelper creates a new helper with common English pluralization rules.
func NewPluralizationHelper() *PluralizationHelper {
	helper := &PluralizationHelper{
		singularToPlural: make(map[string]string),
		pluralToSingular: make(map[string]string),
	}

	// Add common irregular plurals
	helper.AddMapping("child", "children")
	helper.AddMapping("person", "people")
	helper.AddMapping("datum", "data")

	return helper
}

// AddMapping adds a custom singular->plural mapping.
func (p *PluralizationHelper) AddMapping(singular, plural string) {
	p.singularToPlural[strings.ToLower(singular)] = strings.ToLower(plural)
	p.pluralToSingular[strings.ToLower(plural)] = strings.ToLower(singular)
}

// Pluralize converts a singular form to plural using basic English rules.
func (p *PluralizationHelper) Pluralize(singular string) string {
	lower := strings.ToLower(singular)

	// Check custom mappings first
	if plural, exists := p.singularToPlural[lower]; exists {
		return plural
	}

	// Apply basic English pluralization rules
	switch {
	case strings.HasSuffix(lower, "s"), strings.HasSuffix(lower, "sh"),
		strings.HasSuffix(lower, "ch"), strings.HasSuffix(lower, "x"):
		return singular + "es"
	case strings.HasSuffix(lower, "z"):
		return singular + "zes"
	case strings.HasSuffix(lower, "y"):
		if len(singular) > 1 && !isVowel(rune(lower[len(lower)-2])) {
			return singular[:len(singular)-1] + "ies"
		}
		return singular + "s"
	case strings.HasSuffix(lower, "f"):
		return singular[:len(singular)-1] + "ves"
	case strings.HasSuffix(lower, "fe"):
		return singular[:len(singular)-2] + "ves"
	default:
		return singular + "s"
	}
}

// Singularize converts a plural form to singular using basic English rules.
func (p *PluralizationHelper) Singularize(plural string) string {
	lower := strings.ToLower(plural)

	// Check custom mappings first
	if singular, exists := p.pluralToSingular[lower]; exists {
		return singular
	}

	// Apply basic English singularization rules
	switch {
	case strings.HasSuffix(lower, "ies") && len(plural) > 3:
		return plural[:len(plural)-3] + "y"
	case strings.HasSuffix(lower, "ves") && len(plural) > 3:
		base := plural[:len(plural)-3]
		baseLower := strings.ToLower(base)
		if strings.HasSuffix(baseLower, "l") || strings.HasSuffix(baseLower, "r") ||
			strings.HasSuffix(baseLower, "ea") {
			return base + "f"
		}
		return base + "fe"
	case strings.HasSuffix(lower, "es") && len(plural) > 2:
		base := plural[:len(plural)-2]
		baseLower := strings.ToLower(base)
		if strings.HasSuffix(baseLower, "s") || strings.HasSuffix(baseLower, "sh") ||
			strings.HasSuffix(baseLower, "ch") || strings.HasSuffix(baseLower, "x") ||
			strings.HasSuffix(baseLower, "z") {
			return base
		}
		return plural[:len(plural)-1]
	case strings.HasSuffix(lower, "s") && len(plural) > 1:
		return plural[:len(plural)-1]
	default:
		// For very short words or edge cases
		if len(plural) <= 1 {
			return ""
		}
		return plural
	}
}

// isVowel checks if a character is a vowel.
func isVowel(r rune) bool {
	vowels := "aeiou"
	return strings.ContainsRune(vowels, r)
}

// DefaultPluralizationHelper is the global instance used by utility functions.
var DefaultPluralizationHelper = NewPluralizationHelper()

// AutoGenerateGVR attempts to generate a GVR from a GVK using pluralization rules.
// This is used when explicit mappings are not available.
func AutoGenerateGVR(gvk schema.GroupVersionKind) schema.GroupVersionResource {
	plural := DefaultPluralizationHelper.Pluralize(gvk.Kind)
	return schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: strings.ToLower(plural),
	}
}
