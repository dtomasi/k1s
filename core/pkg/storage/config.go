package storage

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// StorageType defines the supported storage backend types.
type StorageType string

const (
	// StorageTypeMemory represents the in-memory storage backend
	StorageTypeMemory StorageType = "memory"

	// StorageTypePebble represents the PebbleDB storage backend
	StorageTypePebble StorageType = "pebble"

	// StorageTypeBolt represents the BoltDB storage backend (future)
	StorageTypeBolt StorageType = "bolt"

	// StorageTypeBadger represents the BadgerDB storage backend (future)
	StorageTypeBadger StorageType = "badger"
)

// FactoryConfig holds configuration for the storage factory.
type FactoryConfig struct {
	// Type specifies which storage backend to use
	Type StorageType `json:"type"`

	// Path is the directory path for persistent storage backends
	Path string `json:"path,omitempty"`

	// DatabaseName is the name of the database file (for file-based backends)
	DatabaseName string `json:"databaseName,omitempty"`

	// TenantPrefix enables multi-tenant storage with a prefix
	TenantPrefix string `json:"tenantPrefix,omitempty"`

	// Namespace provides default namespace for operations
	Namespace string `json:"namespace,omitempty"`

	// HealthCheck configuration
	HealthCheck HealthCheckConfig `json:"healthCheck,omitempty"`

	// Performance configuration
	Performance PerformanceConfig `json:"performance,omitempty"`

	// Security configuration
	Security SecurityConfig `json:"security,omitempty"`
}

// HealthCheckConfig configures health monitoring for storage backends.
type HealthCheckConfig struct {
	// Enabled enables health checking
	Enabled bool `json:"enabled"`

	// Interval is how often to perform health checks
	Interval time.Duration `json:"interval"`

	// Timeout is the maximum time to wait for a health check
	Timeout time.Duration `json:"timeout"`

	// MaxRetries is the maximum number of health check retries
	MaxRetries int `json:"maxRetries"`
}

// PerformanceConfig configures performance settings for storage backends.
type PerformanceConfig struct {
	// MaxConnections limits the number of concurrent connections
	MaxConnections int `json:"maxConnections"`

	// MaxIdleTime is the maximum idle time before closing connections
	MaxIdleTime time.Duration `json:"maxIdleTime"`

	// WriteTimeout is the timeout for write operations
	WriteTimeout time.Duration `json:"writeTimeout"`

	// ReadTimeout is the timeout for read operations
	ReadTimeout time.Duration `json:"readTimeout"`

	// CompactionInterval is how often to perform compaction (if supported)
	CompactionInterval time.Duration `json:"compactionInterval"`
}

// SecurityConfig configures security settings for storage backends.
type SecurityConfig struct {
	// EnableEncryption enables at-rest encryption
	EnableEncryption bool `json:"enableEncryption"`

	// EncryptionKey is the encryption key (base64 encoded)
	EncryptionKey string `json:"encryptionKey,omitempty"`

	// EnableTLS enables TLS for network communications (future)
	EnableTLS bool `json:"enableTLS"`

	// TLSCertFile is the path to the TLS certificate file
	TLSCertFile string `json:"tlsCertFile,omitempty"`

	// TLSKeyFile is the path to the TLS private key file
	TLSKeyFile string `json:"tlsKeyFile,omitempty"`
}

// BackendMetrics holds metrics information for storage backends.
type BackendMetrics struct {
	// Name is the backend name
	Name string `json:"name"`

	// Type is the backend type
	Type StorageType `json:"type"`

	// Status indicates the health status
	Status string `json:"status"`

	// OperationsCount tracks the number of operations
	OperationsCount map[string]int64 `json:"operationsCount"`

	// ErrorsCount tracks the number of errors
	ErrorsCount int64 `json:"errorsCount"`

	// LastHealthCheck is the timestamp of the last health check
	LastHealthCheck time.Time `json:"lastHealthCheck"`

	// StorageSize is the current storage size in bytes
	StorageSize int64 `json:"storageSize"`

	// ObjectCount is the current number of stored objects
	ObjectCount int64 `json:"objectCount"`
}

