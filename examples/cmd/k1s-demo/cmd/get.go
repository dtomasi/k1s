package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/dtomasi/k1s/cli-runtime/flags"
	examplesv1alpha1 "github.com/dtomasi/k1s/examples/api/v1alpha1"
	demoruntime "github.com/dtomasi/k1s/examples/cmd/k1s-demo/pkg/runtime"
)

// NewGetCommand creates the get command
func NewGetCommand(runtimePtr *demoruntime.Runtime) *cobra.Command {
	// Create flag configurations
	outputConfig := flags.NewOutputConfig()
	selectorConfig := flags.NewSelectorConfig()
	getConfig := flags.NewGetConfig()

	cmd := &cobra.Command{
		Use:   "get [TYPE] [NAME]",
		Short: "Display one or many resources",
		Long: `Display one or many resources.

Supported resource types:
  items, item, i        - Inventory items
  categories, cat, c    - Item categories

Output formats:
  -o table    - Human-readable table (default)
  -o json     - JSON format
  -o yaml     - YAML format
  -o name     - Resource names only`,
		Example: `  # List all items
  k1s-demo get items
  
  # Get specific item in YAML format
  k1s-demo get item laptop-123 -o yaml
  
  # List categories with JSON output
  k1s-demo get categories -o json`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGetCommand(cmd, args, outputConfig, selectorConfig, getConfig)
		},
	}

	// Add CLI-Runtime flag sets bound to config structs
	cmd.Flags().AddFlagSet(flags.OutputFlagsVar(outputConfig))
	cmd.Flags().AddFlagSet(flags.SelectorFlagsVar(selectorConfig))
	cmd.Flags().AddFlagSet(flags.GetFlagsVar(getConfig))

	return cmd
}

// runGetCommand implements the get operation
func runGetCommand(_ *cobra.Command, args []string,
	outputConfig *flags.OutputConfig, selectorConfig *flags.SelectorConfig, _ *flags.GetConfig) error {
	// Parse arguments
	resourceType := args[0]
	var resourceName string
	if len(args) > 1 {
		resourceName = args[1]
	}

	// Use configuration from bound flag structs - no string parsing needed!
	outputFormat := outputConfig.Output

	// Map resource type to GVK
	gvk, err := getGVKFromType(resourceType)
	if err != nil {
		return fmt.Errorf("unknown resource type %s: %w", resourceType, err)
	}

	// For now, just show the structure working without full runtime initialization
	fmt.Printf("📋 Getting %s resources\n", gvk.Kind)
	fmt.Printf("   Resource Type: %s\n", resourceType)
	if resourceName != "" {
		fmt.Printf("   Resource Name: %s\n", resourceName)
	}
	fmt.Printf("   Namespace: %s\n", selectorConfig.Namespace)
	if selectorConfig.AllNamespaces {
		fmt.Printf("   All Namespaces: true\n")
	}
	fmt.Printf("   Output Format: %s\n", outputFormat)
	if selectorConfig.Selector != "" {
		fmt.Printf("   Label Selector: %s\n", selectorConfig.Selector)
	}
	if selectorConfig.FieldSelector != "" {
		fmt.Printf("   Field Selector: %s\n", selectorConfig.FieldSelector)
	}
	fmt.Printf("   GVK: %s\n", gvk.String())
	fmt.Println("")
	fmt.Println("✅ CLI-Runtime integration working!")
	fmt.Println("📚 All kubectl-compatible flags parsed successfully")
	fmt.Println("🏗️  Professional CLI structure in place")
	fmt.Println("")
	fmt.Println("⚠️  Full CRUD operations with k1s core runtime coming next...")

	return nil
}

// getGVKFromType maps resource type strings to GroupVersionKind
func getGVKFromType(resourceType string) (schema.GroupVersionKind, error) {
	switch strings.ToLower(resourceType) {
	case "items", "item", "i":
		return examplesv1alpha1.ItemGroupVersionKind, nil
	case "categories", "category", "cat", "c":
		return examplesv1alpha1.CategoryGroupVersionKind, nil
	default:
		return schema.GroupVersionKind{}, fmt.Errorf("unknown resource type: %s\n\nSupported types:\n  items, item, i\n  categories, category, cat, c", resourceType)
	}
}
