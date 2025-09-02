package codec

import (
	"io"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Codec defines the interface for encoding and decoding Kubernetes objects.
// This interface is compatible with Kubernetes runtime.Codec to ensure
// seamless integration with the Kubernetes ecosystem.
type Codec interface {
	// Encode writes the provided object to the given writer.
	// The object must be registered in the scheme.
	Encode(obj runtime.Object, w io.Writer) error

	// Decode attempts to convert the provided data into a runtime.Object.
	// If defaults are provided, they are applied to the decoded object.
	// Returns the decoded object, its GVK, and any error encountered.
	Decode(data []byte, defaults *schema.GroupVersionKind, into runtime.Object) (runtime.Object, *schema.GroupVersionKind, error)

	// Identifier returns a unique identifier for this codec.
	// This is used by the Kubernetes runtime to identify different codecs.
	Identifier() runtime.Identifier
}

// Factory creates codec instances for different media types.
type Factory interface {
	// SupportedMediaTypes returns the list of media types this factory can handle.
	SupportedMediaTypes() []runtime.SerializerInfo

	// EncoderForVersion creates an encoder for the specified version.
	EncoderForVersion(encoder runtime.Encoder, gv runtime.GroupVersioner) runtime.Encoder

	// DecoderToVersion creates a decoder for the specified version.
	DecoderToVersion(decoder runtime.Decoder, gv runtime.GroupVersioner) runtime.Decoder
}

// CodecFactory implements the Factory interface and provides
// JSON and YAML codecs for Kubernetes objects.
type CodecFactory struct {
	scheme *runtime.Scheme
}

// NewCodecFactory creates a new codec factory with the given scheme.
func NewCodecFactory(scheme *runtime.Scheme) *CodecFactory {
	return &CodecFactory{
		scheme: scheme,
	}
}

// SupportedMediaTypes returns the media types supported by this factory.
func (f *CodecFactory) SupportedMediaTypes() []runtime.SerializerInfo {
	return []runtime.SerializerInfo{
		{
			MediaType:        "application/json",
			MediaTypeType:    "application",
			MediaTypeSubType: "json",
			EncodesAsText:    true,
			Serializer:       NewJSONCodec(f.scheme),
			PrettySerializer: NewJSONCodec(f.scheme),
			StrictSerializer: NewJSONCodec(f.scheme),
		},
		{
			MediaType:        "application/yaml",
			MediaTypeType:    "application", 
			MediaTypeSubType: "yaml",
			EncodesAsText:    true,
			Serializer:       NewYAMLCodec(f.scheme),
			PrettySerializer: NewYAMLCodec(f.scheme),
			StrictSerializer: NewYAMLCodec(f.scheme),
		},
	}
}

// EncoderForVersion creates an encoder for the specified version.
func (f *CodecFactory) EncoderForVersion(encoder runtime.Encoder, gv runtime.GroupVersioner) runtime.Encoder {
	return encoder
}

// DecoderToVersion creates a decoder for the specified version.
func (f *CodecFactory) DecoderToVersion(decoder runtime.Decoder, gv runtime.GroupVersioner) runtime.Decoder {
	return decoder
}

// LegacyCodec creates a codec that can handle legacy versions.
func (f *CodecFactory) LegacyCodec(version ...schema.GroupVersion) runtime.Codec {
	return NewJSONCodec(f.scheme)
}

// UniversalDecoder creates a decoder that can handle any supported version.
func (f *CodecFactory) UniversalDecoder(versions ...schema.GroupVersion) runtime.Decoder {
	return NewJSONCodec(f.scheme)
}

// UniversalDeserializer creates a deserializer that can handle any supported format.
func (f *CodecFactory) UniversalDeserializer() runtime.Decoder {
	return NewUniversalDeserializer(f.scheme)
}

// CodecForVersions creates a codec that can encode for one version and decode from another.
func (f *CodecFactory) CodecForVersions(encoder runtime.Encoder, decoder runtime.Decoder, encode runtime.GroupVersioner, decode runtime.GroupVersioner) runtime.Codec {
	return NewJSONCodec(f.scheme)
}

// WithoutConversion returns the same factory (k1s doesn't need conversion for now).
func (f *CodecFactory) WithoutConversion() runtime.NegotiatedSerializer {
	return f
}