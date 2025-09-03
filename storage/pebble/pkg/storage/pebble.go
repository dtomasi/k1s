package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/cockroachdb/pebble"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"

	k1sstorage "github.com/dtomasi/k1s/core/pkg/storage"
)

// pebbleStorage implements a high-performance LSM-tree storage backend
// for k1s using PebbleDB with >3,000 ops/sec performance target.
type pebbleStorage struct {
	// db is the underlying PebbleDB instance
	db *pebble.DB

	// resourceVersions tracks resource versions for each key
	resourceVersions map[string]uint64

	// currentResourceVersion is an atomic counter for generating new resource versions
	currentResourceVersion uint64

	// watchers maintains active watches for keys/prefixes
	watchers map[string][]*k1sstorage.SimpleWatch

	// watchMu protects watcher operations
	watchMu sync.RWMutex

	// versionMu protects resourceVersions map
	versionMu sync.RWMutex

	// versioner handles resource version management
	versioner k1sstorage.SimpleVersioner

	// config contains storage configuration
	config k1sstorage.Config

	// metrics tracks operation statistics
	metrics *pebbleMetrics

	// closed indicates if the storage is closed
	closed atomic.Bool
}

// pebbleMetrics tracks performance and operational metrics
type pebbleMetrics struct {
	operations uint64
	errors     uint64
	watchers   uint64
}

// NewPebbleStorage creates a new high-performance Pebble storage backend
func NewPebbleStorage(config k1sstorage.Config) k1sstorage.Backend {
	return &pebbleStorage{
		resourceVersions: make(map[string]uint64),
		watchers:         make(map[string][]*k1sstorage.SimpleWatch),
		versioner:        k1sstorage.SimpleVersioner{},
		config:           config,
		metrics:          &pebbleMetrics{},
	}
}

// NewPebbleStorageWithPath creates a new Pebble storage backend with a custom path
func NewPebbleStorageWithPath(path string, config k1sstorage.Config) k1sstorage.Backend {
	storage := &pebbleStorage{
		resourceVersions: make(map[string]uint64),
		watchers:         make(map[string][]*k1sstorage.SimpleWatch),
		versioner:        k1sstorage.SimpleVersioner{},
		config:           config,
		metrics:          &pebbleMetrics{},
	}
	// Store the path in the KeyPrefix for custom path handling
	if path != "" {
		storage.config.KeyPrefix = path
	}
	return storage
}

// initDB initializes the PebbleDB instance if not already initialized
func (s *pebbleStorage) initDB() error {
	if s.db != nil {
		return nil
	}

	// Create directory if it doesn't exist - use custom path if provided in KeyPrefix
	dbPath := filepath.Join(".", "data", "pebble")
	if s.config.KeyPrefix != "" && filepath.IsAbs(s.config.KeyPrefix) {
		dbPath = s.config.KeyPrefix
	} else if s.config.KeyPrefix != "" {
		dbPath = s.config.KeyPrefix
	}

	// Configure PebbleDB options for high performance
	opts := &pebble.Options{
		// Optimize for high write throughput
		MemTableSize:                64 << 20, // 64MB
		MemTableStopWritesThreshold: 4,
		L0CompactionThreshold:       2,
		L0StopWritesThreshold:       12,
		MaxOpenFiles:                16384,

		// Performance tuning
		MaxConcurrentCompactions: func() int { return 3 },
		DisableWAL:               false,   // Keep WAL for ACID guarantees
		FlushSplitBytes:          2 << 20, // 2MB
	}

	db, err := pebble.Open(dbPath, opts)
	if err != nil {
		return fmt.Errorf("failed to open pebble database at %s: %w", dbPath, err)
	}

	s.db = db
	return nil
}

// Name returns the name of this storage backend
func (s *pebbleStorage) Name() string {
	return "pebble"
}

// Versioner returns the storage versioner
func (s *pebbleStorage) Versioner() storage.Versioner {
	return s.versioner
}

