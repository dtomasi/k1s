package storage_test

import (
	"context"
	"fmt"
	"sync"

	"github.com/dtomasi/k1s/core/storage"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	k8sstorage "k8s.io/apiserver/pkg/storage"
)

// NewMockStorage creates an in-memory storage implementation for testing purposes.
// This is a simple mock that stores objects in memory and should only be used for tests.
func NewMockStorage() storage.Interface {
	return &mockStorage{
		objects:  make(map[string]runtime.Object),
		watchers: make(map[string][]chan watch.Event),
		nextRV:   1,
	}
}

type mockStorage struct {
	mu       sync.RWMutex
	objects  map[string]runtime.Object
	watchers map[string][]chan watch.Event
	nextRV   int64
}

func (m *mockStorage) Versioner() k8sstorage.Versioner {
	return storage.SimpleVersioner{}
}

func (m *mockStorage) Create(ctx context.Context, key string, obj, out runtime.Object, ttl uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.objects[key]; exists {
		return k8sstorage.NewKeyExistsError(key, 0)
	}

	// Set resource version
	if accessor, err := meta.Accessor(obj); err == nil {
		accessor.SetResourceVersion(fmt.Sprintf("%d", m.nextRV))
		m.nextRV++
	}

	// Deep copy and store
	objCopy := obj.DeepCopyObject()
	m.objects[key] = objCopy

	// Copy to out parameter if provided
	if out != nil {
		if accessor, err := meta.Accessor(objCopy); err == nil {
			if outAccessor, err := meta.Accessor(out); err == nil {
				outAccessor.SetName(accessor.GetName())
				outAccessor.SetNamespace(accessor.GetNamespace())
				outAccessor.SetResourceVersion(accessor.GetResourceVersion())
				outAccessor.SetUID(accessor.GetUID())
			}
		}
	}

	return nil
}

func (m *mockStorage) Get(ctx context.Context, key string, opts k8sstorage.GetOptions, objPtr runtime.Object) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stored, exists := m.objects[key]
	if !exists {
		if opts.IgnoreNotFound {
			return nil
		}
		return k8sstorage.NewKeyNotFoundError(key, 0)
	}

	// Simple copy for testing
	if accessor, err := meta.Accessor(stored); err == nil {
		if objAccessor, err := meta.Accessor(objPtr); err == nil {
			objAccessor.SetName(accessor.GetName())
			objAccessor.SetNamespace(accessor.GetNamespace())
			objAccessor.SetResourceVersion(accessor.GetResourceVersion())
			objAccessor.SetUID(accessor.GetUID())
			objAccessor.SetLabels(accessor.GetLabels())
			objAccessor.SetAnnotations(accessor.GetAnnotations())
		}
	}

	return nil
}

func (m *mockStorage) List(ctx context.Context, key string, opts k8sstorage.ListOptions, listObj runtime.Object) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var items []runtime.Object
	for objKey, obj := range m.objects {
		if len(key) == 0 || objKey == key || (len(objKey) > len(key) && objKey[:len(key)] == key) {
			items = append(items, obj.DeepCopyObject())
		}
	}

	if err := meta.SetList(listObj, items); err != nil {
		return fmt.Errorf("failed to set list items: %w", err)
	}

	return nil
}

func (m *mockStorage) Delete(ctx context.Context, key string, out runtime.Object, preconditions *k8sstorage.Preconditions,
	validateDeletion k8sstorage.ValidateObjectFunc, cachedExistingObject runtime.Object) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	obj, exists := m.objects[key]
	if !exists {
		return k8sstorage.NewKeyNotFoundError(key, 0)
	}

	delete(m.objects, key)

	if out != nil {
		if accessor, err := meta.Accessor(obj); err == nil {
			if outAccessor, err := meta.Accessor(out); err == nil {
				outAccessor.SetName(accessor.GetName())
				outAccessor.SetNamespace(accessor.GetNamespace())
				outAccessor.SetResourceVersion(accessor.GetResourceVersion())
				outAccessor.SetUID(accessor.GetUID())
			}
		}
	}

	return nil
}

func (m *mockStorage) Watch(ctx context.Context, key string, opts k8sstorage.ListOptions) (watch.Interface, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan watch.Event, 100)

	if m.watchers[key] == nil {
		m.watchers[key] = make([]chan watch.Event, 0)
	}
	m.watchers[key] = append(m.watchers[key], ch)

	return &mockWatcher{
		ch:      ch,
		storage: m,
		key:     key,
	}, nil
}

func (m *mockStorage) GuaranteedUpdate(ctx context.Context, key string, destination runtime.Object, ignoreNotFound bool,
	preconditions *k8sstorage.Preconditions, tryUpdate k8sstorage.UpdateFunc, cachedExistingObject runtime.Object) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simple implementation for testing
	existing, exists := m.objects[key]
	if !exists && !ignoreNotFound {
		return k8sstorage.NewKeyNotFoundError(key, 0)
	}

	var currentObj runtime.Object
	if exists {
		currentObj = existing.DeepCopyObject()
	}

	updated, _, err := tryUpdate(currentObj, k8sstorage.ResponseMeta{})
	if err != nil {
		return err
	}

	if updated != nil {
		// Set resource version
		if accessor, err := meta.Accessor(updated); err == nil {
			accessor.SetResourceVersion(fmt.Sprintf("%d", m.nextRV))
			m.nextRV++
		}

		m.objects[key] = updated.DeepCopyObject()

		// Copy to destination
		if accessor, err := meta.Accessor(updated); err == nil {
			if destAccessor, err := meta.Accessor(destination); err == nil {
				destAccessor.SetName(accessor.GetName())
				destAccessor.SetNamespace(accessor.GetNamespace())
				destAccessor.SetResourceVersion(accessor.GetResourceVersion())
				destAccessor.SetUID(accessor.GetUID())
			}
		}
	}

	return nil
}

func (m *mockStorage) RequestWatchProgress(ctx context.Context) error {
	// No-op for testing
	return nil
}

func (m *mockStorage) RequestProgress(ctx context.Context) error {
	// No-op for testing
	return nil
}

type mockWatcher struct {
	ch      chan watch.Event
	storage *mockStorage
	key     string
	stopped bool
	mu      sync.Mutex
}

func (w *mockWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return
	}
	w.stopped = true

	w.storage.removeWatcher(w.key, w.ch)
	close(w.ch)
}

func (w *mockWatcher) ResultChan() <-chan watch.Event {
	return w.ch
}

func (m *mockStorage) removeWatcher(key string, ch chan watch.Event) {
	watchers := m.watchers[key]
	for i, watcher := range watchers {
		if watcher == ch {
			m.watchers[key] = append(watchers[:i], watchers[i+1:]...)
			break
		}
	}
}
