package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ConfigMap holds non-sensitive configuration data in key-value pairs.
// It directly uses the standard Kubernetes corev1.ConfigMap for full compatibility.
type ConfigMap = corev1.ConfigMap

// ConfigMapList represents a list of ConfigMap objects.
type ConfigMapList = corev1.ConfigMapList

var (
	// ConfigMapGVK is the GroupVersionKind for ConfigMap.
	ConfigMapGVK = schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "ConfigMap",
	}

	// ConfigMapGVR is the GroupVersionResource for ConfigMap.
	ConfigMapGVR = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "configmaps",
	}
)

// GetConfigMapGVK returns the GroupVersionKind for ConfigMap.
func GetConfigMapGVK() schema.GroupVersionKind {
	return ConfigMapGVK
}

// GetConfigMapGVR returns the GroupVersionResource for ConfigMap.
func GetConfigMapGVR() schema.GroupVersionResource {
	return ConfigMapGVR
}

// NewConfigMap creates a new ConfigMap with the given name and namespace.
func NewConfigMap(name, namespace string) *ConfigMap {
	return &ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: make(map[string]string),
	}
}

// NewConfigMapWithData creates a new ConfigMap with the given name, namespace and data.
func NewConfigMapWithData(name, namespace string, data map[string]string) *ConfigMap {
	cm := NewConfigMap(name, namespace)
	cm.Data = data
	return cm
}

// IsConfigMapNamespaceScoped returns true as ConfigMap is a namespace-scoped resource.
func IsConfigMapNamespaceScoped() bool {
	return true
}

// GetConfigMapShortNames returns short names for ConfigMap resource.
func GetConfigMapShortNames() []string {
	return []string{"cm"}
}

// GetConfigMapCategories returns categories for ConfigMap resource.
func GetConfigMapCategories() []string {
	return []string{"all"}
}

// GetConfigMapPrintColumns returns table columns for ConfigMap display.
func GetConfigMapPrintColumns() []metav1.TableColumnDefinition {
	return []metav1.TableColumnDefinition{
		{
			Name:        "Name",
			Type:        "string",
			Format:      "name",
			Description: "Name of the configmap",
			Priority:    0,
		},
		{
			Name:        "Data",
			Type:        "integer",
			Format:      "",
			Description: "Number of data entries",
			Priority:    0,
		},
		{
			Name:        "Age",
			Type:        "string",
			Format:      "",
			Description: "Age of the configmap",
			Priority:    0,
		},
	}
}

// GetConfigMapPrintColumnsWithNamespace returns table columns for ConfigMap display including namespace.
func GetConfigMapPrintColumnsWithNamespace() []metav1.TableColumnDefinition {
	return []metav1.TableColumnDefinition{
		{
			Name:        "Namespace",
			Type:        "string",
			Format:      "",
			Description: "Namespace of the configmap",
			Priority:    0,
		},
		{
			Name:        "Name",
			Type:        "string",
			Format:      "name",
			Description: "Name of the configmap",
			Priority:    0,
		},
		{
			Name:        "Data",
			Type:        "integer",
			Format:      "",
			Description: "Number of data entries",
			Priority:    0,
		},
		{
			Name:        "Age",
			Type:        "string",
			Format:      "",
			Description: "Age of the configmap",
			Priority:    0,
		},
	}
}

// AddConfigMapToScheme adds ConfigMap types to the given scheme.
func AddConfigMapToScheme(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(schema.GroupVersion{Group: "", Version: "v1"},
		&ConfigMap{},
		&ConfigMapList{},
	)
	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Group: "", Version: "v1"})
	return nil
}
