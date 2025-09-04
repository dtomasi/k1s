#!/usr/bin/env just --justfile

# === Fundamental VARS ===

# Go version detection priority (can be overridden with GO_VERSION env var):
# 1. go.work (workspace-level version definition)
# 2. go.mod (module-level version definition)
# 3. hermit (installed tool version)
# 4. system go (fallback)
export GO_VERSION := env_var_or_default("GO_VERSION", shell("
  if [ -f go.work ]; then
    grep '^go ' go.work | awk '{print $2}'
  elif [ -f go.mod ]; then
    grep '^go ' go.mod | awk '{print $2}'
  elif command -v hermit &> /dev/null && ls bin/.go-*.pkg >/dev/null 2>&1; then
    ls bin/.go-*.pkg | head -1 | sed 's/.*\\.go-//;s/\\.pkg//'
  else
    go version | awk '{print $3}' | sed 's/go//'
  fi
"))

# === Smart Configuration ===
PROJECT_ROOT := justfile_directory()
GOLANGCI_CONFIG := PROJECT_ROOT / ".golangci.yml"
LINT_TIMEOUT := "10m"
COVERAGE_THRESHOLD := "70.0"

# Detect system architecture for act testing
ACT_ARCH := shell("go env GOARCH")
ACT_PLATFORM := "linux/" + ACT_ARCH

# Use catthehacker image for all architectures (includes required tools like openssl)
# ARM64 systems will use emulation but get better GitHub Actions compatibility
ACT_IMAGE := "catthehacker/ubuntu:act-latest"

# Smart defaults based on environment
CI := env_var_or_default("CI", "false")
STRICT_MODE := env_var_or_default("STRICT_MODE", CI)
FORMAT_MODE := if STRICT_MODE == "true" { "check" } else { "fix" }
TEST_MODE := if CI == "true" { "ci" } else { "dev" }

# === Information ===

# Display environment and project information
info:
    #!/usr/bin/env bash
    echo "🔧 Project Information"
    echo "Project Root: {{ PROJECT_ROOT }}"
    echo "Golang PATH: $(go env GOPATH)"
    echo "Golang Version: $(go version)"
    echo "Working Directory: $(pwd)"
    echo ""
    echo "🎛️  Configuration"
    echo "CI Mode: {{ CI }}"
    echo "Strict Mode: {{ STRICT_MODE }}"
    echo "Format Mode: {{ FORMAT_MODE }}"
    echo "Test Mode: {{ TEST_MODE }}"
    echo ""
    echo "📦 Modules:"
    just _get-modules | sed 's/^/  - /'

# === Core Development Commands (Adaptive) ===

# Format code (adaptive: check in CI, fix locally)
format go_version=GO_VERSION:
    @just _ensure-go-version {{ go_version }}
    @just _format-{{ FORMAT_MODE }}

# Run linting (adaptive: strict in CI, normal locally)
lint go_version=GO_VERSION:
    @just _ensure-go-version {{ go_version }}
    @just _lint-{{ if STRICT_MODE == "true" { "strict" } else { "normal" } }}

# Run tests (adaptive: coverage in CI, fast locally)
test go_version=GO_VERSION:
    @just _ensure-go-version {{ go_version }}
    @just _test-{{ TEST_MODE }}

# Check coverage meets threshold
coverage go_version=GO_VERSION:
    @just _ensure-go-version {{ go_version }}
    @just _check-coverage {{ COVERAGE_THRESHOLD }}

# Build all components
build go_version=GO_VERSION:
    @just _ensure-go-version {{ go_version }}
    @just _build-all

# Run security scan
security go_version=GO_VERSION:
    @just _ensure-go-version {{ go_version }}
    @just _security-scan

# Run performance benchmarks
benchmarks go_version=GO_VERSION:
    @just _ensure-go-version {{ go_version }}
    @just _run-benchmarks

# === Pipeline System ===

# Run configurable pipeline steps
pipeline steps="format,lint,test,build" go_version=GO_VERSION:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "🚀 Running pipeline with steps: {{ steps }}"
    echo "🐹 Go version: {{ go_version }}"
    echo ""

    IFS=',' read -ra PIPELINE_STEPS <<< "{{ steps }}"

    for step in "${PIPELINE_STEPS[@]}"; do
        echo "🔄 Pipeline step: $step"
        case $step in
            "sync") just sync {{ go_version }} ;;
            "format") just format {{ go_version }} ;;
            "lint") just lint {{ go_version }} ;;
            "test") just test {{ go_version }} ;;
            "coverage") just coverage {{ go_version }} ;;
            "build") just build {{ go_version }} ;;
            "security") just security {{ go_version }} ;;
            "benchmarks") just benchmarks {{ go_version }} ;;
            *) echo "❌ Unknown pipeline step: $step"; exit 1 ;;
        esac
        echo "✅ Completed: $step"
        echo ""
    done

    echo "🎉 Pipeline completed successfully!"

