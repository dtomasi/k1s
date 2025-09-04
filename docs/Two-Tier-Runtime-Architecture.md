# K1S Two-Tier Runtime Architecture

**Related Documentation:**
- [Architecture](Architecture.md) - Main k1s system architecture
- [Implementation Plan](Implementation-Plan.md) - Phased implementation strategy
- [CLI-Runtime Package](CLI-Runtime-Package.md) - CLI runtime specification
- [Controller-Runtime Package](Controller-Runtime-Package.md) - Controller runtime specification

## Overview

K1s implements a two-tier runtime architecture optimized for CLI performance while maintaining full compatibility with extended features. This architecture addresses the fundamental difference between basic CLI operations and advanced runtime requirements.

**Design Philosophy:** Most CLI operations (get, create, list, delete, patch) only need basic runtime components, while advanced features (watch, controllers, complex workflows) require full runtime initialization. The two-tier approach provides optimal performance for both scenarios.

## Architecture Tiers

### Tier 1: CoreClient (Standard CLI Runtime)

**Purpose:** Fast, lightweight client for standard CLI operations with plugin-awareness.

**Startup Time:** ~25ms  
**Memory Footprint:** Minimal (~10MB)  
**Use Cases:** get, create, list, delete, patch, apply operations

#### Components
```go
type CoreClient struct {
    // Core infrastructure
    storage         storage.Interface
    scheme          *runtime.Scheme
    registry        registry.Registry
    codecFactory    *codec.CodecFactory
    
    // Essential runtime components
    validator       validation.Validator
    defaulter       defaulting.Defaulter
    
    // Plugin awareness (lightweight)
    pluginDiscovery PluginDiscovery
    
    // Status handling
    statusWriter    StatusWriter
}
```

#### Plugin Discovery Integration
The CoreClient includes lightweight plugin discovery to handle CRDs from plugins without full plugin initialization:

```go
type PluginDiscovery interface {
    // Scan for available plugins and their types
    DiscoverAvailableTypes() ([]schema.GroupVersionKind, error)
    
    // Register discovered types in scheme (lazy loading)
    RegisterTypes(scheme *runtime.Scheme) error
    
    // Get metadata for specific type (for CLI operations)
    GetTypeMetadata(gvk schema.GroupVersionKind) (*TypeMetadata, error)
    
    // Check if type is provided by plugin
    IsPluginType(gvk schema.GroupVersionKind) bool
}

type TypeMetadata struct {
    ShortNames    []string
    PrintColumns  []PrintColumn
    Categories    []string
    Scope         string  // Namespaced or Cluster
}
```

**Discovery Process:**
1. **Startup:** Scan plugin directories for type manifests (not full plugin loading)
2. **Lazy Registration:** Register types in scheme only when accessed
3. **Metadata Caching:** Cache type metadata for subsequent operations
4. **Fallback:** Graceful degradation if plugins unavailable

#### Performance Characteristics
- **Cold Start:** ~25ms (including plugin discovery)
- **Warm Start:** ~10ms (cached plugin metadata)
- **Memory Usage:** ~10MB baseline
- **Throughput:** >5K operations/sec for basic CRUD

### Tier 2: ManagedRuntime (Full-Featured Runtime)

**Purpose:** Complete runtime environment for advanced features and background operations.

**Startup Time:** ~75ms  
**Memory Footprint:** Higher (~50MB)  
**Use Cases:** watch operations, controller execution, complex workflows, event processing

#### Components
```go
type ManagedRuntime struct {
    // Embedded CoreClient for all basic operations
    coreClient      *CoreClient
    
    // Advanced runtime components
    controllers     controller.Manager
    informers       informers.SharedInformerFactory
    events          events.Broadcaster
    workTracking    worktracking.Registry
    
    // Plugin management (full featured)
    pluginManager   plugins.Manager
    
    // Runtime lifecycle
    ctx             context.Context
    cancel          context.CancelFunc
}
```

