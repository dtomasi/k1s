package runtime

import (
	"context"
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/dtomasi/k1s/core/client"
	"github.com/dtomasi/k1s/core/defaulting"
	"github.com/dtomasi/k1s/core/events"
	"github.com/dtomasi/k1s/core/events/sinks"
	"github.com/dtomasi/k1s/core/registry"
	"github.com/dtomasi/k1s/core/storage"
	"github.com/dtomasi/k1s/core/validation"
)

// Runtime represents the core k1s runtime system that orchestrates all components
// including storage, client, event system, and other core services.
type Runtime interface {
	// Start initializes and starts all runtime components
	Start(ctx context.Context) error

	// Stop gracefully shuts down all runtime components
	Stop(ctx context.Context) error

	// GetClient returns the primary client for runtime operations
	GetClient() client.Client

	// GetEventAwareClient returns an event-aware client wrapper
	GetEventAwareClient() client.EventAwareClient

	// GetEventBroadcaster returns the event broadcaster
	GetEventBroadcaster() events.EventBroadcaster

	// GetEventRecorder returns an event recorder for the specified component
	GetEventRecorder(component string) events.EventRecorder

	// IsStarted returns true if the runtime has been started
	IsStarted() bool
}

// k1sRuntime implements the Runtime interface
type k1sRuntime struct {
	mu               sync.RWMutex
	client           client.Client
	eventAwareClient client.EventAwareClient
	eventBroadcaster events.EventBroadcaster
	eventSink        events.EventSink
	scheme           *runtime.Scheme
	started          bool
	ctx              context.Context
	cancel           context.CancelFunc
	options          RuntimeOptions
}

// RuntimeOptions provides configuration options for the k1s runtime
type RuntimeOptions struct {
	// Client is the underlying client to use
	Client client.Client

	// Scheme is the runtime scheme for object operations
	Scheme *runtime.Scheme

	// EnableEvents controls whether the event system is enabled
	EnableEvents bool

	// EventBroadcasterOptions provides configuration for the event broadcaster
	EventBroadcasterOptions events.EventBroadcasterOptions

	// DefaultComponent is the default component name for event recording
	DefaultComponent string
}

// Option is a functional option for configuring the runtime
type Option func(*Config)

// Config holds configuration for the k1s runtime
type Config struct {
	// Tenant name for multi-tenant support
	Tenant string

	// EnableRBAC controls whether RBAC is enabled
	EnableRBAC bool

	// EnableEvents controls whether the event system is enabled
	EnableEvents bool

	// DefaultComponent is the default component name for event recording
	DefaultComponent string

	// EventBroadcasterOptions provides configuration for the event broadcaster
	EventBroadcasterOptions events.EventBroadcasterOptions

	// ValidationConfig provides validation configuration (placeholder for future use)
	ValidationConfig interface{}

	// DefaultingConfig provides defaulting configuration (placeholder for future use)
	DefaultingConfig interface{}
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Tenant:           "default",
		EnableRBAC:       false,
		EnableEvents:     true,
		DefaultComponent: events.DefaultComponent,
		EventBroadcasterOptions: events.EventBroadcasterOptions{
			QueueSize:      events.DefaultEventQueueSize,
			MetricsEnabled: true,
		},
	}
}

// WithTenant sets the tenant name
func WithTenant(name string) Option {
	return func(c *Config) {
		c.Tenant = name
	}
}

// WithRBAC enables RBAC with the given configuration
func WithRBAC(enabled bool) Option {
	return func(c *Config) {
		c.EnableRBAC = enabled
	}
}

// WithEvents enables or disables the event system
func WithEvents(enabled bool) Option {
	return func(c *Config) {
		c.EnableEvents = enabled
	}
}

// WithDefaultComponent sets the default component name for events
func WithDefaultComponent(component string) Option {
	return func(c *Config) {
		c.DefaultComponent = component
	}
}

// WithValidation sets validation configuration (placeholder for future use)
func WithValidation(config interface{}) Option {
	return func(c *Config) {
		c.ValidationConfig = config
	}
}

// WithDefaulting sets defaulting configuration (placeholder for future use)
func WithDefaulting(config interface{}) Option {
	return func(c *Config) {
		c.DefaultingConfig = config
	}
}

