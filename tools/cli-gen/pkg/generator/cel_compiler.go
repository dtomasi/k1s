package generator

import (
	"fmt"
	"hash/fnv"
	"sort"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/ext"
	"google.golang.org/protobuf/proto"

	"github.com/dtomasi/k1s/tools/cli-gen/pkg/extractor"
)

// CompiledCELExpression represents a pre-compiled CEL expression
type CompiledCELExpression struct {
	// The original CEL expression string
	OriginalExpression string
	// The field this validation applies to
	FieldName string
	// The serialized compiled expression protobuf bytes
	SerializedExpr []byte
	// Hash-based variable name for the generated code
	VarName string
	// Error message to display when validation fails
	Message string
}

// CELCompiler compiles CEL expressions at build time for runtime performance
type CELCompiler struct {
	env *cel.Env
}

// NewCELCompiler creates a new CEL compiler with k8s-compatible environment
func NewCELCompiler() (*CELCompiler, error) {
	env, err := cel.NewEnv(
		// Standard library extensions (includes size function)
		ext.Strings(),
		ext.Math(),

		// Kubernetes-compatible variables
		cel.Variable("self", cel.AnyType),
		cel.Variable("oldSelf", cel.AnyType),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return &CELCompiler{env: env}, nil
}

// CompileExpression compiles a CEL expression to bytecode at build time
func (c *CELCompiler) CompileExpression(expression, fieldName, message string) (*CompiledCELExpression, error) {
	// Parse and type-check the expression
	ast, issues := c.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("CEL compilation failed for expression '%s': %w", expression, issues.Err())
	}

	// Serialize the compiled AST to protobuf bytes
	checkedExpr, err := cel.AstToCheckedExpr(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to convert AST to checked expression '%s': %w", expression, err)
	}
	serialized, err := proto.Marshal(checkedExpr)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize CEL expression '%s': %w", expression, err)
	}

	// Generate a unique variable name based on hash of expression + field
	varName := generateVarName(expression, fieldName)

	return &CompiledCELExpression{
		OriginalExpression: expression,
		FieldName:          fieldName,
		SerializedExpr:     serialized,
		VarName:            varName,
		Message:            message,
	}, nil
}

// CompileValidations compiles all CEL validations for a resource
func (c *CELCompiler) CompileValidations(validations map[string][]extractor.ValidationRule) ([]*CompiledCELExpression, error) {
	var compiled []*CompiledCELExpression

	// Process validations in deterministic order for consistent code generation
	var fieldNames []string
	for fieldName := range validations {
		fieldNames = append(fieldNames, fieldName)
	}
	sort.Strings(fieldNames)

	for _, fieldName := range fieldNames {
		rules := validations[fieldName]
		for _, rule := range rules {
			if rule.Type == "CEL" && rule.Rule != "" {
				expr, err := c.CompileExpression(rule.Rule, fieldName, rule.Message)
				if err != nil {
					return nil, fmt.Errorf("failed to compile CEL validation for field %s: %w", fieldName, err)
				}
				compiled = append(compiled, expr)
			}
		}
	}

	return compiled, nil
}

// generateVarName creates a unique variable name for a compiled CEL expression
func generateVarName(expression, fieldName string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(expression + ":" + fieldName)) // hash.Hash.Write never fails
	return fmt.Sprintf("celValidation%x", h.Sum32())
}

// FormatSerializedBytes formats byte array for Go code generation
func (c *CompiledCELExpression) FormatSerializedBytes() string {
	if len(c.SerializedExpr) == 0 {
		return "[]byte{}"
	}

	result := "[]byte{"
	for i, b := range c.SerializedExpr {
		if i > 0 {
			result += ", "
		}
		if i%16 == 0 && i > 0 {
			result += "\n\t\t"
		}
		result += fmt.Sprintf("0x%02x", b)
	}
	result += "}"

	return result
}

// GetProgramVarName returns the variable name for the CEL program
func (c *CompiledCELExpression) GetProgramVarName() string {
	return c.VarName + "Program"
}

// GetExprVarName returns the variable name for the serialized expression
func (c *CompiledCELExpression) GetExprVarName() string {
	return c.VarName + "Expr"
}
