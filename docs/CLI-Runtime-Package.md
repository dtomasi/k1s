# K1S CLI-Runtime Package Specification

**Related Documentation:**
- [Controller-Runtime Package](Controller-Runtime-Package.md) - Controller runtime for CLI environments
- [Architecture](Architecture.md) - Complete k1s system architecture

## Overview

The CLI-Runtime package (`core/pkg/cli-runtime/`) provides helper functions and utilities for building kubectl-compatible command-line interfaces with k1s. It offers resource operations, output formatting, and flag management - everything needed to build CLI applications, but without creating cobra commands directly.

## Core Responsibilities

### 1. **Resource Operation Handlers**
- Provide kubectl-compatible operation implementations
- Handle get, create, apply, delete operations with proper error handling
- Accept k1s runtime as dependency injection

### 2. **Output Formatting & Printers**
- Multiple output formats: table, yaml, json, name, custom-columns
- kubectl-compatible table formatting with proper columns
- Pluggable printer system

### 3. **Resource Builders & Selectors**
- kubectl-style resource builders and selectors
- Label and field selector support
- Namespace scoping and cross-namespace operations

### 4. **Flag Sets & Utilities**
- Reusable pflag.FlagSet for common operations
- Options parsing from flags
- Standard kubectl flag patterns

## Package Structure

```
core/pkg/cli-runtime/
├── handlers/           # Operation handlers (get, create, apply, delete)
│   ├── get_handler.go
│   ├── create_handler.go
│   ├── apply_handler.go
│   └── delete_handler.go
├── builders/           # Resource builders and selectors
│   ├── resource_builder.go
│   └── selector_builder.go
├── printers/          # Output formatters
│   ├── table_printer.go
│   ├── yaml_printer.go
│   ├── json_printer.go
│   └── printer_interface.go
├── flags/             # Reusable flag sets
│   └── flag_sets.go
└── options/           # Command options and parsing
    ├── options_parser.go
    ├── get_options.go
    └── apply_options.go
```

## 1. Operation Handlers

The CLI-Runtime provides handler functions that implement kubectl-compatible operations:

### Handler Functions

```go
// core/pkg/cli-runtime/handlers/get_handler.go
package handlers

import (
    "context"
    "io"
    
    "github.com/dtomasi/k1s/core/pkg/runtime"
    "k8s.io/apimachinery/pkg/runtime/object"
)

// GetHandler handles get operations  
type GetHandler struct {
    runtime *runtime.Runtime
}

// NewGetHandler creates a new get handler
func NewGetHandler(runtime *runtime.Runtime) *GetHandler {
    return &GetHandler{runtime: runtime}
}

// Handle executes the get operation
func (h *GetHandler) Handle(ctx context.Context, req GetRequest) (*GetResponse, error) {
    client := h.runtime.GetClient()
    
    // Build resource query using builder pattern
    builder := NewResourceBuilder().
        WithClient(client).
        WithResourceType(req.ResourceType)
    
    if req.ResourceName != "" {
        builder = builder.WithResourceName(req.ResourceName)
    }
    
    if req.Namespace != "" {
        builder = builder.WithNamespace(req.Namespace)
    }
    
    // Execute query
    result := builder.Do()
    objects, err := result.Objects()
    if err != nil {
        return nil, err
    }
    
    // Format output if writer provided
    if req.Output != nil {
        printer := NewPrinterForFormat(req.OutputFormat)
        err = printer.PrintObjects(objects, req.Output)
        if err != nil {
            return nil, err
        }
    }
    
    return &GetResponse{
        Objects: objects,
        Count:   len(objects),
    }, nil
}

// GetRequest contains parameters for get operations
type GetRequest struct {
    ResourceType  string
    ResourceName  string 
    Namespace     string
    LabelSelector string
    FieldSelector string
    AllNamespaces bool
    OutputFormat  string
    Output        io.Writer
}

// GetResponse contains results from get operations  
type GetResponse struct {
    Objects []runtime.Object
    Count   int
}
```

### Usage Example