// Create adds a new object at a key unless it already exists
func (s *pebbleStorage) Create(ctx context.Context, key string, obj, out runtime.Object, ttl uint64) error {
	if ctx.Err() != nil {
		return k1sstorage.NewContextCancelledError(ctx)
	}

	if s.closed.Load() {
		atomic.AddUint64(&s.metrics.errors, 1)
		return fmt.Errorf("storage is closed")
	}

	if err := s.initDB(); err != nil {
		atomic.AddUint64(&s.metrics.errors, 1)
		return err
	}

	// Apply tenant/namespace prefix
	key = s.buildKey(key)

	// Check if key already exists
	_, closer, err := s.db.Get([]byte(key))
	if err == nil {
		closer.Close()
		atomic.AddUint64(&s.metrics.errors, 1)
		return fmt.Errorf("key already exists: %s", key)
	} else if err != pebble.ErrNotFound {
		atomic.AddUint64(&s.metrics.errors, 1)
		return fmt.Errorf("failed to check key existence: %w", err)
	}

	// Prepare object for storage
	if err := s.versioner.PrepareObjectForStorage(obj); err != nil {
		atomic.AddUint64(&s.metrics.errors, 1)
		return fmt.Errorf("failed to prepare object for storage: %w", err)
	}

	// Generate new resource version
	resourceVersion := atomic.AddUint64(&s.currentResourceVersion, 1)
	if err := s.versioner.UpdateObject(obj, resourceVersion); err != nil {
		atomic.AddUint64(&s.metrics.errors, 1)
		return fmt.Errorf("failed to update resource version: %w", err)
	}

	// Serialize object using JSON
	data, err := json.Marshal(obj)
	if err != nil {
		atomic.AddUint64(&s.metrics.errors, 1)
		return fmt.Errorf("failed to serialize object: %w", err)
	}

	// Store in PebbleDB with atomic transaction
	batch := s.db.NewBatch()
	if err := batch.Set([]byte(key), data, pebble.Sync); err != nil {
		batch.Close()
		atomic.AddUint64(&s.metrics.errors, 1)
		return fmt.Errorf("failed to set key in batch: %w", err)
	}

	// Store resource version mapping
	versionKey := key + "#version"
	versionData := fmt.Sprintf("%d", resourceVersion)
	if err := batch.Set([]byte(versionKey), []byte(versionData), pebble.NoSync); err != nil {
		batch.Close()
		atomic.AddUint64(&s.metrics.errors, 1)
		return fmt.Errorf("failed to set version key in batch: %w", err)
	}

	// Commit the transaction
	if err := batch.Commit(pebble.Sync); err != nil {
		batch.Close()
		atomic.AddUint64(&s.metrics.errors, 1)
		return fmt.Errorf("failed to commit batch: %w", err)
	}
	batch.Close()

	// Update in-memory resource version tracking
	s.versionMu.Lock()
	s.resourceVersions[key] = resourceVersion
	s.versionMu.Unlock()

	// Copy to output object if provided
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			atomic.AddUint64(&s.metrics.errors, 1)
			return fmt.Errorf("failed to unmarshal to output object: %w", err)
		}
	}

	// Notify watchers
	s.notifyWatchers(key, watch.Added, obj)

	atomic.AddUint64(&s.metrics.operations, 1)
	return nil
}

