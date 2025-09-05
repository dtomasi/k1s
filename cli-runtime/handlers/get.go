package handlers

import (
	"context"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/dtomasi/k1s/core/client"
)

// getHandler implements GetHandler.
type getHandler struct {
	client client.Client
}

// Handle executes a get operation.
func (h *getHandler) Handle(ctx context.Context, req *GetRequest) (*GetResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get request cannot be nil")
	}

	// If a specific key is provided, get a single resource
	if req.Key != nil {
		return h.getSingle(ctx, req)
	}

	// Otherwise, list resources
	return h.getList(ctx, req)
}

// getSingle retrieves a single resource by key.
func (h *getHandler) getSingle(ctx context.Context, req *GetRequest) (*GetResponse, error) {
	// Create an instance of the resource type
	obj, err := h.createObjectForGVK(req.ResourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to create object for GVK %s: %w", req.ResourceType, err)
	}

	// Get the resource
	err = h.client.Get(ctx, *req.Key, obj)
	if err != nil {
		return nil, fmt.Errorf("failed to get %s %s: %w", req.ResourceType.Kind, req.Key, err)
	}

	return &GetResponse{
		Object:       obj,
		IsCollection: false,
	}, nil
}

// getList retrieves multiple resources.
func (h *getHandler) getList(ctx context.Context, req *GetRequest) (*GetResponse, error) {
	// Create a list instance for the resource type
	list, err := h.createListForGVK(req.ResourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to create list for GVK %s: %w", req.ResourceType, err)
	}

	// List the resources
	err = h.client.List(ctx, list, req.ListOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to list %s: %w", req.ResourceType.Kind, err)
	}

	// Extract objects from the list
	objects, err := h.extractObjects(list)
	if err != nil {
		return nil, fmt.Errorf("failed to extract objects from list: %w", err)
	}

	return &GetResponse{
		Objects:      objects,
		IsCollection: true,
	}, nil
}

// createObjectForGVK creates an empty object instance for the given GVK.
func (h *getHandler) createObjectForGVK(gvk schema.GroupVersionKind) (client.Object, error) {
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

// createListForGVK creates an empty list instance for the given GVK.
func (h *getHandler) createListForGVK(gvk schema.GroupVersionKind) (client.ObjectList, error) {
	scheme := h.client.Scheme()

	// Convert the GVK to list GVK
	listGVK := gvk
	listGVK.Kind += "List"

	obj, err := scheme.New(listGVK)
	if err != nil {
		return nil, fmt.Errorf("failed to create new list for GVK %s: %w", listGVK, err)
	}

	list, ok := obj.(client.ObjectList)
	if !ok {
		return nil, fmt.Errorf("object for GVK %s does not implement client.ObjectList", listGVK)
	}

	return list, nil
}

// extractObjects extracts individual objects from a list.
func (h *getHandler) extractObjects(list client.ObjectList) ([]client.Object, error) {
	// Use reflection to get items from the list
	listValue := reflect.ValueOf(list)
	if listValue.Kind() == reflect.Ptr {
		listValue = listValue.Elem()
	}

	itemsField := listValue.FieldByName("Items")
	if !itemsField.IsValid() {
		return nil, fmt.Errorf("list does not have an Items field")
	}

	itemsSlice := itemsField.Interface()
	items, ok := itemsSlice.([]runtime.Object)
	if ok {
		// Convert []runtime.Object to []client.Object
		objects := make([]client.Object, 0, len(items))
		for _, item := range items {
			if clientObj, ok := item.(client.Object); ok {
				objects = append(objects, clientObj)
			}
		}
		return objects, nil
	}

	// If it's not []runtime.Object, try to iterate through the slice
	if itemsField.Kind() == reflect.Slice {
		objects := make([]client.Object, 0, itemsField.Len())
		for i := 0; i < itemsField.Len(); i++ {
			item := itemsField.Index(i).Interface()
			if clientObj, ok := item.(client.Object); ok {
				objects = append(objects, clientObj)
			}
		}
		return objects, nil
	}

	return nil, fmt.Errorf("unable to extract objects from list")
}
