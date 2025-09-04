// Package extractor provides kubebuilder marker extraction functionality
// for the cli-gen tool.
package extractor

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
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
	Type  string
	Value string
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
	var resources []*ResourceInfo
	var groupInfo *PackageInfo

	for _, path := range paths {
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
	if genDecl.Doc != nil {
		e.extractTypeMarkers(resource, genDecl.Doc)
	}

	// Also check typeSpec.Doc for any additional comments
	if typeSpec.Doc != nil {
		e.extractTypeMarkers(resource, typeSpec.Doc)
	}

	// Extract field-level markers for validation
	if structType, ok := typeSpec.Type.(*ast.StructType); ok {
		e.extractFieldMarkers(resource, structType)
	}

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

// extractFieldMarkers extracts field-level validation and default markers
func (e *Extractor) extractFieldMarkers(resource *ResourceInfo, structType *ast.StructType) {
	for _, field := range structType.Fields.List {
		if field.Doc != nil {
			fieldName := ""
			if len(field.Names) > 0 {
				fieldName = field.Names[0].Name
			}

			for _, comment := range field.Doc.List {
				text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))

				if strings.HasPrefix(text, "+kubebuilder:validation:") {
					rule := e.parseValidationMarker(text)
					if rule != nil && fieldName != "" {
						resource.Validations[fieldName] = append(resource.Validations[fieldName], *rule)
					}
				} else if strings.HasPrefix(text, "+kubebuilder:default=") {
					defaultValue := strings.TrimPrefix(text, "+kubebuilder:default=")
					if fieldName != "" {
						resource.Defaults[fieldName] = defaultValue
					}
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
	validationRegex := regexp.MustCompile(`^([A-Za-z]+)(?:=(.+))?$`)
	matches := validationRegex.FindStringSubmatch(content)

	if len(matches) >= 2 {
		rule := &ValidationRule{
			Type: matches[1],
		}
		if len(matches) > 2 {
			rule.Value = matches[2]
		}
		return rule
	}

	return nil
}
