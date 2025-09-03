package validation

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// manager is the default implementation of ValidationManager.
type manager struct {
	mu          sync.RWMutex
	strategies  map[schema.GroupVersionKind][]ValidationStrategy
	validations map[schema.GroupVersionKind]*ObjectValidation
	options     ValidationOptions
	fieldVal    *fieldValidator
}

// NewManager creates a new validation manager with optional configuration.
func NewManager(opts ...ValidationOption) ValidationManager {
	options := ValidationOptions{
		FailFast:            false,
		IgnoreUnknownFields: false,
		AllowEmptyObjects:   false,
		MaxErrors:           10,
		Strict:              false,
	}

	for _, opt := range opts {
		opt(&options)
	}

	mgr := &manager{
		strategies:  make(map[schema.GroupVersionKind][]ValidationStrategy),
		validations: make(map[schema.GroupVersionKind]*ObjectValidation),
		options:     options,
	}

	mgr.fieldVal = &fieldValidator{manager: mgr}
	return mgr
}

// ValidationOption allows for functional configuration of the validation manager.
type ValidationOption func(*ValidationOptions)

// WithFailFast sets the fail-fast option.
func WithFailFast(enabled bool) ValidationOption {
	return func(opts *ValidationOptions) {
		opts.FailFast = enabled
	}
}

// WithMaxErrors sets the maximum number of errors to return.
func WithMaxErrors(max int) ValidationOption {
	return func(opts *ValidationOptions) {
		opts.MaxErrors = max
	}
}

// WithStrict enables strict validation mode.
func WithStrict(enabled bool) ValidationOption {
	return func(opts *ValidationOptions) {
		opts.Strict = enabled
	}
}

// Validate validates an object for creation or update.
func (m *manager) Validate(ctx context.Context, obj runtime.Object) error {
	if obj == nil {
		return fmt.Errorf("cannot validate nil object")
	}

	errors := m.validateObject(ctx, obj, nil)
	if len(errors) == 0 {
		return nil
	}

	// Convert to error
	return &ValidationErrors{Errors: errors}
}

// ValidateUpdate validates an object for update, comparing with the old object.
func (m *manager) ValidateUpdate(ctx context.Context, obj, old runtime.Object) error {
	if obj == nil {
		return fmt.Errorf("cannot validate nil object")
	}

	errors := m.validateObject(ctx, obj, old)
	if len(errors) == 0 {
		return nil
	}

	return &ValidationErrors{Errors: errors}
}

// ValidateDelete validates an object for deletion.
func (m *manager) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	if obj == nil {
		return fmt.Errorf("cannot validate nil object")
	}

	// For deletion, we mainly check if the object can be safely deleted
	// This is typically handled by finalizers and admission controllers
	// but we can add custom deletion validation here
	return nil
}

// validateObject performs the actual validation logic.
func (m *manager) validateObject(ctx context.Context, obj runtime.Object, _ runtime.Object) []ValidationError {
	var allErrors []ValidationError

	gvk := obj.GetObjectKind().GroupVersionKind()
	if gvk.Empty() {
		gvk = m.inferGVK(obj)
	}

	m.mu.RLock()
	strategies := m.strategies[gvk]
	allStrategies := m.strategies[schema.GroupVersionKind{Group: "*", Version: "*", Kind: "*"}]
	objValidation := m.validations[gvk]
	m.mu.RUnlock()

	// Apply GVK-specific strategy-based validation
	for _, strategy := range strategies {
		if strategy.SupportsType(obj) {
			errors := strategy.Execute(ctx, obj)
			allErrors = append(allErrors, errors...)

			if m.options.FailFast && len(errors) > 0 {
				return allErrors
			}
		}
	}

	// Apply global strategy-based validation
	for _, strategy := range allStrategies {
		if strategy.SupportsType(obj) {
			errors := strategy.Execute(ctx, obj)
			allErrors = append(allErrors, errors...)

			if m.options.FailFast && len(errors) > 0 {
				return allErrors
			}
		}
	}

	// Apply kubebuilder marker-based validation
	if objValidation != nil {
		errors := m.applyObjectValidation(ctx, obj, objValidation)
		allErrors = append(allErrors, errors...)
	}

	// Limit number of errors if configured
	if m.options.MaxErrors > 0 && len(allErrors) > m.options.MaxErrors {
		allErrors = allErrors[:m.options.MaxErrors]
	}

	return allErrors
}

