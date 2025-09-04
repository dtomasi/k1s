# K1S Graceful Shutdown & Work Tracking Architecture

**Related Documentation:**
- [Architecture](Architecture.md) - Overall k1s system architecture
- [Controller-Runtime Package](Controller-Runtime-Package.md) - Controller runtime for CLI environments  
- [Implementation Plan](Implementation-Plan.md) - Phased implementation strategy

## Overview

The Graceful Shutdown & Work Tracking system ensures that all active operations complete before process termination, preventing data corruption and incomplete operations in CLI environments where controllers and plugins are running concurrently with the main process.

## Problem Statement

In CLI environments with controller-runtime and plugins, the main process can terminate before background work completes, leading to:

- **Data Corruption**: Incomplete write operations to storage
- **Inconsistent State**: Partially completed reconciliation loops
- **Resource Leaks**: Unclosed file handles, database connections
- **Lost Work**: In-progress operations that never complete

## Solution: Metrics-based Work Tracking

Instead of building a separate work registry, we leverage k1s's existing robust metrics infrastructure to track active operations and coordinate graceful shutdown.

## Design Principles

### 1. **Leverage Existing Infrastructure**
- Build on established `atomic.AddUint64()` and `atomic.LoadUint64()` patterns
- Reuse existing metrics collection in storage backends
- Zero performance impact - metrics already exist and are thread-safe

### 2. **Non-Intrusive Integration**
- Decorator pattern to wrap existing components
- No changes to existing storage backend implementations
- Backward compatibility maintained

### 3. **Real-time Visibility**
- Instant feedback on work status via atomic operations
- Structured logging for shutdown progress monitoring
- Observable metrics for debugging and monitoring

## Architecture Overview

```mermaid
graph TB
    subgraph "Graceful Shutdown Architecture"
        subgraph "Signal Handling"
            SIGTERM[SIGTERM/SIGINT<br/>Signal Reception]
            HANDLER[Shutdown Handler<br/>Context Cancellation]
        end
        
        subgraph "Work Tracking Layer"
            REGISTRY[Central Work Registry<br/>• Component Registration<br/>• Metrics Aggregation<br/>• Shutdown Coordination]
            
            subgraph "Enhanced Metrics"
                METRICS[Enhanced Metrics<br/>• operations (completed)<br/>• errors (failed)<br/>• watchers (long-running)<br/>• inFlight (active)<br/>• inFlightPeak (peak)<br/>• totalStarted (total)]
            end
        end
        
        subgraph "Component Integration"
            STORAGE_DEC[Storage Decorator<br/>• StartWork() / EndWork()<br/>• Metrics Integration]
            CLIENT_DEC[Client Decorator<br/>• CRUD Operation Tracking<br/>• Validation Work Tracking]
            CONTROLLER_DEC[Controller Decorator<br/>• Reconciliation Tracking<br/>• Event Processing Tracking]
        end
        
        subgraph "Existing Components"
            STORAGE[Storage Backends<br/>• Memory Storage<br/>• Pebble Storage]
            CLIENT[K1S Client<br/>• CRUD Operations<br/>• Status Updates]
            CONTROLLER[Controller Runtime<br/>• Manager<br/>• Reconcilers]
        end
        
        subgraph "Shutdown Sequence"
            STOP_NEW[Stop New Work<br/>• Reject new operations<br/>• Signal shutdown state]
            WAIT_COMPLETE[Wait for Completion<br/>• Monitor inFlight metrics<br/>• Progress logging<br/>• Timeout handling]
            FORCE_KILL[Force Termination<br/>• Hard kill after timeout<br/>• Resource cleanup]
        end
    end
    
    SIGTERM --> HANDLER
    HANDLER --> REGISTRY
    REGISTRY --> STOP_NEW
    STOP_NEW --> WAIT_COMPLETE
    WAIT_COMPLETE --> FORCE_KILL
    
    REGISTRY --> STORAGE_DEC
    REGISTRY --> CLIENT_DEC
    REGISTRY --> CONTROLLER_DEC
    
    STORAGE_DEC --> STORAGE
    CLIENT_DEC --> CLIENT
    CONTROLLER_DEC --> CONTROLLER
    
    STORAGE --> METRICS
    CLIENT --> METRICS
    CONTROLLER --> METRICS
```

