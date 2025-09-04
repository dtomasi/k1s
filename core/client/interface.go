package client

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

// Client provides a controller-runtime compatible interface for interacting with
// Kubernetes-style objects in k1s. This interface matches the controller-runtime
// client.Client interface for maximum compatibility.
type Client interface {
	Reader
	Writer
	StatusClient

	// Scheme returns the scheme this client is using
	Scheme() *runtime.Scheme

	// RESTMapper returns the rest this client is using
	RESTMapper() meta.RESTMapper
}

// Reader knows how to read and list Kubernetes objects.
type Reader interface {
	// Get retrieves an object for the given object key from the Kubernetes cluster.
	// obj must be a struct pointer so that obj can be updated with the response
	// returned by the Server.
	Get(ctx context.Context, key ObjectKey, obj Object, opts ...GetOption) error

	// List retrieves list of objects for a given namespace and list options.
	// On a successful call, Items field in the list will be populated with the
	// result returned from the server.
	List(ctx context.Context, list ObjectList, opts ...ListOption) error
}

// Writer knows how to create, delete, and update Kubernetes objects.
type Writer interface {
	// Create saves the object obj in the Kubernetes cluster.
	Create(ctx context.Context, obj Object, opts ...CreateOption) error

	// Delete deletes the given obj from Kubernetes cluster.
	Delete(ctx context.Context, obj Object, opts ...DeleteOption) error

	// Update updates the given obj in the Kubernetes cluster.
	// obj must be a struct pointer so that obj can be updated with the content returned by the Server.
	Update(ctx context.Context, obj Object, opts ...UpdateOption) error

	// Patch patches the given obj in the Kubernetes cluster.
	// obj must be a struct pointer so that obj can be updated with the content returned by the Server.
	Patch(ctx context.Context, obj Object, patch Patch, opts ...PatchOption) error
}

// StatusClient knows how to create a client which can update status subresource
// for kubernetes objects.
type StatusClient interface {
	Status() StatusWriter
}

// StatusWriter knows how to update status subresource of a Kubernetes object.
type StatusWriter interface {
	// Update updates the fields corresponding to the status subresource for the
	// given obj. obj must be a struct pointer so that obj can be updated
	// with the content returned by the Server.
	Update(ctx context.Context, obj Object, opts ...UpdateOption) error

	// Patch patches the given object's subresource. obj must be a struct
	// pointer so that obj can be updated with the content returned by the Server.
	Patch(ctx context.Context, obj Object, patch Patch, opts ...PatchOption) error
}

// WithWatch represents a client that can perform watch operations.
type WithWatch interface {
	Client

	// Watch watches objects of the given type. Depending on the WatchOption, this may
	// be scoped to a namespace or across all namespaces.
	Watch(ctx context.Context, obj ObjectList, opts ...WatchOption) (watch.Interface, error)
}

// Object is a Kubernetes object that can be used with the Client.
type Object interface {
	runtime.Object
	metav1.Object
}

// ObjectList is a Kubernetes object that represents a list of objects.
type ObjectList interface {
	runtime.Object
	metav1.ListInterface
}

// ObjectKey identifies a Kubernetes Object.
type ObjectKey struct {
	Namespace string
	Name      string
}

// String returns the general purpose string representation
func (k ObjectKey) String() string {
	if k.Namespace == "" {
		return k.Name
	}
	return k.Namespace + "/" + k.Name
}

// Patch is a patch that can be applied to a Kubernetes object.
type Patch interface {
	// Type is the PatchType of the patch.
	Type() types.PatchType
	// Data is the raw data representing the patch.
	Data(obj Object) ([]byte, error)
}

// Option is a functional option that configures a request.
type Option interface {
	// ApplyToGet applies this configuration to the given get options.
	ApplyToGet(*GetOptions)
	// ApplyToList applies this configuration to the given list options.
	ApplyToList(*ListOptions)
	// ApplyToCreate applies this configuration to the given create options.
	ApplyToCreate(*CreateOptions)
	// ApplyToUpdate applies this configuration to the given update options.
	ApplyToUpdate(*UpdateOptions)
	// ApplyToDelete applies this configuration to the given delete options.
	ApplyToDelete(*DeleteOptions)
	// ApplyToPatch applies this configuration to the given patch options.
	ApplyToPatch(*PatchOptions)
	// ApplyToWatch applies this configuration to the given watch options.
	ApplyToWatch(*WatchOptions)
}

