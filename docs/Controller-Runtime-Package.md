# K1S Controller-Runtime Package Specification

**Related Documentation:**
- [CLI-Runtime Package](CLI-Runtime-Package.md) - CLI instrumentation package  
- [Architecture](Architecture.md) - Overall k1s system architecture

## Overview

The Controller-Runtime package (`core/pkg/controller/`) provides a kubernetes controller-runtime compatible interface optimized for CLI environments. It takes an initialized k1s runtime and creates a manager that maintains compatibility with standard kubernetes controller patterns.

## Core Responsibilities

### 1. **Kubernetes-Compatible Manager**
- Provide `controller.NewManager()` function compatible with controller-runtime
- Accept k1s runtime as dependency injection
- Maintain familiar `mgr.GetClient()`, `mgr.GetScheme()` APIs

### 2. **Controller Registration**
- Support standard `SetupWithManager()` patterns
- Builder API identical to controller-runtime
- Reconciler interface compatibility

### 3. **CLI-Optimized Execution**
- Triggered execution instead of continuous loops
- Context-based start/stop lifecycle
- Event-driven reconciliation

## Package Structure

```
core/pkg/controller/
├── manager.go          # Main manager implementation
├── builder.go          # Controller builder API
├── reconciler.go       # Reconciler interface
├── interfaces.go       # Manager and controller interfaces
└── options.go          # Configuration options
```

## 1. Manager Creation (Kubernetes-Compatible)

### Manager Constructor

```go
// core/pkg/controller/manager.go
package controller

import (
    "context"
    "time"
    
    "github.com/dtomasi/k1s/core/pkg/runtime"
    "github.com/dtomasi/k1s/core/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/runtime/scheme"
)

// Manager provides the controller runtime manager interface
// Compatible with sigs.k8s.io/controller-runtime/pkg/manager.Manager
type Manager interface {
    // Client returns the client for this manager
    GetClient() client.Client
    
    // Scheme returns the scheme for this manager
    GetScheme() *scheme.Scheme
    
    // Start starts all controllers and blocks until context is cancelled
    Start(ctx context.Context) error
    
    // Add registers a controller with the manager
    Add(controller Controller) error
    
    // GetEventRecorderFor returns an event recorder for the given name
    GetEventRecorderFor(name string) record.EventRecorder
    
    // GetRESTMapper returns the REST mapper
    GetRESTMapper() meta.RESTMapper
}

// NewManager creates a new controller manager - kubernetes-style API
func NewManager(runtime *runtime.Runtime, options Options) (Manager, error) {
    mgr := &manager{
        runtime:     runtime,
        client:      runtime.GetClient(),
        scheme:      runtime.GetScheme(),
        options:     options,
        controllers: make([]Controller, 0),
        eventRecorder: runtime.GetEventRecorder(),
    }
    
    return mgr, nil
}

// Options configures the manager
type Options struct {
    // MetricsBindAddress is the address to bind metrics endpoint
    MetricsBindAddress string
    
    // HealthProbeBindAddress is the address to bind health probe endpoint  
    HealthProbeBindAddress string
    
    // LeaderElection enables leader election (usually disabled for CLI)
    LeaderElection bool
    
    // LeaderElectionID is the name of the resource used for leader election
    LeaderElectionID string
    
    // Namespace restricts manager to single namespace
    Namespace string
}

type manager struct {
    runtime       *runtime.Runtime
    client        client.Client
    scheme        *scheme.Scheme
    options       Options
    controllers   []Controller
    eventRecorder record.EventRecorder
    
    started bool
    stopped bool
}

func (m *manager) GetClient() client.Client {
    return m.client
}

func (m *manager) GetScheme() *scheme.Scheme {
    return m.scheme
}

func (m *manager) Start(ctx context.Context) error {
    if m.started {
        return fmt.Errorf("manager already started")
    }
    m.started = true
    
    // Start all registered controllers
    for _, controller := range m.controllers {
        if err := controller.Start(ctx); err != nil {
            return fmt.Errorf("failed to start controller: %w", err)
        }
    }
    
    // Wait for context cancellation
    <-ctx.Done()
    
    // Stop all controllers
    for _, controller := range m.controllers {
        controller.Stop()
    }
    
    m.stopped = true
    return nil
}

func (m *manager) Add(controller Controller) error {
    if m.started {
        return fmt.Errorf("cannot add controller to started manager")
    }
    
    m.controllers = append(m.controllers, controller)
    return nil
}

func (m *manager) GetEventRecorderFor(name string) record.EventRecorder {
    return m.eventRecorder
}
```

## 2. Controller Builder API (Kubernetes-Compatible)

### Builder Interface

