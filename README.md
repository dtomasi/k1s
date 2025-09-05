# k1s

**Status**: 🚧 Under active development - Not functional at all ATM - APIs may change

A Kubernetes-native runtime for building CLI tools with embedded storage and controller capabilities.

## What is k1s?

k1s provides a lightweight, CLI-optimized implementation of Kubernetes patterns for building command-line tools. It implements standard Kubernetes interfaces while adapting them for short-lived CLI processes, enabling developers to build kubectl-style tools with familiar patterns and APIs.

### Key Characteristics

- **CLI-First Design**: Optimized for short-lived processes, not long-running servers
- **No API Server**: Direct storage access without intermediate layers  
- **Process-Safe**: Multiple CLI processes can safely access the same storage
- **Fast Startup**: Sub-100ms initialization for responsive CLI tools
- **Kubernetes-Native**: Uses standard Kubernetes interfaces and patterns

## Core Features

### 🎯 Kubernetes Compatibility
- Implements `client.Client` interface from controller-runtime
- Compatible with `storage.Interface` from Kubernetes apiserver
- Supports standard Kubernetes resources and custom resources
- Works with existing kubebuilder markers and patterns

### 🏗️ Built-in Core Resources
- **Namespace** - Multi-tenancy and resource organization
- **ConfigMap** - Non-sensitive configuration data
- **Secret** - Sensitive data with encoding support
- **ServiceAccount** - Identity for automation
- **Event** - Audit trail and observability

### 💾 High-Performance Storage
- **Memory Backend** - For testing and development (>10,000 ops/sec)
- **Pebble Backend** - LSM-tree based persistent storage (>3,000 ops/sec)
- Pluggable architecture for custom storage backends
- Multi-tenant support with key prefixing

### 🔒 Security & Multi-tenancy
- Lightweight RBAC using standard Kubernetes RBAC resources
- ServiceAccount-based authentication
- Namespace isolation
- Resource-level access control

### 🛠️ Developer Experience
- `cli-gen` tool for code generation from kubebuilder markers
- CLI runtime package for kubectl-style commands
- Controller runtime adapted for CLI environments
- Comprehensive validation and defaulting framework

### ⚡ Performance Optimizations
- Lazy loading of components
- On-demand informers and controllers
- Efficient serialization with JSON/YAML codecs
- Minimal memory footprint

## Architecture Highlights

k1s adapts Kubernetes patterns specifically for CLI tool requirements:

- **Direct Storage Access**: No API server overhead, direct read/write to embedded storage
- **Process Coordination**: File locking and safe concurrent access from multiple CLI processes
- **Triggered Controllers**: Controllers run on-demand, not in continuous loops
- **Fast Initialization**: Components load only when needed for quick startup

## Use Cases

k1s is ideal for building:

- **Development Tools**: Local Kubernetes-like environments for testing
- **CI/CD Tools**: Pipeline orchestration with familiar Kubernetes patterns
- **Administrative CLIs**: Infrastructure management tools
- **Operators**: CLI-based operators that don't require cluster deployment
- **Migration Tools**: Data transformation and migration utilities

## Project Structure

k1s uses a modular Go workspace design:

- `core/` - Core runtime, interfaces, and built-in resources
- `storage/memory/` - In-memory storage backend
- `storage/pebble/` - Persistent storage with Pebble
- `tools/` - Development tools including cli-gen
- `examples/` - Example applications and demos
- `docs/` - Architecture and design documentation

## Documentation

- [Architecture](docs/Architecture.md) - System design and patterns
- [Development Guide](DEVELOPMENT.md) - Contributing and coding standards
- [Implementation Plan](docs/Implementation-Plan.md) - Work packages and roadmap

## License

[Apache License 2.0](LICENSE)

## Acknowledgments

k1s builds upon the excellent work of the Kubernetes community, particularly:
- [Kubernetes](https://kubernetes.io/)
- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder)

---

For questions, discussions, and contributions, please use [GitHub Issues](https://github.com/dtomasi/k1s/issues).
