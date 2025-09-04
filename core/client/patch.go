package client

import (
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

// RawPatch represents a raw patch that can be applied to a Kubernetes object.
type RawPatch struct {
	PatchType types.PatchType
	PatchData []byte
}

// Type implements Patch interface.
func (r RawPatch) Type() types.PatchType {
	return r.PatchType
}

// Data implements Patch interface.
func (r RawPatch) Data(obj Object) ([]byte, error) {
	return r.PatchData, nil
}

// MergeFrom creates a patch that will merge the given object with the current state.
func MergeFrom(obj Object) Patch {
	return &mergeFromPatch{obj: obj}
}

// mergeFromPatch implements strategic merge patch functionality.
type mergeFromPatch struct {
	obj Object
}

// Type returns the patch type.
func (p *mergeFromPatch) Type() types.PatchType {
	return types.StrategicMergePatchType
}

// Data creates the strategic merge patch data.
func (p *mergeFromPatch) Data(obj Object) ([]byte, error) {
	originalData, err := json.Marshal(p.obj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal original object: %w", err)
	}

	modifiedData, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal modified object: %w", err)
	}

	patchData, err := strategicpatch.CreateTwoWayMergePatch(originalData, modifiedData, obj)
	if err != nil {
		return nil, fmt.Errorf("failed to create strategic merge patch: %w", err)
	}

	return patchData, nil
}

// Apply creates a patch that represents a server-side apply operation.
func Apply(obj Object, opts ...ApplyOption) Patch {
	config := &ApplyConfig{
		Force: false,
	}

	for _, opt := range opts {
		opt.ApplyToApply(config)
	}

	return &applyPatch{
		obj:    obj,
		config: config,
	}
}

// ApplyConfig contains configuration for apply patches.
type ApplyConfig struct {
	Force        bool
	FieldManager string
}

// ApplyOption configures apply operations.
type ApplyOption interface {
	ApplyToApply(*ApplyConfig)
}

// ForceOwnership forces the ownership of fields during apply operations.
type ForceOwnership struct{}

// ApplyToApply implements ApplyOption.
func (ForceOwnership) ApplyToApply(config *ApplyConfig) {
	config.Force = true
}

// FieldOwner sets the field manager name for apply operations.
type FieldOwner string

// ApplyToApply implements ApplyOption.
func (f FieldOwner) ApplyToApply(config *ApplyConfig) {
	config.FieldManager = string(f)
}

// applyPatch implements server-side apply patch functionality.
type applyPatch struct {
	obj    Object
	config *ApplyConfig
}

// Type returns the patch type.
func (p *applyPatch) Type() types.PatchType {
	return types.ApplyPatchType
}

// Data creates the apply patch data.
func (p *applyPatch) Data(obj Object) ([]byte, error) {
	// For apply patches, we use the modified object as the patch data
	// The server will handle the merge semantics
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal object for apply patch: %w", err)
	}
	return data, nil
}

// JSONPatch creates a JSON patch from a list of operations.
func JSONPatch(operations []JSONPatchOperation) Patch {
	return &jsonPatch{operations: operations}
}

// JSONPatchOperation represents a single JSON patch operation.
type JSONPatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
	From  string      `json:"from,omitempty"`
}

// jsonPatch implements JSON patch functionality.
type jsonPatch struct {
	operations []JSONPatchOperation
}

// Type returns the patch type.
func (p *jsonPatch) Type() types.PatchType {
	return types.JSONPatchType
}

// Data creates the JSON patch data.
func (p *jsonPatch) Data(obj Object) ([]byte, error) {
	data, err := json.Marshal(p.operations)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON patch operations: %w", err)
	}
	return data, nil
}
