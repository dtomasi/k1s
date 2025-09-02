package storage

import (
	"fmt"
	"path"
	"strconv"
	"strings"
)

const (
	// DefaultKeyPrefix is the default prefix for all k1s storage keys
	DefaultKeyPrefix = "k1s"

	// TenantKeySeparator separates tenant prefixes from resource keys
	TenantKeySeparator = ":"

	// ResourceKeySeparator separates resource type from resource name
	ResourceKeySeparator = "/"
)

// GenerateKey creates a storage key for a resource with optional tenant isolation
func GenerateKey(opts KeyOptions) string {
	parts := make([]string, 0, 5)

	// Add base prefix
	if opts.Tenant != nil && opts.Tenant.Prefix != "" {
		parts = append(parts, opts.Tenant.Prefix)
	} else {
		parts = append(parts, DefaultKeyPrefix)
	}

	// Add tenant ID if provided
	if opts.Tenant != nil && opts.Tenant.ID != "" {
		parts = append(parts, opts.Tenant.ID)
	}

	// Add namespace
	namespace := opts.Namespace
	if namespace == "" && opts.Tenant != nil {
		namespace = opts.Tenant.Namespace
	}
	if namespace == "" {
		namespace = "default"
	}
	parts = append(parts, namespace)

	// Add resource type
	if opts.Resource != "" {
		parts = append(parts, opts.Resource)
	}

	// Add resource name if provided
	if opts.Name != "" {
		parts = append(parts, opts.Name)
	}

	return path.Join(parts...)
}

// GenerateListKey creates a storage key for listing resources
func GenerateListKey(opts KeyOptions) string {
	// For list operations, we don't include the name
	listOpts := opts
	listOpts.Name = ""
	return GenerateKey(listOpts)
}

// ParseKey extracts components from a storage key
func ParseKey(key string) (tenant, namespace, resource, name string, err error) {
	// Clean the key and split by path separator
	cleanKey := strings.Trim(key, "/")
	parts := strings.Split(cleanKey, "/")

	if len(parts) < 2 {
		return "", "", "", "", fmt.Errorf("invalid key format: %s", key)
	}

	// Skip the base prefix
	startIdx := 0
	if parts[0] == DefaultKeyPrefix || strings.Contains(parts[0], TenantKeySeparator) {
		startIdx = 1
	}

	// Check if we have a tenant ID (more than 3 parts total)
	// Format without tenant: k1s/namespace/resource/name (3-4 parts)
	// Format with tenant: k1s/tenant/namespace/resource/name (4-5 parts)
	if len(parts) >= 5 {
		// Format: prefix/tenant/namespace/resource/name
		tenant = parts[startIdx]
		startIdx++
	}

	// Extract remaining components
	remaining := parts[startIdx:]
	switch len(remaining) {
	case 1:
		// Only namespace provided
		namespace = remaining[0]
	case 2:
		// namespace/resource
		namespace = remaining[0]
		resource = remaining[1]
	case 3:
		// namespace/resource/name
		namespace = remaining[0]
		resource = remaining[1]
		name = remaining[2]
	default:
		return "", "", "", "", fmt.Errorf("invalid key format: %s", key)
	}

	return tenant, namespace, resource, name, nil
}

// ValidateKey checks if a key is valid for the given tenant
func ValidateKey(key string, tenantID string) error {
	tenant, _, _, _, err := ParseKey(key)
	if err != nil {
		return fmt.Errorf("invalid key: %w", err)
	}

	// If we're in multi-tenant mode, validate tenant isolation
	if tenantID != "" && tenant != tenantID {
		return fmt.Errorf("key %s does not belong to tenant %s", key, tenantID)
	}

	return nil
}

// BuildTenantPrefix creates a tenant-specific key prefix
func BuildTenantPrefix(config TenantConfig) string {
	if config.Prefix != "" {
		return config.Prefix
	}
	return fmt.Sprintf("%s%s%s", DefaultKeyPrefix, TenantKeySeparator, config.ID)
}

// IsTenantKey checks if a key belongs to a specific tenant
func IsTenantKey(key string, tenantID string) bool {
	if tenantID == "" {
		return true // No tenant restriction
	}

	tenant, _, _, _, err := ParseKey(key)
	if err != nil {
		return false
	}

	return tenant == tenantID
}

// EncodeResourceVersion converts a numeric resource version to string
func EncodeResourceVersion(version uint64) string {
	if version == 0 {
		return ""
	}
	return strconv.FormatUint(version, 10)
}

// ParseResourceVersion converts a string resource version to numeric
func ParseResourceVersion(version string) (uint64, error) {
	if version == "" {
		return 0, nil
	}
	return strconv.ParseUint(version, 10, 64)
}

// CreateKeyGenerator returns a function that generates keys for a specific configuration
func CreateKeyGenerator(config Config) func(resource, namespace, name string) string {
	var tenantConfig *TenantConfig
	if config.TenantID != "" {
		tenantConfig = &TenantConfig{
			ID:        config.TenantID,
			Prefix:    config.KeyPrefix,
			Namespace: config.Namespace,
		}
	}

	return func(resource, namespace, name string) string {
		opts := KeyOptions{
			Tenant:    tenantConfig,
			Namespace: namespace,
			Resource:  resource,
			Name:      name,
		}
		return GenerateKey(opts)
	}
}

// CreateListKeyGenerator returns a function that generates list keys for a specific configuration
func CreateListKeyGenerator(config Config) func(resource, namespace string) string {
	var tenantConfig *TenantConfig
	if config.TenantID != "" {
		tenantConfig = &TenantConfig{
			ID:        config.TenantID,
			Prefix:    config.KeyPrefix,
			Namespace: config.Namespace,
		}
	}

	return func(resource, namespace string) string {
		opts := KeyOptions{
			Tenant:    tenantConfig,
			Namespace: namespace,
			Resource:  resource,
		}
		return GenerateListKey(opts)
	}
}