```go
import (
    "context"
    "os"
    
    "github.com/dtomasi/k1s/core/pkg/cli-runtime/handlers"
    "github.com/dtomasi/k1s/core/pkg/cli-runtime/flags"
    "github.com/dtomasi/k1s/storage/pebble"
    "github.com/spf13/cobra"
    "github.com/spf13/pflag"
)

func main() {
    // Initialize k1s runtime
    storage, _ := pebble.NewStorage("./data/app.db")
    runtime, _ := k1s.NewRuntime(storage, k1s.WithTenant("my-app"))
    
    // Create handlers
    getHandler := handlers.NewGetHandler(runtime)
    
    // Create cobra command manually
    getCmd := &cobra.Command{
        Use:   "get [TYPE] [NAME]",
        Short: "Display one or many resources", 
        RunE:  func(cmd *cobra.Command, args []string) error {
            // Parse flags to request
            req := handlers.GetRequest{
                ResourceType:  args[0],
                OutputFormat:  cmd.Flag("output").Value.String(),
                Output:        cmd.OutOrStdout(),
            }
            
            if len(args) > 1 {
                req.ResourceName = args[1]
            }
            
            // Execute operation
            resp, err := getHandler.Handle(context.Background(), req)
            if err != nil {
                return err
            }
            
            // Optional: additional processing of response
            return nil
        },
    }
    
    // Add flag sets
    getCmd.Flags().AddFlagSet(flags.OutputFlags())
    getCmd.Flags().AddFlagSet(flags.SelectorFlags())
    
    getCmd.Execute()
}
```

## 2. Flag Sets & Options

### Reusable Flag Sets

```go
// core/pkg/cli-runtime/flags/flag_sets.go
package flags

import "github.com/spf13/pflag"

// CommonFlags returns flags used across multiple commands
func CommonFlags() *pflag.FlagSet {
    flags := pflag.NewFlagSet("common", pflag.ContinueOnError)
    
    flags.StringP("namespace", "n", "", "Namespace scope for the request")
    flags.String("context", "", "The name of the k1s context to use")
    flags.Duration("timeout", 0, "Request timeout")
    flags.BoolP("help", "h", false, "Help for the command")
    
    return flags
}

// OutputFlags returns flags for output formatting
func OutputFlags() *pflag.FlagSet {
    flags := pflag.NewFlagSet("output", pflag.ContinueOnError)
    
    flags.StringP("output", "o", "table", "Output format: table|yaml|json|name|custom-columns=...")
    flags.Bool("no-headers", false, "Don't print headers in table output")
    flags.Bool("show-labels", false, "Show labels in table output")
    flags.StringSlice("sort-by", nil, "Sort list by field")
    
    return flags
}

// SelectorFlags returns flags for resource selection
func SelectorFlags() *pflag.FlagSet {
    flags := pflag.NewFlagSet("selector", pflag.ContinueOnError)
    
    flags.StringP("selector", "l", "", "Label selector")
    flags.String("field-selector", "", "Field selector")
    flags.BoolP("all-namespaces", "A", false, "List resources from all namespaces")
    
    return flags
}

// FileFlags returns flags for file input operations
func FileFlags() *pflag.FlagSet {
    flags := pflag.NewFlagSet("file", pflag.ContinueOnError)
    
    flags.StringSliceP("filename", "f", nil, "Filename or directory to process")
    flags.BoolP("recursive", "R", false, "Process directories recursively")
    flags.String("from-literal", "", "Specify literal value")
    
    return flags
}
```

## 3. Resource Builders

### Builder Interface

```go
// core/pkg/cli-runtime/builders/resource_builder.go
package builders

import (
    "k8s.io/apimachinery/pkg/runtime"
    "github.com/dtomasi/k1s/core/pkg/client"
)

// ResourceBuilder builds resources for CLI operations
type ResourceBuilder interface {
    // Configuration methods
    WithClient(client.Client) ResourceBuilder
    WithNamespace(string) ResourceBuilder
    WithAllNamespaces(bool) ResourceBuilder
    WithLabelSelector(string) ResourceBuilder
    WithFieldSelector(string) ResourceBuilder
    
    // Resource specification
    WithResourceType(string) ResourceBuilder
    WithResourceName(string) ResourceBuilder
    WithFilenames(...string) ResourceBuilder
    
    // Execution methods
    Do() *Result
    Stream(func(*ResourceInfo) error) error
}

// ResourceInfo contains information about a single resource
type ResourceInfo struct {
    Name      string
    Namespace string
    Object    runtime.Object
    
    // Metadata
    ResourceVersion string
    Labels          map[string]string
    Annotations     map[string]string
}

// Result contains the results of a resource operation
type Result struct {
    resources []*ResourceInfo
    err       error
}

func (r *Result) Infos() ([]*ResourceInfo, error) {
    return r.resources, r.err
}

func (r *Result) Visit(fn func(*ResourceInfo) error) error {
    if r.err != nil {
        return r.err
    }
    for _, resource := range r.resources {
        if err := fn(resource); err != nil {
            return err
        }
    }
    return nil
}
```

