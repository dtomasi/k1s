# K1S CLI-Runtime Package Specification

**Related Documentation:**
- [Controller-Runtime Package](Controller-Runtime-Package.md) - Controller runtime for CLI environments
- [Architecture](Architecture.md) - Complete k1s system architecture

## Overview

The CLI-Runtime package (`core/pkg/cli-runtime/`) provides a complete, kubectl-compatible command-line interface foundation for k1s. It handles client provisioning, runtime bootstrapping, command execution, and output formatting - everything needed to build robust CLI applications.

## Core Responsibilities

### 1. **Runtime Bootstrapping & Client Provisioning**
- Initialize and boot the core k1s runtime
- Provide configured k1s client to commands
- Handle runtime lifecycle (startup/shutdown)

### 2. **Command Operations**
- kubectl-compatible operations: api-resources, apply, create, patch, get, delete
- Resource discovery and validation
- Error handling and user feedback

### 3. **Output Formatting**
- Multiple output formats: table, yaml, json, name, custom-columns
- Printer factories and builders
- kubectl-compatible formatting

## Package Structure

```
core/pkg/cli-runtime/
├── bootstrap.go         # Runtime bootstrapping
├── client.go           # Client factory and provisioning
├── commands/           # Command implementations
│   ├── api_resources.go
│   ├── apply.go
│   ├── create.go
│   ├── delete.go
│   ├── get.go
│   └── patch.go
├── factories/          # Resource and client factories
│   ├── client_factory.go
│   ├── resource_factory.go
│   └── printer_factory.go
├── builders/           # Resource builders and selectors
│   ├── resource_builder.go
│   └── selector_builder.go
├── printers/          # Output formatters
│   ├── table_printer.go
│   ├── yaml_printer.go
│   ├── json_printer.go
│   └── printer_interface.go
└── options/           # Command options and flags
    ├── common_options.go
    ├── get_options.go
    └── apply_options.go
```

## 1. Runtime Bootstrapping

### Bootstrap Interface

```go
// core/pkg/cli-runtime/bootstrap.go
package cliruntime

import (
    "context"
    "time"
    
    "github.com/dtomasi/k1s/core/pkg/client"
    "github.com/dtomasi/k1s/core/pkg/runtime"
    "github.com/dtomasi/k1s/core/pkg/storage"
)

// Bootstrap provides runtime initialization and client provisioning
type Bootstrap interface {
    // Initialize the k1s runtime with configuration
    Initialize(config Config) error
    
    // Get configured client for commands
    GetClient() client.Client
    
    // Get runtime for advanced operations
    GetRuntime() runtime.Runtime
    
    // Shutdown gracefully
    Shutdown() error
}

type Config struct {
    // Storage configuration
    Storage storage.Config
    
    // CLI-specific settings
    Timeout    time.Duration // Default operation timeout
    Kubeconfig string        // Path to kubeconfig-like file (optional)
    Context    string        // Context name (maps to tenant prefix)
    Namespace  string        // Default namespace
    
    // Output settings
    OutputFormat string      // Default output format
    NoHeaders    bool        // Suppress headers in table output
    
    // Performance settings
    CacheDir     string      // CLI cache directory
    CacheEnabled bool        // Enable cross-process caching
}

// NewBootstrap creates a new CLI runtime bootstrap
func NewBootstrap() Bootstrap {
    return &bootstrap{
        startTime: time.Now(),
    }
}

type bootstrap struct {
    config    Config
    runtime   runtime.Runtime
    client    client.Client
    startTime time.Time
    
    initialized bool
}

func (b *bootstrap) Initialize(config Config) error {
    b.config = config
    
    // Fast initialization for CLI (<100ms target)
    runtimeConfig := runtime.Config{
        Storage: config.Storage,
        FastBoot: true, // CLI-optimized boot
    }
    
    // Set tenant from CLI context
    if config.Context != "" {
        runtimeConfig.Storage.TenantConfig = storage.TenantConfig{
            Prefix: config.Context,
        }
    }
    
    // Initialize core runtime
    rt, err := runtime.NewRuntime(runtimeConfig)
    if err != nil {
        return fmt.Errorf("failed to initialize runtime: %w", err)
    }
    
    b.runtime = rt
    b.client = rt.GetClient()
    b.initialized = true
    
    return nil
}

func (b *bootstrap) GetClient() client.Client {
    if !b.initialized {
        panic("bootstrap not initialized - call Initialize() first")
    }
    return b.client
}

func (b *bootstrap) GetRuntime() runtime.Runtime {
    if !b.initialized {
        panic("bootstrap not initialized - call Initialize() first")
    }
    return b.runtime
}

func (b *bootstrap) Shutdown() error {
    if b.runtime != nil {
        return b.runtime.Stop()
    }
    return nil
}
```

