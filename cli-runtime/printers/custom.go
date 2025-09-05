package printers

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// customColumnsPrinter prints objects with custom columns.
type customColumnsPrinter struct {
	columns    []string
	provider   CustomColumnProvider
	noHeaders  bool
	showLabels bool
}

// CustomColumnProvider provides custom column definitions.
type CustomColumnProvider interface {
	GetCustomColumns(columns []string) []PrintColumn
}

// DefaultCustomColumnProvider provides basic custom column support.
type DefaultCustomColumnProvider struct{}

// GetCustomColumns converts column specifications to PrintColumn definitions.
func (p *DefaultCustomColumnProvider) GetCustomColumns(columns []string) []PrintColumn {
	var printColumns []PrintColumn

	for _, col := range columns {
		// Parse column specification (simplified - real implementation would be more sophisticated)
		parts := strings.Split(col, ":")
		if len(parts) >= 2 {
			printColumns = append(printColumns, PrintColumn{
				Name:     parts[0],
				Type:     "string",
				JSONPath: parts[1],
				Priority: 0, // Fixed priority to avoid overflow
			})
		}
	}

	return printColumns
}

// NewCustomColumnsPrinter creates a new custom columns printer.
func NewCustomColumnsPrinter(columns []string) Printer {
	return &customColumnsPrinter{
		columns:  columns,
		provider: &DefaultCustomColumnProvider{},
	}
}

// PrintObj prints an object with custom columns.
func (p *customColumnsPrinter) PrintObj(obj runtime.Object, writer io.Writer) error {
	// Check if this is a list or single object
	if objList := p.extractList(obj); objList != nil {
		return p.printList(objList, writer)
	}
	return p.printSingle(obj, writer)
}

// printList prints a list of objects with custom columns.
func (p *customColumnsPrinter) printList(objects []runtime.Object, writer io.Writer) error {
	if len(objects) == 0 {
		return nil
	}

	// Print header if not disabled
	if !p.noHeaders {
		err := p.printHeader(writer)
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

// printSingle prints a single object with custom columns.
func (p *customColumnsPrinter) printSingle(obj runtime.Object, writer io.Writer) error {
	// Print header if not disabled
	if !p.noHeaders {
		err := p.printHeader(writer)
		if err != nil {
			return err
		}
	}

	return p.printObjectRow(obj, writer)
}

// printHeader prints the custom column headers.
func (p *customColumnsPrinter) printHeader(writer io.Writer) error {
	printColumns := p.provider.GetCustomColumns(p.columns)

	var headers []string
	for _, col := range printColumns {
		headers = append(headers, strings.ToUpper(col.Name))
	}

	if p.showLabels {
		headers = append(headers, "LABELS")
	}

	header := strings.Join(headers, "\t")
	_, err := fmt.Fprintln(writer, header)
	return err
}

// printObjectRow prints a single object as a custom column row.
func (p *customColumnsPrinter) printObjectRow(obj runtime.Object, writer io.Writer) error {
	printColumns := p.provider.GetCustomColumns(p.columns)

	var values []string
	for _, col := range printColumns {
		value := extractColumnValue(obj, col)
		values = append(values, value)
	}

	// Add labels if requested
	if p.showLabels {
		if objMeta, ok := obj.(metav1.Object); ok {
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
	}

	row := strings.Join(values, "\t")
	_, err := fmt.Fprintln(writer, row)
	return err
}

// extractList extracts objects from a list (reusing table printer logic).
func (p *customColumnsPrinter) extractList(obj runtime.Object) []runtime.Object {
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
