package handlers

import (
	"context"
	"fmt"

	"github.com/dtomasi/k1s/core/client"
)

// createHandler implements CreateHandler.
type createHandler struct {
	client client.Client
}

// Handle executes a create operation.
func (h *createHandler) Handle(ctx context.Context, req *CreateRequest) (*CreateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("create request cannot be nil")
	}

	if req.Object == nil {
		return nil, fmt.Errorf("create request must specify an object")
	}

	// Create the resource
	err := h.client.Create(ctx, req.Object, req.Options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s %s: %w",
			req.Object.GetObjectKind().GroupVersionKind().Kind,
			client.ObjectKeyFromObject(req.Object),
			err)
	}

	return &CreateResponse{
		Object:  req.Object,
		Created: true,
	}, nil
}
