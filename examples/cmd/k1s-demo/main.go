// Package main provides the k1s-demo CLI application that demonstrates
// all k1s capabilities using the inventory system (Items and Categories).
package main

import (
	"fmt"
	"os"

	"github.com/dtomasi/k1s/examples/cmd/k1s-demo/cmd"
)

func main() {
	rootCmd := cmd.NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}
}