## 2. Operation Handlers (Instrumentation)

### Operation Handler Interface

```go
// core/pkg/cli-runtime/handlers/operation_handlers.go
package handlers

import (
    "context"
    "io"
    
    "k8s.io/apimachinery/pkg/runtime"
    "github.com/dtomasi/k1s/core/pkg/client"
)

// OperationHandler provides instrumentation for CLI operations
type OperationHandler interface {
    // Resource operations
    HandleGet(ctx context.Context, req GetRequest) (*GetResponse, error)
    HandleCreate(ctx context.Context, req CreateRequest) (*CreateResponse, error)
    HandleApply(ctx context.Context, req ApplyRequest) (*ApplyResponse, error)
    HandleDelete(ctx context.Context, req DeleteRequest) (*DeleteResponse, error)
    HandlePatch(ctx context.Context, req PatchRequest) (*PatchResponse, error)
    
    // Discovery operations
    HandleAPIResources(ctx context.Context, req APIResourcesRequest) (*APIResourcesResponse, error)
}

// GetRequest contains parameters for get operations
type GetRequest struct {
    // Resource specification
    ResourceType string   // "items", "categories", etc.
    ResourceName string   // specific resource name (optional)
    Namespace    string   // namespace (optional)
    
    // Selection
    LabelSelector string  // label selector
    FieldSelector string  // field selector
    AllNamespaces bool    // get from all namespaces
    
    // Output
    OutputFormat string   // "table", "yaml", "json", "name", "custom-columns"
    NoHeaders    bool     // suppress headers
    ShowLabels   bool     // show labels column
    
    // Writer for output
    Output io.Writer
}

// GetResponse contains results from get operations
type GetResponse struct {
    Resources []runtime.Object
    Count     int
    Printed   bool // whether output was written to GetRequest.Output
}

// CreateRequest contains parameters for create operations
type CreateRequest struct {
    // Input sources
    Filenames []string  // file paths
    Reader    io.Reader // stdin or other reader
    
    // Options
    DryRun    string    // "none", "client", "server"
    Namespace string    // target namespace
    
    // Output
    Output io.Writer
}

// CreateResponse contains results from create operations
type CreateResponse struct {
    Created []runtime.Object
    Errors  []error
}

// ApplyRequest contains parameters for apply operations
type ApplyRequest struct {
    // Input sources
    Filenames []string  // file paths
    Reader    io.Reader // stdin or other reader
    
    // Options
    Force     bool      // force apply
    DryRun    string    // "none", "client", "server"
    Namespace string    // target namespace
    
    // Output
    Output io.Writer
}

// ApplyResponse contains results from apply operations
type ApplyResponse struct {
    Applied  []runtime.Object
    Created  []runtime.Object
    Updated  []runtime.Object
    Errors   []error
}

// DeleteRequest contains parameters for delete operations
type DeleteRequest struct {
    // Resource specification  
    ResourceType string   // "items", "categories", etc.
    ResourceName string   // specific resource name (optional)
    Namespace    string   // namespace (optional)
    
    // Selection (alternative to ResourceName)
    LabelSelector string  // delete by label selector
    FieldSelector string  // delete by field selector
    
    // Input sources (alternative)
    Filenames []string    // delete resources from files
    
    // Options
    Force        bool     // force deletion
    GracePeriod  int64    // grace period for deletion
    
    // Output
    Output io.Writer
}

// DeleteResponse contains results from delete operations
type DeleteResponse struct {
    Deleted []runtime.Object
    Errors  []error
}

// NewOperationHandler creates a new operation handler
func NewOperationHandler(client client.Client, printerFactory PrinterFactory) OperationHandler {
    return &operationHandler{
        client:         client,
        printerFactory: printerFactory,
        resourceBuilder: builders.NewResourceBuilder(),
    }
}

type operationHandler struct {
    client          client.Client
    printerFactory  PrinterFactory
    resourceBuilder builders.ResourceBuilder
}

// HandleGet implements kubectl-compatible get operation
func (h *operationHandler) HandleGet(ctx context.Context, req GetRequest) (*GetResponse, error) {
    // Build resource query
    builder := h.resourceBuilder.
        WithClient(h.client).
        WithResourceType(req.ResourceType)
    
    if req.ResourceName != "" {
        builder = builder.WithResourceName(req.ResourceName)
    }
    
    if req.Namespace != "" {
        builder = builder.WithNamespace(req.Namespace)
    } else if req.AllNamespaces {
        builder = builder.WithAllNamespaces(true)
    }
    
    if req.LabelSelector != "" {
        builder = builder.WithLabelSelector(req.LabelSelector)
    }
    
    if req.FieldSelector != "" {
        builder = builder.WithFieldSelector(req.FieldSelector)
    }
    
    // Execute query
    result := builder.Do()
    infos, err := result.Infos()
    if err != nil {
        return nil, err
    }
    
    // Extract objects
    var objects []runtime.Object
    for _, info := range infos {
        objects = append(objects, info.Object)
    }
    
    response := &GetResponse{
        Resources: objects,
        Count:     len(objects),
    }
    
    // Print if output writer provided
    if req.Output != nil {
        printer, err := h.printerFactory.PrinterForFormat(req.OutputFormat)
        if err != nil {
            return response, err
        }
        
        err = printer.PrintObjects(objects, req.Output)
        if err != nil {
            return response, err
        }
        response.Printed = true
    }
    
    return response, nil
}
```

