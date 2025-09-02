package codec

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// unmarshalFunc represents a function that can unmarshal data into an object
type unmarshalFunc func(data []byte, v interface{}) error

// decodeObject performs the common decoding logic for both JSON and YAML codecs.
// It takes the data, defaults, target object, unmarshal function, scheme, and format name.
func decodeObject(data []byte, defaults *schema.GroupVersionKind, into runtime.Object, 
	unmarshal unmarshalFunc, scheme *runtime.Scheme, format string) (runtime.Object, *schema.GroupVersionKind, error) {
	
	if len(data) == 0 {
		return nil, nil, fmt.Errorf("cannot decode empty data")
	}

	// First, decode just the TypeMeta to determine the GVK
	var typeMeta runtime.TypeMeta
	if err := unmarshal(data, &typeMeta); err != nil {
		return nil, nil, fmt.Errorf("failed to decode TypeMeta: %w", err)
	}

	// Parse the GVK from TypeMeta
	gvk := schema.FromAPIVersionAndKind(typeMeta.APIVersion, typeMeta.Kind)

	// Use defaults if TypeMeta is incomplete
	if gvk.Empty() && defaults != nil {
		gvk = *defaults
	}

	if gvk.Empty() {
		return nil, nil, fmt.Errorf("cannot determine GVK from data and no defaults provided")
	}

	// Create the target object
	var obj runtime.Object
	var err error

	if into != nil {
		obj = into
	} else {
		obj, err = scheme.New(gvk)
		if err != nil {
			return nil, &gvk, fmt.Errorf("failed to create object for GVK %v: %w", gvk, err)
		}
	}

	// Decode the full object
	if err := unmarshal(data, obj); err != nil {
		return nil, &gvk, fmt.Errorf("failed to unmarshal %s into object: %w", format, err)
	}

	// Ensure the object has proper TypeMeta after decoding
	if err := ensureTypeMeta(obj, scheme); err != nil {
		return nil, &gvk, fmt.Errorf("failed to ensure TypeMeta after decode: %w", err)
	}

	return obj, &gvk, nil
}

// ensureTypeMeta ensures the object has proper TypeMeta set.
// This is extracted to avoid duplication between JSON and YAML codecs.
func ensureTypeMeta(obj runtime.Object, scheme *runtime.Scheme) error {
	if obj == nil {
		return fmt.Errorf("cannot set TypeMeta on nil object")
	}

	gvks, _, err := scheme.ObjectKinds(obj)
	if err != nil {
		return fmt.Errorf("failed to get object kinds: %w", err)
	}

	if len(gvks) == 0 {
		return fmt.Errorf("no GVK found for object type %T", obj)
	}

	// Use the first GVK (most specific)
	gvk := gvks[0]

	// Set TypeMeta if the object supports it
	if typed, ok := obj.(interface {
		SetAPIVersion(string)
		SetKind(string)
	}); ok {
		typed.SetAPIVersion(gvk.GroupVersion().String())
		typed.SetKind(gvk.Kind)
	}

	return nil
}