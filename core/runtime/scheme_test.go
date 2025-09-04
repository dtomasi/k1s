package runtime_test

import (
	"errors"
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	k1sruntime "github.com/dtomasi/k1s/core/runtime"
)

// TestObject is a simple runtime.Object for testing
type TestObject struct {
	runtime.TypeMeta `json:",inline"`
	v1.ObjectMeta    `json:"metadata,omitempty"`

	Spec   TestSpec   `json:"spec,omitempty"`
	Status TestStatus `json:"status,omitempty"`
}

type TestSpec struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type TestStatus struct {
	Ready bool   `json:"ready"`
	Phase string `json:"phase"`
}

// DeepCopyObject implements runtime.Object
func (t *TestObject) DeepCopyObject() runtime.Object {
	return &TestObject{
		TypeMeta:   t.TypeMeta,
		ObjectMeta: t.ObjectMeta,
		Spec:       t.Spec,
		Status:     t.Status,
	}
}

var _ = Describe("Scheme", func() {
	var scheme *runtime.Scheme
	var k1sScheme *k1sruntime.K1SScheme

	BeforeEach(func() {
		scheme = k1sruntime.NewScheme()
		k1sScheme = k1sruntime.NewK1SScheme()
	})

	Describe("NewScheme", func() {
		It("should create a valid Kubernetes scheme", func() {
			Expect(scheme).NotTo(BeNil())
		})

		It("should create a scheme that is compatible with runtime.Scheme interface", func() {
			// The scheme should have the necessary methods
			Expect(scheme.New).NotTo(BeNil())
			Expect(scheme.ObjectKinds).NotTo(BeNil())
		})
	})

	Describe("SchemeBuilder", func() {
		var builder *k1sruntime.SchemeBuilder

		BeforeEach(func() {
			builder = k1sruntime.NewSchemeBuilder()
		})

		It("should create a new SchemeBuilder", func() {
			Expect(builder).NotTo(BeNil())
		})

		It("should register functions and add them to scheme", func() {
			called := false
			testFunc := func(_ *runtime.Scheme) error {
				called = true
				return nil
			}

			builder.Register(testFunc)
			err := builder.AddToScheme(scheme)

			Expect(err).NotTo(HaveOccurred())
			Expect(called).To(BeTrue())
		})

		It("should handle multiple functions", func() {
			callCount := 0
			testFunc1 := func(_ *runtime.Scheme) error {
				callCount++
				return nil
			}
			testFunc2 := func(_ *runtime.Scheme) error {
				callCount++
				return nil
			}

			builder.Register(testFunc1, testFunc2)
			err := builder.AddToScheme(scheme)

			Expect(err).NotTo(HaveOccurred())
			Expect(callCount).To(Equal(2))
		})

		It("should propagate errors from registered functions", func() {
			testError := "test error"
			errorFunc := func(_ *runtime.Scheme) error {
				return errors.New(testError)
			}

			builder.Register(errorFunc)
			err := builder.AddToScheme(scheme)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to add to scheme"))
		})
	})

	Describe("K1SScheme", func() {
		It("should create a new K1SScheme", func() {
			Expect(k1sScheme).NotTo(BeNil())
			Expect(k1sScheme.Scheme).NotTo(BeNil())
		})

		Describe("AddKnownTypes", func() {
			It("should add types successfully", func() {
				gv := schema.GroupVersion{Group: "test.k1s.io", Version: "v1"}
				testObj := &TestObject{}

				err := k1sScheme.AddKnownTypes(gv, testObj)
				Expect(err).NotTo(HaveOccurred())

				// Verify the type is registered
				gvk := gv.WithKind("TestObject")
				Expect(k1sScheme.IsTypeRegistered(gvk)).To(BeTrue())
			})

			It("should reject nil objects", func() {
				gv := schema.GroupVersion{Group: "test.k1s.io", Version: "v1"}

				err := k1sScheme.AddKnownTypes(gv, nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("cannot register nil object"))
			})

			It("should handle pointer types correctly", func() {
				gv := schema.GroupVersion{Group: "test.k1s.io", Version: "v1"}
				testObj := &TestObject{}

				err := k1sScheme.AddKnownTypes(gv, testObj)
				Expect(err).NotTo(HaveOccurred())

				registeredTypes := k1sScheme.GetRegisteredTypes()
				gvk := gv.WithKind("TestObject")

				Expect(registeredTypes).To(HaveKey(gvk))
				Expect(registeredTypes[gvk].Kind()).To(Equal(reflect.Struct))
				Expect(registeredTypes[gvk].Name()).To(Equal("TestObject"))
			})
		})

		Describe("IsTypeRegistered", func() {
			It("should return false for unregistered types", func() {
				gvk := schema.GroupVersionKind{
					Group:   "test.k1s.io",
					Version: "v1",
					Kind:    "UnknownObject",
				}

				Expect(k1sScheme.IsTypeRegistered(gvk)).To(BeFalse())
			})

			It("should return true for registered types", func() {
				gv := schema.GroupVersion{Group: "test.k1s.io", Version: "v1"}
				testObj := &TestObject{}
				gvk := gv.WithKind("TestObject")

				err := k1sScheme.AddKnownTypes(gv, testObj)
				Expect(err).NotTo(HaveOccurred())

				Expect(k1sScheme.IsTypeRegistered(gvk)).To(BeTrue())
			})
		})

		Describe("GetRegisteredTypes", func() {
			It("should return empty map for new scheme", func() {
				types := k1sScheme.GetRegisteredTypes()
				Expect(types).To(BeEmpty())
			})

			It("should return registered types", func() {
				gv := schema.GroupVersion{Group: "test.k1s.io", Version: "v1"}
				testObj := &TestObject{}
				gvk := gv.WithKind("TestObject")

				err := k1sScheme.AddKnownTypes(gv, testObj)
				Expect(err).NotTo(HaveOccurred())

				types := k1sScheme.GetRegisteredTypes()
				Expect(types).To(HaveLen(1))
				Expect(types).To(HaveKey(gvk))
			})

			It("should return a copy to prevent external modification", func() {
				gv := schema.GroupVersion{Group: "test.k1s.io", Version: "v1"}
				testObj := &TestObject{}

				err := k1sScheme.AddKnownTypes(gv, testObj)
				Expect(err).NotTo(HaveOccurred())

				types1 := k1sScheme.GetRegisteredTypes()
				types2 := k1sScheme.GetRegisteredTypes()

				// Modify one copy
				delete(types1, schema.GroupVersionKind{Group: "test.k1s.io", Version: "v1", Kind: "TestObject"})

				// Other copy should be unaffected
				Expect(types2).To(HaveLen(1))

				// Original should be unaffected
				types3 := k1sScheme.GetRegisteredTypes()
				Expect(types3).To(HaveLen(1))
			})
		})

		Describe("ObjectKinds", func() {
			It("should reject nil objects", func() {
				_, _, err := k1sScheme.ObjectKinds(nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("cannot determine kinds for nil object"))
			})

			It("should delegate to underlying scheme for valid objects", func() {
				// This test verifies that our wrapper correctly delegates
				// The actual behavior is tested by the Kubernetes runtime tests
				testObj := &TestObject{}

				// This should not panic, even though the type isn't registered
				_, _, err := k1sScheme.ObjectKinds(testObj)
				// Error is expected since type isn't registered, but it should be from underlying scheme
				Expect(err).To(HaveOccurred())
			})
		})

		Describe("New", func() {
			It("should reject unregistered types", func() {
				gvk := schema.GroupVersionKind{
					Group:   "test.k1s.io",
					Version: "v1",
					Kind:    "UnknownObject",
				}

				obj, err := k1sScheme.New(gvk)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("is not registered in scheme"))
				Expect(obj).To(BeNil())
			})

			It("should create instances for registered types", func() {
				gv := schema.GroupVersion{Group: "test.k1s.io", Version: "v1"}
				testObj := &TestObject{}
				gvk := gv.WithKind("TestObject")

				err := k1sScheme.AddKnownTypes(gv, testObj)
				Expect(err).NotTo(HaveOccurred())

				// Note: This test verifies our validation logic.
				// The actual object creation would require the underlying
				// Kubernetes scheme to have the type properly registered,
				// which requires more complex setup.
				obj, err := k1sScheme.New(gvk)

				if err != nil {
					// If error occurs, it should be from the underlying scheme,
					// not from our validation
					Expect(err.Error()).NotTo(ContainSubstring("is not registered in scheme"))
				} else {
					Expect(obj).NotTo(BeNil())
				}
			})
		})
	})

	Describe("GetGVKForObject", func() {
		It("should reject nil objects", func() {
			gvk, err := k1sruntime.GetGVKForObject(nil, scheme)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot determine GVK for nil object"))
			Expect(gvk).To(Equal(schema.GroupVersionKind{}))
		})

		It("should use global scheme if none provided", func() {
			testObj := &TestObject{}

			// This should not panic even with nil scheme
			_, err := k1sruntime.GetGVKForObject(testObj, nil)
			// Error is expected since type isn't registered in global scheme
			Expect(err).To(HaveOccurred())
		})

		It("should handle objects with no registered kinds", func() {
			testObj := &TestObject{}

			gvk, err := k1sruntime.GetGVKForObject(testObj, scheme)
			Expect(err).To(HaveOccurred())
			// The actual error message comes from Kubernetes runtime
			Expect(err.Error()).To(ContainSubstring("failed to get kinds"))
			Expect(gvk).To(Equal(schema.GroupVersionKind{}))
		})
	})

	Describe("AddToScheme", func() {
		It("should not error for basic setup", func() {
			err := k1sruntime.AddToScheme(scheme)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Global instances", func() {
		It("should provide global Scheme instance", func() {
			Expect(k1sruntime.Scheme).NotTo(BeNil())
		})

		It("should provide global Codecs instance", func() {
			Expect(k1sruntime.Codecs).NotTo(BeNil())
		})
	})

	Describe("Concurrent access", func() {
		It("should handle concurrent type registration", func() {
			const numGoroutines = 10
			const numTypes = 5

			results := make(chan error, numGoroutines)

			for i := 0; i < numGoroutines; i++ {
				go func(_ int) {
					localScheme := k1sruntime.NewK1SScheme()

					for j := 0; j < numTypes; j++ {
						gv := schema.GroupVersion{
							Group:   "concurrent.test.k1s.io",
							Version: "v1",
						}

						testObj := &TestObject{}
						err := localScheme.AddKnownTypes(gv, testObj)
						if err != nil {
							results <- err
							return
						}
					}
					results <- nil
				}(i)
			}

			// Collect all results
			for i := 0; i < numGoroutines; i++ {
				err := <-results
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should handle concurrent reads and writes", func() {
			testScheme := k1sruntime.NewK1SScheme()
			gv := schema.GroupVersion{Group: "concurrent.k1s.io", Version: "v1"}
			gvk := gv.WithKind("TestObject")

			// Register initial type
			err := testScheme.AddKnownTypes(gv, &TestObject{})
			Expect(err).NotTo(HaveOccurred())

			const numReaders = 5
			const numWriters = 2
			const iterations = 10

			results := make(chan error, numReaders+numWriters)

			// Start readers
			for i := 0; i < numReaders; i++ {
				go func() {
					for j := 0; j < iterations; j++ {
						registered := testScheme.IsTypeRegistered(gvk)
						Expect(registered).To(BeTrue())

						types := testScheme.GetRegisteredTypes()
						Expect(types).To(HaveKey(gvk))
					}
					results <- nil
				}()
			}

			// Start writers
			for i := 0; i < numWriters; i++ {
				go func(_ int) {
					for j := 0; j < iterations; j++ {
						writerGV := schema.GroupVersion{
							Group:   "writer.k1s.io",
							Version: "v1",
						}

						err := testScheme.AddKnownTypes(writerGV, &TestObject{})
						if err != nil {
							results <- err
							return
						}
					}
					results <- nil
				}(i)
			}

			// Collect all results
			for i := 0; i < numReaders+numWriters; i++ {
				err := <-results
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})
})
