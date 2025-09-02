// Package main provides the cli-gen tool for generating k1s instrumentation
// from kubebuilder markers, similar to controller-gen.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cli-gen",
	Short: "Generate k1s instrumentation from kubebuilder markers",
	Long: `cli-gen is a code generation tool that extracts kubebuilder markers
from Go source files and generates k1s runtime instrumentation.`,
	Run: func(_ *cobra.Command, args []string) {
		fmt.Println("cli-gen tool - k1s code generation")
		fmt.Println("Usage: cli-gen [flags] paths...")
		if len(args) > 0 {
			fmt.Printf("Processing paths: %v\n", args)
		}
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
