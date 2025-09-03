package storage_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"

	k1sstorage "github.com/dtomasi/k1s/core/pkg/storage"
)

// mockBackend implements the Backend interface for testing
type mockBackend struct {
	name string
}

func newMockBackend(name string) *mockBackend {
	return &mockBackend{name: name}
}

func (m *mockBackend) Name() string {
	return m.name
}

func (m *mockBackend) Versioner() storage.Versioner {
	return k1sstorage.SimpleVersioner{}
}

func (m *mockBackend) Create(ctx context.Context, key string, obj, out runtime.Object, ttl uint64) error {
	return nil
}

func (m *mockBackend) Delete(ctx context.Context, key string, out runtime.Object,
	preconditions *storage.Preconditions, validateDeletion storage.ValidateObjectFunc,
	cachedExistingObject runtime.Object) error {
	return nil
}

func (m *mockBackend) Watch(ctx context.Context, key string, opts storage.ListOptions) (watch.Interface, error) {
	return k1sstorage.NewSimpleWatch(), nil
}

func (m *mockBackend) Get(ctx context.Context, key string, opts storage.GetOptions, objPtr runtime.Object) error {
	return nil
}

func (m *mockBackend) List(ctx context.Context, key string, opts storage.ListOptions, listObj runtime.Object) error {
	return nil
}

func (m *mockBackend) Close() error {
	return nil
}

func (m *mockBackend) Compact(ctx context.Context) error {
	return nil
}

func (m *mockBackend) Count(ctx context.Context, key string) (int64, error) {
	return 0, nil
}

var _ = Describe("SimpleFactory", func() {
	var (
		mockBack k1sstorage.Backend
		factory  k1sstorage.Factory
	)

	BeforeEach(func() {
		mockBack = newMockBackend("test-backend")
		factory = k1sstorage.NewSimpleFactory(mockBack)
	})

	Describe("Factory Creation", func() {
		It("should create a factory from a backend", func() {
			Expect(factory).NotTo(BeNil())
		})

		It("should report supported backends", func() {
			supportedBackends := factory.SupportedBackends()
			Expect(supportedBackends).To(HaveLen(1))
			Expect(supportedBackends[0]).To(Equal("test-backend"))
		})
	})

	Describe("Storage Creation", func() {
		It("should create storage interface", func() {
			config := k1sstorage.Config{
				TenantID:  "test-tenant",
				Namespace: "test-namespace",
			}

			storageInterface, err := factory.Create(config)
			Expect(err).NotTo(HaveOccurred())
			Expect(storageInterface).NotTo(BeNil())
		})

		It("should create backend", func() {
			config := k1sstorage.Config{
				TenantID:  "test-tenant",
				Namespace: "test-namespace",
			}

			backend, err := factory.CreateBackend(config)
			Expect(err).NotTo(HaveOccurred())
			Expect(backend).NotTo(BeNil())
			Expect(backend.Name()).To(Equal("test-backend"))
		})

		It("should return the same backend instance", func() {
			config1 := k1sstorage.Config{TenantID: "tenant1"}
			config2 := k1sstorage.Config{TenantID: "tenant2"}

			backend1, err := factory.CreateBackend(config1)
			Expect(err).NotTo(HaveOccurred())

			backend2, err := factory.CreateBackend(config2)
			Expect(err).NotTo(HaveOccurred())

			// Should return the same instance since we're wrapping
			Expect(backend1).To(BeIdenticalTo(backend2))
		})
	})

	Describe("Backend Operations", func() {
		It("should support all backend operations", func() {
			config := k1sstorage.Config{}
			backend, err := factory.CreateBackend(config)
			Expect(err).NotTo(HaveOccurred())

			ctx := context.Background()

			// Test backend-specific operations
			count, err := backend.Count(ctx, "test-key")
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(int64(0)))

			err = backend.Compact(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = backend.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should support storage interface operations", func() {
			config := k1sstorage.Config{}
			storageInterface, err := factory.Create(config)
			Expect(err).NotTo(HaveOccurred())

			ctx := context.Background()

			// Test storage interface operations
			versioner := storageInterface.Versioner()
			Expect(versioner).NotTo(BeNil())

			err = storageInterface.Get(ctx, "test-key", storage.GetOptions{}, &runtime.Unknown{})
			Expect(err).NotTo(HaveOccurred())

			err = storageInterface.List(ctx, "test-key", storage.ListOptions{}, &runtime.Unknown{})
			Expect(err).NotTo(HaveOccurred())

			watchInterface, err := storageInterface.Watch(ctx, "test-key", storage.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(watchInterface).NotTo(BeNil())
			watchInterface.Stop()
		})
	})

	Describe("Multiple Factories", func() {
		It("should handle different backend types", func() {
			memoryMock := newMockBackend("memory")
			boltMock := newMockBackend("bolt")

			memoryFactory := k1sstorage.NewSimpleFactory(memoryMock)
			boltFactory := k1sstorage.NewSimpleFactory(boltMock)

			memoryBackends := memoryFactory.SupportedBackends()
			boltBackends := boltFactory.SupportedBackends()

			Expect(memoryBackends).To(ContainElement("memory"))
			Expect(boltBackends).To(ContainElement("bolt"))
		})

		It("should create independent factories", func() {
			backend1 := newMockBackend("backend1")
			backend2 := newMockBackend("backend2")

			factory1 := k1sstorage.NewSimpleFactory(backend1)
			factory2 := k1sstorage.NewSimpleFactory(backend2)

			config := k1sstorage.Config{}

			result1, err := factory1.CreateBackend(config)
			Expect(err).NotTo(HaveOccurred())
			Expect(result1.Name()).To(Equal("backend1"))

			result2, err := factory2.CreateBackend(config)
			Expect(err).NotTo(HaveOccurred())
			Expect(result2.Name()).To(Equal("backend2"))

			Expect(result1).NotTo(BeIdenticalTo(result2))
		})
	})
})
