package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ItemSpec defines the desired state of Item
type ItemSpec struct {
	// Name is the display name of the item
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=100
	Name string `json:"name"`

	// Description provides details about the item
	// +kubebuilder:validation:MaxLength=500
	// +optional
	Description string `json:"description,omitempty"`

	// Quantity is the number of items available
	// +kubebuilder:validation:CEL:rule="self >= 0",message="quantity must be non-negative"
	// +kubebuilder:validation:CEL:rule="self <= 10000",message="quantity cannot exceed 10000"
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=1
	Quantity int32 `json:"quantity"`

	// Price is the cost of the item in cents
	// +kubebuilder:validation:CEL:rule="self > 0",message="price must be positive"
	// +kubebuilder:validation:Minimum=0
	// +optional
	Price *int64 `json:"price,omitempty"`

	// Category is the name of the category this item belongs to
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Category string `json:"category"`
}

// ItemStatus defines the observed state of Item
type ItemStatus struct {
	// Status represents the current state of the item
	// +kubebuilder:validation:Enum=Available;Reserved;Sold;Discontinued
	// +kubebuilder:default=Available
	Status string `json:"status"`

	// LastUpdated is the timestamp when the item was last modified
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`

	// ReservedBy indicates who has reserved this item
	// +optional
	ReservedBy string `json:"reservedBy,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=itm
// +kubebuilder:printcolumn:name="Item Name",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="Quantity",type=integer,JSONPath=`.spec.quantity`
// +kubebuilder:printcolumn:name="Category",type=string,JSONPath=`.spec.category`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
// +kubebuilder:printcolumn:name="Price",type=string,JSONPath=`.spec.price`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Item is the Schema for the items API
type Item struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ItemSpec   `json:"spec,omitempty"`
	Status ItemStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ItemList contains a list of Item
type ItemList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Item `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Item{}, &ItemList{})
}