## Enhanced Metrics System

### Current Metrics (Already Implemented)

Both storage backends already have robust metrics:

```go
type Metrics struct {
    operations uint64  // Completed work count
    errors     uint64  // Failed work count  
    watchers   uint64  // Long-running work count
}
```

### Enhanced Metrics (New Addition)

```go
type EnhancedMetrics struct {
    // Existing metrics (maintained for compatibility)
    operations    uint64  // Completed work
    errors        uint64  // Failed work  
    watchers      uint64  // Long-running work (watchers)
    
    // New in-flight tracking metrics
    inFlight      uint64  // Currently active operations
    inFlightPeak  uint64  // Peak concurrent operations
    totalStarted  uint64  // Total work started (for completion rates)
}

// Work lifecycle tracking methods
func (m *EnhancedMetrics) StartWork() uint64 {
    workID := atomic.AddUint64(&m.totalStarted, 1)
    current := atomic.AddUint64(&m.inFlight, 1)
    
    // Update peak if necessary (lock-free)
    for {
        peak := atomic.LoadUint64(&m.inFlightPeak)
        if current <= peak || atomic.CompareAndSwapUint64(&m.inFlightPeak, peak, current) {
            break
        }
    }
    return workID
}

func (m *EnhancedMetrics) EndWork(success bool) {
    atomic.AddUint64(&m.inFlight, ^uint64(0)) // atomic decrement
    if success {
        atomic.AddUint64(&m.operations, 1)
    } else {
        atomic.AddUint64(&m.errors, 1)
    }
}
```

## Component Integration Strategy

### 1. Storage Backend Integration

**Decorator Pattern for Zero-Impact Integration:**

```go
// MetricsDecorator wraps existing storage backends
type MetricsDecorator struct {
    k1sstorage.Backend
    metrics *EnhancedMetrics
    shutdown chan struct{}
}

func (d *MetricsDecorator) Create(ctx context.Context, key string, obj runtime.Object, out runtime.Object, ttl uint64) error {
    // Check shutdown state
    select {
    case <-d.shutdown:
        return fmt.Errorf("operation rejected: shutting down")
    default:
    }
    
    // Track work lifecycle
    workID := d.metrics.StartWork()
    defer func(success *bool) {
        d.metrics.EndWork(*success)
    }(&success)
    
    // Execute original operation
    err := d.Backend.Create(ctx, key, obj, out, ttl)
    success = (err == nil)
    return err
}

// Identical pattern for Get, List, Delete, Watch, etc.
```

### 2. Client Layer Integration

```go
type ClientDecorator struct {
    client.Client
    metrics *EnhancedMetrics
    shutdown chan struct{}
}

func (d *ClientDecorator) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
    select {
    case <-d.shutdown:
        return fmt.Errorf("operation rejected: shutting down")
    default:
    }
    
    workID := d.metrics.StartWork()
    defer func(success *bool) {
        d.metrics.EndWork(*success)
    }(&success)
    
    err := d.Client.Get(ctx, key, obj, opts...)
    success = (err == nil)
    return err
}
```

### 3. Controller Runtime Integration

```go
type ControllerManager struct {
    manager.Manager
    metrics *EnhancedMetrics
    shutdown chan struct{}
    activeReconciles sync.WaitGroup
}

func (cm *ControllerManager) Start(ctx context.Context) error {
    // Monitor shutdown signal
    go func() {
        <-ctx.Done()
        close(cm.shutdown) // Stop accepting new reconciliation requests
        
        // Wait for active reconciles to complete
        done := make(chan struct{})
        go func() {
            cm.activeReconciles.Wait()
            close(done)
        }()
        
        // Wait with timeout
        select {
        case <-done:
            log.Info("All reconciliation completed")
        case <-time.After(30 * time.Second):
            log.Warn("Reconciliation timeout, forcing shutdown")
        }
    }()
    
    return cm.Manager.Start(ctx)
}

// Reconciler wrapper
func (cm *ControllerManager) wrapReconciler(reconciler reconcile.Reconciler) reconcile.Reconciler {
    return reconcile.Func(func(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
        select {
        case <-cm.shutdown:
            return reconcile.Result{}, fmt.Errorf("reconciliation rejected: shutting down")
        default:
        }
        
        cm.activeReconciles.Add(1)
        defer cm.activeReconciles.Done()
        
        workID := cm.metrics.StartWork()
        defer func(success *bool) {
            cm.metrics.EndWork(*success)
        }(&success)
        
        result, err := reconciler.Reconcile(ctx, req)
        success = (err == nil)
        return result, err
    })
}
```