// RegisterStrategy registers a validation strategy for objects.
func (m *manager) RegisterStrategy(strategy ValidationStrategy) error {
	if strategy == nil {
		return fmt.Errorf("strategy cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Register strategy globally for now - could be enhanced to be GVK-specific
	allGVK := schema.GroupVersionKind{Group: "*", Version: "*", Kind: "*"}
	m.strategies[allGVK] = append(m.strategies[allGVK], strategy)

	return nil
}

// UnregisterStrategy removes a validation strategy.
func (m *manager) UnregisterStrategy(strategy ValidationStrategy) error {
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

// HasValidationFor returns true if validation is configured for the given object type.
func (m *manager) HasValidationFor(obj runtime.Object) bool {
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

	// Check for object validation rules
	_, hasValidation := m.validations[gvk]
	return hasValidation
}

// RegisterObjectValidation registers validation rules for a specific object type.
func (m *manager) RegisterObjectValidation(objValidation *ObjectValidation) error {
	if objValidation == nil {
		return fmt.Errorf("object validation cannot be nil")
	}

	if objValidation.GVK.Empty() {
		return fmt.Errorf("object validation must specify a valid GVK")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.validations[objValidation.GVK] = objValidation
	return nil
}

// applyObjectValidation applies the validation rules defined in ObjectValidation.
func (m *manager) applyObjectValidation(_ context.Context, obj runtime.Object, objValidation *ObjectValidation) []ValidationError {
	var errors []ValidationError

	for _, rule := range objValidation.Rules {
		ruleErrors := m.applyValidationRule(obj, rule)
		errors = append(errors, ruleErrors...)

		if m.options.FailFast && len(ruleErrors) > 0 {
			break
		}
	}

	return errors
}

// applyValidationRule applies a single validation rule to the object.
func (m *manager) applyValidationRule(obj runtime.Object, rule ValidationRule) []ValidationError {
	// Get the field value
	fieldValue, err := m.fieldVal.GetFieldValue(obj, rule.Field)
	if err != nil {
		return []ValidationError{{
			Field:   rule.Field,
			Type:    ValidationErrorTypeInvalid,
			Message: fmt.Sprintf("cannot access field: %v", err),
		}}
	}

	switch rule.Type {
	case ValidationRuleTypeRequired:
		return m.validateRequired(rule.Field, fieldValue, rule.Message)
	case ValidationRuleTypeMinLength:
		return m.validateMinLength(rule.Field, fieldValue, rule.Value, rule.Message)
	case ValidationRuleTypeMaxLength:
		return m.validateMaxLength(rule.Field, fieldValue, rule.Value, rule.Message)
	case ValidationRuleTypeMinimum:
		return m.validateMinimum(rule.Field, fieldValue, rule.Value, rule.Message)
	case ValidationRuleTypeMaximum:
		return m.validateMaximum(rule.Field, fieldValue, rule.Value, rule.Message)
	case ValidationRuleTypeEnum:
		return m.validateEnum(rule.Field, fieldValue, rule.Value, rule.Message)
	case ValidationRuleTypePattern:
		return m.validatePattern(rule.Field, fieldValue, rule.Value, rule.Message)
	default:
		return []ValidationError{{
			Field:   rule.Field,
			Type:    ValidationErrorTypeInvalid,
			Message: fmt.Sprintf("unsupported validation rule type: %s", rule.Type),
		}}
	}
}

// Validation rule implementations

func (m *manager) validateRequired(field string, value interface{}, customMessage string) []ValidationError {
	if m.isZeroValue(value) {
		message := customMessage
		if message == "" {
			message = "field is required"
		}
		return []ValidationError{{
			Field:   field,
			Type:    ValidationErrorTypeRequired,
			Message: message,
			Value:   value,
		}}
	}
	return nil
}

func (m *manager) validateMinLength(field string, value interface{}, minLen interface{}, customMessage string) []ValidationError {
	str, ok := value.(string)
	if !ok {
		if value == nil {
			return nil // nil values are handled by required validation
		}
		return []ValidationError{{
			Field:   field,
			Type:    ValidationErrorTypeInvalid,
			Message: "value must be a string",
			Value:   value,
		}}
	}

	min, err := m.parseIntValue(minLen)
	if err != nil {
		return []ValidationError{{
			Field:   field,
			Type:    ValidationErrorTypeInvalid,
			Message: fmt.Sprintf("invalid min length parameter: %v", err),
		}}
	}

	if len(str) < min {
		message := customMessage
		if message == "" {
			message = fmt.Sprintf("string must be at least %d characters long", min)
		}
		return []ValidationError{{
			Field:   field,
			Type:    ValidationErrorTypeTooShort,
			Message: message,
			Value:   value,
		}}
	}
	return nil
}

func (m *manager) validateMaxLength(field string, value interface{}, maxLen interface{}, customMessage string) []ValidationError {
	str, ok := value.(string)
	if !ok {
		if value == nil {
			return nil // nil values are handled by required validation
		}
		return []ValidationError{{
			Field:   field,
			Type:    ValidationErrorTypeInvalid,
			Message: "value must be a string",
			Value:   value,
		}}
	}

	max, err := m.parseIntValue(maxLen)
	if err != nil {
		return []ValidationError{{
			Field:   field,
			Type:    ValidationErrorTypeInvalid,
			Message: fmt.Sprintf("invalid max length parameter: %v", err),
		}}
	}

	if len(str) > max {
		message := customMessage
		if message == "" {
			message = fmt.Sprintf("string must be at most %d characters long", max)
		}
		return []ValidationError{{
			Field:   field,
			Type:    ValidationErrorTypeTooLong,
			Message: message,
			Value:   value,
		}}
	}
	return nil
}

func (m *manager) validateMinimum(field string, value interface{}, minVal interface{}, customMessage string) []ValidationError {
	num, err := m.parseNumericValue(value)
	if err != nil {
		if value == nil {
			return nil // nil values are handled by required validation
		}
		return []ValidationError{{
			Field:   field,
			Type:    ValidationErrorTypeInvalid,
			Message: fmt.Sprintf("value must be numeric: %v", err),
			Value:   value,
		}}
	}

	min, err := m.parseNumericValue(minVal)
	if err != nil {
		return []ValidationError{{
			Field:   field,
			Type:    ValidationErrorTypeInvalid,
			Message: fmt.Sprintf("invalid minimum parameter: %v", err),
		}}
	}

	if num < min {
		message := customMessage
		if message == "" {
			message = fmt.Sprintf("value must be at least %v", minVal)
		}
		return []ValidationError{{
			Field:   field,
			Type:    ValidationErrorTypeRange,
			Message: message,
			Value:   value,
		}}
	}
	return nil
}

func (m *manager) validateMaximum(field string, value interface{}, maxVal interface{}, customMessage string) []ValidationError {
	num, err := m.parseNumericValue(value)
	if err != nil {
		if value == nil {
			return nil // nil values are handled by required validation
		}
		return []ValidationError{{
			Field:   field,
			Type:    ValidationErrorTypeInvalid,
			Message: fmt.Sprintf("value must be numeric: %v", err),
			Value:   value,
		}}
	}

	max, err := m.parseNumericValue(maxVal)
	if err != nil {
		return []ValidationError{{
			Field:   field,
			Type:    ValidationErrorTypeInvalid,
			Message: fmt.Sprintf("invalid maximum parameter: %v", err),
		}}
	}

	if num > max {
		message := customMessage
		if message == "" {
			message = fmt.Sprintf("value must be at most %v", maxVal)
		}
		return []ValidationError{{
			Field:   field,
			Type:    ValidationErrorTypeRange,
			Message: message,
			Value:   value,
		}}
	}
	return nil
}

func (m *manager) validateEnum(field string, value interface{}, enumVal interface{}, customMessage string) []ValidationError {
	if value == nil {
		return nil // nil values are handled by required validation
	}

	// Parse enum values - expect semicolon-separated string
	enumStr, ok := enumVal.(string)
	if !ok {
		return []ValidationError{{
			Field:   field,
			Type:    ValidationErrorTypeInvalid,
			Message: "enum parameter must be a string",
		}}
	}

	allowedValues := strings.Split(enumStr, ";")
	valueStr := fmt.Sprintf("%v", value)

	for _, allowed := range allowedValues {
		if strings.TrimSpace(allowed) == valueStr {
			return nil // Valid enum value
		}
	}

	message := customMessage
	if message == "" {
		message = fmt.Sprintf("value must be one of: %s", enumStr)
	}
	return []ValidationError{{
		Field:   field,
		Type:    ValidationErrorTypeEnum,
		Message: message,
		Value:   value,
	}}
}

func (m *manager) validatePattern(field string, value interface{}, pattern interface{}, customMessage string) []ValidationError {
	str, ok := value.(string)
	if !ok {
		if value == nil {
			return nil // nil values are handled by required validation
		}
		return []ValidationError{{
			Field:   field,
			Type:    ValidationErrorTypeInvalid,
			Message: "value must be a string for pattern validation",
			Value:   value,
		}}
	}

	patternStr, ok := pattern.(string)
	if !ok {
		return []ValidationError{{
			Field:   field,
			Type:    ValidationErrorTypeInvalid,
			Message: "pattern parameter must be a string",
		}}
	}

	regex, err := regexp.Compile(patternStr)
	if err != nil {
		return []ValidationError{{
			Field:   field,
			Type:    ValidationErrorTypeInvalid,
			Message: fmt.Sprintf("invalid regex pattern: %v", err),
		}}
	}

	if !regex.MatchString(str) {
		message := customMessage
		if message == "" {
			message = fmt.Sprintf("value does not match pattern: %s", patternStr)
		}
		return []ValidationError{{
			Field:   field,
			Type:    ValidationErrorTypeFormat,
			Message: message,
			Value:   value,
		}}
	}
	return nil
}

// Helper methods

func (m *manager) isZeroValue(value interface{}) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func, reflect.Interface:
		return v.IsNil()
	default:
		zero := reflect.Zero(v.Type())
		return reflect.DeepEqual(value, zero.Interface())
	}
}

func (m *manager) parseIntValue(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("cannot convert %T to int", value)
	}
}

func (m *manager) parseNumericValue(value interface{}) (float64, error) {
	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to numeric value", value)
	}
}

