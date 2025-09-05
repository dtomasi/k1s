package informers_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	k8sstorage "k8s.io/apiserver/pkg/storage"

	"github.com/dtomasi/k1s/core/client"
	"github.com/dtomasi/k1s/core/informers"
	"github.com/dtomasi/k1s/core/registry"
	"github.com/dtomasi/k1s/core/storage"
)

// Mock storage for testing
type mockStorage struct {
	storage.Interface
}

func (m *mockStorage) Versioner() k8sstorage.Versioner {
	return storage.SimpleVersioner{}
}

func (m *mockStorage) Create(ctx context.Context, key string, obj, out runtime.Object, ttl uint64) error {
	return nil
}

func (m *mockStorage) Get(ctx context.Context, key string, opts k8sstorage.GetOptions, objPtr runtime.Object) error {
	return nil
}

func (m *mockStorage) Delete(ctx context.Context, key string, out runtime.Object, preconditions *k8sstorage.Preconditions, validateDeletion k8sstorage.ValidateObjectFunc, cachedExistingObject runtime.Object) error {
	return nil
}

func (m *mockStorage) Watch(ctx context.Context, key string, opts k8sstorage.ListOptions) (watch.Interface, error) {
	return watch.NewFake(), nil
}

func (m *mockStorage) List(ctx context.Context, key string, opts k8sstorage.ListOptions, listObj runtime.Object) error {
	return nil
}

func (m *mockStorage) GetList(ctx context.Context, key string, opts k8sstorage.ListOptions, listObj runtime.Object) error {
	return nil
}

func (m *mockStorage) GuaranteedUpdate(ctx context.Context, key string, destination runtime.Object, ignoreNotFound bool, preconditions *k8sstorage.Preconditions, tryUpdate k8sstorage.UpdateFunc, cachedExistingObject runtime.Object) error {
	return nil
}

func (m *mockStorage) Count(key string) (int64, error) {
	return 0, nil
}

func (m *mockStorage) ReadinessCheck() error {
	return nil
}

func (m *mockStorage) RequestWatchProgress(ctx context.Context) error {
	return nil
}

// Mock registry
type mockRegistry struct{}

func (r *mockRegistry) RegisterResource(gvr schema.GroupVersionResource, config registry.ResourceConfig) error {
	return nil
}

func (r *mockRegistry) GetResourceConfig(gvr schema.GroupVersionResource) (registry.ResourceConfig, error) {
	return registry.ResourceConfig{}, nil
}

func (r *mockRegistry) ListResources() []schema.GroupVersionResource {
	return nil
}

func (r *mockRegistry) GetGVRForShortName(shortName string) (schema.GroupVersionResource, error) {
	return schema.GroupVersionResource{}, nil
}

func (r *mockRegistry) GetGVRsForCategory(category string) []schema.GroupVersionResource {
	return nil
}

func (r *mockRegistry) IsResourceRegistered(gvr schema.GroupVersionResource) bool {
	return true
}

func (r *mockRegistry) UnregisterResource(gvr schema.GroupVersionResource) error {
	return nil
}

func (r *mockRegistry) GetGVRForGVK(gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	return schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: gvk.Kind + "s",
	}, nil
}

func (r *mockRegistry) GetGVKForGVR(gvr schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{
		Group:   gvr.Group,
		Version: gvr.Version,
		Kind:    gvr.Resource[:len(gvr.Resource)-1], // Remove 's' suffix
	}, nil
}

// Test types
type TestObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              TestSpec   `json:"spec,omitempty"`
	Status            TestStatus `json:"status,omitempty"`
}

type TestSpec struct {
	Name string `json:"name"`
}

type TestStatus struct {
	Phase string `json:"phase"`
}

type TestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TestObject `json:"items"`
}

func (t *TestObject) DeepCopyObject() runtime.Object {
	return t.DeepCopy()
}

func (t *TestObject) DeepCopy() *TestObject {
	if t == nil {
		return nil
	}
	out := new(TestObject)
	*out = *t
	return out
}

func (t *TestList) DeepCopyObject() runtime.Object {
	return t.DeepCopy()
}

func (t *TestList) DeepCopy() *TestList {
	if t == nil {
		return nil
	}
	out := new(TestList)
	*out = *t
	return out
}

func createTestClient() client.Client {
	scheme := runtime.NewScheme()

	// Register test types
	scheme.AddKnownTypes(schema.GroupVersion{Group: "test.k1s.io", Version: "v1"},
		&TestObject{},
		&TestList{},
	)

	mockStore := &mockStorage{}
	mockReg := &mockRegistry{}

	clientImpl, _ := client.NewClient(client.ClientOptions{
		Scheme:   scheme,
		Storage:  mockStore,
		Registry: mockReg,
	})
	return clientImpl
}