// Validate validates the factory configuration.
func (c *FactoryConfig) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("storage type must be specified")
	}

	switch c.Type {
	case StorageTypeMemory:
		// Memory storage doesn't need additional validation
		return nil
	case StorageTypePebble, StorageTypeBolt, StorageTypeBadger:
		if c.Path == "" {
			return fmt.Errorf("path is required for %s storage", c.Type)
		}
		return c.validatePersistentBackend()
	default:
		return fmt.Errorf("unsupported storage type: %s", c.Type)
	}
}

// validatePersistentBackend validates configuration for persistent storage backends.
func (c *FactoryConfig) validatePersistentBackend() error {
	// Validate path
	if !filepath.IsAbs(c.Path) && !strings.HasPrefix(c.Path, "./") {
		return fmt.Errorf("path must be absolute or relative: %s", c.Path)
	}

	// Set default database name if not provided
	if c.DatabaseName == "" {
		c.DatabaseName = "k1s.db"
	}

	// Validate health check configuration
	if c.HealthCheck.Enabled {
		if c.HealthCheck.Interval <= 0 {
			c.HealthCheck.Interval = 30 * time.Second // Default to 30 seconds
		}
		if c.HealthCheck.Timeout <= 0 {
			c.HealthCheck.Timeout = 5 * time.Second // Default to 5 seconds
		}
		if c.HealthCheck.MaxRetries <= 0 {
			c.HealthCheck.MaxRetries = 3 // Default to 3 retries
		}
	}

	// Validate performance configuration
	if c.Performance.MaxConnections <= 0 {
		c.Performance.MaxConnections = 10 // Default to 10 connections
	}
	if c.Performance.WriteTimeout <= 0 {
		c.Performance.WriteTimeout = 10 * time.Second
	}
	if c.Performance.ReadTimeout <= 0 {
		c.Performance.ReadTimeout = 5 * time.Second
	}

	return nil
}

// GetDatabasePath returns the full path to the database file.
func (c *FactoryConfig) GetDatabasePath() string {
	if c.Path == "" {
		return c.DatabaseName
	}
	return filepath.Join(c.Path, c.DatabaseName)
}

// ToStorageConfig converts FactoryConfig to the core storage Config.
func (c *FactoryConfig) ToStorageConfig() Config {
	return Config{
		TenantID:  c.TenantPrefix,
		Namespace: c.Namespace,
		KeyPrefix: c.TenantPrefix,
		// Transformer will be set by the factory based on security config
	}
}

// DefaultFactoryConfig returns a default factory configuration.
func DefaultFactoryConfig() *FactoryConfig {
	return &FactoryConfig{
		Type:         StorageTypeMemory,
		DatabaseName: "k1s.db",
		HealthCheck: HealthCheckConfig{
			Enabled:    true,
			Interval:   30 * time.Second,
			Timeout:    5 * time.Second,
			MaxRetries: 3,
		},
		Performance: PerformanceConfig{
			MaxConnections:     10,
			MaxIdleTime:        5 * time.Minute,
			WriteTimeout:       10 * time.Second,
			ReadTimeout:        5 * time.Second,
			CompactionInterval: 1 * time.Hour,
		},
		Security: SecurityConfig{
			EnableEncryption: false,
			EnableTLS:        false,
		},
	}
}

// MemoryFactoryConfig returns a factory configuration for memory storage.
func MemoryFactoryConfig() *FactoryConfig {
	config := DefaultFactoryConfig()
	config.Type = StorageTypeMemory
	return config
}

// PebbleFactoryConfig returns a factory configuration for Pebble storage.
func PebbleFactoryConfig(path string) *FactoryConfig {
	config := DefaultFactoryConfig()
	config.Type = StorageTypePebble
	config.Path = path
	return config
}
