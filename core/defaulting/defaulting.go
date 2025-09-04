package defaulting

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// manager is the default implementation of DefaultingManager.
type manager struct {
	mu         sync.RWMutex
	strategies map[schema.GroupVersionKind][]DefaultingStrategy
	defaults   map[schema.GroupVersionKind]*ObjectDefaults
}

// NewManager creates a new defaulting manager.
func NewManager() DefaultingManager {
	return &manager{
		strategies: make(map[schema.GroupVersionKind][]DefaultingStrategy),
		defaults:   make(map[schema.GroupVersionKind]*ObjectDefaults),
	}
}

// Default applies default values to the given object using registered strategies.
func (m *manager) Default(ctx context.Context, obj runtime.Object) error {
	if obj == nil {
		return fmt.Errorf("cannot apply defaults to nil object")
	}

	gvk := obj.GetObjectKind().GroupVersionKind()
	if gvk.Empty() {
		// Try to infer GVK from the object type
		gvk = m.inferGVK(obj)
	}

	m.mu.RLock()
	strategies := m.strategies[gvk]
	allStrategies := m.strategies[schema.GroupVersionKind{Group: "*", Version: "*", Kind: "*"}]
	objDefaults := m.defaults[gvk]
	m.mu.RUnlock()

	// Apply GVK-specific strategy-based defaults first
	for _, strategy := range strategies {
		if strategy.SupportsType(obj) {
			if err := strategy.Apply(ctx, obj); err != nil {
				return fmt.Errorf("failed to apply defaulting strategy: %w", err)
			}
		}
	}

	// Apply global strategy-based defaults
	for _, strategy := range allStrategies {
		if strategy.SupportsType(obj) {
			if err := strategy.Apply(ctx, obj); err != nil {
				return fmt.Errorf("failed to apply defaulting strategy: %w", err)
			}
		}
	}

	// Apply kubebuilder marker-based defaults
	if objDefaults != nil {
		if err := m.applyObjectDefaults(ctx, obj, objDefaults); err != nil {
			return fmt.Errorf("failed to apply object defaults: %w", err)
		}
	}

	return nil
}

// RegisterStrategy registers a defaulting strategy for objects.
func (m *manager) RegisterStrategy(strategy DefaultingStrategy) error {
	if strategy == nil {
		return fmt.Errorf("strategy cannot be nil")
	}

	// For now, we register the strategy for all GVKs and let the strategy
	// determine if it supports the object type via SupportsType method.
	// A more sophisticated implementation could use type registration.

	m.mu.Lock()
	defer m.mu.Unlock()

	// We'll add this strategy to a special "all" GVK entry for now
	allGVK := schema.GroupVersionKind{Group: "*", Version: "*", Kind: "*"}
	m.strategies[allGVK] = append(m.strategies[allGVK], strategy)

	return nil
}

