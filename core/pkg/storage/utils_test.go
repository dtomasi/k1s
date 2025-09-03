package storage_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/dtomasi/k1s/core/pkg/storage"
)

var _ = Describe("Storage Utils", func() {

	Describe("Resource Version Encoding/Decoding", func() {
		It("should encode zero resource version as empty string", func() {
			encoded := storage.EncodeResourceVersion(0)
			Expect(encoded).To(Equal(""))
		})

		It("should encode non-zero resource version as string", func() {
			encoded := storage.EncodeResourceVersion(12345)
			Expect(encoded).To(Equal("12345"))
		})

		It("should parse empty resource version as zero", func() {
			version, err := storage.ParseResourceVersion("")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal(uint64(0)))
		})

		It("should parse valid resource version string", func() {
			version, err := storage.ParseResourceVersion("12345")
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal(uint64(12345)))
		})

		It("should reject invalid resource version format", func() {
			version, err := storage.ParseResourceVersion("invalid")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid syntax"))
			Expect(version).To(Equal(uint64(0)))
		})

		It("should handle round-trip encoding/decoding", func() {
			original := uint64(98765)
			encoded := storage.EncodeResourceVersion(original)
			decoded, err := storage.ParseResourceVersion(encoded)
			Expect(err).NotTo(HaveOccurred())
			Expect(decoded).To(Equal(original))
		})
	})

	Describe("Key Building", func() {
		It("should handle empty components", func() {
			key := storage.BuildKey()
			Expect(key).To(Equal(""))
		})

		It("should build key from single component", func() {
			key := storage.BuildKey("component1")
			Expect(key).To(Equal("component1"))
		})

		It("should build key from multiple components", func() {
			key := storage.BuildKey("tenant", "namespace", "resource", "name")
			Expect(key).To(Equal("tenant/namespace/resource/name"))
		})

		It("should filter out empty components", func() {
			key := storage.BuildKey("tenant", "", "resource", "name", "")
			Expect(key).To(Equal("tenant/resource/name"))
		})

		It("should handle all empty components", func() {
			key := storage.BuildKey("", "", "")
			Expect(key).To(Equal(""))
		})
	})

	Describe("Storage Type Validation", func() {
		It("should validate supported storage types", func() {
			Expect(storage.IsValidStorageType(storage.StorageTypeMemory)).To(BeTrue())
			Expect(storage.IsValidStorageType(storage.StorageTypePebble)).To(BeTrue())
			Expect(storage.IsValidStorageType(storage.StorageTypeBolt)).To(BeTrue())
			Expect(storage.IsValidStorageType(storage.StorageTypeBadger)).To(BeTrue())
		})

		It("should reject invalid storage types", func() {
			Expect(storage.IsValidStorageType("invalid")).To(BeFalse())
			Expect(storage.IsValidStorageType("")).To(BeFalse())
		})

		It("should convert valid string to storage type", func() {
			storageType, err := storage.StorageTypeFromString("memory")
			Expect(err).NotTo(HaveOccurred())
			Expect(storageType).To(Equal(storage.StorageTypeMemory))

			storageType, err = storage.StorageTypeFromString("pebble")
			Expect(err).NotTo(HaveOccurred())
			Expect(storageType).To(Equal(storage.StorageTypePebble))
		})

		It("should reject invalid string conversion", func() {
			storageType, err := storage.StorageTypeFromString("invalid")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid storage type"))
			Expect(storageType).To(Equal(storage.StorageType("")))
		})

		It("should return all supported storage types", func() {
			allTypes := storage.GetAllStorageTypes()
			Expect(allTypes).To(HaveLen(4))
			Expect(allTypes).To(ContainElements(
				storage.StorageTypeMemory,
				storage.StorageTypePebble,
				storage.StorageTypeBolt,
				storage.StorageTypeBadger,
			))
		})
	})

	Describe("Backend Type Classification", func() {
		It("should identify memory backends", func() {
			Expect(storage.IsMemoryBackend(storage.StorageTypeMemory)).To(BeTrue())
			Expect(storage.IsMemoryBackend(storage.StorageTypePebble)).To(BeFalse())
			Expect(storage.IsMemoryBackend(storage.StorageTypeBolt)).To(BeFalse())
			Expect(storage.IsMemoryBackend(storage.StorageTypeBadger)).To(BeFalse())
		})

		It("should identify persistent backends", func() {
			Expect(storage.IsPersistentBackend(storage.StorageTypeMemory)).To(BeFalse())
			Expect(storage.IsPersistentBackend(storage.StorageTypePebble)).To(BeTrue())
			Expect(storage.IsPersistentBackend(storage.StorageTypeBolt)).To(BeTrue())
			Expect(storage.IsPersistentBackend(storage.StorageTypeBadger)).To(BeTrue())
		})

		It("should identify backends requiring filesystem paths", func() {
			Expect(storage.StorageTypeRequiresPath(storage.StorageTypeMemory)).To(BeFalse())
			Expect(storage.StorageTypeRequiresPath(storage.StorageTypePebble)).To(BeTrue())
			Expect(storage.StorageTypeRequiresPath(storage.StorageTypeBolt)).To(BeTrue())
			Expect(storage.StorageTypeRequiresPath(storage.StorageTypeBadger)).To(BeTrue())
		})
	})

	Describe("Default Database Names", func() {
		It("should provide appropriate default database names", func() {
			Expect(storage.GetDefaultDatabaseName(storage.StorageTypeMemory)).To(Equal("k1s.db"))
			Expect(storage.GetDefaultDatabaseName(storage.StorageTypeBolt)).To(Equal("k1s.bolt"))
			Expect(storage.GetDefaultDatabaseName(storage.StorageTypePebble)).To(Equal("k1s.pebble"))
			Expect(storage.GetDefaultDatabaseName(storage.StorageTypeBadger)).To(Equal("k1s.badger"))
		})

		It("should provide appropriate file extensions", func() {
			Expect(storage.GetDefaultFileExtension(storage.StorageTypeMemory)).To(Equal(".db"))
			Expect(storage.GetDefaultFileExtension(storage.StorageTypeBolt)).To(Equal(".bolt"))
			Expect(storage.GetDefaultFileExtension(storage.StorageTypePebble)).To(Equal(".pebble"))
			Expect(storage.GetDefaultFileExtension(storage.StorageTypeBadger)).To(Equal(".badger"))
		})
	})
})
