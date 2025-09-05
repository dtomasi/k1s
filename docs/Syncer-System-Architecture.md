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

## Sync Status Tracking and Resource Management

### Resource Status Integration

The syncer system automatically tracks synchronization metadata through optional resource status updates. This enables intelligent sync optimizations and provides visibility into sync operations.

```go
// Resource status annotations added by syncer
type SyncStatus struct {
    LastSyncTime    *metav1.Time `json:"lastSyncTime,omitempty"`
    ContentHash     string       `json:"contentHash,omitempty"`
    SourceName      string       `json:"sourceName,omitempty"`
    SourceType      string       `json:"sourceType,omitempty"`
    SyncGeneration  int64        `json:"syncGeneration,omitempty"`
}
```

### Multi-Instance Resource Status Management

To support multiple k1s instances syncing the same resource without conflicts, the syncer uses instance-specific status tracking:

```go
const (
    // Core sync annotations - instance-specific
    AnnotationSyncSource      = "sync.k1s.io/source"
    AnnotationSyncType        = "sync.k1s.io/source-type"  
    AnnotationSyncTime        = "sync.k1s.io/last-sync"
    AnnotationContentHash     = "sync.k1s.io/content-hash"
    AnnotationSyncGeneration  = "sync.k1s.io/generation"
    AnnotationInstanceID      = "sync.k1s.io/instance-id"     // New: unique k1s instance ID
    
    // Optional status tracking
    AnnotationSyncEnabled     = "sync.k1s.io/enabled"
    AnnotationSyncPath        = "sync.k1s.io/source-path"
)

// Instance-specific annotation format for multiple k1s instances
const (
    // Pattern: sync.k1s.io/{instance-id}/property
    AnnotationInstancePrefix  = "sync.k1s.io/"
    
    // Examples:
    // sync.k1s.io/dev-cluster-001/last-sync
    // sync.k1s.io/prod-cluster-002/content-hash
    // sync.k1s.io/local-dev-alice/source
)
```

### Instance ID Generation

Each k1s runtime instance generates or is configured with a unique identifier:

```go
// Runtime configuration with instance ID
type RuntimeOptions struct {
    InstanceID string  // Optional: custom instance ID
    // ... other options
}

// Instance ID generation strategies
func generateInstanceID() string {
    hostname, _ := os.Hostname()
    timestamp := time.Now().Unix()
    random := rand.Int31()
    
    // Format: hostname-timestamp-random
    return fmt.Sprintf("%s-%d-%d", hostname, timestamp, random)
}

// Usage in syncer config
type Config struct {
    InstanceID string           // Required: unique instance identifier
    Sources    []SourceConfig
    Options    Options
}
```

### Intelligent Sync Optimization with Multi-Instance Support

Content hashing enables skip-on-no-change behavior while supporting multiple k1s instances:

```go
// Sync source implementation with instance-specific tracking
func (f *FilesystemSource) shouldSync(resource *unstructured.Unstructured, content []byte) bool {
    // Calculate content hash
    currentHash := sha256.Sum256(content)
    hashString := hex.EncodeToString(currentHash[:])
    
    // Check instance-specific hash annotation
    annotations := resource.GetAnnotations()
    if annotations != nil {
        instanceKey := fmt.Sprintf("%s%s/content-hash", AnnotationInstancePrefix, f.instanceID)
        if existingHash := annotations[instanceKey]; existingHash == hashString {
            // Content unchanged for this instance - skip sync
            return false
        }
    }
    
    return true
}

func (f *FilesystemSource) syncResource(resource *unstructured.Unstructured, content []byte) error {
    // Update instance-specific sync annotations
    annotations := resource.GetAnnotations()
    if annotations == nil {
        annotations = make(map[string]string)
    }
    
    // Instance-specific annotations prevent conflicts
    instancePrefix := fmt.Sprintf("%s%s", AnnotationInstancePrefix, f.instanceID)
    
    annotations[instancePrefix+"/source"] = f.Name()
    annotations[instancePrefix+"/source-type"] = f.Type()
    annotations[instancePrefix+"/last-sync"] = time.Now().Format(time.RFC3339)
    annotations[instancePrefix+"/content-hash"] = f.calculateHash(content)
    annotations[instancePrefix+"/generation"] = strconv.FormatInt(f.generation, 10)
    annotations[instancePrefix+"/source-path"] = f.Path
    
    // Global instance tracking (for discovery)
    annotations[AnnotationInstanceID] = f.instanceID
    
    resource.SetAnnotations(annotations)
    
    return f.client.Update(context.TODO(), resource)
}

// Multi-instance status reading
func (f *FilesystemSource) getInstanceSyncStatus(resource *unstructured.Unstructured) map[string]InstanceSyncInfo {
    annotations := resource.GetAnnotations()
    instances := make(map[string]InstanceSyncInfo)
    
    for key, value := range annotations {
        if strings.HasPrefix(key, AnnotationInstancePrefix) {
            parts := strings.Split(strings.TrimPrefix(key, AnnotationInstancePrefix), "/")
            if len(parts) >= 2 {
                instanceID := parts[0]
                property := parts[1]
                
                if _, exists := instances[instanceID]; !exists {
                    instances[instanceID] = InstanceSyncInfo{}
                }
                
                info := instances[instanceID]
                switch property {
                case "last-sync":
                    info.LastSyncTime = value
                case "content-hash":
                    info.ContentHash = value
                case "source":
                    info.SourceName = value
                case "source-type":
                    info.SourceType = value
                }
                instances[instanceID] = info
            }
        }
    }
    
    return instances
}

type InstanceSyncInfo struct {
    LastSyncTime string
    ContentHash  string
    SourceName   string
    SourceType   string
}
```

