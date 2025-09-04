// Package extractor provides kubebuilder marker extraction functionality
// for the cli-gen tool.
package extractor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ResourceInfo contains extracted resource information
type ResourceInfo struct {
	Name         string
	Kind         string
	Group        string
	Version      string
	Plural       string
	Singular     string
	ShortNames   []string
	Scope        string
	PrintColumns []PrintColumn
	Validations  map[string][]ValidationRule
	Defaults     map[string]string
	HasStatus    bool
}

// PrintColumn represents a kubebuilder printcolumn marker
type PrintColumn struct {
	Name        string
	Type        string
	JSONPath    string
	Description string
	Priority    int
}

// ValidationRule represents a kubebuilder validation marker
type ValidationRule struct {
	Type    string // "CEL", "Required", "Min", "Max", etc.
	Value   string // For simple validations like "Required" or range values
	Rule    string // CEL expression: "self >= 0"
	Message string // Validation error message
	Field   string // Field path this validation applies to
}

// Extractor handles kubebuilder marker extraction from Go source files
type Extractor struct {
	fileSet *token.FileSet
}

// NewExtractor creates a new marker extractor
func NewExtractor() *Extractor {
	return &Extractor{
		fileSet: token.NewFileSet(),
	}
}

// Extract processes Go source files and extracts kubebuilder markers
func (e *Extractor) Extract(paths []string) ([]*ResourceInfo, error) {
	resources := make([]*ResourceInfo, 0)
	var groupInfo *PackageInfo

	for _, path := range paths {
		// Check if path exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not exist: %s", path)
		}

		matches, err := filepath.Glob(filepath.Join(path, "*.go"))
		if err != nil {
			return nil, err
		}

		// First pass: extract package-level information from groupversion_info.go
		for _, file := range matches {
			if strings.Contains(file, "groupversion_info.go") {
				info, err := e.extractPackageInfo(file)
				if err != nil {
					return nil, err
				}
				if info != nil {
					groupInfo = info
				}
				break
			}
		}

		// Second pass: extract resource types
		for _, file := range matches {
			if strings.HasSuffix(file, "_test.go") || strings.HasSuffix(file, "zz_generated.go") {
				continue
			}

			resourceInfo, err := e.extractFromFile(file)
			if err != nil {
				return nil, err
			}

			if resourceInfo != nil {
				// Apply package-level group information
				if groupInfo != nil {
					resourceInfo.Group = groupInfo.Group
					if resourceInfo.Version == "" {
						resourceInfo.Version = groupInfo.Version
					}
				}
				resources = append(resources, resourceInfo)
			}
		}
	}

	return resources, nil
}

// PackageInfo contains package-level information
type PackageInfo struct {
	Group   string
	Version string
}

// extractPackageInfo extracts package-level information from groupversion_info.go
func (e *Extractor) extractPackageInfo(filename string) (*PackageInfo, error) {
	src, err := parser.ParseFile(e.fileSet, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	info := &PackageInfo{}

	// Check all comments for groupName marker
	for _, commentGroup := range src.Comments {
		for _, comment := range commentGroup.List {
			text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
			if strings.HasPrefix(text, "+groupName=") {
				info.Group = strings.TrimPrefix(text, "+groupName=")
			}
		}
	}

	// Try to extract version from package name
	packageName := src.Name.Name
	switch {
	case strings.Contains(packageName, "v1alpha"):
		info.Version = "v1alpha1"
	case strings.Contains(packageName, "v1beta"):
		info.Version = "v1beta1"
	case strings.Contains(packageName, "v1"):
		info.Version = "v1"
	}

	return info, nil
}

// extractFromFile extracts markers from a single Go file
func (e *Extractor) extractFromFile(filename string) (*ResourceInfo, error) {
	src, err := parser.ParseFile(e.fileSet, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var resource *ResourceInfo

	// Find type declarations and their preceding comments
	ast.Inspect(src, func(n ast.Node) bool {
		if genDecl, ok := n.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
			// Check if this is a type declaration
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if isResourceType(typeSpec) {
						resource = e.extractResourceInfo(typeSpec, src, genDecl)
					}
				}
			}
		}
		return true
	})

	return resource, nil
}

// isResourceType checks if a type is a Kubernetes resource
func isResourceType(typeSpec *ast.TypeSpec) bool {
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return false
	}

	// Check if it embeds TypeMeta and ObjectMeta
	hasTypeMeta := false
	hasObjectMeta := false

	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 { // Embedded field
			if sel, ok := field.Type.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "TypeMeta" {
					hasTypeMeta = true
				}
				if sel.Sel.Name == "ObjectMeta" {
					hasObjectMeta = true
				}
			}
		}
	}

	return hasTypeMeta && hasObjectMeta
}

