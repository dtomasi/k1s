package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dtomasi/k1s/tools/pkg/extractor"
)

func TestNewGenerator(t *testing.T) {
	outputDir := "/tmp/test"
	generator := NewGenerator(outputDir)

	if generator == nil {
		t.Error("NewGenerator returned nil")
		return
	}
	if generator.outputDir != outputDir {
		t.Errorf("Expected outputDir '%s', got '%s'", outputDir, generator.outputDir)
	}
}

func TestGenerator_Generate(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	generator := NewGenerator(tmpDir)

	// Create test resources
	resources := []*extractor.ResourceInfo{
		{
			Kind:       "TestKind",
			Name:       "testkind",
			Group:      "test.example.com",
			Version:    "v1",
			Plural:     "testkinds",
			Singular:   "testkind",
			ShortNames: []string{"tk"},
			Scope:      "Namespaced",
			HasStatus:  true,
			PrintColumns: []extractor.PrintColumn{
				{
					Name:        "Name",
					Type:        "string",
					JSONPath:    ".metadata.name",
					Description: "Name of the resource",
					Priority:    0,
				},
			},
			Validations: map[string][]extractor.ValidationRule{
				"Name": {
					{Type: "Required", Value: ""},
					{Type: "MinLength", Value: "1"},
				},
			},
			Defaults: map[string]string{
				"Status": "Active",
			},
		},
	}

	err := generator.Generate(resources)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check if files were created
	expectedFiles := []string{
		"zz_generated.resource_metadata.go",
		"zz_generated.validation_strategies.go",
		"zz_generated.print_columns.go",
		"zz_generated.defaulting_strategies.go",
	}

	for _, filename := range expectedFiles {
		filePath := filepath.Join(tmpDir, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", filename)
		}
	}

	// Check resource_metadata.go content
	content, err := os.ReadFile(filepath.Join(tmpDir, "zz_generated.resource_metadata.go")) // #nosec G304 - Test file in temp dir // #nosec G304 - Test file in temp dir
	if err != nil {
		t.Fatalf("Failed to read resource_metadata.go: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "TestKind") {
		t.Error("Generated file should contain TestKind")
	}
	if !strings.Contains(contentStr, "test.example.com") {
		t.Error("Generated file should contain test.example.com group")
	}
	if !strings.Contains(contentStr, `ShortNames: []string{"tk"}`) {
		t.Error("Generated file should contain shortNames")
	}
}

func TestGenerator_GenerateEmptyResources(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewGenerator(tmpDir)

	err := generator.Generate([]*extractor.ResourceInfo{})
	if err == nil {
		t.Error("Expected error for empty resources, got nil")
	}
	if !strings.Contains(err.Error(), "no resources to generate") {
		t.Errorf("Expected 'no resources to generate' error, got: %v", err)
	}
}

func TestGenerator_GenerateInvalidOutputDir(t *testing.T) {
	// Use a file path that can't be created as directory
	generator := NewGenerator("/dev/null/invalid")

	resources := []*extractor.ResourceInfo{
		{Kind: "Test", Name: "test"},
	}

	err := generator.Generate(resources)
	if err == nil {
		t.Error("Expected error for invalid output directory, got nil")
	}
}

func TestGenerator_ExecuteTemplateFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewGenerator(tmpDir)

	data := map[string]interface{}{
		"Resources": []*extractor.ResourceInfo{
			{
				Kind:    "TestResource",
				Group:   "test.example.com",
				Version: "v1",
			},
		},
	}

	// Test with valid template
	err := generator.executeTemplateFromFile("test_output.go", "templates/resource_metadata.go.tmpl", data)
	if err != nil {
		t.Errorf("executeTemplateFromFile failed: %v", err)
	}

	// Check if file was created
	outputPath := filepath.Join(tmpDir, "test_output.go")
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}

	// Test with invalid template path
	err = generator.executeTemplateFromFile("invalid_output.go", "templates/nonexistent.tmpl", data)
	if err == nil {
		t.Error("Expected error for nonexistent template, got nil")
	}
}