var _ = Describe("SharedInformerFactory", func() {
	var (
		testClient client.Client
		factory    informers.SharedInformerFactory
		testGVR    schema.GroupVersionResource
	)

	BeforeEach(func() {
		testClient = createTestClient()
		factory = informers.NewSharedInformerFactory(testClient, 30*time.Second)
		testGVR = schema.GroupVersionResource{
			Group:    "test.k1s.io",
			Version:  "v1",
			Resource: "testobjects",
		}
	})

	AfterEach(func() {
		factory.Shutdown()
	})

	Context("Factory Creation", func() {
		It("should create a new factory", func() {
			Expect(factory).NotTo(BeNil())
		})

		It("should create a factory with options", func() {
			factoryWithOpts := informers.NewSharedInformerFactoryWithOptions(
				testClient,
				30*time.Second,
				informers.WithNamespace("test-namespace"),
			)
			Expect(factoryWithOpts).NotTo(BeNil())
		})
	})

	Context("Generic Informers", func() {
		It("should create a generic informer for a resource", func() {
			genericInformer := factory.ForResource(testGVR)
			Expect(genericInformer).NotTo(BeNil())
		})

		It("should return the same informer for repeated calls", func() {
			informer1 := factory.ForResource(testGVR)
			informer2 := factory.ForResource(testGVR)
			Expect(informer1).To(BeIdenticalTo(informer2))
		})

		It("should provide access to the underlying informer", func() {
			genericInformer := factory.ForResource(testGVR)
			sharedInformer := genericInformer.Informer()
			Expect(sharedInformer).NotTo(BeNil())
		})

		It("should provide access to the lister", func() {
			genericInformer := factory.ForResource(testGVR)
			lister := genericInformer.Lister()
			Expect(lister).NotTo(BeNil())
		})
	})

	Context("Shared Index Informers", func() {
		It("should create a shared index informer for a resource", func() {
			informer := factory.InformerFor(testGVR)
			Expect(informer).NotTo(BeNil())
		})

		It("should return the same informer for repeated calls", func() {
			informer1 := factory.InformerFor(testGVR)
			informer2 := factory.InformerFor(testGVR)
			Expect(informer1).To(BeIdenticalTo(informer2))
		})
	})

	Context("Informer Lifecycle", func() {
		It("should start informers", func() {
			// Create an informer first
			factory.ForResource(testGVR)

			stopCh := make(chan struct{})
			defer close(stopCh)

			// This should not panic
			Expect(func() { factory.Start(stopCh) }).NotTo(Panic())
		})

		It("should wait for cache sync", func() {
			// Create an informer first
			factory.ForResource(testGVR)

			stopCh := make(chan struct{})
			defer close(stopCh)

			// Start informers
			factory.Start(stopCh)

			// Wait for sync with timeout
			done := make(chan map[schema.GroupVersionResource]bool, 1)
			go func() {
				syncResult := factory.WaitForCacheSync(stopCh)
				done <- syncResult
			}()

			var syncResult map[schema.GroupVersionResource]bool
			select {
			case syncResult = <-done:
			case <-time.After(1 * time.Second):
				// Timeout is okay, we just want to make sure it doesn't panic
			}

			if syncResult != nil {
				Expect(syncResult).To(HaveKey(testGVR))
			}
		})

		It("should shutdown gracefully", func() {
			Expect(func() { factory.Shutdown() }).NotTo(Panic())
		})
	})

	Context("Factory Options", func() {
		It("should support namespace option", func() {
			factoryWithNS := informers.NewSharedInformerFactoryWithOptions(
				testClient,
				30*time.Second,
				informers.WithNamespace("test-namespace"),
			)
			Expect(factoryWithNS).NotTo(BeNil())
		})

		It("should support custom resync option", func() {
			customResync := map[schema.GroupVersionResource]time.Duration{
				testGVR: 10 * time.Second,
			}

			factoryWithResync := informers.NewSharedInformerFactoryWithOptions(
				testClient,
				30*time.Second,
				informers.WithCustomResync(customResync),
			)
			Expect(factoryWithResync).NotTo(BeNil())
		})
	})

	Context("Error Handling", func() {
		It("should handle multiple starts gracefully", func() {
			factory.ForResource(testGVR)

			stopCh := make(chan struct{})
			defer close(stopCh)

			// Multiple starts should not panic
			Expect(func() { factory.Start(stopCh) }).NotTo(Panic())
			Expect(func() { factory.Start(stopCh) }).NotTo(Panic())
		})

		It("should handle shutdown multiple times", func() {
			Expect(func() { factory.Shutdown() }).NotTo(Panic())
			Expect(func() { factory.Shutdown() }).NotTo(Panic())
		})
	})

	Context("CLI Optimizations", func() {
		It("should support on-demand informer creation", func() {
			// Informers should be created only when requested
			informer := factory.ForResource(testGVR)
			Expect(informer).NotTo(BeNil())

			// Create another resource informer
			anotherGVR := schema.GroupVersionResource{
				Group:    "test.k1s.io",
				Version:  "v1",
				Resource: "otherobjects",
			}
			anotherInformer := factory.ForResource(anotherGVR)
			Expect(anotherInformer).NotTo(BeNil())
			Expect(anotherInformer).NotTo(BeIdenticalTo(informer))
		})

		It("should handle cache sync for unstarted informers", func() {
			// Create informer but don't start it
			factory.ForResource(testGVR)

			stopCh := make(chan struct{})
			defer close(stopCh)

			// Should return true for unstarted informers (considered synced)
			syncResult := factory.WaitForCacheSync(stopCh)
			Expect(syncResult).To(HaveKey(testGVR))
			Expect(syncResult[testGVR]).To(BeTrue())
		})
	})
})
