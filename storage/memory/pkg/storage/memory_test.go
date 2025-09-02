package storage

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage"

	k1sstorage "github.com/dtomasi/k1s/core/pkg/storage"
)

// TestObject is a simple object for testing
type TestObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Data              string `json:"data,omitempty"`
}

func (t *TestObject) DeepCopyObject() runtime.Object {
	if t == nil {
		return nil
	}
	out := new(TestObject)
	out.TypeMeta = t.TypeMeta
	out.Data = t.Data
	out.ObjectMeta = *t.ObjectMeta.DeepCopy() //nolint:staticcheck // QF1008: embedded field access needed for proper DeepCopy
	return out
}

func TestMemoryStorage_BasicOperations(t *testing.T) {
	ctx := context.Background()
	config := k1sstorage.Config{}
	s := NewMemoryStorage(config)
	defer func() { _ = s.Close() }()

	// Test Create
	obj := &TestObject{
		ObjectMeta: metav1.ObjectMeta{Name: "test-obj"},
		Data:       "test data",
	}

	err := s.Create(ctx, "test-key", obj, nil, 0)
	if err != nil {
		t.Errorf("Create failed: %v", err)
	}

	// Test duplicate create should fail
	err = s.Create(ctx, "test-key", obj, nil, 0)
	if err == nil {
		t.Error("Expected error when creating duplicate key")
	}

	// Test Get
	var retrieved TestObject
	err = s.Get(ctx, "test-key", storage.GetOptions{}, &retrieved)
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}

	// Test Get non-existing
	err = s.Get(ctx, "non-existing", storage.GetOptions{}, &retrieved)
	if err == nil {
		t.Error("Expected error when getting non-existing key")
	}

	// Test Get with IgnoreNotFound
	err = s.Get(ctx, "non-existing", storage.GetOptions{IgnoreNotFound: true}, &retrieved)
	if err != nil {
		t.Errorf("Get with IgnoreNotFound failed: %v", err)
	}

	// Test Delete
	var deleted TestObject
	err = s.Delete(ctx, "test-key", &deleted, nil, nil, nil)
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	// Test Delete non-existing
	err = s.Delete(ctx, "non-existing", &deleted, nil, nil, nil)
	if err == nil {
		t.Error("Expected error when deleting non-existing key")
	}
}

func TestMemoryStorage_MultiTenancy(t *testing.T) {
	ctx := context.Background()

	tenant1Config := k1sstorage.Config{TenantID: "tenant1"}
	tenant2Config := k1sstorage.Config{TenantID: "tenant2"}

	s1 := NewMemoryStorage(tenant1Config)
	s2 := NewMemoryStorage(tenant2Config)
	defer func() { _ = s1.Close() }()
	defer func() { _ = s2.Close() }()

	obj1 := &TestObject{
		ObjectMeta: metav1.ObjectMeta{Name: "obj"},
		Data:       "tenant1 data",
	}
	obj2 := &TestObject{
		ObjectMeta: metav1.ObjectMeta{Name: "obj"},
		Data:       "tenant2 data",
	}

	// Both should succeed because they're isolated by tenant
	err := s1.Create(ctx, "test-key", obj1, nil, 0)
	if err != nil {
		t.Errorf("Tenant1 Create failed: %v", err)
	}

	err = s2.Create(ctx, "test-key", obj2, nil, 0)
	if err != nil {
		t.Errorf("Tenant2 Create failed: %v", err)
	}

	// Verify isolation
	var retrieved1, retrieved2 TestObject
	err = s1.Get(ctx, "test-key", storage.GetOptions{}, &retrieved1)
	if err != nil {
		t.Errorf("Tenant1 Get failed: %v", err)
	}

	err = s2.Get(ctx, "test-key", storage.GetOptions{}, &retrieved2)
	if err != nil {
		t.Errorf("Tenant2 Get failed: %v", err)
	}
}

func TestMemoryStorage_BackendInterface(t *testing.T) {
	ctx := context.Background()
	config := k1sstorage.Config{}
	s := NewMemoryStorage(config)
	defer func() { _ = s.Close() }()

	// Test Name
	if s.Name() != "memory" {
		t.Errorf("Expected name 'memory', got %s", s.Name())
	}

	// Test Versioner
	versioner := s.Versioner()
	if versioner == nil {
		t.Error("Versioner should not be nil")
	}

	// Test Compact (no-op)
	err := s.Compact(ctx)
	if err != nil {
		t.Errorf("Compact failed: %v", err)
	}

	// Test Count
	count, err := s.Count(ctx, "test/")
	if err != nil {
		t.Errorf("Count failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}

	// Create some objects and test count
	for i := 0; i < 3; i++ {
		obj := &TestObject{
			ObjectMeta: metav1.ObjectMeta{Name: "obj"},
			Data:       "test data",
		}
		err = s.Create(ctx, "test/obj", obj, nil, 0)
		if err != nil && i == 0 { // Only first should succeed
			t.Errorf("Create failed: %v", err)
		}
	}

	count, err = s.Count(ctx, "test/")
	if err != nil {
		t.Errorf("Count after create failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}
}

func TestMemoryStorage_ContextHandling(t *testing.T) {
	config := k1sstorage.Config{}
	s := NewMemoryStorage(config)
	defer func() { _ = s.Close() }()

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	obj := &TestObject{
		ObjectMeta: metav1.ObjectMeta{Name: "test-obj"},
		Data:       "test data",
	}

	// Test Create with cancelled context
	err := s.Create(ctx, "test-key", obj, nil, 0)
	if err == nil {
		t.Error("Expected error with cancelled context")
	}
	if !k1sstorage.IsContextCancelled(err) {
		t.Error("Expected context cancelled error")
	}

	// Test Get with cancelled context
	var retrieved TestObject
	err = s.Get(ctx, "test-key", storage.GetOptions{}, &retrieved)
	if err == nil {
		t.Error("Expected error with cancelled context")
	}
	if !k1sstorage.IsContextCancelled(err) {
		t.Error("Expected context cancelled error")
	}
}
