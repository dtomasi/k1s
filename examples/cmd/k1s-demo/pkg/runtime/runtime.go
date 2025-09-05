package runtime

import (
	"context"
	"fmt"

	coreruntime "github.com/dtomasi/k1s/core/runtime"
)

// Runtime wraps the core k1s runtime for demo CLI usage
type Runtime interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	GetCoreRuntime() coreruntime.Runtime
}

// runtime implements Runtime interface
type runtime struct {
	coreRuntime coreruntime.Runtime
}

// Config contains configuration for the demo runtime
type Config struct {
	StorageType string
	DBPath      string
	TenantID    string
}

// NewRuntime creates a new demo runtime from configuration
func NewRuntime(config Config) (Runtime, error) {
	// Create simple runtime config
	runtimeConfig := coreruntime.SimpleRuntimeConfig{
		Type:     coreruntime.RuntimeType(config.StorageType),
		DBPath:   config.DBPath,
		TenantID: config.TenantID,
	}

	// Create core runtime
	coreRuntime, err := coreruntime.NewRuntimeFromConfig(runtimeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create core runtime: %w", err)
	}

	return &runtime{
		coreRuntime: coreRuntime,
	}, nil
}

// Start starts the runtime
func (r *runtime) Start(ctx context.Context) error {
	return r.coreRuntime.Start(ctx)
}

// Stop stops the runtime
func (r *runtime) Stop(ctx context.Context) error {
	return r.coreRuntime.Stop(ctx)
}

// GetCoreRuntime returns the underlying core runtime
func (r *runtime) GetCoreRuntime() coreruntime.Runtime {
	return r.coreRuntime
}
