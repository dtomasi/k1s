package codec

import (
	"bytes"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

// YAMLCodec implements the Codec interface for YAML serialization.
type YAMLCodec struct {
	scheme *runtime.Scheme
}

// NewYAMLCodec creates a new YAML codec with the given scheme.
func NewYAMLCodec(scheme *runtime.Scheme) *YAMLCodec {
	return &YAMLCodec{
		scheme: scheme,
	}
}

// Encode encodes the given object to YAML and writes it to the writer.
func (c *YAMLCodec) Encode(obj runtime.Object, w io.Writer) error {
	if obj == nil {
		return fmt.Errorf("cannot encode nil object")
	}

	// Ensure the object has proper TypeMeta
	if err := c.ensureTypeMeta(obj); err != nil {
		return fmt.Errorf("failed to ensure TypeMeta: %w", err)
	}

	data, err := yaml.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal object to YAML: %w", err)
	}

	_, err = w.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write YAML data: %w", err)
	}

	return nil
}

// Decode decodes YAML data into a runtime.Object.
func (c *YAMLCodec) Decode(data []byte, defaults *schema.GroupVersionKind, into runtime.Object) (runtime.Object, *schema.GroupVersionKind, error) {
	yamlUnmarshal := func(data []byte, v interface{}) error {
		return yaml.Unmarshal(data, v)
	}
	return decodeObject(data, defaults, into, yamlUnmarshal, c.scheme, "YAML")
}

// Identifier returns a unique identifier for this codec.
func (c *YAMLCodec) Identifier() runtime.Identifier {
	return runtime.Identifier("yaml")
}

// ensureTypeMeta ensures the object has proper TypeMeta set for YAML encoding.
func (c *YAMLCodec) ensureTypeMeta(obj runtime.Object) error {
	return ensureTypeMeta(obj, c.scheme)
}

// YAMLSerializer implements runtime.Serializer for YAML encoding/decoding.
type YAMLSerializer struct {
	*YAMLCodec
}

// NewYAMLSerializer creates a new YAML serializer.
func NewYAMLSerializer(scheme *runtime.Scheme) *YAMLSerializer {
	return &YAMLSerializer{
		YAMLCodec: NewYAMLCodec(scheme),
	}
}

// Decode implements runtime.Decoder interface.
func (s *YAMLSerializer) Decode(data []byte, defaults *schema.GroupVersionKind, into runtime.Object) (runtime.Object, *schema.GroupVersionKind, error) {
	return s.YAMLCodec.Decode(data, defaults, into)
}

// Encode implements runtime.Encoder interface.
func (s *YAMLSerializer) Encode(obj runtime.Object, w io.Writer) error {
	return s.YAMLCodec.Encode(obj, w)
}

// Identifier implements runtime.Serializer interface.
func (s *YAMLSerializer) Identifier() runtime.Identifier {
	return s.YAMLCodec.Identifier()
}

// RecognizesData implements runtime.RecognizingDecoder interface.
func (s *YAMLSerializer) RecognizesData(data []byte) (bool, bool, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return false, false, nil
	}

	// Try to detect YAML by looking for YAML-specific patterns
	// YAML documents can start with "---" or contain key-value pairs
	if bytes.HasPrefix(data, []byte("---")) {
		return true, false, nil
	}

	// Look for key: value patterns (basic YAML detection)
	if bytes.Contains(data, []byte(":")) && !bytes.HasPrefix(data, []byte("{")) {
		return true, false, nil
	}

	return false, false, nil
}

// UniversalDeserializer can decode both JSON and YAML formats.
type UniversalDeserializer struct {
	scheme         *runtime.Scheme
	jsonCodec      *JSONCodec
	yamlCodec      *YAMLCodec
	jsonSerializer *JSONSerializer
	yamlSerializer *YAMLSerializer
}

// NewUniversalDeserializer creates a new universal deserializer.
func NewUniversalDeserializer(scheme *runtime.Scheme) *UniversalDeserializer {
	return &UniversalDeserializer{
		scheme:         scheme,
		jsonCodec:      NewJSONCodec(scheme),
		yamlCodec:      NewYAMLCodec(scheme),
		jsonSerializer: NewJSONSerializer(scheme),
		yamlSerializer: NewYAMLSerializer(scheme),
	}
}

// Decode attempts to decode data as either JSON or YAML.
func (d *UniversalDeserializer) Decode(data []byte, defaults *schema.GroupVersionKind, into runtime.Object) (runtime.Object, *schema.GroupVersionKind, error) {
	if len(data) == 0 {
		return nil, nil, fmt.Errorf("cannot decode empty data")
	}

	// Try JSON first
	if jsonRecognized, _, _ := d.jsonSerializer.RecognizesData(data); jsonRecognized {
		obj, gvk, err := d.jsonCodec.Decode(data, defaults, into)
		if err == nil {
			return obj, gvk, nil
		}
		// If JSON fails, continue to try YAML
	}

	// Try YAML
	if yamlRecognized, _, _ := d.yamlSerializer.RecognizesData(data); yamlRecognized {
		return d.yamlCodec.Decode(data, defaults, into)
	}

	// If neither format is recognized, try JSON as fallback
	return d.jsonCodec.Decode(data, defaults, into)
}

// Identifier returns a unique identifier for this deserializer.
func (d *UniversalDeserializer) Identifier() runtime.Identifier {
	return runtime.Identifier("universal")
}
