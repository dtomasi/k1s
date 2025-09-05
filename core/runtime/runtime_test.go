package runtime_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	k8sstorage "k8s.io/apiserver/pkg/storage"

	"github.com/dtomasi/k1s/core/client"
	"github.com/dtomasi/k1s/core/registry"
	k1sruntime "github.com/dtomasi/k1s/core/runtime"
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
	return nil, nil
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

func createTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	return scheme
}

func createTestClient() client.Client {
	mockStore := &mockStorage{}
	testScheme := createTestScheme()
	mockReg := &mockRegistry{}

	clientImpl, _ := client.NewClient(client.ClientOptions{
		Scheme:   testScheme,
		Storage:  mockStore,
		Registry: mockReg,
	})
	return clientImpl
}

var _ = Describe("Runtime Tests", func() {
	var testClient client.Client
	var testScheme *runtime.Scheme
	var mockStore storage.Interface

	BeforeEach(func() {
		testClient = createTestClient()
		testScheme = createTestScheme()
		mockStore = &mockStorage{}
	})

	Context("New Dependency Injection API", func() {
		It("should create runtime with storage backend injection", func() {
			runtime, err := k1sruntime.NewRuntime(mockStore)
			Expect(err).NotTo(HaveOccurred())
			Expect(runtime).NotTo(BeNil())
			Expect(runtime.IsStarted()).To(BeFalse())
		})

		It("should create runtime with functional options", func() {
			runtime, err := k1sruntime.NewRuntime(
				mockStore,
				k1sruntime.WithTenant("test-tenant"),
				k1sruntime.WithEvents(false),
				k1sruntime.WithDefaultComponent("test-component"),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(runtime).NotTo(BeNil())
		})

		It("should reject nil storage backend", func() {
			_, err := k1sruntime.NewRuntime(nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("storage backend is required"))
		})

		It("should create runtime with default configuration", func() {
			runtime, err := k1sruntime.CreateRuntimeWithStorage(mockStore)
			Expect(err).NotTo(HaveOccurred())
			Expect(runtime).NotTo(BeNil())
		})

		It("should create runtime with default configuration and component", func() {
			runtime, err := k1sruntime.CreateDefaultRuntimeWithStorage(mockStore, "test-app")
			Expect(err).NotTo(HaveOccurred())
			Expect(runtime).NotTo(BeNil())
		})
	})

	Context("Legacy Runtime creation functions", func() {
		It("should test CreateRuntimeWithClient", func() {
			runtime, err := k1sruntime.CreateRuntimeWithClient(testClient, testScheme)
			Expect(err).NotTo(HaveOccurred())
			Expect(runtime).NotTo(BeNil())
		})

		It("should test CreateRuntimeWithoutEvents", func() {
			runtime, err := k1sruntime.CreateRuntimeWithoutEvents(testClient, testScheme)
			Expect(err).NotTo(HaveOccurred())
			Expect(runtime).NotTo(BeNil())
		})

		It("should test CreateDefaultRuntime", func() {
			runtime, err := k1sruntime.CreateDefaultRuntime(testClient, testScheme, "test-component")
			Expect(err).NotTo(HaveOccurred())
			Expect(runtime).NotTo(BeNil())
		})
	})

	Context("RuntimeFactory", func() {
		It("should test DefaultRuntimeFactory.CreateRuntime (legacy)", func() {
			factory := &k1sruntime.DefaultRuntimeFactory{}
			runtime, err := factory.CreateRuntime(k1sruntime.RuntimeOptions{
				Client:           testClient,
				Scheme:           testScheme,
				EnableEvents:     false, // Disable events to avoid deadlock
				DefaultComponent: "test",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(runtime).NotTo(BeNil())
		})

		It("should test DefaultRuntimeFactory.CreateRuntimeWithStorage (new API)", func() {
			factory := &k1sruntime.DefaultRuntimeFactory{}
			runtime, err := factory.CreateRuntimeWithStorage(
				mockStore,
				k1sruntime.WithEvents(false), // Disable events to avoid deadlock
				k1sruntime.WithDefaultComponent("test"),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(runtime).NotTo(BeNil())
		})
	})

	Context("RuntimeManager", func() {
		It("should test RuntimeManager lifecycle with new API", func() {
			runtime, err := k1sruntime.NewRuntime(
				mockStore,
				k1sruntime.WithEvents(false),
			)
			Expect(err).NotTo(HaveOccurred())

			manager := k1sruntime.NewRuntimeManager(runtime)
			Expect(manager).NotTo(BeNil())

			// Test GetRuntime
			retrievedRuntime := manager.GetRuntime()
			Expect(retrievedRuntime).To(Equal(runtime))

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			// Test Start
			err = manager.Start(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Test Stop
			err = manager.Stop(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should test RuntimeManager lifecycle with legacy API", func() {
			runtime, err := k1sruntime.NewRuntimeWithOptions(k1sruntime.RuntimeOptions{
				Client:       testClient,
				Scheme:       testScheme,
				EnableEvents: false,
			})
			Expect(err).NotTo(HaveOccurred())

			manager := k1sruntime.NewRuntimeManager(runtime)
			Expect(manager).NotTo(BeNil())

			// Test GetRuntime
			retrievedRuntime := manager.GetRuntime()
			Expect(retrievedRuntime).To(Equal(runtime))

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			// Test Start
			err = manager.Start(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Test Stop
			err = manager.Stop(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Runtime basic functionality", func() {
		It("should test basic runtime operations with new API", func() {
			runtime, err := k1sruntime.NewRuntime(
				mockStore,
				k1sruntime.WithEvents(false),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(runtime).NotTo(BeNil())

			// Test not started initially
			Expect(runtime.IsStarted()).To(BeFalse())

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			// Start runtime
			err = runtime.Start(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Test IsStarted
			Expect(runtime.IsStarted()).To(BeTrue())

			// Test GetClient
			client := runtime.GetClient()
			Expect(client).NotTo(BeNil())

			// With events disabled, these should be nil
			broadcaster := runtime.GetEventBroadcaster()
			Expect(broadcaster).To(BeNil())

			recorder := runtime.GetEventRecorder("test")
			Expect(recorder).To(BeNil())

			eventClient := runtime.GetEventAwareClient()
			Expect(eventClient).To(BeNil())

			// Stop runtime
			err = runtime.Stop(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Test IsStarted after stop
			Expect(runtime.IsStarted()).To(BeFalse())
		})

		It("should test basic runtime operations with legacy API", func() {
			runtime, err := k1sruntime.NewRuntimeWithOptions(k1sruntime.RuntimeOptions{
				Client:       testClient,
				Scheme:       testScheme,
				EnableEvents: false,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(runtime).NotTo(BeNil())

			// Test not started initially
			Expect(runtime.IsStarted()).To(BeFalse())

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			// Start runtime
			err = runtime.Start(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Test IsStarted
			Expect(runtime.IsStarted()).To(BeTrue())

			// Test GetClient
			client := runtime.GetClient()
			Expect(client).NotTo(BeNil())

			// With events disabled, these should be nil
			broadcaster := runtime.GetEventBroadcaster()
			Expect(broadcaster).To(BeNil())

			recorder := runtime.GetEventRecorder("test")
			Expect(recorder).To(BeNil())

			eventClient := runtime.GetEventAwareClient()
			Expect(eventClient).To(BeNil())

			// Stop runtime
			err = runtime.Stop(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Test IsStarted after stop
			Expect(runtime.IsStarted()).To(BeFalse())
		})
	})

	Context("Error conditions", func() {
		It("should handle double start with new API", func() {
			runtime, err := k1sruntime.NewRuntime(
				mockStore,
				k1sruntime.WithEvents(false),
			)
			Expect(err).NotTo(HaveOccurred())

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			// First start should succeed
			err = runtime.Start(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Second start should fail
			err = runtime.Start(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already started"))

			err = runtime.Stop(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle double start with legacy API", func() {
			runtime, err := k1sruntime.NewRuntimeWithOptions(k1sruntime.RuntimeOptions{
				Client:       testClient,
				Scheme:       testScheme,
				EnableEvents: false,
			})
			Expect(err).NotTo(HaveOccurred())

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			// First start should succeed
			err = runtime.Start(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Second start should fail
			err = runtime.Start(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already started"))

			err = runtime.Stop(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle stop without start with new API", func() {
			runtime, err := k1sruntime.NewRuntime(
				mockStore,
				k1sruntime.WithEvents(false),
			)
			Expect(err).NotTo(HaveOccurred())

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			// Stop without start should not error
			err = runtime.Stop(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle stop without start with legacy API", func() {
			runtime, err := k1sruntime.NewRuntimeWithOptions(k1sruntime.RuntimeOptions{
				Client:       testClient,
				Scheme:       testScheme,
				EnableEvents: false,
			})
			Expect(err).NotTo(HaveOccurred())

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			// Stop without start should not error
			err = runtime.Stop(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle invalid runtime options with new API", func() {
			// Missing storage backend
			_, err := k1sruntime.NewRuntime(nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("storage backend is required"))

			// Invalid tenant name
			_, err = k1sruntime.NewRuntime(mockStore, k1sruntime.WithTenant(""))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("tenant name cannot be empty"))

			// Invalid component name
			_, err = k1sruntime.NewRuntime(mockStore, k1sruntime.WithDefaultComponent(""))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("default component cannot be empty"))
		})

		It("should handle invalid runtime options with legacy API", func() {
			// Missing client
			_, err := k1sruntime.NewRuntimeWithOptions(k1sruntime.RuntimeOptions{
				Scheme: testScheme,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("client is required"))

			// Missing scheme
			_, err = k1sruntime.NewRuntimeWithOptions(k1sruntime.RuntimeOptions{
				Client: testClient,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("scheme is required"))
		})
	})

	Context("Configuration and Options", func() {
		It("should apply functional options correctly", func() {
			// Test all options together
			runtime, err := k1sruntime.NewRuntime(
				mockStore,
				k1sruntime.WithTenant("custom-tenant"),
				k1sruntime.WithRBAC(true),
				k1sruntime.WithEvents(true),
				k1sruntime.WithDefaultComponent("custom-component"),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(runtime).NotTo(BeNil())
		})

		It("should use default config when no options provided", func() {
			runtime, err := k1sruntime.NewRuntime(mockStore)
			Expect(err).NotTo(HaveOccurred())
			Expect(runtime).NotTo(BeNil())
		})
	})
})
