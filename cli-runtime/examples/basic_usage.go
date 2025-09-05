// Package examples provides usage examples for the CLI-Runtime package.
package examples

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/dtomasi/k1s/cli-runtime/builders"
	"github.com/dtomasi/k1s/cli-runtime/flags"
	"github.com/dtomasi/k1s/cli-runtime/handlers"
	"github.com/dtomasi/k1s/cli-runtime/options"
	"github.com/dtomasi/k1s/cli-runtime/printers"
	"github.com/dtomasi/k1s/core/client"
)

// BasicGetExample demonstrates how to use CLI-Runtime for a basic get operation.
func BasicGetExample(client client.Client) error {
	// Set up flags
	flagSet := pflag.NewFlagSet("get", pflag.ExitOnError)
	outputFlags := flags.OutputFlags()
	selectorFlags := flags.SelectorFlags()
	getFlags := flags.GetFlags()

	flagSet.AddFlagSet(outputFlags)
	flagSet.AddFlagSet(selectorFlags)
	flagSet.AddFlagSet(getFlags)

	// Parse command line (in a real CLI, this would come from os.Args)
	args := []string{"--output=table", "--namespace=default", "--selector=app=test"}
	err := flagSet.Parse(args)
	if err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	// Parse options from flags
	outputOpts, err := options.ParseOutputOptions(outputFlags)
	if err != nil {
		return fmt.Errorf("failed to parse output options: %w", err)
	}

	selectorOpts, err := options.ParseSelectorOptions(selectorFlags)
	if err != nil {
		return fmt.Errorf("failed to parse selector options: %w", err)
	}

	// Create handler
	factory := handlers.NewHandlerFactory(client)
	getHandler := factory.Get()

	// Create get request
	gvk := schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	}

	req := &handlers.GetRequest{
		ResourceType:  gvk,
		ListOptions:   selectorOpts.ToListOptions(),
		OutputOptions: outputOpts.ToHandlerOutputOptions(),
	}

	// Execute the get operation
	ctx := context.TODO()
	resp, err := getHandler.Handle(ctx, req)
	if err != nil {
		return fmt.Errorf("get operation failed: %w", err)
	}

	// Print the results
	printerFactory := printers.NewPrinterFactory(&printers.PrinterOptions{
		NoHeaders:  outputOpts.NoHeaders,
		ShowLabels: outputOpts.ShowLabels,
		Wide:       outputOpts.Wide,
	})

	printer, err := printerFactory.NewPrinter(outputOpts.Format)
	if err != nil {
		return fmt.Errorf("failed to create printer: %w", err)
	}

	if resp.IsCollection {
		// Print multiple objects
		for _, obj := range resp.Objects {
			err = printer.PrintObj(obj, os.Stdout)
			if err != nil {
				return fmt.Errorf("failed to print object: %w", err)
			}
		}
	} else {
		// Print single object
		err = printer.PrintObj(resp.Object, os.Stdout)
		if err != nil {
			return fmt.Errorf("failed to print object: %w", err)
		}
	}

	return nil
}

// ResourceBuilderExample demonstrates how to use the resource builder.
func ResourceBuilderExample(client client.Client) error {
	// Use resource builder for complex queries
	gvk := schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	}

	// Build query using fluent API
	result := builders.NewResourceBuilder().
		WithClient(client).
		WithResourceType(gvk).
		WithNamespace("production").
		WithLabelSelector(map[string]string{
			"app":     "backend",
			"version": "stable",
		}).
		Do(context.TODO())

	if result.Error() != nil {
		return fmt.Errorf("builder query failed: %w", result.Error())
	}

	// Get the results
	objects, err := result.Objects()
	if err != nil {
		return fmt.Errorf("failed to get objects: %w", err)
	}

	fmt.Printf("Found %d objects matching criteria\n", len(objects))
	return nil
}

// PrintingFormatsExample demonstrates different output formats.
func PrintingFormatsExample() error {
	// Create a sample object (in real usage, this would come from API)
	mockObj := &ExampleObject{
		Name:      "example-deployment",
		Namespace: "default",
		Labels: map[string]string{
			"app":     "example",
			"version": "v1.0",
		},
	}

	// Test different output formats
	formats := []string{"table", "json", "yaml", "name"}

	for _, format := range formats {
		fmt.Printf("\n--- Output in %s format ---\n", strings.ToUpper(format))

		factory := printers.NewPrinterFactory(&printers.PrinterOptions{})
		printer, err := factory.NewPrinter(format)
		if err != nil {
			return fmt.Errorf("failed to create %s printer: %w", format, err)
		}

		err = printer.PrintObj(mockObj, os.Stdout)
		if err != nil {
			return fmt.Errorf("failed to print in %s format: %w", format, err)
		}
	}

	return nil
}

