package storage

import (
	"context"
	"testing"
)

func TestPebbleStorage_Get(t *testing.T) {
	storage := NewPebbleStorage("/tmp/test-pebble")
	ctx := context.Background()

	// Test getting non-existent object
	obj, err := storage.Get(ctx, "test-key")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if obj != nil {
		t.Errorf("Expected nil object, got %v", obj)
	}
}
