package v1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "github.com/dtomasi/k1s/core/types/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("Register", func() {
	var scheme *runtime.Scheme

	BeforeEach(func() {
		scheme = runtime.NewScheme()
	})

	Describe("AddToScheme", func() {
		It("should register all core types", func() {
			err := v1.AddToScheme(scheme)
			Expect(err).NotTo(HaveOccurred())

			// Verify some core types are registered
			gvks := v1.GetAllGVKs()
			Expect(len(gvks)).To(BeNumerically(">", 0))
		})
	})

	Describe("Resource and Kind functions", func() {
		It("should return valid resource names", func() {
			resource := v1.Resource("configmaps")
			Expect(resource.Resource).To(Equal("configmaps"))
		})

		It("should return valid kind names", func() {
			kind := v1.Kind("ConfigMap")
			Expect(kind.Kind).To(Equal("ConfigMap"))
		})
	})
})