### Multi-Instance Status Reporting

Sources can optionally update resource status with instance-specific sync information:

```go
type ResourceSyncStatus struct {
    // Standard Kubernetes status fields
    Conditions []metav1.Condition `json:"conditions,omitempty"`
    
    // Multi-instance sync tracking
    Instances  map[string]InstanceStatus `json:"instances,omitempty"`      // Instance-specific status
    Summary    SyncSummary              `json:"summary,omitempty"`        // Overall sync summary
}

type InstanceStatus struct {
    InstanceID     string      `json:"instanceId"`
    LastSyncResult string      `json:"lastSyncResult"`               // "Success", "Failed", "Skipped"
    LastSyncTime   metav1.Time `json:"lastSyncTime"`
    SyncHistory    []SyncEntry `json:"syncHistory,omitempty"`        // Recent sync attempts
    SourceInfo     SourceInfo  `json:"sourceInfo"`                   // Source metadata
    ContentHash    string      `json:"contentHash,omitempty"`        // Current content hash
}

type SyncSummary struct {
    TotalInstances   int                    `json:"totalInstances"`
    ActiveInstances  int                    `json:"activeInstances"`      // Recently synced
    LastActivity     metav1.Time           `json:"lastActivity"`
    ConsensusHash    string                `json:"consensusHash,omitempty"` // Hash agreed by majority
}

type SyncEntry struct {
    Timestamp  metav1.Time `json:"timestamp"`
    Result     string      `json:"result"`
    Message    string      `json:"message,omitempty"`
    Generation int64       `json:"generation"`
}

type SourceInfo struct {
    Name string `json:"name"`
    Type string `json:"type"`
    Path string `json:"path,omitempty"`
}

// Instance-specific status update
func (f *FilesystemSource) updateSyncStatus(resource *unstructured.Unstructured, result string, message string) error {
    // Get existing status or create new
    existingStatus := &ResourceSyncStatus{}
    if statusField := resource.Object["status"]; statusField != nil {
        // Parse existing status
    }
    
    // Update this instance's status
    if existingStatus.Instances == nil {
        existingStatus.Instances = make(map[string]InstanceStatus)
    }
    
    instanceStatus := InstanceStatus{
        InstanceID:     f.instanceID,
        LastSyncResult: result,
        LastSyncTime:   metav1.Now(),
        SourceInfo: SourceInfo{
            Name: f.Name(),
            Type: f.Type(),
            Path: f.Path,
        },
        ContentHash: f.currentContentHash,
        SyncHistory: []SyncEntry{
            {
                Timestamp:  metav1.Now(),
                Result:     result,
                Message:    message,
                Generation: f.generation,
            },
        },
    }
    
    // Preserve existing history (limited to last N entries)
    if existing, exists := existingStatus.Instances[f.instanceID]; exists {
        // Keep last 5 history entries
        maxHistory := 5
        if len(existing.SyncHistory) > 0 {
            instanceStatus.SyncHistory = append(instanceStatus.SyncHistory, existing.SyncHistory...)
            if len(instanceStatus.SyncHistory) > maxHistory {
                instanceStatus.SyncHistory = instanceStatus.SyncHistory[:maxHistory]
            }
        }
    }
    
    existingStatus.Instances[f.instanceID] = instanceStatus
    
    // Update summary information
    existingStatus.Summary = f.calculateSummary(existingStatus.Instances)
    
    // Update resource status subresource  
    resource.Object["status"] = existingStatus
    return f.client.Status().Update(context.TODO(), resource)
}

func (f *FilesystemSource) calculateSummary(instances map[string]InstanceStatus) SyncSummary {
    summary := SyncSummary{
        TotalInstances: len(instances),
    }
    
    now := time.Now()
    activeThreshold := 5 * time.Minute // Consider instance active if synced within 5min
    hashCounts := make(map[string]int)
    
    var lastActivity time.Time
    
    for _, instance := range instances {
        // Check if instance is active
        if now.Sub(instance.LastSyncTime.Time) < activeThreshold {
            summary.ActiveInstances++
        }
        
        // Track latest activity
        if instance.LastSyncTime.After(lastActivity) {
            lastActivity = instance.LastSyncTime.Time
        }
        
        // Count content hashes for consensus
        if instance.ContentHash != "" {
            hashCounts[instance.ContentHash]++
        }
    }
    
    summary.LastActivity = metav1.NewTime(lastActivity)
    
    // Determine consensus hash (most common hash)
    var consensusHash string
    maxCount := 0
    for hash, count := range hashCounts {
        if count > maxCount {
            maxCount = count
            consensusHash = hash
        }
    }
    summary.ConsensusHash = consensusHash
    
    return summary
}
```

