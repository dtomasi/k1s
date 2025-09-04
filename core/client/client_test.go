package client_test

import (
	"context"
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"

	"github.com/dtomasi/k1s/core/client"
	"github.com/dtomasi/k1s/core/codec"
	"github.com/dtomasi/k1s/core/defaulting"
	"github.com/dtomasi/k1s/core/registry"
	k1sruntime "github.com/dtomasi/k1s/core/runtime"
	"github.com/dtomasi/k1s/core/validation"
)

// Mock implementations for testing

// mockStorage implements the storage.Interface for testing
type mockStorage struct {
	objects map[string]runtime.Object
	watches map[string][]chan watch.Event
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		objects: make(map[string]runtime.Object),
		watches: make(map[string][]chan watch.Event),
	}
}

func (m *mockStorage) Versioner() storage.Versioner {
	return storage.APIObjectVersioner{}
}

func (m *mockStorage) Create(ctx context.Context, key string, obj, out runtime.Object, ttl uint64) error {
	if m.objects[key] != nil {
		return errors.New("object already exists")
	}
	m.objects[key] = obj.DeepCopyObject()
	if out != nil {
		copyObjectFields(obj, out)
	}
	return nil
}

func (m *mockStorage) Delete(ctx context.Context, key string, out runtime.Object, preconditions *storage.Preconditions, validateDeletion storage.ValidateObjectFunc, cachedExistingObject runtime.Object) error {
	obj, exists := m.objects[key]
	if !exists {
		return errors.New("not found")
	}
	delete(m.objects, key)
	if out != nil {
		copyObjectFields(obj, out)
	}
	return nil
}

func (m *mockStorage) Get(ctx context.Context, key string, opts storage.GetOptions, objPtr runtime.Object) error {
	obj, exists := m.objects[key]
	if !exists {
		return errors.New("not found")
	}
	copyObjectFields(obj, objPtr)
	return nil
}

func (m *mockStorage) List(ctx context.Context, key string, opts storage.ListOptions, listObj runtime.Object) error {
	// Simple implementation - return all objects that start with the key prefix
	items := []runtime.Object{}
	for k, obj := range m.objects {
		if len(k) >= len(key) && k[:len(key)] == key {
			items = append(items, obj)
		}
	}

	// Set the items in the list object using reflection
	if list, ok := listObj.(*TestItemList); ok {
		list.Items = []TestItem{}
		for _, item := range items {
			if testItem, ok := item.(*TestItem); ok {
				list.Items = append(list.Items, *testItem)
			}
		}
	}
	return nil
}

func (m *mockStorage) Watch(ctx context.Context, key string, opts storage.ListOptions) (watch.Interface, error) {
	ch := make(chan watch.Event, 100)
	if m.watches[key] == nil {
		m.watches[key] = []chan watch.Event{}
	}
	m.watches[key] = append(m.watches[key], ch)
	return &mockWatcher{ch: ch}, nil
}

// mockWatcher implements watch.Interface
type mockWatcher struct {
	ch chan watch.Event
}

func (w *mockWatcher) Stop() {
	close(w.ch)
}

func (w *mockWatcher) ResultChan() <-chan watch.Event {
	return w.ch
}

// mockValidator implements validation.Validator
type mockValidator struct {
	shouldFailValidation bool
	shouldFailUpdate     bool
	shouldFailDelete     bool
}

var _ validation.Validator = (*mockValidator)(nil)

func (v *mockValidator) Validate(ctx context.Context, obj runtime.Object) error {
	if v.shouldFailValidation {
		return errors.New("validation failed")
	}
	return nil
}

func (v *mockValidator) ValidateUpdate(ctx context.Context, obj, old runtime.Object) error {
	if v.shouldFailUpdate {
		return errors.New("update validation failed")
	}
	return nil
}

func (v *mockValidator) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	if v.shouldFailDelete {
		return errors.New("delete validation failed")
	}
	return nil
}

// mockDefaulter implements defaulting.Defaulter
type mockDefaulter struct {
	shouldFail bool
}

var _ defaulting.Defaulter = (*mockDefaulter)(nil)

func (d *mockDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	if d.shouldFail {
		return errors.New("defaulting failed")
	}

	// Apply some defaults for testing
	if testItem, ok := obj.(*TestItem); ok {
		if testItem.Spec.Quantity == 0 {
			testItem.Spec.Quantity = 1
		}
		if testItem.Status.Status == "" {
			testItem.Status.Status = "Available"
		}
	}
	return nil
}

// mockRegistry implements registry.Registry
type mockRegistry struct{}

func (r *mockRegistry) RegisterResource(gvr schema.GroupVersionResource, config registry.ResourceConfig) error {
	return nil
}

func (r *mockRegistry) GetResourceConfig(gvr schema.GroupVersionResource) (registry.ResourceConfig, error) {
	return registry.ResourceConfig{
		Singular:   "testitem",
		Plural:     "testitems",
		Kind:       "TestItem",
		ListKind:   "TestItemList",
		Namespaced: true,
	}, nil
}

