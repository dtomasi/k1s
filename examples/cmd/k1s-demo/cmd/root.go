package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	demoruntime "github.com/dtomasi/k1s/examples/cmd/k1s-demo/pkg/runtime"
)

var (
	// Global runtime instance
	k1sRuntime demoruntime.Runtime
	// Command-line flags
	dbPath      string
	tenantID    string
	storageType string
)

// NewRootCommand creates the root k1s-demo command
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "k1s-demo",
		Short: "k1s demo CLI application for inventory management",
		Long: `k1s-demo is a complete CLI application that demonstrates all k1s capabilities
including CRUD operations, validation, and output formatting using an inventory system.

This demo showcases:
- Item and Category resource management
- All output formats: table, JSON, YAML, name
- Kubernetes-compatible CLI patterns
- Storage backend selection (memory or pebble)
- Multi-tenant support`,
		// Remove PersistentPreRunE to avoid runtime init on help
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if k1sRuntime != nil {
				ctx := context.Background()
				if err := k1sRuntime.Stop(ctx); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to stop k1s runtime: %v\n", err)
				}
			}
			return nil
		},
		Run: func(_ *cobra.Command, args []string) {
			fmt.Println("🎯 k1s-demo - Inventory Management CLI")
			fmt.Println("")
			fmt.Println("Available Commands:")
			fmt.Println("  get        List items or categories")
			fmt.Println("  create     Create resources from files")
			fmt.Println("  apply      Apply configuration files")
			fmt.Println("  delete     Delete resources")
			fmt.Println("")
			fmt.Println("Use 'k1s-demo <command> --help' for more information about a command.")
		},
	}

	// Global persistent flags
	rootCmd.PersistentFlags().StringVar(&dbPath, "db-path", "", "Path to database file (empty = memory storage)")
	rootCmd.PersistentFlags().StringVar(&tenantID, "tenant", "", "Tenant ID for multi-tenant usage")
	rootCmd.PersistentFlags().StringVar(&storageType, "storage", "memory", "Storage type: memory or pebble")

	// Add subcommands
	rootCmd.AddCommand(NewGetCommand(&k1sRuntime))

	return rootCmd
}