```go
// core/pkg/controller/builder.go
package controller

import (
    "sigs.k8s.io/controller-runtime/pkg/controller"
    "sigs.k8s.io/controller-runtime/pkg/reconcile"
    "sigs.k8s.io/controller-runtime/pkg/builder"
)

// Builder is compatible with controller-runtime builder
type Builder interface {
    // For sets the resource type to reconcile
    For(client.Object, ...builder.ForOption) Builder
    
    // Owns sets child resources owned by the primary resource
    Owns(client.Object, ...builder.OwnsOption) Builder
    
    // Watches sets additional resources to watch
    Watches(client.Object, handler.EventHandler, ...builder.WatchesOption) Builder
    
    // WithOptions configures the controller
    WithOptions(controller.Options) Builder
    
    // Complete creates the controller
    Complete(reconcile.Reconciler) error
}

// NewControllerManagedBy creates a new builder - kubernetes-style API
func NewControllerManagedBy(mgr Manager) Builder {
    return &controllerBuilder{
        manager: mgr,
        options: controller.Options{},
    }
}

type controllerBuilder struct {
    manager         Manager
    forType         client.Object
    ownedTypes      []client.Object
    watchedTypes    []watchedType
    options         controller.Options
}

func (b *controllerBuilder) For(obj client.Object, opts ...builder.ForOption) Builder {
    b.forType = obj
    return b
}

func (b *controllerBuilder) Owns(obj client.Object, opts ...builder.OwnsOption) Builder {
    b.ownedTypes = append(b.ownedTypes, obj)
    return b
}

func (b *controllerBuilder) Complete(reconciler reconcile.Reconciler) error {
    ctrl := &controllerImpl{
        manager:     b.manager,
        reconciler:  reconciler,
        forType:     b.forType,
        ownedTypes:  b.ownedTypes,
        options:     b.options,
    }
    
    return b.manager.Add(ctrl)
}
```

## 3. Reconciler Interface (Kubernetes-Compatible)

### Reconciler Definition

```go
// core/pkg/controller/reconciler.go
package controller

import (
    "context"
    "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler interface is identical to controller-runtime
// This allows existing controllers to work unchanged
type Reconciler interface {
    Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error)
}

// Request represents a reconciliation request
type Request struct {
    NamespacedName types.NamespacedName
}

// Result contains the result of a reconcile run
type Result struct {
    // Requeue tells the controller to requeue the reconcile key
    Requeue bool
    
    // RequeueAfter if greater than 0, tells the controller to requeue after the specified duration
    RequeueAfter time.Duration
}
```

## 4. Usage Examples

### Basic Controller Setup (Identical to controller-runtime)

```go
import (
    "context"
    "github.com/dtomasi/k1s/core/pkg/controller"
    "github.com/dtomasi/k1s/storage/memory"
    ctrl "sigs.k8s.io/controller-runtime"
)

// ItemReconciler - identical to kubernetes controller
type ItemReconciler struct {
    client.Client
    Scheme *runtime.Scheme
}

func (r *ItemReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Standard reconciler logic - exactly like kubernetes
    var item v1alpha1.Item
    if err := r.Get(ctx, req.NamespacedName, &item); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }
    
    // Business logic here...
    return ctrl.Result{}, nil
}

func (r *ItemReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&v1alpha1.Item{}).
        Complete(r)
}

func main() {
    // 1. Create k1s runtime
    storage := memory.NewStorage()
    runtime, err := k1s.NewRuntime(storage, k1s.WithTenant("controller-app"))
    if err != nil {
        log.Fatal(err)
    }
    
    // 2. Create controller manager (kubernetes-style)
    mgr, err := controller.NewManager(runtime, controller.Options{
        MetricsBindAddress:     ":8080",
        HealthProbeBindAddress: ":8081",
        LeaderElection:         false, // CLI-optimized
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // 3. Register controller (identical to controller-runtime)
    if err = (&ItemReconciler{
        Client: mgr.GetClient(),
        Scheme: mgr.GetScheme(),
    }).SetupWithManager(mgr); err != nil {
        log.Fatal(err)
    }
    
    // 4. Start manager (identical to controller-runtime)
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := mgr.Start(ctx); err != nil {
        log.Fatal(err)
    }
}
```

### Multiple Controllers

```go
func main() {
    // Runtime setup
    storage, _ := pebble.NewStorage("./data/controllers.db")
    runtime, _ := k1s.NewRuntime(storage, k1s.WithTenant("multi-controller"))
    
    // Manager setup
    mgr, _ := controller.NewManager(runtime, controller.Options{
        LeaderElection: false,
    })
    
    // Register multiple controllers (standard pattern)
    if err := (&ItemReconciler{
        Client: mgr.GetClient(),
        Scheme: mgr.GetScheme(),
    }).SetupWithManager(mgr); err != nil {
        log.Fatal(err)
    }
    
    if err := (&CategoryReconciler{
        Client: mgr.GetClient(),
        Scheme: mgr.GetScheme(),
    }).SetupWithManager(mgr); err != nil {
        log.Fatal(err)
    }
    
    // Start all controllers
    ctx := context.Background()
    mgr.Start(ctx)
}
```

## Benefits

### 1. **100% Controller-Runtime Compatible**
- Existing controller code works unchanged
- Standard `SetupWithManager()` patterns
- Identical reconciler interfaces

### 2. **CLI-Optimized**
- Context-based lifecycle management
- No continuous background loops
- Event-driven reconciliation

### 3. **Direct k1s Integration**
- Takes k1s runtime as input
- Uses k1s client and scheme directly
- Integrates with k1s event system

### 4. **Standard Kubernetes Patterns**
- Familiar manager.Manager interface
- Controller builder API
- Event recording integration

## Implementation Notes

The Controller-Runtime package focuses on **compatibility** with kubernetes controller-runtime while optimizing for CLI environments:

- **Manager Creation**: Takes k1s runtime instead of kubeconfig
- **Execution Model**: Context-driven instead of continuous loops  
- **Resource Access**: Direct storage access instead of API server
- **Event Integration**: Uses k1s event system for recording

This allows kubernetes controller code to run unchanged in k1s CLI environments while maintaining the familiar development patterns.