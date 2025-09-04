// Package generator provides code generation functionality for k1s instrumentation
package generator

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/dtomasi/k1s/tools/cli-gen/pkg/extractor"
)

//go:embed templates/*.tmpl
var templates embed.FS

// Generator handles code generation from extracted resource information
type Generator struct {
	outputDir         string
	enabledGenerators map[string]bool
	verbose           bool
}

// NewGenerator creates a new code generator
func NewGenerator(outputDir string) *Generator {
	return &Generator{
		outputDir:         outputDir,
		enabledGenerators: make(map[string]bool),
		verbose:           false,
	}
}

// SetVerbose enables or disables verbose output
func (g *Generator) SetVerbose(verbose bool) {
	g.verbose = verbose
}

// SetEnabledGenerators configures which generators should be enabled
func (g *Generator) SetEnabledGenerators(generators []string) {
	// If empty list provided, enable all generators
	if len(generators) == 0 {
		g.enabledGenerators = make(map[string]bool)
		return
	}

	// Clear existing configuration
	g.enabledGenerators = make(map[string]bool)

	// Enable specified generators
	for _, gen := range generators {
		g.enabledGenerators[gen] = true
	}
}

// isGeneratorEnabled checks if a specific generator is enabled
func (g *Generator) isGeneratorEnabled(generator string) bool {
	// If no specific generators configured, all are enabled
	if len(g.enabledGenerators) == 0 {
		return true
	}

	return g.enabledGenerators[generator]
}

// Generate creates k1s instrumentation code from resource information
func (g *Generator) Generate(resources []*extractor.ResourceInfo) error {
	if len(resources) == 0 {
		return fmt.Errorf("no resources to generate")
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(g.outputDir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Detect package name from output directory
	packageName, err := g.detectPackageName()
	if err != nil {
		// Fallback to "generated" if we can't detect the package
		packageName = "generated"
	}

	// Generate resource metadata (object generator)
	if g.isGeneratorEnabled("object") {
		if err := g.generateResourceMetadata(resources, packageName); err != nil {
			return fmt.Errorf("failed to generate resource metadata: %w", err)
		}
	}

	// Generate validation strategies
	if g.isGeneratorEnabled("validation") {
		if err := g.generateValidationStrategies(resources, packageName); err != nil {
			return fmt.Errorf("failed to generate validation strategies: %w", err)
		}
	}

	// Generate defaulting strategies
	if g.isGeneratorEnabled("defaulting") {
		if err := g.generateDefaultingStrategies(resources, packageName); err != nil {
			return fmt.Errorf("failed to generate defaulting strategies: %w", err)
		}
	}

	return nil
}

// generateResourceMetadata generates resource metadata lookup functions
func (g *Generator) generateResourceMetadata(resources []*extractor.ResourceInfo, packageName string) error {
	return g.executeTemplateFromFile("zz_generated.resource_metadata.go", "templates/resource_metadata.go.tmpl", map[string]interface{}{
		"Resources":   resources,
		"PackageName": packageName,
	})
}

// generateValidationStrategies generates validation strategy functions
func (g *Generator) generateValidationStrategies(resources []*extractor.ResourceInfo, packageName string) error {
	// Compile CEL expressions for optimal runtime performance
	var allCompiledCEL []*CompiledCELExpression

	celCompiler, err := NewCELCompiler()
	if err != nil {
		return fmt.Errorf("failed to create CEL compiler: %w", err)
	}

	for _, resource := range resources {
		if resource.Validations != nil {
			compiledCEL, err := celCompiler.CompileValidations(resource.Validations)
			if err != nil {
				return fmt.Errorf("failed to compile CEL validations for %s: %w", resource.Kind, err)
			}
			allCompiledCEL = append(allCompiledCEL, compiledCEL...)
		}
	}

	if g.verbose && len(allCompiledCEL) > 0 {
		fmt.Printf("Compiled %d CEL expressions for optimal runtime performance\n", len(allCompiledCEL))
	}

	return g.executeTemplateFromFile("zz_generated.validation_strategies.go", "templates/validation_strategies.go.tmpl", map[string]interface{}{
		"Resources":              resources,
		"PackageName":            packageName,
		"CompiledCELExpressions": allCompiledCEL,
	})
}

// generateDefaultingStrategies generates defaulting strategy functions
func (g *Generator) generateDefaultingStrategies(resources []*extractor.ResourceInfo, packageName string) error {
	return g.executeTemplateFromFile("zz_generated.defaulting_strategies.go", "templates/defaulting_strategies.go.tmpl", map[string]interface{}{
		"Resources":   resources,
		"PackageName": packageName,
	})
}

// executeTemplateFromFile executes a template from embed.FS and writes the output to a file
func (g *Generator) executeTemplateFromFile(filename, templatePath string, data interface{}) error {
	// Read template from embedded filesystem
	templateContent, err := templates.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	// Create custom functions for templates
	funcMap := template.FuncMap{
		"pluralize": func(s string) string {
			// Simple pluralization - can be enhanced
			lower := strings.ToLower(s)
			if strings.HasSuffix(lower, "y") {
				return lower[:len(lower)-1] + "ies"
			}
			if strings.HasSuffix(lower, "s") {
				return lower + "es"
			}
			return lower + "s"
		},
		"split":   strings.Split,
		"toLower": strings.ToLower,
		"or": func(a, b string) string {
			if a != "" {
				return a
			}
			return b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"len": func(v interface{}) int {
			if v == nil {
				return 0
			}
			rv := reflect.ValueOf(v)
			switch rv.Kind() {
			case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
				return rv.Len()
			default:
				return 0
			}
		},
	}

	tmpl, err := template.New(filepath.Base(templatePath)).Funcs(funcMap).Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Format the generated Go code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// If formatting fails, write the unformatted code for debugging
		fmt.Printf("Warning: failed to format generated code: %v\n", err)
		formatted = buf.Bytes()
	}

	outputPath := filepath.Join(g.outputDir, filename)
	if err := os.WriteFile(outputPath, formatted, 0600); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	if g.verbose {
		fmt.Printf("Generated: %s\n", outputPath)
	}
	return nil
}

// detectPackageName detects the Go package name from existing files in the output directory
func (g *Generator) detectPackageName() (string, error) {
	files, err := filepath.Glob(filepath.Join(g.outputDir, "*.go"))
	if err != nil {
		return "", err
	}

	fileSet := token.NewFileSet()
	for _, file := range files {
		// Skip generated files to avoid conflicts
		if strings.Contains(filepath.Base(file), "zz_generated") {
			continue
		}

		src, err := parser.ParseFile(fileSet, file, nil, parser.PackageClauseOnly)
		if err != nil {
			continue // Skip files that can't be parsed
		}

		if src.Name != nil {
			return src.Name.Name, nil
		}
	}

	return "", fmt.Errorf("could not detect package name from %s", g.outputDir)
}
