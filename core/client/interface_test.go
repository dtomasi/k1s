package client_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/dtomasi/k1s/core/client"
)

var _ = Describe("Interface Utilities", func() {
	var testItem *TestItem

	BeforeEach(func() {
		testItem = &TestItem{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-item",
				Namespace: "test-namespace",
				UID:       "test-uid-12345",
			},
		}
	})

	Describe("ObjectKey", func() {
		It("should create ObjectKey from object", func() {
			key := client.ObjectKeyFromObject(testItem)
			Expect(key.Name).To(Equal("test-item"))
			Expect(key.Namespace).To(Equal("test-namespace"))
		})

		It("should create ObjectKey from UID", func() {
			uid := types.UID("test-uid-12345")
			key := client.ObjectKeyFromUID(uid)
			Expect(key.Name).To(Equal("test-uid-12345"))
			Expect(key.Namespace).To(BeEmpty())
		})

		It("should format ObjectKey as string with namespace", func() {
			key := client.ObjectKey{
				Namespace: "test-namespace",
				Name:      "test-item",
			}
			Expect(key.String()).To(Equal("test-namespace/test-item"))
		})

		It("should format ObjectKey as string without namespace", func() {
			key := client.ObjectKey{
				Name: "test-item",
			}
			Expect(key.String()).To(Equal("test-item"))
		})
	})

	Describe("List Options", func() {
		Describe("MatchingLabels", func() {
			It("should create MatchingLabelsSelector from labels map", func() {
				labels := map[string]string{
					"app":     "test-app",
					"version": "v1.0.0",
				}

				selector := client.MatchingLabels(labels)

				listOpts := &client.ListOptions{}
				selector.ApplyToList(listOpts)

				Expect(listOpts.LabelSelector.MatchLabels).To(Equal(labels))
			})

			It("should apply to watch options", func() {
				labels := map[string]string{
					"environment": "production",
				}

				selector := client.MatchingLabels(labels)

				watchOpts := &client.WatchOptions{}
				selector.ApplyToWatch(watchOpts)

				Expect(watchOpts.LabelSelector.MatchLabels).To(Equal(labels))
			})
		})

		Describe("MatchingFields", func() {
			It("should create MatchingFieldsSelector from fields map", func() {
				fields := map[string]string{
					"metadata.name":      "test-item",
					"metadata.namespace": "default",
				}

				selector := client.MatchingFields(fields)

				listOpts := &client.ListOptions{}
				selector.ApplyToList(listOpts)

				// Should contain both field selectors separated by comma
				Expect(listOpts.FieldSelector).To(ContainSubstring("metadata.name=test-item"))
				Expect(listOpts.FieldSelector).To(ContainSubstring("metadata.namespace=default"))
				Expect(listOpts.FieldSelector).To(ContainSubstring(","))
			})

			It("should create single field selector", func() {
				fields := map[string]string{
					"spec.status": "Available",
				}

				selector := client.MatchingFields(fields)

				listOpts := &client.ListOptions{}
				selector.ApplyToList(listOpts)

				Expect(listOpts.FieldSelector).To(Equal("spec.status=Available"))
			})

			It("should apply to watch options", func() {
				fields := map[string]string{
					"status.phase": "Running",
				}

				selector := client.MatchingFields(fields)

				watchOpts := &client.WatchOptions{}
				selector.ApplyToWatch(watchOpts)

				Expect(watchOpts.FieldSelector).To(Equal("status.phase=Running"))
			})

			It("should handle empty fields map", func() {
				fields := map[string]string{}

				selector := client.MatchingFields(fields)

				listOpts := &client.ListOptions{}
				selector.ApplyToList(listOpts)

				Expect(listOpts.FieldSelector).To(BeEmpty())
			})
		})

		Describe("InNamespace", func() {
			It("should create InNamespaceSelector", func() {
				namespace := "kube-system"

				selector := client.InNamespace(namespace)

				listOpts := &client.ListOptions{}
				selector.ApplyToList(listOpts)

				Expect(listOpts.Namespace).To(Equal(namespace))
			})

			It("should apply to watch options", func() {
				namespace := "monitoring"

				selector := client.InNamespace(namespace)

				watchOpts := &client.WatchOptions{}
				selector.ApplyToWatch(watchOpts)

				Expect(watchOpts.Namespace).To(Equal(namespace))
			})
		})
	})

	Describe("IgnoreNotFound", func() {
		It("should return nil for not found errors", func() {
			err := errors.New("not found")
			result := client.IgnoreNotFound(err)
			Expect(result).To(BeNil())
		})

		It("should return original error for other errors", func() {
			err := errors.New("some other error")
			result := client.IgnoreNotFound(err)
			Expect(result).To(Equal(err))
		})

		It("should return nil when error is nil", func() {
			var err error = nil
			result := client.IgnoreNotFound(err)
			Expect(result).To(BeNil())
		})
	})

	Describe("Options Structures", func() {
		It("should initialize GetOptions properly", func() {
			opts := &client.GetOptions{
				Raw: &metav1.GetOptions{
					ResourceVersion: "12345",
				},
			}

			Expect(opts.Raw.ResourceVersion).To(Equal("12345"))
		})

		It("should initialize CreateOptions properly", func() {
			opts := &client.CreateOptions{
				DryRun:       []string{"All"},
				FieldManager: "test-manager",
				Raw: &metav1.CreateOptions{
					DryRun: []string{"All"},
				},
			}

			Expect(opts.DryRun).To(Equal([]string{"All"}))
			Expect(opts.FieldManager).To(Equal("test-manager"))
			Expect(opts.Raw.DryRun).To(Equal([]string{"All"}))
		})

		It("should initialize UpdateOptions properly", func() {
			opts := &client.UpdateOptions{
				DryRun:       []string{"Server"},
				FieldManager: "kubectl",
			}

			Expect(opts.DryRun).To(Equal([]string{"Server"}))
			Expect(opts.FieldManager).To(Equal("kubectl"))
		})

		It("should initialize DeleteOptions properly", func() {
			gracePeriod := int64(30)
			propagationPolicy := metav1.DeletePropagationBackground

			opts := &client.DeleteOptions{
				GracePeriodSeconds: &gracePeriod,
				PropagationPolicy:  &propagationPolicy,
				Preconditions: &metav1.Preconditions{
					UID: &testItem.UID,
				},
			}

			Expect(*opts.GracePeriodSeconds).To(Equal(int64(30)))
			Expect(*opts.PropagationPolicy).To(Equal(metav1.DeletePropagationBackground))
			Expect(*opts.Preconditions.UID).To(Equal(testItem.UID))
		})

		It("should initialize PatchOptions properly", func() {
			force := true
			opts := &client.PatchOptions{
				DryRun:       []string{"All"},
				Force:        &force,
				FieldManager: "kubectl-patch",
			}

			Expect(opts.DryRun).To(Equal([]string{"All"}))
			Expect(*opts.Force).To(BeTrue())
			Expect(opts.FieldManager).To(Equal("kubectl-patch"))
		})

		It("should initialize WatchOptions properly", func() {
			opts := &client.WatchOptions{
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "test",
					},
				},
				FieldSelector: "metadata.name=test",
				Namespace:     "default",
				Raw: &metav1.ListOptions{
					TimeoutSeconds: &[]int64{300}[0],
				},
			}

			Expect(opts.LabelSelector.MatchLabels["app"]).To(Equal("test"))
			Expect(opts.FieldSelector).To(Equal("metadata.name=test"))
			Expect(opts.Namespace).To(Equal("default"))
			Expect(*opts.Raw.TimeoutSeconds).To(Equal(int64(300)))
		})
	})
})
