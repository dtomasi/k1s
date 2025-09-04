// Package generator provides code generation functionality for k1s instrumentation
package generator

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/dtomasi/k1s/tools/pkg/extractor"
)

//go:embed templates/*.tmpl
var templates embed.FS

// Generator handles code generation from extracted resource information
type Generator struct {
	outputDir string
}

// NewGenerator creates a new code generator
func NewGenerator(outputDir string) *Generator {
	return &Generator{
		outputDir: outputDir,
	}
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

	// Generate resource metadata
	if err := g.generateResourceMetadata(resources); err != nil {
		return fmt.Errorf("failed to generate resource metadata: %w", err)
	}

	// Generate validation strategies
	if err := g.generateValidationStrategies(resources); err != nil {
		return fmt.Errorf("failed to generate validation strategies: %w", err)
	}

	// Generate print columns
	if err := g.generatePrintColumns(resources); err != nil {
		return fmt.Errorf("failed to generate print columns: %w", err)
	}

	// Generate defaulting strategies
	if err := g.generateDefaultingStrategies(resources); err != nil {
		return fmt.Errorf("failed to generate defaulting strategies: %w", err)
	}

	return nil
}

// generateResourceMetadata generates resource metadata lookup functions
func (g *Generator) generateResourceMetadata(resources []*extractor.ResourceInfo) error {
	return g.executeTemplateFromFile("zz_generated.resource_metadata.go", "templates/resource_metadata.go.tmpl", map[string]interface{}{
		"Resources": resources,
	})
}

// generateValidationStrategies generates validation strategy functions
func (g *Generator) generateValidationStrategies(resources []*extractor.ResourceInfo) error {
	return g.executeTemplateFromFile("zz_generated.validation_strategies.go", "templates/validation_strategies.go.tmpl", map[string]interface{}{
		"Resources": resources,
	})
}

// generatePrintColumns generates print column definitions
func (g *Generator) generatePrintColumns(resources []*extractor.ResourceInfo) error {
	return g.executeTemplateFromFile("zz_generated.print_columns.go", "templates/print_columns.go.tmpl", map[string]interface{}{
		"Resources": resources,
	})
}

// generateDefaultingStrategies generates defaulting strategy functions
func (g *Generator) generateDefaultingStrategies(resources []*extractor.ResourceInfo) error {
	return g.executeTemplateFromFile("zz_generated.defaulting_strategies.go", "templates/defaulting_strategies.go.tmpl", map[string]interface{}{
		"Resources": resources,
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

	fmt.Printf("Generated: %s\n", outputPath)
	return nil
}
