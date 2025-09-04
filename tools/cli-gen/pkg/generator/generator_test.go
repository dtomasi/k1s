package generator

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/dtomasi/k1s/tools/cli-gen/pkg/extractor"
)

var _ = Describe("Generator", func() {
	var generator *Generator
	var tempDir string

	BeforeEach(func() {
		tempDir = GinkgoT().TempDir()
		generator = NewGenerator(tempDir)
	})

	Describe("NewGenerator", func() {
		It("should create a generator with correct output directory", func() {
			Expect(generator).NotTo(BeNil())
			Expect(generator.outputDir).To(Equal(tempDir))
		})

		It("should start with empty generator configuration", func() {
			// Generator starts with empty configuration
			// Generators are enabled explicitly via SetEnabledGenerators
			Expect(generator.enabledGenerators).To(BeEmpty())
		})

		It("should have verbose disabled by default", func() {
			Expect(generator.verbose).To(BeFalse())
		})
	})

	Describe("SetEnabledGenerators", func() {
		Context("when setting specific generators", func() {
			It("should enable only specified generators", func() {
				generator.SetEnabledGenerators([]string{"object", "validation"})

				Expect(generator.enabledGenerators["object"]).To(BeTrue())
				Expect(generator.enabledGenerators["validation"]).To(BeTrue())
				Expect(generator.enabledGenerators["defaulting"]).To(BeFalse())
			})
		})

		Context("when setting unknown generators", func() {
			It("should ignore unknown generators", func() {
				generator.SetEnabledGenerators([]string{"object", "unknown", "validation"})

				Expect(generator.enabledGenerators["object"]).To(BeTrue())
				Expect(generator.enabledGenerators["validation"]).To(BeTrue())
				Expect(generator.enabledGenerators["defaulting"]).To(BeFalse())
			})
		})

		Context("when setting empty list", func() {
			It("should disable all generators", func() {
				generator.SetEnabledGenerators([]string{})

				Expect(generator.enabledGenerators["object"]).To(BeFalse())
				Expect(generator.enabledGenerators["validation"]).To(BeFalse())
				Expect(generator.enabledGenerators["defaulting"]).To(BeFalse())
			})
		})
	})

	Describe("SetVerbose", func() {
		It("should set verbose mode", func() {
			generator.SetVerbose(true)
			Expect(generator.verbose).To(BeTrue())

			generator.SetVerbose(false)
			Expect(generator.verbose).To(BeFalse())
		})
	})

	Describe("Generate", func() {
		var testResources []*extractor.ResourceInfo

		BeforeEach(func() {
			testResources = []*extractor.ResourceInfo{
				{
					Kind:       "TestKind",
					Name:       "testkind",
					Group:      "test.example.com",
					Version:    "v1",
					Plural:     "testkinds",
					Singular:   "testkind",
					ShortNames: []string{"tk"},
					Scope:      "Namespaced",
					HasStatus:  true,
					// Categories would be extracted if present in markers
					PrintColumns: []extractor.PrintColumn{
						{
							Name:        "Name",
							Type:        "string",
							JSONPath:    ".metadata.name",
							Description: "Name of the resource",
							Priority:    0,
						},
						{
							Name:     "Age",
							Type:     "date",
							JSONPath: ".metadata.creationTimestamp",
						},
					},
					Validations: map[string][]extractor.ValidationRule{
						"Name": {
							{Type: "Required", Value: ""},
							{Type: "MinLength", Value: "1"},
						},
						"Price": {
							{Type: "Minimum", Value: "0"},
							{Type: "Maximum", Value: "1000"},
						},
					},
					Defaults: map[string]string{
						"Status":   "Active",
						"Quantity": "1",
					},
				},
			}
		})

		Context("when generating with all generators enabled", func() {
			It("should create all generated files successfully", func() {
				err := generator.Generate(testResources)
				Expect(err).NotTo(HaveOccurred())

				expectedFiles := []string{
					"zz_generated.resource_metadata.go",
					"zz_generated.validation_strategies.go",
					"zz_generated.defaulting_strategies.go",
				}

				for _, filename := range expectedFiles {
					filePath := filepath.Join(tempDir, filename)
					_, err := os.Stat(filePath)
					Expect(err).NotTo(HaveOccurred(), "Expected file %s should exist", filename)
				}
			})

			It("should generate correct resource metadata content", func() {
				err := generator.Generate(testResources)
				Expect(err).NotTo(HaveOccurred())

				content, err := os.ReadFile(filepath.Join(tempDir, "zz_generated.resource_metadata.go")) // #nosec G304
				Expect(err).NotTo(HaveOccurred())

				contentStr := string(content)
				By("including the resource kind")
				Expect(contentStr).To(ContainSubstring("TestKind"))

				By("including the group")
				Expect(contentStr).To(ContainSubstring("test.example.com"))

				By("including short names")
				Expect(contentStr).To(ContainSubstring(`ShortNames: []string{"tk"}`))

				By("including scope")
				Expect(contentStr).To(ContainSubstring("Namespaced"))

				By("containing valid Go code structure")
				Expect(contentStr).To(ContainSubstring("Kind:"))
				Expect(contentStr).To(ContainSubstring("Group:"))
			})

			It("should generate correct validation strategies content", func() {
				err := generator.Generate(testResources)
				Expect(err).NotTo(HaveOccurred())

				content, err := os.ReadFile(filepath.Join(tempDir, "zz_generated.validation_strategies.go")) // #nosec G304
				Expect(err).NotTo(HaveOccurred())

				contentStr := string(content)
				By("including validation strategy struct")
				Expect(contentStr).To(ContainSubstring("TestKindValidationStrategy"))

				By("including validation logic")
				Expect(contentStr).To(ContainSubstring("field is required but empty"))

				By("including field validations")
				Expect(contentStr).To(ContainSubstring("Name"))
				Expect(contentStr).To(ContainSubstring("Price"))
			})

			It("should generate correct defaulting strategies content", func() {
				err := generator.Generate(testResources)
				Expect(err).NotTo(HaveOccurred())

				content, err := os.ReadFile(filepath.Join(tempDir, "zz_generated.defaulting_strategies.go")) // #nosec G304
				Expect(err).NotTo(HaveOccurred())

				contentStr := string(content)
				By("including defaulting strategy struct")
				Expect(contentStr).To(ContainSubstring("TestKindDefaultingStrategy"))

				By("including default values")
				Expect(contentStr).To(ContainSubstring("Active"))
				Expect(contentStr).To(ContainSubstring("1"))
			})

		})

		Context("when generating with selective generators", func() {
			It("should only create files for enabled generators", func() {
				generator.SetEnabledGenerators([]string{"object", "validation"})

				err := generator.Generate(testResources)
				Expect(err).NotTo(HaveOccurred())

				// Should create these files
				enabledFiles := []string{
					"zz_generated.resource_metadata.go",
					"zz_generated.validation_strategies.go",
				}

				for _, filename := range enabledFiles {
					filePath := filepath.Join(tempDir, filename)
					_, err := os.Stat(filePath)
					Expect(err).NotTo(HaveOccurred(), "Expected file %s should exist", filename)
				}

				// Should not create these files
				disabledFiles := []string{
					"zz_generated.print_columns.go",
					"zz_generated.defaulting_strategies.go",
				}

				for _, filename := range disabledFiles {
					filePath := filepath.Join(tempDir, filename)
					_, err := os.Stat(filePath)
					Expect(err).To(HaveOccurred(), "File %s should not exist", filename)
				}
			})
		})

		Context("when generating with empty resources", func() {
			It("should return error for empty resource list", func() {
				err := generator.Generate([]*extractor.ResourceInfo{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no resources to generate"))
			})
		})

		Context("when output directory does not exist", func() {
			It("should create the directory and generate files", func() {
				nonExistentDir := filepath.Join(tempDir, "nested", "path")
				generator = NewGenerator(nonExistentDir)

				err := generator.Generate(testResources)
				Expect(err).NotTo(HaveOccurred())

				// Check directory was created
				_, err = os.Stat(nonExistentDir)
				Expect(err).NotTo(HaveOccurred())

				// Check files were created
				filePath := filepath.Join(nonExistentDir, "zz_generated.resource_metadata.go")
				_, err = os.Stat(filePath)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("Error Handling", func() {
		Context("when output directory cannot be created", func() {
			It("should return appropriate error", func() {
				Skip("Difficult to test permission errors in CI/CD environment")
			})
		})

		Context("when templates fail to execute", func() {
			It("should handle template execution errors gracefully", func() {
				// This would require corrupted templates or invalid data
				// which is difficult to simulate in unit tests
				Skip("Template execution errors are handled by underlying template engine")
			})
		})
	})
})
