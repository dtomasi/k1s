package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Namespace provides multi-tenancy and resource organization capabilities.
// It directly uses the standard Kubernetes corev1.Namespace for full compatibility.
type Namespace = corev1.Namespace

// NamespaceList represents a list of Namespace objects.
type NamespaceList = corev1.NamespaceList

var (
	// NamespaceGVK is the GroupVersionKind for Namespace.
	NamespaceGVK = schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Namespace",
	}

	// NamespaceGVR is the GroupVersionResource for Namespace.
	NamespaceGVR = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "namespaces",
	}
)

// GetNamespaceGVK returns the GroupVersionKind for Namespace.
func GetNamespaceGVK() schema.GroupVersionKind {
	return NamespaceGVK
}

// GetNamespaceGVR returns the GroupVersionResource for Namespace.
func GetNamespaceGVR() schema.GroupVersionResource {
	return NamespaceGVR
}

// NewNamespace creates a new Namespace with the given name.
func NewNamespace(name string) *Namespace {
	return &Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.NamespaceSpec{},
	}
}

// IsNamespaceScoped returns false as Namespace is a cluster-scoped resource.
func IsNamespaceScoped() bool {
	return false
}

// GetNamespaceShortNames returns short names for Namespace resource.
func GetNamespaceShortNames() []string {
	return []string{"ns"}
}

// GetNamespaceCategories returns categories for Namespace resource.
func GetNamespaceCategories() []string {
	return []string{"all"}
}

// GetNamespacePrintColumns returns table columns for Namespace display.
func GetNamespacePrintColumns() []metav1.TableColumnDefinition {
	return []metav1.TableColumnDefinition{
		{
			Name:        "Name",
			Type:        "string",
			Format:      "name",
			Description: "Name of the namespace",
			Priority:    0,
		},
		{
			Name:        "Status",
			Type:        "string",
			Format:      "",
			Description: "The status of the namespace",
			Priority:    0,
		},
		{
			Name:        "Age",
			Type:        "string",
			Format:      "",
			Description: "Age of the namespace",
			Priority:    0,
		},
	}
}

// AddNamespaceToScheme adds Namespace types to the given scheme.
func AddNamespaceToScheme(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(schema.GroupVersion{Group: "", Version: "v1"},
		&Namespace{},
		&NamespaceList{},
	)
	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Group: "", Version: "v1"})
	return nil
}
