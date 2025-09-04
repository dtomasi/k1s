package v1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	typesv1 "github.com/dtomasi/k1s/core/types/v1"
)

var _ = Describe("Core Types", func() {
	Describe("Namespace", func() {
		It("should create a new namespace with correct metadata", func() {
			ns := typesv1.NewNamespace("test-namespace")

			Expect(ns.Name).To(Equal("test-namespace"))
			Expect(ns.APIVersion).To(Equal("v1"))
			Expect(ns.Kind).To(Equal("Namespace"))
			Expect(ns.Namespace).To(BeEmpty()) // Namespace is cluster-scoped
		})

		It("should return correct GVK and GVR", func() {
			expectedGVK := schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Namespace",
			}
			expectedGVR := schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "namespaces",
			}

			Expect(typesv1.GetNamespaceGVK()).To(Equal(expectedGVK))
			Expect(typesv1.GetNamespaceGVR()).To(Equal(expectedGVR))
		})

		It("should return correct namespace metadata", func() {
			Expect(typesv1.IsNamespaceScoped()).To(BeFalse())
			Expect(typesv1.GetNamespaceShortNames()).To(ContainElement("ns"))
			Expect(typesv1.GetNamespaceCategories()).To(ContainElement("all"))
			Expect(typesv1.GetNamespacePrintColumns()).ToNot(BeEmpty())
		})
	})

	Describe("ConfigMap", func() {
		It("should create a new configmap with correct metadata", func() {
			cm := typesv1.NewConfigMap("test-cm", "test-namespace")

			Expect(cm.Name).To(Equal("test-cm"))
			Expect(cm.Namespace).To(Equal("test-namespace"))
			Expect(cm.APIVersion).To(Equal("v1"))
			Expect(cm.Kind).To(Equal("ConfigMap"))
			Expect(cm.Data).ToNot(BeNil())
		})

		It("should create configmap with data", func() {
			data := map[string]string{
				"key1": "value1",
				"key2": "value2",
			}
			cm := typesv1.NewConfigMapWithData("test-cm", "test-namespace", data)

			Expect(cm.Data).To(Equal(data))
		})

		It("should return correct GVK and GVR", func() {
			expectedGVK := schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "ConfigMap",
			}
			expectedGVR := schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "configmaps",
			}

			Expect(typesv1.GetConfigMapGVK()).To(Equal(expectedGVK))
			Expect(typesv1.GetConfigMapGVR()).To(Equal(expectedGVR))
		})

		It("should return correct configmap metadata", func() {
			Expect(typesv1.IsConfigMapNamespaceScoped()).To(BeTrue())
			Expect(typesv1.GetConfigMapShortNames()).To(ContainElement("cm"))
			Expect(typesv1.GetConfigMapCategories()).To(ContainElement("all"))
			Expect(typesv1.GetConfigMapPrintColumns()).ToNot(BeEmpty())
			Expect(typesv1.GetConfigMapPrintColumnsWithNamespace()).ToNot(BeEmpty())
		})
	})

	Describe("Secret", func() {
		It("should create a new opaque secret with correct metadata", func() {
			secret := typesv1.NewOpaqueSecret("test-secret", "test-namespace")

			Expect(secret.Name).To(Equal("test-secret"))
			Expect(secret.Namespace).To(Equal("test-namespace"))
			Expect(secret.APIVersion).To(Equal("v1"))
			Expect(secret.Kind).To(Equal("Secret"))
			Expect(secret.Type).To(Equal(corev1.SecretTypeOpaque))
			Expect(secret.Data).ToNot(BeNil())
		})

		It("should create secret with data", func() {
			data := map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("secret123"),
			}
			secret := typesv1.NewSecretWithData("test-secret", "test-namespace", corev1.SecretTypeOpaque, data)

			Expect(secret.Data).To(Equal(data))
		})

		It("should create secret from string data", func() {
			stringData := map[string]string{
				"username": "admin",
				"password": "secret123",
			}
			secret := typesv1.NewSecretFromStringData("test-secret", "test-namespace", corev1.SecretTypeOpaque, stringData)

			Expect(secret.StringData).To(Equal(stringData))
		})

		It("should return correct secret metadata", func() {
			Expect(typesv1.IsSecretNamespaceScoped()).To(BeTrue())
			Expect(typesv1.GetSecretShortNames()).To(BeEmpty())
			Expect(typesv1.GetSecretCategories()).To(ContainElement("all"))
			Expect(typesv1.GetSecretPrintColumns()).ToNot(BeEmpty())
		})
	})

	Describe("ServiceAccount", func() {
		It("should create a new service account with correct metadata", func() {
			sa := typesv1.NewServiceAccount("test-sa", "test-namespace")

			Expect(sa.Name).To(Equal("test-sa"))
			Expect(sa.Namespace).To(Equal("test-namespace"))
			Expect(sa.APIVersion).To(Equal("v1"))
			Expect(sa.Kind).To(Equal("ServiceAccount"))
			Expect(sa.AutomountServiceAccountToken).ToNot(BeNil())
			Expect(*sa.AutomountServiceAccountToken).To(BeTrue())
		})

		It("should create service account with secrets", func() {
			secrets := []corev1.ObjectReference{
				{Name: "secret1", Namespace: "test-namespace"},
				{Name: "secret2", Namespace: "test-namespace"},
			}
			sa := typesv1.NewServiceAccountWithSecrets("test-sa", "test-namespace", secrets)

			Expect(sa.Secrets).To(Equal(secrets))
		})

		It("should return correct service account metadata", func() {
			Expect(typesv1.IsServiceAccountNamespaceScoped()).To(BeTrue())
			Expect(typesv1.GetServiceAccountShortNames()).To(ContainElement("sa"))
			Expect(typesv1.GetServiceAccountCategories()).To(ContainElement("all"))
			Expect(typesv1.GetServiceAccountPrintColumns()).ToNot(BeEmpty())
		})
	})

	Describe("Event", func() {
		It("should create a new normal event with correct metadata", func() {
			involvedObject := corev1.ObjectReference{
				Kind:      "Pod",
				Namespace: "test-namespace",
				Name:      "test-pod",
			}
			event := typesv1.NewNormalEvent("test-namespace", "test-event", "Created", "Pod was created", involvedObject)

			Expect(event.Name).To(Equal("test-event"))
			Expect(event.Namespace).To(Equal("test-namespace"))
			Expect(event.APIVersion).To(Equal("v1"))
			Expect(event.Kind).To(Equal("Event"))
			Expect(event.Type).To(Equal(corev1.EventTypeNormal))
			Expect(event.Reason).To(Equal("Created"))
			Expect(event.Message).To(Equal("Pod was created"))
			Expect(event.InvolvedObject).To(Equal(involvedObject))
			Expect(event.Count).To(Equal(int32(1)))
		})

		It("should create a warning event", func() {
			involvedObject := corev1.ObjectReference{
				Kind:      "Pod",
				Namespace: "test-namespace",
				Name:      "test-pod",
			}
			event := typesv1.NewWarningEvent("test-namespace", "test-event", "Failed", "Pod failed to start", involvedObject)

			Expect(event.Type).To(Equal(corev1.EventTypeWarning))
			Expect(event.Reason).To(Equal("Failed"))
			Expect(event.Message).To(Equal("Pod failed to start"))
		})

		It("should return correct event metadata", func() {
			Expect(typesv1.IsEventNamespaceScoped()).To(BeTrue())
			Expect(typesv1.GetEventShortNames()).To(ContainElement("ev"))
			Expect(typesv1.GetEventCategories()).To(ContainElement("all"))
			Expect(typesv1.GetEventPrintColumns()).ToNot(BeEmpty())
		})
	})

	Describe("Resource Info", func() {
		It("should return all core resource infos", func() {
			infos := typesv1.GetCoreResourceInfos()

			Expect(infos).To(HaveKey("Namespace"))
			Expect(infos).To(HaveKey("ConfigMap"))
			Expect(infos).To(HaveKey("Secret"))
			Expect(infos).To(HaveKey("ServiceAccount"))
			Expect(infos).To(HaveKey("Event"))

			// Verify namespace info
			nsInfo := infos["Namespace"]
			Expect(nsInfo.Singular).To(Equal("namespace"))
			Expect(nsInfo.Plural).To(Equal("namespaces"))
			Expect(nsInfo.NamespaceScoped).To(BeFalse())

			// Verify configmap info
			cmInfo := infos["ConfigMap"]
			Expect(cmInfo.Singular).To(Equal("configmap"))
			Expect(cmInfo.Plural).To(Equal("configmaps"))
			Expect(cmInfo.NamespaceScoped).To(BeTrue())
		})

		It("should find resource info by GVK", func() {
			gvk := typesv1.GetNamespaceGVK()
			info, found := typesv1.GetResourceInfoByGVK(gvk)

			Expect(found).To(BeTrue())
			Expect(info.GVK).To(Equal(gvk))
			Expect(info.Singular).To(Equal("namespace"))
		})

		It("should find resource info by GVR", func() {
			gvr := typesv1.GetConfigMapGVR()
			info, found := typesv1.GetResourceInfoByGVR(gvr)

			Expect(found).To(BeTrue())
			Expect(info.GVR).To(Equal(gvr))
			Expect(info.Singular).To(Equal("configmap"))
		})

		It("should find resource info by kind", func() {
			info, found := typesv1.GetResourceInfoByKind("Secret")

			Expect(found).To(BeTrue())
			Expect(info.GVK.Kind).To(Equal("Secret"))
			Expect(info.Singular).To(Equal("secret"))
		})

		It("should return false for non-existent resources", func() {
			_, found := typesv1.GetResourceInfoByKind("NonExistentKind")
			Expect(found).To(BeFalse())
		})
	})

	Describe("GVK/GVR Mappings", func() {
		It("should return all GVKs", func() {
			gvks := typesv1.GetAllGVKs()

			Expect(gvks).To(ContainElement(typesv1.GetNamespaceGVK()))
			Expect(gvks).To(ContainElement(typesv1.GetConfigMapGVK()))
			Expect(gvks).To(ContainElement(typesv1.GetSecretGVK()))
			Expect(gvks).To(ContainElement(typesv1.GetServiceAccountGVK()))
			Expect(gvks).To(ContainElement(typesv1.GetEventGVK()))
		})

		It("should return all GVRs", func() {
			gvrs := typesv1.GetAllGVRs()

			Expect(gvrs).To(ContainElement(typesv1.GetNamespaceGVR()))
			Expect(gvrs).To(ContainElement(typesv1.GetConfigMapGVR()))
			Expect(gvrs).To(ContainElement(typesv1.GetSecretGVR()))
			Expect(gvrs).To(ContainElement(typesv1.GetServiceAccountGVR()))
			Expect(gvrs).To(ContainElement(typesv1.GetEventGVR()))
		})

		It("should return correct GVK to GVR mappings", func() {
			mappings := typesv1.GetGVKToGVRMappings()

			Expect(mappings[typesv1.GetNamespaceGVK()]).To(Equal(typesv1.GetNamespaceGVR()))
			Expect(mappings[typesv1.GetConfigMapGVK()]).To(Equal(typesv1.GetConfigMapGVR()))
			Expect(mappings[typesv1.GetSecretGVK()]).To(Equal(typesv1.GetSecretGVR()))
		})

		It("should return correct GVR to GVK mappings", func() {
			mappings := typesv1.GetGVRToGVKMappings()

			Expect(mappings[typesv1.GetNamespaceGVR()]).To(Equal(typesv1.GetNamespaceGVK()))
			Expect(mappings[typesv1.GetConfigMapGVR()]).To(Equal(typesv1.GetConfigMapGVK()))
			Expect(mappings[typesv1.GetSecretGVR()]).To(Equal(typesv1.GetSecretGVK()))
		})
	})
})
