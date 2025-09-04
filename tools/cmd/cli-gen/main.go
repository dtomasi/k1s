// Package main provides the cli-gen tool for generating k1s instrumentation
// from kubebuilder markers, similar to controller-gen.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dtomasi/k1s/tools/pkg/extractor"
	"github.com/dtomasi/k1s/tools/pkg/generator"
)

var (
	outputDir string
	pathsFlag string
	verbose   bool
)

var rootCmd = &cobra.Command{
	Use:   "cli-gen [flags]",
	Short: "Generate k1s instrumentation from kubebuilder markers",
	Long: `cli-gen generates k1s instrumentation code from kubebuilder markers in Go source files.
It extracts resource metadata, validation rules, print columns, and defaulting strategies
to create runtime configuration for k1s.

Compatible with controller-gen syntax, it processes kubebuilder markers to generate:
- Resource metadata lookup functions  
- Validation strategy implementations
- Print column definitions for CLI output
- Defaulting strategy implementations

Examples:
  # Generate k1s instrumentation in source directory (default behavior)
  cli-gen paths=./apis/v1alpha1/...

  # Generate k1s instrumentation for a specific API package with custom output
  cli-gen paths=./apis/v1alpha1/... output:dir=./pkg/generated

  # Generate for multiple paths
  cli-gen paths=./apis/...,./pkg/types/... output:dir=./generated

  # Using flags (alternative syntax)
  cli-gen --paths=./apis/v1alpha1/... --output-dir=./generated`,
	RunE: func(_ *cobra.Command, args []string) error {
		// Parse controller-gen style arguments
		var parsedPaths []string
		var parsedOutputDir string

		for _, arg := range args {
			if strings.HasPrefix(arg, "paths=") {
				pathsStr := strings.TrimPrefix(arg, "paths=")
				parsedPaths = append(parsedPaths, strings.Split(pathsStr, ",")...)
			} else if strings.HasPrefix(arg, "output:dir=") {
				parsedOutputDir = strings.TrimPrefix(arg, "output:dir=")
			}
		}

		// Use flag values if no args provided
		if len(parsedPaths) == 0 && len(pathsFlag) > 0 {
			parsedPaths = strings.Split(pathsFlag, ",")
		}
		if parsedOutputDir == "" {
			parsedOutputDir = outputDir
		}

		if len(parsedPaths) == 0 {
			return fmt.Errorf("no paths provided - use paths=<path1>,<path2>... or --paths flag")
		}

		// If no output directory specified, use the source directory (controller-gen behavior)
		if parsedOutputDir == "" {
			firstPath := parsedPaths[0]
			// Handle ./path/... pattern
			if strings.HasSuffix(firstPath, "/...") {
				parsedOutputDir = strings.TrimSuffix(firstPath, "/...")
			} else {
				// If it's a file, use its directory; if it's a directory, use it directly
				if stat, err := os.Stat(firstPath); err == nil && !stat.IsDir() {
					parsedOutputDir = filepath.Dir(firstPath)
				} else {
					parsedOutputDir = firstPath
				}
			}
		}

		if verbose {
			fmt.Printf("cli-gen: processing paths: %v\n", parsedPaths)
			fmt.Printf("cli-gen: output directory: %s\n", parsedOutputDir)
		}

		// Expand glob patterns in paths
		var expandedPaths []string
		for _, path := range parsedPaths {
			// Handle ./path/... pattern
			if strings.HasSuffix(path, "/...") {
				baseDir := strings.TrimSuffix(path, "/...")
				matches, err := filepath.Glob(filepath.Join(baseDir, "*"))
				if err != nil {
					return fmt.Errorf("failed to expand path %s: %w", path, err)
				}
				expandedPaths = append(expandedPaths, matches...)
			} else {
				matches, err := filepath.Glob(path)
				if err != nil {
					return fmt.Errorf("invalid glob pattern %s: %w", path, err)
				}
				if len(matches) == 0 {
					expandedPaths = append(expandedPaths, path)
				} else {
					expandedPaths = append(expandedPaths, matches...)
				}
			}
		}

		if verbose {
			fmt.Printf("cli-gen: expanded paths: %v\n", expandedPaths)
		}

		// Extract resource information
		ext := extractor.NewExtractor()
		resources, err := ext.Extract(expandedPaths)
		if err != nil {
			return fmt.Errorf("failed to extract resource information: %w", err)
		}

		if len(resources) == 0 {
			fmt.Println("cli-gen: no resources found")
			return nil
		}

		if verbose {
			fmt.Printf("cli-gen: extracted %d resources\n", len(resources))
			for _, res := range resources {
				fmt.Printf("  - %s (%s)\n", res.Kind, res.Name)
			}
		}

		// Generate k1s instrumentation
		gen := generator.NewGenerator(parsedOutputDir)
		if err := gen.Generate(resources); err != nil {
			return fmt.Errorf("failed to generate k1s instrumentation: %w", err)
		}

		fmt.Printf("cli-gen: successfully generated k1s instrumentation in %s\n", parsedOutputDir)
		return nil
	},
}

func init() {
	rootCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Directory to write generated files")
	rootCmd.Flags().StringVarP(&pathsFlag, "paths", "p", "", "Comma-separated list of paths to process")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