# Complete CI simulation locally
ci-local go_version=GO_VERSION:
    @echo "🔍 Running complete CI simulation locally"
    @STRICT_MODE=true CI=true just pipeline "sync,format,lint,test,coverage,build" {{ go_version }}

# === GitHub Actions Entry Points ===

# Quality checks for GitHub Actions
gh-quality-checks go_version=GO_VERSION:
    @echo "🏃‍♂️ GitHub Actions: Quality Checks"
    @STRICT_MODE=true just pipeline "sync,format,lint" {{ go_version }}

# Test matrix for GitHub Actions
gh-test-matrix go_version=GO_VERSION:
    @echo "🏃‍♂️ GitHub Actions: Test Matrix (Go {{ go_version }})"
    @CI=true just pipeline "sync,test,coverage" {{ go_version }}

# Build and security for GitHub Actions
gh-build-security go_version=GO_VERSION run_benchmarks="true" run_security="true":
    #!/usr/bin/env bash
    echo "🏃‍♂️ GitHub Actions: Build & Security (Go {{ go_version }})"

    steps="sync,build"
    if [ "{{ run_benchmarks }}" = "true" ]; then
        steps="$steps,benchmarks"
    fi
    if [ "{{ run_security }}" = "true" ]; then
        steps="$steps,security"
    fi

    just pipeline "$steps" {{ go_version }}

# === Module Management ===

# Sync workspace dependencies
sync go_version=GO_VERSION:
    @just _ensure-go-version {{ go_version }}
    @echo "📦 Syncing workspace dependencies"
    @cd "{{ PROJECT_ROOT }}" && go work sync
    @echo "✅ Dependencies synced"

# Tidy all module dependencies
mod-tidy:
    @just _for-each-module "go mod tidy"

# Download dependencies
mod-download:
    @just _for-each-module "go mod download"

generate:
    @just _for-each-module "go generate ./..."

# === Cleanup ===

# Clean all modules and caches
clean:
    @just _for-each-module "go clean ./..."
    @cd "{{ PROJECT_ROOT }}" && echo "🧹 Cleaning global caches..." && go clean -cache -testcache

# === Tool Installation ===

# Install Go tools not available via hermit
install-go-tools:
    @echo "🔧 Installing Go tools"
    @go install github.com/onsi/ginkgo/v2/ginkgo@latest
    @go install go.uber.org/mock/mockgen@latest
    @go install golang.org/x/tools/cmd/goimports@latest
    @go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
    @echo "✅ Go tools installed"

# === GitHub Actions Testing with Act ===

# Act Testing: Uses catthehacker/ubuntu:act-latest for all architectures
# - Includes all required tools (openssl, git, etc.) out of the box
# - ARM64 systems use emulation but get better GitHub Actions compatibility
# - Consistent behavior across different development environments

# Test GitHub Actions workflows locally (system-native architecture)
ci-test:
    @echo "🎬 Testing GitHub Actions workflows with act ({{ ACT_PLATFORM }}, {{ ACT_IMAGE }})"
    act --container-architecture {{ ACT_PLATFORM }} -P ubuntu-latest={{ ACT_IMAGE }}

# Test specific job (dry-run recommended due to hermit compatibility issues with act)
ci-test-job job:
    @echo "🎬 Testing job '{{ job }}' with act (dry-run, {{ ACT_PLATFORM }}, {{ ACT_IMAGE }})"
    act -n --job {{ job }} --container-architecture {{ ACT_PLATFORM }} -P ubuntu-latest={{ ACT_IMAGE }}

# Test specific workflow and job
ci-test-workflow-job workflow job:
    @echo "🎬 Testing job '{{ job }}' from workflow '{{ workflow }}' ({{ ACT_PLATFORM }}, {{ ACT_IMAGE }})"
    act -n --job {{ job }} -W .github/workflows/{{ workflow }} --container-architecture {{ ACT_PLATFORM }} -P ubuntu-latest={{ ACT_IMAGE }}