#### Extended Capabilities
- **Controller Execution:** Full controller-runtime compatibility
- **Watch Operations:** Real-time resource change notifications
- **Event System:** Kubernetes-compatible event recording and broadcasting
- **Work Tracking:** Graceful shutdown with operation completion guarantees
- **Full Plugin Management:** Plugin lifecycle, sandboxing, and security

#### Performance Characteristics
- **Startup Time:** ~75ms (full component initialization)
- **Memory Usage:** ~50MB (controllers, informers, event queues)
- **Background Processing:** Controllers, informers, event broadcasting
- **Throughput:** >1K operations/sec (with full validation and event recording)

## API Design

### Unified Client Interface

Both tiers implement the same `Client` interface for seamless interoperability:

```go
type Client interface {
    // Standard CRUD operations
    Get(ctx context.Context, key ObjectKey, obj Object, opts ...GetOption) error
    List(ctx context.Context, list ObjectList, opts ...ListOption) error
    Create(ctx context.Context, obj Object, opts ...CreateOption) error
    Update(ctx context.Context, obj Object, opts ...UpdateOption) error
    Delete(ctx context.Context, obj Object, opts ...DeleteOption) error
    Patch(ctx context.Context, obj Object, patch Patch, opts ...PatchOption) error
    
    // Status operations
    Status() StatusWriter
    
    // Metadata
    Scheme() *runtime.Scheme
    RESTMapper() meta.RESTMapper
}
```

### Runtime Creation APIs

#### CoreClient Creation
```go
// Standard CLI client with plugin discovery
func NewCoreClient(opts CoreClientOptions) (Client, error)

type CoreClientOptions struct {
    Storage         storage.Interface
    Scheme          *runtime.Scheme
    Registry        registry.Registry
    Validator       validation.Validator    // optional
    Defaulter       defaulting.Defaulter    // optional
    PluginPaths     []string               // optional, defaults to standard paths
    SkipPlugins     bool                   // performance optimization flag
}

// Usage example
client, err := k1s.NewCoreClient(k1s.CoreClientOptions{
    Storage:  memoryStorage,
    Scheme:   k1sScheme,
    Registry: resourceRegistry,
})
```

#### ManagedRuntime Creation
```go
// Full runtime for advanced features
func NewManagedRuntime(opts ManagedRuntimeOptions) (*ManagedRuntime, error)

type ManagedRuntimeOptions struct {
    // All CoreClient options
    CoreClientOptions
    
    // Extended runtime options
    EnableControllers   bool
    EnableInformers     bool
    EnableEvents        bool
    PluginConfig       *plugins.Config
    WorkTrackingConfig *worktracking.Config
}

// Usage example
runtime, err := k1s.NewManagedRuntime(k1s.ManagedRuntimeOptions{
    CoreClientOptions: coreOpts,
    EnableControllers: true,
    EnableInformers:   true,
    EnableEvents:      true,
})

// Access the same Client interface
client := runtime.Client()
```

## Plugin Discovery Implementation

### Lightweight Discovery Process

The CoreClient implements a lightweight plugin discovery mechanism that scans for plugin metadata without loading full plugins:

#### 1. Plugin Type Manifest
Each plugin provides a `types.yaml` manifest:

```yaml
apiVersion: k1s.io/v1alpha1
kind: PluginTypeManifest
metadata:
  name: inventory-plugin
spec:
  types:
  - group: inventory.example.com
    version: v1alpha1
    kind: Item
    plural: items
    shortNames: [item, itm]
    namespaced: true
    printColumns:
    - name: NAME
      type: string
      description: Name of the item
      jsonPath: .metadata.name
    - name: QUANTITY
      type: integer  
      description: Available quantity
      jsonPath: .spec.quantity
  - group: inventory.example.com
    version: v1alpha1
    kind: Category
    plural: categories
    shortNames: [cat, category]
    namespaced: true
```

