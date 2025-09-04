// Package v1alpha1 contains API definitions for the k1s inventory system,
// demonstrating custom resource definitions (Items and Categories) with
// kubebuilder markers for validation and CLI generation.

// +kubebuilder:object:generate=true
// +groupName=examples.k1s.dtomasi.github.io

//go:generate controller-gen object paths=./...
//go:generate controller-gen crd paths=./... output:crd:dir=./../../config/crds
//go:generate go run ../../../tools/cmd/cli-gen paths=./...
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "examples.k1s.dtomasi.github.io", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
