package validation

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
)

// celValidator implements CELValidator interface.
// This is a simplified implementation without the full CEL library.
// In a production system, you would use github.com/google/cel-go.
type celValidator struct{}

// NewCELValidator creates a new CEL validator.
func NewCELValidator() CELValidator {
	return &celValidator{}
}

// ValidateCEL evaluates a CEL expression against an object.
// This is a simplified implementation that doesn't actually evaluate CEL.
// In a real implementation, you would use the CEL library.
func (c *celValidator) ValidateCEL(ctx context.Context, obj runtime.Object, expression string) error {
	if obj == nil {
		return fmt.Errorf("cannot evaluate CEL expression on nil object")
	}

	if expression == "" {
		return fmt.Errorf("CEL expression cannot be empty")
	}

	// For now, just validate that the expression looks like a CEL expression
	// Real implementation would compile and evaluate the expression
	if len(expression) < 3 {
		return fmt.Errorf("CEL expression too short: %s", expression)
	}

	// Simulate validation success for basic expressions
	// In a real implementation, this would:
	// 1. Compile the CEL expression
	// 2. Create a CEL environment with the object's fields
	// 3. Evaluate the expression and return the result
	return nil
}

// CompileCEL compiles a CEL expression for efficient reuse.
func (c *celValidator) CompileCEL(expression string) (CompiledCELExpression, error) {
	if expression == "" {
		return nil, fmt.Errorf("CEL expression cannot be empty")
	}

	// Return a compiled expression wrapper
	return &compiledCELExpression{
		expression: expression,
	}, nil
}

// compiledCELExpression implements CompiledCELExpression interface.
type compiledCELExpression struct {
	expression string
}

// Evaluate executes the compiled CEL expression against an object.
func (c *compiledCELExpression) Evaluate(ctx context.Context, obj runtime.Object) (bool, error) {
	if obj == nil {
		return false, fmt.Errorf("cannot evaluate CEL expression on nil object")
	}

	// Simplified evaluation - in reality this would use the compiled CEL program
	// For demonstration, we'll consider all expressions as valid
	return true, nil
}

// String returns the original CEL expression.
func (c *compiledCELExpression) String() string {
	return c.expression
}

// Example CEL validation strategy that uses CEL expressions.
type celValidationStrategy struct {
	expressions []CompiledCELExpression
	celVal      CELValidator
}

// NewCELValidationStrategy creates a validation strategy that uses CEL expressions.
func NewCELValidationStrategy(expressions ...string) (ValidationStrategy, error) {
	celVal := NewCELValidator()
	var compiled []CompiledCELExpression

	for _, expr := range expressions {
		compiledExpr, err := celVal.CompileCEL(expr)
		if err != nil {
			return nil, fmt.Errorf("failed to compile CEL expression %q: %w", expr, err)
		}
		compiled = append(compiled, compiledExpr)
	}

	return &celValidationStrategy{
		expressions: compiled,
		celVal:      celVal,
	}, nil
}

func (c *celValidationStrategy) Execute(ctx context.Context, obj runtime.Object) []ValidationError {
	var errors []ValidationError

	for _, expr := range c.expressions {
		result, err := expr.Evaluate(ctx, obj)
		if err != nil {
			errors = append(errors, ValidationError{
				Type:    ValidationErrorTypeInvalid,
				Message: fmt.Sprintf("CEL evaluation error: %v", err),
				Rule:    expr.String(),
			})
			continue
		}

		if !result {
			errors = append(errors, ValidationError{
				Type:    ValidationErrorTypeInvalid,
				Message: fmt.Sprintf("CEL expression failed: %s", expr.String()),
				Rule:    expr.String(),
			})
		}
	}

	return errors
}

func (c *celValidationStrategy) SupportsType(obj runtime.Object) bool {
	// CEL validation can work with any object type
	return true
}
