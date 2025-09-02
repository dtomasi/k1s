// Package main provides the k1s-demo CLI application that demonstrates
// all k1s capabilities using the inventory system (Items and Categories).
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "k1s-demo",
	Short: "k1s demo CLI application",
	Long: `k1s-demo is a complete CLI application that demonstrates all k1s capabilities
including CRUD operations, validation, and output formatting using an inventory system.`,
	Run: func(_ *cobra.Command, args []string) {
		fmt.Println("k1s-demo - Inventory Management CLI")
		fmt.Println("Commands: get, create, apply, delete")
		if len(args) > 0 {
			fmt.Printf("Additional args: %v\n", args)
		}
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
