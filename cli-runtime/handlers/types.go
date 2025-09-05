package handlers

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/dtomasi/k1s/core/client"
)

// GetRequest represents a request to get one or more resources.
type GetRequest struct {
	// ResourceType specifies the GVK of the resource to get
	ResourceType schema.GroupVersionKind
	// Key identifies a specific resource (for single resource gets)
	Key *client.ObjectKey
	// Options for list operations (when Key is nil)
	ListOptions []client.ListOption
	// OutputOptions control how the response should be formatted
	OutputOptions *OutputOptions
}

// GetResponse contains the result of a get operation.
type GetResponse struct {
	// Object contains the single resource (when Key was provided)
	Object client.Object
	// Objects contains multiple resources (when listing)
	Objects []client.Object
	// IsCollection indicates if this represents multiple objects
	IsCollection bool
}

// CreateRequest represents a request to create a resource.
type CreateRequest struct {
	// Object is the resource to create
	Object client.Object
	// Options for the create operation
	Options []client.CreateOption
	// OutputOptions control how the response should be formatted
	OutputOptions *OutputOptions
}

// CreateResponse contains the result of a create operation.
type CreateResponse struct {
	// Object is the created resource
	Object client.Object
	// Created indicates if the resource was actually created
	Created bool
}

// ApplyRequest represents a request to apply a resource declaratively.
type ApplyRequest struct {
	// Object is the resource to apply
	Object client.Object
	// Force indicates whether to force apply
	Force bool
	// FieldManager specifies the field manager for server-side apply
	FieldManager string
	// OutputOptions control how the response should be formatted
	OutputOptions *OutputOptions
}

// ApplyResponse contains the result of an apply operation.
type ApplyResponse struct {
	// Object is the applied resource
	Object client.Object
	// Applied indicates the operation performed (created, updated, unchanged)
	Applied ApplyOperation
}

// ApplyOperation indicates what operation was performed during apply.
type ApplyOperation string

const (
	// ApplyOperationCreated indicates the resource was created
	ApplyOperationCreated ApplyOperation = "created"
	// ApplyOperationUpdated indicates the resource was updated
	ApplyOperationUpdated ApplyOperation = "updated"
	// ApplyOperationUnchanged indicates the resource was unchanged
	ApplyOperationUnchanged ApplyOperation = "unchanged"
)

// DeleteRequest represents a request to delete a resource.
type DeleteRequest struct {
	// ResourceType specifies the GVK of resources to delete
	ResourceType schema.GroupVersionKind
	// Key identifies a specific resource (for single resource deletes)
	Key *client.ObjectKey
	// Objects contains specific objects to delete
	Objects []client.Object
	// Options for the delete operation
	Options []client.DeleteOption
	// OutputOptions control how the response should be formatted
	OutputOptions *OutputOptions
}

// DeleteResponse contains the result of a delete operation.
type DeleteResponse struct {
	// Deleted contains the resources that were deleted
	Deleted []client.Object
}

// OutputOptions control how operation responses should be formatted.
type OutputOptions struct {
	// Format specifies the output format (table, json, yaml, name)
	Format string
	// NoHeaders indicates whether to omit headers in table output
	NoHeaders bool
	// ShowLabels indicates whether to show labels in table output
	ShowLabels bool
	// Wide indicates whether to use wide output format
	Wide bool
	// CustomColumns specifies custom column definitions
	CustomColumns []string
}

// NewOutputOptions creates OutputOptions with default values.
func NewOutputOptions() *OutputOptions {
	return &OutputOptions{
		Format:     "table",
		NoHeaders:  false,
		ShowLabels: false,
		Wide:       false,
	}
}

// ResourceInfo contains information about a resource type.
type ResourceInfo struct {
	// GVK is the GroupVersionKind of the resource
	GVK schema.GroupVersionKind
	// GVR is the GroupVersionResource of the resource
	GVR schema.GroupVersionResource
	// Namespaced indicates if the resource is namespaced
	Namespaced bool
	// Kind is the kind name
	Kind string
	// Resource is the resource name (plural)
	Resource string
	// SingularResource is the singular form of the resource name
	SingularResource string
	// ShortNames contains short names for the resource
	ShortNames []string
}

// ObjectList is a generic object list interface.
type ObjectList interface {
	runtime.Object
	// GetItems returns the items in the list
	GetItems() []runtime.Object
	// SetItems sets the items in the list
	SetItems([]runtime.Object)
}
