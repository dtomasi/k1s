// Package handlers provides operation handlers for CLI operations in k1s.
// These handlers implement kubectl-compatible patterns and provide the building
// blocks for CLI applications without directly creating cobra commands.
package handlers

import (
	"context"

	"github.com/dtomasi/k1s/core/client"
)

// GetHandler handles GET operations for resources.
type GetHandler interface {
	// Handle executes a get operation based on the provided request
	Handle(ctx context.Context, req *GetRequest) (*GetResponse, error)
}

// CreateHandler handles CREATE operations for resources.
type CreateHandler interface {
	// Handle executes a create operation based on the provided request
	Handle(ctx context.Context, req *CreateRequest) (*CreateResponse, error)
}

// ApplyHandler handles APPLY operations for resources (declarative management).
type ApplyHandler interface {
	// Handle executes an apply operation based on the provided request
	Handle(ctx context.Context, req *ApplyRequest) (*ApplyResponse, error)
}

// DeleteHandler handles DELETE operations for resources.
type DeleteHandler interface {
	// Handle executes a delete operation based on the provided request
	Handle(ctx context.Context, req *DeleteRequest) (*DeleteResponse, error)
}

// HandlerFactory creates handlers with a given client.
type HandlerFactory struct {
	client client.Client
}

// NewHandlerFactory creates a new handler factory with the given client.
func NewHandlerFactory(client client.Client) *HandlerFactory {
	return &HandlerFactory{client: client}
}

// Get creates a new GetHandler.
func (f *HandlerFactory) Get() GetHandler {
	return &getHandler{client: f.client}
}

// Create creates a new CreateHandler.
func (f *HandlerFactory) Create() CreateHandler {
	return &createHandler{client: f.client}
}

// Apply creates a new ApplyHandler.
func (f *HandlerFactory) Apply() ApplyHandler {
	return &applyHandler{client: f.client}
}

// Delete creates a new DeleteHandler.
func (f *HandlerFactory) Delete() DeleteHandler {
	return &deleteHandler{client: f.client}
}
