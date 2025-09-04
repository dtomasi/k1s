// Package cli provides kubectl-compatible CLI operations for k1s.
//
// This package implements the CLI runtime layer that provides standard
// kubectl-style commands and operations, built on top of the k1s core.
//
// It includes:
//   - Resource CRUD operations (get, create, apply, delete, patch)
//   - Output formatters (table, JSON, YAML, custom columns)
//   - Operation builders with fluent API
//   - Selector and filtering support
//   - Watch and streaming operations
package cli
