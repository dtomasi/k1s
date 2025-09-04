// Package main provides the cli-gen tool for generating k1s instrumentation
// from kubebuilder markers, similar to controller-gen.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dtomasi/k1s/tools/cli-gen/pkg/extractor"
	"github.com/dtomasi/k1s/tools/cli-gen/pkg/generator"
	"github.com/dtomasi/k1s/tools/cli-gen/pkg/version"
)

var (
	outputDir  string
	pathsFlag  string
	verbose    bool
	generators string
)

var rootCmd = &cobra.Command{
	Use:   "cli-gen [flags] [paths=<paths>] [output:dir=<dir>]",
	Short: "Generate k1s instrumentation from kubebuilder markers",
	Long: `cli-gen generates k1s instrumentation code from kubebuilder markers in Go source files.
It extracts resource metadata, validation rules, and defaulting strategies
to create runtime configuration for k1s.

Compatible with controller-gen syntax, it processes kubebuilder markers to generate:
- Resource metadata lookup functions (including print columns)  
- Validation strategy implementations
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
		// Parse paths from flags
		var parsedPaths []string
		if pathsFlag != "" {
			parsedPaths = strings.Split(pathsFlag, ",")
		}

		if len(parsedPaths) == 0 {
			return fmt.Errorf("no paths provided - use paths=<path1>,<path2>... or --paths flag")
		}

		parsedOutputDir := outputDir

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
			// Handle ./path/... pattern - find all Go package directories recursively
			if strings.HasSuffix(path, "/...") {
				baseDir := strings.TrimSuffix(path, "/...")
				err := filepath.WalkDir(baseDir, func(walkPath string, d os.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if d.IsDir() {
						// Check if this directory contains Go files (indicating it's a Go package)
						goFiles, _ := filepath.Glob(filepath.Join(walkPath, "*.go"))
						if len(goFiles) > 0 {
							expandedPaths = append(expandedPaths, walkPath)
						}
					}
					return nil
				})
				if err != nil {
					return fmt.Errorf("failed to expand path %s: %w", path, err)
				}
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

		// Parse generator selection
		enabledGenerators := parseGenerators(generators)
		if verbose && len(enabledGenerators) > 0 {
			fmt.Printf("cli-gen: enabled generators: %v\n", enabledGenerators)
		}

		// Process each path separately to generate files in the correct directories
		ext := extractor.NewExtractor()
		totalResources := 0

		for _, path := range expandedPaths {

			resources, err := ext.Extract([]string{path})
			if err != nil {
				return fmt.Errorf("failed to extract resources from %s: %w", path, err)
			}

			if len(resources) > 0 {
				totalResources += len(resources)

				// Generate in the same directory as the source files
				gen := generator.NewGenerator(path)
				gen.SetVerbose(verbose)

				// Configure selective generation
				if len(enabledGenerators) > 0 {
					gen.SetEnabledGenerators(enabledGenerators)
				}

				if err := gen.Generate(resources); err != nil {
					return fmt.Errorf("failed to generate code in %s: %w", path, err)
				}

				if verbose {
					fmt.Printf("cli-gen: generated %d resources in %s\n", len(resources), path)
					for _, res := range resources {
						fmt.Printf("  - %s (%s)\n", res.Kind, res.Name)
					}
				}
			}
		}

		if totalResources == 0 && verbose {
			fmt.Println("cli-gen: no resources found")
			return nil
		}

		if verbose {
			fmt.Printf("cli-gen: extracted %d resources total\n", totalResources)
			fmt.Printf("cli-gen: successfully generated k1s instrumentation\n")
		}
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Run: func(_ *cobra.Command, _ []string) {
		versionInfo := version.GetVersionInfo()
		fmt.Println(versionInfo.String())
	},
}

var versionJSONCmd = &cobra.Command{
	Use:   "json",
	Short: "Print version information in JSON format",
	Run: func(_ *cobra.Command, _ []string) {
		versionInfo := version.GetVersionInfo()
		jsonBytes, _ := json.MarshalIndent(versionInfo, "", "  ")
		fmt.Println(string(jsonBytes))
	},
}

func init() {
	rootCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Directory to write generated files")
	rootCmd.Flags().StringVarP(&pathsFlag, "paths", "p", "", "Comma-separated list of paths to process")
	rootCmd.Flags().StringVarP(&generators, "generators", "g", "all", "Comma-separated list of generators to run (object,validation,defaulting,all)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Add version subcommands
	versionCmd.AddCommand(versionJSONCmd)
	rootCmd.AddCommand(versionCmd)
}

// parseGenerators parses the generator selection string
func parseGenerators(generatorsFlag string) []string {
	if generatorsFlag == "" || generatorsFlag == "all" {
		return []string{} // Empty means all generators enabled
	}

	generators := strings.Split(generatorsFlag, ",")
	var enabled []string

	validGenerators := map[string]bool{
		"object":     true,
		"validation": true,
		"defaulting": true,
	}

	for _, gen := range generators {
		gen = strings.TrimSpace(gen)
		if validGenerators[gen] {
			enabled = append(enabled, gen)
		}
	}

	return enabled
}

func main() {
	// Pre-process controller-gen style arguments before Cobra
	args := preprocessArgs(os.Args[1:])

	// Set the processed args
	rootCmd.SetArgs(args)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// preprocessArgs converts controller-gen style arguments to cobra-compatible flags
func preprocessArgs(args []string) []string {
	processed := make([]string, 0, len(args))

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "paths="):
			pathsStr := strings.TrimPrefix(arg, "paths=")
			processed = append(processed, "--paths", pathsStr)
		case strings.HasPrefix(arg, "output:dir="):
			outputDir := strings.TrimPrefix(arg, "output:dir=")
			processed = append(processed, "--output-dir", outputDir)
		default:
			processed = append(processed, arg)
		}
	}

	return processed
}
