package client

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/storage"

	"github.com/dtomasi/k1s/core/pkg/codec"
	"github.com/dtomasi/k1s/core/pkg/defaulting"
	"github.com/dtomasi/k1s/core/pkg/registry"
	k1sstorage "github.com/dtomasi/k1s/core/pkg/storage"
	"github.com/dtomasi/k1s/core/pkg/validation"
)

// client implements the Client interface for k1s.
type client struct {
	scheme       *runtime.Scheme
	restMapper   meta.RESTMapper
	storage      k1sstorage.Interface
	validator    validation.Validator
	defaulter    defaulting.Defaulter
	registry     registry.Registry
	codecFactory *codec.CodecFactory
	statusWriter StatusWriter
}

// ClientOptions contains options for creating a new client.
type ClientOptions struct {
	Scheme       *runtime.Scheme
	RESTMapper   meta.RESTMapper
	Storage      k1sstorage.Interface
	Validator    validation.Validator
	Defaulter    defaulting.Defaulter
	Registry     registry.Registry
	CodecFactory *codec.CodecFactory
}

// NewClient creates a new k1s client with the provided options.
func NewClient(opts ClientOptions) (Client, error) {
	if opts.Scheme == nil {
		return nil, fmt.Errorf("scheme is required")
	}
	if opts.Storage == nil {
		return nil, fmt.Errorf("storage is required")
	}
	if opts.Registry == nil {
		return nil, fmt.Errorf("registry is required")
	}

	c := &client{
		scheme:       opts.Scheme,
		restMapper:   opts.RESTMapper,
		storage:      opts.Storage,
		validator:    opts.Validator,
		defaulter:    opts.Defaulter,
		registry:     opts.Registry,
		codecFactory: opts.CodecFactory,
	}

	if c.codecFactory == nil {
		c.codecFactory = codec.NewCodecFactory(c.scheme)
	}

	c.statusWriter = &statusWriter{client: c}

	return c, nil
}

// Scheme returns the scheme this client is using.
func (c *client) Scheme() *runtime.Scheme {
	return c.scheme
}

// RESTMapper returns the rest mapper this client is using.
func (c *client) RESTMapper() meta.RESTMapper {
	return c.restMapper
}

// Status returns the status writer for this client.
func (c *client) Status() StatusWriter {
	return c.statusWriter
}

// Get retrieves an object for the given object key from the k1s storage.
func (c *client) Get(ctx context.Context, key ObjectKey, obj Object, opts ...GetOption) error {
	options := &GetOptions{}
	for _, opt := range opts {
		opt.ApplyToGet(options)
	}

	gvk, err := c.getGVKForObject(obj)
	if err != nil {
		return fmt.Errorf("failed to get GVK for object: %w", err)
	}

	gvr, err := c.registry.GetGVRForGVK(gvk)
	if err != nil {
		return fmt.Errorf("failed to get GVR for GVK %s: %w", gvk, err)
	}

	storageKey := c.buildStorageKey(gvr, key)

	getOpts := storage.GetOptions{}
	if options.Raw != nil {
		getOpts.ResourceVersion = options.Raw.ResourceVersion
	}

	if err := c.storage.Get(ctx, storageKey, getOpts, obj); err != nil {
		return fmt.Errorf("failed to get object: %w", err)
	}

	return nil
}

// List retrieves a list of objects from the k1s storage.
func (c *client) List(ctx context.Context, list ObjectList, opts ...ListOption) error {
	options := &ListOptions{}
	for _, opt := range opts {
		opt.ApplyToList(options)
	}

	gvk, err := c.getGVKForObjectList(list)
	if err != nil {
		return fmt.Errorf("failed to get GVK for object list: %w", err)
	}

	gvr, err := c.registry.GetGVRForGVK(gvk)
	if err != nil {
		return fmt.Errorf("failed to get GVR for GVK %s: %w", gvk, err)
	}

	storageKey := c.buildListStorageKey(gvr, options.Namespace)

	listOpts := storage.ListOptions{
		ResourceVersion: "0",
	}
	if options.Raw != nil {
		if options.Raw.ResourceVersion != "" {
			listOpts.ResourceVersion = options.Raw.ResourceVersion
		}
		// Note: storage.ListOptions doesn't have Continue and Limit fields
		// These would be handled by the storage implementation if needed
	}

	if err := c.storage.List(ctx, storageKey, listOpts, list); err != nil {
		return fmt.Errorf("failed to list objects: %w", err)
	}

	return nil
}

