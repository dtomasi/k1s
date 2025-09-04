package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Secret holds sensitive data such as passwords, OAuth tokens, and ssh keys.
// It directly uses the standard Kubernetes corev1.Secret for full compatibility.
type Secret = corev1.Secret

// SecretList represents a list of Secret objects.
type SecretList = corev1.SecretList

var (
	// SecretGVK is the GroupVersionKind for Secret.
	SecretGVK = schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Secret",
	}

	// SecretGVR is the GroupVersionResource for Secret.
	SecretGVR = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "secrets",
	}
)

// GetSecretGVK returns the GroupVersionKind for Secret.
func GetSecretGVK() schema.GroupVersionKind {
	return SecretGVK
}

// GetSecretGVR returns the GroupVersionResource for Secret.
func GetSecretGVR() schema.GroupVersionResource {
	return SecretGVR
}

// NewSecret creates a new Secret with the given name and namespace.
func NewSecret(name, namespace string, secretType corev1.SecretType) *Secret {
	return &Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: secretType,
		Data: make(map[string][]byte),
	}
}

// NewOpaqueSecret creates a new Secret with Opaque type.
func NewOpaqueSecret(name, namespace string) *Secret {
	return NewSecret(name, namespace, corev1.SecretTypeOpaque)
}

// NewSecretWithData creates a new Secret with the given name, namespace, type and data.
func NewSecretWithData(name, namespace string, secretType corev1.SecretType, data map[string][]byte) *Secret {
	secret := NewSecret(name, namespace, secretType)
	secret.Data = data
	return secret
}

// NewSecretFromStringData creates a new Secret from string data (will be base64 encoded).
func NewSecretFromStringData(name, namespace string, secretType corev1.SecretType, stringData map[string]string) *Secret {
	secret := NewSecret(name, namespace, secretType)
	secret.StringData = stringData
	return secret
}

// IsSecretNamespaceScoped returns true as Secret is a namespace-scoped resource.
func IsSecretNamespaceScoped() bool {
	return true
}

// GetSecretShortNames returns short names for Secret resource.
func GetSecretShortNames() []string {
	return []string{}
}

// GetSecretCategories returns categories for Secret resource.
func GetSecretCategories() []string {
	return []string{"all"}
}

// GetSecretPrintColumns returns table columns for Secret display.
func GetSecretPrintColumns() []metav1.TableColumnDefinition {
	return []metav1.TableColumnDefinition{
		{
			Name:        "Name",
			Type:        "string",
			Format:      "name",
			Description: "Name of the secret",
			Priority:    0,
		},
		{
			Name:        "Type",
			Type:        "string",
			Format:      "",
			Description: "Type of the secret",
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
			Description: "Age of the secret",
			Priority:    0,
		},
	}
}

// GetSecretPrintColumnsWithNamespace returns table columns for Secret display including namespace.
func GetSecretPrintColumnsWithNamespace() []metav1.TableColumnDefinition {
	return []metav1.TableColumnDefinition{
		{
			Name:        "Namespace",
			Type:        "string",
			Format:      "",
			Description: "Namespace of the secret",
			Priority:    0,
		},
		{
			Name:        "Name",
			Type:        "string",
			Format:      "name",
			Description: "Name of the secret",
			Priority:    0,
		},
		{
			Name:        "Type",
			Type:        "string",
			Format:      "",
			Description: "Type of the secret",
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
			Description: "Age of the secret",
			Priority:    0,
		},
	}
}

// AddSecretToScheme adds Secret types to the given scheme.
func AddSecretToScheme(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(schema.GroupVersion{Group: "", Version: "v1"},
		&Secret{},
		&SecretList{},
	)
	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Group: "", Version: "v1"})
	return nil
}
