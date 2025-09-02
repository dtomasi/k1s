package runtime_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime/schema"

	k1sruntime "github.com/dtomasi/k1s/core/pkg/runtime"
)

var _ = Describe("GVK Utilities", func() {
	var mapper *k1sruntime.GVKMapper

	BeforeEach(func() {
		mapper = k1sruntime.NewGVKMapper()
	})

	Describe("GVKMapper", func() {
		Describe("NewGVKMapper", func() {
			It("should create a new GVKMapper", func() {
				Expect(mapper).NotTo(BeNil())
			})
		})

		Describe("RegisterMapping", func() {
			It("should register bidirectional mapping", func() {
				gvk := schema.GroupVersionKind{Group: "test.k1s.io", Version: "v1", Kind: "Item"}
				gvr := schema.GroupVersionResource{Group: "test.k1s.io", Version: "v1", Resource: "items"}

				mapper.RegisterMapping(gvk, gvr)

				// Test GVK -> GVR
				resultGVR, err := mapper.ResourceFor(gvk)
				Expect(err).NotTo(HaveOccurred())
				Expect(resultGVR).To(Equal(gvr))

				// Test GVR -> GVK
				resultGVK, err := mapper.KindFor(gvr)
				Expect(err).NotTo(HaveOccurred())
				Expect(resultGVK).To(Equal(gvk))
			})

			It("should index by kind name", func() {
				gvk := schema.GroupVersionKind{Group: "test.k1s.io", Version: "v1", Kind: "Item"}
				gvr := schema.GroupVersionResource{Group: "test.k1s.io", Version: "v1", Resource: "items"}

				mapper.RegisterMapping(gvk, gvr)

				kinds := mapper.KindsByName("item")
				Expect(kinds).To(HaveLen(1))
				Expect(kinds[0]).To(Equal(gvk))
			})

			It("should handle multiple kinds with same name", func() {
				gvk1 := schema.GroupVersionKind{Group: "test.k1s.io", Version: "v1", Kind: "Item"}
				gvr1 := schema.GroupVersionResource{Group: "test.k1s.io", Version: "v1", Resource: "items"}

				gvk2 := schema.GroupVersionKind{Group: "test.k1s.io", Version: "v2", Kind: "Item"}
				gvr2 := schema.GroupVersionResource{Group: "test.k1s.io", Version: "v2", Resource: "items"}

				mapper.RegisterMapping(gvk1, gvr1)
				mapper.RegisterMapping(gvk2, gvr2)

				kinds := mapper.KindsByName("item")
				Expect(kinds).To(HaveLen(2))
				Expect(kinds).To(ContainElements(gvk1, gvk2))
			})
		})

		Describe("KindFor", func() {
			It("should return error for unregistered resource", func() {
				gvr := schema.GroupVersionResource{Group: "unknown", Version: "v1", Resource: "unknown"}

				gvk, err := mapper.KindFor(gvr)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no kind registered"))
				Expect(gvk).To(Equal(schema.GroupVersionKind{}))
			})

			It("should return kind for registered resource", func() {
				expectedGVK := schema.GroupVersionKind{Group: "test.k1s.io", Version: "v1", Kind: "Item"}
				gvr := schema.GroupVersionResource{Group: "test.k1s.io", Version: "v1", Resource: "items"}

				mapper.RegisterMapping(expectedGVK, gvr)

				gvk, err := mapper.KindFor(gvr)
				Expect(err).NotTo(HaveOccurred())
				Expect(gvk).To(Equal(expectedGVK))
			})
		})

		Describe("ResourceFor", func() {
			It("should return error for unregistered kind", func() {
				gvk := schema.GroupVersionKind{Group: "unknown", Version: "v1", Kind: "Unknown"}

				gvr, err := mapper.ResourceFor(gvk)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no resource registered"))
				Expect(gvr).To(Equal(schema.GroupVersionResource{}))
			})

			It("should return resource for registered kind", func() {
				gvk := schema.GroupVersionKind{Group: "test.k1s.io", Version: "v1", Kind: "Item"}
				expectedGVR := schema.GroupVersionResource{Group: "test.k1s.io", Version: "v1", Resource: "items"}

				mapper.RegisterMapping(gvk, expectedGVR)

				gvr, err := mapper.ResourceFor(gvk)
				Expect(err).NotTo(HaveOccurred())
				Expect(gvr).To(Equal(expectedGVR))
			})
		})

		Describe("KindsFor", func() {
			It("should return error for unregistered resource", func() {
				gvr := schema.GroupVersionResource{Group: "unknown", Version: "v1", Resource: "unknown"}

				gvks, err := mapper.KindsFor(gvr)
				Expect(err).To(HaveOccurred())
				Expect(gvks).To(BeNil())
			})

			It("should return kinds for registered resource", func() {
				gvk := schema.GroupVersionKind{Group: "test.k1s.io", Version: "v1", Kind: "Item"}
				gvr := schema.GroupVersionResource{Group: "test.k1s.io", Version: "v1", Resource: "items"}

				mapper.RegisterMapping(gvk, gvr)

				gvks, err := mapper.KindsFor(gvr)
				Expect(err).NotTo(HaveOccurred())
				Expect(gvks).To(HaveLen(1))
				Expect(gvks[0]).To(Equal(gvk))
			})
		})

		Describe("ResourcesFor", func() {
			It("should return error for unregistered kind", func() {
				gvk := schema.GroupVersionKind{Group: "unknown", Version: "v1", Kind: "Unknown"}

				gvrs, err := mapper.ResourcesFor(gvk)
				Expect(err).To(HaveOccurred())
				Expect(gvrs).To(BeNil())
			})

			It("should return resources for registered kind", func() {
				gvk := schema.GroupVersionKind{Group: "test.k1s.io", Version: "v1", Kind: "Item"}
				gvr := schema.GroupVersionResource{Group: "test.k1s.io", Version: "v1", Resource: "items"}

				mapper.RegisterMapping(gvk, gvr)

				gvrs, err := mapper.ResourcesFor(gvk)
				Expect(err).NotTo(HaveOccurred())
				Expect(gvrs).To(HaveLen(1))
				Expect(gvrs[0]).To(Equal(gvr))
			})
		})

		Describe("KindsByName", func() {
			It("should return nil for unknown kind name", func() {
				kinds := mapper.KindsByName("unknown")
				Expect(kinds).To(BeNil())
			})

			It("should be case insensitive", func() {
				gvk := schema.GroupVersionKind{Group: "test.k1s.io", Version: "v1", Kind: "Item"}
				gvr := schema.GroupVersionResource{Group: "test.k1s.io", Version: "v1", Resource: "items"}

				mapper.RegisterMapping(gvk, gvr)

				kinds1 := mapper.KindsByName("item")
				kinds2 := mapper.KindsByName("Item")
				kinds3 := mapper.KindsByName("ITEM")

				Expect(kinds1).To(Equal(kinds2))
				Expect(kinds2).To(Equal(kinds3))
				Expect(kinds1).To(HaveLen(1))
				Expect(kinds1[0]).To(Equal(gvk))
			})

			It("should return a copy to prevent external modification", func() {
				gvk := schema.GroupVersionKind{Group: "test.k1s.io", Version: "v1", Kind: "Item"}
				gvr := schema.GroupVersionResource{Group: "test.k1s.io", Version: "v1", Resource: "items"}

				mapper.RegisterMapping(gvk, gvr)

				kinds1 := mapper.KindsByName("item")
				kinds2 := mapper.KindsByName("item")

				// Modify first slice
				kinds1[0] = schema.GroupVersionKind{Group: "modified", Version: "v1", Kind: "Modified"}

				// Second slice should be unaffected
				Expect(kinds2[0]).To(Equal(gvk))
			})
		})

		Describe("GetAllMappings", func() {
			It("should return empty map for new mapper", func() {
				mappings := mapper.GetAllMappings()
				Expect(mappings).To(BeEmpty())
			})

			It("should return all registered mappings", func() {
				gvk1 := schema.GroupVersionKind{Group: "test.k1s.io", Version: "v1", Kind: "Item"}
				gvr1 := schema.GroupVersionResource{Group: "test.k1s.io", Version: "v1", Resource: "items"}

				gvk2 := schema.GroupVersionKind{Group: "test.k1s.io", Version: "v1", Kind: "Category"}
				gvr2 := schema.GroupVersionResource{Group: "test.k1s.io", Version: "v1", Resource: "categories"}

				mapper.RegisterMapping(gvk1, gvr1)
				mapper.RegisterMapping(gvk2, gvr2)

				mappings := mapper.GetAllMappings()
				Expect(mappings).To(HaveLen(2))
				Expect(mappings[gvk1]).To(Equal(gvr1))
				Expect(mappings[gvk2]).To(Equal(gvr2))
			})

			It("should return a copy to prevent external modification", func() {
				gvk := schema.GroupVersionKind{Group: "test.k1s.io", Version: "v1", Kind: "Item"}
				gvr := schema.GroupVersionResource{Group: "test.k1s.io", Version: "v1", Resource: "items"}

				mapper.RegisterMapping(gvk, gvr)

				mappings1 := mapper.GetAllMappings()
				mappings2 := mapper.GetAllMappings()

				// Modify first map
				delete(mappings1, gvk)

				// Second map should be unaffected
				Expect(mappings2).To(HaveKey(gvk))
				Expect(mappings2[gvk]).To(Equal(gvr))
			})
		})
	})

	Describe("Global functions", func() {
		var testGVK schema.GroupVersionKind
		var testGVR schema.GroupVersionResource

		BeforeEach(func() {
			testGVK = schema.GroupVersionKind{Group: "global.test.k1s.io", Version: "v1", Kind: "GlobalTest"}
			testGVR = schema.GroupVersionResource{Group: "global.test.k1s.io", Version: "v1", Resource: "globaltests"}
		})

		Describe("GetGVKForObject", func() {
			It("should handle nil object", func() {
				scheme := k1sruntime.NewScheme()
				gvk, err := k1sruntime.GetGVKForObject(nil, scheme)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("cannot determine GVK for nil object"))
				Expect(gvk).To(Equal(schema.GroupVersionKind{}))
			})

			It("should use global scheme when scheme is nil", func() {
				testObj := &TestObject{}

				// Should use global scheme - won't error due to nil scheme
				_, err := k1sruntime.GetGVKForObject(testObj, nil)
				// May error because object not registered, but not due to nil scheme
				if err != nil {
					Expect(err.Error()).NotTo(ContainSubstring("nil scheme"))
				}
			})
		})

		Describe("GetGVRForGVK", func() {
			It("should use global mapper", func() {
				// Register in global mapper
				k1sruntime.RegisterGlobalMapping(testGVK, testGVR)

				gvr, err := k1sruntime.GetGVRForGVK(testGVK)
				Expect(err).NotTo(HaveOccurred())
				Expect(gvr).To(Equal(testGVR))
			})

			It("should return error for unregistered GVK", func() {
				unknownGVK := schema.GroupVersionKind{Group: "unknown", Version: "v1", Kind: "Unknown"}

				gvr, err := k1sruntime.GetGVRForGVK(unknownGVK)
				Expect(err).To(HaveOccurred())
				Expect(gvr).To(Equal(schema.GroupVersionResource{}))
			})
		})

		Describe("GetGVKForGVR", func() {
			It("should use global mapper", func() {
				// Register in global mapper
				k1sruntime.RegisterGlobalMapping(testGVK, testGVR)

				gvk, err := k1sruntime.GetGVKForGVR(testGVR)
				Expect(err).NotTo(HaveOccurred())
				Expect(gvk).To(Equal(testGVK))
			})

			It("should return error for unregistered GVR", func() {
				unknownGVR := schema.GroupVersionResource{Group: "unknown", Version: "v1", Resource: "unknowns"}

				gvk, err := k1sruntime.GetGVKForGVR(unknownGVR)
				Expect(err).To(HaveOccurred())
				Expect(gvk).To(Equal(schema.GroupVersionKind{}))
			})
		})

		Describe("RegisterGlobalMapping", func() {
			It("should register mapping in global mapper", func() {
				k1sruntime.RegisterGlobalMapping(testGVK, testGVR)

				// Verify it's registered by attempting to retrieve it
				gvr, err := k1sruntime.GetGVRForGVK(testGVK)
				Expect(err).NotTo(HaveOccurred())
				Expect(gvr).To(Equal(testGVR))

				gvk, err := k1sruntime.GetGVKForGVR(testGVR)
				Expect(err).NotTo(HaveOccurred())
				Expect(gvk).To(Equal(testGVK))
			})
		})
	})

	Describe("PluralizationHelper", func() {
		var helper *k1sruntime.PluralizationHelper

		BeforeEach(func() {
			helper = k1sruntime.NewPluralizationHelper()
		})

		Describe("NewPluralizationHelper", func() {
			It("should create a new helper with common mappings", func() {
				Expect(helper).NotTo(BeNil())

				// Test some common irregular plurals
				Expect(helper.Pluralize("child")).To(Equal("children"))
				Expect(helper.Pluralize("person")).To(Equal("people"))
				Expect(helper.Singularize("data")).To(Equal("datum"))
			})
		})

		Describe("AddMapping", func() {
			It("should add custom mappings", func() {
				helper.AddMapping("mouse", "mice")

				Expect(helper.Pluralize("mouse")).To(Equal("mice"))
				Expect(helper.Singularize("mice")).To(Equal("mouse"))
			})

			It("should be case insensitive", func() {
				helper.AddMapping("Mouse", "Mice")

				Expect(helper.Pluralize("mouse")).To(Equal("mice"))
				Expect(helper.Pluralize("MOUSE")).To(Equal("mice"))
				Expect(helper.Singularize("MICE")).To(Equal("mouse"))
			})
		})

		Describe("Pluralize", func() {
			DescribeTable("basic pluralization rules",
				func(singular, expectedPlural string) {
					result := helper.Pluralize(singular)
					Expect(result).To(Equal(expectedPlural))
				},
				Entry("regular noun", "cat", "cats"),
				Entry("noun ending in s", "bus", "buses"),
				Entry("noun ending in sh", "dish", "dishes"),
				Entry("noun ending in ch", "watch", "watches"),
				Entry("noun ending in x", "box", "boxes"),
				Entry("noun ending in z", "quiz", "quizzes"),
				Entry("noun ending in consonant+y", "city", "cities"),
				Entry("noun ending in vowel+y", "day", "days"),
				Entry("noun ending in f", "leaf", "leaves"),
				Entry("noun ending in fe", "knife", "knives"),
			)

			It("should handle custom mappings first", func() {
				helper.AddMapping("foot", "feet")
				result := helper.Pluralize("foot")
				Expect(result).To(Equal("feet"))
			})
		})

		Describe("Singularize", func() {
			DescribeTable("basic singularization rules",
				func(plural, expectedSingular string) {
					result := helper.Singularize(plural)
					Expect(result).To(Equal(expectedSingular))
				},
				Entry("regular plural", "cats", "cat"),
				Entry("plural ending in ies", "cities", "city"),
				Entry("plural ending in ves (from f)", "leaves", "leaf"),
				Entry("plural ending in ves (from fe)", "knives", "knife"),
				Entry("plural ending in es", "boxes", "box"),
				Entry("plural ending in s", "dogs", "dog"),
			)

			It("should handle custom mappings first", func() {
				helper.AddMapping("foot", "feet")
				result := helper.Singularize("feet")
				Expect(result).To(Equal("foot"))
			})

			It("should handle edge cases", func() {
				// Very short words
				result := helper.Singularize("as")
				Expect(result).To(Equal("a"))

				// Single character
				result = helper.Singularize("s")
				Expect(result).To(Equal(""))
			})
		})
	})

	Describe("AutoGenerateGVR", func() {
		It("should generate GVR from GVK using pluralization", func() {
			gvk := schema.GroupVersionKind{
				Group:   "inventory.k1s.io",
				Version: "v1alpha1",
				Kind:    "Item",
			}

			expectedGVR := schema.GroupVersionResource{
				Group:    "inventory.k1s.io",
				Version:  "v1alpha1",
				Resource: "items",
			}

			result := k1sruntime.AutoGenerateGVR(gvk)
			Expect(result).To(Equal(expectedGVR))
		})

		It("should handle complex pluralization", func() {
			gvk := schema.GroupVersionKind{
				Group:   "test.k1s.io",
				Version: "v1",
				Kind:    "Category",
			}

			expectedGVR := schema.GroupVersionResource{
				Group:    "test.k1s.io",
				Version:  "v1",
				Resource: "categories",
			}

			result := k1sruntime.AutoGenerateGVR(gvk)
			Expect(result).To(Equal(expectedGVR))
		})

		It("should handle irregular plurals from default helper", func() {
			gvk := schema.GroupVersionKind{
				Group:   "test.k1s.io",
				Version: "v1",
				Kind:    "Person",
			}

			expectedGVR := schema.GroupVersionResource{
				Group:    "test.k1s.io",
				Version:  "v1",
				Resource: "people",
			}

			result := k1sruntime.AutoGenerateGVR(gvk)
			Expect(result).To(Equal(expectedGVR))
		})
	})

	Describe("Concurrent access", func() {
		It("should handle concurrent mapper operations", func() {
			concurrentMapper := k1sruntime.NewGVKMapper()
			const numGoroutines = 10
			const numMappings = 5

			results := make(chan error, numGoroutines)

			for i := 0; i < numGoroutines; i++ {
				go func(_ int) {
					for j := 0; j < numMappings; j++ {
						gvk := schema.GroupVersionKind{
							Group:   "concurrent.test.k1s.io",
							Version: "v1",
							Kind:    "ConcurrentTest",
						}
						gvr := schema.GroupVersionResource{
							Group:    "concurrent.test.k1s.io",
							Version:  "v1",
							Resource: "concurrenttests",
						}

						// Register mapping
						concurrentMapper.RegisterMapping(gvk, gvr)

						// Test retrieval
						retrievedGVR, err := concurrentMapper.ResourceFor(gvk)
						if err != nil {
							results <- err
							return
						}
						if retrievedGVR != gvr {
							results <- errors.New("concurrent test failed")
							return
						}

						// Test reverse lookup
						retrievedGVK, err := concurrentMapper.KindFor(gvr)
						if err != nil {
							results <- err
							return
						}
						if retrievedGVK != gvk {
							results <- errors.New("concurrent reverse test failed")
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
	})
})