// extractResourceInfo extracts resource information from a type declaration
func (e *Extractor) extractResourceInfo(typeSpec *ast.TypeSpec, file *ast.File, genDecl *ast.GenDecl) *ResourceInfo {
	resource := &ResourceInfo{
		Kind:        typeSpec.Name.Name,
		Name:        strings.ToLower(typeSpec.Name.Name),
		Validations: make(map[string][]ValidationRule),
		Defaults:    make(map[string]string),
	}

	// Extract package-level markers (group/version)
	if file.Doc != nil {
		e.extractPackageMarkers(resource, file.Doc)
	}

	// Also check all comments in the file for package-level markers
	for _, commentGroup := range file.Comments {
		e.extractPackageMarkers(resource, commentGroup)
	}

	// Extract version from package name if not found in markers
	if resource.Version == "" {
		packageName := file.Name.Name
		switch {
		case strings.Contains(packageName, "v1alpha"):
			resource.Version = "v1alpha1"
		case strings.Contains(packageName, "v1beta"):
			resource.Version = "v1beta1"
		case strings.Contains(packageName, "v1"):
			resource.Version = "v1"
		}
	}

	// Extract type-level markers from GenDecl (these are the kubebuilder markers above type)
	foundMarkers := false
	if genDecl.Doc != nil {
		e.extractTypeMarkers(resource, genDecl.Doc)
		foundMarkers = true
	}

	// Also check typeSpec.Doc for any additional comments
	if typeSpec.Doc != nil {
		e.extractTypeMarkers(resource, typeSpec.Doc)
		foundMarkers = true
	}

	// If no markers were found through normal AST association, try to find them
	// by searching all comments in the file
	if !foundMarkers || (len(resource.ShortNames) == 0 && len(resource.PrintColumns) == 0 && !resource.HasStatus && resource.Scope == "") {
		e.extractTypeMarkersFromFileComments(resource, file, typeSpec)
	}

	// Extract field-level markers for validation from all structs in the file
	e.extractAllFieldMarkers(resource, file)

	return resource
}

// extractPackageMarkers extracts group and version information
func (e *Extractor) extractPackageMarkers(resource *ResourceInfo, docGroup *ast.CommentGroup) {
	for _, comment := range docGroup.List {
		text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		if strings.HasPrefix(text, "+groupName=") {
			resource.Group = strings.TrimPrefix(text, "+groupName=")
		}
		// Note: other kubebuilder markers could be handled here in the future
	}
}

// extractTypeMarkers extracts type-level kubebuilder markers
func (e *Extractor) extractTypeMarkers(resource *ResourceInfo, docGroup *ast.CommentGroup) {
	for _, comment := range docGroup.List {
		text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))

		// Process kubebuilder markers

		switch {
		case strings.HasPrefix(text, "+kubebuilder:resource:"):
			e.parseResourceMarker(resource, text)
		case strings.HasPrefix(text, "+kubebuilder:printcolumn:"):
			column := e.parsePrintColumnMarker(text)
			if column != nil {
				resource.PrintColumns = append(resource.PrintColumns, *column)
			}
		case text == "+kubebuilder:subresource:status":
			resource.HasStatus = true
		}
	}
}

// extractAllFieldMarkers extracts field markers from all structs in the file
func (e *Extractor) extractAllFieldMarkers(resource *ResourceInfo, file *ast.File) {
	// Walk through all declarations in the file to find structs
	ast.Inspect(file, func(n ast.Node) bool {
		if genDecl, ok := n.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if structType, ok := typeSpec.Type.(*ast.StructType); ok {
						e.extractFieldMarkers(resource, structType, file)
					}
				}
			}
		}
		return true
	})
}

// extractFieldMarkers extracts field-level validation and default markers
func (e *Extractor) extractFieldMarkers(resource *ResourceInfo, structType *ast.StructType, file *ast.File) {
	for _, field := range structType.Fields.List {
		fieldName := ""
		if len(field.Names) > 0 {
			fieldName = field.Names[0].Name
		}

		// Skip embedded fields (no names)
		if fieldName == "" {
			continue
		}

		// First try field.Doc (direct association)
		if field.Doc != nil {
			e.extractFieldMarkersFromComments(resource, fieldName, field.Doc)
		}

		// If no markers found, search for markers in comments preceding this field
		if len(resource.Validations[fieldName]) == 0 && resource.Defaults[fieldName] == "" {
			e.extractFieldMarkersFromFileComments(resource, fieldName, field, file)
		}
	}
}

