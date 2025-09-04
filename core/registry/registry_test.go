package registry_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/dtomasi/k1s/core/registry"
)

var _ = Describe("Registry", func() {
	var (
		reg registry.Registry

		// Test GVRs
		itemGVR = schema.GroupVersionResource{
			Group:    "test.k1s.io",
			Version:  "v1alpha1",
			Resource: "items",
		}

		categoryGVR = schema.GroupVersionResource{
			Group:    "test.k1s.io",
			Version:  "v1alpha1",
			Resource: "categories",
		}

		// Test GVKs
		itemGVK = schema.GroupVersionKind{
			Group:   "test.k1s.io",
			Version: "v1alpha1",
			Kind:    "Item",
		}
	)

	BeforeEach(func() {
		reg = registry.NewRegistry()
	})

	Describe("NewRegistry", func() {
		It("should create a registry with default configuration", func() {
			Expect(reg).ToNot(BeNil())
			Expect(reg.ListResources()).To(BeEmpty())
		})

		It("should create a registry with custom options", func() {
			customReg := registry.NewRegistry(
				registry.WithDefaultCategories("custom", "test"),
				registry.WithShortNameValidation(false),
				registry.WithCaseSensitiveShortNames(true),
			)
			Expect(customReg).ToNot(BeNil())
		})
	})

	Describe("RegisterResource", func() {
		It("should register a resource with complete configuration", func() {
			config := registry.ResourceConfig{
				Singular:    "item",
				Plural:      "items",
				Kind:        "Item",
				ListKind:    "ItemList",
				Namespaced:  true,
				Description: "Test item resource",
				ShortNames:  []string{"itm", "it"},
				Categories:  []string{"inventory", "all"},
				PrintColumns: []metav1.TableColumnDefinition{
					{
						Name:        "Name",
						Type:        "string",
						Format:      "name",
						Description: "Name of the item",
						Priority:    0,
					},
					{
						Name:        "Description",
						Type:        "string",
						Format:      "",
						Description: "Item description",
						Priority:    1,
					},
				},
			}

			err := reg.RegisterResource(itemGVR, config)
			Expect(err).ToNot(HaveOccurred())
			Expect(reg.IsResourceRegistered(itemGVR)).To(BeTrue())
		})

		It("should auto-generate ListKind if not provided", func() {
			config := registry.ResourceConfig{
				Singular: "item",
				Plural:   "items",
				Kind:     "Item",
			}

			err := reg.RegisterResource(itemGVR, config)
			Expect(err).ToNot(HaveOccurred())

			retrievedConfig, err := reg.GetResourceConfig(itemGVR)
			Expect(err).ToNot(HaveOccurred())
			Expect(retrievedConfig.ListKind).To(Equal("ItemList"))
		})

		It("should add default categories if none specified", func() {
			config := registry.ResourceConfig{
				Singular: "item",
				Plural:   "items",
				Kind:     "Item",
			}

			err := reg.RegisterResource(itemGVR, config)
			Expect(err).ToNot(HaveOccurred())

			retrievedConfig, err := reg.GetResourceConfig(itemGVR)
			Expect(err).ToNot(HaveOccurred())
			Expect(retrievedConfig.Categories).To(ContainElement("all"))
		})

		It("should merge default categories with user-specified ones", func() {
			config := registry.ResourceConfig{
				Singular:   "item",
				Plural:     "items",
				Kind:       "Item",
				Categories: []string{"inventory", "custom"},
			}

			err := reg.RegisterResource(itemGVR, config)
			Expect(err).ToNot(HaveOccurred())

			retrievedConfig, err := reg.GetResourceConfig(itemGVR)
			Expect(err).ToNot(HaveOccurred())
			Expect(retrievedConfig.Categories).To(ContainElements("inventory", "custom", "all"))
		})

		It("should fail if required fields are missing", func() {
			testCases := []struct {
				name   string
				config registry.ResourceConfig
			}{
				{
					name: "missing singular",
					config: registry.ResourceConfig{
						Plural: "items",
						Kind:   "Item",
					},
				},
				{
					name: "missing plural",
					config: registry.ResourceConfig{
						Singular: "item",
						Kind:     "Item",
					},
				},
				{
					name: "missing kind",
					config: registry.ResourceConfig{
						Singular: "item",
						Plural:   "items",
					},
				},
			}

			for _, tc := range testCases {
				err := reg.RegisterResource(itemGVR, tc.config)
				Expect(err).To(HaveOccurred(), "Expected error for %s", tc.name)
			}
		})

		It("should fail if resource is already registered", func() {
			config := registry.ResourceConfig{
				Singular: "item",
				Plural:   "items",
				Kind:     "Item",
			}

			err := reg.RegisterResource(itemGVR, config)
			Expect(err).ToNot(HaveOccurred())

			err = reg.RegisterResource(itemGVR, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already registered"))
		})

		It("should fail if short name conflicts", func() {
			// Register first resource with short name
			config1 := registry.ResourceConfig{
				Singular:   "item",
				Plural:     "items",
				Kind:       "Item",
				ShortNames: []string{"itm"},
			}
			err := reg.RegisterResource(itemGVR, config1)
			Expect(err).ToNot(HaveOccurred())

			// Try to register second resource with same short name
			config2 := registry.ResourceConfig{
				Singular:   "category",
				Plural:     "categories",
				Kind:       "Category",
				ShortNames: []string{"itm"}, // Conflicting short name
			}
			err = reg.RegisterResource(categoryGVR, config2)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("conflicts"))
		})
	})

	Describe("GetResourceConfig", func() {
		BeforeEach(func() {
			config := registry.ResourceConfig{
				Singular:    "item",
				Plural:      "items",
				Kind:        "Item",
				ListKind:    "ItemList",
				Namespaced:  true,
				Description: "Test item resource",
				ShortNames:  []string{"itm"},
				Categories:  []string{"inventory"},
			}
			err := reg.RegisterResource(itemGVR, config)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return resource configuration for registered resource", func() {
			config, err := reg.GetResourceConfig(itemGVR)
			Expect(err).ToNot(HaveOccurred())
			Expect(config.Singular).To(Equal("item"))
			Expect(config.Plural).To(Equal("items"))
			Expect(config.Kind).To(Equal("Item"))
			Expect(config.Namespaced).To(BeTrue())
		})

		It("should fail for unregistered resource", func() {
			unregisteredGVR := schema.GroupVersionResource{
				Group:    "unknown.k1s.io",
				Version:  "v1",
				Resource: "unknown",
			}

			_, err := reg.GetResourceConfig(unregisteredGVR)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not registered"))
		})
	})

	Describe("ListResources", func() {
		It("should return empty list for new registry", func() {
			resources := reg.ListResources()
			Expect(resources).To(BeEmpty())
		})

		It("should return all registered resources", func() {
			// Register multiple resources
			itemConfig := registry.ResourceConfig{
				Singular: "item",
				Plural:   "items",
				Kind:     "Item",
			}
			categoryConfig := registry.ResourceConfig{
				Singular: "category",
				Plural:   "categories",
				Kind:     "Category",
			}

			err := reg.RegisterResource(itemGVR, itemConfig)
			Expect(err).ToNot(HaveOccurred())

			err = reg.RegisterResource(categoryGVR, categoryConfig)
			Expect(err).ToNot(HaveOccurred())

			resources := reg.ListResources()
			Expect(resources).To(HaveLen(2))
			Expect(resources).To(ContainElements(itemGVR, categoryGVR))
		})
	})

	Describe("Short Name Resolution", func() {
		BeforeEach(func() {
			config := registry.ResourceConfig{
				Singular:   "item",
				Plural:     "items",
				Kind:       "Item",
				ShortNames: []string{"itm", "it", "items"},
			}
			err := reg.RegisterResource(itemGVR, config)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should resolve short names to GVR", func() {
			shortNames := []string{"itm", "it", "items"}
			for _, shortName := range shortNames {
				gvr, err := reg.GetGVRForShortName(shortName)
				Expect(err).ToNot(HaveOccurred())
				Expect(gvr).To(Equal(itemGVR))
			}
		})

		It("should handle case insensitive short names by default", func() {
			gvr, err := reg.GetGVRForShortName("ITM")
			Expect(err).ToNot(HaveOccurred())
			Expect(gvr).To(Equal(itemGVR))
		})

		It("should fail for unregistered short name", func() {
			_, err := reg.GetGVRForShortName("unknown")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not registered"))
		})

		It("should handle case sensitive short names when configured", func() {
			caseSensitiveReg := registry.NewRegistry(
				registry.WithCaseSensitiveShortNames(true),
			)

			config := registry.ResourceConfig{
				Singular:   "item",
				Plural:     "items",
				Kind:       "Item",
				ShortNames: []string{"Item"},
			}
			err := caseSensitiveReg.RegisterResource(itemGVR, config)
			Expect(err).ToNot(HaveOccurred())

			// Exact case should work
			gvr, err := caseSensitiveReg.GetGVRForShortName("Item")
			Expect(err).ToNot(HaveOccurred())
			Expect(gvr).To(Equal(itemGVR))

			// Different case should fail
			_, err = caseSensitiveReg.GetGVRForShortName("item")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Category Management", func() {
		BeforeEach(func() {
			itemConfig := registry.ResourceConfig{
				Singular:   "item",
				Plural:     "items",
				Kind:       "Item",
				Categories: []string{"inventory", "custom"},
			}
			categoryConfig := registry.ResourceConfig{
				Singular:   "category",
				Plural:     "categories",
				Kind:       "Category",
				Categories: []string{"inventory"},
			}

			err := reg.RegisterResource(itemGVR, itemConfig)
			Expect(err).ToNot(HaveOccurred())

			err = reg.RegisterResource(categoryGVR, categoryConfig)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return resources by category", func() {
			inventoryResources := reg.GetGVRsForCategory("inventory")
			Expect(inventoryResources).To(HaveLen(2))
			Expect(inventoryResources).To(ContainElements(itemGVR, categoryGVR))

			customResources := reg.GetGVRsForCategory("custom")
			Expect(customResources).To(HaveLen(1))
			Expect(customResources).To(ContainElement(itemGVR))

			allResources := reg.GetGVRsForCategory("all")
			Expect(allResources).To(HaveLen(2)) // Both have default "all" category
		})

		It("should return nil for non-existent category", func() {
			resources := reg.GetGVRsForCategory("nonexistent")
			Expect(resources).To(BeNil())
		})
	})

	Describe("GVK/GVR Conversion", func() {
		BeforeEach(func() {
			config := registry.ResourceConfig{
				Singular: "item",
				Plural:   "items",
				Kind:     "Item",
			}
			err := reg.RegisterResource(itemGVR, config)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should convert GVR to GVK", func() {
			gvk, err := reg.GetGVKForGVR(itemGVR)
			Expect(err).ToNot(HaveOccurred())
			Expect(gvk).To(Equal(itemGVK))
		})

		It("should convert GVK to GVR", func() {
			gvr, err := reg.GetGVRForGVK(itemGVK)
			Expect(err).ToNot(HaveOccurred())
			Expect(gvr).To(Equal(itemGVR))
		})

		It("should fail conversion for unregistered GVR", func() {
			unregisteredGVR := schema.GroupVersionResource{
				Group:    "unknown.k1s.io",
				Version:  "v1",
				Resource: "unknown",
			}

			_, err := reg.GetGVKForGVR(unregisteredGVR)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mapping not found"))
		})

		It("should fail conversion for unregistered GVK", func() {
			unregisteredGVK := schema.GroupVersionKind{
				Group:   "unknown.k1s.io",
				Version: "v1",
				Kind:    "Unknown",
			}

			_, err := reg.GetGVRForGVK(unregisteredGVK)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mapping not found"))
		})
	})

	Describe("Resource Status", func() {
		It("should check if resource is registered", func() {
			Expect(reg.IsResourceRegistered(itemGVR)).To(BeFalse())

			config := registry.ResourceConfig{
				Singular: "item",
				Plural:   "items",
				Kind:     "Item",
			}
			err := reg.RegisterResource(itemGVR, config)
			Expect(err).ToNot(HaveOccurred())

			Expect(reg.IsResourceRegistered(itemGVR)).To(BeTrue())
		})
	})

	Describe("UnregisterResource", func() {
		BeforeEach(func() {
			config := registry.ResourceConfig{
				Singular:   "item",
				Plural:     "items",
				Kind:       "Item",
				ShortNames: []string{"itm"},
				Categories: []string{"inventory"},
			}
			err := reg.RegisterResource(itemGVR, config)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should unregister a resource completely", func() {
			Expect(reg.IsResourceRegistered(itemGVR)).To(BeTrue())

			err := reg.UnregisterResource(itemGVR)
			Expect(err).ToNot(HaveOccurred())

			Expect(reg.IsResourceRegistered(itemGVR)).To(BeFalse())

			// Check short name is also removed
			_, err = reg.GetGVRForShortName("itm")
			Expect(err).To(HaveOccurred())

			// Check category is cleaned up
			inventoryResources := reg.GetGVRsForCategory("inventory")
			Expect(inventoryResources).To(BeNil())

			// Check GVK mapping is removed
			_, err = reg.GetGVKForGVR(itemGVR)
			Expect(err).To(HaveOccurred())
		})

		It("should fail to unregister non-existent resource", func() {
			unregisteredGVR := schema.GroupVersionResource{
				Group:    "unknown.k1s.io",
				Version:  "v1",
				Resource: "unknown",
			}

			err := reg.UnregisterResource(unregisteredGVR)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not registered"))
		})
	})

	Describe("Thread Safety", func() {
		It("should handle concurrent registration safely", func() {
			done := make(chan bool)

			// Start multiple goroutines registering resources
			for i := 0; i < 10; i++ {
				go func(index int) {
					defer GinkgoRecover()
					gvr := schema.GroupVersionResource{
						Group:    "test.k1s.io",
						Version:  "v1alpha1",
						Resource: fmt.Sprintf("resource%d", index),
					}
					config := registry.ResourceConfig{
						Singular: fmt.Sprintf("resource%d", index),
						Plural:   fmt.Sprintf("resource%ds", index),
						Kind:     fmt.Sprintf("Resource%d", index),
					}

					err := reg.RegisterResource(gvr, config)
					Expect(err).ToNot(HaveOccurred())
					done <- true
				}(i)
			}

			// Wait for all goroutines to complete
			for i := 0; i < 10; i++ {
				<-done
			}

			// Verify all resources were registered
			resources := reg.ListResources()
			Expect(resources).To(HaveLen(10))
		})
	})
})

var _ = Describe("Default Print Columns", func() {
	It("should provide default print columns", func() {
		columns := registry.GetDefaultPrintColumns()
		Expect(columns).To(HaveLen(2))
		Expect(columns[0].Name).To(Equal("Name"))
		Expect(columns[1].Name).To(Equal("Age"))
	})

	It("should provide default print columns with namespace", func() {
		columns := registry.GetDefaultPrintColumnsWithNamespace()
		Expect(columns).To(HaveLen(3))
		Expect(columns[0].Name).To(Equal("Namespace"))
		Expect(columns[1].Name).To(Equal("Name"))
		Expect(columns[2].Name).To(Equal("Age"))
	})
})