// Delete removes the specified key and returns the value that existed at that key
func (s *pebbleStorage) Delete(ctx context.Context, key string, out runtime.Object,
	preconditions *storage.Preconditions, validateDeletion storage.ValidateObjectFunc,
	cachedExistingObject runtime.Object) error {

	if ctx.Err() != nil {
		return k1sstorage.NewContextCancelledError(ctx)
	}

	if s.closed.Load() {
		atomic.AddUint64(&s.metrics.errors, 1)
		return fmt.Errorf("storage is closed")
	}

	if err := s.initDB(); err != nil {
		atomic.AddUint64(&s.metrics.errors, 1)
		return err
	}

	// Apply tenant/namespace prefix
	key = s.buildKey(key)

	// Get existing object
	var existingObj runtime.Object
	if cachedExistingObject != nil {
		existingObj = cachedExistingObject
	} else {
		data, closer, err := s.db.Get([]byte(key))
		if err == pebble.ErrNotFound {
			atomic.AddUint64(&s.metrics.errors, 1)
			return fmt.Errorf("key not found: %s", key)
		} else if err != nil {
			atomic.AddUint64(&s.metrics.errors, 1)
			return fmt.Errorf("failed to get existing object: %w", err)
		}
		defer closer.Close()

		existingObj = &runtime.Unknown{}
		if err := json.Unmarshal(data, existingObj); err != nil {
			atomic.AddUint64(&s.metrics.errors, 1)
			return fmt.Errorf("failed to unmarshal existing object: %w", err)
		}
	}

	// Validate preconditions if provided
	if preconditions != nil {
		if err := s.checkPreconditions(existingObj, preconditions); err != nil {
			atomic.AddUint64(&s.metrics.errors, 1)
			return err
		}
	}

	// Validate deletion if provided
	if validateDeletion != nil {
		if err := validateDeletion(ctx, existingObj); err != nil {
			atomic.AddUint64(&s.metrics.errors, 1)
			return err
		}
	}

	// Copy existing object to output if provided
	if out != nil {
		if err := s.copyObject(existingObj, out); err != nil {
			atomic.AddUint64(&s.metrics.errors, 1)
			return fmt.Errorf("failed to copy to output object: %w", err)
		}
	}

	// Delete from PebbleDB with atomic transaction
	batch := s.db.NewBatch()
	if err := batch.Delete([]byte(key), pebble.Sync); err != nil {
		batch.Close()
		atomic.AddUint64(&s.metrics.errors, 1)
		return fmt.Errorf("failed to delete key in batch: %w", err)
	}

	// Delete resource version mapping
	versionKey := key + "#version"
	if err := batch.Delete([]byte(versionKey), pebble.NoSync); err != nil {
		batch.Close()
		atomic.AddUint64(&s.metrics.errors, 1)
		return fmt.Errorf("failed to delete version key in batch: %w", err)
	}

	// Commit the transaction
	if err := batch.Commit(pebble.Sync); err != nil {
		batch.Close()
		atomic.AddUint64(&s.metrics.errors, 1)
		return fmt.Errorf("failed to commit batch: %w", err)
	}
	batch.Close()

	// Update in-memory resource version tracking
	s.versionMu.Lock()
	delete(s.resourceVersions, key)
	s.versionMu.Unlock()

	// Notify watchers
	s.notifyWatchers(key, watch.Deleted, existingObj)

	atomic.AddUint64(&s.metrics.operations, 1)
	return nil
}

// Get unmarshals object found at key into objPtr
func (s *pebbleStorage) Get(ctx context.Context, key string, opts storage.GetOptions, objPtr runtime.Object) error {
	if ctx.Err() != nil {
		return k1sstorage.NewContextCancelledError(ctx)
	}

	if s.closed.Load() {
		atomic.AddUint64(&s.metrics.errors, 1)
		return fmt.Errorf("storage is closed")
	}

	if err := s.initDB(); err != nil {
		atomic.AddUint64(&s.metrics.errors, 1)
		return err
	}

	// Apply tenant/namespace prefix
	key = s.buildKey(key)

	// Get data from PebbleDB
	data, closer, err := s.db.Get([]byte(key))
	if err == pebble.ErrNotFound {
		atomic.AddUint64(&s.metrics.errors, 1)
		if opts.IgnoreNotFound {
			return nil
		}
		return fmt.Errorf("key not found: %s", key)
	} else if err != nil {
		atomic.AddUint64(&s.metrics.errors, 1)
		return fmt.Errorf("failed to get key: %w", err)
	}
	defer closer.Close()

	// Check resource version if specified
	if opts.ResourceVersion != "" {
		requestedVersion, err := s.versioner.ParseWatchResourceVersion(opts.ResourceVersion)
		if err != nil {
			atomic.AddUint64(&s.metrics.errors, 1)
			return fmt.Errorf("failed to parse resource version %s: %w", opts.ResourceVersion, err)
		}

		s.versionMu.RLock()
		storedVersion := s.resourceVersions[key]
		s.versionMu.RUnlock()

		if requestedVersion != 0 && requestedVersion != storedVersion {
			atomic.AddUint64(&s.metrics.errors, 1)
			return fmt.Errorf("resource version mismatch: requested %s, stored %d", opts.ResourceVersion, storedVersion)
		}
	}

	// Unmarshal data into objPtr
	if err := json.Unmarshal(data, objPtr); err != nil {
		atomic.AddUint64(&s.metrics.errors, 1)
		return fmt.Errorf("failed to unmarshal object: %w", err)
	}

	atomic.AddUint64(&s.metrics.operations, 1)
	return nil
}