// GetOption is some configuration that modifies options for a get request.
type GetOption interface {
	// ApplyToGet applies this configuration to the given get options.
	ApplyToGet(*GetOptions)
}

// ListOption is some configuration that modifies options for a list request.
type ListOption interface {
	// ApplyToList applies this configuration to the given list options.
	ApplyToList(*ListOptions)
}

// CreateOption is some configuration that modifies options for a create request.
type CreateOption interface {
	// ApplyToCreate applies this configuration to the given create options.
	ApplyToCreate(*CreateOptions)
}

// UpdateOption is some configuration that modifies options for an update request.
type UpdateOption interface {
	// ApplyToUpdate applies this configuration to the given update options.
	ApplyToUpdate(*UpdateOptions)
}

// DeleteOption is some configuration that modifies options for a delete request.
type DeleteOption interface {
	// ApplyToDelete applies this configuration to the given delete options.
	ApplyToDelete(*DeleteOptions)
}

// PatchOption is some configuration that modifies options for a patch request.
type PatchOption interface {
	// ApplyToPatch applies this configuration to the given patch options.
	ApplyToPatch(*PatchOptions)
}

// WatchOption is some configuration that modifies options for a watch request.
type WatchOption interface {
	// ApplyToWatch applies this configuration to the given watch options.
	ApplyToWatch(*WatchOptions)
}

// GetOptions contains options for get requests.
type GetOptions struct {
	// Raw represents raw GetOptions, as passed to the API server.
	Raw *metav1.GetOptions
}

// ListOptions contains options for list requests.
type ListOptions struct {
	// LabelSelector is a label query over a set of resources.
	LabelSelector metav1.LabelSelector
	// FieldSelector is a field query over a set of resources.
	FieldSelector string
	// Namespace represents the namespace to list for, or empty for
	// non-namespaced objects, or to list across all namespaces.
	Namespace string
	// Raw represents raw ListOptions, as passed to the API server.
	Raw *metav1.ListOptions
}

// CreateOptions contains options for create requests.
type CreateOptions struct {
	// DryRun, when present, indicates that modifications should not be
	// persisted. An invalid or unrecognized dryRun directive will
	// result in an error response and no further processing of the request.
	DryRun []string
	// FieldManager is a name associated with the actor or entity
	// that is making these changes.
	FieldManager string
	// Raw represents raw CreateOptions, as passed to the API server.
	Raw *metav1.CreateOptions
}

// UpdateOptions contains options for update requests.
type UpdateOptions struct {
	// DryRun, when present, indicates that modifications should not be
	// persisted. An invalid or unrecognized dryRun directive will
	// result in an error response and no further processing of the request.
	DryRun []string
	// FieldManager is a name associated with the actor or entity
	// that is making these changes.
	FieldManager string
	// Raw represents raw UpdateOptions, as passed to the API server.
	Raw *metav1.UpdateOptions
}

// DeleteOptions contains options for delete requests.
type DeleteOptions struct {
	// GracePeriodSeconds is the duration in seconds before the object should be deleted.
	GracePeriodSeconds *int64
	// Preconditions must be fulfilled before a deletion is carried out.
	Preconditions *metav1.Preconditions
	// PropagationPolicy determines whether and how garbage collection will be performed.
	PropagationPolicy *metav1.DeletionPropagation
	// Raw represents raw DeleteOptions, as passed to the API server.
	Raw *metav1.DeleteOptions
}

// PatchOptions contains options for patch requests.
type PatchOptions struct {
	// DryRun, when present, indicates that modifications should not be
	// persisted. An invalid or unrecognized dryRun directive will
	// result in an error response and no further processing of the request.
	DryRun []string
	// Force is going to force Apply requests.
	Force *bool
	// FieldManager is a name associated with the actor or entity
	// that is making these changes.
	FieldManager string
	// Raw represents raw PatchOptions, as passed to the API server.
	Raw *metav1.PatchOptions
}