### Resource Builder System (unchanged from previous design but as instrumentation)

```go
// core/pkg/cli-runtime/builders/resource_builder.go
package builders

import (
    "context"
    "fmt"
    
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/labels"
    "k8s.io/apimachinery/pkg/fields"
    
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

func (r *Result) Object() (runtime.Object, error) {
    if r.err != nil {
        return nil, r.err
    }
    if len(r.resources) != 1 {
        return nil, fmt.Errorf("expected 1 resource, got %d", len(r.resources))
    }
    return r.resources[0].Object, nil
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

// NewResourceBuilder creates a new resource builder
func NewResourceBuilder() ResourceBuilder {
    return &resourceBuilder{}
}

type resourceBuilder struct {
    client       client.Client
    namespace    string
    allNamespaces bool
    labelSelector string
    fieldSelector string
    resourceType string
    resourceName string
    filenames    []string
}

func (b *resourceBuilder) WithClient(c client.Client) ResourceBuilder {
    b.client = c
    return b
}

func (b *resourceBuilder) WithNamespace(ns string) ResourceBuilder {
    b.namespace = ns
    return b
}

func (b *resourceBuilder) WithResourceType(rt string) ResourceBuilder {
    b.resourceType = rt
    return b
}

func (b *resourceBuilder) Do() *Result {
    if b.resourceName != "" {
        return b.getSingleResource()
    }
    return b.listResources()
}
```

### Flag Sets for CLI Integration

