package validation

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"k8s.io/apimachinery/pkg/runtime"
)

// celValidator implements CELValidator interface using google/cel-go library.
type celValidator struct {
	env   *cel.Env
	cache map[string]CompiledCELProgram
	mu    sync.RWMutex
}

// compiledCELProgram implements CompiledCELProgram interface.
type compiledCELProgram struct {
	program    cel.Program
	expression string
}

// NewCELValidator creates a new CEL validator with a standard environment.
func NewCELValidator() CELValidator {
	env, err := cel.NewEnv(
		cel.Variable("self", cel.DynType),
		cel.HomogeneousAggregateLiterals(),
		cel.EagerlyValidateDeclarations(true),
		cel.DefaultUTCTimeZone(true),
	)
	if err != nil {
		// This should never happen with our standard configuration
		panic(fmt.Sprintf("failed to create CEL environment: %v", err))
	}

	return &celValidator{
		env:   env,
		cache: make(map[string]CompiledCELProgram),
	}
}

// ValidateCEL evaluates a CEL expression against an object.
func (c *celValidator) ValidateCEL(ctx context.Context, obj runtime.Object, expression string) error {
	if obj == nil {
		return fmt.Errorf("cannot evaluate CEL expression on nil object")
	}
	return c.ValidateCELValue(ctx, obj, expression)
}

// ValidateCELValue evaluates a CEL expression against any value.
func (c *celValidator) ValidateCELValue(ctx context.Context, value interface{}, expression string) error {
	if expression == "" {
		return fmt.Errorf("CEL expression cannot be empty")
	}

	// Compile the expression (with caching)
	compiled, err := c.CompileCEL(expression)
	if err != nil {
		return fmt.Errorf("failed to compile CEL expression: %w", err)
	}

	// Evaluate the expression
	result, err := compiled.Eval(ctx, value)
	if err != nil {
		return fmt.Errorf("CEL evaluation failed: %w", err)
	}

	if !result {
		return fmt.Errorf("CEL expression evaluated to false: %s", expression)
	}

	return nil
}

// CompileCEL compiles a CEL expression for efficient reuse with caching.
func (c *celValidator) CompileCEL(expression string) (CompiledCELProgram, error) {
	if expression == "" {
		return nil, fmt.Errorf("CEL expression cannot be empty")
	}

	// Check cache first
	c.mu.RLock()
	if cached, exists := c.cache[expression]; exists {
		c.mu.RUnlock()
		return cached, nil
	}
	c.mu.RUnlock()

	// Compile the expression
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check cache after acquiring write lock
	if cached, exists := c.cache[expression]; exists {
		return cached, nil
	}

	// Parse and check the expression
	ast, issues := c.env.Parse(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("failed to parse CEL expression '%s': %w", expression, issues.Err())
	}

	// Type-check the expression
	checked, issues := c.env.Check(ast)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("failed to type-check CEL expression '%s': %w", expression, issues.Err())
	}

	// Create the program
	program, err := c.env.Program(checked)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL program for '%s': %w", expression, err)
	}

	compiled := &compiledCELProgram{
		program:    program,
		expression: expression,
	}

	// Cache the compiled program
	c.cache[expression] = compiled

	return compiled, nil
}

// Eval executes the compiled CEL expression against a value.
func (p *compiledCELProgram) Eval(ctx context.Context, value interface{}) (bool, error) {
	// Handle nil pointers specially
	if value == nil {
		// For nil values, we only support expressions like "self == null"
		vars := map[string]interface{}{
			"self": nil,
		}
		result, _, err := p.program.Eval(vars)
		if err != nil {
			return false, fmt.Errorf("evaluation error on nil value: %w", err)
		}
		return p.convertToBool(result)
	}

	// For complex objects or unsupported types, try to convert to CEL-compatible format
	celValue := p.convertToValue(value)

	// Create evaluation variables
	vars := map[string]interface{}{
		"self": celValue,
	}

	// Execute the program
	result, _, err := p.program.Eval(vars)
	if err != nil {
		return false, fmt.Errorf("evaluation error: %w", err)
	}

	return p.convertToBool(result)
}

// convertToValue converts Go values to CEL-compatible values
func (p *compiledCELProgram) convertToValue(value interface{}) interface{} {
	// Handle nil pointers
	if value == nil {
		return nil
	}

	// Handle pointers by dereferencing them
	if reflect.ValueOf(value).Kind() == reflect.Ptr {
		if reflect.ValueOf(value).IsNil() {
			return nil
		}
		return reflect.ValueOf(value).Elem().Interface()
	}

	// For most basic types, CEL can handle them directly
	return value
}

// convertToBool converts CEL result to boolean
func (p *compiledCELProgram) convertToBool(result ref.Val) (bool, error) {
	// Convert result to boolean
	if result == nil {
		return false, fmt.Errorf("CEL expression returned nil")
	}

	// Handle CEL boolean values
	if result.Equal(types.True) == types.True {
		return true, nil
	}
	if result.Equal(types.False) == types.True {
		return false, nil
	}

	// Try to convert to native boolean if possible
	if boolVal, ok := result.(types.Bool); ok {
		return bool(boolVal), nil
	}

	return false, fmt.Errorf("CEL expression returned non-boolean result: %T", result)
}
