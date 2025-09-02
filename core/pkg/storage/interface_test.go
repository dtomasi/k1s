package storage_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/dtomasi/k1s/core/pkg/storage"
	"k8s.io/apimachinery/pkg/watch"
)

var _ = Describe("Storage Interface", func() {
	Describe("SimpleVersioner", func() {

		Describe("resource version encoding/decoding", func() {
			DescribeTable("resource version conversion",
				func(input uint64, expected string) {
					encoded := storage.EncodeResourceVersion(input)
					Expect(encoded).To(Equal(expected))

					decoded, err := storage.ParseResourceVersion(encoded)
					Expect(err).NotTo(HaveOccurred())
					Expect(decoded).To(Equal(input))
				},
				Entry("zero version", uint64(0), ""),
				Entry("small version", uint64(1), "1"),
				Entry("large version", uint64(1234567890), "1234567890"),
				Entry("max uint64", uint64(18446744073709551615), "18446744073709551615"),
			)

			It("should handle empty string parsing", func() {
				decoded, err := storage.ParseResourceVersion("")
				Expect(err).NotTo(HaveOccurred())
				Expect(decoded).To(Equal(uint64(0)))
			})

			It("should return error for invalid resource version", func() {
				_, err := storage.ParseResourceVersion("invalid")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Key Generation", func() {
		Describe("GenerateKey", func() {
			It("should generate basic key without tenant", func() {
				opts := storage.KeyOptions{
					Resource: "pods",
					Name:     "my-pod",
				}
				key := storage.GenerateKey(opts)
				Expect(key).To(Equal("k1s/default/pods/my-pod"))
			})

			It("should generate key with custom namespace", func() {
				opts := storage.KeyOptions{
					Namespace: "kube-system",
					Resource:  "configmaps",
					Name:      "my-config",
				}
				key := storage.GenerateKey(opts)
				Expect(key).To(Equal("k1s/kube-system/configmaps/my-config"))
			})

			It("should generate key with tenant", func() {
				tenant := &storage.TenantConfig{
					ID:     "tenant1",
					Prefix: "custom",
				}
				opts := storage.KeyOptions{
					Tenant:   tenant,
					Resource: "secrets",
					Name:     "my-secret",
				}
				key := storage.GenerateKey(opts)
				Expect(key).To(Equal("custom/tenant1/default/secrets/my-secret"))
			})

			It("should generate key with tenant namespace", func() {
				tenant := &storage.TenantConfig{
					ID:        "tenant1",
					Namespace: "tenant-ns",
				}
				opts := storage.KeyOptions{
					Tenant:   tenant,
					Resource: "services",
					Name:     "my-service",
				}
				key := storage.GenerateKey(opts)
				Expect(key).To(Equal("k1s/tenant1/tenant-ns/services/my-service"))
			})
		})

		Describe("GenerateListKey", func() {
			It("should generate list key without name", func() {
				opts := storage.KeyOptions{
					Namespace: "default",
					Resource:  "pods",
					Name:      "should-be-ignored",
				}
				key := storage.GenerateListKey(opts)
				Expect(key).To(Equal("k1s/default/pods"))
			})
		})

		Describe("ParseKey", func() {
			DescribeTable("key parsing",
				func(key, expectedTenant, expectedNamespace, expectedResource, expectedName string, shouldErr bool) {
					tenant, namespace, resource, name, err := storage.ParseKey(key)
					if shouldErr {
						Expect(err).To(HaveOccurred())
					} else {
						Expect(err).NotTo(HaveOccurred())
						Expect(tenant).To(Equal(expectedTenant))
						Expect(namespace).To(Equal(expectedNamespace))
						Expect(resource).To(Equal(expectedResource))
						Expect(name).To(Equal(expectedName))
					}
				},
				Entry("basic key", "k1s/default/pods/my-pod", "", "default", "pods", "my-pod", false),
				Entry("namespace only", "k1s/kube-system", "", "kube-system", "", "", false),
				Entry("namespace and resource", "k1s/default/configmaps", "", "default", "configmaps", "", false),
				Entry("tenant key", "k1s/tenant1/default/secrets/my-secret", "tenant1", "default", "secrets", "my-secret", false),
				Entry("invalid key", "invalid", "", "", "", "", true),
				Entry("too few parts", "k1s", "", "", "", "", true),
			)
		})

		Describe("ValidateKey", func() {
			It("should validate key without tenant restriction", func() {
				err := storage.ValidateKey("k1s/default/pods/my-pod", "")
				Expect(err).NotTo(HaveOccurred())
			})

			It("should validate key with matching tenant", func() {
				err := storage.ValidateKey("k1s/tenant1/default/pods/my-pod", "tenant1")
				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject key with mismatched tenant", func() {
				err := storage.ValidateKey("k1s/tenant1/default/pods/my-pod", "tenant2")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("does not belong to tenant"))
			})

			It("should handle invalid key format", func() {
				err := storage.ValidateKey("invalid", "tenant1")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid key"))
			})
		})

		Describe("IsTenantKey", func() {
			It("should return true for any key when no tenant restriction", func() {
				Expect(storage.IsTenantKey("k1s/default/pods/my-pod", "")).To(BeTrue())
			})

			It("should return true for matching tenant", func() {
				Expect(storage.IsTenantKey("k1s/tenant1/default/pods/my-pod", "tenant1")).To(BeTrue())
			})

			It("should return false for non-matching tenant", func() {
				Expect(storage.IsTenantKey("k1s/tenant1/default/pods/my-pod", "tenant2")).To(BeFalse())
			})

			It("should return false for invalid key", func() {
				Expect(storage.IsTenantKey("invalid", "tenant1")).To(BeFalse())
			})
		})

		Describe("BuildTenantPrefix", func() {
			It("should use custom prefix if provided", func() {
				config := storage.TenantConfig{
					ID:     "tenant1",
					Prefix: "custom-prefix",
				}
				prefix := storage.BuildTenantPrefix(config)
				Expect(prefix).To(Equal("custom-prefix"))
			})

			It("should generate default prefix", func() {
				config := storage.TenantConfig{
					ID: "tenant1",
				}
				prefix := storage.BuildTenantPrefix(config)
				Expect(prefix).To(Equal("k1s:tenant1"))
			})
		})
	})

	Describe("Key Generators", func() {
		Describe("CreateKeyGenerator", func() {
			It("should create generator without tenant", func() {
				config := storage.Config{
					Namespace: "test-ns",
				}
				generator := storage.CreateKeyGenerator(config)
				key := generator("pods", "override-ns", "my-pod")
				Expect(key).To(Equal("k1s/override-ns/pods/my-pod"))
			})

			It("should create generator with tenant", func() {
				config := storage.Config{
					TenantID:  "tenant1",
					Namespace: "tenant-ns",
				}
				generator := storage.CreateKeyGenerator(config)
				key := generator("secrets", "override-ns", "my-secret")
				Expect(key).To(Equal("k1s/tenant1/override-ns/secrets/my-secret"))
			})
		})

		Describe("CreateListKeyGenerator", func() {
			It("should create list generator", func() {
				config := storage.Config{
					TenantID: "tenant1",
				}
				generator := storage.CreateListKeyGenerator(config)
				key := generator("configmaps", "kube-system")
				Expect(key).To(Equal("k1s/tenant1/kube-system/configmaps"))
			})
		})
	})

	Describe("SimpleWatch", func() {
		var simpleWatch *storage.SimpleWatch

		BeforeEach(func() {
			simpleWatch = storage.NewSimpleWatch()
		})

		AfterEach(func() {
			simpleWatch.Stop()
		})

		It("should create watch with result channel", func() {
			resultChan := simpleWatch.ResultChan()
			Expect(resultChan).NotTo(BeNil())
		})

		It("should send and receive events", func() {
			resultChan := simpleWatch.ResultChan()

			// Send an event
			go func() {
				defer GinkgoRecover()
				simpleWatch.Send(watch.Added, nil)
			}()

			// Receive the event
			Eventually(resultChan).Should(Receive(Equal(watch.Event{
				Type:   watch.Added,
				Object: nil,
			})))
		})

		It("should stop properly", func() {
			resultChan := simpleWatch.ResultChan()
			simpleWatch.Stop()
			Eventually(resultChan).Should(BeClosed())
		})
	})

	Describe("Error Types", func() {
		Describe("ContextCancelledError", func() {
			It("should create and identify context cancelled error", func() {
				originalErr := storage.ContextCancelledError{Err: nil}
				Expect(storage.IsContextCancelled(originalErr)).To(BeTrue())
			})

			It("should return false for non-context errors", func() {
				err := storage.ContextCancelledError{Err: nil}
				Expect(storage.IsContextCancelled(err)).To(BeTrue())
			})
		})
	})
})

func TestStorage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Storage Suite")
}