```go
// core/pkg/cli-runtime/flags/flag_sets.go
package flags

import (
    "github.com/spf13/pflag"
)

// FlagSets provides reusable pflag FlagSets for common CLI operations
type FlagSets interface {
    // Common flags used across multiple commands
    CommonFlags() *pflag.FlagSet
    
    // Output formatting flags
    OutputFlags() *pflag.FlagSet
    
    // Resource selection flags  
    SelectorFlags() *pflag.FlagSet
    
    // File input flags
    FileFlags() *pflag.FlagSet
    
    // Apply operation flags
    ApplyFlags() *pflag.FlagSet
    
    // Delete operation flags
    DeleteFlags() *pflag.FlagSet
}

// NewFlagSets creates a new flag sets provider
func NewFlagSets() FlagSets {
    return &flagSets{}
}

type flagSets struct{}

// CommonFlags returns flags used across multiple commands
func (f *flagSets) CommonFlags() *pflag.FlagSet {
    flags := pflag.NewFlagSet("common", pflag.ContinueOnError)
    
    flags.StringP("namespace", "n", "", "Namespace scope for the request")
    flags.String("context", "", "The name of the k1s context to use")
    flags.Duration("timeout", 0, "Request timeout")
    flags.BoolP("help", "h", false, "Help for the command")
    
    return flags
}

// OutputFlags returns flags for output formatting
func (f *flagSets) OutputFlags() *pflag.FlagSet {
    flags := pflag.NewFlagSet("output", pflag.ContinueOnError)
    
    flags.StringP("output", "o", "table", "Output format: table|yaml|json|name|custom-columns=...")
    flags.Bool("no-headers", false, "Don't print headers in table output")
    flags.Bool("show-labels", false, "Show labels in table output")
    flags.StringSlice("sort-by", nil, "Sort list by field")
    
    return flags
}

// SelectorFlags returns flags for resource selection
func (f *flagSets) SelectorFlags() *pflag.FlagSet {
    flags := pflag.NewFlagSet("selector", pflag.ContinueOnError)
    
    flags.StringP("selector", "l", "", "Label selector (supports '=', '==', '!=')")
    flags.String("field-selector", "", "Field selector")
    flags.BoolP("all-namespaces", "A", false, "List resources from all namespaces")
    
    return flags
}

// FileFlags returns flags for file input operations
func (f *flagSets) FileFlags() *pflag.FlagSet {
    flags := pflag.NewFlagSet("file", pflag.ContinueOnError)
    
    flags.StringSliceP("filename", "f", nil, "Filename or directory to process")
    flags.BoolP("recursive", "R", false, "Process directories recursively")
    flags.String("from-literal", "", "Specify literal value")
    
    return flags
}

// ApplyFlags returns flags specific to apply operations
func (f *flagSets) ApplyFlags() *pflag.FlagSet {
    flags := pflag.NewFlagSet("apply", pflag.ContinueOnError)
    
    flags.String("dry-run", "none", "Dry run mode: none|client|server")
    flags.Bool("force", false, "Force apply (may result in data loss)")
    flags.Bool("validate", true, "Validate resources before applying")
    flags.String("field-manager", "k1s", "Name of the field manager")
    
    return flags
}

// DeleteFlags returns flags specific to delete operations
func (f *flagSets) DeleteFlags() *pflag.FlagSet {
    flags := pflag.NewFlagSet("delete", pflag.ContinueOnError)
    
    flags.Bool("force", false, "Force delete (no confirmation)")
    flags.Int64("grace-period", -1, "Grace period for deletion")
    flags.Bool("ignore-not-found", false, "Ignore if resource not found")
    flags.Bool("wait", true, "Wait for deletion to complete")
    
    return flags
}
```

### Options Parsing from Flags

```go
// core/pkg/cli-runtime/options/options_parser.go
package options

import (
    "time"
    "github.com/spf13/pflag"
)

// OptionsParser extracts typed options from pflag values
type OptionsParser interface {
    // Parse common options from flags
    ParseCommonOptions(flags *pflag.FlagSet) (*CommonOptions, error)
    ParseOutputOptions(flags *pflag.FlagSet) (*OutputOptions, error)
    ParseSelectorOptions(flags *pflag.FlagSet) (*SelectorOptions, error)
    ParseFileOptions(flags *pflag.FlagSet) (*FileOptions, error)
    ParseApplyOptions(flags *pflag.FlagSet) (*ApplyOptions, error)
    ParseDeleteOptions(flags *pflag.FlagSet) (*DeleteOptions, error)
    
    // Parse complex operation requests
    ParseGetRequest(flags *pflag.FlagSet, args []string) (*GetRequest, error)
    ParseCreateRequest(flags *pflag.FlagSet, args []string) (*CreateRequest, error)
    ParseApplyRequest(flags *pflag.FlagSet, args []string) (*ApplyRequest, error)
    ParseDeleteRequest(flags *pflag.FlagSet, args []string) (*DeleteRequest, error)
}

// CommonOptions contains common command options
type CommonOptions struct {
    Namespace string
    Context   string
    Timeout   time.Duration
}

// OutputOptions contains output formatting options
type OutputOptions struct {
    Format     string
    NoHeaders  bool
    ShowLabels bool
    SortBy     []string
}

// SelectorOptions contains resource selection options
type SelectorOptions struct {
    LabelSelector string
    FieldSelector string
    AllNamespaces bool
}

// FileOptions contains file input options
type FileOptions struct {
    Filenames    []string
    Recursive    bool
    FromLiteral  string
}

// ApplyOptions contains apply-specific options
type ApplyOptions struct {
    DryRun       string
    Force        bool
    Validate     bool
    FieldManager string
}

// DeleteOptions contains delete-specific options
type DeleteOptions struct {
    Force           bool
    GracePeriod     int64
    IgnoreNotFound  bool
    Wait            bool
}

// NewOptionsParser creates a new options parser
func NewOptionsParser() OptionsParser {
    return &optionsParser{}
}

type optionsParser struct{}

func (p *optionsParser) ParseCommonOptions(flags *pflag.FlagSet) (*CommonOptions, error) {
    namespace, _ := flags.GetString("namespace")
    context, _ := flags.GetString("context")
    timeout, _ := flags.GetDuration("timeout")
    
    return &CommonOptions{
        Namespace: namespace,
        Context:   context,
        Timeout:   timeout,
    }, nil
}

func (p *optionsParser) ParseOutputOptions(flags *pflag.FlagSet) (*OutputOptions, error) {
    format, _ := flags.GetString("output")
    noHeaders, _ := flags.GetBool("no-headers")
    showLabels, _ := flags.GetBool("show-labels")
    sortBy, _ := flags.GetStringSlice("sort-by")
    
    return &OutputOptions{
        Format:     format,
        NoHeaders:  noHeaders,
        ShowLabels: showLabels,
        SortBy:     sortBy,
    }, nil
}

// ParseGetRequest creates a GetRequest from flags and args
func (p *optionsParser) ParseGetRequest(flags *pflag.FlagSet, args []string) (*GetRequest, error) {
    // Parse individual option groups
    commonOpts, _ := p.ParseCommonOptions(flags)
    outputOpts, _ := p.ParseOutputOptions(flags)
    selectorOpts, _ := p.ParseSelectorOptions(flags)
    
    req := &GetRequest{
        Namespace:     commonOpts.Namespace,
        OutputFormat:  outputOpts.Format,
        NoHeaders:     outputOpts.NoHeaders,
        ShowLabels:    outputOpts.ShowLabels,
        LabelSelector: selectorOpts.LabelSelector,
        FieldSelector: selectorOpts.FieldSelector,
        AllNamespaces: selectorOpts.AllNamespaces,
    }
    
    // Parse positional arguments
    if len(args) >= 1 {
        req.ResourceType = args[0]
    }
    if len(args) >= 2 {
        req.ResourceName = args[1]
    }
    
    return req, nil
}
```

