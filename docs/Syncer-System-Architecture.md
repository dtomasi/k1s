# K1S Syncer System Architecture

## Overview

The K1S Syncer System provides automatic synchronization of resources from external sources into the k1s runtime. This enables seamless integration with existing workflows, GitOps patterns, and migration scenarios from Kubernetes clusters or other configuration sources.

## Design Philosophy

The syncer system follows these core principles:

1. **Pure Dependency Injection**: Core runtime defines only interfaces, implementations are injected
2. **Modular Architecture**: Sync sources are separate Go modules under `sync/<source>`
3. **Zero Core Dependencies**: The core runtime has no knowledge of concrete sync implementations
4. **Optional Integration**: Syncing is completely optional and disabled by default
5. **Extensible Design**: Users can implement custom sync sources

## Architecture Components

### Core Interfaces

The core runtime defines only the essential interfaces and configuration types:

```go
// core/pkg/syncer/interface.go
type Source interface {
    Initialize(ctx context.Context, client client.Client) error
    Start(ctx context.Context) (<-chan SyncEvent, error)
    Stop() error
    Sync(ctx context.Context) error
    Name() string
    Type() string
}

type Config struct {
    Sources []SourceConfig
    Options Options
}

type SourceConfig struct {
    Name    string
    Source  Source  // Injected implementation
    Enabled bool
}
```

### Module Structure

```
k1s/
├── core/                    # Core runtime - interface definitions only
│   └── pkg/syncer/
│       ├── interface.go     # Source interface
│       ├── manager.go       # Runtime manager
│       └── config.go        # Configuration types
├── sync/                    # Sync source implementations
│   ├── filesystem/          # Local file system synchronization
│   │   ├── go.mod
│   │   └── pkg/filesystem/source.go
│   ├── kubernetes/          # Remote Kubernetes cluster sync
│   │   ├── go.mod
│   │   └── pkg/kubernetes/source.go
│   └── gogetter/           # HashiCorp go-getter integration
│       ├── go.mod
│       └── pkg/gogetter/source.go
```

## Sync Source Implementations

### Filesystem Sync (`sync/filesystem`)

Synchronizes YAML/JSON resources from local filesystem paths.

**Features:**
- Glob pattern matching (`*.yaml`, `*.yml`)
- Recursive directory traversal
- Real-time file watching with fsnotify
- Support for nested directory structures

**Dependencies:**
- `github.com/fsnotify/fsnotify`
- `sigs.k8s.io/yaml`

**Example Usage:**
```go
import filesystem "github.com/dtomasi/k1s/sync/filesystem/pkg/filesystem"

fsSource := filesystem.New("./manifests")
// or with custom options
fsSource := filesystem.NewWithOptions(
    "./config",
    []string{"*.yaml", "*.json"},
    true,  // recursive
    true,  // watch
)
```

### Kubernetes Sync (`sync/kubernetes`)

Synchronizes resources from remote Kubernetes clusters.

**Features:**
- Multi-context cluster support
- Namespace filtering
- Resource type filtering
- Label-based resource selection
- Real-time watch for cluster changes

**Dependencies:**
- `k8s.io/client-go`
- `k8s.io/apimachinery`

**Example Usage:**
```go
import kubernetes "github.com/dtomasi/k1s/sync/kubernetes/pkg/kubernetes"

k8sSource := kubernetes.New("production", "inventory")
// with resource filtering
k8sSource := kubernetes.NewWithOptions("staging", "default", []string{"items", "categories"})
```

### Go-Getter Sync (`sync/gogetter`)

Leverages HashiCorp's go-getter for fetching resources from various remote sources.

**Supported Sources:**
- Git repositories (`git::https://github.com/user/repo`)
- HTTP/HTTPS URLs
- S3 buckets
- Local file paths
- Archive files (tar, zip)

**Dependencies:**
- `github.com/hashicorp/go-getter`

**Example Usage:**
```go
import gogetter "github.com/dtomasi/k1s/sync/gogetter/pkg/gogetter"

// Git repository
gitSource := gogetter.New("git::https://github.com/company/k1s-configs")

// HTTP endpoint
httpSource := gogetter.New("https://config-server.company.com/k1s-configs.tar.gz")
```

## Runtime Integration

### Configuration

The syncer is configured through the runtime options pattern:

