package storage

import (
	"fmt"
)

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

// IsValidStorageType checks if a storage type is supported
func IsValidStorageType(storageType StorageType) bool {
	switch storageType {
	case StorageTypeMemory, StorageTypePebble, StorageTypeBolt, StorageTypeBadger:
		return true
	default:
		return false
	}
}

// StorageTypeFromString converts a string to StorageType with validation
func StorageTypeFromString(s string) (StorageType, error) {
	storageType := StorageType(s)
	if !IsValidStorageType(storageType) {
		return "", fmt.Errorf("invalid storage type: %s", s)
	}
	return storageType, nil
}

// GetAllStorageTypes returns all supported storage types
func GetAllStorageTypes() []StorageType {
	return []StorageType{
		StorageTypeMemory,
		StorageTypePebble,
		StorageTypeBolt,
		StorageTypeBadger,
	}
}

// IsMemoryBackend checks if the storage type is memory-based
func IsMemoryBackend(storageType StorageType) bool {
	return storageType == StorageTypeMemory
}

// IsPersistentBackend checks if the storage type is persistent (file-based)
func IsPersistentBackend(storageType StorageType) bool {
	return !IsMemoryBackend(storageType)
}

// GetDefaultDatabaseName returns the default database filename for a storage type
func GetDefaultDatabaseName(storageType StorageType) string {
	switch storageType {
	case StorageTypeBolt:
		return "k1s.bolt"
	case StorageTypePebble:
		return "k1s.pebble"
	case StorageTypeBadger:
		return "k1s.badger"
	default:
		return "k1s.db"
	}
}

// GetDefaultFileExtension returns the default file extension for a storage type
func GetDefaultFileExtension(storageType StorageType) string {
	switch storageType {
	case StorageTypeBolt:
		return ".bolt"
	case StorageTypePebble:
		return ".pebble"
	case StorageTypeBadger:
		return ".badger"
	default:
		return ".db"
	}
}

// StorageTypeRequiresPath checks if a storage type requires a filesystem path
func StorageTypeRequiresPath(storageType StorageType) bool {
	return IsPersistentBackend(storageType)
}