// List unmarshalls objects found at key into a List api object
func (s *pebbleStorage) List(ctx context.Context, key string, opts storage.ListOptions, listObj runtime.Object) error {
	if ctx.Err() != nil {
		return k1sstorage.NewContextCancelledError(ctx)
	}

	if s.closed.Load() {
		atomic.AddUint64(&s.metrics.errors, 1)
		return fmt.Errorf("storage is closed")
	}

	if err := s.initDB(); err != nil {
		atomic.AddUint64(&s.metrics.errors, 1)
		return err
	}

	// Apply tenant/namespace prefix
	key = s.buildKey(key)

	var matchedObjects []runtime.Object
	maxResourceVersion := uint64(0)

	// Create iterator for efficient prefix scanning
	prefixIterOptions := &pebble.IterOptions{}
	if opts.Recursive {
		prefixIterOptions.LowerBound = []byte(key)
		prefixIterOptions.UpperBound = []byte(key + "\xFF") // Prefix scan
	} else {
		// Exact match
		prefixIterOptions.LowerBound = []byte(key)
		prefixIterOptions.UpperBound = []byte(key + "\x00")
	}

	iter, err := s.db.NewIter(prefixIterOptions)
	if err != nil {
		atomic.AddUint64(&s.metrics.errors, 1)
		return fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()

	// Iterate through matching keys
	for iter.First(); iter.Valid(); iter.Next() {
		storageKey := string(iter.Key())

		// Skip version keys
		if strings.HasSuffix(storageKey, "#version") {
			continue
		}

		var matches bool
		if opts.Recursive {
			matches = strings.HasPrefix(storageKey, key)
		} else {
			matches = storageKey == key
		}

		if !matches {
			continue
		}

		// For now, create a placeholder object for listing
		// Real implementation should properly unmarshal based on object type
		obj := &runtime.Unknown{
			Raw: iter.Value(),
		}
		obj.APIVersion = "unknown/v1"
		obj.Kind = "Unknown"

		matchedObjects = append(matchedObjects, obj)

		// Track max resource version
		s.versionMu.RLock()
		if objVersion := s.resourceVersions[storageKey]; objVersion > maxResourceVersion {
			maxResourceVersion = objVersion
		}
		s.versionMu.RUnlock()
	}

	// Check for iterator errors
	if err := iter.Error(); err != nil {
		atomic.AddUint64(&s.metrics.errors, 1)
		return fmt.Errorf("iterator error: %w", err)
	}

	// Set list metadata
	if err := s.versioner.UpdateList(listObj, maxResourceVersion, "", nil); err != nil {
		atomic.AddUint64(&s.metrics.errors, 1)
		return fmt.Errorf("failed to update list metadata: %w", err)
	}

	// Set the items in the list
	if err := meta.SetList(listObj, matchedObjects); err != nil {
		atomic.AddUint64(&s.metrics.errors, 1)
		return fmt.Errorf("failed to set list items: %w", err)
	}

	atomic.AddUint64(&s.metrics.operations, 1)
	return nil
}

// Watch begins watching a specific key or key prefix for changes
func (s *pebbleStorage) Watch(ctx context.Context, key string, opts storage.ListOptions) (watch.Interface, error) {
	if ctx.Err() != nil {
		return nil, k1sstorage.NewContextCancelledError(ctx)
	}

	if s.closed.Load() {
		return nil, fmt.Errorf("storage is closed")
	}

	if err := s.initDB(); err != nil {
		return nil, err
	}

	// Apply tenant/namespace prefix
	key = s.buildKey(key)

	// Create watch instance
	w := k1sstorage.NewSimpleWatch()

	s.watchMu.Lock()
	// Add to watchers
	s.watchers[key] = append(s.watchers[key], w)
	atomic.AddUint64(&s.metrics.watchers, 1)
	s.watchMu.Unlock()

	// Start background cleanup when context is cancelled
	go func() {
		<-ctx.Done()
		s.removeWatcher(key, w)
		w.Stop()
	}()

	// Send initial events if requested
	if opts.SendInitialEvents != nil && *opts.SendInitialEvents {
		prefixIterOptions := &pebble.IterOptions{}
		if opts.Recursive {
			prefixIterOptions.LowerBound = []byte(key)
			prefixIterOptions.UpperBound = []byte(key + "\xFF")
		} else {
			prefixIterOptions.LowerBound = []byte(key)
			prefixIterOptions.UpperBound = []byte(key + "\x00")
		}

		iter, err := s.db.NewIter(prefixIterOptions)
		if err == nil {
			defer iter.Close()
			for iter.First(); iter.Valid(); iter.Next() {
				storageKey := string(iter.Key())

				// Skip version keys
				if strings.HasSuffix(storageKey, "#version") {
					continue
				}

				var matches bool
				if opts.Recursive {
					matches = strings.HasPrefix(storageKey, key)
				} else {
					matches = storageKey == key
				}

				if matches {
					obj := &runtime.Unknown{}
					if json.Unmarshal(iter.Value(), obj) == nil {
						w.Send(watch.Added, obj)
					}
				}
			}
		}
	}

	return w, nil
}

// Close closes the storage backend and cleans up resources
func (s *pebbleStorage) Close() error {
	s.closed.Store(true)

	s.watchMu.Lock()
	// Stop all watchers
	for _, watchList := range s.watchers {
		for _, w := range watchList {
			w.Stop()
		}
	}
	// Clear watchers
	s.watchers = make(map[string][]*k1sstorage.SimpleWatch)
	s.watchMu.Unlock()

	// Clear resource versions
	s.versionMu.Lock()
	s.resourceVersions = make(map[string]uint64)
	s.versionMu.Unlock()

	// Close PebbleDB
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			return fmt.Errorf("failed to close pebble database: %w", err)
		}
		s.db = nil
	}

	return nil
}