#### 2. Discovery Workflow
```go
func (d *pluginDiscovery) DiscoverAvailableTypes() ([]schema.GroupVersionKind, error) {
    var types []schema.GroupVersionKind
    
    // Scan standard plugin directories
    for _, pluginPath := range d.pluginPaths {
        manifests, err := d.scanPluginManifests(pluginPath)
        if err != nil {
            continue // graceful degradation
        }
        
        for _, manifest := range manifests {
            for _, typeInfo := range manifest.Spec.Types {
                gvk := schema.GroupVersionKind{
                    Group:   typeInfo.Group,
                    Version: typeInfo.Version,
                    Kind:    typeInfo.Kind,
                }
                types = append(types, gvk)
                
                // Cache metadata for CLI operations
                d.cache[gvk] = &TypeMetadata{
                    ShortNames:   typeInfo.ShortNames,
                    PrintColumns: typeInfo.PrintColumns,
                    Categories:   typeInfo.Categories,
                    Scope:        typeInfo.Scope,
                }
            }
        }
    }
    
    return types, nil
}
```

#### 3. Lazy Type Registration
Types are registered in the scheme only when first accessed:

```go
func (c *CoreClient) ensureTypeRegistered(gvk schema.GroupVersionKind) error {
    // Check if already registered
    if c.scheme.Recognizes(gvk) {
        return nil
    }
    
    // Check if this is a plugin type
    if !c.pluginDiscovery.IsPluginType(gvk) {
        return fmt.Errorf("unknown type: %s", gvk)
    }
    
    // Lazy load the type from plugin
    return c.pluginDiscovery.RegisterTypes(c.scheme, gvk)
}
```

## Usage Patterns

### Standard CLI Operations (CoreClient)

Most CLI tools should use CoreClient for optimal performance:

```go
// CLI command implementation
func runGetCommand(cmd *cobra.Command, args []string) error {
    // Fast client initialization (~25ms)
    client, err := k1s.NewCoreClient(k1s.CoreClientOptions{
        Storage:  getStorageBackend(),
        Scheme:   k1s.GetScheme(),
        Registry: k1s.GetRegistry(),
    })
    if err != nil {
        return err
    }
    
    // Works with both built-in and plugin types
    var item inventoryv1alpha1.Item
    key := client.ObjectKey{
        Namespace: namespace,
        Name:      itemName,
    }
    
    if err := client.Get(ctx, key, &item); err != nil {
        return err
    }
    
    // Output formatting handled by CLI-runtime package
    return outputItem(&item, outputFormat)
}
```

### Advanced Operations (ManagedRuntime)

For watch operations, controllers, or complex workflows:

```go
// Advanced CLI tool with watch capability
func runWatchCommand(cmd *cobra.Command, args []string) error {
    // Full runtime initialization (~75ms)
    runtime, err := k1s.NewManagedRuntime(k1s.ManagedRuntimeOptions{
        CoreClientOptions: k1s.CoreClientOptions{
            Storage:  getStorageBackend(),
            Scheme:   k1s.GetScheme(),
            Registry: k1s.GetRegistry(),
        },
        EnableInformers: true,
        EnableEvents:    true,
    })
    if err != nil {
        return err
    }
    defer runtime.Shutdown(ctx)
    
    // Start background components
    if err := runtime.Start(ctx); err != nil {
        return err
    }
    
    // Use informers for efficient watching
    informer := runtime.GetInformerFor(&inventoryv1alpha1.Item{})
    informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc: func(obj interface{}) {
            item := obj.(*inventoryv1alpha1.Item)
            fmt.Printf("Item added: %s\n", item.Name)
        },
        UpdateFunc: func(oldObj, newObj interface{}) {
            item := newObj.(*inventoryv1alpha1.Item)
            fmt.Printf("Item updated: %s\n", item.Name)
        },
        DeleteFunc: func(obj interface{}) {
            item := obj.(*inventoryv1alpha1.Item)
            fmt.Printf("Item deleted: %s\n", item.Name)
        },
    })
    
    // Wait for termination
    <-ctx.Done()
    return nil
}
```

