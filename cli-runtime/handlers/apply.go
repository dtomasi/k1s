package handlers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/dtomasi/k1s/core/client"
)

// applyHandler implements ApplyHandler.
type applyHandler struct {
	client client.Client
}

// Handle executes an apply operation.
func (h *applyHandler) Handle(ctx context.Context, req *ApplyRequest) (*ApplyResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("apply request cannot be nil")
	}

	if req.Object == nil {
		return nil, fmt.Errorf("apply request must specify an object")
	}

	// Set default field manager if not provided
	fieldManager := req.FieldManager
	if fieldManager == "" {
		fieldManager = "k1s-cli"
	}

	// Try to perform server-side apply using patch
	patch := &serverSideApplyPatch{
		Object:       req.Object,
		FieldManager: fieldManager,
		Force:        req.Force,
	}

	// Clone the object to preserve the original
	original := req.Object.DeepCopyObject().(client.Object)

	// Try to patch (apply) the resource
	err := h.client.Patch(ctx, req.Object, patch)

	if err != nil {
		if errors.IsNotFound(err) {
			// Resource doesn't exist, create it
			createErr := h.client.Create(ctx, req.Object)
			if createErr != nil {
				return nil, fmt.Errorf("failed to create %s %s: %w",
					req.Object.GetObjectKind().GroupVersionKind().Kind,
					client.ObjectKeyFromObject(req.Object),
					createErr)
			}
			return &ApplyResponse{
				Object:  req.Object,
				Applied: ApplyOperationCreated,
			}, nil
		}
		return nil, fmt.Errorf("failed to apply %s %s: %w",
			req.Object.GetObjectKind().GroupVersionKind().Kind,
			client.ObjectKeyFromObject(req.Object),
			err)
	}

	// Determine if the resource was actually changed
	applied := ApplyOperationUpdated
	if h.objectsEqual(original, req.Object) {
		applied = ApplyOperationUnchanged
	}

	return &ApplyResponse{
		Object:  req.Object,
		Applied: applied,
	}, nil
}

// objectsEqual compares two objects for equality (simplified implementation).
func (h *applyHandler) objectsEqual(obj1, obj2 client.Object) bool {
	// Simple comparison based on resource version
	// In a real implementation, this might be more sophisticated
	return obj1.GetResourceVersion() == obj2.GetResourceVersion()
}

// serverSideApplyPatch implements client.Patch for server-side apply.
type serverSideApplyPatch struct {
	Object       client.Object
	FieldManager string
	Force        bool
}

// Type returns the patch type.
func (p *serverSideApplyPatch) Type() types.PatchType {
	return types.ApplyPatchType
}

// Data returns the patch data.
func (p *serverSideApplyPatch) Data(obj client.Object) ([]byte, error) {
	// For server-side apply, we serialize the entire object
	scheme := obj.GetObjectKind().GroupVersionKind()

	// TODO: Implement proper serialization to JSON
	// For now, return a simple placeholder
	return []byte(fmt.Sprintf(`{"apiVersion":"%s/%s","kind":"%s","metadata":{"name":"%s"}}`,
		scheme.Group, scheme.Version, scheme.Kind, obj.GetName())), nil
}