## 3. Output Formatters & Printer System

### Printer Factory Interface

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

// NewPrinterFactory creates a new printer factory
func NewPrinterFactory() PrinterFactory {
    return &printerFactory{}
}

type printerFactory struct{}

func (pf *printerFactory) PrinterForFormat(format string) (Printer, error) {
    switch format {
    case "table", "":
        return NewTablePrinter(), nil
    case "yaml":
        return NewYAMLPrinter(), nil
    case "json":
        return NewJSONPrinter(), nil
    case "name":
        return NewNamePrinter(), nil
    default:
        return nil, fmt.Errorf("unsupported output format: %s", format)
    }
}

func (pf *printerFactory) SupportedFormats() []string {
    return []string{"table", "yaml", "json", "name", "custom-columns"}
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
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// tablePrinter implements kubectl-compatible table output
type tablePrinter struct {
    showHeaders bool
    showLabels  bool
    sortBy      []string
}

// NewTablePrinter creates a new table printer
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

func (tp *tablePrinter) WithSortBy(columns []string) TablePrinter {
    tp.sortBy = columns
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

func (tp *tablePrinter) PrintObject(obj runtime.Object, w io.Writer) error {
    return tp.PrintObjects([]runtime.Object{obj}, w)
}

func (tp *tablePrinter) printHeaders(obj runtime.Object, w io.Writer) error {
    headers := []string{"NAME", "READY", "STATUS", "RESTARTS", "AGE"}
    
    if tp.showLabels {
        headers = append(headers, "LABELS")
    }
    
    fmt.Fprintf(w, "%s\n", strings.Join(headers, "\t"))
    return nil
}

func (tp *tablePrinter) printObjectRow(obj runtime.Object, w io.Writer) error {
    // Extract object metadata
    objMeta, err := metav1.ObjectMetaFor(obj)
    if err != nil {
        return err
    }
    
    // Basic columns
    columns := []string{
        objMeta.GetName(),
        "1/1",           // Ready (placeholder)
        "Running",       // Status (placeholder)
        "0",             // Restarts (placeholder)  
        "5m",            // Age (placeholder - should calculate from CreationTimestamp)
    }
    
    if tp.showLabels {
        labels := formatLabels(objMeta.GetLabels())
        columns = append(columns, labels)
    }
    
    fmt.Fprintf(w, "%s\n", strings.Join(columns, "\t"))
    return nil
}

func formatLabels(labels map[string]string) string {
    if len(labels) == 0 {
        return "<none>"
    }
    
    var pairs []string
    for k, v := range labels {
        pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
    }
    return strings.Join(pairs, ",")
}
```

## 4. Usage Example

### Complete CLI Integration Example

```go
// Example: Using CLI-Runtime in a Cobra application
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/spf13/cobra"
    "github.com/spf13/pflag"
    
    cliruntime "github.com/dtomasi/k1s/core/pkg/cli-runtime"
    "github.com/dtomasi/k1s/core/pkg/cli-runtime/flags"
    "github.com/dtomasi/k1s/core/pkg/cli-runtime/options"
    "github.com/dtomasi/k1s/core/pkg/cli-runtime/handlers"
    "github.com/dtomasi/k1s/core/pkg/cli-runtime/printers"
)

// CLI application using k1s CLI-Runtime
type K1sCLI struct {
    bootstrap       cliruntime.Bootstrap
    flagSets        flags.FlagSets
    optionsParser   options.OptionsParser
    operationHandler handlers.OperationHandler
    printerFactory  printers.PrinterFactory
}

func NewK1sCLI() *K1sCLI {
    return &K1sCLI{
        bootstrap:     cliruntime.NewBootstrap(),
        flagSets:      flags.NewFlagSets(),
        optionsParser: options.NewOptionsParser(),
        printerFactory: printers.NewPrinterFactory(),
    }
}

func (cli *K1sCLI) Initialize(config cliruntime.Config) error {
    // Bootstrap k1s runtime
    if err := cli.bootstrap.Initialize(config); err != nil {
        return fmt.Errorf("failed to initialize k1s: %w", err)
    }
    
    // Create operation handler with initialized client
    cli.operationHandler = handlers.NewOperationHandler(
        cli.bootstrap.GetClient(),
        cli.printerFactory,
    )
    
    return nil
}

func (cli *K1sCLI) NewGetCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:     "get [TYPE] [NAME]",
        Short:   "Display one or many resources",
        Example: "  k1s get items\n  k1s get item my-item -o yaml",
        RunE:    cli.runGetCommand,
    }
    
    // Add reusable flag sets
    cmd.Flags().AddFlagSet(cli.flagSets.CommonFlags())
    cmd.Flags().AddFlagSet(cli.flagSets.OutputFlags())
    cmd.Flags().AddFlagSet(cli.flagSets.SelectorFlags())
    
    return cmd
}