## Performance Optimization Strategies

### CoreClient Optimizations

1. **Lazy Loading:**
   - Plugin types registered only when accessed
   - Validation/defaulting components optional
   - Codec factory created on-demand

2. **Caching Strategy:**
   - Plugin metadata cached after first discovery
   - Scheme registration cached
   - GVK/GVR mappings cached

3. **Minimal Dependencies:**
   - No background goroutines
   - Direct storage access
   - Simplified error handling

### ManagedRuntime Optimizations

1. **Selective Initialization:**
   - Enable only needed components via options
   - Controllers started on-demand
   - Informers created lazily

2. **Resource Management:**
   - Graceful shutdown with work completion
   - Background goroutine lifecycle management
   - Memory pool reuse for high-frequency operations

3. **Plugin Integration:**
   - Full plugin lifecycle management
   - Sandboxed plugin execution
   - Resource limit enforcement

## Migration and Compatibility

### Existing Code Migration

Code using the current client implementation can migrate incrementally:

```go
// Before (current implementation)
client, err := client.NewClient(client.ClientOptions{...})

// After (CoreClient - recommended for CLI)
client, err := k1s.NewCoreClient(k1s.CoreClientOptions{...})

// After (ManagedRuntime - for advanced features)
runtime, err := k1s.NewManagedRuntime(k1s.ManagedRuntimeOptions{...})
client := runtime.Client() // Same interface
```

### Backward Compatibility

- All existing `Client` interface methods remain unchanged
- Current `NewClient` function can be aliased to `NewCoreClient`
- Configuration structures maintain backward compatibility
- Plugin types work transparently with both tiers

## Implementation Phases

### Phase 1: CoreClient Foundation (2 weeks)
- Implement CoreClient structure with existing components
- Create PluginDiscovery interface and basic implementation
- Add lazy type registration mechanism
- Performance benchmarking and optimization

### Phase 2: Plugin Discovery Enhancement (1 week)
- Implement plugin type manifest format
- Add plugin directory scanning
- Create metadata caching system
- Integration testing with example plugins

### Phase 3: ManagedRuntime Wrapper (1 week)
- Implement ManagedRuntime with CoreClient embedding
- Add controller, informer, and event system integration
- Implement runtime lifecycle management
- Advanced feature testing

### Phase 4: CLI Integration (1 week)
- Update CLI-runtime package for CoreClient usage
- Add performance flags and options
- Create migration documentation
- End-to-end integration testing

### Success Criteria

1. **Performance Targets:**
   - CoreClient startup: <25ms
   - ManagedRuntime startup: <75ms
   - Memory usage: CoreClient <10MB, ManagedRuntime <50MB
   - Throughput: CoreClient >5K ops/sec, ManagedRuntime >1K ops/sec

2. **Functionality Requirements:**
   - Both tiers implement identical Client interface
   - Plugin types work transparently in both tiers
   - All existing CLI operations maintain compatibility
   - Advanced features (watch, controllers) work in ManagedRuntime

3. **Code Quality:**
   - Zero breaking changes to existing Client interface
   - Comprehensive test coverage (>90%)
   - Zero golangci-lint errors
   - Complete documentation and examples

## Conclusion

The two-tier runtime architecture provides k1s with optimal performance for CLI operations while maintaining full compatibility with advanced runtime features. By separating basic client operations from advanced runtime functionality, k1s can deliver fast CLI experiences without sacrificing extensibility or plugin compatibility.

The lightweight plugin discovery mechanism ensures that CLI tools can work with plugin-provided types without the overhead of full plugin initialization, making k1s practical for real-world CLI applications while maintaining its extensible architecture.