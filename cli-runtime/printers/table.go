package printers

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// tablePrinter prints objects in a table format.
type tablePrinter struct {
	options  *PrinterOptions
	provider ColumnDefinitionProvider
}

// NewTablePrinter creates a new table printer.
func NewTablePrinter(options *PrinterOptions) Printer {
	if options == nil {
		options = &PrinterOptions{}
	}
	return &tablePrinter{
		options:  options,
		provider: &DefaultColumnProvider{},
	}
}

// NewTablePrinterWithColumns creates a table printer with a custom column provider.
func NewTablePrinterWithColumns(options *PrinterOptions, provider ColumnDefinitionProvider) Printer {
	if options == nil {
		options = &PrinterOptions{}
	}
	if provider == nil {
		provider = &DefaultColumnProvider{}
	}
	return &tablePrinter{
		options:  options,
		provider: provider,
	}
}

// PrintObj prints an object in table format.
func (p *tablePrinter) PrintObj(obj runtime.Object, writer io.Writer) error {
	// Check if this is a list or single object
	if objList := p.extractList(obj); objList != nil {
		return p.printList(objList, writer)
	}
	return p.printSingle(obj, writer)
}

// printList prints a list of objects.
func (p *tablePrinter) printList(objects []runtime.Object, writer io.Writer) error {
	if len(objects) == 0 {
		return nil
	}

	// Print header if not disabled
	if !p.options.NoHeaders {
		err := p.printHeader(objects[0], writer)
		if err != nil {
			return err
		}
	}

	// Print each object
	for _, obj := range objects {
		err := p.printObjectRow(obj, writer)
		if err != nil {
			return err
		}
	}

	return nil
}

// printSingle prints a single object.
func (p *tablePrinter) printSingle(obj runtime.Object, writer io.Writer) error {
	// Print header if not disabled
	if !p.options.NoHeaders {
		err := p.printHeader(obj, writer)
		if err != nil {
			return err
		}
	}

	return p.printObjectRow(obj, writer)
}

// printHeader prints the table header.
func (p *tablePrinter) printHeader(obj runtime.Object, writer io.Writer) error {
	columns := p.getColumns(obj)
	header := strings.Join(columns, "\t")
	_, err := fmt.Fprintln(writer, header)
	return err
}

// printObjectRow prints a single object as a table row.
func (p *tablePrinter) printObjectRow(obj runtime.Object, writer io.Writer) error {
	values := p.getValues(obj)
	row := strings.Join(values, "\t")
	_, err := fmt.Fprintln(writer, row)
	return err
}

// getColumns returns the column headers for the table.
func (p *tablePrinter) getColumns(obj runtime.Object) []string {
	// Get the GVK from the object
	gvk := obj.GetObjectKind().GroupVersionKind()

	// Get column definitions from provider
	printColumns := p.provider.GetColumns(gvk, p.options.Wide)

	// Convert to string headers
	var columns []string
	for _, col := range printColumns {
		columns = append(columns, strings.ToUpper(col.Name))
	}

	// Add labels column if requested
	if p.options.ShowLabels {
		columns = append(columns, "LABELS")
	}

	return columns
}

// getValues returns the column values for an object.
func (p *tablePrinter) getValues(obj runtime.Object) []string {
	objMeta, ok := obj.(metav1.Object)
	if !ok {
		return []string{"<unknown>"}
	}

	// Get the GVK from the object
	gvk := obj.GetObjectKind().GroupVersionKind()

	// Get column definitions from provider
	printColumns := p.provider.GetColumns(gvk, p.options.Wide)

	// Extract values for each column
	var values []string
	for _, col := range printColumns {
		value := extractColumnValue(obj, col)
		values = append(values, value)
	}

	// Add labels if requested
	if p.options.ShowLabels {
		labels := objMeta.GetLabels()
		if len(labels) == 0 {
			values = append(values, "<none>")
		} else {
			var labelPairs []string
			for k, v := range labels {
				labelPairs = append(labelPairs, fmt.Sprintf("%s=%s", k, v))
			}
			values = append(values, strings.Join(labelPairs, ","))
		}
	}

	return values
}

// extractList extracts objects from a list.
func (p *tablePrinter) extractList(obj runtime.Object) []runtime.Object {
	// Use reflection to check if this is a list
	objValue := reflect.ValueOf(obj)
	if objValue.Kind() == reflect.Ptr {
		objValue = objValue.Elem()
	}

	// Look for Items field
	itemsField := objValue.FieldByName("Items")
	if !itemsField.IsValid() {
		return nil
	}

	// Convert to slice of runtime.Object
	if itemsField.Kind() == reflect.Slice {
		objects := make([]runtime.Object, 0, itemsField.Len())
		for i := 0; i < itemsField.Len(); i++ {
			item := itemsField.Index(i).Interface()
			if runtimeObj, ok := item.(runtime.Object); ok {
				objects = append(objects, runtimeObj)
			}
		}
		return objects
	}

	return nil
}

// formatAge formats a duration as an age string.
func formatAge(d time.Duration) string {
	switch {
	case d.Hours() > 24:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	case d.Hours() > 1:
		hours := int(d.Hours())
		return fmt.Sprintf("%dh", hours)
	case d.Minutes() > 1:
		minutes := int(d.Minutes())
		return fmt.Sprintf("%dm", minutes)
	default:
		seconds := int(d.Seconds())
		return fmt.Sprintf("%ds", seconds)
	}
}