func (cli *K1sCLI) runGetCommand(cmd *cobra.Command, args []string) error {
    // Parse request from flags and args
    req, err := cli.optionsParser.ParseGetRequest(cmd.Flags(), args)
    if err != nil {
        return err
    }
    
    // Set output writer
    req.Output = cmd.OutOrStdout()
    
    // Execute operation
    resp, err := cli.operationHandler.HandleGet(context.Background(), *req)
    if err != nil {
        return err
    }
    
    // Success feedback
    if !resp.Printed {
        fmt.Fprintf(cmd.OutOrStdout(), "Found %d resources\n", resp.Count)
    }
    
    return nil
}

func main() {
    cli := NewK1sCLI()
    
    // Initialize k1s runtime
    config := cliruntime.Config{
        Storage: storage.Config{
            Type: "bolt",
            Path: "./data/k1s.db",
            TenantConfig: storage.TenantConfig{
                Prefix: "my-app",
            },
        },
        Timeout: 30 * time.Second,
    }
    
    if err := cli.Initialize(config); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
    defer cli.bootstrap.Shutdown()
    
    // Create root command
    rootCmd := &cobra.Command{
        Use:   "k1s",
        Short: "k1s CLI tool",
    }
    
    // Add commands using CLI-Runtime
    rootCmd.AddCommand(
        cli.NewGetCommand(),
        // cli.NewCreateCommand(),
        // cli.NewApplyCommand(),
        // cli.NewDeleteCommand(),
    )
    
    // Execute
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

<function_calls>
<invoke name="TodoWrite">
<parameter name="todos">[{"content": "Design CLI-Runtime package structure and interfaces", "status": "completed", "activeForm": "Designing CLI-Runtime package structure"}, {"content": "Specify client provisioning and runtime bootstrapping", "status": "in_progress", "activeForm": "Specifying client provisioning"}, {"content": "Design command factories for kubectl-compatible operations", "status": "pending", "activeForm": "Designing command factories"}, {"content": "Specify output formatters and printer system", "status": "pending", "activeForm": "Specifying output formatters"}, {"content": "Create comprehensive CLI-Runtime documentation", "status": "pending", "activeForm": "Creating CLI-Runtime documentation"}]