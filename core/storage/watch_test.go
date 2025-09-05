package storage_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/dtomasi/k1s/core/storage"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

var _ = Describe("Watch", func() {
	Describe("SimpleWatch", func() {
		It("should create and manage watch", func() {
			w := storage.NewSimpleWatch()
			Expect(w).NotTo(BeNil())

			// Get result channel
			resultChan := w.ResultChan()
			Expect(resultChan).NotTo(BeNil())

			// Send an event
			obj := &corev1.ConfigMap{}
			w.Send(watch.Added, obj)

			// Should receive the event
			Eventually(resultChan).Should(Receive())

			// Stop watch
			w.Stop()
		})

		It("should handle context cancellation error", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			err := storage.NewContextCancelledError(ctx)
			Expect(storage.IsContextCancelled(err)).To(BeTrue())
		})
	})
})