// extractFieldMarkersFromComments extracts field markers from a comment group
func (e *Extractor) extractFieldMarkersFromComments(resource *ResourceInfo, fieldName string, docGroup *ast.CommentGroup) {
	for _, comment := range docGroup.List {
		text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))

		switch {
		case strings.HasPrefix(text, "+kubebuilder:validation:"):
			rule := e.parseValidationMarker(text)
			if rule != nil {
				// Set the field path for the validation rule
				rule.Field = fieldName
				resource.Validations[fieldName] = append(resource.Validations[fieldName], *rule)
			}
		case strings.HasPrefix(text, "+kubebuilder:pruning:"):
			// Handle pruning markers like PreserveUnknownFields as validation rules
			rule := e.parsePruningMarker(text)
			if rule != nil {
				resource.Validations[fieldName] = append(resource.Validations[fieldName], *rule)
			}
		case strings.HasPrefix(text, "+kubebuilder:default="):
			defaultValue := strings.TrimPrefix(text, "+kubebuilder:default=")
			// Remove quotes from default values if present
			if strings.HasPrefix(defaultValue, "\"") && strings.HasSuffix(defaultValue, "\"") {
				defaultValue = strings.Trim(defaultValue, "\"")
			}
			resource.Defaults[fieldName] = defaultValue
		}
	}
}

// extractFieldMarkersFromFileComments searches file comments for field markers
func (e *Extractor) extractFieldMarkersFromFileComments(resource *ResourceInfo, fieldName string, field *ast.Field, file *ast.File) {
	fieldPos := e.fileSet.Position(field.Pos())

	// Look for comments immediately before this field
	for _, commentGroup := range file.Comments {
		commentPos := e.fileSet.Position(commentGroup.Pos())
		commentEndPos := e.fileSet.Position(commentGroup.End())

		// Consider comments that are within 3 lines before the field
		lineDiff := fieldPos.Line - commentEndPos.Line
		if commentPos.Filename == fieldPos.Filename &&
			lineDiff >= 0 && lineDiff <= 3 {

			// Check if this comment group has field-level markers
			for _, comment := range commentGroup.List {
				text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
				if strings.HasPrefix(text, "+kubebuilder:validation:") ||
					strings.HasPrefix(text, "+kubebuilder:pruning:") ||
					strings.HasPrefix(text, "+kubebuilder:default=") {
					e.extractFieldMarkersFromComments(resource, fieldName, commentGroup)
					return
				}
			}
		}
	}
}

// parseResourceMarker parses +kubebuilder:resource: markers
func (e *Extractor) parseResourceMarker(resource *ResourceInfo, marker string) {
	content := strings.TrimPrefix(marker, "+kubebuilder:resource:")

	// Parse key=value pairs
	pairs := strings.Split(content, ",")
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])

			switch key {
			case "scope":
				resource.Scope = value
			case "shortName":
				// Handle multiple short names separated by semicolon
				if strings.Contains(value, ";") {
					resource.ShortNames = strings.Split(value, ";")
				} else {
					resource.ShortNames = []string{value}
				}
			case "plural":
				resource.Plural = value
			case "singular":
				resource.Singular = value
			}
		}
	}
}

// parsePrintColumnMarker parses +kubebuilder:printcolumn: markers
func (e *Extractor) parsePrintColumnMarker(marker string) *PrintColumn {
	content := strings.TrimPrefix(marker, "+kubebuilder:printcolumn:")

	column := &PrintColumn{}
	pairs := strings.Split(content, ",")

	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.Trim(strings.TrimSpace(kv[1]), "`\"")

			switch key {
			case "name":
				column.Name = value
			case "type":
				column.Type = value
			case "JSONPath":
				column.JSONPath = value
			case "description":
				column.Description = value
			case "priority":
				// Parse priority as integer
				if priority, err := strconv.Atoi(value); err == nil {
					column.Priority = priority
				}
			}
		}
	}

	if column.Name != "" && column.Type != "" && column.JSONPath != "" {
		return column
	}
	return nil
}