// FilteringExample demonstrates resource filtering capabilities.
func FilteringExample() {
	// Create some sample objects
	objects := []client.Object{
		&ExampleObject{Name: "app-frontend-1", Namespace: "production", Labels: map[string]string{"app": "frontend", "tier": "web"}},
		&ExampleObject{Name: "app-backend-1", Namespace: "production", Labels: map[string]string{"app": "backend", "tier": "api"}},
		&ExampleObject{Name: "app-frontend-2", Namespace: "staging", Labels: map[string]string{"app": "frontend", "tier": "web"}},
		&ExampleObject{Name: "cache-redis-1", Namespace: "production", Labels: map[string]string{"app": "cache", "tier": "data"}},
	}

	// Filter by namespace
	productionFilter := builders.NewResourceFilter().
		Include(func(obj client.Object) bool {
			return obj.GetNamespace() == "production"
		})

	productionObjects := productionFilter.Filter(objects)
	fmt.Printf("Production objects: %d\n", len(productionObjects))

	// Filter by label
	frontendFilter := builders.NewResourceFilter().
		Include(func(obj client.Object) bool {
			labels := obj.GetLabels()
			return labels["app"] == "frontend"
		})

	frontendObjects := frontendFilter.Filter(objects)
	fmt.Printf("Frontend objects: %d\n", len(frontendObjects))

	// Combined filtering
	combinedFilter := builders.NewResourceFilter().
		Include(func(obj client.Object) bool {
			return obj.GetNamespace() == "production"
		}).
		Include(func(obj client.Object) bool {
			labels := obj.GetLabels()
			return labels["tier"] == "web" || labels["tier"] == "api"
		})

	combinedObjects := combinedFilter.Filter(objects)
	fmt.Printf("Production web/api objects: %d\n", len(combinedObjects))
}

// ExampleObject is a simple implementation for demonstration purposes.
type ExampleObject struct {
	Name      string
	Namespace string
	Labels    map[string]string
}

// Implement client.Object interface methods
func (e *ExampleObject) GetName() string                    { return e.Name }
func (e *ExampleObject) SetName(name string)                { e.Name = name }
func (e *ExampleObject) GetNamespace() string               { return e.Namespace }
func (e *ExampleObject) SetNamespace(namespace string)      { e.Namespace = namespace }
func (e *ExampleObject) GetLabels() map[string]string       { return e.Labels }
func (e *ExampleObject) SetLabels(labels map[string]string) { e.Labels = labels }
func (e *ExampleObject) GetObjectKind() schema.ObjectKind   { return schema.EmptyObjectKind }
func (e *ExampleObject) DeepCopyObject() runtime.Object     { return e } // Simplified

// Minimal implementations for other required methods
func (e *ExampleObject) GetSelfLink() string                                        { return "" }
func (e *ExampleObject) SetSelfLink(selfLink string)                                {}
func (e *ExampleObject) GetGenerateName() string                                    { return "" }
func (e *ExampleObject) SetGenerateName(name string)                                {}
func (e *ExampleObject) GetUID() types.UID                                          { return "" }
func (e *ExampleObject) SetUID(uid types.UID)                                       {}
func (e *ExampleObject) GetResourceVersion() string                                 { return "" }
func (e *ExampleObject) SetResourceVersion(version string)                          {}
func (e *ExampleObject) GetGeneration() int64                                       { return 0 }
func (e *ExampleObject) SetGeneration(generation int64)                             {}
func (e *ExampleObject) GetCreationTimestamp() metav1.Time                          { return metav1.Time{} }
func (e *ExampleObject) SetCreationTimestamp(timestamp metav1.Time)                 {}
func (e *ExampleObject) GetDeletionTimestamp() *metav1.Time                         { return nil }
func (e *ExampleObject) SetDeletionTimestamp(timestamp *metav1.Time)                {}
func (e *ExampleObject) GetDeletionGracePeriodSeconds() *int64                      { return nil }
func (e *ExampleObject) SetDeletionGracePeriodSeconds(gracePeriodSeconds *int64)    {}
func (e *ExampleObject) GetAnnotations() map[string]string                          { return nil }
func (e *ExampleObject) SetAnnotations(annotations map[string]string)               {}
func (e *ExampleObject) GetFinalizers() []string                                    { return nil }
func (e *ExampleObject) SetFinalizers(finalizers []string)                          {}
func (e *ExampleObject) GetOwnerReferences() []metav1.OwnerReference                { return nil }
func (e *ExampleObject) SetOwnerReferences(ownerReferences []metav1.OwnerReference) {}
func (e *ExampleObject) GetManagedFields() []metav1.ManagedFieldsEntry              { return nil }
func (e *ExampleObject) SetManagedFields(managedFields []metav1.ManagedFieldsEntry) {}
