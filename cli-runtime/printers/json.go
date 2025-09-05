package printers

import (
	"encoding/json"
	"io"

	"k8s.io/apimachinery/pkg/runtime"
)

// jsonPrinter prints objects as JSON.
type jsonPrinter struct{}

// NewJSONPrinter creates a new JSON printer.
func NewJSONPrinter() Printer {
	return &jsonPrinter{}
}

// PrintObj prints an object as JSON.
func (p *jsonPrinter) PrintObj(obj runtime.Object, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(obj)
}
