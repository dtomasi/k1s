package codec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

// JSONCodec implements the Codec interface for JSON serialization.
type JSONCodec struct {
	scheme *runtime.Scheme
	pretty bool
}

// NewJSONCodec creates a new JSON codec with the given scheme.
func NewJSONCodec(scheme *runtime.Scheme) *JSONCodec {
	return &JSONCodec{
		scheme: scheme,
		pretty: false,
	}
}

// NewPrettyJSONCodec creates a new JSON codec with pretty printing enabled.
func NewPrettyJSONCodec(scheme *runtime.Scheme) *JSONCodec {
	return &JSONCodec{
		scheme: scheme,
		pretty: true,
	}
}

// Encode encodes the given object to JSON and writes it to the writer.
func (c *JSONCodec) Encode(obj runtime.Object, w io.Writer) error {
	if obj == nil {
		return fmt.Errorf("cannot encode nil object")
	}

	// Ensure the object has proper TypeMeta
	if err := c.ensureTypeMeta(obj); err != nil {
		return fmt.Errorf("failed to ensure TypeMeta: %w", err)
	}

	var data []byte
	var err error

	if c.pretty {
		data, err = json.MarshalIndent(obj, "", "  ")
	} else {
		data, err = json.Marshal(obj)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal object to JSON: %w", err)
	}

	_, err = w.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write JSON data: %w", err)
	}

	return nil
}

// Decode decodes JSON data into a runtime.Object.
func (c *JSONCodec) Decode(data []byte, defaults *schema.GroupVersionKind, into runtime.Object) (runtime.Object, *schema.GroupVersionKind, error) {
	return decodeObject(data, defaults, into, json.Unmarshal, c.scheme, "JSON")
}

// Identifier returns a unique identifier for this codec.
func (c *JSONCodec) Identifier() runtime.Identifier {
	return runtime.Identifier("json")
}

// ensureTypeMeta ensures the object has proper TypeMeta set for JSON encoding.
func (c *JSONCodec) ensureTypeMeta(obj runtime.Object) error {
	return ensureTypeMeta(obj, c.scheme)
}

// JSONSerializer implements runtime.Serializer for JSON encoding/decoding.
type JSONSerializer struct {
	*JSONCodec
}

// NewJSONSerializer creates a new JSON serializer.
func NewJSONSerializer(scheme *runtime.Scheme) *JSONSerializer {
	return &JSONSerializer{
		JSONCodec: NewJSONCodec(scheme),
	}
}

// Decode implements runtime.Decoder interface.
func (s *JSONSerializer) Decode(data []byte, defaults *schema.GroupVersionKind, into runtime.Object) (runtime.Object, *schema.GroupVersionKind, error) {
	return s.JSONCodec.Decode(data, defaults, into)
}

// Encode implements runtime.Encoder interface.
func (s *JSONSerializer) Encode(obj runtime.Object, w io.Writer) error {
	return s.JSONCodec.Encode(obj, w)
}

// Identifier implements runtime.Serializer interface.
func (s *JSONSerializer) Identifier() runtime.Identifier {
	return s.JSONCodec.Identifier()
}

// RecognizesData implements runtime.RecognizingDecoder interface.
func (s *JSONSerializer) RecognizesData(data []byte) (bool, bool, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return false, false, nil
	}

	// JSON data should start with '{' or '['
	if data[0] == '{' || data[0] == '[' {
		return true, false, nil
	}

	return false, false, nil
}

func init() {
	// Register JSON as a known serializer with the runtime
	utilruntime.Must(nil)
}
