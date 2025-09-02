# K1S Development Guide

This guide contains all development standards, coding guidelines, and contribution requirements for the k1s project.

## Table of Contents

- [Development Setup](#development-setup)
- [Coding Standards](#coding-standards)
- [Testing Requirements](#testing-requirements)
- [Project Structure](#project-structure)
- [Build and Development Commands](#build-and-development-commands)
- [Performance Guidelines](#performance-guidelines)
- [Documentation Standards](#documentation-standards)
- [Contribution Workflow](#contribution-workflow)

## Development Setup

### Prerequisites

- Go 1.23.0 or later
- Hermit for tool management (optional but recommended)
- Just for task automation
- golangci-lint for code quality
- Ginkgo v2 for testing

### Initial Setup

```bash
# Clone the repository
git clone git@github.com:dtomasi/k1s.git
cd k1s

# Activate hermit (if available)
. ./bin/activate-hermit

# Install dependencies
go mod download

# Run tests
just work-test

# Run linting
just work-lint
```

## Coding Standards

### Go Standards

1. **Standard Go Conventions**
   - Follow standard Go conventions (gofmt, go vet)
   - Use goimports for import organization
   - Prefer explicit error handling over panic
   - Write clear and concise comments for exported functions and types

2. **Code Quality**
   - **CRITICAL**: Use golangci-lint for comprehensive linting - ALL code MUST pass golangci-lint rules
   - Configuration in `.golangci.yml` with best practices enabled (50+ linters)
   - Run `just lint` before any commit to ensure compliance
   - No exceptions - fix ALL linting issues before proceeding

3. **Design Patterns**
   - Use functional options pattern for complex constructors
   - Prefer composition over inheritance for struct design
   - Use interfaces to define behavior, not data
   - Keep packages self-contained and focused on a single responsibility

4. **Error Handling**
   - Use `errors.Is()` and `errors.As()` for error type checking
   - Use Error variables like `var ErrNotFound = errors.New("not found")` for common error cases
   - Always wrap errors with context using `fmt.Errorf("failed to do X: %w", err)`
   - Return errors early to reduce nesting

5. **Best Practices**
   - Avoid using strings for constants - use typed constants or enums with enumer tool
   - Do not abuse context.Context - use it only for request-scoped values like deadlines, cancellation signals
   - Keep functions small and focused on a single task
   - Avoid global state and mutable package-level variables
   - Avoid direct dependencies between packages; use interfaces

6. **Code Generation**
   - All generated code should be placed in files with a `zz_generated.` prefix
   - Use uber mockgen for generating mocks:
     ```go
     //go:generate go run go.uber.org/mock/mockgen@latest -source=<source-file>.go -destination=zz_generated.<file-name>.go -package="<package-name>"
     ```
   - Never edit generated files manually - they will be overwritten

7. **Constructor Patterns**
   - Required configuration options should be enforced via constructor parameters
   - Optional parameters should use functional options pattern:
     ```go
     type Option func(*Config)
     
     func NewThing(required string, opts ...Option) *Thing {
         // implementation
     }
     ```

## Testing Requirements

### Test Framework

- **Ginkgo v2 for BDD-style testing** - All tests must use Ginkgo
- Test files use `_test.go` suffix
- Integration tests in `_suite_test.go` files
- Mock external dependencies in tests

### Coverage Requirements

- **Minimum 70% test coverage** for all components (automatically enforced by CI)
- Coverage is measured using `go test -covermode=atomic`
- CI automatically fails if any module falls below 70% coverage
- Run coverage locally with: `go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out`

### Test Organization

```go
var _ = Describe("ComponentName", func() {
    Context("when condition", func() {
        It("should behave correctly", func() {
            // test implementation
            Expect(result).To(Equal(expected))
        })
    })
})
```

### Mock Generation

Use uber mockgen to generate mocks for interfaces instead of generating them manually:

```bash
# Generate mock for an interface
go run go.uber.org/mock/mockgen@latest -source=interface.go -destination=zz_generated.mock.go -package=mocks
```

### Test Categories

1. **Unit Tests**: Test individual functions and methods
2. **Integration Tests**: Test component interactions
3. **Performance Tests**: Benchmark critical paths
4. **End-to-End Tests**: Test complete workflows

## Project Structure

The complete project structure and module organization is documented in:

**→ [docs/Project-Structure.md](docs/Project-Structure.md)**

### Module Guidelines

- Each storage backend is a separate module
- Core functionality in the `core` module  
- Examples in the `examples` module
- Tools in the `tools` module
- All modules listed in `go.work`

## Build and Development Commands

### Using Just (Recommended)

```bash
# Run all tests
just work-test

# Run linting
just work-lint

# Fix auto-fixable lint issues
just lint-fix

# Build all modules
just work-build

# Run tests with coverage
just work-test-coverage

# Clean build artifacts
just clean
```

### Direct Go Commands

```bash
# Test specific module
cd core && go test ./...

# Run with verbose output
go test -v ./...

# Run specific test
go test -run TestName ./pkg/...

# Build binary
go build -o bin/k1s ./cmd/k1s

# Generate code
go generate ./...
```

## Performance Guidelines

### Performance Targets

1. **Storage Backends** (automatically benchmarked by CI)
   - Memory Storage: >10,000 operations/second
   - Pebble Storage: >3,000 operations/second

2. **CLI Operations**
   - Startup time: <100ms for basic operations
   - Resource listing: <200ms for 1000 resources
   - Single resource operations: <50ms

3. **Memory Usage**
   - Base memory footprint: <20MB
   - Memory per resource: <1KB
   - No memory leaks in long-running operations

### Performance Testing

```bash
# Run benchmarks
go test -bench=. ./...

# Run benchmarks with memory profiling
go test -bench=. -benchmem ./...

# Generate CPU profile
go test -cpuprofile=cpu.prof -bench=. ./...

# Analyze profile
go tool pprof cpu.prof
```

### Optimization Guidelines

1. **Avoid Premature Optimization**: Profile first, optimize second
2. **Use Efficient Data Structures**: Choose the right data structure for the job
3. **Minimize Allocations**: Reuse objects where possible
4. **Batch Operations**: Group related operations to reduce overhead
5. **Cache Strategically**: Cache expensive computations and I/O operations

## Documentation Standards

### Code Documentation

1. **GoDoc Comments**
   - All exported functions, types, and packages must have GoDoc comments
   - Comments should start with the name of the item being documented
   - Example:
     ```go
     // NewClient creates a new k1s client with the given configuration.
     // It returns an error if the configuration is invalid.
     func NewClient(config *Config) (*Client, error) {
         // implementation
     }
     ```

2. **Package Documentation**
   - Each package must have a `doc.go` file with package documentation
   - Describe the package purpose and main types
   - Include usage examples where appropriate

3. **README Files**
   - Each module should have a README.md
   - Include: Purpose, Installation, Usage, Examples
   - Keep documentation up-to-date with code changes

### Architecture Documentation

1. **Design Documents**: Major features need design docs in `docs/`
2. **Mermaid Diagrams**: Use for complex workflows and architecture
3. **API Documentation**: Document all public APIs thoroughly
4. **Migration Guides**: Document breaking changes and migration paths

## Contribution Workflow

### Branch Strategy

```bash
# Create feature branch
git checkout -b feature/description

# Create bugfix branch  
git checkout -b fix/description

# Create documentation branch
git checkout -b docs/description
```

### Commit Guidelines

1. **Commit Message Format**
   ```
   type(scope): brief description
   
   Longer explanation if needed.
   
   Fixes #issue-number
   ```

2. **Types**
   - `feat`: New feature
   - `fix`: Bug fix
   - `docs`: Documentation changes
   - `test`: Test additions or changes
   - `refactor`: Code refactoring
   - `perf`: Performance improvements
   - `chore`: Maintenance tasks

3. **Scope Examples**
   - `runtime`, `storage`, `client`, `cli`, `auth`, `events`

### Pull Request Process

1. **Before Opening PR**
   - Run `just work-test` - all tests must pass
   - Run `just work-lint` - zero linting errors
   - Update documentation if needed
   - Add tests for new functionality

2. **PR Description**
   - Describe what changed and why
   - Reference related issues
   - Include testing instructions
   - List breaking changes (if any)

3. **Review Process**
   - Address all review comments
   - Keep PR focused and small
   - Squash commits before merge
   - Ensure CI passes

### Quality Gates

**No PR can be merged unless:**
- ✅ All tests pass (automatically checked by CI across Linux, macOS, Windows)
- ✅ Zero linting errors (50+ golangci-lint rules enforced)
- ✅ Code coverage meets requirements (70% minimum for all modules)
- ✅ All builds succeed (multi-platform verification)
- ✅ Security scans pass (gosec vulnerability detection)
- ✅ Documentation is updated
- ✅ PR has been reviewed and approved
- ✅ Complete CI/CD pipeline is green

**Local verification:**
```bash
# Run all quality checks locally before pushing
go work sync
go test -race ./...                    # Run tests with race detection
golangci-lint run --config .golangci.yml  # Run comprehensive linting
go test -coverprofile=coverage.out ./...  # Check coverage
go build ./...                             # Verify build
```

## Work Package Implementation

When implementing a work package from the [Implementation Plan](docs/Implementation-Plan.md):

1. **Review the Work Package** specification thoroughly
2. **Create the Interface First** (if applicable)
3. **Write Tests** before implementation (TDD)
4. **Implement the Functionality**
5. **Ensure Quality Gates** are met
6. **Update Documentation**
7. **Submit PR** with reference to work package

### Example Workflow

```bash
# Start work on WP-001
git checkout -b feat/wp-001-runtime-interfaces

# Create interface files
touch core/pkg/runtime/interfaces.go

# Write interface tests first
touch core/pkg/runtime/interfaces_test.go

# Implement interfaces
# ... coding ...

# Test your implementation
just work-test

# Check linting
just work-lint

# Commit with proper message
git commit -m "feat(runtime): implement core runtime interfaces WP-001

- Add runtime.Object interface
- Add runtime.Scheme interface
- Add GVK/GVR utilities
- Add comprehensive tests

Implements #WP-001"

# Push and create PR
git push origin feat/wp-001-runtime-interfaces
```

## Debugging and Troubleshooting

### Common Issues

1. **Import Cycles**
   - Use interfaces to break cycles
   - Move shared types to separate packages
   - Consider package structure refactoring

2. **Test Failures**
   - Run tests with `-v` for verbose output
   - Use focused specs in Ginkgo: `FIt()` or `FContext()`
   - Check test isolation - tests should not depend on order

3. **Linting Errors**
   - Run `just lint-fix` for auto-fixable issues
   - Check `.golangci.yml` for rule configuration
   - Some linters can be disabled per-line with comments (use sparingly)

4. **Performance Issues**
   - Profile with pprof to identify bottlenecks
   - Check for unnecessary allocations
   - Review algorithm complexity
   - Consider caching strategies

### Useful Commands

```bash
# Debug test failures
go test -v -run TestName ./path/to/package

# Check for race conditions
go test -race ./...

# Find unused dependencies
go mod tidy -v

# Update dependencies
go get -u ./...

# Check for vulnerabilities
go list -m all | nancy sleuth
```

## Contact and Support

- **GitHub Issues**: Report bugs and request features
- **Documentation**: Check `docs/` folder for detailed guides
- **Examples**: See `examples/` for usage patterns

---

*This document is the authoritative source for k1s development standards. All contributors must follow these guidelines.*