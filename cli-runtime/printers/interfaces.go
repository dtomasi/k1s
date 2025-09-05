// Package printers provides output formatters for CLI operations.
// These printers implement kubectl-compatible output formats including
// table, JSON, YAML, and name-only output.
package printers

import (
	"io"

	"k8s.io/apimachinery/pkg/runtime"
)

// Printer knows how to print objects.
type Printer interface {
	// PrintObj prints the given object to the writer
	PrintObj(obj runtime.Object, writer io.Writer) error
}

// PrinterFactory creates printers for different output formats.
type PrinterFactory struct {
	options *PrinterOptions
}

// PrinterOptions contains configuration for printers.
type PrinterOptions struct {
	// NoHeaders indicates whether to omit headers in table output
	NoHeaders bool
	// ShowLabels indicates whether to show labels in table output
	ShowLabels bool
	// Wide indicates whether to use wide output format
	Wide bool
	// CustomColumns specifies custom column definitions
	CustomColumns []string
	// AllowMissingTemplateKeys indicates whether to allow missing template keys
	AllowMissingTemplateKeys bool
}

// NewPrinterFactory creates a new printer factory with the given options.
func NewPrinterFactory(options *PrinterOptions) *PrinterFactory {
	if options == nil {
		options = &PrinterOptions{}
	}
	return &PrinterFactory{options: options}
}

// NewPrinter creates a printer for the specified format.
func (f *PrinterFactory) NewPrinter(format string) (Printer, error) {
	switch format {
	case "json":
		return NewJSONPrinter(), nil
	case "yaml":
		return NewYAMLPrinter(), nil
	case "name":
		return NewNamePrinter(), nil
	case "table":
		return NewTablePrinter(f.options), nil
	case "wide":
		opts := *f.options
		opts.Wide = true
		return NewTablePrinter(&opts), nil
	default:
		// Try custom columns
		if len(f.options.CustomColumns) > 0 {
			return NewCustomColumnsPrinter(f.options.CustomColumns), nil
		}
		return NewTablePrinter(f.options), nil
	}
}

// GetSupportedFormats returns the list of supported output formats.
func (f *PrinterFactory) GetSupportedFormats() []string {
	return []string{"json", "yaml", "name", "table", "wide"}
}
