# Local CI Validation Guide

This document provides commands to validate your changes locally before pushing, ensuring they'll pass the CI pipeline.

## Pre-Push Validation Script

Run all these commands in sequence to validate your changes:

```bash
#!/bin/bash
set -e

echo "🔍 K1S Local CI Validation"
echo "=========================="

# 1. Format Check
echo "1. Checking code formatting..."
if [ -n "$(gofmt -l .)" ]; then
  echo "❌ Code is not formatted. Running gofmt..."
  gofmt -w .
else
  echo "✅ Code formatting is correct"
fi

# 2. Import Check  
echo "2. Checking imports..."
go install golang.org/x/tools/cmd/goimports@latest
if [ -n "$(goimports -l .)" ]; then
  echo "❌ Imports are not formatted. Running goimports..."
  goimports -w .
else
  echo "✅ Import formatting is correct"
fi

# 3. Workspace Sync
echo "3. Syncing workspace..."
go work sync

# 4. Build Check
echo "4. Building all modules..."
just build

# 5. Test Execution
echo "5. Running all tests..."
just test

# 6. Coverage Check (70% minimum)
echo "6. Checking coverage requirements..."
# Note: This will fail if coverage < 70%, which is expected during development
just coverage-check || echo "⚠️ Some modules don't meet 70% coverage yet (expected during development)"

# 7. Linting (comprehensive)
echo "7. Running comprehensive linting..."
just lint || echo "⚠️ Linting issues found - please fix before pushing"

echo ""
echo "🎉 Local validation complete!"
echo "Note: Coverage and linting warnings are expected during active development."
echo "All issues must be resolved before PR merge."
```

## Quick Validation Commands

### Format & Build Only
```bash
# Quick check before commit
gofmt -w . && goimports -w . && just build && just test
```

### Coverage for Specific Module
```bash
# Check coverage for a single module
cd core && go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

### Linting for Specific Module
```bash  
# Lint a single module
cd core && golangci-lint run --config=../.golangci.yml --timeout=10m ./...
```

## CI Pipeline Jobs Overview

Our CI runs these jobs in sequence:

1. **Format Check** - `gofmt` and `goimports` validation
2. **Lint** - 50+ golangci-lint rules across all modules
3. **Test** - Cross-platform testing (Linux, macOS, Windows)  
4. **Coverage** - 70% minimum enforcement with automatic failure
5. **Build** - Multi-module build verification
6. **Performance** - Benchmark execution
7. **Security** - gosec vulnerability scanning

## Troubleshooting

### act Docker Issues
If `act` fails with TLS certificate errors, this is a known Docker networking issue. The workflow is correct, but Docker container certificate validation fails in some environments.

**Workaround**: Validate locally with the commands above instead of `act`.

### Module Path Issues
Ensure you're in the project root when running `just` commands. The justfile handles module path resolution automatically.

### Coverage Issues  
Coverage below 70% will cause CI failure. During development, this is expected. Focus on implementing functionality first, then add comprehensive tests to meet coverage requirements.

### Linting Issues
Our golangci-lint configuration is strict (50+ linters). Use `just lint-fix` to auto-fix issues where possible. Some issues require manual resolution.