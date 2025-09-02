#!/usr/bin/env just --justfile

# Project constants
PROJECT_ROOT := justfile_directory()
GOLANGCI_CONFIG := PROJECT_ROOT / ".golangci.yml"
LINT_TIMEOUT := "10m"
COVERAGE_THRESHOLD := "70.0"

# Displays information about the Golang environment
info:
    #!/usr/bin/env bash
    echo "Project Root: {{ PROJECT_ROOT }}"
    echo "Golang PATH: $(go env GOPATH)"
    echo "Golang GOBIN: $(go env GOBIN)"
    echo "Golang Version: $(go version)"
    echo "Working Directory: $(pwd)"
    echo "Modules:"
    just _get-modules | sed 's/^/  - /'

# === Development Commands ===

# Formats golang code across all modules
format:
    @just _for-each-module "go fmt ./..."

# Runs go vet for static analysis across all modules
vet:
    @just _for-each-module "go vet ./..."

# Runs all tests across all modules
test:
    @just _for-each-module "go test ./..."

# Runs tests with verbose output across all modules
test-verbose:
    @just _for-each-module "go test -v ./..."

# Runs tests with race detector across all modules
test-race:
    @just _for-each-module "go test -race ./..."

# Runs tests and generates coverage report across all modules
test-cover:
    @just _for-each-module "go test -v -race -coverprofile=coverage.txt -covermode=atomic ./..."

# Checks coverage meets 70% requirement (same as CI)
coverage-check:
    @just _check-coverage {{ COVERAGE_THRESHOLD }}

# === Linting Commands ===

# Runs golangci-lint across all modules
lint:
    @just _run-golangci-lint

# Runs golangci-lint with fixes across all modules
lint-fix:
    @just _run-golangci-lint --fix

# Runs golangci-lint with all issues shown across all modules
lint-issues:
    @just _run-golangci-lint --issues-exit-code=0 --out-format=tab

# === Module Management ===

# Syncs all modules in workspace
sync:
    @cd "{{ PROJECT_ROOT }}" && go work sync

# Updates dependencies across all modules
mod-tidy:
    @just _for-each-module "go mod tidy"

# Downloads dependencies across all modules
mod-download:
    @just _for-each-module "go mod download"

# === Build Commands ===

# Builds all binaries across modules
build:
    @just _build-binaries

# Builds a specific binary
build-cli-gen:
    @cd "{{ PROJECT_ROOT }}/tools" && go build -o ../bin/cli-gen ./cmd/cli-gen

# Builds the demo CLI
build-demo:
    @cd "{{ PROJECT_ROOT }}/examples" && go build -o ../bin/k1s-demo ./cmd/k1s-demo

# === Cleanup Commands ===

# Cleans all modules and caches
clean:
    @just _for-each-module "go clean ./..."
    @cd "{{ PROJECT_ROOT }}" && echo "Cleaning global caches..." && go clean -cache -testcache

# === Tool Installation ===

# Install Go tools that are not available via hermit
install-go-tools:
    @cd "{{ PROJECT_ROOT }}" && go install github.com/onsi/ginkgo/v2/ginkgo@latest
    @cd "{{ PROJECT_ROOT }}" && go install go.uber.org/mock/mockgen@latest

# === CI/CD Simulation ===

# Run all CI checks locally (format, lint, test, coverage, build)
ci-local:
    @echo "🔍 Running local CI simulation"
    @echo "1. Format check..."
    @just format
    @echo "2. Linting..."
    @just lint
    @echo "3. Testing..."
    @just test
    @echo "4. Coverage check..."
    @just coverage-check
    @echo "5. Building..."
    @just build
    @echo "✅ Local CI simulation complete!"

# === Helper Functions ===

# Get all module paths from go.work
_get-modules:
    @cd "{{ PROJECT_ROOT }}" && grep -A 20 "use (" go.work | grep -E "^\s*\./.*" | sed 's/[[:space:]]*\.\///' | sed 's/[[:space:]]*$//'

# Execute a command for each module in the workspace
_for-each-module cmd:
    #!/usr/bin/env bash
    set -euo pipefail
    cd "{{ PROJECT_ROOT }}"
    
    modules=$(just _get-modules)
    for module in $modules; do
        echo "==> Running '{{cmd}}' in module: $module"
        cd "{{ PROJECT_ROOT }}/$module" && {{cmd}} || exit 1
    done

# Run golangci-lint with optional arguments
_run-golangci-lint +args="":
    #!/usr/bin/env bash
    set -euo pipefail
    cd "{{ PROJECT_ROOT }}"
    
    modules=$(just _get-modules)
    for module in $modules; do
        echo "==> Running golangci-lint in module: $module"
        cd "{{ PROJECT_ROOT }}/$module"
        golangci-lint run --config="{{ GOLANGCI_CONFIG }}" --timeout={{ LINT_TIMEOUT }} {{ args }} ./...
    done

# Check coverage meets minimum requirement
_check-coverage min_coverage:
    #!/usr/bin/env bash
    set -euo pipefail
    cd "{{ PROJECT_ROOT }}"
    
    modules=$(just _get-modules)
    failed_modules=()
    
    for module in $modules; do
        echo "==> Checking coverage in module: $module"
        cd "{{ PROJECT_ROOT }}/$module"
        
        # Run tests with coverage
        if ! go test -coverprofile=coverage.out -covermode=atomic ./... 2>/dev/null; then
            echo "No tests in module $module, skipping coverage check"
            continue
        fi
        
        # Check if coverage file exists and has content
        if [[ ! -s coverage.out ]]; then
            echo "No coverage data for module $module"
            continue
        fi
        
        # Calculate coverage percentage
        COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
        echo "Module $module coverage: ${COVERAGE}%"
        
        # Check if coverage meets requirement
        if (( $(echo "${COVERAGE} < {{ min_coverage }}" | bc -l) )); then
            echo "❌ Module $module coverage ${COVERAGE}% is below required {{ min_coverage }}%"
            failed_modules+=("$module")
        else
            echo "✅ Module $module coverage ${COVERAGE}% meets requirement"
        fi
    done
    
    if [ ${#failed_modules[@]} -ne 0 ]; then
        echo ""
        echo "❌ Coverage check failed for modules: ${failed_modules[*]}"
        exit 1
    fi
    
    echo "✅ All modules meet {{ min_coverage }}% coverage requirement"

# Build all binaries
_build-binaries:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Building all binaries..."
    
    # Build cli-gen
    cd "{{ PROJECT_ROOT }}/tools"
    go build -o ../bin/cli-gen ./cmd/cli-gen
    echo "✅ Built cli-gen"
    
    # Build k1s-demo
    cd "{{ PROJECT_ROOT }}/examples"
    go build -o ../bin/k1s-demo ./cmd/k1s-demo
    echo "✅ Built k1s-demo"
    
    echo "✅ All binaries built successfully"