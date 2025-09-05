package handlers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/dtomasi/k1s/core/client"
)

// deleteHandler implements DeleteHandler.
type deleteHandler struct {
	client client.Client
}

// Handle executes a delete operation.
func (h *deleteHandler) Handle(ctx context.Context, req *DeleteRequest) (*DeleteResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete request cannot be nil")
	}

	var deleted []client.Object

	// If specific objects are provided, delete them
	if len(req.Objects) > 0 {
		for _, obj := range req.Objects {
			err := h.client.Delete(ctx, obj, req.Options...)
			if err != nil {
				// Don't fail the entire operation if one object fails to delete
				// This matches kubectl behavior
				continue
			}
			deleted = append(deleted, obj)
		}
		return &DeleteResponse{Deleted: deleted}, nil
	}

	// If a specific key is provided, delete that resource
	if req.Key != nil {
		obj, err := h.createObjectForGVK(req.ResourceType)
		if err != nil {
			return nil, fmt.Errorf("failed to create object for GVK %s: %w", req.ResourceType, err)
		}

		// Set the object's namespace and name
		obj.SetNamespace(req.Key.Namespace)
		obj.SetName(req.Key.Name)

		err = h.client.Delete(ctx, obj, req.Options...)
		if err != nil {
			return nil, fmt.Errorf("failed to delete %s %s: %w",
				req.ResourceType.Kind, req.Key, err)
		}

		deleted = append(deleted, obj)
		return &DeleteResponse{Deleted: deleted}, nil
	}

	return nil, fmt.Errorf("delete request must specify either objects or a key")
}

// createObjectForGVK creates an empty object instance for the given GVK.
func (h *deleteHandler) createObjectForGVK(gvk schema.GroupVersionKind) (client.Object, error) {
	scheme := h.client.Scheme()
	obj, err := scheme.New(gvk)
	if err != nil {
		return nil, fmt.Errorf("failed to create new object for GVK %s: %w", gvk, err)
	}

	clientObj, ok := obj.(client.Object)
	if !ok {
		return nil, fmt.Errorf("object for GVK %s does not implement client.Object", gvk)
	}

	return clientObj, nil
}
