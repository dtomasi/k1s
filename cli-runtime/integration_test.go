package cliruntime_test

import (
	"context"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/dtomasi/k1s/cli-runtime/builders"
	"github.com/dtomasi/k1s/cli-runtime/flags"
	"github.com/dtomasi/k1s/cli-runtime/handlers"
	"github.com/dtomasi/k1s/cli-runtime/options"
	"github.com/dtomasi/k1s/cli-runtime/printers"
)

// TestBasicIntegration tests the basic integration of CLI-Runtime components.
func TestBasicIntegration(t *testing.T) {
	// Test that all components can be instantiated without errors

	// Test handler factory
	factory := handlers.NewHandlerFactory(nil)
	if factory == nil {
		t.Fatal("Handler factory should not be nil")
	}

	// Test handlers creation
	getHandler := factory.Get()
	createHandler := factory.Create()
	applyHandler := factory.Apply()
	deleteHandler := factory.Delete()

	if getHandler == nil || createHandler == nil || applyHandler == nil || deleteHandler == nil {
		t.Fatal("Handlers should not be nil")
	}

	// Test printer factory
	printerFactory := printers.NewPrinterFactory(nil)
	if printerFactory == nil {
		t.Fatal("Printer factory should not be nil")
	}

	// Test printer creation for different formats
	formats := printerFactory.GetSupportedFormats()
	if len(formats) == 0 {
		t.Fatal("Should support at least one output format")
	}

	for _, format := range formats {
		printer, err := printerFactory.NewPrinter(format)
		if err != nil {
			t.Fatalf("Failed to create printer for format %s: %v", format, err)
		}
		if printer == nil {
			t.Fatalf("Printer for format %s should not be nil", format)
		}
	}

	// Test resource builder
	builder := builders.NewResourceBuilder()
	if builder == nil {
		t.Fatal("Resource builder should not be nil")
	}

	// Test resource selector
	selector := builders.NewResourceSelector()
	if selector == nil {
		t.Fatal("Resource selector should not be nil")
	}

	// Test fluent API chaining
	gvk := schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	}

	chainedBuilder := builders.NewResourceSelector().
		ForType(gvk).
		WithNames("test-deployment").
		InNamespace("default").
		WithLabels(map[string]string{"app": "test"})

	if chainedBuilder == nil {
		t.Fatal("Chained builder should not be nil")
	}
}

// TestFlagsParsing tests the flags and options parsing integration.
func TestFlagsParsing(t *testing.T) {
	// Test output flags
	outputFlags := flags.OutputFlags()
	if outputFlags == nil {
		t.Fatal("Output flags should not be nil")
	}

	// Test parsing output options
	opts, err := options.ParseOutputOptions(outputFlags)
	if err != nil {
		t.Fatalf("Failed to parse output options: %v", err)
	}
	if opts == nil {
		t.Fatal("Output options should not be nil")
	}

	// Test selector flags
	selectorFlags := flags.SelectorFlags()
	if selectorFlags == nil {
		t.Fatal("Selector flags should not be nil")
	}

	// Test parsing selector options
	selectorOpts, err := options.ParseSelectorOptions(selectorFlags)
	if err != nil {
		t.Fatalf("Failed to parse selector options: %v", err)
	}
	if selectorOpts == nil {
		t.Fatal("Selector options should not be nil")
	}

	// Test conversion to handler options
	handlerOpts := opts.ToHandlerOutputOptions()
	if handlerOpts == nil {
		t.Fatal("Handler output options should not be nil")
	}

	// Test conversion to list options
	listOpts := selectorOpts.ToListOptions()
	// List options slice can be empty, that's valid
	_ = listOpts
}

// TestPrintingIntegration tests the printing integration.
func TestPrintingIntegration(t *testing.T) {
	// Create a mock object for testing
	mockObj := &MockObject{
		Name:      "test-object",
		Namespace: "test-namespace",
		Labels:    map[string]string{"app": "test", "version": "v1"},
	}

	// Test different output formats
	formats := []string{"json", "yaml", "name", "table"}

	for _, format := range formats {
		t.Run("format_"+format, func(t *testing.T) {
			factory := printers.NewPrinterFactory(&printers.PrinterOptions{})
			printer, err := factory.NewPrinter(format)
			if err != nil {
				t.Fatalf("Failed to create printer for format %s: %v", format, err)
			}

			var buf strings.Builder
			err = printer.PrintObj(mockObj, &buf)
			if err != nil {
				t.Fatalf("Failed to print object in format %s: %v", format, err)
			}

			output := buf.String()
			if len(output) == 0 {
				t.Fatalf("Output should not be empty for format %s", format)
			}

			// Basic validation based on format
			switch format {
			case "json":
				if !strings.Contains(output, "{") || !strings.Contains(output, "}") {
					t.Fatalf("JSON output should contain braces: %s", output)
				}
			case "name":
				if !strings.Contains(output, mockObj.Name) {
					t.Fatalf("Name output should contain object name: %s", output)
				}
			case "table":
				// Table output should have some structure
				lines := strings.Split(strings.TrimSpace(output), "\n")
				if len(lines) < 1 {
					t.Fatalf("Table output should have at least one line: %s", output)
				}
			}
		})
	}
}