func (m *manager) inferGVK(obj runtime.Object) schema.GroupVersionKind {
	objType := reflect.TypeOf(obj)
	if objType.Kind() == reflect.Ptr {
		objType = objType.Elem()
	}

	// Extract package and type name to construct GVK
	pkg := objType.PkgPath()
	name := objType.Name()

	var group, version string
	if pkg != "" && name != "" {
		if len(pkg) > 0 {
			parts := strings.Split(pkg, "/")
			if len(parts) > 0 {
				version = parts[len(parts)-1] // Last part is version
				group = "inventory.k1s.io"    // Hardcoded for our examples
			}
		}
	}

	return schema.GroupVersionKind{
		Group:   group,
		Version: version,
		Kind:    name,
	}
}

// ValidationErrors aggregates multiple validation errors.
type ValidationErrors struct {
	Errors []ValidationError
}

func (v *ValidationErrors) Error() string {
	if len(v.Errors) == 0 {
		return "validation failed"
	}

	if len(v.Errors) == 1 {
		return v.Errors[0].Error()
	}

	var messages []string
	for _, err := range v.Errors {
		messages = append(messages, err.Error())
	}
	return fmt.Sprintf("validation failed: %s", strings.Join(messages, "; "))
}

// IsValidationError checks if an error is a validation error.
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationErrors)
	return ok
}

// GetValidationErrors extracts validation errors from an error.
func GetValidationErrors(err error) []ValidationError {
	if verr, ok := err.(*ValidationErrors); ok {
		return verr.Errors
	}
	return nil
}
