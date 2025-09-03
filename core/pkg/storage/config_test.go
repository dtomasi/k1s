package storage_test

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/dtomasi/k1s/core/pkg/storage"
)

var _ = Describe("Storage Configuration", func() {

	Describe("FactoryConfig Validation", func() {
		It("should reject empty storage type", func() {
			config := &storage.FactoryConfig{}

			err := config.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("storage type must be specified"))
		})

		It("should accept memory storage without additional configuration", func() {
			config := &storage.FactoryConfig{
				Type: storage.StorageTypeMemory,
			}

			err := config.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should require path for persistent storage backends", func() {
			backends := []storage.StorageType{
				storage.StorageTypePebble,
				storage.StorageTypeBolt,
				storage.StorageTypeBadger,
			}

			for _, backend := range backends {
				config := &storage.FactoryConfig{
					Type: backend,
				}

				err := config.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("path is required"))
			}
		})

		It("should reject unsupported storage types", func() {
			config := &storage.FactoryConfig{
				Type: "unsupported",
			}

			err := config.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported storage type"))
		})

		It("should accept persistent backends with valid path", func() {
			tempDir, err := os.MkdirTemp("", "k1s-config-test-*")
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				err := os.RemoveAll(tempDir)
				Expect(err).NotTo(HaveOccurred())
			}()

			config := &storage.FactoryConfig{
				Type: storage.StorageTypeBolt,
				Path: tempDir,
			}

			err = config.Validate()
			Expect(err).NotTo(HaveOccurred())
			Expect(config.DatabaseName).To(Equal("k1s.db"))
		})
	})

	Describe("Persistent Backend Validation", func() {
		var tempDir string

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "k1s-config-test-*")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			if tempDir != "" {
				err := os.RemoveAll(tempDir)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should accept absolute paths", func() {
			config := &storage.FactoryConfig{
				Type: storage.StorageTypeBolt,
				Path: tempDir,
			}

			err := config.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should accept relative paths", func() {
			config := &storage.FactoryConfig{
				Type: storage.StorageTypeBolt,
				Path: "./relative/path",
			}

			err := config.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject invalid paths", func() {
			config := &storage.FactoryConfig{
				Type: storage.StorageTypeBolt,
				Path: "invalid-path",
			}

			err := config.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("path must be absolute or relative"))
		})

		It("should set default database name when not provided", func() {
			config := &storage.FactoryConfig{
				Type: storage.StorageTypeBolt,
				Path: tempDir,
			}

			err := config.Validate()
			Expect(err).NotTo(HaveOccurred())
			Expect(config.DatabaseName).To(Equal("k1s.db"))
		})

		It("should preserve provided database name", func() {
			config := &storage.FactoryConfig{
				Type:         storage.StorageTypeBolt,
				Path:         tempDir,
				DatabaseName: "custom.db",
			}

			err := config.Validate()
			Expect(err).NotTo(HaveOccurred())
			Expect(config.DatabaseName).To(Equal("custom.db"))
		})
	})

	Describe("Health Check Configuration", func() {
		var config *storage.FactoryConfig
		var tempDir string

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "k1s-config-test-*")
			Expect(err).NotTo(HaveOccurred())

			config = &storage.FactoryConfig{
				Type: storage.StorageTypeBolt,
				Path: tempDir,
				HealthCheck: storage.HealthCheckConfig{
					Enabled: true,
				},
			}
		})

		AfterEach(func() {
			if tempDir != "" {
				err := os.RemoveAll(tempDir)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should set default health check values when enabled", func() {
			err := config.Validate()
			Expect(err).NotTo(HaveOccurred())

			Expect(config.HealthCheck.Interval).To(Equal(30 * time.Second))
			Expect(config.HealthCheck.Timeout).To(Equal(5 * time.Second))
			Expect(config.HealthCheck.MaxRetries).To(Equal(3))
		})

		It("should preserve provided health check values", func() {
			config.HealthCheck.Interval = 60 * time.Second
			config.HealthCheck.Timeout = 10 * time.Second
			config.HealthCheck.MaxRetries = 5

			err := config.Validate()
			Expect(err).NotTo(HaveOccurred())

			Expect(config.HealthCheck.Interval).To(Equal(60 * time.Second))
			Expect(config.HealthCheck.Timeout).To(Equal(10 * time.Second))
			Expect(config.HealthCheck.MaxRetries).To(Equal(5))
		})
	})

	Describe("Performance Configuration", func() {
		var config *storage.FactoryConfig
		var tempDir string

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "k1s-config-test-*")
			Expect(err).NotTo(HaveOccurred())

			config = &storage.FactoryConfig{
				Type: storage.StorageTypeBolt,
				Path: tempDir,
			}
		})

		AfterEach(func() {
			if tempDir != "" {
				err := os.RemoveAll(tempDir)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should set default performance values", func() {
			err := config.Validate()
			Expect(err).NotTo(HaveOccurred())

			Expect(config.Performance.MaxConnections).To(Equal(10))
			Expect(config.Performance.WriteTimeout).To(Equal(10 * time.Second))
			Expect(config.Performance.ReadTimeout).To(Equal(5 * time.Second))
		})

		It("should preserve provided performance values", func() {
			config.Performance.MaxConnections = 20
			config.Performance.WriteTimeout = 15 * time.Second
			config.Performance.ReadTimeout = 8 * time.Second

			err := config.Validate()
			Expect(err).NotTo(HaveOccurred())

			Expect(config.Performance.MaxConnections).To(Equal(20))
			Expect(config.Performance.WriteTimeout).To(Equal(15 * time.Second))
			Expect(config.Performance.ReadTimeout).To(Equal(8 * time.Second))
		})
	})

	Describe("Database Path Generation", func() {
		It("should return database name when path is empty", func() {
			config := &storage.FactoryConfig{
				DatabaseName: "test.db",
			}

			path := config.GetDatabasePath()
			Expect(path).To(Equal("test.db"))
		})

		It("should join path and database name", func() {
			config := &storage.FactoryConfig{
				Path:         "/tmp/storage",
				DatabaseName: "test.db",
			}

			path := config.GetDatabasePath()
			Expect(path).To(Equal(filepath.Join("/tmp/storage", "test.db")))
		})
	})

	Describe("Storage Config Conversion", func() {
		It("should convert factory config to storage config", func() {
			factoryConfig := &storage.FactoryConfig{
				TenantPrefix: "test-tenant",
				Namespace:    "test-namespace",
			}

			storageConfig := factoryConfig.ToStorageConfig()
			Expect(storageConfig.TenantID).To(Equal("test-tenant"))
			Expect(storageConfig.Namespace).To(Equal("test-namespace"))
			Expect(storageConfig.KeyPrefix).To(Equal("test-tenant"))
		})
	})

	Describe("Default Configurations", func() {
		It("should create default factory config", func() {
			config := storage.DefaultFactoryConfig()

			Expect(config.Type).To(Equal(storage.StorageTypeMemory))
			Expect(config.DatabaseName).To(Equal("k1s.db"))
			Expect(config.HealthCheck.Enabled).To(BeTrue())
			Expect(config.HealthCheck.Interval).To(Equal(30 * time.Second))
			Expect(config.Performance.MaxConnections).To(Equal(10))
			Expect(config.Security.EnableEncryption).To(BeFalse())
		})

		It("should create memory factory config", func() {
			config := storage.MemoryFactoryConfig()

			Expect(config.Type).To(Equal(storage.StorageTypeMemory))
			Expect(config.DatabaseName).To(Equal("k1s.db"))
		})

		It("should create pebble factory config", func() {
			path := "/tmp/test"
			config := storage.PebbleFactoryConfig(path)

			Expect(config.Type).To(Equal(storage.StorageTypePebble))
			Expect(config.Path).To(Equal(path))
		})
	})

	Describe("Backend Metrics", func() {
		It("should create backend metrics structure", func() {
			metrics := &storage.BackendMetrics{
				Name:   "test-backend",
				Type:   storage.StorageTypeMemory,
				Status: "healthy",
				OperationsCount: map[string]int64{
					"get":    100,
					"create": 50,
					"delete": 25,
				},
				ErrorsCount:     5,
				LastHealthCheck: time.Now(),
				StorageSize:     1024,
				ObjectCount:     75,
			}

			Expect(metrics.Name).To(Equal("test-backend"))
			Expect(metrics.Type).To(Equal(storage.StorageTypeMemory))
			Expect(metrics.Status).To(Equal("healthy"))
			Expect(metrics.OperationsCount["get"]).To(Equal(int64(100)))
			Expect(metrics.ErrorsCount).To(Equal(int64(5)))
			Expect(metrics.ObjectCount).To(Equal(int64(75)))
		})
	})
})
