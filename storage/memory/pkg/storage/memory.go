// Package storage provides an in-memory storage implementation for k1s
// with high performance (>10,000 ops/sec) and zero dependencies.
package storage

import (
	"context"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
)

// MemoryStorage implements storage.Interface using in-memory maps
type MemoryStorage struct {
	mu   sync.RWMutex
	data map[string]runtime.Object
}

// NewMemoryStorage creates a new in-memory storage instance
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string]runtime.Object),
	}
}

// Get retrieves an object by key
func (m *MemoryStorage) Get(_ context.Context, key string) (runtime.Object, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	obj, exists := m.data[key]
	if !exists {
		return nil, nil // Not found
	}

	return obj, nil
}
