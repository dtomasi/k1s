package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ServiceAccount provides identity for processes that run in pods.
// It directly uses the standard Kubernetes corev1.ServiceAccount for full compatibility.
type ServiceAccount = corev1.ServiceAccount

// ServiceAccountList represents a list of ServiceAccount objects.
type ServiceAccountList = corev1.ServiceAccountList

var (
	// ServiceAccountGVK is the GroupVersionKind for ServiceAccount.
	ServiceAccountGVK = schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "ServiceAccount",
	}

	// ServiceAccountGVR is the GroupVersionResource for ServiceAccount.
	ServiceAccountGVR = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "serviceaccounts",
	}
)

// GetServiceAccountGVK returns the GroupVersionKind for ServiceAccount.
func GetServiceAccountGVK() schema.GroupVersionKind {
	return ServiceAccountGVK
}

// GetServiceAccountGVR returns the GroupVersionResource for ServiceAccount.
func GetServiceAccountGVR() schema.GroupVersionResource {
	return ServiceAccountGVR
}

// NewServiceAccount creates a new ServiceAccount with the given name and namespace.
func NewServiceAccount(name, namespace string) *ServiceAccount {
	return &ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		AutomountServiceAccountToken: func() *bool { b := true; return &b }(),
	}
}

// NewServiceAccountWithSecrets creates a new ServiceAccount with associated secrets.
func NewServiceAccountWithSecrets(name, namespace string, secrets []corev1.ObjectReference) *ServiceAccount {
	sa := NewServiceAccount(name, namespace)
	sa.Secrets = secrets
	return sa
}

// IsServiceAccountNamespaceScoped returns true as ServiceAccount is a namespace-scoped resource.
func IsServiceAccountNamespaceScoped() bool {
	return true
}

// GetServiceAccountShortNames returns short names for ServiceAccount resource.
func GetServiceAccountShortNames() []string {
	return []string{"sa"}
}

// GetServiceAccountCategories returns categories for ServiceAccount resource.
func GetServiceAccountCategories() []string {
	return []string{"all"}
}

// GetServiceAccountPrintColumns returns table columns for ServiceAccount display.
func GetServiceAccountPrintColumns() []metav1.TableColumnDefinition {
	return []metav1.TableColumnDefinition{
		{
			Name:        "Name",
			Type:        "string",
			Format:      "name",
			Description: "Name of the service account",
			Priority:    0,
		},
		{
			Name:        "Secrets",
			Type:        "integer",
			Format:      "",
			Description: "Number of associated secrets",
			Priority:    0,
		},
		{
			Name:        "Age",
			Type:        "string",
			Format:      "",
			Description: "Age of the service account",
			Priority:    0,
		},
	}
}

// GetServiceAccountPrintColumnsWithNamespace returns table columns for ServiceAccount display including namespace.
func GetServiceAccountPrintColumnsWithNamespace() []metav1.TableColumnDefinition {
	return []metav1.TableColumnDefinition{
		{
			Name:        "Namespace",
			Type:        "string",
			Format:      "",
			Description: "Namespace of the service account",
			Priority:    0,
		},
		{
			Name:        "Name",
			Type:        "string",
			Format:      "name",
			Description: "Name of the service account",
			Priority:    0,
		},
		{
			Name:        "Secrets",
			Type:        "integer",
			Format:      "",
			Description: "Number of associated secrets",
			Priority:    0,
		},
		{
			Name:        "Age",
			Type:        "string",
			Format:      "",
			Description: "Age of the service account",
			Priority:    0,
		},
	}
}

// AddServiceAccountToScheme adds ServiceAccount types to the given scheme.
func AddServiceAccountToScheme(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(schema.GroupVersion{Group: "", Version: "v1"},
		&ServiceAccount{},
		&ServiceAccountList{},
	)
	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Group: "", Version: "v1"})
	return nil
}
