package printers

import (
	"fmt"
	"io"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// namePrinter prints objects as name-only output.
type namePrinter struct{}

// NewNamePrinter creates a new name printer.
func NewNamePrinter() Printer {
	return &namePrinter{}
}

// PrintObj prints an object as name-only output.
func (p *namePrinter) PrintObj(obj runtime.Object, writer io.Writer) error {
	// Try to get object metadata
	objMeta, ok := obj.(metav1.Object)
	if !ok {
		return fmt.Errorf("object does not implement metav1.Object")
	}

	// Get GVK information
	gvk := obj.GetObjectKind().GroupVersionKind()

	// Format the name output
	name := fmt.Sprintf("%s/%s", gvk.Kind, objMeta.GetName())

	_, err := fmt.Fprintln(writer, name)
	return err
}
