package runtime

import (
	"k8s.io/apimachinery/pkg/runtime"

	corev1 "github.com/dtomasi/k1s/core/types/v1"
)

// RegisterCoreResources registers all core Kubernetes resources with the given scheme.
// This function is called during scheme initialization to automatically register
// core resources like Namespace, ConfigMap, Secret, ServiceAccount, and Event.
func RegisterCoreResources(scheme *runtime.Scheme) error {
	// Register all core resource types using the resources/v1 package
	return corev1.AddToScheme(scheme)
}

// RegisterCoreResourceMappings registers GVK/GVR mappings for core resources
// with the global GVK mapper.
func RegisterCoreResourceMappings() {
	// Get all GVK to GVR mappings for core resources
	mappings := corev1.GetGVKToGVRMappings()

	// Register each mapping with the global mapper
	for gvk, gvr := range mappings {
		RegisterGlobalMapping(gvk, gvr)
	}
}
