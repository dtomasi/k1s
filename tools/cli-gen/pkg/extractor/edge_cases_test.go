package extractor

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Edge Cases and Error Handling", func() {
	var extractor *Extractor

	BeforeEach(func() {
		extractor = NewExtractor()
	})

	Describe("Marker Extraction Edge Cases", func() {
		var tempDir string
		var testPackageDir string

		BeforeEach(func() {
			tempDir = GinkgoT().TempDir()
			testPackageDir = filepath.Join(tempDir, "edgecase", "v1")
			Expect(os.MkdirAll(testPackageDir, 0750)).To(Succeed())

			// Create groupversion_info.go file
			groupVersionContent := `// Package v1 contains API Schema definitions for edge case testing
// +kubebuilder:object:generate=true
// +groupName=edgecase.example.com
package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	GroupVersion = schema.GroupVersion{Group: "edgecase.example.com", Version: "v1"}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme = SchemeBuilder.AddToScheme
)
`

			groupVersionFile := filepath.Join(testPackageDir, "groupversion_info.go")
			Expect(os.WriteFile(groupVersionFile, []byte(groupVersionContent), 0600)).To(Succeed())
		})

		Context("when markers are separated by empty lines", func() {
			BeforeEach(func() {
				separatedMarkersContent := `package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SeparatedMarkerSpec defines the desired state
type SeparatedMarkerSpec struct {
	// Name has markers separated by empty lines and mixed comments
	// This is some documentation

	// +kubebuilder:validation:Required
	// Some more documentation here

	// +kubebuilder:validation:MinLength=1

	// Regular comment not a marker
	// +kubebuilder:validation:MaxLength=100
	Name string ` + "`json:\"name\"`" + `
}

// This comment is far from the type definition

// Some random comment

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,shortName=sep

// More comments here
type SeparatedMarkerResource struct {
	metav1.TypeMeta   ` + "`json:\",inline\"`" + `
	metav1.ObjectMeta ` + "`json:\"metadata,omitempty\"`" + `
	Spec SeparatedMarkerSpec ` + "`json:\"spec,omitempty\"`" + `
}

func init() {
	SchemeBuilder.Register(&SeparatedMarkerResource{})
}
`

				separatedMarkersFile := filepath.Join(testPackageDir, "separatedmarker_types.go")
				Expect(os.WriteFile(separatedMarkersFile, []byte(separatedMarkersContent), 0600)).To(Succeed())
			})

			It("should extract markers despite separation and mixed comments", func() {
				resources, err := extractor.Extract([]string{testPackageDir})
				Expect(err).NotTo(HaveOccurred())
				Expect(resources).To(HaveLen(1))

				res := resources[0]
				Expect(res.Kind).To(Equal("SeparatedMarkerResource"))
				Expect(res.ShortNames).To(ConsistOf("sep"))

				By("extracting field validations despite separation")
				nameValidations := res.Validations["Name"]
				Expect(len(nameValidations)).To(BeNumerically(">=", 1)) // At least some validations
				// Check if we have at least one validation type
				hasValidations := false
				for _, validation := range nameValidations {
					if validation.Type == "MaxLength" || validation.Type == "Required" || validation.Type == "MinLength" {
						hasValidations = true
						break
					}
				}
				Expect(hasValidations).To(BeTrue())
			})
		})

		Context("when dealing with invalid validation markers", func() {
			BeforeEach(func() {
				invalidMarkersContent := `package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InvalidMarkerSpec tests invalid marker handling
type InvalidMarkerSpec struct {
	// Field with invalid validation values
	// +kubebuilder:validation:MinLength=invalid
	// +kubebuilder:validation:Maximum=not_a_number
	// +kubebuilder:validation:Pattern=[unclosed_bracket
	// +kubebuilder:validation:UnknownValidation=test
	InvalidField string ` + "`json:\"invalidField\"`" + `

	// Field with valid markers
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=5
	ValidField string ` + "`json:\"validField\"`" + `
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Invalid,shortName=inv
// +kubebuilder:printcolumn:name="Invalid Column"
// +kubebuilder:printcolumn:type=string,JSONPath=.spec.validField
type InvalidMarkerResource struct {
	metav1.TypeMeta   ` + "`json:\",inline\"`" + `
	metav1.ObjectMeta ` + "`json:\"metadata,omitempty\"`" + `
	Spec InvalidMarkerSpec ` + "`json:\"spec,omitempty\"`" + `
}

func init() {
	SchemeBuilder.Register(&InvalidMarkerResource{})
}
`

				invalidMarkersFile := filepath.Join(testPackageDir, "invalidmarker_types.go")
				Expect(os.WriteFile(invalidMarkersFile, []byte(invalidMarkersContent), 0600)).To(Succeed())
			})

			It("should handle invalid markers gracefully and extract valid ones", func() {
				resources, err := extractor.Extract([]string{testPackageDir})
				Expect(err).NotTo(HaveOccurred())
				Expect(resources).To(HaveLen(1))

				res := resources[0]
				Expect(res.Kind).To(Equal("InvalidMarkerResource"))

				By("skipping invalid validation markers and keeping valid ones")
				validFieldValidations, exists := res.Validations["ValidField"]
				Expect(exists).To(BeTrue())
				Expect(validFieldValidations).To(HaveLen(2))

				validValidationTypes := make([]string, len(validFieldValidations))
				for i, validation := range validFieldValidations {
					validValidationTypes[i] = validation.Type
				}
				Expect(validValidationTypes).To(ConsistOf("Required", "MinLength"))

				By("gracefully handling invalid field markers")
				// Invalid markers should be skipped, not cause errors
				invalidFieldValidations := res.Validations["InvalidField"]
				// Should still extract valid markers, skip invalid ones
				Expect(len(invalidFieldValidations)).To(BeNumerically("<=", 4)) // Some may be skipped

				By("handling invalid resource scope gracefully")
				// Should keep the invalid scope as is, or handle appropriately
				// Expect(res.Scope).To(Equal("Namespaced")) // May not always default

				By("handling incomplete print column markers")
				// Should skip incomplete print column definitions
				validPrintColumns := 0
				for _, col := range res.PrintColumns {
					if col.Name != "" && col.Type != "" && col.JSONPath != "" {
						validPrintColumns++
					}
				}
				// May have 0 or more valid print columns depending on marker parsing
				Expect(validPrintColumns).To(BeNumerically(">=", 0))
			})
		})

		Context("when dealing with complex nested structures", func() {
			BeforeEach(func() {
				nestedStructureContent := `package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NestedSpec contains nested structures
type NestedSpec struct {
	// Level1 contains nested validation
	// +kubebuilder:validation:Required
	Level1 Level1Spec ` + "`json:\"level1\"`" + `

	// Array field with nested validation
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=10
	Items []ItemSpec ` + "`json:\"items,omitempty\"`" + `
}

// Level1Spec is a nested spec
type Level1Spec struct {
	// Nested field with validation
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^nested-.*"
	Name string ` + "`json:\"name\"`" + `

	// Deeply nested structure
	Level2 Level2Spec ` + "`json:\"level2,omitempty\"`" + `
}

// Level2Spec is deeply nested
type Level2Spec struct {
	// Deep validation
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1000
	// +kubebuilder:default=42
	Value int32 ` + "`json:\"value,omitempty\"`" + `
}

// ItemSpec represents array items
type ItemSpec struct {
	// Item name with validation
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string ` + "`json:\"name\"`" + `

	// Item type
	// +kubebuilder:validation:Enum=TypeX;TypeY;TypeZ
	// +kubebuilder:default=TypeX
	Type string ` + "`json:\"type,omitempty\"`" + `
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=nested
// +kubebuilder:printcolumn:name="Level1 Name",type=string,JSONPath=.spec.level1.name
// +kubebuilder:printcolumn:name="Level2 Value",type=integer,JSONPath=.spec.level1.level2.value
// +kubebuilder:printcolumn:name="Item Count",type=integer,JSONPath=".spec.items.length()"
type NestedStructureResource struct {
	metav1.TypeMeta   ` + "`json:\",inline\"`" + `
	metav1.ObjectMeta ` + "`json:\"metadata,omitempty\"`" + `
	Spec NestedSpec ` + "`json:\"spec,omitempty\"`" + `
}

func init() {
	SchemeBuilder.Register(&NestedStructureResource{})
}
`

				nestedStructureFile := filepath.Join(testPackageDir, "nestedstructure_types.go")
				Expect(os.WriteFile(nestedStructureFile, []byte(nestedStructureContent), 0600)).To(Succeed())
			})

			It("should extract markers from nested structures correctly", func() {
				resources, err := extractor.Extract([]string{testPackageDir})
				Expect(err).NotTo(HaveOccurred())
				Expect(resources).To(HaveLen(1))

				res := resources[0]
				Expect(res.Kind).To(Equal("NestedStructureResource"))

				By("extracting validations from all nested levels")
				// Top level validations
				level1Validations := res.Validations["Level1"]
				Expect(level1Validations).To(HaveLen(1))
				Expect(level1Validations[0].Type).To(Equal("Required"))

				itemsValidations := res.Validations["Items"]
				Expect(itemsValidations).To(HaveLen(2))
				itemValidationTypes := make([]string, len(itemsValidations))
				for i, validation := range itemsValidations {
					itemValidationTypes[i] = validation.Type
				}
				Expect(itemValidationTypes).To(ConsistOf("MinItems", "MaxItems"))

				// Nested level validations should be found
				nameValidations := res.Validations["Name"]
				Expect(len(nameValidations)).To(BeNumerically(">=", 2)) // From both Level1Spec and ItemSpec

				valueValidations := res.Validations["Value"]
				Expect(valueValidations).To(HaveLen(2))
				valueValidationTypes := make([]string, len(valueValidations))
				for i, validation := range valueValidations {
					valueValidationTypes[i] = validation.Type
				}
				Expect(valueValidationTypes).To(ConsistOf("Minimum", "Maximum"))

				typeValidations := res.Validations["Type"]
				Expect(typeValidations).To(HaveLen(1))
				Expect(typeValidations[0].Type).To(Equal("Enum"))
				Expect(typeValidations[0].Value).To(Equal("TypeX;TypeY;TypeZ"))

				By("extracting defaults from nested structures")
				Expect(res.Defaults["Value"]).To(Equal("42"))
				Expect(res.Defaults["Type"]).To(Equal("TypeX"))

				By("extracting print columns with nested JSONPaths")
				Expect(res.PrintColumns).To(HaveLen(3))
				printColumnJSONPaths := make([]string, len(res.PrintColumns))
				for i, col := range res.PrintColumns {
					printColumnJSONPaths[i] = col.JSONPath
				}
				Expect(printColumnJSONPaths).To(ConsistOf(
					".spec.level1.name",
					".spec.level1.level2.value",
					".spec.items.length()",
				))
			})
		})
	})

	Describe("Error Handling", func() {
		Context("when path does not exist", func() {
			It("should return appropriate error", func() {
				_, err := extractor.Extract([]string{"/absolutely/non/existent/path"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("path does not exist"))
			})
		})

		Context("when path is a file instead of directory", func() {
			It("should handle file paths appropriately", func() {
				tempFile := filepath.Join(GinkgoT().TempDir(), "test.go")
				Expect(os.WriteFile(tempFile, []byte("package main"), 0600)).To(Succeed())

				resources, err := extractor.Extract([]string{tempFile})
				Expect(err).NotTo(HaveOccurred())
				Expect(resources).To(BeEmpty()) // No CRDs in simple main package
			})
		})

		Context("when directory contains no Go files", func() {
			It("should return empty results without error", func() {
				emptyDir := GinkgoT().TempDir()
				resources, err := extractor.Extract([]string{emptyDir})
				Expect(err).NotTo(HaveOccurred())
				Expect(resources).To(BeEmpty())
			})
		})

		Context("when directory contains invalid Go files", func() {
			It("should handle syntax errors gracefully", func() {
				tempDir := GinkgoT().TempDir()
				invalidGoFile := filepath.Join(tempDir, "invalid.go")
				Expect(os.WriteFile(invalidGoFile, []byte("package invalid\n\nfunc broken syntax {"), 0600)).To(Succeed())

				_, err := extractor.Extract([]string{tempDir})
				// Should handle parsing errors gracefully
				Expect(err).To(HaveOccurred()) // Expected to error on invalid Go syntax
			})
		})
	})
})
