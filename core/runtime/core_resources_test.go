package runtime_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/runtime"

	k1sruntime "github.com/dtomasi/k1s/core/runtime"
	typesv1 "github.com/dtomasi/k1s/core/types/v1"
)

var _ = Describe("Core Resources Integration", func() {
	var scheme *runtime.Scheme

	BeforeEach(func() {
		scheme = runtime.NewScheme()
	})

	Describe("RegisterCoreResources", func() {
		It("should register all core resource types with the scheme", func() {
			err := k1sruntime.RegisterCoreResources(scheme)
			Expect(err).ToNot(HaveOccurred())

			// Test that we can create instances of each core type
			gvks := typesv1.GetAllGVKs()
			for _, gvk := range gvks {
				obj, err := scheme.New(gvk)
				Expect(err).ToNot(HaveOccurred(), "Failed to create instance of %v", gvk)
				Expect(obj).ToNot(BeNil(), "Created object should not be nil for %v", gvk)
			}
		})

		It("should register GVK to GVR mappings", func() {
			k1sruntime.RegisterCoreResourceMappings()

			// Test that mappings are registered correctly
			mappings := typesv1.GetGVKToGVRMappings()
			for gvk, expectedGVR := range mappings {
				actualGVR, err := k1sruntime.GetGVRForGVK(gvk)
				Expect(err).ToNot(HaveOccurred(), "Failed to get GVR for GVK %v", gvk)
				Expect(actualGVR).To(Equal(expectedGVR), "GVR mapping mismatch for GVK %v", gvk)
			}
		})
	})

	Describe("Core Resources Integration (with separate registries)", func() {
		It("should register all core resource types with scheme", func() {
			err := k1sruntime.RegisterCoreResources(scheme)
			Expect(err).ToNot(HaveOccurred())

			// Verify scheme registration
			gvks := typesv1.GetAllGVKs()
			for _, gvk := range gvks {
				obj, err := scheme.New(gvk)
				Expect(err).ToNot(HaveOccurred())
				Expect(obj).ToNot(BeNil())
			}
		})

		It("should register GVK/GVR mappings", func() {
			k1sruntime.RegisterCoreResourceMappings()

			// Test that mappings are registered correctly
			mappings := typesv1.GetGVKToGVRMappings()
			for gvk, expectedGVR := range mappings {
				actualGVR, err := k1sruntime.GetGVRForGVK(gvk)
				Expect(err).ToNot(HaveOccurred(), "Failed to get GVR for GVK %v", gvk)
				Expect(actualGVR).To(Equal(expectedGVR), "GVR mapping mismatch for GVK %v", gvk)
			}
		})
	})
})
