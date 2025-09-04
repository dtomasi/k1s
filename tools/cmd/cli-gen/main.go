// Package main provides the cli-gen tool for generating k1s instrumentation
// from kubebuilder markers, similar to controller-gen.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/dtomasi/k1s/tools/pkg/extractor"
	"github.com/dtomasi/k1s/tools/pkg/generator"
)

var (
	outputDir string
	verbose   bool
)

var rootCmd = &cobra.Command{
	Use:   "cli-gen [flags] paths...",
	Short: "Generate k1s instrumentation from kubebuilder markers",
	Long: `cli-gen is a code generation tool that extracts kubebuilder markers
from Go source files and generates k1s runtime instrumentation.

Similar to controller-gen, cli-gen processes kubebuilder markers in Go source
files to generate:
- Resource metadata lookup functions
- Validation strategies based on markers
- Print column definitions for CLI output

Example usage:
  cli-gen -output-dir=./generated ./api/v1alpha1
  cli-gen --verbose --output-dir=./pkg/generated ./examples/api/...`,
	RunE: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("no input paths specified")
		}

		if outputDir == "" {
			return fmt.Errorf("output directory is required (use --output-dir)")
		}

		if verbose {
			fmt.Printf("cli-gen: processing %d paths\n", len(args))
			fmt.Printf("cli-gen: output directory: %s\n", outputDir)
		}

		// Process paths and expand wildcards
		var expandedPaths []string
		for _, arg := range args {
			if filepath.Base(arg) == "..." {
				// Handle ./path/... pattern
				baseDir := filepath.Dir(arg)
				matches, err := filepath.Glob(filepath.Join(baseDir, "*"))
				if err != nil {
					return fmt.Errorf("failed to expand path %s: %w", arg, err)
				}
				expandedPaths = append(expandedPaths, matches...)
			} else {
				expandedPaths = append(expandedPaths, arg)
			}
		}

		if verbose {
			fmt.Printf("cli-gen: expanded to %d paths: %v\n", len(expandedPaths), expandedPaths)
		}

		// Create extractor and extract resource information
		ext := extractor.NewExtractor()
		resources, err := ext.Extract(expandedPaths)
		if err != nil {
			return fmt.Errorf("failed to extract markers: %w", err)
		}

		if len(resources) == 0 {
			fmt.Println("cli-gen: no resources found")
			return nil
		}

		if verbose {
			fmt.Printf("cli-gen: found %d resources:\n", len(resources))
			for _, res := range resources {
				fmt.Printf("  - %s (%s)\n", res.Kind, res.Name)
			}
		}

		// Create generator and generate code
		gen := generator.NewGenerator(outputDir)
		if err := gen.Generate(resources); err != nil {
			return fmt.Errorf("failed to generate code: %w", err)
		}

		fmt.Printf("cli-gen: successfully generated k1s instrumentation in %s\n", outputDir)
		return nil
	},
}

func init() {
	rootCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Directory to write generated files")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
