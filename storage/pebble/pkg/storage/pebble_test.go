package storage

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	k8storage "k8s.io/apiserver/pkg/storage"

	k1sstorage "github.com/dtomasi/k1s/core/pkg/storage"
)

// Test constants to avoid goconst violations
const (
	testAPIVersion      = "test/v1"
	testObjectKind      = "TestObject"
	testObjectListKind  = "TestObjectList"
	testNamespace       = "default"
	testObjectName      = "test-object"
	testObjects         = "test-objects"
	testObjectsTestName = "test-objects/test-object"
	testObjectsNonExist = "test-objects/non-existent"
)

func TestPebbleStorage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pebble Storage Suite")
}

// TestObject is a simple test object for storage tests
type TestObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TestSpec   `json:"spec,omitempty"`
	Status TestStatus `json:"status,omitempty"`
}

type TestSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type TestStatus struct {
	Phase string `json:"phase"`
}

// DeepCopyObject implements runtime.Object
func (t *TestObject) DeepCopyObject() runtime.Object {
	if t == nil {
		return nil
	}
	out := &TestObject{
		TypeMeta:   t.TypeMeta,
		ObjectMeta: *t.DeepCopy(),
		Spec:       t.Spec,
		Status:     t.Status,
	}
	return out
}

// TestObjectList represents a list of test objects
type TestObjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []TestObject `json:"items"`
}

// DeepCopyObject implements runtime.Object
func (t *TestObjectList) DeepCopyObject() runtime.Object {
	if t == nil {
		return nil
	}
	out := &TestObjectList{
		TypeMeta: t.TypeMeta,
		ListMeta: *t.DeepCopy(),
	}
	if t.Items != nil {
		out.Items = make([]TestObject, len(t.Items))
		for i := range t.Items {
			out.Items[i] = *t.Items[i].DeepCopyObject().(*TestObject)
		}
	}
	return out
}