## Central Work Registry

### Registry Interface

```go
type WorkRegistry interface {
    // Component registration
    RegisterComponent(name string, component WorkTrackingComponent) error
    
    // Work status queries
    IsAnyWorkInProgress() bool
    GetActiveWorkCount() map[string]uint64
    
    // Shutdown coordination
    InitiateShutdown() error
    WaitForWorkCompletion(timeout time.Duration) error
}

type WorkTrackingComponent interface {
    IsWorkInProgress() bool
    GetActiveWorkCount() uint64
    StopAcceptingWork() error
}
```

### Registry Implementation

```go
type centralRegistry struct {
    mu         sync.RWMutex
    components map[string]WorkTrackingComponent
    shutdown   chan struct{}
    shutdownOnce sync.Once
}

func (r *centralRegistry) IsAnyWorkInProgress() bool {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    for _, component := range r.components {
        if component.IsWorkInProgress() {
            return true
        }
    }
    return false
}

func (r *centralRegistry) WaitForWorkCompletion(timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()
    
    for time.Now().Before(deadline) {
        if !r.IsAnyWorkInProgress() {
            return nil
        }
        
        // Log progress
        activeWork := r.GetActiveWorkCount()
        for name, count := range activeWork {
            if count > 0 {
                log.Info("Waiting for work completion", "component", name, "activeWork", count)
            }
        }
        
        select {
        case <-ticker.C:
            continue
        case <-time.After(time.Until(deadline)):
            return fmt.Errorf("work completion timeout")
        }
    }
    
    return fmt.Errorf("shutdown timeout exceeded")
}
```

## Graceful Shutdown Sequence

### 1. Signal Handler Setup

```go
func setupGracefulShutdown() context.Context {
    ctx, cancel := context.WithCancel(context.Background())
    
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    
    go func() {
        sig := <-c
        log.Info("Shutdown signal received", "signal", sig)
        cancel()
    }()
    
    return ctx
}
```

### 2. Runtime Integration

```go
func (r *Runtime) Start() error {
    // Setup signal handling
    ctx := setupGracefulShutdown()
    
    // Start components
    for _, component := range r.components {
        go component.Start(ctx)
    }
    
    // Wait for shutdown signal
    <-ctx.Done()
    log.Info("Shutdown signal received, initiating graceful shutdown")
    
    return r.gracefulShutdown()
}

func (r *Runtime) gracefulShutdown() error {
    // Phase 1: Stop accepting new work
    log.Info("Phase 1: Stopping acceptance of new work")
    for _, component := range r.components {
        if err := component.StopAcceptingWork(); err != nil {
            log.Error(err, "Failed to stop accepting work", "component", component.Name())
        }
    }
    
    // Phase 2: Wait for work completion
    log.Info("Phase 2: Waiting for active work completion")
    if err := r.workRegistry.WaitForWorkCompletion(30 * time.Second); err != nil {
        log.Warn("Work completion timeout", "error", err)
        return err
    }
    
    log.Info("Phase 3: All work completed successfully")
    return nil
}
```

## Performance Guarantees

### 1. **Zero Performance Impact**
- Built on existing atomic operations (already optimized)
- No additional locks or synchronization primitives
- Decorator pattern adds minimal function call overhead (<1%)

### 2. **Memory Efficiency**
- Single `EnhancedMetrics` struct per component (~64 bytes)
- No work item tracking - only counters
- Existing metrics infrastructure reused

### 3. **Scalability**
- Lock-free atomic operations scale to high concurrency
- No centralized bottlenecks during normal operation
- Shutdown coordination scales with number of components (not operations)

## Testing Strategy

