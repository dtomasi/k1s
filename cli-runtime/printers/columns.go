package printers

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	unknownValue = "<unknown>"
)

// PrintColumn defines a single column in table output.
type PrintColumn struct {
	Name        string
	Type        string
	Description string
	JSONPath    string
	Priority    int32
}

// ColumnDefinitionProvider provides column definitions for resources.
type ColumnDefinitionProvider interface {
	GetColumns(gvk schema.GroupVersionKind, wide bool) []PrintColumn
}

// DefaultColumnProvider provides basic column definitions.
type DefaultColumnProvider struct{}

// GetColumns returns default columns for any resource type.
func (p *DefaultColumnProvider) GetColumns(gvk schema.GroupVersionKind, wide bool) []PrintColumn {
	columns := []PrintColumn{
		{Name: "Name", Type: "string", JSONPath: ".metadata.name"},
		{Name: "Ready", Type: "string", JSONPath: ".status.conditions[?(@.type==\"Ready\")].status"},
		{Name: "Status", Type: "string", JSONPath: ".status.phase"},
		{Name: "Restarts", Type: "integer", JSONPath: ".status.restartCount"},
		{Name: "Age", Type: "date", JSONPath: ".metadata.creationTimestamp"},
	}

	// Add namespace column if the resource is namespaced
	if p.isNamespaced(gvk) {
		namespacedColumns := []PrintColumn{
			{Name: "Namespace", Type: "string", JSONPath: ".metadata.namespace"},
		}
		columns = append(namespacedColumns, columns...)
	}

	// Add wide columns if requested
	if wide {
		columns = append(columns, []PrintColumn{
			{Name: "Node", Type: "string", JSONPath: ".spec.nodeName", Priority: 1},
			{Name: "IP", Type: "string", JSONPath: ".status.podIP", Priority: 1},
		}...)
	}

	return columns
}

// isNamespaced determines if a resource type is namespaced (simplified heuristic).
func (p *DefaultColumnProvider) isNamespaced(gvk schema.GroupVersionKind) bool {
	// Simple heuristic - most resources are namespaced
	// In a real implementation, this would query the API server or scheme
	namespacedKinds := map[string]bool{
		"Namespace":          false,
		"Node":               false,
		"PersistentVolume":   false,
		"ClusterRole":        false,
		"ClusterRoleBinding": false,
		"StorageClass":       false,
		"PriorityClass":      false,
		"VolumeAttachment":   false,
		"CSIDriver":          false,
		"CSINode":            false,
		"RuntimeClass":       false,
	}

	return !namespacedKinds[gvk.Kind]
}

// extractColumnValue extracts a value from an object using a JSONPath-like expression.
func extractColumnValue(obj runtime.Object, column PrintColumn) string {
	objMeta, ok := obj.(metav1.Object)
	if !ok {
		return unknownValue
	}

	switch column.JSONPath {
	case ".metadata.name":
		return objMeta.GetName()
	case ".metadata.namespace":
		return objMeta.GetNamespace()
	case ".metadata.creationTimestamp":
		if !objMeta.GetCreationTimestamp().Time.IsZero() {
			age := time.Since(objMeta.GetCreationTimestamp().Time).Round(time.Second)
			return formatAge(age)
		}
		return unknownValue
	case ".metadata.labels":
		labels := objMeta.GetLabels()
		if len(labels) == 0 {
			return "<none>"
		}
		var pairs []string
		for k, v := range labels {
			pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
		}
		return strings.Join(pairs, ",")
	default:
		// For other JSONPath expressions, try reflection-based extraction
		return extractValueByReflection(obj, column.JSONPath)
	}
}

// extractValueByReflection uses reflection to extract values (simplified implementation).
func extractValueByReflection(obj runtime.Object, jsonPath string) string {
	// This is a very simplified JSONPath implementation
	// A real implementation would use a proper JSONPath library

	objValue := reflect.ValueOf(obj)
	if objValue.Kind() == reflect.Ptr {
		objValue = objValue.Elem()
	}

	// Handle some common patterns
	switch jsonPath {
	case ".status.phase":
		if statusField := objValue.FieldByName("Status"); statusField.IsValid() {
			if phaseField := statusField.FieldByName("Phase"); phaseField.IsValid() {
				return fmt.Sprintf("%v", phaseField.Interface())
			}
		}
		return "Unknown"
	case ".status.restartCount":
		return "0" // Simplified
	case ".status.conditions[?(@.type==\"Ready\")].status":
		return "True" // Simplified
	case ".spec.nodeName":
		if specField := objValue.FieldByName("Spec"); specField.IsValid() {
			if nodeField := specField.FieldByName("NodeName"); nodeField.IsValid() {
				return fmt.Sprintf("%v", nodeField.Interface())
			}
		}
		return ""
	case ".status.podIP":
		if statusField := objValue.FieldByName("Status"); statusField.IsValid() {
			if ipField := statusField.FieldByName("PodIP"); ipField.IsValid() {
				return fmt.Sprintf("%v", ipField.Interface())
			}
		}
		return ""
	default:
		return unknownValue
	}
}
