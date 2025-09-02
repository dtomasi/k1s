package runtime

import (
	"fmt"
	"reflect"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

// Global scheme and codec instances for k1s runtime
var (
	Scheme = NewScheme()
	Codecs = serializer.NewCodecFactory(Scheme)
)

// SchemeBuilder collects functions that add types to a scheme.
// This follows the Kubernetes pattern for registering types.
type SchemeBuilder struct {
	funcs []func(*runtime.Scheme) error
}

// NewSchemeBuilder creates a new SchemeBuilder with the given functions.
func NewSchemeBuilder(funcs ...func(*runtime.Scheme) error) *SchemeBuilder {
	return &SchemeBuilder{funcs: funcs}
}

// AddToScheme applies all stored functions to the scheme.
func (sb *SchemeBuilder) AddToScheme(s *runtime.Scheme) error {
	for _, f := range sb.funcs {
		if err := f(s); err != nil {
			return fmt.Errorf("failed to add to scheme: %w", err)
		}
	}
	return nil
}

// Register adds functions to the SchemeBuilder.
func (sb *SchemeBuilder) Register(funcs ...func(*runtime.Scheme) error) {
	sb.funcs = append(sb.funcs, funcs...)
}

// K1SScheme wraps the Kubernetes runtime.Scheme with additional k1s functionality
// while maintaining full compatibility with the Kubernetes runtime interfaces.
type K1SScheme struct {
	*runtime.Scheme

	// Thread-safe access to scheme operations
	mu sync.RWMutex

	// Track registered types for validation
	registeredTypes map[schema.GroupVersionKind]reflect.Type
}

// NewScheme creates a new k1s scheme with Kubernetes compatibility.
func NewScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

	// Add core Kubernetes types
	if err := addKnownTypes(scheme); err != nil {
		panic(fmt.Errorf("failed to initialize k1s scheme: %w", err))
	}

	return scheme
}

// NewK1SScheme creates a new K1SScheme wrapper with enhanced functionality.
func NewK1SScheme() *K1SScheme {
	return &K1SScheme{
		Scheme:          NewScheme(),
		registeredTypes: make(map[schema.GroupVersionKind]reflect.Type),
	}
}

// AddKnownTypes adds types to the scheme with validation.
func (s *K1SScheme) AddKnownTypes(gv schema.GroupVersion, objs ...runtime.Object) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, obj := range objs {
		if obj == nil {
			return fmt.Errorf("cannot register nil object")
		}

		objType := reflect.TypeOf(obj)
		if objType == nil {
			return fmt.Errorf("cannot determine type of object")
		}

		// Get the element type if it's a pointer
		if objType.Kind() == reflect.Ptr {
			objType = objType.Elem()
		}

		gvk := gv.WithKind(objType.Name())
		s.registeredTypes[gvk] = objType
	}

	s.Scheme.AddKnownTypes(gv, objs...)
	return nil
}

// IsTypeRegistered checks if a GVK is registered in the scheme.
func (s *K1SScheme) IsTypeRegistered(gvk schema.GroupVersionKind) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.registeredTypes[gvk]
	return exists
}

// GetRegisteredTypes returns all registered types.
func (s *K1SScheme) GetRegisteredTypes() map[schema.GroupVersionKind]reflect.Type {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[schema.GroupVersionKind]reflect.Type)
	for gvk, t := range s.registeredTypes {
		result[gvk] = t
	}
	return result
}

// ObjectKinds returns the GVKs for an object with enhanced validation.
func (s *K1SScheme) ObjectKinds(obj runtime.Object) ([]schema.GroupVersionKind, bool, error) {
	if obj == nil {
		return nil, false, fmt.Errorf("cannot determine kinds for nil object")
	}

	return s.Scheme.ObjectKinds(obj)
}

// New creates a new instance of the type for the given GVK with validation.
func (s *K1SScheme) New(kind schema.GroupVersionKind) (runtime.Object, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, exists := s.registeredTypes[kind]; !exists {
		return nil, fmt.Errorf("type %v is not registered in scheme", kind)
	}

	return s.Scheme.New(kind)
}

// addKnownTypes adds the base Kubernetes types to the scheme.
func addKnownTypes(_ *runtime.Scheme) error {
	// This function will be extended when we add specific k1s types
	// For now, it provides a placeholder for future type registration
	return nil
}

// AddToScheme is the global function to add k1s types to a scheme.
func AddToScheme(s *runtime.Scheme) error {
	return addKnownTypes(s)
}