// Compact performs storage compaction
func (s *pebbleStorage) Compact(ctx context.Context) error {
	if s.closed.Load() {
		return fmt.Errorf("storage is closed")
	}

	if err := s.initDB(); err != nil {
		return err
	}

	// Perform manual compaction on the entire keyspace
	if err := s.db.Compact([]byte(""), []byte("\xFF"), true); err != nil {
		return fmt.Errorf("failed to compact database: %w", err)
	}

	return nil
}

// Count returns the number of objects stored under the given key prefix
func (s *pebbleStorage) Count(ctx context.Context, key string) (int64, error) {
	if ctx.Err() != nil {
		return 0, k1sstorage.NewContextCancelledError(ctx)
	}

	if s.closed.Load() {
		return 0, fmt.Errorf("storage is closed")
	}

	if err := s.initDB(); err != nil {
		return 0, err
	}

	// Apply tenant/namespace prefix
	key = s.buildKey(key)

	count := int64(0)

	// Create iterator for efficient prefix scanning
	prefixIterOptions := &pebble.IterOptions{
		LowerBound: []byte(key),
		UpperBound: []byte(key + "\xFF"),
	}

	iter, err := s.db.NewIter(prefixIterOptions)
	if err != nil {
		return 0, fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()

	// Count matching keys
	for iter.First(); iter.Valid(); iter.Next() {
		storageKey := string(iter.Key())

		// Skip version keys
		if strings.HasSuffix(storageKey, "#version") {
			continue
		}

		if strings.HasPrefix(storageKey, key) {
			count++
		}
	}

	// Check for iterator errors
	if err := iter.Error(); err != nil {
		return 0, fmt.Errorf("iterator error: %w", err)
	}

	return count, nil
}

// buildKey constructs the final storage key with tenant/namespace prefixes
func (s *pebbleStorage) buildKey(key string) string {
	parts := []string{}

	// Add tenant prefix if configured
	if s.config.TenantID != "" {
		parts = append(parts, "tenants", s.config.TenantID)
	}

	// Add custom key prefix if configured
	if s.config.KeyPrefix != "" {
		parts = append(parts, s.config.KeyPrefix)
	}

	// Add namespace prefix if configured
	if s.config.Namespace != "" {
		parts = append(parts, "namespaces", s.config.Namespace)
	}

	// Add the actual key
	parts = append(parts, key)

	return strings.Join(parts, "/")
}

// checkPreconditions validates storage preconditions
func (s *pebbleStorage) checkPreconditions(obj runtime.Object, preconditions *storage.Preconditions) error {
	if preconditions == nil {
		return nil
	}

	// Check UID precondition
	if preconditions.UID != nil {
		accessor, err := meta.Accessor(obj)
		if err != nil {
			return fmt.Errorf("failed to get object accessor: %w", err)
		}
		if accessor.GetUID() != *preconditions.UID {
			return fmt.Errorf("UID mismatch: expected %s, got %s", *preconditions.UID, accessor.GetUID())
		}
	}

	// Check ResourceVersion precondition
	if preconditions.ResourceVersion != nil {
		accessor, err := meta.Accessor(obj)
		if err != nil {
			return fmt.Errorf("failed to get object accessor: %w", err)
		}
		if accessor.GetResourceVersion() != *preconditions.ResourceVersion {
			return fmt.Errorf("resource version mismatch: expected %s, got %s",
				*preconditions.ResourceVersion, accessor.GetResourceVersion())
		}
	}

	return nil
}

// copyObject copies source object to destination
func (s *pebbleStorage) copyObject(src, dst runtime.Object) error {
	data, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("failed to marshal source object: %w", err)
	}

	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("failed to unmarshal to destination object: %w", err)
	}

	return nil
}