func (r *mockRegistry) ListResources() []schema.GroupVersionResource {
	return []schema.GroupVersionResource{
		{Group: "test.k1s.io", Version: "v1", Resource: "testitems"},
	}
}

func (r *mockRegistry) GetGVRForShortName(shortName string) (schema.GroupVersionResource, error) {
	return schema.GroupVersionResource{}, errors.New("not found")
}

func (r *mockRegistry) GetGVRsForCategory(category string) []schema.GroupVersionResource {
	return []schema.GroupVersionResource{}
}

func (r *mockRegistry) IsResourceRegistered(gvr schema.GroupVersionResource) bool {
	return gvr.Resource == "testitems"
}

func (r *mockRegistry) UnregisterResource(gvr schema.GroupVersionResource) error {
	return nil
}

func (r *mockRegistry) GetGVKForGVR(gvr schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	if gvr.Resource == "testitems" {
		return schema.GroupVersionKind{Group: "test.k1s.io", Version: "v1", Kind: "TestItem"}, nil
	}
	return schema.GroupVersionKind{}, errors.New("not found")
}

func (r *mockRegistry) GetGVRForGVK(gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	if gvk.Kind == "TestItem" {
		return schema.GroupVersionResource{Group: "test.k1s.io", Version: "v1", Resource: "testitems"}, nil
	}
	if gvk.Kind == "TestItemList" {
		return schema.GroupVersionResource{Group: "test.k1s.io", Version: "v1", Resource: "testitems"}, nil
	}
	return schema.GroupVersionResource{}, errors.New("not found")
}

// Test object types
type TestItem struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              TestItemSpec   `json:"spec,omitempty"`
	Status            TestItemStatus `json:"status,omitempty"`
}

func (t *TestItem) DeepCopyObject() runtime.Object {
	return &TestItem{
		TypeMeta:   t.TypeMeta,
		ObjectMeta: *t.DeepCopy(),
		Spec:       t.Spec,
		Status:     t.Status,
	}
}

type TestItemSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Quantity    int32  `json:"quantity"`
}

type TestItemStatus struct {
	Status string `json:"status"`
}

type TestItemList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TestItem `json:"items"`
}

func (t *TestItemList) DeepCopyObject() runtime.Object {
	items := make([]TestItem, len(t.Items))
	copy(items, t.Items)
	return &TestItemList{
		TypeMeta: t.TypeMeta,
		ListMeta: *t.DeepCopy(),
		Items:    items,
	}
}

// Helper functions
func copyObjectFields(src, dst runtime.Object) {
	if srcItem, ok := src.(*TestItem); ok {
		if dstItem, ok := dst.(*TestItem); ok {
			*dstItem = *srcItem
		}
	}
}

func createTestScheme() *runtime.Scheme {
	scheme := k1sruntime.NewScheme()
	scheme.AddKnownTypes(schema.GroupVersion{Group: "test.k1s.io", Version: "v1"}, &TestItem{}, &TestItemList{})
	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Group: "test.k1s.io", Version: "v1"})
	return scheme
}

