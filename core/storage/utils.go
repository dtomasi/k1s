package storage

// BuildKey constructs a storage key with optional prefix components
func BuildKey(components ...string) string {
	if len(components) == 0 {
		return ""
	}

	// Filter out empty components
	var validComponents []string
	for _, component := range components {
		if component != "" {
			validComponents = append(validComponents, component)
		}
	}

	if len(validComponents) == 0 {
		return ""
	}

	// Join components with forward slash
	result := validComponents[0]
	for i := 1; i < len(validComponents); i++ {
		result += "/" + validComponents[i]
	}

	return result
}

// Note: Storage type utilities removed as part of factory removal.
// Users now create storage instances directly from backend modules.