// notifyWatchers sends watch events to all registered watchers
func (s *pebbleStorage) notifyWatchers(key string, eventType watch.EventType, obj runtime.Object) {
	s.watchMu.RLock()
	defer s.watchMu.RUnlock()

	// Notify direct key watchers
	if watchers, exists := s.watchers[key]; exists {
		for _, w := range watchers {
			w.Send(eventType, obj)
		}
	}

	// Notify prefix watchers
	for watchKey, watchers := range s.watchers {
		if watchKey != key && strings.HasPrefix(key, watchKey) {
			for _, w := range watchers {
				w.Send(eventType, obj)
			}
		}
	}
}

// removeWatcher removes a watcher from the watchers map
func (s *pebbleStorage) removeWatcher(key string, watcher *k1sstorage.SimpleWatch) {
	s.watchMu.Lock()
	defer s.watchMu.Unlock()

	if watchers, exists := s.watchers[key]; exists {
		for i, w := range watchers {
			if w == watcher {
				// Remove watcher from slice
				s.watchers[key] = append(watchers[:i], watchers[i+1:]...)
				atomic.AddUint64(&s.metrics.watchers, ^uint64(0)) // atomic decrement
				break
			}
		}

		// Clean up empty watcher lists
		if len(s.watchers[key]) == 0 {
			delete(s.watchers, key)
		}
	}
}

// GetMetrics returns current performance metrics
func (s *pebbleStorage) GetMetrics() (operations, errors, watchers uint64) {
	return atomic.LoadUint64(&s.metrics.operations),
		atomic.LoadUint64(&s.metrics.errors),
		atomic.LoadUint64(&s.metrics.watchers)
}

// GetStats returns PebbleDB internal statistics
func (s *pebbleStorage) GetStats() string {
	if s.db == nil {
		return "database not initialized"
	}

	return s.db.Metrics().String()
}