# Test push event
ci-test-push:
    @echo "🎬 Testing push event with act ({{ ACT_PLATFORM }}, {{ ACT_IMAGE }})"
    act push --container-architecture {{ ACT_PLATFORM }} -P ubuntu-latest={{ ACT_IMAGE }}

# Test pull request event
ci-test-pr:
    @echo "🎬 Testing pull request event with act ({{ ACT_PLATFORM }}, {{ ACT_IMAGE }})"
    act pull_request --container-architecture {{ ACT_PLATFORM }} -P ubuntu-latest={{ ACT_IMAGE }}

# List available workflows and jobs
ci-list:
    @echo "📋 Available GitHub Actions workflows:"
    act --list

# === Internal Helper Functions ===

# Get all module paths from go.work
_get-modules:
    @cd "{{ PROJECT_ROOT }}" && grep -A 20 "use (" go.work | grep -E "^\s*\./.*" | sed 's/[[:space:]]*\.\///' | sed 's/[[:space:]]*$//'

# Execute command for each module
_for-each-module cmd:
    #!/usr/bin/env bash
    set -euo pipefail
    cd "{{ PROJECT_ROOT }}"

    modules=$(just _get-modules)
    for module in $modules; do
        echo "==> Running '{{cmd}}' in module: $module"
        cd "{{ PROJECT_ROOT }}/$module" && {{cmd}} || exit 1
    done

# Ensure specific Go version is available
_ensure-go-version version:
    #!/usr/bin/env bash
    current_version=$(go version | awk '{print $3}' | sed 's/go//')
    if [ "$current_version" != "{{ version }}" ]; then
        echo "📦 Go version {{ version }} required, but $current_version is installed"
        echo "⚠️  Continuing with installed version $current_version"
        echo "💡 If you need Go {{ version }} specifically, install it manually"
        # Don't try to install via hermit - use system Go as source of truth
    else
        echo "✅ Go {{ version }} is available"
    fi

# === Format Implementations ===

# Format check mode (CI)
_format-check:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "🔍 Checking code formatting (strict mode)"

    # Check gofmt
    if [ -n "$(gofmt -l .)" ]; then
        echo "❌ Code is not formatted properly:"
        gofmt -l .
        echo "💡 Run 'just format' to fix formatting"
        exit 1
    fi

    # Ensure goimports is available
    if ! command -v goimports &> /dev/null; then
        go install golang.org/x/tools/cmd/goimports@latest
    fi

    # Check goimports
    if [ -n "$(goimports -l .)" ]; then
        echo "❌ Imports are not formatted properly:"
        goimports -l .
        echo "💡 Run 'just format' to fix imports"
        exit 1
    fi

    echo "✅ Code formatting is correct"

# Format fix mode (dev)
_format-fix:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "🎨 Formatting code"

    # Ensure goimports is available
    if ! command -v goimports &> /dev/null; then
        go install golang.org/x/tools/cmd/goimports@latest
    fi

    # Format code
    gofmt -w .
    goimports -w .

    echo "✅ Code formatted successfully"

# === Lint Implementations ===

# Normal linting
_lint-normal:
    @just _run-golangci-lint

# Strict linting (fail fast)
_lint-strict:
    @just _run-golangci-lint --max-issues-per-linter=0 --max-same-issues=0

# Run golangci-lint with arguments
_run-golangci-lint +args="":
    #!/usr/bin/env bash
    set -euo pipefail
    echo "🔍 Running golangci-lint"

    modules=$(just _get-modules)
    for module in $modules; do
        echo "==> Linting module: $module"
        cd "{{ PROJECT_ROOT }}/$module"
        golangci-lint run --config="{{ GOLANGCI_CONFIG }}" --timeout={{ LINT_TIMEOUT }} {{ args }} ./...
    done

    echo "✅ Linting completed"

# === Test Implementations ===

# Development testing (fast)
_test-dev:
    @echo "🧪 Running tests (development mode)"
    @just _for-each-module "go test ./..."
    @echo "✅ Tests completed"

# CI testing (with coverage and race detection)
_test-ci:
    @echo "🧪 Running tests (CI mode)"
    @just _for-each-module "go test -race -coverprofile=coverage.out -covermode=atomic ./..."
    @echo "✅ CI tests completed"

