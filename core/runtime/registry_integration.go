package runtime

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/dtomasi/k1s/core/registry"
)

// DefaultRegistry is the global registry instance used by the runtime.
var DefaultRegistry = registry.NewRegistry(
	registry.WithDefaultCategories("all"),
	registry.WithShortNameValidation(true),
	registry.WithCaseSensitiveShortNames(false),
)

// RegisterCoreResourcesWithRegistry registers all core resources with the default registry.
// This function is called during runtime initialization to set up resource metadata,
// print columns, and short names for CLI operations.
func RegisterCoreResourcesWithRegistry() error {
	return registry.RegisterCoreResources(DefaultRegistry)
}

// GetResourceRegistry returns the global resource registry instance.
func GetResourceRegistry() registry.Registry {
	return DefaultRegistry
}

// InitializeCoreResources performs complete initialization of core resources
// including scheme registration and registry setup.
func InitializeCoreResources(scheme *runtime.Scheme) error {
	// Register core resources with the scheme
	if err := RegisterCoreResources(scheme); err != nil {
		return fmt.Errorf("failed to register core resources with scheme: %w", err)
	}

	// Register GVK/GVR mappings
	RegisterCoreResourceMappings()

	// Register resources with the registry for CLI support
	if err := RegisterCoreResourcesWithRegistry(); err != nil {
		return fmt.Errorf("failed to register core resources with registry: %w", err)
	}

	return nil
}
