package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ResourceInfo contains metadata about a core resource type.
type ResourceInfo struct {
	// GVK is the GroupVersionKind for this resource
	GVK schema.GroupVersionKind
	// GVR is the GroupVersionResource for this resource
	GVR schema.GroupVersionResource
	// Singular is the singular name of the resource
	Singular string
	// Plural is the plural name of the resource
	Plural string
	// ShortNames are the short names for CLI usage
	ShortNames []string
	// Categories are the resource categories for grouping
	Categories []string
	// NamespaceScoped indicates if the resource is namespace-scoped
	NamespaceScoped bool
	// PrintColumns defines the table columns for CLI display
	PrintColumns []metav1.TableColumnDefinition
	// PrintColumnsWithNamespace defines the table columns including namespace
	PrintColumnsWithNamespace []metav1.TableColumnDefinition
}

// GetCoreResourceInfos returns metadata for all core resource types.
func GetCoreResourceInfos() map[string]ResourceInfo {
	return map[string]ResourceInfo{
		"Namespace": {
			GVK:                       GetNamespaceGVK(),
			GVR:                       GetNamespaceGVR(),
			Singular:                  "namespace",
			Plural:                    "namespaces",
			ShortNames:                GetNamespaceShortNames(),
			Categories:                GetNamespaceCategories(),
			NamespaceScoped:           IsNamespaceScoped(),
			PrintColumns:              GetNamespacePrintColumns(),
			PrintColumnsWithNamespace: GetNamespacePrintColumns(), // Namespace doesn't need namespace column
		},
		"ConfigMap": {
			GVK:                       GetConfigMapGVK(),
			GVR:                       GetConfigMapGVR(),
			Singular:                  "configmap",
			Plural:                    "configmaps",
			ShortNames:                GetConfigMapShortNames(),
			Categories:                GetConfigMapCategories(),
			NamespaceScoped:           IsConfigMapNamespaceScoped(),
			PrintColumns:              GetConfigMapPrintColumns(),
			PrintColumnsWithNamespace: GetConfigMapPrintColumnsWithNamespace(),
		},
		"Secret": {
			GVK:                       GetSecretGVK(),
			GVR:                       GetSecretGVR(),
			Singular:                  "secret",
			Plural:                    "secrets",
			ShortNames:                GetSecretShortNames(),
			Categories:                GetSecretCategories(),
			NamespaceScoped:           IsSecretNamespaceScoped(),
			PrintColumns:              GetSecretPrintColumns(),
			PrintColumnsWithNamespace: GetSecretPrintColumnsWithNamespace(),
		},
		"ServiceAccount": {
			GVK:                       GetServiceAccountGVK(),
			GVR:                       GetServiceAccountGVR(),
			Singular:                  "serviceaccount",
			Plural:                    "serviceaccounts",
			ShortNames:                GetServiceAccountShortNames(),
			Categories:                GetServiceAccountCategories(),
			NamespaceScoped:           IsServiceAccountNamespaceScoped(),
			PrintColumns:              GetServiceAccountPrintColumns(),
			PrintColumnsWithNamespace: GetServiceAccountPrintColumnsWithNamespace(),
		},
		"Event": {
			GVK:                       GetEventGVK(),
			GVR:                       GetEventGVR(),
			Singular:                  "event",
			Plural:                    "events",
			ShortNames:                GetEventShortNames(),
			Categories:                GetEventCategories(),
			NamespaceScoped:           IsEventNamespaceScoped(),
			PrintColumns:              GetEventPrintColumns(),
			PrintColumnsWithNamespace: GetEventPrintColumnsWithNamespace(),
		},
	}
}

// GetResourceInfoByGVK returns the ResourceInfo for a given GroupVersionKind.
func GetResourceInfoByGVK(gvk schema.GroupVersionKind) (ResourceInfo, bool) {
	infos := GetCoreResourceInfos()
	for _, info := range infos {
		if info.GVK == gvk {
			return info, true
		}
	}
	return ResourceInfo{}, false
}

// GetResourceInfoByGVR returns the ResourceInfo for a given GroupVersionResource.
func GetResourceInfoByGVR(gvr schema.GroupVersionResource) (ResourceInfo, bool) {
	infos := GetCoreResourceInfos()
	for _, info := range infos {
		if info.GVR == gvr {
			return info, true
		}
	}
	return ResourceInfo{}, false
}

// GetResourceInfoByKind returns the ResourceInfo for a given kind name.
func GetResourceInfoByKind(kind string) (ResourceInfo, bool) {
	infos := GetCoreResourceInfos()
	if info, exists := infos[kind]; exists {
		return info, true
	}
	return ResourceInfo{}, false
}
