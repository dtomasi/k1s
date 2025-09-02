package storage

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestMemoryStorage_Get(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Test getting non-existent object
	obj, err := storage.Get(ctx, "test-key")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if obj != nil {
		t.Errorf("Expected nil object, got %v", obj)
	}

	// Test with object in storage
	testObj := &unstructured.Unstructured{}
	testObj.SetName("test-object")
	storage.data["test-key"] = testObj

	obj, err = storage.Get(ctx, "test-key")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if obj == nil {
		t.Errorf("Expected object, got nil")
	}
}