## Event System Integration

Sync events are integrated with the k1s event system and include status information:

```go
type SyncEvent struct {
    Source      string                         // Source name
    Type        SyncEventType                  // Created, Updated, Deleted, Error, Skipped
    Resource    *unstructured.Unstructured     // Kubernetes resource
    ContentHash string                         // Content hash for optimization
    Skipped     bool                           // True if sync was skipped (no changes)
    Error       error                          // Error details if applicable
}

type SyncEventType string

const (
    SyncEventCreated SyncEventType = "Created"
    SyncEventUpdated SyncEventType = "Updated" 
    SyncEventDeleted SyncEventType = "Deleted"
    SyncEventSkipped SyncEventType = "Skipped"   // New: content unchanged
    SyncEventError   SyncEventType = "Error"
)
```

Events are recorded for:
- Successful resource synchronization with timing information
- Skipped syncs due to unchanged content
- Conflict resolution actions with before/after state
- Sync errors and warnings with retry information  
- Source connectivity issues and recovery

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

### Intelligent Sync Optimizations
- **Content Hashing**: SHA-256 content hashing prevents unnecessary resource updates
- **Skip-on-No-Change**: Resources with unchanged content hashes are automatically skipped
- **Generation Tracking**: Sync generation numbers enable efficient state management
- **Annotation Caching**: Previous sync metadata is preserved to optimize repeat syncs

### Filesystem Sources
- Uses efficient fsnotify for file watching
- Batches multiple file changes in rapid succession  
- Supports configurable debounce intervals
- **Hash-based Skip Logic**: File content hashing prevents redundant YAML parsing
- **Incremental Sync**: Only processes files that have changed since last sync

### Kubernetes Sources
- Leverages Kubernetes watch APIs for real-time updates
- Implements exponential backoff for connection failures
- Supports efficient resource filtering to reduce network traffic
- **Resource Version Tracking**: Uses Kubernetes resource versions for change detection
- **Selective Sync**: Only syncs resources that have actually changed in the cluster

### Memory Management
- Sources implement proper cleanup in Stop() methods
- Event channels are properly closed to prevent goroutine leaks
- Configurable buffer sizes for high-throughput scenarios
- **Status History Pruning**: Configurable retention of sync history entries
- **Hash Cache Management**: Efficient in-memory content hash caching

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
- **Content hash hit/miss ratios**: Efficiency of skip-on-no-change optimization
- **Sync skip rates**: Percentage of resources skipped due to unchanged content
- **Status update frequency**: Rate of resource status updates

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