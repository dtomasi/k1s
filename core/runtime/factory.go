// Package runtime provides factory functions for easy k1s runtime initialization.
package runtime

import (
	"fmt"
	"path/filepath"

	"github.com/dtomasi/k1s/core/storage"
	memorystorage "github.com/dtomasi/k1s/storage/memory"
	pebblestorage "github.com/dtomasi/k1s/storage/pebble"
)

// NewDefaultRuntime creates a k1s runtime with sensible defaults for CLI applications.
// It uses memory storage by default for quick startup and no persistence requirements.
func NewDefaultRuntime() (Runtime, error) {
	// Use memory storage with default config
	memoryStorage := memorystorage.NewMemoryStorage(storage.Config{})

	return NewRuntime(memoryStorage)
}

// NewRuntimeWithMemoryStorage creates a k1s runtime with in-memory storage.
// Good for development, testing, and CLI applications that don't need persistence.
func NewRuntimeWithMemoryStorage() (Runtime, error) {
	memoryStorage := memorystorage.NewMemoryStorage(storage.Config{})
	return NewRuntime(memoryStorage)
}

// NewRuntimeWithPebbleStorage creates a k1s runtime with PebbleDB storage.
// Good for CLI applications that need persistence and high performance.
func NewRuntimeWithPebbleStorage(dbPath string) (Runtime, error) {
	if dbPath == "" {
		dbPath = "./data/k1s.db"
	}

	// Ensure directory exists by expanding the path
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve database path %s: %w", dbPath, err)
	}

	pebbleStorage := pebblestorage.NewPebbleStorageWithPath(absPath, storage.Config{})

	return NewRuntime(pebbleStorage)
}

// NewRuntimeWithTenant creates a k1s runtime with the specified tenant ID.
// Useful for multi-tenant CLI applications or namespace isolation.
func NewRuntimeWithTenant(tenantID string, dbPath string) (Runtime, error) {
	config := storage.Config{
		TenantID: tenantID,
	}

	var backend storage.Interface
	if dbPath == "" {
		// Use memory storage for empty path
		backend = memorystorage.NewMemoryStorage(config)
	} else {
		absPath, err := filepath.Abs(dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve database path %s: %w", dbPath, err)
		}
		backend = pebblestorage.NewPebbleStorageWithPath(absPath, config)
	}

	return NewRuntime(backend, WithTenant(tenantID))
}

// RuntimeType represents different types of runtime configurations
type RuntimeType string

const (
	// RuntimeTypeMemory uses in-memory storage (no persistence)
	RuntimeTypeMemory RuntimeType = "memory"
	// RuntimeTypePebble uses PebbleDB storage (persistent)
	RuntimeTypePebble RuntimeType = "pebble"
)

// SimpleRuntimeConfig contains basic configuration for creating a runtime
type SimpleRuntimeConfig struct {
	Type     RuntimeType
	DBPath   string
	TenantID string
}

// NewRuntimeFromConfig creates a k1s runtime from simple configuration.
// This is the most flexible factory function for CLI applications.
func NewRuntimeFromConfig(config SimpleRuntimeConfig) (Runtime, error) {
	switch config.Type {
	case RuntimeTypeMemory:
		if config.TenantID != "" {
			return NewRuntimeWithTenant(config.TenantID, "")
		}
		return NewRuntimeWithMemoryStorage()

	case RuntimeTypePebble:
		if config.TenantID != "" {
			return NewRuntimeWithTenant(config.TenantID, config.DBPath)
		}
		return NewRuntimeWithPebbleStorage(config.DBPath)

	default:
		return nil, fmt.Errorf("unsupported runtime type: %s", config.Type)
	}
}
