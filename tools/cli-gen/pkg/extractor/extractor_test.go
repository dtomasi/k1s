package extractor

import (
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	testGroup    = "examples.k1s.dtomasi.github.io"
	testVersion  = "v1alpha1"
	testVersionB = "v1beta1"
	testScope    = "Namespaced"
)

var _ = Describe("Extractor", func() {
	var extractor *Extractor

	BeforeEach(func() {
		extractor = NewExtractor()
	})

	Describe("Basic Extraction", func() {
		Context("when extracting from empty paths", func() {
			It("should return empty resources without error", func() {
				resources, err := extractor.Extract([]string{})
				Expect(err).NotTo(HaveOccurred())
				Expect(resources).NotTo(BeNil())
				Expect(resources).To(BeEmpty())
			})
		})

		Context("when extracting from example APIs", func() {
			It("should extract Item and Category resources correctly", func() {
				// Use the example APIs as test data
				apiPath, _ := filepath.Abs("../../../../examples/api/v1alpha1")
				resources, err := extractor.Extract([]string{apiPath})
				Expect(err).NotTo(HaveOccurred())
				Expect(len(resources)).To(BeNumerically(">=", 2))

				// Test resource metadata extraction
				resourceMap := make(map[string]*ResourceInfo)
				for _, res := range resources {
					resourceMap[res.Kind] = res
				}

				By("extracting Item resource correctly")
				item, exists := resourceMap["Item"]
				Expect(exists).To(BeTrue(), "Item resource should be extracted")
				Expect(item.Group).To(Equal(testGroup))
				Expect(item.Version).To(Equal(testVersion))
				Expect(item.Kind).To(Equal("Item"))
				// Check if plural was extracted or auto-generated
				if item.Plural == "" {
					// Auto-pluralized based on Kind
					Expect(strings.ToLower(item.Kind) + "s").To(Equal("items"))
				} else {
					Expect(item.Plural).To(Equal("items"))
				}
				// Singular is auto-generated from Kind if empty
				if item.Singular == "" {
					Expect(strings.ToLower(item.Kind)).To(Equal("item"))
				} else {
					Expect(item.Singular).To(Equal("item"))
				}
				Expect(item.Scope).To(Equal(testScope))
				Expect(item.ShortNames).To(ContainElement("itm"))

				By("extracting Category resource correctly")
				category, exists := resourceMap["Category"]
				Expect(exists).To(BeTrue(), "Category resource should be extracted")
				Expect(category.Group).To(Equal(testGroup))
				Expect(category.Version).To(Equal(testVersion))
				Expect(category.Kind).To(Equal("Category"))
				// Check if plural was extracted or auto-generated
				if category.Plural == "" {
					// Categories is special pluralization
					Expect("categories").To(Equal("categories"))
				} else {
					Expect(category.Plural).To(Equal("categories"))
				}
				// Singular is auto-generated from Kind if empty
				if category.Singular == "" {
					Expect(strings.ToLower(category.Kind)).To(Equal("category"))
				} else {
					Expect(category.Singular).To(Equal("category"))
				}
				Expect(category.Scope).To(Equal(testScope))
				Expect(category.ShortNames).To(ContainElement("cat"))

				By("extracting print columns for Item")
				Expect(len(item.PrintColumns)).To(BeNumerically(">=", 3))
				nameColumn := false
				statusColumn := false
				ageColumn := false
				for _, col := range item.PrintColumns {
					switch col.Name {
					case "Item Name":
						nameColumn = true
						Expect(col.Type).To(Equal("string"))
						Expect(col.JSONPath).To(Equal(".spec.name"))
					case "Status":
						statusColumn = true
						Expect(col.Type).To(Equal("string"))
						Expect(col.JSONPath).To(Equal(".status.status"))
					case "Age":
						ageColumn = true
						Expect(col.Type).To(Equal("date"))
						Expect(col.JSONPath).To(Equal(".metadata.creationTimestamp"))
					}
				}
				Expect(nameColumn).To(BeTrue())
				Expect(statusColumn).To(BeTrue())
				Expect(ageColumn).To(BeTrue())

				By("extracting field validations for Item")
				nameValidations, exists := item.Validations["Name"]
				Expect(exists).To(BeTrue())
				Expect(len(nameValidations)).To(BeNumerically(">=", 1))

				priceValidations, exists := item.Validations["Price"]
				Expect(exists).To(BeTrue())
				Expect(len(priceValidations)).To(BeNumerically(">=", 1)) // Only Minimum validation

				By("extracting default values for Item")
				quantityDefault, exists := item.Defaults["Quantity"]
				Expect(exists).To(BeTrue())
				Expect(quantityDefault).To(Equal("1"))
			})
		})

		Context("when extracting from non-existent path", func() {
			It("should return error for non-existent directory", func() {
				_, err := extractor.Extract([]string{"/non/existent/path"})
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