// WatchOptions contains options for watch requests.
type WatchOptions struct {
	// LabelSelector is a label query over a set of resources.
	LabelSelector metav1.LabelSelector
	// FieldSelector is a field query over a set of resources.
	FieldSelector string
	// Namespace represents the namespace to watch for, or empty for
	// non-namespaced objects, or to watch across all namespaces.
	Namespace string
	// Raw represents raw WatchOptions, as passed to the API server.
	Raw *metav1.ListOptions
}

// ObjectKeyFromObject returns an ObjectKey for the given object.
func ObjectKeyFromObject(obj Object) ObjectKey {
	return ObjectKey{Namespace: obj.GetNamespace(), Name: obj.GetName()}
}

// ObjectKeyFromUID returns an ObjectKey that represents the given UID.
func ObjectKeyFromUID(uid types.UID) ObjectKey {
	return ObjectKey{Name: string(uid)}
}

// MatchingLabels filters the list/delete operation on the given set of labels.
func MatchingLabels(labels map[string]string) MatchingLabelsSelector {
	return MatchingLabelsSelector{
		metav1.LabelSelector{
			MatchLabels: labels,
		},
	}
}

// MatchingLabelsSelector filters the list/delete operation on the given label selector.
type MatchingLabelsSelector struct {
	metav1.LabelSelector
}

// ApplyToList applies this selector to the given list options.
func (m MatchingLabelsSelector) ApplyToList(opts *ListOptions) {
	opts.LabelSelector = m.LabelSelector
}

// ApplyToWatch applies this selector to the given watch options.
func (m MatchingLabelsSelector) ApplyToWatch(opts *WatchOptions) {
	opts.LabelSelector = m.LabelSelector
}

// Ensure MatchingLabelsSelector implements ListOption and WatchOption
var _ ListOption = MatchingLabelsSelector{}
var _ WatchOption = MatchingLabelsSelector{}

// MatchingFields filters the list operation on the given set of fields.
func MatchingFields(fields map[string]string) MatchingFieldsSelector {
	selector := ""
	for k, v := range fields {
		if len(selector) > 0 {
			selector += ","
		}
		selector += k + "=" + v
	}
	return MatchingFieldsSelector{selector}
}

// MatchingFieldsSelector filters the list operation on the given field selector.
type MatchingFieldsSelector struct {
	Selector string
}

// ApplyToList applies this selector to the given list options.
func (m MatchingFieldsSelector) ApplyToList(opts *ListOptions) {
	opts.FieldSelector = m.Selector
}

// ApplyToWatch applies this selector to the given watch options.
func (m MatchingFieldsSelector) ApplyToWatch(opts *WatchOptions) {
	opts.FieldSelector = m.Selector
}

// Ensure MatchingFieldsSelector implements ListOption and WatchOption
var _ ListOption = MatchingFieldsSelector{}
var _ WatchOption = MatchingFieldsSelector{}

// InNamespace restricts the list to the given namespace.
func InNamespace(namespace string) InNamespaceSelector {
	return InNamespaceSelector{namespace}
}

// InNamespaceSelector filters the list to the given namespace.
type InNamespaceSelector struct {
	Namespace string
}

// ApplyToList applies this selector to the given list options.
func (m InNamespaceSelector) ApplyToList(opts *ListOptions) {
	opts.Namespace = m.Namespace
}

// ApplyToWatch applies this selector to the given watch options.
func (m InNamespaceSelector) ApplyToWatch(opts *WatchOptions) {
	opts.Namespace = m.Namespace
}

// Ensure InNamespaceSelector implements ListOption and WatchOption
var _ ListOption = InNamespaceSelector{}
var _ WatchOption = InNamespaceSelector{}

// IgnoreNotFound returns nil on NotFound errors.
func IgnoreNotFound(err error) error {
	if meta.IsNoMatchError(err) {
		return nil
	}
	// Note: We don't have access to apierrors here, so we check the error message
	// This is a simplified implementation that may need refinement
	if err != nil && err.Error() == "not found" {
		return nil
	}
	return err
}