var _ = Describe("PebbleStorage", func() {
	var (
		storage    k1sstorage.Backend
		ctx        context.Context
		cancel     context.CancelFunc
		tempDir    string
		testObject *TestObject
		testList   *TestObjectList
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)

		var err error
		tempDir, err = os.MkdirTemp("", "pebble-storage-test-*")
		Expect(err).NotTo(HaveOccurred())

		config := k1sstorage.Config{}

		storage = NewPebbleStorageWithPath(tempDir, config)
		Expect(storage).NotTo(BeNil())
		Expect(storage.Name()).To(Equal("pebble"))

		testObject = &TestObject{
			TypeMeta: metav1.TypeMeta{
				APIVersion: testAPIVersion,
				Kind:       testObjectKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testObjectName,
				Namespace: testNamespace,
			},
			Spec: TestSpec{
				Name:        "Test Object",
				Description: "A test object for storage tests",
			},
			Status: TestStatus{
				Phase: "Active",
			},
		}

		testList = &TestObjectList{
			TypeMeta: metav1.TypeMeta{
				APIVersion: testAPIVersion,
				Kind:       testObjectListKind,
			},
		}
	})

	AfterEach(func() {
		if storage != nil {
			Expect(storage.Close()).To(Succeed())
		}
		if tempDir != "" {
			if err := os.RemoveAll(tempDir); err != nil {
				GinkgoT().Logf("Warning: failed to remove temp dir: %v", err)
			}
		}
		cancel()
	})

	Describe("Basic Operations", func() {
		It("should create an object successfully", func() {
			key := testObjectsTestName
			out := &TestObject{}

			err := storage.Create(ctx, key, testObject, out, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(out.Name).To(Equal(testObject.Name))
		})

		It("should fail to create duplicate objects", func() {
			key := testObjectsTestName

			err := storage.Create(ctx, key, testObject, nil, 0)
			Expect(err).NotTo(HaveOccurred())

			// Try to create again - should fail
			err = storage.Create(ctx, key, testObject, nil, 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("key already exists"))
		})

		It("should get an object successfully", func() {
			key := testObjectsTestName

			// Create object first
			err := storage.Create(ctx, key, testObject, nil, 0)
			Expect(err).NotTo(HaveOccurred())

			// Get the object
			retrieved := &TestObject{}
			err = storage.Get(ctx, key, k8storage.GetOptions{}, retrieved)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved.Name).To(Equal(testObject.Name))
			Expect(retrieved.Spec.Description).To(Equal(testObject.Spec.Description))
		})

		It("should fail to get non-existent objects", func() {
			key := testObjectsNonExist
			retrieved := &TestObject{}

			err := storage.Get(ctx, key, k8storage.GetOptions{}, retrieved)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("key not found"))
		})

		It("should ignore not found when requested", func() {
			key := testObjectsNonExist
			retrieved := &TestObject{}

			err := storage.Get(ctx, key, k8storage.GetOptions{IgnoreNotFound: true}, retrieved)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should delete an object successfully", func() {
			key := testObjectsTestName

			// Create object first
			err := storage.Create(ctx, key, testObject, nil, 0)
			Expect(err).NotTo(HaveOccurred())

			// Delete the object
			out := &TestObject{}
			err = storage.Delete(ctx, key, out, nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())

			// Verify deletion - should not be found
			retrieved := &TestObject{}
			err = storage.Get(ctx, key, k8storage.GetOptions{}, retrieved)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("key not found"))
		})

		It("should fail to delete non-existent objects", func() {
			key := testObjectsNonExist
			out := &TestObject{}

			err := storage.Delete(ctx, key, out, nil, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("key not found"))
		})
	})

	Describe("List Operations", func() {
		BeforeEach(func() {
			// Create multiple test objects
			for i := 0; i < 5; i++ {
				obj := testObject.DeepCopyObject().(*TestObject)
				obj.Name = fmt.Sprintf("test-object-%d", i)
				key := fmt.Sprintf("test-objects/test-object-%d", i)

				err := storage.Create(ctx, key, obj, nil, 0)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should list objects with recursive option", func() {
			err := storage.List(ctx, "test-objects", k8storage.ListOptions{Recursive: true}, testList)
			Expect(err).NotTo(HaveOccurred())

			items := []runtime.Object{}
			err = meta.EachListItem(testList, func(obj runtime.Object) error {
				items = append(items, obj)
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(items)).To(Equal(5))
		})

		It("should return empty list for non-existent prefix", func() {
			err := storage.List(ctx, "non-existent", k8storage.ListOptions{Recursive: true}, testList)
			Expect(err).NotTo(HaveOccurred())

			items := []runtime.Object{}
			err = meta.EachListItem(testList, func(obj runtime.Object) error {
				items = append(items, obj)
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(items)).To(Equal(0))
		})
	})

	Describe("Watch Operations", func() {
		It("should create a watch successfully", func() {
			key := testObjects
			opts := k8storage.ListOptions{Recursive: true}

			watcher, err := storage.Watch(ctx, key, opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(watcher).NotTo(BeNil())

			// Clean up
			watcher.Stop()
		})

		It("should receive watch events for creates", func() {
			key := testObjects
			opts := k8storage.ListOptions{Recursive: true}

			watcher, err := storage.Watch(ctx, key, opts)
			Expect(err).NotTo(HaveOccurred())
			defer watcher.Stop()

			// Create object in background
			go func() {
				defer GinkgoRecover()
				time.Sleep(100 * time.Millisecond)
				objKey := "test-objects/watch-test"
				err := storage.Create(ctx, objKey, testObject, nil, 0)
				Expect(err).NotTo(HaveOccurred())
			}()

			// Wait for watch event
			select {
			case event := <-watcher.ResultChan():
				Expect(event.Type).To(Equal(watch.Added))
				Expect(event.Object).NotTo(BeNil())
			case <-time.After(5 * time.Second):
				Fail("Expected watch event was not received")
			}
		})

		It("should receive watch events for deletes", func() {
			key := testObjects
			objKey := "test-objects/watch-delete-test"

			// Create object first
			err := storage.Create(ctx, objKey, testObject, nil, 0)
			Expect(err).NotTo(HaveOccurred())

			// Start watching
			opts := k8storage.ListOptions{Recursive: true}
			watcher, err := storage.Watch(ctx, key, opts)
			Expect(err).NotTo(HaveOccurred())
			defer watcher.Stop()

			// Delete object in background
			go func() {
				defer GinkgoRecover()
				time.Sleep(100 * time.Millisecond)
				err := storage.Delete(ctx, objKey, nil, nil, nil, nil)
				Expect(err).NotTo(HaveOccurred())
			}()

			// Wait for watch event
			select {
			case event := <-watcher.ResultChan():
				Expect(event.Type).To(Equal(watch.Deleted))
				Expect(event.Object).NotTo(BeNil())
			case <-time.After(5 * time.Second):
				Fail("Expected watch event was not received")
			}
		})

		It("should send initial events when requested", func() {
			objKey := "test-objects/initial-event-test"

			// Create object first
			err := storage.Create(ctx, objKey, testObject, nil, 0)
			Expect(err).NotTo(HaveOccurred())

			// Start watching with initial events
			key := testObjects
			sendInitial := true
			opts := k8storage.ListOptions{
				Recursive:         true,
				SendInitialEvents: &sendInitial,
			}

			watcher, err := storage.Watch(ctx, key, opts)
			Expect(err).NotTo(HaveOccurred())
			defer watcher.Stop()

			// Should receive initial event
			select {
			case event := <-watcher.ResultChan():
				Expect(event.Type).To(Equal(watch.Added))
				Expect(event.Object).NotTo(BeNil())
			case <-time.After(2 * time.Second):
				Fail("Expected initial watch event was not received")
			}
		})
	})

	Describe("Resource Versioning", func() {
		It("should track resource versions correctly", func() {
			key := "test-objects/version-test"

			// Create object
			out := &TestObject{}
			err := storage.Create(ctx, key, testObject, out, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(out.ResourceVersion).NotTo(BeEmpty())

			firstVersion := out.ResourceVersion

			// Delete and recreate
			err = storage.Delete(ctx, key, nil, nil, nil, nil)
			Expect(err).NotTo(HaveOccurred())

			err = storage.Create(ctx, key, testObject, out, 0)
			Expect(err).NotTo(HaveOccurred())

			// Version should be different
			Expect(out.ResourceVersion).NotTo(Equal(firstVersion))
		})

		It("should validate resource versions on get", func() {
			key := "test-objects/version-validation-test"

			// Create object
			out := &TestObject{}
			err := storage.Create(ctx, key, testObject, out, 0)
			Expect(err).NotTo(HaveOccurred())

			version := out.ResourceVersion

			// Get with correct version - should succeed
			retrieved := &TestObject{}
			err = storage.Get(ctx, key, k8storage.GetOptions{ResourceVersion: version}, retrieved)
			Expect(err).NotTo(HaveOccurred())

			// Get with incorrect version - should fail
			err = storage.Get(ctx, key, k8storage.GetOptions{ResourceVersion: "999999"}, retrieved)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("resource version mismatch"))
		})
	})

	Describe("Preconditions", func() {
		It("should validate UID preconditions", func() {
			key := "test-objects/uid-precondition-test"

			// Create object
			testObject.UID = "test-uid-123"
			err := storage.Create(ctx, key, testObject, nil, 0)
			Expect(err).NotTo(HaveOccurred())

			// Delete with correct UID - should succeed
			preconditions := &k8storage.Preconditions{UID: &testObject.UID}
			err = storage.Delete(ctx, key, nil, preconditions, nil, testObject)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail on incorrect UID preconditions", func() {
			key := "test-objects/uid-precondition-fail-test"

			// Create object
			testObject.UID = "test-uid-123"
			err := storage.Create(ctx, key, testObject, nil, 0)
			Expect(err).NotTo(HaveOccurred())

			// Delete with incorrect UID - should fail
			incorrectUID := types.UID("wrong-uid")
			preconditions := &k8storage.Preconditions{UID: &incorrectUID}
			err = storage.Delete(ctx, key, nil, preconditions, nil, testObject)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("UID mismatch"))
		})
	})

	Describe("Count Operations", func() {
		BeforeEach(func() {
			// Create test objects
			for i := 0; i < 3; i++ {
				obj := testObject.DeepCopyObject().(*TestObject)
				obj.Name = fmt.Sprintf("count-test-%d", i)
				key := fmt.Sprintf("count-objects/count-test-%d", i)

				err := storage.Create(ctx, key, obj, nil, 0)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should count objects correctly", func() {
			count, err := storage.Count(ctx, "count-objects")
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(int64(3)))
		})

		It("should return zero for non-existent prefix", func() {
			count, err := storage.Count(ctx, "non-existent-objects")
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(int64(0)))
		})
	})

	Describe("Compaction", func() {
		It("should perform compaction successfully", func() {
			// Create some objects first
			for i := 0; i < 10; i++ {
				obj := testObject.DeepCopyObject().(*TestObject)
				obj.Name = fmt.Sprintf("compact-test-%d", i)
				key := fmt.Sprintf("compact-objects/compact-test-%d", i)

				err := storage.Create(ctx, key, obj, nil, 0)
				Expect(err).NotTo(HaveOccurred())
			}

			// Perform compaction
			err := storage.Compact(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Verify objects are still accessible after compaction
			count, err := storage.Count(ctx, "compact-objects")
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(int64(10)))
		})
	})

	Describe("Metrics", func() {
		It("should track operation metrics", func() {
			key := "test-objects/metrics-test"

			// Cast to concrete type to access GetMetrics
			pebbleStorage, ok := storage.(*pebbleStorage)
			Expect(ok).To(BeTrue())

			// Get initial metrics
			ops1, errors1, watchers1 := pebbleStorage.GetMetrics()

			// Perform operations
			err := storage.Create(ctx, key, testObject, nil, 0)
			Expect(err).NotTo(HaveOccurred())

			retrieved := &TestObject{}
			err = storage.Get(ctx, key, k8storage.GetOptions{}, retrieved)
			Expect(err).NotTo(HaveOccurred())

			// Check metrics increased
			ops2, errors2, watchers2 := pebbleStorage.GetMetrics()
			Expect(ops2).To(BeNumerically(">", ops1))
			Expect(errors2).To(Equal(errors1))     // No errors expected
			Expect(watchers2).To(Equal(watchers1)) // No new watchers
		})

		It("should track error metrics", func() {
			key := "test-objects/error-metrics-test"

			// Cast to concrete type to access GetMetrics
			pebbleStorage, ok := storage.(*pebbleStorage)
			Expect(ok).To(BeTrue())

			// Get initial metrics
			_, errors1, _ := pebbleStorage.GetMetrics()

			// Perform operation that should fail
			retrieved := &TestObject{}
			err := storage.Get(ctx, key, k8storage.GetOptions{}, retrieved)
			Expect(err).To(HaveOccurred())

			// Check error metrics increased
			_, errors2, _ := pebbleStorage.GetMetrics()
			Expect(errors2).To(BeNumerically(">", errors1))
		})
	})

	Describe("Context Handling", func() {
		It("should handle cancelled context", func() {
			cancelledCtx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			key := "test-objects/cancelled-context-test"
			err := storage.Create(cancelledCtx, key, testObject, nil, 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("context"))
		})

		It("should handle timeout context", func() {
			timeoutCtx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
			defer cancel()
			time.Sleep(2 * time.Nanosecond) // Ensure timeout

			key := "test-objects/timeout-context-test"
			err := storage.Create(timeoutCtx, key, testObject, nil, 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("context"))
		})
	})

	Describe("Multi-tenancy Support", func() {
		var (
			tenant1Storage k1sstorage.Backend
			tenant2Storage k1sstorage.Backend
			tenant1Dir     string
			tenant2Dir     string
		)

		BeforeEach(func() {
			var err error

			// Create separate directories for tenants
			tenant1Dir, err = os.MkdirTemp("", "tenant1-*")
			Expect(err).NotTo(HaveOccurred())

			tenant2Dir, err = os.MkdirTemp("", "tenant2-*")
			Expect(err).NotTo(HaveOccurred())

			// Create tenant-specific storages
			tenant1Storage = NewPebbleStorageWithPath(tenant1Dir, k1sstorage.Config{
				TenantID: "tenant1",
			})

			tenant2Storage = NewPebbleStorageWithPath(tenant2Dir, k1sstorage.Config{
				TenantID: "tenant2",
			})
		})

		AfterEach(func() {
			if tenant1Storage != nil {
				if err := tenant1Storage.Close(); err != nil {
					GinkgoT().Logf("Warning: failed to close tenant1 storage: %v", err)
				}
			}
			if tenant2Storage != nil {
				if err := tenant2Storage.Close(); err != nil {
					GinkgoT().Logf("Warning: failed to close tenant2 storage: %v", err)
				}
			}
			if err := os.RemoveAll(tenant1Dir); err != nil {
				GinkgoT().Logf("Warning: failed to remove tenant1 temp dir: %v", err)
			}
			if err := os.RemoveAll(tenant2Dir); err != nil {
				GinkgoT().Logf("Warning: failed to remove tenant2 temp dir: %v", err)
			}
		})

		It("should isolate data between tenants", func() {
			key := "test-objects/tenant-isolation-test"

			// Create object in tenant1
			obj1 := testObject.DeepCopyObject().(*TestObject)
			obj1.Name = "tenant1-object"
			err := tenant1Storage.Create(ctx, key, obj1, nil, 0)
			Expect(err).NotTo(HaveOccurred())

			// Create object in tenant2 with same key
			obj2 := testObject.DeepCopyObject().(*TestObject)
			obj2.Name = "tenant2-object"
			err = tenant2Storage.Create(ctx, key, obj2, nil, 0)
			Expect(err).NotTo(HaveOccurred())

			// Verify isolation - tenant1 should only see its object
			retrieved1 := &TestObject{}
			err = tenant1Storage.Get(ctx, key, k8storage.GetOptions{}, retrieved1)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved1.Name).To(Equal("tenant1-object"))

			// Verify isolation - tenant2 should only see its object
			retrieved2 := &TestObject{}
			err = tenant2Storage.Get(ctx, key, k8storage.GetOptions{}, retrieved2)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved2.Name).To(Equal("tenant2-object"))
		})
	})

	Describe("Concurrent Operations", func() {
		It("should handle concurrent creates safely", func() {
			const numGoroutines = 10
			done := make(chan bool, numGoroutines)
			errors := make(chan error, numGoroutines)

			for i := 0; i < numGoroutines; i++ {
				go func(id int) {
					defer GinkgoRecover()
					obj := testObject.DeepCopyObject().(*TestObject)
					obj.Name = fmt.Sprintf("concurrent-test-%d", id)
					key := fmt.Sprintf("concurrent-objects/concurrent-test-%d", id)

					err := storage.Create(ctx, key, obj, nil, 0)
					if err != nil {
						errors <- err
					}
					done <- true
				}(i)
			}

			// Wait for all goroutines to complete
			for i := 0; i < numGoroutines; i++ {
				select {
				case <-done:
					// Success
				case err := <-errors:
					Fail(fmt.Sprintf("Concurrent operation failed: %v", err))
				case <-time.After(10 * time.Second):
					Fail("Concurrent operations timed out")
				}
			}

			// Verify all objects were created
			count, err := storage.Count(ctx, "concurrent-objects")
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(int64(numGoroutines)))
		})
	})

	Describe("Database Statistics", func() {
		It("should provide database statistics", func() {
			// Cast to concrete type to access GetStats
			pebbleStorage, ok := storage.(*pebbleStorage)
			Expect(ok).To(BeTrue())

			// Initially should show "not initialized"
			stats := pebbleStorage.GetStats()
			Expect(stats).To(ContainSubstring("database not initialized"))

			// Create an object to initialize the database
			key := "test-objects/stats-test"
			err := storage.Create(ctx, key, testObject, nil, 0)
			Expect(err).NotTo(HaveOccurred())

			// Now should provide real statistics
			stats = pebbleStorage.GetStats()
			Expect(stats).NotTo(ContainSubstring("database not initialized"))
			Expect(stats).To(ContainSubstring("level"))
		})
	})

	Describe("Close Operations", func() {
		It("should close storage cleanly", func() {
			key := "test-objects/close-test"

			// Create an object
			err := storage.Create(ctx, key, testObject, nil, 0)
			Expect(err).NotTo(HaveOccurred())

			// Close storage
			err = storage.Close()
			Expect(err).NotTo(HaveOccurred())

			// Operations after close should fail
			err = storage.Create(ctx, "test-objects/after-close", testObject, nil, 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("storage is closed"))
		})

		It("should stop watches on close", func() {
			key := testObjects
			opts := k8storage.ListOptions{Recursive: true}

			watcher, err := storage.Watch(ctx, key, opts)
			Expect(err).NotTo(HaveOccurred())

			// Close storage
			err = storage.Close()
			Expect(err).NotTo(HaveOccurred())

			// Watch should be stopped
			select {
			case _, ok := <-watcher.ResultChan():
				Expect(ok).To(BeFalse(), "Watch channel should be closed")
			case <-time.After(2 * time.Second):
				Fail("Watch channel should have been closed")
			}
		})
	})
})