var _ = Describe("Client", func() {
	var (
		testClient  client.Client
		mockStore   *mockStorage
		mockValid   *mockValidator
		mockDefault *mockDefaulter
		mockReg     *mockRegistry
		testScheme  *runtime.Scheme
		testItem    *TestItem
		ctx         context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockStore = newMockStorage()
		mockValid = &mockValidator{}
		mockDefault = &mockDefaulter{}
		mockReg = &mockRegistry{}
		testScheme = createTestScheme()

		var err error
		testClient, err = client.NewClient(client.ClientOptions{
			Scheme:       testScheme,
			Storage:      mockStore,
			Validator:    mockValid,
			Defaulter:    mockDefault,
			Registry:     mockReg,
			CodecFactory: codec.NewCodecFactory(testScheme),
		})
		Expect(err).NotTo(HaveOccurred())

		testItem = &TestItem{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "test.k1s.io/v1",
				Kind:       "TestItem",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-item",
				Namespace: "default",
			},
			Spec: TestItemSpec{
				Name:        "Test Item",
				Description: "A test item",
				Quantity:    0, // Will be defaulted to 1
			},
		}
	})

	Describe("NewClient", func() {
		It("should create a new client with valid options", func() {
			c, err := client.NewClient(client.ClientOptions{
				Scheme:   testScheme,
				Storage:  mockStore,
				Registry: mockReg,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(c).NotTo(BeNil())
			Expect(c.Scheme()).To(Equal(testScheme))
		})

		It("should return error when scheme is nil", func() {
			_, err := client.NewClient(client.ClientOptions{
				Storage:  mockStore,
				Registry: mockReg,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("scheme is required"))
		})

		It("should return error when storage is nil", func() {
			_, err := client.NewClient(client.ClientOptions{
				Scheme:   testScheme,
				Registry: mockReg,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("storage is required"))
		})

		It("should return error when registry is nil", func() {
			_, err := client.NewClient(client.ClientOptions{
				Scheme:  testScheme,
				Storage: mockStore,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("registry is required"))
		})
	})

	Describe("Create", func() {
		It("should create an object successfully", func() {
			err := testClient.Create(ctx, testItem)
			Expect(err).NotTo(HaveOccurred())
			Expect(testItem.Spec.Quantity).To(Equal(int32(1)))    // Applied by defaulter
			Expect(testItem.Status.Status).To(Equal("Available")) // Applied by defaulter
		})

		It("should fail when validation fails", func() {
			mockValid.shouldFailValidation = true
			err := testClient.Create(ctx, testItem)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("validation failed"))
		})

		It("should fail when defaulting fails", func() {
			mockDefault.shouldFail = true
			err := testClient.Create(ctx, testItem)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("defaulting failed"))
		})

		It("should fail when object already exists", func() {
			// Create the object first
			err := testClient.Create(ctx, testItem)
			Expect(err).NotTo(HaveOccurred())

			// Try to create again
			err = testClient.Create(ctx, testItem)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists"))
		})
	})

	Describe("Get", func() {
		BeforeEach(func() {
			err := testClient.Create(ctx, testItem)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should get an object successfully", func() {
			key := client.ObjectKey{
				Namespace: "default",
				Name:      "test-item",
			}

			gotItem := &TestItem{}
			err := testClient.Get(ctx, key, gotItem)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotItem.Name).To(Equal("test-item"))
			Expect(gotItem.Spec.Name).To(Equal("Test Item"))
		})

		It("should fail when object does not exist", func() {
			key := client.ObjectKey{
				Namespace: "default",
				Name:      "nonexistent",
			}

			gotItem := &TestItem{}
			err := testClient.Get(ctx, key, gotItem)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})
	})

	Describe("Update", func() {
		BeforeEach(func() {
			err := testClient.Create(ctx, testItem)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should update an object successfully", func() {
			testItem.Spec.Description = "Updated description"
			err := testClient.Update(ctx, testItem)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail when update validation fails", func() {
			mockValid.shouldFailUpdate = true
			testItem.Spec.Description = "Updated description"
			err := testClient.Update(ctx, testItem)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("update validation failed"))
		})
	})

	Describe("Delete", func() {
		BeforeEach(func() {
			err := testClient.Create(ctx, testItem)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should delete an object successfully", func() {
			err := testClient.Delete(ctx, testItem)
			Expect(err).NotTo(HaveOccurred())

			// Verify object is deleted
			key := client.ObjectKey{
				Namespace: "default",
				Name:      "test-item",
			}
			gotItem := &TestItem{}
			err = testClient.Get(ctx, key, gotItem)
			Expect(err).To(HaveOccurred())
		})

		It("should fail when delete validation fails", func() {
			mockValid.shouldFailDelete = true
			err := testClient.Delete(ctx, testItem)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("delete validation failed"))
		})
	})

	Describe("List", func() {
		BeforeEach(func() {
			// Create multiple test items
			for i := 0; i < 3; i++ {
				item := &TestItem{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "test.k1s.io/v1",
						Kind:       "TestItem",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("test-item-%d", i),
						Namespace: "default",
					},
					Spec: TestItemSpec{
						Name: fmt.Sprintf("Test Item %d", i),
					},
				}
				err := testClient.Create(ctx, item)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should list objects successfully", func() {
			list := &TestItemList{}
			err := testClient.List(ctx, list)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(list.Items)).To(BeNumerically(">=", 3))
		})

		It("should list objects in specific namespace", func() {
			list := &TestItemList{}
			err := testClient.List(ctx, list, client.InNamespace("default"))
			Expect(err).NotTo(HaveOccurred())
			Expect(len(list.Items)).To(BeNumerically(">=", 3))
		})
	})

	Describe("Patch", func() {
		BeforeEach(func() {
			err := testClient.Create(ctx, testItem)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should patch an object with merge patch", func() {
			patch := client.RawPatch{
				PatchType: types.MergePatchType,
				PatchData: []byte(`{"spec":{"description":"Patched description"}}`),
			}

			err := testClient.Patch(ctx, testItem, patch)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should patch an object with strategic merge patch", func() {
			original := testItem.DeepCopyObject().(*TestItem)
			testItem.Spec.Description = "Updated via patch"

			patch := client.MergeFrom(original)
			err := testClient.Patch(ctx, testItem, patch)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Status", func() {
		BeforeEach(func() {
			err := testClient.Create(ctx, testItem)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should update status subresource", func() {
			testItem.Status.Status = "Reserved"
			err := testClient.Status().Update(ctx, testItem)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should patch status subresource", func() {
			patch := client.RawPatch{
				PatchType: types.MergePatchType,
				PatchData: []byte(`{"status":{"status":"Sold"}}`),
			}

			err := testClient.Status().Patch(ctx, testItem, patch)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