// UnregisterStrategy removes a defaulting strategy.
func (m *manager) UnregisterStrategy(strategy DefaultingStrategy) error {
	if strategy == nil {
		return fmt.Errorf("strategy cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove from all GVK entries
	for gvk, strategies := range m.strategies {
		for i, s := range strategies {
			if s == strategy {
				// Remove strategy from slice
				m.strategies[gvk] = append(strategies[:i], strategies[i+1:]...)
				break
			}
		}
		// Clean up empty slices
		if len(m.strategies[gvk]) == 0 {
			delete(m.strategies, gvk)
		}
	}

	return nil
}

// HasDefaultsFor returns true if defaults are configured for the given object type.
func (m *manager) HasDefaultsFor(obj runtime.Object) bool {
	if obj == nil {
		return false
	}

	gvk := obj.GetObjectKind().GroupVersionKind()
	if gvk.Empty() {
		gvk = m.inferGVK(obj)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check for GVK-specific strategies
	if strategies, exists := m.strategies[gvk]; exists && len(strategies) > 0 {
		for _, strategy := range strategies {
			if strategy.SupportsType(obj) {
				return true
			}
		}
	}

	// Check for global strategies
	allGVK := schema.GroupVersionKind{Group: "*", Version: "*", Kind: "*"}
	if strategies, exists := m.strategies[allGVK]; exists && len(strategies) > 0 {
		for _, strategy := range strategies {
			if strategy.SupportsType(obj) {
				return true
			}
		}
	}

	// Check for object defaults
	_, hasDefaults := m.defaults[gvk]
	return hasDefaults
}

// RegisterObjectDefaults registers default value configurations for a specific object type.
func (m *manager) RegisterObjectDefaults(objDefaults *ObjectDefaults) error {
	if objDefaults == nil {
		return fmt.Errorf("object defaults cannot be nil")
	}

	if objDefaults.GVK.Empty() {
		return fmt.Errorf("object defaults must specify a valid GVK")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.defaults[objDefaults.GVK] = objDefaults
	return nil
}

// applyObjectDefaults applies the default values defined in ObjectDefaults to the object.
func (m *manager) applyObjectDefaults(_ context.Context, obj runtime.Object, objDefaults *ObjectDefaults) error {
	objValue := reflect.ValueOf(obj)
	if objValue.Kind() == reflect.Ptr {
		objValue = objValue.Elem()
	}

	for _, defaultValue := range objDefaults.Defaults {
		if err := m.applyFieldDefault(objValue, defaultValue); err != nil {
			return fmt.Errorf("failed to apply default for field %s: %w", defaultValue.FieldPath, err)
		}
	}

	return nil
}

// applyFieldDefault applies a single field default to the object.
func (m *manager) applyFieldDefault(objValue reflect.Value, defaultValue DefaultValue) error {
	// Parse the field path and navigate to the field
	fieldValue, err := m.getFieldByPath(objValue, defaultValue.FieldPath)
	if err != nil {
		return fmt.Errorf("cannot access field %s: %w", defaultValue.FieldPath, err)
	}

	// Check if field is already set (non-zero value)
	if !m.isZeroValue(fieldValue) {
		return nil // Field already has a value, don't override
	}

	// Convert and set the default value
	convertedValue, err := m.convertValue(defaultValue.Value, fieldValue.Type())
	if err != nil {
		return fmt.Errorf("cannot convert default value %v to type %s: %w",
			defaultValue.Value, fieldValue.Type(), err)
	}

	if !fieldValue.CanSet() {
		return fmt.Errorf("field %s is not settable", defaultValue.FieldPath)
	}

	fieldValue.Set(convertedValue)
	return nil
}

// getFieldByPath navigates to a field using a dot-separated path like ".spec.quantity"
func (m *manager) getFieldByPath(objValue reflect.Value, fieldPath string) (reflect.Value, error) {
	if len(fieldPath) == 0 {
		return reflect.Value{}, fmt.Errorf("empty field path")
	}

	// Remove leading dot if present
	if fieldPath[0] == '.' {
		fieldPath = fieldPath[1:]
	}

	// Split path into components
	parts := splitPath(fieldPath)
	current := objValue

	for _, part := range parts {
		if !current.IsValid() {
			return reflect.Value{}, fmt.Errorf("invalid value encountered in path")
		}

		// Handle pointer types
		if current.Kind() == reflect.Ptr {
			if current.IsNil() {
				// Initialize nil pointer
				current.Set(reflect.New(current.Type().Elem()))
			}
			current = current.Elem()
		}

		// Navigate to the field
		if current.Kind() != reflect.Struct {
			return reflect.Value{}, fmt.Errorf("cannot access field %s on non-struct type %s", part, current.Type())
		}

		field := current.FieldByName(capitalizeField(part))
		if !field.IsValid() {
			return reflect.Value{}, fmt.Errorf("field %s not found in type %s", part, current.Type())
		}

		current = field
	}

	return current, nil
}

// isZeroValue checks if a value is the zero value for its type
func (m *manager) isZeroValue(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}

	switch value.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func, reflect.Interface:
		return value.IsNil()
	default:
		zero := reflect.Zero(value.Type())
		return reflect.DeepEqual(value.Interface(), zero.Interface())
	}
}

// convertValue converts a default value to the target type
func (m *manager) convertValue(value interface{}, targetType reflect.Type) (reflect.Value, error) {
	if value == nil {
		return reflect.Zero(targetType), nil
	}

	sourceValue := reflect.ValueOf(value)

	// Direct assignment if types match
	if sourceValue.Type() == targetType {
		return sourceValue, nil
	}

	// Handle string conversions for basic types
	if sourceValue.Kind() == reflect.String {
		return m.convertFromString(sourceValue.String(), targetType)
	}

	// Try direct conversion
	if sourceValue.Type().ConvertibleTo(targetType) {
		return sourceValue.Convert(targetType), nil
	}

	return reflect.Value{}, fmt.Errorf("cannot convert %s to %s", sourceValue.Type(), targetType)
}

// convertFromString converts a string value to the target type
func (m *manager) convertFromString(value string, targetType reflect.Type) (reflect.Value, error) {
	switch targetType.Kind() {
	case reflect.String:
		return reflect.ValueOf(value), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(i).Convert(targetType), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(u).Convert(targetType), nil
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(f).Convert(targetType), nil
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(b), nil
	case reflect.Ptr:
		// Handle pointer to basic type
		elemType := targetType.Elem()
		elemValue, err := m.convertFromString(value, elemType)
		if err != nil {
			return reflect.Value{}, err
		}
		ptrValue := reflect.New(elemType)
		ptrValue.Elem().Set(elemValue)
		return ptrValue, nil
	default:
		return reflect.Value{}, fmt.Errorf("unsupported string conversion to type %s", targetType)
	}
}

// inferGVK tries to determine the GVK for an object
func (m *manager) inferGVK(obj runtime.Object) schema.GroupVersionKind {
	// This is a simplified implementation. In a real scenario, you'd use
	// the runtime scheme to get the GVK information.
	objType := reflect.TypeOf(obj)
	if objType.Kind() == reflect.Ptr {
		objType = objType.Elem()
	}

	// Extract package and type name to construct GVK
	// This is a basic heuristic - real implementations would use scheme registration
	pkg := objType.PkgPath()
	name := objType.Name()

	// For our examples: "github.com/dtomasi/k1s/examples/api/v1alpha1" -> Group: "inventory.k1s.io", Version: "v1alpha1"
	var group, version string
	if pkg != "" && name != "" {
		// Simple parsing - this would be more sophisticated in production
		if len(pkg) > 0 {
			parts := splitPath(pkg)
			if len(parts) > 0 {
				version = parts[len(parts)-1] // Last part is version
				group = "inventory.k1s.io"    // Hardcoded for our examples
			}
		}
	}

	gvk := schema.GroupVersionKind{
		Group:   group,
		Version: version,
		Kind:    name,
	}

	return gvk
}

// Utility functions

func splitPath(path string) []string {
	if path == "" {
		return []string{}
	}

	var parts []string
	current := ""

	for _, r := range path {
		if r == '.' || r == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

func capitalizeField(field string) string {
	if len(field) == 0 {
		return field
	}

	// Capitalize first letter to match Go struct field naming
	return string(field[0]-'a'+'A') + field[1:]
}

// basicStrategy is a simple example strategy for demonstration
type basicStrategy struct {
	supportedTypes map[reflect.Type]bool
	defaultFunc    func(ctx context.Context, obj runtime.Object) error
}

// NewBasicStrategy creates a basic defaulting strategy with a custom function.
func NewBasicStrategy(defaultFunc func(ctx context.Context, obj runtime.Object) error, supportedTypes ...reflect.Type) DefaultingStrategy {
	typeMap := make(map[reflect.Type]bool)
	for _, t := range supportedTypes {
		typeMap[t] = true
	}

	return &basicStrategy{
		supportedTypes: typeMap,
		defaultFunc:    defaultFunc,
	}
}

func (s *basicStrategy) Apply(ctx context.Context, obj runtime.Object) error {
	if s.defaultFunc == nil {
		return nil
	}
	return s.defaultFunc(ctx, obj)
}

func (s *basicStrategy) SupportsType(obj runtime.Object) bool {
	if len(s.supportedTypes) == 0 {
		return true // Support all types if none specified
	}

	objType := reflect.TypeOf(obj)
	return s.supportedTypes[objType]
}

// fieldDefaulter implements FieldDefaulter interface
type fieldDefaulter struct {
	manager *manager
}

// NewFieldDefaulter creates a new field defaulter.
func NewFieldDefaulter(mgr DefaultingManager) FieldDefaulter {
	if m, ok := mgr.(*manager); ok {
		return &fieldDefaulter{manager: m}
	}
	// Fallback for interface usage - create a new manager
	newMgr := NewManager()
	if m, ok := newMgr.(*manager); ok {
		return &fieldDefaulter{manager: m}
	}
	// Last resort fallback (should not happen)
	return &fieldDefaulter{manager: &manager{
		strategies: make(map[schema.GroupVersionKind][]DefaultingStrategy),
		defaults:   make(map[schema.GroupVersionKind]*ObjectDefaults),
	}}
}

func (f *fieldDefaulter) DefaultField(ctx context.Context, obj runtime.Object, fieldPath string, defaultValue interface{}) error {
	objValue := reflect.ValueOf(obj)
	if objValue.Kind() == reflect.Ptr {
		objValue = objValue.Elem()
	}

	fieldValue, err := f.manager.getFieldByPath(objValue, fieldPath)
	if err != nil {
		return err
	}

	if !f.manager.isZeroValue(fieldValue) {
		return nil // Field already has a value
	}

	convertedValue, err := f.manager.convertValue(defaultValue, fieldValue.Type())
	if err != nil {
		return err
	}

	if !fieldValue.CanSet() {
		return fmt.Errorf("field %s is not settable", fieldPath)
	}

	fieldValue.Set(convertedValue)
	return nil
}

func (f *fieldDefaulter) ShouldDefault(ctx context.Context, obj runtime.Object, fieldPath string) bool {
	objValue := reflect.ValueOf(obj)
	if objValue.Kind() == reflect.Ptr {
		objValue = objValue.Elem()
	}

	fieldValue, err := f.manager.getFieldByPath(objValue, fieldPath)
	if err != nil {
		return false
	}

	return f.manager.isZeroValue(fieldValue)
}
