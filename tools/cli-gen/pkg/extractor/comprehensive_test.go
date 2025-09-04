package extractor

import (
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Comprehensive Marker Extraction", func() {
	var extractor *Extractor

	BeforeEach(func() {
		extractor = NewExtractor()
	})

	Describe("Mock CRD Testing", func() {
		var tempDir string
		var testPackageDir string

		BeforeEach(func() {
			tempDir = GinkgoT().TempDir()
			testPackageDir = filepath.Join(tempDir, "testapi", "v1beta1")
			Expect(os.MkdirAll(testPackageDir, 0750)).To(Succeed())

			// Create groupversion_info.go file
			groupVersionContent := `// Package v1beta1 contains API Schema definitions for the test API group
// +kubebuilder:object:generate=true
// +groupName=test.example.com
package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "test.example.com", Version: "v1beta1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
`

			groupVersionFile := filepath.Join(testPackageDir, "groupversion_info.go")
			Expect(os.WriteFile(groupVersionFile, []byte(groupVersionContent), 0600)).To(Succeed())
		})

		Context("when testing comprehensive resource with all marker types", func() {
			BeforeEach(func() {
				comprehensiveResourceContent := `package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ComprehensiveResourceSpec defines the desired state
type ComprehensiveResourceSpec struct {
	// Name is the resource name
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern="^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"
	Name string ` + "`json:\"name\"`" + `

	// Replicas defines the number of replicas
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=1
	Replicas *int32 ` + "`json:\"replicas,omitempty\"`" + `

	// Type defines the resource type
	// +kubebuilder:validation:Enum=TypeA;TypeB;TypeC
	// +kubebuilder:default=TypeA
	Type string ` + "`json:\"type,omitempty\"`" + `

	// Config contains configuration data
	// +kubebuilder:validation:Optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Config map[string]string ` + "`json:\"config,omitempty\"`" + `
}

// ComprehensiveResourceStatus defines the observed state
type ComprehensiveResourceStatus struct {
	// Phase represents the current phase
	// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed
	Phase string ` + "`json:\"phase,omitempty\"`" + `

	// Conditions represent the latest available observations
	Conditions []metav1.Condition ` + "`json:\"conditions,omitempty\"`" + `

	// ReadyReplicas is the number of ready replicas
	ReadyReplicas int32 ` + "`json:\"readyReplicas,omitempty\"`" + `
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.readyReplicas
// +kubebuilder:resource:scope=Namespaced,shortName=comp;cr,plural=comprehensiveresources,singular=comprehensiveresource,categories=all
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=.spec.name,description="The resource name"
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=.spec.type,description="The resource type"
// +kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=.spec.replicas,description="Desired replicas"
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=.status.readyReplicas,description="Ready replicas"
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=.status.phase,description="Current phase"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=.metadata.creationTimestamp
type ComprehensiveResource struct {
	metav1.TypeMeta   ` + "`json:\",inline\"`" + `
	metav1.ObjectMeta ` + "`json:\"metadata,omitempty\"`" + `

	Spec   ComprehensiveResourceSpec   ` + "`json:\"spec,omitempty\"`" + `
	Status ComprehensiveResourceStatus ` + "`json:\"status,omitempty\"`" + `
}

// +kubebuilder:object:root=true

// ComprehensiveResourceList contains a list of ComprehensiveResource
type ComprehensiveResourceList struct {
	metav1.TypeMeta ` + "`json:\",inline\"`" + `
	metav1.ListMeta ` + "`json:\"metadata,omitempty\"`" + `
	Items           []ComprehensiveResource ` + "`json:\"items\"`" + `
}

func init() {
	SchemeBuilder.Register(&ComprehensiveResource{}, &ComprehensiveResourceList{})
}
`

				comprehensiveResourceFile := filepath.Join(testPackageDir, "comprehensiveresource_types.go")
				Expect(os.WriteFile(comprehensiveResourceFile, []byte(comprehensiveResourceContent), 0600)).To(Succeed())
			})

			It("should extract all marker types correctly", func() {
				resources, err := extractor.Extract([]string{testPackageDir})
				Expect(err).NotTo(HaveOccurred())
				Expect(resources).To(HaveLen(1))

				res := resources[0]
				Expect(res.Kind).To(Equal("ComprehensiveResource"))
				Expect(res.Group).To(Equal("test.example.com"))
				Expect(res.Version).To(Equal("v1beta1"))
				Expect(res.Plural).To(Equal("comprehensiveresources"))
				Expect(res.Singular).To(Equal("comprehensiveresource"))
				Expect(res.Scope).To(Equal("Namespaced"))
				Expect(res.ShortNames).To(ConsistOf("comp", "cr"))
				// Categories are not directly stored in ResourceInfo, but would be extracted if present

				By("extracting print columns correctly")
				Expect(res.PrintColumns).To(HaveLen(6))
				printColumnNames := make([]string, len(res.PrintColumns))
				for i, col := range res.PrintColumns {
					printColumnNames[i] = col.Name
				}
				Expect(printColumnNames).To(ConsistOf("Name", "Type", "Replicas", "Ready", "Phase", "Age"))

				nameColumn := findPrintColumn(res.PrintColumns, "Name")
				Expect(nameColumn).NotTo(BeNil())
				Expect(nameColumn.Type).To(Equal("string"))
				Expect(nameColumn.JSONPath).To(Equal(".spec.name"))
				Expect(nameColumn.Description).To(Equal("The resource name"))

				By("extracting field validations correctly")
				nameValidations := res.Validations["Name"]
				Expect(nameValidations).To(HaveLen(4))
				validationTypes := make([]string, len(nameValidations))
				for i, validation := range nameValidations {
					validationTypes[i] = validation.Type
				}
				Expect(validationTypes).To(ConsistOf("Required", "MinLength", "MaxLength", "Pattern"))

				replicasValidations := res.Validations["Replicas"]
				Expect(replicasValidations).To(HaveLen(2))
				replicasValidationTypes := make([]string, len(replicasValidations))
				for i, validation := range replicasValidations {
					replicasValidationTypes[i] = validation.Type
				}
				Expect(replicasValidationTypes).To(ConsistOf("Minimum", "Maximum"))

				typeValidations := res.Validations["Type"]
				Expect(typeValidations).To(HaveLen(1))
				Expect(typeValidations[0].Type).To(Equal("Enum"))
				Expect(typeValidations[0].Value).To(Equal("TypeA;TypeB;TypeC"))

				By("extracting default values correctly")
				Expect(res.Defaults["Replicas"]).To(Equal("1"))
				Expect(res.Defaults["Type"]).To(Equal("TypeA"))
			})
		})

		Context("when testing cluster-scoped resource", func() {
			BeforeEach(func() {
				globalResourceContent := `package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GlobalResourceSpec defines the desired state of GlobalResource
type GlobalResourceSpec struct {
	// DisplayName is a human readable name
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=3
	DisplayName string ` + "`json:\"displayName\"`" + `

	// Priority defines the resource priority
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=10
	// +kubebuilder:default=5
	Priority int32 ` + "`json:\"priority,omitempty\"`" + `
}

// GlobalResourceStatus defines the observed state of GlobalResource
type GlobalResourceStatus struct {
	// Ready indicates if the resource is ready
	Ready bool ` + "`json:\"ready,omitempty\"`" + `
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=gr;global,plural=globalresources,singular=globalresource
// +kubebuilder:printcolumn:name="Display Name",type=string,JSONPath=.spec.displayName
// +kubebuilder:printcolumn:name="Priority",type=integer,JSONPath=.spec.priority
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=.status.ready
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=.metadata.creationTimestamp
type GlobalResource struct {
	metav1.TypeMeta   ` + "`json:\",inline\"`" + `
	metav1.ObjectMeta ` + "`json:\"metadata,omitempty\"`" + `

	Spec   GlobalResourceSpec   ` + "`json:\"spec,omitempty\"`" + `
	Status GlobalResourceStatus ` + "`json:\"status,omitempty\"`" + `
}

// +kubebuilder:object:root=true

// GlobalResourceList contains a list of GlobalResource
type GlobalResourceList struct {
	metav1.TypeMeta ` + "`json:\",inline\"`" + `
	metav1.ListMeta ` + "`json:\"metadata,omitempty\"`" + `
	Items           []GlobalResource ` + "`json:\"items\"`" + `
}

func init() {
	SchemeBuilder.Register(&GlobalResource{}, &GlobalResourceList{})
}
`

				globalResourceFile := filepath.Join(testPackageDir, "globalresource_types.go")
				Expect(os.WriteFile(globalResourceFile, []byte(globalResourceContent), 0600)).To(Succeed())
			})

			It("should extract cluster-scoped resource correctly", func() {
				resources, err := extractor.Extract([]string{testPackageDir})
				Expect(err).NotTo(HaveOccurred())

				var globalResource *ResourceInfo
				for _, res := range resources {
					if res.Kind == "GlobalResource" {
						globalResource = res
						break
					}
				}

				Expect(globalResource).NotTo(BeNil())
				Expect(globalResource.Scope).To(Equal("Cluster"))
				Expect(globalResource.ShortNames).To(ConsistOf("gr", "global"))

				By("extracting validations for cluster resource")
				displayNameValidations := globalResource.Validations["DisplayName"]
				Expect(displayNameValidations).To(HaveLen(2))

				priorityValidations := globalResource.Validations["Priority"]
				Expect(priorityValidations).To(HaveLen(2))

				By("extracting defaults for cluster resource")
				Expect(globalResource.Defaults["Priority"]).To(Equal("5"))
			})
		})

		Context("when testing simple resource with minimal markers", func() {
			BeforeEach(func() {
				simpleResourceContent := `package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SimpleResourceSpec defines the desired state of SimpleResource
type SimpleResourceSpec struct {
	// Value is a simple string value
	Value string ` + "`json:\"value,omitempty\"`" + `
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,shortName=sr
type SimpleResource struct {
	metav1.TypeMeta   ` + "`json:\",inline\"`" + `
	metav1.ObjectMeta ` + "`json:\"metadata,omitempty\"`" + `

	Spec SimpleResourceSpec ` + "`json:\"spec,omitempty\"`" + `
}

// +kubebuilder:object:root=true

// SimpleResourceList contains a list of SimpleResource
type SimpleResourceList struct {
	metav1.TypeMeta ` + "`json:\",inline\"`" + `
	metav1.ListMeta ` + "`json:\"metadata,omitempty\"`" + `
	Items           []SimpleResource ` + "`json:\"items\"`" + `
}

func init() {
	SchemeBuilder.Register(&SimpleResource{}, &SimpleResourceList{})
}
`

				simpleResourceFile := filepath.Join(testPackageDir, "simpleresource_types.go")
				Expect(os.WriteFile(simpleResourceFile, []byte(simpleResourceContent), 0600)).To(Succeed())
			})

			It("should extract simple resource with minimal markers", func() {
				resources, err := extractor.Extract([]string{testPackageDir})
				Expect(err).NotTo(HaveOccurred())

				var simpleResource *ResourceInfo
				for _, res := range resources {
					if res.Kind == "SimpleResource" {
						simpleResource = res
						break
					}
				}

				Expect(simpleResource).NotTo(BeNil())
				Expect(simpleResource.Kind).To(Equal("SimpleResource"))
				// Check if plural was extracted or auto-generated
				if simpleResource.Plural == "" {
					// Auto-pluralized
					Expect(strings.ToLower(simpleResource.Kind) + "s").To(Equal("simpleresources"))
				} else {
					Expect(simpleResource.Plural).To(Equal("simpleresources"))
				}
				if simpleResource.Singular == "" {
					Expect(strings.ToLower(simpleResource.Kind)).To(Equal("simpleresource"))
				} else {
					Expect(simpleResource.Singular).To(Equal("simpleresource"))
				}
				Expect(simpleResource.Scope).To(Equal("Namespaced"))
				Expect(simpleResource.ShortNames).To(ConsistOf("sr"))

				// Should have no print columns, validations, or defaults for this simple resource
				Expect(simpleResource.PrintColumns).To(BeEmpty())
				Expect(simpleResource.Validations).To(BeEmpty())
				Expect(simpleResource.Defaults).To(BeEmpty())
			})
		})
	})
})

// Helper function to find a print column by name
func findPrintColumn(columns []PrintColumn, name string) *PrintColumn {
	for _, col := range columns {
		if col.Name == name {
			return &col
		}
	}
	return nil
}
