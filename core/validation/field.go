package validation

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// fieldValidator implements FieldValidator interface.
type fieldValidator struct {
	manager *manager
}

// NewFieldValidator creates a new field validator.
func NewFieldValidator(mgr ValidationManager) FieldValidator {
	if m, ok := mgr.(*manager); ok {
		return &fieldValidator{manager: m}
	}
	// Fallback for interface usage - create a new manager
	newMgr := NewManager()
	if m, ok := newMgr.(*manager); ok {
		return &fieldValidator{manager: m}
	}
	// Last resort fallback (should not happen)
	return &fieldValidator{manager: &manager{
		strategies:  make(map[schema.GroupVersionKind][]ValidationStrategy),
		validations: make(map[schema.GroupVersionKind]*ObjectValidation),
	}}
}

// ValidateField validates a specific field in the object.
func (f *fieldValidator) ValidateField(ctx context.Context, obj runtime.Object, fieldPath string) []ValidationError {
	if obj == nil {
		return []ValidationError{{
			Field:   fieldPath,
			Type:    ValidationErrorTypeInvalid,
			Message: "cannot validate field on nil object",
		}}
	}

	if fieldPath == "" {
		return []ValidationError{{
			Field:   fieldPath,
			Type:    ValidationErrorTypeInvalid,
			Message: "field path cannot be empty",
		}}
	}

	// Get the field value
	fieldValue, err := f.GetFieldValue(obj, fieldPath)
	if err != nil {
		return []ValidationError{{
			Field:   fieldPath,
			Type:    ValidationErrorTypeInvalid,
			Message: fmt.Sprintf("cannot access field: %v", err),
		}}
	}

	// For now, just check if field exists and is accessible
	// In a real implementation, you'd apply specific validation rules for the field
	_ = fieldValue
	return nil
}

// GetFieldValue retrieves the value of a field from an object.
func (f *fieldValidator) GetFieldValue(obj runtime.Object, fieldPath string) (interface{}, error) {
	if obj == nil {
		return nil, fmt.Errorf("object is nil")
	}

	if fieldPath == "" {
		return nil, fmt.Errorf("field path is empty")
	}

	objValue := reflect.ValueOf(obj)
	if objValue.Kind() == reflect.Ptr {
		objValue = objValue.Elem()
	}

	return f.getFieldByPath(objValue, fieldPath)
}

// getFieldByPath navigates to a field using a dot-separated path like ".spec.name".
func (f *fieldValidator) getFieldByPath(objValue reflect.Value, fieldPath string) (interface{}, error) {
	// Remove leading dot if present
	if fieldPath[0] == '.' {
		fieldPath = fieldPath[1:]
	}

	// Split path into components
	parts := f.splitPath(fieldPath)
	current := objValue

	for _, part := range parts {
		if !current.IsValid() {
			return nil, fmt.Errorf("invalid value encountered in path")
		}

		// Handle pointer types
		if current.Kind() == reflect.Ptr {
			if current.IsNil() {
				return nil, nil // nil pointer, return nil value
			}
			current = current.Elem()
		}

		// Navigate to the field
		if current.Kind() != reflect.Struct {
			return nil, fmt.Errorf("cannot access field %s on non-struct type %s", part, current.Type())
		}

		field := current.FieldByName(f.capitalizeField(part))
		if !field.IsValid() {
			return nil, fmt.Errorf("field %s not found in type %s", part, current.Type())
		}

		current = field
	}

	// Return the interface{} value
	if !current.IsValid() {
		return nil, nil
	}

	return current.Interface(), nil
}

// splitPath splits a field path into components.
func (f *fieldValidator) splitPath(path string) []string {
	if path == "" {
		return []string{}
	}

	var parts []string
	current := ""

	for _, r := range path {
		if r == '.' {
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

// capitalizeField capitalizes the first letter of a field name for Go struct field access.
func (f *fieldValidator) capitalizeField(field string) string {
	if len(field) == 0 {
		return field
	}

	// Capitalize first letter to match Go struct field naming
	return strings.ToUpper(string(field[0])) + field[1:]
}

// basicStrategy is a simple validation strategy for demonstration.
type basicStrategy struct {
	supportedTypes map[reflect.Type]bool
	validateFunc   func(ctx context.Context, obj runtime.Object) []ValidationError
}

// NewBasicStrategy creates a basic validation strategy with a custom function.
func NewBasicStrategy(validateFunc func(ctx context.Context, obj runtime.Object) []ValidationError, supportedTypes ...reflect.Type) ValidationStrategy {
	typeMap := make(map[reflect.Type]bool)
	for _, t := range supportedTypes {
		typeMap[t] = true
	}

	return &basicStrategy{
		supportedTypes: typeMap,
		validateFunc:   validateFunc,
	}
}

func (s *basicStrategy) Execute(ctx context.Context, obj runtime.Object) []ValidationError {
	if s.validateFunc == nil {
		return nil
	}
	return s.validateFunc(ctx, obj)
}

func (s *basicStrategy) SupportsType(obj runtime.Object) bool {
	if len(s.supportedTypes) == 0 {
		return true // Support all types if none specified
	}

	objType := reflect.TypeOf(obj)
	return s.supportedTypes[objType]
}