// Create saves the object obj in the k1s storage.
func (c *client) Create(ctx context.Context, obj Object, opts ...CreateOption) error {
	options := &CreateOptions{}
	for _, opt := range opts {
		opt.ApplyToCreate(options)
	}

	// Apply defaults if defaulter is available
	if c.defaulter != nil {
		if err := c.defaulter.Default(ctx, obj); err != nil {
			return fmt.Errorf("failed to apply defaults: %w", err)
		}
	}

	// Validate the object if validator is available
	if c.validator != nil {
		if err := c.validator.Validate(ctx, obj); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	gvk, err := c.getGVKForObject(obj)
	if err != nil {
		return fmt.Errorf("failed to get GVK for object: %w", err)
	}

	gvr, err := c.registry.GetGVRForGVK(gvk)
	if err != nil {
		return fmt.Errorf("failed to get GVR for GVK %s: %w", gvk, err)
	}

	key := ObjectKeyFromObject(obj)
	storageKey := c.buildStorageKey(gvr, key)

	// Ensure the object has proper metadata
	c.ensureObjectMetadata(obj)

	if err := c.storage.Create(ctx, storageKey, obj, obj, 0); err != nil {
		return fmt.Errorf("failed to create object: %w", err)
	}

	return nil
}

// Update updates the given obj in the k1s storage.
func (c *client) Update(ctx context.Context, obj Object, opts ...UpdateOption) error {
	options := &UpdateOptions{}
	for _, opt := range opts {
		opt.ApplyToUpdate(options)
	}

	// Get the existing object for validation
	key := ObjectKeyFromObject(obj)
	gvk, err := c.getGVKForObject(obj)
	if err != nil {
		return fmt.Errorf("failed to get GVK for object: %w", err)
	}

	// Create a new object of the same type to get the existing version
	existing, err := c.scheme.New(gvk)
	if err != nil {
		return fmt.Errorf("failed to create object for existing version: %w", err)
	}
	existingObj := existing.(Object)

	// Get the existing object
	if err := c.Get(ctx, key, existingObj); err != nil {
		return fmt.Errorf("failed to get existing object for update: %w", err)
	}

	// Apply defaults if defaulter is available
	if c.defaulter != nil {
		if err := c.defaulter.Default(ctx, obj); err != nil {
			return fmt.Errorf("failed to apply defaults: %w", err)
		}
	}

	// Validate the update if validator is available
	if c.validator != nil {
		if err := c.validator.ValidateUpdate(ctx, obj, existingObj); err != nil {
			return fmt.Errorf("update validation failed: %w", err)
		}
	}

	gvr, err := c.registry.GetGVRForGVK(gvk)
	if err != nil {
		return fmt.Errorf("failed to get GVR for GVK %s: %w", gvk, err)
	}

	storageKey := c.buildStorageKey(gvr, key)

	// Update resource version
	obj.SetResourceVersion(existingObj.GetResourceVersion())
	obj.SetGeneration(existingObj.GetGeneration() + 1)

	// For update operations, we delete the old and create the new
	existingRV := existingObj.GetResourceVersion()
	preconditions := &storage.Preconditions{
		ResourceVersion: &existingRV,
	}

	if err := c.storage.Delete(ctx, storageKey, obj, preconditions, nil, existingObj); err != nil {
		return fmt.Errorf("failed to delete existing object during update: %w", err)
	}

	if err := c.storage.Create(ctx, storageKey, obj, obj, 0); err != nil {
		return fmt.Errorf("failed to create updated object: %w", err)
	}

	return nil
}

// Delete deletes the given obj from k1s storage.
func (c *client) Delete(ctx context.Context, obj Object, opts ...DeleteOption) error {
	options := &DeleteOptions{}
	for _, opt := range opts {
		opt.ApplyToDelete(options)
	}

	// Validate deletion if validator is available
	if c.validator != nil {
		if err := c.validator.ValidateDelete(ctx, obj); err != nil {
			return fmt.Errorf("delete validation failed: %w", err)
		}
	}

	gvk, err := c.getGVKForObject(obj)
	if err != nil {
		return fmt.Errorf("failed to get GVK for object: %w", err)
	}

	gvr, err := c.registry.GetGVRForGVK(gvk)
	if err != nil {
		return fmt.Errorf("failed to get GVR for GVK %s: %w", gvk, err)
	}

	key := ObjectKeyFromObject(obj)
	storageKey := c.buildStorageKey(gvr, key)

	var preconditions *storage.Preconditions
	if options.Preconditions != nil {
		preconditions = &storage.Preconditions{
			UID:             options.Preconditions.UID,
			ResourceVersion: options.Preconditions.ResourceVersion,
		}
	}

	if err := c.storage.Delete(ctx, storageKey, obj, preconditions, nil, nil); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// Patch patches the given obj in k1s storage.
func (c *client) Patch(ctx context.Context, obj Object, patch Patch, opts ...PatchOption) error {
	options := &PatchOptions{}
	for _, opt := range opts {
		opt.ApplyToPatch(options)
	}

	key := ObjectKeyFromObject(obj)
	gvk, err := c.getGVKForObject(obj)
	if err != nil {
		return fmt.Errorf("failed to get GVK for object: %w", err)
	}

	// Get the existing object
	existing, err := c.scheme.New(gvk)
	if err != nil {
		return fmt.Errorf("failed to create object for existing version: %w", err)
	}
	existingObj := existing.(Object)

	if err := c.Get(ctx, key, existingObj); err != nil {
		return fmt.Errorf("failed to get existing object for patch: %w", err)
	}

	// Get patch data
	patchData, err := patch.Data(obj)
	if err != nil {
		return fmt.Errorf("failed to get patch data: %w", err)
	}

	// Apply the patch based on patch type
	var patchedObj Object
	switch patch.Type() {
	case types.StrategicMergePatchType:
		patchedObj, err = c.applyStrategicMergePatch(existingObj, patchData)
	case types.MergePatchType:
		patchedObj, err = c.applyMergePatch(existingObj, patchData)
	case types.JSONPatchType:
		patchedObj, err = c.applyJSONPatch(existingObj, patchData)
	case types.ApplyPatchType:
		patchedObj, err = c.applyServerSideApply(existingObj, obj, options)
	default:
		return fmt.Errorf("unsupported patch type: %s", patch.Type())
	}

	if err != nil {
		return fmt.Errorf("failed to apply patch: %w", err)
	}

	// Update the object with patched values
	return c.Update(ctx, patchedObj)
}

// Helper methods

// getGVKForObject returns the GroupVersionKind for an object.
func (c *client) getGVKForObject(obj Object) (schema.GroupVersionKind, error) {
	gvks, _, err := c.scheme.ObjectKinds(obj)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	if len(gvks) == 0 {
		return schema.GroupVersionKind{}, fmt.Errorf("no GroupVersionKind found for object")
	}
	return gvks[0], nil
}

// getGVKForObjectList returns the GroupVersionKind for an object list.
func (c *client) getGVKForObjectList(list ObjectList) (schema.GroupVersionKind, error) {
	gvks, _, err := c.scheme.ObjectKinds(list)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	if len(gvks) == 0 {
		return schema.GroupVersionKind{}, fmt.Errorf("no GroupVersionKind found for object list")
	}

	// For lists, convert from "ItemList" to "Item"
	gvk := gvks[0]
	switch gvk.Kind {
	case "ItemList":
		gvk.Kind = "Item"
	case "CategoryList":
		gvk.Kind = "Category"
	default:
		// Add more list type mappings as needed
	}

	return gvk, nil
}

// buildStorageKey creates a storage key for an object.
func (c *client) buildStorageKey(gvr schema.GroupVersionResource, key ObjectKey) string {
	if key.Namespace != "" {
		return fmt.Sprintf("/%s/%s/%s/%s", gvr.Group, gvr.Version, gvr.Resource, key.Namespace+"/"+key.Name)
	}
	return fmt.Sprintf("/%s/%s/%s/%s", gvr.Group, gvr.Version, gvr.Resource, key.Name)
}

// buildListStorageKey creates a storage key for listing objects.
func (c *client) buildListStorageKey(gvr schema.GroupVersionResource, namespace string) string {
	if namespace != "" {
		return fmt.Sprintf("/%s/%s/%s/%s/", gvr.Group, gvr.Version, gvr.Resource, namespace)
	}
	return fmt.Sprintf("/%s/%s/%s/", gvr.Group, gvr.Version, gvr.Resource)
}

// ensureObjectMetadata ensures that the object has proper metadata set.
func (c *client) ensureObjectMetadata(obj Object) {
	if obj.GetUID() == "" {
		obj.SetUID(types.UID(fmt.Sprintf("%s-%d", obj.GetName(), obj.GetGeneration())))
	}
	if obj.GetResourceVersion() == "" {
		obj.SetResourceVersion("1")
	}
	if obj.GetCreationTimestamp().Time.IsZero() {
		now := metav1.Now()
		obj.SetCreationTimestamp(now)
	}
}

// Patch application methods - simplified implementations

func (c *client) applyStrategicMergePatch(existing Object, patchData []byte) (Object, error) {
	// For now, use JSON merge patch as a fallback
	return c.applyMergePatch(existing, patchData)
}

func (c *client) applyMergePatch(existing Object, patchData []byte) (Object, error) {
	// This is a simplified implementation
	// In a real implementation, you'd properly merge the patch with the existing object
	// For now, just return the existing object
	// TODO: Implement proper merge patch logic
	return existing, nil
}

func (c *client) applyJSONPatch(existing Object, patchData []byte) (Object, error) {
	// This is a simplified implementation
	// In a real implementation, you'd apply JSON patch operations
	return existing, nil
}

func (c *client) applyServerSideApply(existing Object, desired Object, options *PatchOptions) (Object, error) {
	// For server-side apply, we typically replace the object with the desired state
	// while preserving certain fields managed by other field managers
	return desired, nil
}