// NewRuntime creates a new k1s runtime instance with dependency injection
// This is the new primary constructor that takes a storage backend and options
func NewRuntime(storageBackend storage.Interface, opts ...Option) (Runtime, error) {
	if storageBackend == nil {
		return nil, fmt.Errorf("storage backend is required")
	}

	// Apply default configuration
	config := DefaultConfig()

	// Apply functional options
	for _, opt := range opts {
		opt(config)
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create scheme with core resources auto-registered
	scheme := NewScheme()
	// Core resources are automatically registered in NewScheme()

	// Create resource registry
	resourceRegistry := registry.NewRegistry()

	// Initialize validation engine (use nil for now)
	var validator validation.Validator

	// Initialize defaulting engine (use nil for now)
	var defaulter defaulting.Defaulter

	// Create client with all components
	clientOptions := client.ClientOptions{
		Scheme:    scheme,
		Storage:   storageBackend,
		Registry:  resourceRegistry,
		Validator: validator,
		Defaulter: defaulter,
	}

	client, err := client.NewClient(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &k1sRuntime{
		client: client,
		scheme: scheme,
		options: RuntimeOptions{
			Client:                  client,
			Scheme:                  scheme,
			EnableEvents:            config.EnableEvents,
			DefaultComponent:        config.DefaultComponent,
			EventBroadcasterOptions: config.EventBroadcasterOptions,
		},
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// NewRuntimeWithOptions creates a new k1s runtime instance (legacy API for backward compatibility)
func NewRuntimeWithOptions(options RuntimeOptions) (Runtime, error) {
	if options.Client == nil {
		return nil, fmt.Errorf("client is required")
	}
	if options.Scheme == nil {
		return nil, fmt.Errorf("scheme is required")
	}

	// Set default component name
	if options.DefaultComponent == "" {
		options.DefaultComponent = events.DefaultComponent
	}

	// Set default event broadcaster options
	if options.EventBroadcasterOptions.QueueSize <= 0 {
		options.EventBroadcasterOptions.QueueSize = events.DefaultEventQueueSize
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &k1sRuntime{
		client:  options.Client,
		scheme:  options.Scheme,
		options: options,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

// validateConfig validates the runtime configuration
func validateConfig(config *Config) error {
	if config.Tenant == "" {
		return fmt.Errorf("tenant name cannot be empty")
	}
	if config.DefaultComponent == "" {
		return fmt.Errorf("default component cannot be empty")
	}
	return nil
}

// Start initializes and starts all runtime components
func (r *k1sRuntime) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.started {
		return fmt.Errorf("runtime is already started")
	}

	// Initialize event system if enabled
	if r.options.EnableEvents {
		r.initializeEventSystem()
	}

	r.started = true
	return nil
}

// Stop gracefully shuts down all runtime components
func (r *k1sRuntime) Stop(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.started {
		return nil
	}

	// Shutdown event system
	if r.eventBroadcaster != nil {
		r.eventBroadcaster.Shutdown()
	}

	// Cancel runtime context
	r.cancel()

	r.started = false
	return nil
}

// GetClient returns the primary client for runtime operations
func (r *k1sRuntime) GetClient() client.Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.client
}

// GetEventAwareClient returns an event-aware client wrapper
func (r *k1sRuntime) GetEventAwareClient() client.EventAwareClient {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.eventAwareClient
}

// GetEventBroadcaster returns the event broadcaster
func (r *k1sRuntime) GetEventBroadcaster() events.EventBroadcaster {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.eventBroadcaster
}

// GetEventRecorder returns an event recorder for the specified component
func (r *k1sRuntime) GetEventRecorder(component string) events.EventRecorder {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.eventBroadcaster == nil {
		return nil
	}

	source := events.NewEventSource(component)
	return r.eventBroadcaster.NewRecorder(r.scheme, source)
}

// IsStarted returns true if the runtime has been started
func (r *k1sRuntime) IsStarted() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.started
}

// initializeEventSystem sets up the event broadcasting and recording system
func (r *k1sRuntime) initializeEventSystem() {
	// Create event broadcaster
	r.eventBroadcaster = events.NewEventBroadcaster(r.options.EventBroadcasterOptions)

	// Create storage sink for events
	r.eventSink = sinks.NewStorageSink(sinks.StorageSinkOptions{
		Client:  r.client,
		Context: r.ctx,
	})

	// Start recording to storage sink
	r.eventBroadcaster.StartRecordingToSink(r.eventSink)

	// Create event-aware client wrapper
	eventRecorder := r.GetEventRecorder(r.options.DefaultComponent)
	r.eventAwareClient = client.WithEventRecording(r.client, eventRecorder)
}

// RuntimeFactory provides a factory interface for creating runtime instances
type RuntimeFactory interface {
	// CreateRuntime creates a new runtime instance (legacy API)
	CreateRuntime(options RuntimeOptions) (Runtime, error)

	// CreateRuntimeWithStorage creates a new runtime instance with dependency injection
	CreateRuntimeWithStorage(storageBackend storage.Interface, opts ...Option) (Runtime, error)
}

// DefaultRuntimeFactory implements RuntimeFactory
type DefaultRuntimeFactory struct{}

// CreateRuntime creates a new runtime instance using the default implementation
func (f *DefaultRuntimeFactory) CreateRuntime(options RuntimeOptions) (Runtime, error) {
	return NewRuntimeWithOptions(options)
}

// CreateRuntimeWithStorage creates a new runtime instance with storage backend injection
func (f *DefaultRuntimeFactory) CreateRuntimeWithStorage(storageBackend storage.Interface, opts ...Option) (Runtime, error) {
	return NewRuntime(storageBackend, opts...)
}

// Helper functions for runtime creation and management

// CreateRuntimeWithStorage creates a runtime with the specified storage backend and default options
func CreateRuntimeWithStorage(storageBackend storage.Interface) (Runtime, error) {
	return NewRuntime(storageBackend, WithEvents(true))
}

// CreateRuntimeWithClient creates a runtime with the specified client and default options (legacy)
func CreateRuntimeWithClient(client client.Client, scheme *runtime.Scheme) (Runtime, error) {
	return NewRuntimeWithOptions(RuntimeOptions{
		Client:       client,
		Scheme:       scheme,
		EnableEvents: true,
	})
}

// CreateRuntimeWithoutEvents creates a runtime with events disabled (legacy)
func CreateRuntimeWithoutEvents(client client.Client, scheme *runtime.Scheme) (Runtime, error) {
	return NewRuntimeWithOptions(RuntimeOptions{
		Client:       client,
		Scheme:       scheme,
		EnableEvents: false,
	})
}

// CreateDefaultRuntime creates a runtime with standard configuration (new API)
func CreateDefaultRuntimeWithStorage(storageBackend storage.Interface, component string) (Runtime, error) {
	return NewRuntime(
		storageBackend,
		WithEvents(true),
		WithDefaultComponent(component),
	)
}

// CreateDefaultRuntime creates a runtime with standard configuration (legacy)
func CreateDefaultRuntime(client client.Client, scheme *runtime.Scheme, component string) (Runtime, error) {
	return NewRuntimeWithOptions(RuntimeOptions{
		Client:           client,
		Scheme:           scheme,
		EnableEvents:     true,
		DefaultComponent: component,
		EventBroadcasterOptions: events.EventBroadcasterOptions{
			QueueSize:      events.DefaultEventQueueSize,
			MetricsEnabled: true,
		},
	})
}

// RuntimeManager provides lifecycle management for runtime instances
type RuntimeManager struct {
	runtime Runtime
	mu      sync.RWMutex
}

// NewRuntimeManager creates a new runtime manager
func NewRuntimeManager(runtime Runtime) *RuntimeManager {
	return &RuntimeManager{
		runtime: runtime,
	}
}

// Start starts the managed runtime
func (rm *RuntimeManager) Start(ctx context.Context) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	return rm.runtime.Start(ctx)
}

// Stop stops the managed runtime
func (rm *RuntimeManager) Stop(ctx context.Context) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	return rm.runtime.Stop(ctx)
}

// GetRuntime returns the managed runtime instance
func (rm *RuntimeManager) GetRuntime() Runtime {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.runtime
}

// Ensure implementations satisfy their interfaces
var _ Runtime = (*k1sRuntime)(nil)
var _ RuntimeFactory = (*DefaultRuntimeFactory)(nil)
