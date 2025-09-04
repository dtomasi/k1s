package storage_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/dtomasi/k1s/core/storage"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Utils", func() {
	Describe("BuildKey", func() {
		It("should build key from multiple components", func() {
			result := storage.BuildKey("namespace", "pods", "my-pod")
			Expect(result).To(Equal("namespace/pods/my-pod"))
		})

		It("should handle single component", func() {
			result := storage.BuildKey("test")
			Expect(result).To(Equal("test"))
		})

		It("should return empty string for no components", func() {
			result := storage.BuildKey()
			Expect(result).To(Equal(""))
		})

		It("should filter out empty components", func() {
			result := storage.BuildKey("namespace", "", "pods", "", "my-pod")
			Expect(result).To(Equal("namespace/pods/my-pod"))
		})

		It("should return empty string when all components are empty", func() {
			result := storage.BuildKey("", "", "")
			Expect(result).To(Equal(""))
		})
	})

	Describe("SimpleVersioner", func() {
		var versioner storage.SimpleVersioner
		var obj *corev1.ConfigMap

		BeforeEach(func() {
			versioner = storage.SimpleVersioner{}
			obj = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			}
		})

		It("should update object resource version", func() {
			err := versioner.UpdateObject(obj, 123)
			Expect(err).NotTo(HaveOccurred())
			Expect(obj.GetResourceVersion()).To(Equal("123"))
		})
	})
})