```go
import (
    "github.com/dtomasi/k1s/core/pkg/runtime"
    "github.com/dtomasi/k1s/core/pkg/syncer"
    filesystem "github.com/dtomasi/k1s/sync/filesystem/pkg/filesystem"
    kubernetes "github.com/dtomasi/k1s/sync/kubernetes/pkg/kubernetes"
)

func main() {
    var sources []syncer.SourceConfig
    
    // Add filesystem source
    if enableLocalSync {
        fsSource := filesystem.New("./config")
        sources = append(sources, syncer.SourceConfig{
            Name:    "local-configs",
            Source:  fsSource,
            Enabled: true,
        })
    }
    
    // Add Kubernetes source  
    if enableK8sSync {
        k8sSource := kubernetes.New("staging", "inventory")
        sources = append(sources, syncer.SourceConfig{
            Name:    "staging-cluster",
            Source:  k8sSource,
            Enabled: true,
        })
    }
    
    // Configure runtime with syncer
    var opts []runtime.Option
    if len(sources) > 0 {
        syncerConfig := &syncer.Config{
            Sources: sources,
            Options: syncer.Options{
                ConflictResolution: syncer.ConflictSourceWins,
                Interval:          30 * time.Second,
            },
        }
        opts = append(opts, runtime.WithSyncer(syncerConfig))
    }
    
    rt, err := runtime.New(opts...)
    if err != nil {
        log.Fatal(err)
    }
    
    if err := rt.Start(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

### Lifecycle Management

The syncer manager integrates with the k1s runtime lifecycle:

1. **Initialization**: Sources are initialized with the k1s client
2. **Start**: Each enabled source begins synchronization
3. **Runtime**: Continuous monitoring and event processing
4. **Shutdown**: Graceful cleanup of watchers and connections

### Conflict Resolution

The syncer supports multiple conflict resolution strategies:

- **SourceWins** (default): Source changes override local modifications
- **LocalWins**: Local changes are preserved, source changes ignored
- **MergeStrategy**: Attempt to merge changes (implementation-dependent)
- **PromptUser**: Interactive resolution (CLI environments only)

## Event System Integration

Sync events are integrated with the k1s event system:

```go
type SyncEvent struct {
    Source   string                         // Source name
    Type     SyncEventType                  // Created, Updated, Deleted, Error
    Resource *unstructured.Unstructured     // Kubernetes resource
    Error    error                          // Error details if applicable
}
```

Events are recorded for:
- Successful resource synchronization
- Conflict resolution actions
- Sync errors and warnings
- Source connectivity issues

## Extensibility

### Custom Sync Sources

Users can implement custom sync sources by implementing the `Source` interface:

```go
type DatabaseSource struct {
    ConnectionString string
    Query           string
    client          client.Client
}

func (d *DatabaseSource) Initialize(ctx context.Context, client client.Client) error {
    d.client = client
    // Setup database connection
    return nil
}

func (d *DatabaseSource) Start(ctx context.Context) (<-chan syncer.SyncEvent, error) {
    events := make(chan syncer.SyncEvent)
    // Implement polling logic
    return events, nil
}

// ... implement other interface methods
```

### Plugin Architecture

Future versions may support plugin-based sync sources through:
- Go plugin system
- WebAssembly modules
- gRPC-based external processes

## Performance Considerations

### Filesystem Sources
- Uses efficient fsnotify for file watching
- Batches multiple file changes in rapid succession
- Supports configurable debounce intervals

### Kubernetes Sources
- Leverages Kubernetes watch APIs for real-time updates
- Implements exponential backoff for connection failures
- Supports efficient resource filtering to reduce network traffic

### Memory Management
- Sources implement proper cleanup in Stop() methods
- Event channels are properly closed to prevent goroutine leaks
- Configurable buffer sizes for high-throughput scenarios

## Security Considerations

### Authentication
- Kubernetes sources use standard kubeconfig authentication
- Go-getter sources support credential providers
- Filesystem sources respect file system permissions

### Access Control
- Sources can implement namespace-based filtering
- Label selectors provide fine-grained resource filtering
- Integration with k1s RBAC system (when enabled)

### Validation
- All synced resources undergo standard k1s validation
- Malformed resources are rejected with appropriate error events
- Schema validation prevents incompatible resource types

## Monitoring and Observability

### Metrics
- Sync operation success/failure rates
- Resource processing latency
- Source connectivity status
- Conflict resolution statistics

### Logging
- Structured logging for all sync operations
- Configurable log levels per source
- Integration with k1s audit logging system

### Health Checks
- Source health monitoring
- Automatic retry with exponential backoff
- Circuit breaker patterns for failing sources

## Future Enhancements

### Planned Features
- **Bidirectional Sync**: Push local changes back to sources
- **Multi-Master Sync**: Coordinate between multiple k1s instances
- **Change Batching**: Optimize high-frequency change scenarios
- **Schema Evolution**: Handle API version migrations automatically

### Potential Integrations
- **ArgoCD Integration**: GitOps workflow compatibility
- **Helm Integration**: Chart-based resource templating
- **Kustomize Integration**: Declarative configuration management
- **Vault Integration**: Secret synchronization and rotation

## Migration Path

For existing k1s deployments:

1. **Phase 1**: Add syncer configuration to existing runtime initialization
2. **Phase 2**: Enable filesystem sync for local development workflows
3. **Phase 3**: Implement Kubernetes sync for migration scenarios
4. **Phase 4**: Explore advanced sync sources based on requirements

The syncer system is designed to be completely backward compatible - existing k1s deployments continue to work without any syncer configuration.

## Implementation Priority

This system should be implemented after the core k1s runtime stabilizes:

**Dependencies:**
- Core runtime with storage backends ✅
- Client implementation ✅  
- Event system ✅
- Controller runtime (optional, for advanced features)

**Estimated Implementation Time:** 4-6 weeks
- Week 1-2: Core interface and manager implementation
- Week 3: Filesystem sync source
- Week 4: Kubernetes sync source  
- Week 5: Go-getter sync source
- Week 6: Integration testing and documentation

This syncer system transforms k1s from a standalone runtime into a powerful integration platform that can seamlessly connect with existing infrastructure and workflows.