package registry_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/dtomasi/k1s/core/registry"
	typesv1 "github.com/dtomasi/k1s/core/types/v1"
)

var _ = Describe("Core Resources Registry", func() {
	var testRegistry registry.Registry

	BeforeEach(func() {
		testRegistry = registry.NewRegistry(
			registry.WithDefaultCategories("all"),
			registry.WithShortNameValidation(true),
		)
	})

	Describe("RegisterCoreResources", func() {
		It("should register all core resources successfully", func() {
			err := registry.RegisterCoreResources(testRegistry)
			Expect(err).ToNot(HaveOccurred())

			// Verify all core resources are registered
			registeredResources := testRegistry.ListResources()
			expectedGVRs := typesv1.GetAllGVRs()

			Expect(len(registeredResources)).To(Equal(len(expectedGVRs)))
			for _, expectedGVR := range expectedGVRs {
				Expect(registeredResources).To(ContainElement(expectedGVR))
			}
		})

		It("should register resources with correct configuration", func() {
			err := registry.RegisterCoreResources(testRegistry)
			Expect(err).ToNot(HaveOccurred())

			// Test Namespace configuration
			nsConfig, err := testRegistry.GetResourceConfig(typesv1.GetNamespaceGVR())
			Expect(err).ToNot(HaveOccurred())
			Expect(nsConfig.Singular).To(Equal("namespace"))
			Expect(nsConfig.Plural).To(Equal("namespaces"))
			Expect(nsConfig.Kind).To(Equal("Namespace"))
			Expect(nsConfig.ListKind).To(Equal("NamespaceList"))
			Expect(nsConfig.Namespaced).To(BeFalse())
			Expect(nsConfig.ShortNames).To(ContainElement("ns"))
			Expect(nsConfig.Categories).To(ContainElement("all"))
			Expect(nsConfig.PrintColumns).ToNot(BeEmpty())
			Expect(nsConfig.Description).To(ContainSubstring("multi-tenancy"))

			// Test ConfigMap configuration
			cmConfig, err := testRegistry.GetResourceConfig(typesv1.GetConfigMapGVR())
			Expect(err).ToNot(HaveOccurred())
			Expect(cmConfig.Singular).To(Equal("configmap"))
			Expect(cmConfig.Plural).To(Equal("configmaps"))
			Expect(cmConfig.Kind).To(Equal("ConfigMap"))
			Expect(cmConfig.Namespaced).To(BeTrue())
			Expect(cmConfig.ShortNames).To(ContainElement("cm"))

			// Test Secret configuration
			secretConfig, err := testRegistry.GetResourceConfig(typesv1.GetSecretGVR())
			Expect(err).ToNot(HaveOccurred())
			Expect(secretConfig.Singular).To(Equal("secret"))
			Expect(secretConfig.Plural).To(Equal("secrets"))
			Expect(secretConfig.Kind).To(Equal("Secret"))
			Expect(secretConfig.Namespaced).To(BeTrue())
			Expect(secretConfig.Description).To(ContainSubstring("sensitive data"))

			// Test ServiceAccount configuration
			saConfig, err := testRegistry.GetResourceConfig(typesv1.GetServiceAccountGVR())
			Expect(err).ToNot(HaveOccurred())
			Expect(saConfig.Singular).To(Equal("serviceaccount"))
			Expect(saConfig.Plural).To(Equal("serviceaccounts"))
			Expect(saConfig.Kind).To(Equal("ServiceAccount"))
			Expect(saConfig.Namespaced).To(BeTrue())
			Expect(saConfig.ShortNames).To(ContainElement("sa"))

			// Test Event configuration
			eventConfig, err := testRegistry.GetResourceConfig(typesv1.GetEventGVR())
			Expect(err).ToNot(HaveOccurred())
			Expect(eventConfig.Singular).To(Equal("event"))
			Expect(eventConfig.Plural).To(Equal("events"))
			Expect(eventConfig.Kind).To(Equal("Event"))
			Expect(eventConfig.Namespaced).To(BeTrue())
			Expect(eventConfig.ShortNames).To(ContainElement("ev"))
		})

		It("should register short names correctly", func() {
			err := registry.RegisterCoreResources(testRegistry)
			Expect(err).ToNot(HaveOccurred())

			// Test short name resolution
			nsGVR, err := testRegistry.GetGVRForShortName("ns")
			Expect(err).ToNot(HaveOccurred())
			Expect(nsGVR).To(Equal(typesv1.GetNamespaceGVR()))

			cmGVR, err := testRegistry.GetGVRForShortName("cm")
			Expect(err).ToNot(HaveOccurred())
			Expect(cmGVR).To(Equal(typesv1.GetConfigMapGVR()))

			saGVR, err := testRegistry.GetGVRForShortName("sa")
			Expect(err).ToNot(HaveOccurred())
			Expect(saGVR).To(Equal(typesv1.GetServiceAccountGVR()))

			evGVR, err := testRegistry.GetGVRForShortName("ev")
			Expect(err).ToNot(HaveOccurred())
			Expect(evGVR).To(Equal(typesv1.GetEventGVR()))
		})

		It("should register categories correctly", func() {
			err := registry.RegisterCoreResources(testRegistry)
			Expect(err).ToNot(HaveOccurred())

			// All core resources should be in the 'all' category
			allResources := testRegistry.GetGVRsForCategory("all")
			expectedGVRs := typesv1.GetAllGVRs()

			Expect(len(allResources)).To(Equal(len(expectedGVRs)))
			for _, expectedGVR := range expectedGVRs {
				Expect(allResources).To(ContainElement(expectedGVR))
			}
		})

		It("should register GVK/GVR mappings correctly", func() {
			err := registry.RegisterCoreResources(testRegistry)
			Expect(err).ToNot(HaveOccurred())

			// Test GVK to GVR conversion
			mappings := typesv1.GetGVKToGVRMappings()
			for gvk, expectedGVR := range mappings {
				actualGVR, err := testRegistry.GetGVRForGVK(gvk)
				Expect(err).ToNot(HaveOccurred())
				Expect(actualGVR).To(Equal(expectedGVR))
			}

			// Test GVR to GVK conversion
			reverseMappings := typesv1.GetGVRToGVKMappings()
			for gvr, expectedGVK := range reverseMappings {
				actualGVK, err := testRegistry.GetGVKForGVR(gvr)
				Expect(err).ToNot(HaveOccurred())
				Expect(actualGVK).To(Equal(expectedGVK))
			}
		})
	})

	Describe("Helper Functions", func() {
		It("should return correct core resource GVRs", func() {
			gvrs := registry.GetCoreResourceGVRs()

			Expect(gvrs).To(ContainElement("v1/namespaces"))
			Expect(gvrs).To(ContainElement("v1/configmaps"))
			Expect(gvrs).To(ContainElement("v1/secrets"))
			Expect(gvrs).To(ContainElement("v1/serviceaccounts"))
			Expect(gvrs).To(ContainElement("v1/events"))
		})

		It("should correctly identify core resources", func() {
			Expect(registry.IsCoreResource("v1/namespaces")).To(BeTrue())
			Expect(registry.IsCoreResource("v1/configmaps")).To(BeTrue())
			Expect(registry.IsCoreResource("v1/secrets")).To(BeTrue())
			Expect(registry.IsCoreResource("v1/serviceaccounts")).To(BeTrue())
			Expect(registry.IsCoreResource("v1/events")).To(BeTrue())

			Expect(registry.IsCoreResource("apps/v1/deployments")).To(BeFalse())
			Expect(registry.IsCoreResource("custom/v1/mycrd")).To(BeFalse())
		})
	})

	Describe("Error Handling", func() {
		It("should prevent duplicate registration", func() {
			err := registry.RegisterCoreResources(testRegistry)
			Expect(err).ToNot(HaveOccurred())

			// Try to register again - should fail
			err = registry.RegisterCoreResources(testRegistry)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already registered"))
		})
	})
})
