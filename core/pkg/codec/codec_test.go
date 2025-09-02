package codec_test

import (
	"bytes"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"

	"github.com/dtomasi/k1s/core/pkg/codec"
	k1sruntime "github.com/dtomasi/k1s/core/pkg/runtime"
)

// TestObject is a simple object for testing serialization
type TestObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	
	Spec TestObjectSpec `json:"spec,omitempty"`
}

type TestObjectSpec struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Value       int32  `json:"value"`
}

// DeepCopyObject implements runtime.Object
func (t *TestObject) DeepCopyObject() runtime.Object {
	if t == nil {
		return nil
	}
	out := new(TestObject)
	t.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all properties from this object into another object of the same type
func (t *TestObject) DeepCopyInto(out *TestObject) {
	*out = *t
	t.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
}

// TestObjectList is a list of TestObjects
type TestObjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TestObject `json:"items"`
}

// DeepCopyObject implements runtime.Object
func (t *TestObjectList) DeepCopyObject() runtime.Object {
	if t == nil {
		return nil
	}
	out := new(TestObjectList)
	t.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all properties from this object into another object of the same type
func (t *TestObjectList) DeepCopyInto(out *TestObjectList) {
	*out = *t
	t.ListMeta.DeepCopyInto(&out.ListMeta)
	if t.Items != nil {
		in, out := &t.Items, &out.Items
		*out = make([]TestObject, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

var _ = Describe("Codec", func() {
	var (
		scheme       *runtime.Scheme
		codecFactory *codec.CodecFactory
		testObj      *TestObject
		testGVK      schema.GroupVersionKind
	)

	BeforeEach(func() {
		// Create a new scheme and register our test types
		scheme = k1sruntime.NewScheme()
		
		// Define the GVK for our test object
		testGV := schema.GroupVersion{Group: "test.k1s.io", Version: "v1alpha1"}
		testGVK = testGV.WithKind("TestObject")
		
		// Register test types
		scheme.AddKnownTypes(testGV, &TestObject{}, &TestObjectList{})
		metav1.AddToGroupVersion(scheme, testGV)
		
		// Create codec factory
		codecFactory = codec.NewCodecFactory(scheme)
		
		// Create test object
		testObj = &TestObject{
			TypeMeta: metav1.TypeMeta{
				APIVersion: testGV.String(),
				Kind:       "TestObject",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-object",
				Namespace: "default",
			},
			Spec: TestObjectSpec{
				Name:        "Test Item",
				Description: "A test object for codec testing",
				Value:       42,
			},
		}
	})

	Describe("CodecFactory", func() {
		It("should support JSON and YAML media types", func() {
			mediaTypes := codecFactory.SupportedMediaTypes()
			Expect(mediaTypes).To(HaveLen(2))
			
			mediaTypeNames := make([]string, len(mediaTypes))
			for i, mt := range mediaTypes {
				mediaTypeNames[i] = mt.MediaType
			}
			
			Expect(mediaTypeNames).To(ContainElements("application/json", "application/yaml"))
		})

		It("should create universal decoder", func() {
			decoder := codecFactory.UniversalDecoder()
			Expect(decoder).ToNot(BeNil())
			// Universal decoder returns JSONCodec which has Identifier method
			if jsonCodec, ok := decoder.(*codec.JSONCodec); ok {
				Expect(jsonCodec.Identifier()).To(Equal(runtime.Identifier("json")))
			}
		})

		It("should create universal deserializer", func() {
			deserializer := codecFactory.UniversalDeserializer()
			Expect(deserializer).ToNot(BeNil())
			// Universal deserializer has Identifier method
			if univDeser, ok := deserializer.(*codec.UniversalDeserializer); ok {
				Expect(univDeser.Identifier()).To(Equal(runtime.Identifier("universal")))
			}
		})
	})

	Describe("JSON Codec", func() {
		var jsonCodec *codec.JSONCodec

		BeforeEach(func() {
			jsonCodec = codec.NewJSONCodec(scheme)
		})

		It("should have correct identifier", func() {
			Expect(jsonCodec.Identifier()).To(Equal(runtime.Identifier("json")))
		})

		It("should encode object to JSON", func() {
			var buf bytes.Buffer
			err := jsonCodec.Encode(testObj, &buf)
			Expect(err).ToNot(HaveOccurred())
			
			data := buf.Bytes()
			Expect(data).ToNot(BeEmpty())
			
			// Verify it's valid JSON
			var jsonObj map[string]interface{}
			err = json.Unmarshal(data, &jsonObj)
			Expect(err).ToNot(HaveOccurred())
			
			// Check key fields
			Expect(jsonObj["apiVersion"]).To(Equal("test.k1s.io/v1alpha1"))
			Expect(jsonObj["kind"]).To(Equal("TestObject"))
			Expect(jsonObj["metadata"].(map[string]interface{})["name"]).To(Equal("test-object"))
		})

		It("should decode JSON to object", func() {
			// First encode to get JSON data
			var buf bytes.Buffer
			err := jsonCodec.Encode(testObj, &buf)
			Expect(err).ToNot(HaveOccurred())
			
			// Now decode it back
			decoded, gvk, err := jsonCodec.Decode(buf.Bytes(), nil, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(gvk).ToNot(BeNil())
			Expect(*gvk).To(Equal(testGVK))
			
			decodedObj, ok := decoded.(*TestObject)
			Expect(ok).To(BeTrue())
			Expect(decodedObj.Name).To(Equal(testObj.Name))
			Expect(decodedObj.Namespace).To(Equal(testObj.Namespace))
			Expect(decodedObj.Spec.Name).To(Equal(testObj.Spec.Name))
			Expect(decodedObj.Spec.Value).To(Equal(testObj.Spec.Value))
		})

		It("should decode JSON into provided object", func() {
			// Encode test object
			var buf bytes.Buffer
			err := jsonCodec.Encode(testObj, &buf)
			Expect(err).ToNot(HaveOccurred())
			
			// Decode into pre-allocated object
			into := &TestObject{}
			decoded, gvk, err := jsonCodec.Decode(buf.Bytes(), nil, into)
			Expect(err).ToNot(HaveOccurred())
			Expect(gvk).ToNot(BeNil())
			Expect(decoded).To(Equal(into))
			
			// Verify content
			Expect(into.Name).To(Equal(testObj.Name))
			Expect(into.Spec.Name).To(Equal(testObj.Spec.Name))
		})

		It("should handle encoding nil object gracefully", func() {
			var buf bytes.Buffer
			err := jsonCodec.Encode(nil, &buf)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot encode nil object"))
		})

		It("should handle decoding empty data gracefully", func() {
			_, _, err := jsonCodec.Decode([]byte{}, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot decode empty data"))
		})

		It("should use defaults when TypeMeta is incomplete", func() {
			incompleteJSON := `{"metadata":{"name":"test"},"spec":{"name":"Test","value":123}}`
			
			defaultGVK := testGVK
			decoded, gvk, err := jsonCodec.Decode([]byte(incompleteJSON), &defaultGVK, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(*gvk).To(Equal(defaultGVK))
			
			decodedObj := decoded.(*TestObject)
			Expect(decodedObj.Spec.Name).To(Equal("Test"))
			Expect(decodedObj.Spec.Value).To(Equal(int32(123)))
		})
	})

	Describe("YAML Codec", func() {
		var yamlCodec *codec.YAMLCodec

		BeforeEach(func() {
			yamlCodec = codec.NewYAMLCodec(scheme)
		})

		It("should have correct identifier", func() {
			Expect(yamlCodec.Identifier()).To(Equal(runtime.Identifier("yaml")))
		})

		It("should encode object to YAML", func() {
			var buf bytes.Buffer
			err := yamlCodec.Encode(testObj, &buf)
			Expect(err).ToNot(HaveOccurred())
			
			data := buf.Bytes()
			Expect(data).ToNot(BeEmpty())
			
			// Verify it's valid YAML by unmarshaling
			var yamlObj map[string]interface{}
			err = yaml.Unmarshal(data, &yamlObj)
			Expect(err).ToNot(HaveOccurred())
			
			// Check key fields
			Expect(yamlObj["apiVersion"]).To(Equal("test.k1s.io/v1alpha1"))
			Expect(yamlObj["kind"]).To(Equal("TestObject"))
		})

		It("should decode YAML to object", func() {
			// First encode to get YAML data
			var buf bytes.Buffer
			err := yamlCodec.Encode(testObj, &buf)
			Expect(err).ToNot(HaveOccurred())
			
			// Now decode it back
			decoded, gvk, err := yamlCodec.Decode(buf.Bytes(), nil, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(gvk).ToNot(BeNil())
			Expect(*gvk).To(Equal(testGVK))
			
			decodedObj, ok := decoded.(*TestObject)
			Expect(ok).To(BeTrue())
			Expect(decodedObj.Name).To(Equal(testObj.Name))
			Expect(decodedObj.Spec.Name).To(Equal(testObj.Spec.Name))
			Expect(decodedObj.Spec.Value).To(Equal(testObj.Spec.Value))
		})

		It("should handle YAML document separator", func() {
			yamlData := `---
apiVersion: test.k1s.io/v1alpha1
kind: TestObject
metadata:
  name: test-yaml
spec:
  name: YAML Test
  value: 999`

			decoded, gvk, err := yamlCodec.Decode([]byte(yamlData), nil, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(*gvk).To(Equal(testGVK))
			
			decodedObj := decoded.(*TestObject)
			Expect(decodedObj.Name).To(Equal("test-yaml"))
			Expect(decodedObj.Spec.Name).To(Equal("YAML Test"))
			Expect(decodedObj.Spec.Value).To(Equal(int32(999)))
		})
	})

	Describe("UniversalDeserializer", func() {
		var deserializer *codec.UniversalDeserializer

		BeforeEach(func() {
			deserializer = codec.NewUniversalDeserializer(scheme)
		})

		It("should have correct identifier", func() {
			Expect(deserializer.Identifier()).To(Equal(runtime.Identifier("universal")))
		})

		It("should decode JSON data", func() {
			jsonData := `{
				"apiVersion": "test.k1s.io/v1alpha1",
				"kind": "TestObject",
				"metadata": {"name": "json-test"},
				"spec": {"name": "JSON Test", "value": 100}
			}`

			decoded, gvk, err := deserializer.Decode([]byte(jsonData), nil, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(*gvk).To(Equal(testGVK))
			
			decodedObj := decoded.(*TestObject)
			Expect(decodedObj.Name).To(Equal("json-test"))
			Expect(decodedObj.Spec.Name).To(Equal("JSON Test"))
			Expect(decodedObj.Spec.Value).To(Equal(int32(100)))
		})

		It("should decode YAML data", func() {
			yamlData := `apiVersion: test.k1s.io/v1alpha1
kind: TestObject
metadata:
  name: yaml-test
spec:
  name: YAML Test
  value: 200`

			decoded, gvk, err := deserializer.Decode([]byte(yamlData), nil, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(*gvk).To(Equal(testGVK))
			
			decodedObj := decoded.(*TestObject)
			Expect(decodedObj.Name).To(Equal("yaml-test"))
			Expect(decodedObj.Spec.Name).To(Equal("YAML Test"))
			Expect(decodedObj.Spec.Value).To(Equal(int32(200)))
		})

		It("should handle empty data gracefully", func() {
			_, _, err := deserializer.Decode([]byte{}, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot decode empty data"))
		})
	})

	Describe("JSONSerializer", func() {
		var serializer *codec.JSONSerializer

		BeforeEach(func() {
			serializer = codec.NewJSONSerializer(scheme)
		})

		It("should recognize JSON data", func() {
			jsonData := `{"apiVersion":"test.k1s.io/v1alpha1","kind":"TestObject"}`
			recognized, unknown, err := serializer.RecognizesData([]byte(jsonData))
			Expect(err).ToNot(HaveOccurred())
			Expect(recognized).To(BeTrue())
			Expect(unknown).To(BeFalse())
		})

		It("should not recognize non-JSON data", func() {
			nonJsonData := `apiVersion: test.k1s.io/v1alpha1`
			recognized, unknown, err := serializer.RecognizesData([]byte(nonJsonData))
			Expect(err).ToNot(HaveOccurred())
			Expect(recognized).To(BeFalse())
			Expect(unknown).To(BeFalse())
		})

		It("should handle empty data", func() {
			recognized, unknown, err := serializer.RecognizesData([]byte{})
			Expect(err).ToNot(HaveOccurred())
			Expect(recognized).To(BeFalse())
			Expect(unknown).To(BeFalse())
		})
	})

	Describe("YAMLSerializer", func() {
		var serializer *codec.YAMLSerializer

		BeforeEach(func() {
			serializer = codec.NewYAMLSerializer(scheme)
		})

		It("should recognize YAML data with document separator", func() {
			yamlData := `---
apiVersion: test.k1s.io/v1alpha1
kind: TestObject`
			recognized, unknown, err := serializer.RecognizesData([]byte(yamlData))
			Expect(err).ToNot(HaveOccurred())
			Expect(recognized).To(BeTrue())
			Expect(unknown).To(BeFalse())
		})

		It("should recognize YAML data with key-value pairs", func() {
			yamlData := `apiVersion: test.k1s.io/v1alpha1
kind: TestObject`
			recognized, unknown, err := serializer.RecognizesData([]byte(yamlData))
			Expect(err).ToNot(HaveOccurred())
			Expect(recognized).To(BeTrue())
			Expect(unknown).To(BeFalse())
		})

		It("should not recognize JSON data", func() {
			jsonData := `{"apiVersion":"test.k1s.io/v1alpha1"}`
			recognized, unknown, err := serializer.RecognizesData([]byte(jsonData))
			Expect(err).ToNot(HaveOccurred())
			Expect(recognized).To(BeFalse())
			Expect(unknown).To(BeFalse())
		})
	})

	Describe("Error Handling", func() {
		var jsonCodec *codec.JSONCodec

		BeforeEach(func() {
			jsonCodec = codec.NewJSONCodec(scheme)
		})

		It("should handle unregistered types gracefully", func() {
			unregisteredGVK := schema.GroupVersionKind{
				Group:   "unknown.k1s.io",
				Version: "v1",
				Kind:    "Unknown",
			}

			unknownJSON := fmt.Sprintf(`{
				"apiVersion": "%s",
				"kind": "%s",
				"metadata": {"name": "test"}
			}`, unregisteredGVK.GroupVersion().String(), unregisteredGVK.Kind)

			_, _, err := jsonCodec.Decode([]byte(unknownJSON), nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no kind"))
		})

		It("should handle malformed JSON", func() {
			malformedJSON := `{"apiVersion":"test.k1s.io/v1alpha1","kind":"TestObject",}`
			
			_, _, err := jsonCodec.Decode([]byte(malformedJSON), nil, nil)
			Expect(err).To(HaveOccurred())
		})

		It("should handle write errors", func() {
			// Create a writer that always fails
			failingWriter := &failingWriter{}
			
			err := jsonCodec.Encode(testObj, failingWriter)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to write"))
		})
	})
})

// failingWriter is a writer that always returns an error
type failingWriter struct{}

func (w *failingWriter) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("write failed")
}