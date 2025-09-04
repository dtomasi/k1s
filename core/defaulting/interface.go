package defaulting

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Defaulter provides the interface for applying default values to objects.
// This interface is compatible with controller-runtime defaulting patterns.
type Defaulter interface {
	// Default applies default values to the given object.
	// This method should be idempotent and safe to call multiple times.
	Default(ctx context.Context, obj runtime.Object) error
}

// DefaultingStrategy defines how to apply default values for a specific object type.
// Implementations should be thread-safe as they may be called concurrently.
type DefaultingStrategy interface {
	// Apply applies default values to the object based on the configured strategy.
	Apply(ctx context.Context, obj runtime.Object) error

	// SupportsType returns true if this strategy can handle the given object type.
	SupportsType(obj runtime.Object) bool
}

// DefaultingManager coordinates multiple defaulting strategies and provides
// a unified interface for applying defaults to objects.
type DefaultingManager interface {
	Defaulter

	// RegisterStrategy registers a defaulting strategy for a specific type.
	RegisterStrategy(strategy DefaultingStrategy) error

	// UnregisterStrategy removes a defaulting strategy.
	UnregisterStrategy(strategy DefaultingStrategy) error

	// HasDefaultsFor returns true if defaults are configured for the given object type.
	HasDefaultsFor(obj runtime.Object) bool

	// RegisterObjectDefaults registers default value configurations for a specific object type.
	RegisterObjectDefaults(objDefaults *ObjectDefaults) error
}

// FieldDefaulter provides fine-grained defaulting for individual fields.
// This interface allows for conditional defaulting based on object state.
type FieldDefaulter interface {
	// DefaultField applies a default value to a specific field in the object.
	// The fieldPath parameter specifies the JSON path to the field.
	DefaultField(ctx context.Context, obj runtime.Object, fieldPath string, defaultValue interface{}) error

	// ShouldDefault determines if a field should have its default value applied
	// based on the current state of the object.
	ShouldDefault(ctx context.Context, obj runtime.Object, fieldPath string) bool
}

// DefaultValue represents a default value configuration extracted from kubebuilder markers.
type DefaultValue struct {
	// FieldPath is the JSON path to the field (e.g., ".spec.quantity")
	FieldPath string

	// Value is the default value to apply
	Value interface{}

	// Condition specifies when this default should be applied (optional)
	Condition string

	// Type is the expected type of the value
	Type string
}

// ObjectDefaults contains all default value configurations for a specific object type.
type ObjectDefaults struct {
	// GVK identifies the object type these defaults apply to
	GVK schema.GroupVersionKind

	// Defaults is a list of default value configurations
	Defaults []DefaultValue
}
