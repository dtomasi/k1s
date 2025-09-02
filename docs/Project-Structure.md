# K1S Project Structure

**Related Documentation:**
- [Architecture](Architecture.md) - Complete system architecture with visual diagrams
- [Implementation Plan](Implementation-Plan.md) - Development roadmap and phases

## Go Workspace Layout

K1s uses a Go workspace with multiple modules for clean separation and optional dependencies.

```
k1s/
├── go.work                    # Workspace configuration
├── go.work.sum               # Workspace dependencies
│
├── core/                     # Core k1s functionality
│   ├── go.mod               # Main module
│   └── pkg/
│       ├── runtime/         # Type system & orchestration
│       ├── client/          # Client interface implementation
│       ├── storage/         # Multi-tenant storage interface & factory
│       ├── codec/           # Serialization
│       ├── registry/        # Resource management
│       ├── validation/      # Validation engine
│       ├── defaulting/      # Defaulting engine
│       ├── events/          # Kubernetes event system
│       ├── informers/       # Informer factory for CLI-optimized queries
│       ├── controller/      # Controller-runtime package
│       └── cli-runtime/     # CLI builders, factories, formatters
│
├── storage/                  # Storage backend modules
│   ├── memory/
│   │   ├── go.mod           # Memory storage module
│   │   └── memory.go
│   └── pebble/              # Pebble module (primary persistent backend)
│       ├── go.mod
│       └── pebble.go
│
├── tools/                    # Development tools
│   ├── go.mod               # Tools module
│   └── cmd/
│       └── cli-gen/         # Code generation tool
│
└── examples/                 # Examples and demos
    ├── go.mod               # Examples module
    ├── api/v1alpha1/        # Example CRDs
    └── cmd/k1s-demo/        # Demo CLI application
```

## Module Dependencies

```mermaid
graph TD
    Core[core module]
    Memory[storage/memory module]
    Pebble[storage/pebble module]
    Tools[tools module]
    Examples[examples module]
    
    %% Dependencies
    Memory --> Core
    Pebble --> Core
    Tools --> Core
    Examples --> Core
    Examples --> Memory
    Examples --> Pebble
    
    %% Styling
    classDef coreModule fill:#e8f5e8
    classDef storageModule fill:#fce4ec
    classDef toolModule fill:#fff3e0
    classDef exampleModule fill:#e1f5fe
    
    class Core coreModule
    class Memory,Bolt,Badger,Pebble storageModule
    class Tools toolModule
    class Examples exampleModule
```

## Module Descriptions

### Core Module (`core/`)

**Purpose:** Main k1s runtime and API interfaces

**Key Packages:**
- `runtime/` - Type system, scheme, orchestration
- `client/` - Kubernetes-compatible client interface
- `storage/` - Storage abstraction and factory
- `events/` - Kubernetes event system
- `controller/` - Controller-runtime compatibility
- `cli-runtime/` - CLI builders and formatters

**Dependencies:** Standard library + k8s.io packages

### Storage Modules (`storage/*/`)

**Purpose:** Pluggable storage backend implementations

**Modules:**
- `storage/memory/` - In-memory storage for development/testing (>10,000 ops/sec)
- `storage/pebble/` - LSM-tree persistent storage with high performance (>3,000 ops/sec)

**Dependencies:** Core module + respective database libraries

### Tools Module (`tools/`)

**Purpose:** Development and code generation tools

**Components:**
- `cli-gen/` - kubebuilder-compatible code generator
- Schema generation for IDE integration
- Validation strategy generation

**Dependencies:** Core module + code generation libraries

### Examples Module (`examples/`)

**Purpose:** Example implementations and demos

**Components:**
- `api/v1alpha1/` - Example CRD definitions (Items, Categories)
- `cmd/k1s-demo/` - Complete CLI application demo
- Integration testing and documentation

**Dependencies:** Core + storage modules

## Workspace Configuration

### go.work File

```go
go 1.25.0

use (
    ./core
    ./storage/memory
    ./storage/pebble
    ./tools
    ./examples
)
```

### Benefits of Multi-Module Structure

1. **Optional Dependencies:** Applications can import only needed storage backends
2. **Clean Separation:** Core functionality independent from storage implementations
3. **Independent Versioning:** Each module can evolve at its own pace
4. **Reduced Binary Size:** Only used storage backends compiled into final binary
5. **Testing Isolation:** Each module tested independently
6. **Development Flexibility:** Teams can work on modules independently

## Import Patterns

### Application Using Core + Memory Storage

```go
import (
    "github.com/dtomasi/k1s/core/pkg/runtime"
    "github.com/dtomasi/k1s/core/pkg/client"
    "github.com/dtomasi/k1s/storage/memory"
)
```

### CLI Application Using CLI-Runtime

```go
import (
    "github.com/dtomasi/k1s/core/pkg/cli-runtime"
    "github.com/dtomasi/k1s/core/pkg/runtime"
    "github.com/dtomasi/k1s/storage/bolt"
)
```

### Controller Application

```go
import (
    "github.com/dtomasi/k1s/core/pkg/controller"
    "github.com/dtomasi/k1s/core/pkg/runtime"
    "github.com/dtomasi/k1s/storage/pebble"
)
```

This modular structure enables flexible usage while maintaining clean separation of concerns and optional dependencies.