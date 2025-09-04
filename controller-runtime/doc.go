// Package controller provides simplified controller-runtime for k1s.
//
// This package implements a lightweight controller-runtime that adapts
// standard Kubernetes controller patterns for CLI-optimized environments.
//
// It includes:
//   - Manager interface for controller lifecycle
//   - Controller interface for resource watching
//   - Reconciler interface for business logic
//   - Builder API for fluent controller configuration
//   - Event recorder integration
package controller
