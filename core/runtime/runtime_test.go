package runtime_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

var _ = Describe("Runtime Coverage Tests", func() {
	var testClient client.Client
	var testScheme *runtime.Scheme

	BeforeEach(func() {
		testClient = createTestClient()
		testScheme = createTestScheme()
	})

	Context("Runtime creation functions", func() {
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
		It("should test DefaultRuntimeFactory.CreateRuntime", func() {
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
	})

	Context("RuntimeManager", func() {
		It("should test RuntimeManager lifecycle", func() {
			runtime, err := k1sruntime.NewRuntime(k1sruntime.RuntimeOptions{
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
		It("should test basic runtime operations without events", func() {
			runtime, err := k1sruntime.NewRuntime(k1sruntime.RuntimeOptions{
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
		It("should handle double start", func() {
			runtime, err := k1sruntime.NewRuntime(k1sruntime.RuntimeOptions{
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

		It("should handle stop without start", func() {
			runtime, err := k1sruntime.NewRuntime(k1sruntime.RuntimeOptions{
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

		It("should handle invalid runtime options", func() {
			// Missing client
			_, err := k1sruntime.NewRuntime(k1sruntime.RuntimeOptions{
				Scheme: testScheme,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("client is required"))

			// Missing scheme
			_, err = k1sruntime.NewRuntime(k1sruntime.RuntimeOptions{
				Client: testClient,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("scheme is required"))
		})
	})
})
