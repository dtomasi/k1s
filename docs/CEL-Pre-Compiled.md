# CEL Pre-Compilation for K1S CLI-Gen

## Why Pre-Compile CEL?

CLI tools need <100ms startup time. Runtime CEL compilation takes 5-10ms per expression. Pre-compilation reduces this to ~0.1ms by doing compilation at build time and embedding bytecode.

## How It Works

### 1. Build-Time Compilation
- cli-gen parses kubebuilder CEL markers: `+kubebuilder:validation:XValidation:rule="self.quantity >= 0"`
- Compiles CEL expression using google/cel-go
- Serializes compiled AST to protobuf bytes
- Embeds bytes as Go byte arrays in generated code

### 2. Runtime Execution
- Generated init() function deserializes bytes back to CEL programs
- Validation functions call pre-compiled programs directly
- No compilation overhead at runtime

## Implementation

### CEL Compiler
```go
// tools/pkg/generators/cel_compiler.go
type CELCompiler struct {
    env *cel.Env
}

func (c *CELCompiler) CompileExpression(expression string) (*CompiledCELExpression, error) {
    ast, issues := c.env.Compile(expression)
    if issues != nil && issues.Err() != nil {
        return nil, issues.Err()
    }
    
    serialized, err := proto.Marshal(ast.Expr())
    if err != nil {
        return nil, err
    }
    
    return &CompiledCELExpression{
        OriginalExpression: expression,
        SerializedExpr:    serialized,
    }, nil
}
```

### Code Generation Template
```go
// Generate this pattern:
var celValidation0Expr = []byte{0x08, 0x01, 0x1a, 0x2f...} // serialized AST
var celValidation0Program cel.Program

func init() {
    var expr exprpb.Expr
    proto.Unmarshal(celValidation0Expr, &expr)
    ast := cel.ParsedExprToAst(&exprpb.ParsedExpr{Expr: &expr})
    celValidation0Program, _ = env.Program(ast)
}

func ValidateItem(obj *Item) error {
    result, _, err := celValidation0Program.Eval(map[string]interface{}{
        "self": obj.Spec.Quantity,
    })
    return checkResult(result, err)
}
```

### Integration with Existing Generator
Add CEL compilation to existing validation generator. Parse XValidation markers, compile expressions, generate pre-compiled validation functions.

## Key Points
- Catch CEL syntax errors at build time, not runtime
- 50-100x performance improvement for validation
- Essential for CLI <100ms startup requirement
- Standard pattern used by Kubernetes CRDs