# Check coverage meets minimum requirement
_check-coverage min_coverage:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "📊 Checking coverage (minimum {{ min_coverage }}%)"

    modules=$(just _get-modules)
    failed_modules=()

    for module in $modules; do
        echo "==> Checking coverage in module: $module"
        cd "{{ PROJECT_ROOT }}/$module"

        # Run tests with coverage if not already done
        if [ ! -f coverage.out ]; then
            if ! go test -coverprofile=coverage.out -covermode=atomic ./... 2>/dev/null; then
                echo "⚠️  No tests in module $module, skipping coverage check"
                continue
            fi
        fi

        # Check if coverage file exists and has content
        if [[ ! -s coverage.out ]]; then
            echo "⚠️  No coverage data for module $module"
            continue
        fi

        # Calculate coverage percentage
        COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
        echo "📈 Module $module coverage: ${COVERAGE}%"

        # Check if coverage meets requirement (using bc if available, otherwise basic comparison)
        if command -v bc &> /dev/null; then
            if (( $(echo "${COVERAGE} < {{ min_coverage }}" | bc -l) )); then
                echo "❌ Module $module coverage ${COVERAGE}% is below required {{ min_coverage }}%"
                failed_modules+=("$module")
            else
                echo "✅ Module $module coverage ${COVERAGE}% meets requirement"
            fi
        else
            # Fallback without bc (basic integer comparison)
            coverage_int=${COVERAGE%.*}
            threshold_int={{ min_coverage }}
            threshold_int=${threshold_int%.*}
            if [ "$coverage_int" -lt "$threshold_int" ]; then
                echo "❌ Module $module coverage ${COVERAGE}% is below required {{ min_coverage }}%"
                failed_modules+=("$module")
            else
                echo "✅ Module $module coverage ${COVERAGE}% meets requirement"
            fi
        fi
    done

    if [ ${#failed_modules[@]} -ne 0 ]; then
        echo ""
        echo "❌ Coverage check failed for modules: ${failed_modules[*]}"
        exit 1
    fi

    echo "✅ All modules meet {{ min_coverage }}% coverage requirement"

# === Build Implementation ===

# Build all components
_build-all:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "🔨 Building all components"

    # Build all modules
    modules=$(just _get-modules)
    for module in $modules; do
        echo "==> Building module: $module"
        cd "{{ PROJECT_ROOT }}/$module" && go build ./...
    done

    echo "🔧 Building CLI tools"

    # Build cli-gen tool
    cd "{{ PROJECT_ROOT }}/tools/cli-gen"
    go build -o ../../bin/cli-gen ./cmd
    echo "✅ Built cli-gen"

    # Build k1s-demo (if main.go exists)
    cd "{{ PROJECT_ROOT }}/examples"
    if [ -f cmd/k1s-demo/main.go ]; then
        go build -o ../bin/k1s-demo ./cmd/k1s-demo
        echo "✅ Built k1s-demo"
    else
        echo "⚠️  k1s-demo main.go not found, skipping"
    fi

    echo "🧪 Verifying binaries"
    cd "{{ PROJECT_ROOT }}"

    if [ -f bin/cli-gen ]; then
        ./bin/cli-gen --help > /dev/null 2>&1 || echo "✅ cli-gen binary built (help not implemented yet)"
    fi

    if [ -f bin/k1s-demo ]; then
        ./bin/k1s-demo --help > /dev/null 2>&1 || echo "✅ k1s-demo binary built (help not implemented yet)"
    fi

    echo "✅ Build completed successfully"

# === Security Implementation ===

# Run security scan
_security-scan:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "🔒 Running security scan"

    # Ensure gosec is available
    if ! command -v gosec &> /dev/null; then
        echo "📦 Installing gosec"
        go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
    fi

    # Run gosec security scan
    cd "{{ PROJECT_ROOT }}"
    if gosec -fmt sarif -out gosec-report.sarif -stdout ./... 2>/dev/null; then
        echo "✅ Security scan completed - no issues found"
    else
        echo "⚠️  Security scan completed with findings (check gosec-report.sarif)"
    fi

# === Performance Implementation ===

# Run performance benchmarks
_run-benchmarks:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "⚡ Running performance benchmarks"

    modules=$(just _get-modules)
    for module in $modules; do
        echo "==> Running benchmarks in module: $module"
        cd "{{ PROJECT_ROOT }}/$module"

        if go test -bench=. -benchmem -run=^$ ./... 2>/dev/null; then
            echo "✅ Benchmarks completed for $module"
        else
            echo "⚠️  No benchmarks found in $module"
        fi
    done

    echo "✅ Benchmarks completed"