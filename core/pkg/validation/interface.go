package validation

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Validator provides the interface for validating objects.
// This interface is compatible with controller-runtime validation patterns.
type Validator interface {
	// Validate validates an object for creation or update.
	Validate(ctx context.Context, obj runtime.Object) error

	// ValidateUpdate validates an object for update, comparing with the old object.
	ValidateUpdate(ctx context.Context, obj, old runtime.Object) error

	// ValidateDelete validates an object for deletion.
	ValidateDelete(ctx context.Context, obj runtime.Object) error
}

// ValidationStrategy defines how to validate objects for a specific type.
// Implementations should be thread-safe as they may be called concurrently.
type ValidationStrategy interface {
	// Execute performs validation on the object and returns validation errors.
	Execute(ctx context.Context, obj runtime.Object) []ValidationError

	// SupportsType returns true if this strategy can handle the given object type.
	SupportsType(obj runtime.Object) bool
}

// ValidationManager coordinates multiple validation strategies and provides
// a unified interface for validating objects.
type ValidationManager interface {
	Validator

	// RegisterStrategy registers a validation strategy for a specific type.
	RegisterStrategy(strategy ValidationStrategy) error

	// UnregisterStrategy removes a validation strategy.
	UnregisterStrategy(strategy ValidationStrategy) error

	// HasValidationFor returns true if validation is configured for the given object type.
	HasValidationFor(obj runtime.Object) bool

	// RegisterObjectValidation registers validation rules for a specific object type.
	RegisterObjectValidation(objValidation *ObjectValidation) error
}

// FieldValidator provides fine-grained validation for individual fields.
type FieldValidator interface {
	// ValidateField validates a specific field in the object.
	// The fieldPath parameter specifies the JSON path to the field.
	ValidateField(ctx context.Context, obj runtime.Object, fieldPath string) []ValidationError

	// GetFieldValue retrieves the value of a field from an object.
	GetFieldValue(obj runtime.Object, fieldPath string) (interface{}, error)
}

// ValidationError represents a validation error with context information.
type ValidationError struct {
	// Field is the JSON path to the field that failed validation
	Field string

	// Value is the invalid value that caused the error
	Value interface{}

	// Type describes the type of validation that failed
	Type ValidationErrorType

	// Message is a human-readable error message
	Message string

	// Rule is the validation rule that was violated (optional)
	Rule string

	// Code is a machine-readable error code (optional)
	Code string
}

// Error implements the error interface.
func (v ValidationError) Error() string {
	if v.Field != "" {
		return v.Field + ": " + v.Message
	}
	return v.Message
}

// ValidationErrorType defines the type of validation error.
type ValidationErrorType string

const (
	// ValidationErrorTypeRequired indicates a required field is missing
	ValidationErrorTypeRequired ValidationErrorType = "Required"

	// ValidationErrorTypeInvalid indicates a field has an invalid value
	ValidationErrorTypeInvalid ValidationErrorType = "Invalid"

	// ValidationErrorTypeForbidden indicates a field value is not allowed
	ValidationErrorTypeForbidden ValidationErrorType = "Forbidden"

	// ValidationErrorTypeTooLong indicates a string field is too long
	ValidationErrorTypeTooLong ValidationErrorType = "TooLong"

	// ValidationErrorTypeTooShort indicates a string field is too short
	ValidationErrorTypeTooShort ValidationErrorType = "TooShort"

	// ValidationErrorTypeTooMany indicates an array field has too many items
	ValidationErrorTypeTooMany ValidationErrorType = "TooMany"

	// ValidationErrorTypeTooFew indicates an array field has too few items
	ValidationErrorTypeTooFew ValidationErrorType = "TooFew"

	// ValidationErrorTypeFormat indicates a field has an invalid format
	ValidationErrorTypeFormat ValidationErrorType = "Format"

	// ValidationErrorTypeEnum indicates a field value is not in the allowed enum
	ValidationErrorTypeEnum ValidationErrorType = "Enum"

	// ValidationErrorTypeRange indicates a numeric field is out of range
	ValidationErrorTypeRange ValidationErrorType = "Range"
)

// ValidationRule represents a single validation rule extracted from kubebuilder markers.
type ValidationRule struct {
	// Field is the JSON path to the field (e.g., ".spec.name")
	Field string

	// Type is the type of validation rule
	Type ValidationRuleType

	// Value is the parameter for the validation rule (e.g., "100" for MaxLength)
	Value interface{}

	// Message is a custom error message (optional)
	Message string

	// Condition specifies when this validation should be applied (optional)
	Condition string
}

// ValidationRuleType defines the type of validation rule.
type ValidationRuleType string

const (
	// ValidationRuleTypeRequired indicates a field is required
	ValidationRuleTypeRequired ValidationRuleType = "Required"

	// ValidationRuleTypeMinLength indicates minimum string length
	ValidationRuleTypeMinLength ValidationRuleType = "MinLength"

	// ValidationRuleTypeMaxLength indicates maximum string length
	ValidationRuleTypeMaxLength ValidationRuleType = "MaxLength"

	// ValidationRuleTypeMinimum indicates minimum numeric value
	ValidationRuleTypeMinimum ValidationRuleType = "Minimum"

	// ValidationRuleTypeMaximum indicates maximum numeric value
	ValidationRuleTypeMaximum ValidationRuleType = "Maximum"

	// ValidationRuleTypeEnum indicates allowed values
	ValidationRuleTypeEnum ValidationRuleType = "Enum"

	// ValidationRuleTypePattern indicates regex pattern validation
	ValidationRuleTypePattern ValidationRuleType = "Pattern"

	// ValidationRuleTypeFormat indicates format validation
	ValidationRuleTypeFormat ValidationRuleType = "Format"

	// ValidationRuleTypeCEL indicates CEL expression validation
	ValidationRuleTypeCEL ValidationRuleType = "CEL"

	// ValidationRuleTypeUnique indicates uniqueness validation
	ValidationRuleTypeUnique ValidationRuleType = "Unique"
)

// ObjectValidation contains all validation rules for a specific object type.
type ObjectValidation struct {
	// GVK identifies the object type these validations apply to
	GVK schema.GroupVersionKind

	// Rules is a list of validation rules for this object type
	Rules []ValidationRule

	// OpenAPISchema is the OpenAPI v3 schema for additional validation (optional)
	OpenAPISchema map[string]interface{}
}

// CELValidator provides Common Expression Language (CEL) validation capabilities.
type CELValidator interface {
	// ValidateCEL evaluates a CEL expression against an object
	ValidateCEL(ctx context.Context, obj runtime.Object, expression string) error

	// ValidateCELValue evaluates a CEL expression against any value (for testing)
	ValidateCELValue(ctx context.Context, value interface{}, expression string) error

	// CompileCEL compiles a CEL expression for efficient reuse
	CompileCEL(expression string) (CompiledCELProgram, error)
}

// CompiledCELProgram represents a compiled CEL program for efficient execution.
// This replaces CompiledCELExpression for clarity.
type CompiledCELProgram interface {
	// Eval executes the compiled CEL expression against a value
	Eval(ctx context.Context, value interface{}) (bool, error)
}

// ValidationOptions provides configuration options for validation behavior.
type ValidationOptions struct {
	// FailFast stops validation on the first error if true
	FailFast bool

	// IgnoreUnknownFields ignores fields not defined in the schema
	IgnoreUnknownFields bool

	// AllowEmptyObjects allows objects with no required fields set
	AllowEmptyObjects bool

	// MaxErrors limits the number of validation errors returned
	MaxErrors int

	// Strict enables strict validation mode with additional checks
	Strict bool
}
