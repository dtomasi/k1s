package v1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: "", Version: "v1"}

// AddToScheme adds all core resource types to the given scheme.
// This function registers all the core Kubernetes resource types:
// Namespace, ConfigMap, Secret, ServiceAccount, and Event.
func AddToScheme(s *runtime.Scheme) error {
	// Add Namespace types
	if err := AddNamespaceToScheme(s); err != nil {
		return err
	}

	// Add ConfigMap types
	if err := AddConfigMapToScheme(s); err != nil {
		return err
	}

	// Add Secret types
	if err := AddSecretToScheme(s); err != nil {
		return err
	}

	// Add ServiceAccount types
	if err := AddServiceAccountToScheme(s); err != nil {
		return err
	}

	// Add Event types
	if err := AddEventToScheme(s); err != nil {
		return err
	}

	return nil
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

// Kind takes an unqualified kind and returns back a Group qualified GroupVersionKind
func Kind(kind string) schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind(kind)
}

// GetAllGVKs returns all GroupVersionKinds for core resources.
func GetAllGVKs() []schema.GroupVersionKind {
	return []schema.GroupVersionKind{
		GetNamespaceGVK(),
		GetConfigMapGVK(),
		GetSecretGVK(),
		GetServiceAccountGVK(),
		GetEventGVK(),
	}
}

// GetAllGVRs returns all GroupVersionResources for core resources.
func GetAllGVRs() []schema.GroupVersionResource {
	return []schema.GroupVersionResource{
		GetNamespaceGVR(),
		GetConfigMapGVR(),
		GetSecretGVR(),
		GetServiceAccountGVR(),
		GetEventGVR(),
	}
}

// GetGVKToGVRMappings returns all GVK to GVR mappings for core resources.
func GetGVKToGVRMappings() map[schema.GroupVersionKind]schema.GroupVersionResource {
	return map[schema.GroupVersionKind]schema.GroupVersionResource{
		GetNamespaceGVK():      GetNamespaceGVR(),
		GetConfigMapGVK():      GetConfigMapGVR(),
		GetSecretGVK():         GetSecretGVR(),
		GetServiceAccountGVK(): GetServiceAccountGVR(),
		GetEventGVK():          GetEventGVR(),
	}
}

// GetGVRToGVKMappings returns all GVR to GVK mappings for core resources.
func GetGVRToGVKMappings() map[schema.GroupVersionResource]schema.GroupVersionKind {
	return map[schema.GroupVersionResource]schema.GroupVersionKind{
		GetNamespaceGVR():      GetNamespaceGVK(),
		GetConfigMapGVR():      GetConfigMapGVK(),
		GetSecretGVR():         GetSecretGVK(),
		GetServiceAccountGVR(): GetServiceAccountGVK(),
		GetEventGVR():          GetEventGVK(),
	}
}