// TestBuilderIntegration tests resource builder integration.
func TestBuilderIntegration(t *testing.T) {
	// Test builder without client (should return error)
	builder := builders.NewResourceBuilder()
	result := builder.Do(context.TODO())
	if result.Error() == nil {
		t.Fatal("Builder without client should return error")
	}

	// Test resource selector conversions
	gvk := schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	}

	selector := builders.NewResourceSelector().
		ForType(gvk).
		WithNames("test-deployment").
		InNamespace("test-namespace").
		WithLabels(map[string]string{"app": "test"})

	// Test conversion to list options
	// Note: This would require a real client to fully test
	// For now, just ensure the method doesn't panic
	listOpts := selector.ToListOptions()
	if listOpts == nil {
		t.Fatal("List options should not be nil")
	}
}

// MockObject implements the necessary interfaces for testing.
type MockObject struct {
	Name      string
	Namespace string
	Labels    map[string]string
}

// Implement runtime.Object interface
func (m *MockObject) GetObjectKind() schema.ObjectKind {
	return schema.EmptyObjectKind
}

func (m *MockObject) DeepCopyObject() runtime.Object {
	return &MockObject{
		Name:      m.Name,
		Namespace: m.Namespace,
		Labels:    copyMap(m.Labels),
	}
}

// Implement metav1.Object interface
func (m *MockObject) GetName() string                    { return m.Name }
func (m *MockObject) SetName(name string)                { m.Name = name }
func (m *MockObject) GetNamespace() string               { return m.Namespace }
func (m *MockObject) SetNamespace(namespace string)      { m.Namespace = namespace }
func (m *MockObject) GetLabels() map[string]string       { return m.Labels }
func (m *MockObject) SetLabels(labels map[string]string) { m.Labels = labels }

// Minimal implementations for other metav1.Object methods
func (m *MockObject) GetSelfLink() string                                        { return "" }
func (m *MockObject) SetSelfLink(selfLink string)                                {}
func (m *MockObject) GetGenerateName() string                                    { return "" }
func (m *MockObject) SetGenerateName(name string)                                {}
func (m *MockObject) GetUID() types.UID                                          { return "" }
func (m *MockObject) SetUID(uid types.UID)                                       {}
func (m *MockObject) GetResourceVersion() string                                 { return "" }
func (m *MockObject) SetResourceVersion(version string)                          {}
func (m *MockObject) GetGeneration() int64                                       { return 0 }
func (m *MockObject) SetGeneration(generation int64)                             {}
func (m *MockObject) GetCreationTimestamp() metav1.Time                          { return metav1.Time{} }
func (m *MockObject) SetCreationTimestamp(timestamp metav1.Time)                 {}
func (m *MockObject) GetDeletionTimestamp() *metav1.Time                         { return nil }
func (m *MockObject) SetDeletionTimestamp(timestamp *metav1.Time)                {}
func (m *MockObject) GetDeletionGracePeriodSeconds() *int64                      { return nil }
func (m *MockObject) SetDeletionGracePeriodSeconds(gracePeriodSeconds *int64)    {}
func (m *MockObject) GetAnnotations() map[string]string                          { return nil }
func (m *MockObject) SetAnnotations(annotations map[string]string)               {}
func (m *MockObject) GetFinalizers() []string                                    { return nil }
func (m *MockObject) SetFinalizers(finalizers []string)                          {}
func (m *MockObject) GetOwnerReferences() []metav1.OwnerReference                { return nil }
func (m *MockObject) SetOwnerReferences(ownerReferences []metav1.OwnerReference) {}
func (m *MockObject) GetManagedFields() []metav1.ManagedFieldsEntry              { return nil }
func (m *MockObject) SetManagedFields(managedFields []metav1.ManagedFieldsEntry) {}

// Helper function
func copyMap(original map[string]string) map[string]string {
	if original == nil {
		return nil
	}
	copied := make(map[string]string)
	for k, v := range original {
		copied[k] = v
	}
	return copied
}