func TestGenerator_TemplateCustomFunctions(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewGenerator(tmpDir)

	// Test data that will exercise template functions
	resources := []*extractor.ResourceInfo{
		{
			Kind:    "TestKind",
			Name:    "testkind",
			Group:   "test.example.com",
			Version: "v1",
			// Test pluralization
			Plural:     "", // Should be auto-generated
			Singular:   "", // Should be auto-generated
			ShortNames: []string{"tk", "test"},
			HasStatus:  false,
		},
	}

	err := generator.Generate(resources)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check that pluralization worked
	content, err := os.ReadFile(filepath.Join(tmpDir, "zz_generated.resource_metadata.go")) // #nosec G304 - Test file in temp dir
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	contentStr := string(content)
	// Should contain auto-pluralized "testkinds"
	if !strings.Contains(contentStr, "testkinds") {
		t.Error("Template pluralize function should generate 'testkinds'")
	}
	// Should contain auto-lowercased singular
	if !strings.Contains(contentStr, "testkind") {
		t.Error("Template toLower function should generate 'testkind'")
	}
}

func TestGenerator_PrintColumnsGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewGenerator(tmpDir)

	resources := []*extractor.ResourceInfo{
		{
			Kind:    "TestResource",
			Group:   "test.example.com",
			Version: "v1",
			PrintColumns: []extractor.PrintColumn{
				{
					Name:        "Name",
					Type:        "string",
					JSONPath:    ".metadata.name",
					Description: "Resource name",
					Priority:    0,
				},
				{
					Name:        "Age",
					Type:        "date",
					JSONPath:    ".metadata.creationTimestamp",
					Description: "Age of resource",
					Priority:    1,
				},
			},
		},
	}

	err := generator.Generate(resources)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check print columns content
	content, err := os.ReadFile(filepath.Join(tmpDir, "zz_generated.print_columns.go")) // #nosec G304 - Test file in temp dir
	if err != nil {
		t.Fatalf("Failed to read print_columns.go: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, `Name:        "Name"`) {
		t.Error("Generated file should contain Name column")
	}
	if !strings.Contains(contentStr, `JSONPath:    ".metadata.name"`) {
		t.Error("Generated file should contain JSONPath")
	}
	if !strings.Contains(contentStr, `Type:        "date"`) {
		t.Error("Generated file should contain date type")
	}
}

func TestGenerator_ValidationStrategiesGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewGenerator(tmpDir)

	resources := []*extractor.ResourceInfo{
		{
			Kind:    "ValidatedResource",
			Group:   "test.example.com",
			Version: "v1",
			Validations: map[string][]extractor.ValidationRule{
				"Name": {
					{Type: "Required", Value: ""},
					{Type: "MinLength", Value: "3"},
				},
				"Age": {
					{Type: "Minimum", Value: "0"},
					{Type: "Maximum", Value: "120"},
				},
			},
		},
	}

	err := generator.Generate(resources)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check validation strategies content
	content, err := os.ReadFile(filepath.Join(tmpDir, "zz_generated.validation_strategies.go")) // #nosec G304 - Test file in temp dir
	if err != nil {
		t.Fatalf("Failed to read validation_strategies.go: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "ValidatedResourceValidationStrategy") {
		t.Error("Generated file should contain validation strategy struct")
	}
	if !strings.Contains(contentStr, `"ValidatedResource": &ValidatedResourceValidationStrategy{}`) {
		t.Error("Generated file should contain strategy registration")
	}
}

func TestGenerator_DefaultingStrategiesGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewGenerator(tmpDir)

	resources := []*extractor.ResourceInfo{
		{
			Kind:    "DefaultableResource",
			Group:   "test.example.com",
			Version: "v1",
			Defaults: map[string]string{
				"Status":   "Active",
				"Priority": "Medium",
				"Timeout":  "30",
			},
		},
	}

	err := generator.Generate(resources)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check defaulting strategies content
	content, err := os.ReadFile(filepath.Join(tmpDir, "zz_generated.defaulting_strategies.go")) // #nosec G304 - Test file in temp dir
	if err != nil {
		t.Fatalf("Failed to read defaulting_strategies.go: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "DefaultableResourceDefaultingStrategy") {
		t.Error("Generated file should contain defaulting strategy struct")
	}
	if !strings.Contains(contentStr, `"DefaultableResource": &DefaultableResourceDefaultingStrategy{}`) {
		t.Error("Generated file should contain strategy registration")
	}
	if !strings.Contains(contentStr, "ApplyDefaults") {
		t.Error("Generated file should contain ApplyDefaults method")
	}
}
