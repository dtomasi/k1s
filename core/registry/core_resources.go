package registry

import (
	"fmt"

	corev1 "github.com/dtomasi/k1s/core/types/v1"
)

// RegisterCoreResources registers all core Kubernetes resources with the given registry.
// This function adds Namespace, ConfigMap, Secret, ServiceAccount, and Event resources
// with their metadata, print columns, and short names.
func RegisterCoreResources(registry Registry) error {
	// Get all core resource information
	coreResourceInfos := corev1.GetCoreResourceInfos()

	// Register each core resource
	for _, info := range coreResourceInfos {
		config := ResourceConfig{
			Singular:     info.Singular,
			Plural:       info.Plural,
			Kind:         info.GVK.Kind,
			ListKind:     info.GVK.Kind + "List",
			Namespaced:   info.NamespaceScoped,
			ShortNames:   info.ShortNames,
			Categories:   info.Categories,
			PrintColumns: info.PrintColumns,
			Description:  getResourceDescription(info.GVK.Kind),
		}

		if err := registry.RegisterResource(info.GVR, config); err != nil {
			return fmt.Errorf("failed to register core resource %s: %w", info.GVK.Kind, err)
		}
	}

	return nil
}

// getResourceDescription returns a human-readable description for core resources.
func getResourceDescription(kind string) string {
	descriptions := map[string]string{
		"Namespace":      "Provides multi-tenancy and resource organization",
		"ConfigMap":      "Holds non-sensitive configuration data in key-value pairs",
		"Secret":         "Holds sensitive data such as passwords, OAuth tokens, and ssh keys",
		"ServiceAccount": "Provides identity for processes that run in pods",
		"Event":          "Records events in the system for observability and debugging",
	}

	if desc, exists := descriptions[kind]; exists {
		return desc
	}
	return fmt.Sprintf("Core Kubernetes resource: %s", kind)
}

// GetCoreResourceGVRs returns all GVRs for core resources.
func GetCoreResourceGVRs() []string {
	return []string{
		"v1/namespaces",
		"v1/configmaps",
		"v1/secrets",
		"v1/serviceaccounts",
		"v1/events",
	}
}

// IsCoreResource checks if the given GVR is a core resource.
func IsCoreResource(gvr string) bool {
	coreGVRs := GetCoreResourceGVRs()
	for _, core := range coreGVRs {
		if core == gvr {
			return true
		}
	}
	return false
}
