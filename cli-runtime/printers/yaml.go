package printers

import (
	"encoding/json"
	"io"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

// yamlPrinter prints objects as YAML.
type yamlPrinter struct{}

// NewYAMLPrinter creates a new YAML printer.
func NewYAMLPrinter() Printer {
	return &yamlPrinter{}
}

// PrintObj prints an object as YAML.
func (p *yamlPrinter) PrintObj(obj runtime.Object, writer io.Writer) error {
	// Convert runtime.Object to JSON first
	data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return err
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Convert JSON to YAML
	yamlData, err := yaml.JSONToYAML(jsonData)
	if err != nil {
		return err
	}

	_, err = writer.Write(yamlData)
	return err
}
