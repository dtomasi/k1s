package client

import (
	"context"
	"fmt"
	"reflect"

	"k8s.io/apiserver/pkg/storage"
)

// statusWriter implements the StatusWriter interface.
type statusWriter struct {
	client *client
}

// Update updates the status subresource for the given obj.
func (sw *statusWriter) Update(ctx context.Context, obj Object, opts ...UpdateOption) error {
	options := &UpdateOptions{}
	for _, opt := range opts {
		opt.ApplyToUpdate(options)
	}

	// Get the existing object to merge status with
	key := ObjectKeyFromObject(obj)
	gvk, err := sw.client.getGVKForObject(obj)
	if err != nil {
		return fmt.Errorf("failed to get GVK for object: %w", err)
	}

	existing, err := sw.client.scheme.New(gvk)
	if err != nil {
		return fmt.Errorf("failed to create object for existing version: %w", err)
	}
	existingObj := existing.(Object)

	// Get the current object
	if err := sw.client.Get(ctx, key, existingObj); err != nil {
		return fmt.Errorf("failed to get existing object for status update: %w", err)
	}

	// Update only the status field while preserving other fields
	if err := sw.updateObjectStatus(existingObj, obj); err != nil {
		return fmt.Errorf("failed to update status field: %w", err)
	}

	// Validate the status update if validator is available
	if sw.client.validator != nil {
		if err := sw.client.validator.ValidateUpdate(ctx, existingObj, existingObj); err != nil {
			return fmt.Errorf("status update validation failed: %w", err)
		}
	}

	gvr, err := sw.client.registry.GetGVRForGVK(gvk)
	if err != nil {
		return fmt.Errorf("failed to get GVR for GVK %s: %w", gvk, err)
	}

	storageKey := sw.client.buildStorageKey(gvr, key)

	// Increment the generation for status updates
	existingObj.SetGeneration(existingObj.GetGeneration() + 1)

	// For status updates, we need to update the existing object instead of creating a new one
	// First delete the existing one, then create the updated version
	existingRV := existingObj.GetResourceVersion()
	preconditions := &storage.Preconditions{
		ResourceVersion: &existingRV,
	}

	if err := sw.client.storage.Delete(ctx, storageKey, existingObj, preconditions, nil, existingObj); err != nil {
		return fmt.Errorf("failed to delete existing object during status update: %w", err)
	}

	if err := sw.client.storage.Create(ctx, storageKey, existingObj, existingObj, 0); err != nil {
		return fmt.Errorf("failed to update object status: %w", err)
	}

	// Copy the updated object back to the input object
	sw.copyObject(existingObj, obj)

	return nil
}

// Patch patches the status subresource for the given obj.
func (sw *statusWriter) Patch(ctx context.Context, obj Object, patch Patch, opts ...PatchOption) error {
	options := &PatchOptions{}
	for _, opt := range opts {
		opt.ApplyToPatch(options)
	}

	// Get the existing object
	key := ObjectKeyFromObject(obj)
	gvk, err := sw.client.getGVKForObject(obj)
	if err != nil {
		return fmt.Errorf("failed to get GVK for object: %w", err)
	}

	existing, err := sw.client.scheme.New(gvk)
	if err != nil {
		return fmt.Errorf("failed to create object for existing version: %w", err)
	}
	existingObj := existing.(Object)

	if err := sw.client.Get(ctx, key, existingObj); err != nil {
		return fmt.Errorf("failed to get existing object for status patch: %w", err)
	}

	// Apply the patch to the status field only
	patchedObj, err := sw.applyStatusPatch(existingObj, obj, patch)
	if err != nil {
		return fmt.Errorf("failed to apply status patch: %w", err)
	}

	// Update the object with the patched status
	return sw.Update(ctx, patchedObj)
}

// updateObjectStatus updates the status field of the target object with the status from the source object.
func (sw *statusWriter) updateObjectStatus(target, source Object) error {
	targetValue := reflect.ValueOf(target)
	sourceValue := reflect.ValueOf(source)

	// Dereference pointers
	if targetValue.Kind() == reflect.Ptr {
		targetValue = targetValue.Elem()
	}
	if sourceValue.Kind() == reflect.Ptr {
		sourceValue = sourceValue.Elem()
	}

	// Find the status field
	targetStatusField := targetValue.FieldByName("Status")
	sourceStatusField := sourceValue.FieldByName("Status")

	if !targetStatusField.IsValid() || !sourceStatusField.IsValid() {
		return fmt.Errorf("status field not found in object")
	}

	if !targetStatusField.CanSet() {
		return fmt.Errorf("cannot set status field")
	}

	// Copy the status field
	if sourceStatusField.Type() != targetStatusField.Type() {
		return fmt.Errorf("status field types do not match")
	}

	targetStatusField.Set(sourceStatusField)
	return nil
}

// applyStatusPatch applies a patch to only the status field of an object.
func (sw *statusWriter) applyStatusPatch(existing Object, obj Object, _ Patch) (Object, error) {
	// For status patches, we typically only care about updating the status field
	// This is a simplified implementation that just updates the status field
	if err := sw.updateObjectStatus(existing, obj); err != nil {
		return nil, err
	}
	return existing, nil
}

// copyObject copies fields from source to target object.
func (sw *statusWriter) copyObject(source, target Object) {
	sourceValue := reflect.ValueOf(source)
	targetValue := reflect.ValueOf(target)

	// Dereference pointers
	if sourceValue.Kind() == reflect.Ptr {
		sourceValue = sourceValue.Elem()
	}
	if targetValue.Kind() == reflect.Ptr {
		targetValue = targetValue.Elem()
	}

	// Copy all fields
	if sourceValue.Type() == targetValue.Type() && targetValue.CanSet() {
		targetValue.Set(sourceValue)
	}
}
