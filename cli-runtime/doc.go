// Package cli-runtime provides building blocks for creating kubectl-compatible
// command-line interfaces with k1s.
//
// This package implements the CLI-Runtime pattern, providing helpers and utilities
// for CLI operations without creating cobra commands directly. It offers:
//
//   - Operation handlers (get, create, apply, delete) that work with any k1s client
//   - Output formatters (table, JSON, YAML, name) for consistent kubectl-style output
//   - Resource builders for fluent resource selection and filtering
//   - Reusable flag sets for common CLI patterns
//   - Options parsing utilities to convert flags to typed requests
//
// # Architecture
//
// The CLI-Runtime package follows a layered approach:
//
//   - Handlers: Process CLI operations using k1s client interfaces
//   - Printers: Format output in various formats (table, JSON, YAML, name)
//   - Builders: Provide fluent APIs for resource selection and filtering
//   - Flags: Define reusable pflag.FlagSet implementations
//   - Options: Parse flags into structured configuration objects
//
// # Usage Example
//
//	// Create a handler factory with a k1s client
//	factory := handlers.NewHandlerFactory(client)
//
//	// Create a get handler
//	getHandler := factory.Get()
//
//	// Create a get request
//	req := &handlers.GetRequest{
//	    ResourceType: gvk,
//	    Key: &client.ObjectKey{Name: "my-resource"},
//	    OutputOptions: options.NewOutputOptions(),
//	}
//
//	// Execute the request
//	resp, err := getHandler.Handle(ctx, req)
//
//	// Format and print the output
//	printerFactory := printers.NewPrinterFactory(nil)
//	printer, _ := printerFactory.NewPrinter("table")
//	printer.PrintObj(resp.Object, os.Stdout)
//
// # Integration with Two-Tier Runtime
//
// CLI-Runtime handlers work seamlessly with both CoreClient and ManagedRuntime
// tiers, automatically leveraging the appropriate performance optimizations
// based on the client implementation provided.
//
// # Compatibility
//
// This package is designed to be kubectl-compatible, providing familiar patterns
// and behaviors that kubectl users expect. It supports standard kubectl flags,
// output formats, and operational semantics.
package cliruntime
