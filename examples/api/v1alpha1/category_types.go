package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CategorySpec defines the desired state of Category
type CategorySpec struct {
	// Name is the display name of the category
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=50
	Name string `json:"name"`

	// Description provides details about the category
	// +kubebuilder:validation:MaxLength=200
	// +optional
	Description string `json:"description,omitempty"`

	// Parent is the name of the parent category for hierarchical organization
	// +optional
	Parent string `json:"parent,omitempty"`

	// Tags are labels for organizing and filtering categories
	// +optional
	Tags []string `json:"tags,omitempty"`
}

// CategoryStatus defines the observed state of Category
type CategoryStatus struct {
	// ItemCount is the number of items in this category
	// +kubebuilder:default=0
	ItemCount int32 `json:"itemCount"`

	// SubCategoryCount is the number of subcategories
	// +kubebuilder:default=0
	SubCategoryCount int32 `json:"subCategoryCount"`

	// LastUpdated is the timestamp when the category was last modified
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=cat
// +kubebuilder:printcolumn:name="Category Name",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="Items",type=integer,JSONPath=`.status.itemCount`
// +kubebuilder:printcolumn:name="Subcategories",type=integer,JSONPath=`.status.subCategoryCount`
// +kubebuilder:printcolumn:name="Parent",type=string,JSONPath=`.spec.parent`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Category is the Schema for the categories API
type Category struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CategorySpec   `json:"spec,omitempty"`
	Status CategoryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CategoryList contains a list of Category
type CategoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Category `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Category{}, &CategoryList{})
}