// parseValidationMarker parses +kubebuilder:validation: markers
func (e *Extractor) parseValidationMarker(marker string) *ValidationRule {
	content := strings.TrimPrefix(marker, "+kubebuilder:validation:")

	// Handle different validation types
	// Allow for validation rules without values (like "Required")
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}

	// Special handling for CEL validation markers
	// Format: CEL:rule="expression",message="error message"
	if strings.HasPrefix(content, "CEL:") {
		return e.parseCELValidationMarker(content)
	}

	validationRegex := regexp.MustCompile(`^([A-Za-z0-9]+)(?:=(.+))?$`)
	matches := validationRegex.FindStringSubmatch(content)

	if len(matches) >= 2 {
		ruleType := matches[1]

		// List of known kubebuilder validation types - ignore unknown ones
		knownRules := map[string]bool{
			"Required": true, "Optional": true, "MinLength": true, "MaxLength": true,
			"MinItems": true, "MaxItems": true, "UniqueItems": true, "Minimum": true,
			"Maximum": true, "ExclusiveMinimum": true, "ExclusiveMaximum": true,
			"Pattern": true, "Enum": true, "Format": true, "Type": true,
			"PreserveUnknownFields": true, "EmbeddedResource": true,
		}

		if !knownRules[ruleType] {
			// Skip unknown validation rules
			return nil
		}
		rule := &ValidationRule{
			Type: matches[1],
		}
		if len(matches) > 2 {
			// Remove quotes from the value if present
			value := matches[2]
			if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
				value = strings.Trim(value, "\"")
			}
			rule.Value = value
		}
		return rule
	}

	return nil
}

// parseCELValidationMarker parses CEL validation markers
// Format: CEL:rule="expression",message="error message"
func (e *Extractor) parseCELValidationMarker(content string) *ValidationRule {
	// Remove the "CEL:" prefix
	celContent := strings.TrimPrefix(content, "CEL:")

	rule := &ValidationRule{
		Type: "CEL",
	}

	// Parse rule= and message= parameters
	// Use a simple approach to handle quoted values with commas
	parts := e.parseCELParameters(celContent)

	for key, value := range parts {
		switch key {
		case "rule":
			rule.Rule = value
		case "message":
			rule.Message = value
		}
	}

	// Rule is required for CEL validation
	if rule.Rule == "" {
		return nil
	}

	return rule
}

// parseCELParameters parses key="value" pairs from CEL marker content
func (e *Extractor) parseCELParameters(content string) map[string]string {
	params := make(map[string]string)

	// Simple parser for key="value",key2="value2" format
	// This handles quoted values with commas inside them
	paramRegex := regexp.MustCompile(`(\w+)="([^"]*)"`)
	matches := paramRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			key := match[1]
			value := match[2]
			params[key] = value
		}
	}

	return params
}

// parsePruningMarker parses +kubebuilder:pruning: markers
func (e *Extractor) parsePruningMarker(marker string) *ValidationRule {
	content := strings.TrimPrefix(marker, "+kubebuilder:pruning:")

	// For pruning markers, the type is the content itself (like "PreserveUnknownFields")
	if content != "" {
		return &ValidationRule{
			Type:  content,
			Value: "",
		}
	}

	return nil
}

// extractTypeMarkersFromFileComments searches all file comments for markers that might be associated with this type
// This is needed because Go AST sometimes doesn't properly associate comments with declarations
func (e *Extractor) extractTypeMarkersFromFileComments(resource *ResourceInfo, file *ast.File, typeSpec *ast.TypeSpec) {
	// Get the position of the type declaration
	typePos := e.fileSet.Position(typeSpec.Pos())

	// Debug: print type position and name
	// fmt.Printf("DEBUG: Looking for markers for type %s at line %d\n", typeSpec.Name.Name, typePos.Line)

	// Look through all comment groups in the file
	for _, commentGroup := range file.Comments {
		// Get the position of this comment group
		commentPos := e.fileSet.Position(commentGroup.Pos())
		commentEndPos := e.fileSet.Position(commentGroup.End())

		// Debug: print comment position and check for markers
		/*
			hasMarkers := false
			for _, comment := range commentGroup.List {
				text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
				if strings.HasPrefix(text, "+kubebuilder:") {
					hasMarkers = true
					break
				}
			}
			if hasMarkers {
				fmt.Printf("DEBUG: Found kubebuilder comment group at lines %d-%d\n", commentPos.Line, commentEndPos.Line)
			}
		*/

		// Consider comments that are immediately before the type declaration
		// Allow up to 10 lines between comment end and type start (for separated markers)
		lineDiff := typePos.Line - commentEndPos.Line
		if commentPos.Filename == typePos.Filename &&
			lineDiff >= 0 && lineDiff <= 10 {

			// Check if any comment in this group contains kubebuilder markers
			for _, comment := range commentGroup.List {
				text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
				if strings.HasPrefix(text, "+kubebuilder:") {
					// Found kubebuilder markers near our type, extract them
					e.extractTypeMarkers(resource, commentGroup)
					break
				}
			}
		}
	}
}
