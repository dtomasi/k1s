#!/usr/bin/env just --justfile

# Displays information about the Golang environment
info:
    #!/usr/bin/env bash
    echo "Golang PATH: $(go env GOPATH)"
    echo "Golang GOBIN: $(go env GOBIN)"
    echo "Golang Version: $(go version)"
    echo "Working Directory: $(pwd)"

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

# Updates dependencies across all modules
mod-tidy:
    @just _for-each-module "go mod tidy"

# Downloads dependencies across all modules
mod-download:
    @just _for-each-module "go mod download"

# Runs golangci-lint across all modules
lint:
    @just _for-each-module "golangci-lint run --timeout=5m ./..."

# Runs golangci-lint with fixes across all modules
lint-fix:
    @just _for-each-module "golangci-lint run --timeout=5m --fix ./..."


# Runs golangci-lint with all issues shown across all modules
lint-issues:
    @just _for-each-module "golangci-lint run --timeout=5m --issues-exit-code=0 --out-format=tab ./..."


# Helper function to get all module paths from go.work
_get-modules:
    @grep -A 20 "use (" go.work | grep -E "^\s*\./.*" | sed 's/[[:space:]]*\.\///' | sed 's/[[:space:]]*$//'

# Execute a command for each module in the workspace
_for-each-module cmd:
    #!/usr/bin/env bash
    set -euo pipefail
    modules=$(grep -A 20 "use (" go.work | grep -E "^\s*\./.*" | sed 's/[[:space:]]*\.\///' | sed 's/[[:space:]]*$//')
    for module in $modules; do
        echo "==> Running '{{cmd}}' in module: $module"
        cd "$module" && {{cmd}} && cd - > /dev/null || exit 1
    done

# Syncs all modules in workspace
sync:
    go work sync

# Builds all binaries across modules
build:
    @echo "Building all binaries..."
    cd tools && go build -o ../bin/cli-gen ./cmd/cli-gen
    cd examples && go build -o ../bin/k1s-demo ./cmd/k1s-demo

# Cleans all modules and caches
clean:
    @just _for-each-module "go clean ./..."
    @echo "Cleaning global caches..."
    go clean -cache -testcache

# Install Go tools that are not available via hermit
install-go-tools:
    go install github.com/onsi/ginkgo/v2/ginkgo@latest
    go install go.uber.org/mock/mockgen@latest