### 1. **Unit Tests**
```go
func TestEnhancedMetrics_WorkLifecycle(t *testing.T) {
    metrics := &EnhancedMetrics{}
    
    // Start work
    workID := metrics.StartWork()
    assert.Equal(t, uint64(1), atomic.LoadUint64(&metrics.totalStarted))
    assert.Equal(t, uint64(1), atomic.LoadUint64(&metrics.inFlight))
    
    // End work successfully
    metrics.EndWork(true)
    assert.Equal(t, uint64(0), atomic.LoadUint64(&metrics.inFlight))
    assert.Equal(t, uint64(1), atomic.LoadUint64(&metrics.operations))
}
```

### 2. **Integration Tests**
```go
func TestGracefulShutdown_StorageOperations(t *testing.T) {
    registry := NewWorkRegistry()
    storage := NewMetricsDecorator(NewMemoryStorage(), registry)
    
    // Start background operations
    ctx, cancel := context.WithCancel(context.Background())
    
    go func() {
        for i := 0; i < 100; i++ {
            storage.Create(ctx, fmt.Sprintf("key-%d", i), &TestObject{}, nil, 0)
        }
    }()
    
    // Trigger shutdown
    time.Sleep(50 * time.Millisecond)
    cancel()
    
    // Verify graceful completion
    err := registry.WaitForWorkCompletion(1 * time.Second)
    assert.NoError(t, err)
    assert.False(t, registry.IsAnyWorkInProgress())
}
```

### 3. **Load Testing**
```go
func TestConcurrentWorkTracking(t *testing.T) {
    metrics := &EnhancedMetrics{}
    var wg sync.WaitGroup
    
    // Start 1000 concurrent operations
    for i := 0; i < 1000; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            workID := metrics.StartWork()
            time.Sleep(10 * time.Millisecond) // Simulate work
            metrics.EndWork(true)
        }()
    }
    
    wg.Wait()
    
    // Verify final state
    assert.Equal(t, uint64(0), atomic.LoadUint64(&metrics.inFlight))
    assert.Equal(t, uint64(1000), atomic.LoadUint64(&metrics.operations))
}
```

## Error Handling

### 1. **Timeout Scenarios**
- **Soft timeout**: Log warnings and continue waiting (configurable)
- **Hard timeout**: Force termination with error logging
- **Context cancellation**: Immediate termination with cleanup

### 2. **Component Failures**
- Individual component failures don't block shutdown
- Failed components logged but shutdown continues
- Error aggregation for troubleshooting

### 3. **Resource Cleanup**
- Database connections closed properly
- File handles released
- Memory freed (garbage collector assisted)

## Configuration

```go
type ShutdownConfig struct {
    // Graceful shutdown timeout before force kill
    GracefulTimeout time.Duration `default:"30s"`
    
    // Progress logging interval during shutdown
    ProgressInterval time.Duration `default:"500ms"`
    
    // Enable detailed work tracking metrics
    DetailedMetrics bool `default:"true"`
    
    // Maximum concurrent operations before backpressure
    MaxConcurrentOps uint64 `default:"1000"`
}
```

## Implementation Phases

### Phase 1: Enhanced Metrics Infrastructure (Week 1-2)
- Implement `EnhancedMetrics` struct and methods
- Add metrics to existing storage backends via decorator pattern
- Unit tests for metrics functionality

### Phase 2: Central Work Registry (Week 3)
- Implement work registry interface and central coordination
- Component registration and shutdown signaling
- Integration tests for multi-component scenarios

### Phase 3: Component Integration (Week 4-5)
- Storage backend decorator implementation
- Client layer decorator implementation
- Controller runtime integration

### Phase 4: Runtime Integration & Testing (Week 6)
- Runtime-level graceful shutdown coordination
- Signal handler integration
- End-to-end testing and load testing
- Performance validation and optimization

## Success Criteria

- ✅ **Zero Data Loss**: No incomplete write operations during shutdown
- ✅ **Work Completion**: All active operations complete before process exit
- ✅ **Performance**: <1% overhead during normal operations
- ✅ **Timeout Compliance**: Configurable timeout behavior
- ✅ **Observability**: Structured logging and metrics export
- ✅ **Test Coverage**: >95% test coverage for shutdown scenarios
- ✅ **Concurrency Safety**: Thread-safe operation under high load
- ✅ **Resource Cleanup**: No resource leaks after shutdown

This metrics-based approach provides robust work tracking with minimal performance impact by leveraging k1s's existing, proven infrastructure.