## 4. Output Formatters

### Printer Factory

```go
// core/pkg/cli-runtime/printers/printer_factory.go
package printers

import (
    "io"
    "k8s.io/apimachinery/pkg/runtime"
)

// PrinterFactory creates output formatters for different formats
type PrinterFactory interface {
    PrinterForFormat(format string) (Printer, error)
    SupportedFormats() []string
}

// Printer formats and outputs k1s resources
type Printer interface {
    PrintObjects(objects []runtime.Object, w io.Writer) error
    PrintObject(obj runtime.Object, w io.Writer) error
}

// TablePrinter prints resources in kubectl-compatible table format
type TablePrinter interface {
    Printer
    WithHeaders(show bool) TablePrinter
    WithLabels(show bool) TablePrinter
    WithSortBy(columns []string) TablePrinter
}
```

### Table Printer Implementation

```go
// core/pkg/cli-runtime/printers/table_printer.go  
package printers

import (
    "fmt"
    "io"
    "strings"
    "text/tabwriter"
    "k8s.io/apimachinery/pkg/runtime"
)

// tablePrinter implements kubectl-compatible table output
type tablePrinter struct {
    showHeaders bool
    showLabels  bool
    sortBy      []string
}

func NewTablePrinter() TablePrinter {
    return &tablePrinter{
        showHeaders: true,
        showLabels:  false,
    }
}

func (tp *tablePrinter) WithHeaders(show bool) TablePrinter {
    tp.showHeaders = show
    return tp
}

func (tp *tablePrinter) WithLabels(show bool) TablePrinter {
    tp.showLabels = show
    return tp
}

func (tp *tablePrinter) PrintObjects(objects []runtime.Object, w io.Writer) error {
    if len(objects) == 0 {
        return nil
    }
    
    // Create tabwriter for aligned output
    tw := tabwriter.NewWriter(w, 0, 8, 2, ' ', 0)
    defer tw.Flush()
    
    // Print headers if enabled
    if tp.showHeaders {
        if err := tp.printHeaders(objects[0], tw); err != nil {
            return err
        }
    }
    
    // Print each object
    for _, obj := range objects {
        if err := tp.printObjectRow(obj, tw); err != nil {
            return err
        }
    }
    
    return nil
}
```

## Benefits

### 1. **Helper-Based Approach**
- Provides building blocks, not finished commands
- User controls cobra command creation
- Maximum flexibility for custom CLI design

### 2. **kubectl Compatibility** 
- Standard operation handlers and patterns
- Compatible flag sets and behaviors
- Familiar output formats and table printing

### 3. **Modular Design**
- Reusable flag sets across different commands
- Pluggable printer system for output formatting
- Builder pattern for resource operations

### 4. **Developer Friendly**
- Clear separation of concerns
- Type-safe request/response patterns
- Easy integration with existing cobra applications

## Implementation Notes

The CLI-Runtime package focuses on **utilities and helpers** rather than complete solutions. It provides:

- **Operation Handlers**: Implement kubectl-compatible operations (get, create, apply, delete)
- **Flag Sets**: Reusable pflag.FlagSet instances for common operation flags
- **Builders**: kubectl-style resource selection and filtering
- **Printers**: Multiple output format support with pluggable system
- **Options Parsing**: Utilities to parse flags into typed request structures

Users create their own cobra commands and use these helpers to implement the functionality. This keeps the CLI-Runtime focused on providing the essential building blocks while letting users control the command structure and user experience.