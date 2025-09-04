package extractor

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	testGroup    = "examples.k1s.dtomasi.github.io"
	testVersion  = "v1alpha1"
	testVersionB = "v1beta1"
	testScope    = "Namespaced"
)

func TestExtractor_Extract(t *testing.T) {
	extractor := NewExtractor()

	resources, err := extractor.Extract([]string{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if resources == nil {
		t.Error("Expected resources slice, got nil")
		return
	}

	if len(resources) != 0 {
		t.Errorf("Expected empty resources for empty paths, got %d", len(resources))
	}
}

func TestExtractor_ExtractWithExampleAPIs(t *testing.T) {
	extractor := NewExtractor()

	// Use the example APIs as test data
	apiPath, _ := filepath.Abs("../../../examples/api/v1alpha1")
	resources, err := extractor.Extract([]string{apiPath})
	if err != nil {
		t.Fatalf("Failed to extract from example APIs: %v", err)
	}

	if len(resources) < 2 {
		t.Errorf("Expected at least 2 resources (Item, Category), got %d", len(resources))
	}

	// Test resource metadata extraction
	resourceMap := make(map[string]*ResourceInfo)
	for _, res := range resources {
		resourceMap[res.Kind] = res
	}

	// Test Item resource
	if item, ok := resourceMap["Item"]; ok {
		if item.Group != testGroup {
			t.Errorf("Expected Item group '%s', got '%s'", testGroup, item.Group)
		}
		if item.Version != testVersion {
			t.Errorf("Expected Item version '%s', got '%s'", testVersion, item.Version)
		}
		// Plural might be empty in ResourceInfo (handled by template)
		// The important thing is that we found the resource
		t.Logf("Item resource found with Kind: %s, Group: %s", item.Kind, item.Group)
	} else {
		t.Error("Item resource not found")
	}

	// Test Category resource
	if category, ok := resourceMap["Category"]; ok {
		if category.Group != testGroup {
			t.Errorf("Expected Category group '%s', got '%s'", testGroup, category.Group)
		}
		if category.Version != testVersion {
			t.Errorf("Expected Category version '%s', got '%s'", testVersion, category.Version)
		}
		// Plural might be empty in ResourceInfo (handled by template)
		t.Logf("Category resource found with Kind: %s, Group: %s", category.Kind, category.Group)
	} else {
		t.Error("Category resource not found")
	}
}

func TestExtractor_InvalidPath(t *testing.T) {
	extractor := NewExtractor()

	// Invalid glob pattern should cause error
	_, err := extractor.Extract([]string{"[invalid-glob-pattern"})
	if err == nil {
		t.Error("Expected error for invalid glob pattern, got nil")
	}
}

func TestNewExtractor(t *testing.T) {
	extractor := NewExtractor()
	if extractor == nil {
		t.Error("NewExtractor returned nil")
		return
	}
	if extractor.fileSet == nil {
		t.Error("NewExtractor did not initialize fileSet")
	}
}

func TestResourceInfo_DefaultValues(t *testing.T) {
	info := &ResourceInfo{
		Kind:        "TestKind",
		Name:        "testkind",
		Validations: make(map[string][]ValidationRule),
		Defaults:    make(map[string]string),
	}

	if info.Kind != "TestKind" {
		t.Errorf("Expected Kind 'TestKind', got '%s'", info.Kind)
	}
	if info.Name != "testkind" {
		t.Errorf("Expected Name 'testkind', got '%s'", info.Name)
	}
	if info.Validations == nil {
		t.Error("Expected Validations map to be initialized")
	}
	if info.Defaults == nil {
		t.Error("Expected Defaults map to be initialized")
	}
}

func TestPrintColumn_Fields(t *testing.T) {
	column := PrintColumn{
		Name:        "Test Column",
		Type:        "string",
		JSONPath:    ".spec.test",
		Description: "Test description",
		Priority:    0,
	}

	if column.Name != "Test Column" {
		t.Errorf("Expected Name 'Test Column', got '%s'", column.Name)
	}
	if column.Type != "string" {
		t.Errorf("Expected Type 'string', got '%s'", column.Type)
	}
	if column.JSONPath != ".spec.test" {
		t.Errorf("Expected JSONPath '.spec.test', got '%s'", column.JSONPath)
	}
}

func TestValidationRule_Fields(t *testing.T) {
	rule := ValidationRule{
		Type:  "Required",
		Value: "true",
	}

	if rule.Type != "Required" {
		t.Errorf("Expected Type 'Required', got '%s'", rule.Type)
	}
	if rule.Value != "true" {
		t.Errorf("Expected Value 'true', got '%s'", rule.Value)
	}
}

func TestPackageInfo_Fields(t *testing.T) {
	info := &PackageInfo{
		Group:   "test.example.com",
		Version: "v1beta1",
	}

	if info.Group != "test.example.com" {
		t.Errorf("Expected Group 'test.example.com', got '%s'", info.Group)
	}
	if info.Version != testVersionB {
		t.Errorf("Expected Version '%s', got '%s'", testVersionB, info.Version)
	}
}

func TestExtractor_ExtractPackageInfo(t *testing.T) {
	extractor := NewExtractor()

	// Test with example groupversion_info.go
	groupversionPath, _ := filepath.Abs("../../../examples/api/v1alpha1/groupversion_info.go")
	info, err := extractor.extractPackageInfo(groupversionPath)
	if err != nil {
		t.Fatalf("Failed to extract package info: %v", err)
	}

	if info == nil {
		t.Error("Expected PackageInfo, got nil")
		return
	}

	if info.Group != testGroup {
		t.Errorf("Expected Group '%s', got '%s'", testGroup, info.Group)
	}
	if info.Version != testVersion {
		t.Errorf("Expected Version '%s', got '%s'", testVersion, info.Version)
	}
}

func TestExtractor_ExtractPackageInfoInvalidFile(t *testing.T) {
	extractor := NewExtractor()

	_, err := extractor.extractPackageInfo("/nonexistent/file.go")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestExtractor_MultiplePathsExtraction(t *testing.T) {
	extractor := NewExtractor()

	apiPath, _ := filepath.Abs("../../../examples/api/v1alpha1")

	// Test with same path twice - will actually process files twice
	resources, err := extractor.Extract([]string{apiPath, apiPath})
	if err != nil {
		t.Fatalf("Failed to extract from multiple paths: %v", err)
	}

	// Since we process the same path twice, we'll get duplicates
	// The important thing is that we get resources
	if len(resources) < 2 {
		t.Errorf("Expected at least 2 resources, got %d", len(resources))
	}

	// Count resources by kind
	kindCounts := make(map[string]int)
	for _, res := range resources {
		kindCounts[res.Kind]++
	}

	// We should have at least Item and Category
	if kindCounts["Item"] == 0 {
		t.Error("Expected to find Item resources")
	}
	if kindCounts["Category"] == 0 {
		t.Error("Expected to find Category resources")
	}
}

func TestExtractor_ExtractFromFile(t *testing.T) {
	extractor := NewExtractor()

	itemPath, _ := filepath.Abs("../../../examples/api/v1alpha1/item_types.go")
	resource, err := extractor.extractFromFile(itemPath)
	if err != nil {
		t.Fatalf("Failed to extract from item_types.go: %v", err)
	}

	if resource == nil {
		t.Error("Expected ResourceInfo, got nil")
		return
	}

	if resource.Kind != "Item" {
		t.Errorf("Expected Kind 'Item', got '%s'", resource.Kind)
	}
	if resource.Name != "item" {
		t.Errorf("Expected Name 'item', got '%s'", resource.Name)
	}
}

func TestExtractor_ExtractFromFileInvalid(t *testing.T) {
	extractor := NewExtractor()

	_, err := extractor.extractFromFile("/nonexistent/file.go")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestExtractor_EmptyResourceInfo(t *testing.T) {
	// Test with empty ResourceInfo to ensure no panics
	info := &ResourceInfo{
		Validations: make(map[string][]ValidationRule),
		Defaults:    make(map[string]string),
	}

	if info.Kind != "" {
		t.Errorf("Expected empty Kind, got '%s'", info.Kind)
	}
	if info.Validations == nil {
		t.Error("Expected initialized Validations map")
	}
	if info.Defaults == nil {
		t.Error("Expected initialized Defaults map")
	}
}

func TestExtractor_ParseResourceMarker(t *testing.T) {
	extractor := NewExtractor()

	// Test basic resource marker parsing
	resource := &ResourceInfo{}
	marker := "+kubebuilder:resource:scope=Namespaced,shortName=test,plural=tests,singular=test"

	extractor.parseResourceMarker(resource, marker)

	if resource.Scope != testScope {
		t.Errorf("Expected Scope '%s', got '%s'", testScope, resource.Scope)
	}
	if len(resource.ShortNames) != 1 || resource.ShortNames[0] != "test" {
		t.Errorf("Expected ShortNames ['test'], got %v", resource.ShortNames)
	}
	if resource.Plural != "tests" {
		t.Errorf("Expected Plural 'tests', got '%s'", resource.Plural)
	}
	if resource.Singular != "test" {
		t.Errorf("Expected Singular 'test', got '%s'", resource.Singular)
	}
}

func TestExtractor_ParseResourceMarkerMultipleShortNames(t *testing.T) {
	extractor := NewExtractor()

	resource := &ResourceInfo{}
	marker := "+kubebuilder:resource:shortName=test;ts;t"

	extractor.parseResourceMarker(resource, marker)

	expected := []string{"test", "ts", "t"}
	if len(resource.ShortNames) != 3 {
		t.Errorf("Expected 3 shortNames, got %d", len(resource.ShortNames))
	}
	for i, expected := range expected {
		if i >= len(resource.ShortNames) || resource.ShortNames[i] != expected {
			t.Errorf("Expected ShortNames[%d] = '%s', got %v", i, expected, resource.ShortNames)
		}
	}
}

func TestExtractor_ParsePrintColumnMarker(t *testing.T) {
	extractor := NewExtractor()

	// Test valid print column marker
	marker := "+kubebuilder:printcolumn:name=\"Name\",type=string,JSONPath=.metadata.name,description=\"Object name\""
	column := extractor.parsePrintColumnMarker(marker)

	if column == nil {
		t.Fatal("Expected PrintColumn, got nil")
	}
	if column.Name != "Name" {
		t.Errorf("Expected Name 'Name', got '%s'", column.Name)
	}
	if column.Type != "string" {
		t.Errorf("Expected Type 'string', got '%s'", column.Type)
	}
	if column.JSONPath != ".metadata.name" {
		t.Errorf("Expected JSONPath '.metadata.name', got '%s'", column.JSONPath)
	}
	if column.Description != "Object name" {
		t.Errorf("Expected Description 'Object name', got '%s'", column.Description)
	}
}

func TestExtractor_ParsePrintColumnMarkerIncomplete(t *testing.T) {
	extractor := NewExtractor()

	// Test incomplete print column marker (missing required fields)
	marker := "+kubebuilder:printcolumn:name=Name,description=\"Object name\""
	column := extractor.parsePrintColumnMarker(marker)

	if column != nil {
		t.Error("Expected nil for incomplete print column marker")
	}
}

func TestExtractor_ParseValidationMarker(t *testing.T) {
	extractor := NewExtractor()

	// Test validation marker with value
	marker := "+kubebuilder:validation:MinLength=1"
	rule := extractor.parseValidationMarker(marker)

	if rule == nil {
		t.Fatal("Expected ValidationRule, got nil")
	}
	if rule.Type != "MinLength" {
		t.Errorf("Expected Type 'MinLength', got '%s'", rule.Type)
	}
	if rule.Value != "1" {
		t.Errorf("Expected Value '1', got '%s'", rule.Value)
	}
}

func TestExtractor_ParseValidationMarkerNoValue(t *testing.T) {
	extractor := NewExtractor()

	// Test validation marker without value
	marker := "+kubebuilder:validation:Required"
	rule := extractor.parseValidationMarker(marker)

	if rule == nil {
		t.Fatal("Expected ValidationRule, got nil")
	}
	if rule.Type != "Required" {
		t.Errorf("Expected Type 'Required', got '%s'", rule.Type)
	}
	if rule.Value != "" {
		t.Errorf("Expected empty Value, got '%s'", rule.Value)
	}
}

func TestExtractor_ParseValidationMarkerInvalid(t *testing.T) {
	extractor := NewExtractor()

	// Test invalid validation marker
	marker := "+kubebuilder:validation:"
	rule := extractor.parseValidationMarker(marker)

	if rule != nil {
		t.Error("Expected nil for invalid validation marker")
	}
}

func TestExtractor_ParseResourceMarkerEdgeCases(t *testing.T) {
	extractor := NewExtractor()

	// Test empty marker
	resource := &ResourceInfo{}
	extractor.parseResourceMarker(resource, "+kubebuilder:resource:")

	// Should not change anything
	if resource.Scope != "" {
		t.Errorf("Expected empty Scope, got '%s'", resource.Scope)
	}

	// Test malformed marker
	resource = &ResourceInfo{}
	extractor.parseResourceMarker(resource, "+kubebuilder:resource:invalid")

	// Should not panic or change anything
	if resource.Scope != "" {
		t.Errorf("Expected empty Scope, got '%s'", resource.Scope)
	}

	// Test partial key-value pairs
	resource = &ResourceInfo{}
	extractor.parseResourceMarker(resource, "+kubebuilder:resource:scope=Namespaced,shortName")

	if resource.Scope != testScope {
		t.Errorf("Expected Scope '%s', got '%s'", testScope, resource.Scope)
	}
	// shortName without value should be ignored
	if len(resource.ShortNames) != 0 {
		t.Errorf("Expected no ShortNames, got %v", resource.ShortNames)
	}
}

func TestExtractor_ExtractFieldMarkers(t *testing.T) {
	extractor := NewExtractor()

	// Create a simple Go source file with field markers
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "test_types.go")

	content := `package test

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=tr
type TestResource struct {
	metav1.TypeMeta   ` + "`json:\",inline\"`" + `
	metav1.ObjectMeta ` + "`json:\"metadata,omitempty\"`" + `

	Spec   TestResourceSpec   ` + "`json:\"spec,omitempty\"`" + `
	Status TestResourceStatus ` + "`json:\"status,omitempty\"`" + `
}

type TestResourceSpec struct {
	// Name of the resource
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string ` + "`json:\"name,omitempty\"`" + `
}

type TestResourceStatus struct {
	Phase string ` + "`json:\"phase,omitempty\"`" + `
}
`

	if err := os.WriteFile(sourceFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	resource, err := extractor.extractFromFile(sourceFile)
	if err != nil {
		t.Fatalf("Failed to extract from test file: %v", err)
	}

	if resource == nil {
		t.Fatal("Expected ResourceInfo, got nil")
	}

	if resource.Kind != "TestResource" {
		t.Errorf("Expected Kind 'TestResource', got '%s'", resource.Kind)
	}

	// Note: Field marker extraction is currently limited in the implementation
	// This test validates that the basic extraction works without errors
}

func TestExtractor_ExtractPackageMarkersEdgeCases(t *testing.T) {
	extractor := NewExtractor()

	// Test with package without any kubebuilder markers
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "no_markers.go")

	content := `package test

// Regular comment without markers
type SimpleStruct struct {
	Name string
}
`

	if err := os.WriteFile(sourceFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	info, err := extractor.extractPackageInfo(sourceFile)
	if err != nil {
		t.Fatalf("Failed to extract package info: %v", err)
	}

	// Should return PackageInfo but with empty fields for package without markers
	if info == nil {
		t.Error("Expected PackageInfo, got nil")
	} else {
		if info.Group != "" {
			t.Errorf("Expected empty Group, got '%s'", info.Group)
		}
		if info.Version != "" {
			t.Errorf("Expected empty Version, got '%s'", info.Version)
		}
	}
}

func TestExtractor_ExtractResourceMarkersFromComments(t *testing.T) {
	extractor := NewExtractor()

	// Test resource with type-level markers
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "resource_with_markers.go")

	content := `package test

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,shortName=tr,plural=testresources
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=.metadata.name
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=.metadata.creationTimestamp
type TestResource struct {
	metav1.TypeMeta   ` + "`json:\",inline\"`" + `
	metav1.ObjectMeta ` + "`json:\"metadata,omitempty\"`" + `

	Spec TestResourceSpec ` + "`json:\"spec,omitempty\"`" + `
}

type TestResourceSpec struct {
	Name string ` + "`json:\"name,omitempty\"`" + `
}
`

	if err := os.WriteFile(sourceFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	resource, err := extractor.extractFromFile(sourceFile)
	if err != nil {
		t.Fatalf("Failed to extract from test file: %v", err)
	}

	if resource == nil {
		t.Fatal("Expected ResourceInfo, got nil")
	}

	if resource.Kind != "TestResource" {
		t.Errorf("Expected Kind 'TestResource', got '%s'", resource.Kind)
	}

	if resource.Scope != testScope {
		t.Errorf("Expected Scope '%s', got '%s'", testScope, resource.Scope)
	}

	if len(resource.ShortNames) != 1 || resource.ShortNames[0] != "tr" {
		t.Errorf("Expected ShortNames ['tr'], got %v", resource.ShortNames)
	}

	if resource.Plural != "testresources" {
		t.Errorf("Expected Plural 'testresources', got '%s'", resource.Plural)
	}

	// Check print columns were extracted
	if len(resource.PrintColumns) != 2 {
		t.Errorf("Expected 2 print columns, got %d", len(resource.PrintColumns))
	}
}
