package runtime

import (
	"context"
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/dtomasi/k1s/core/client"
	"github.com/dtomasi/k1s/core/events"
	"github.com/dtomasi/k1s/core/events/sinks"
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

// NewRuntime creates a new k1s runtime instance
func NewRuntime(options RuntimeOptions) (Runtime, error) {
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
	// CreateRuntime creates a new runtime instance
	CreateRuntime(options RuntimeOptions) (Runtime, error)
}

// DefaultRuntimeFactory implements RuntimeFactory
type DefaultRuntimeFactory struct{}

// CreateRuntime creates a new runtime instance using the default implementation
func (f *DefaultRuntimeFactory) CreateRuntime(options RuntimeOptions) (Runtime, error) {
	return NewRuntime(options)
}

// Helper functions for runtime creation and management

// CreateRuntimeWithClient creates a runtime with the specified client and default options
func CreateRuntimeWithClient(client client.Client, scheme *runtime.Scheme) (Runtime, error) {
	return NewRuntime(RuntimeOptions{
		Client:       client,
		Scheme:       scheme,
		EnableEvents: true,
	})
}

// CreateRuntimeWithoutEvents creates a runtime with events disabled
func CreateRuntimeWithoutEvents(client client.Client, scheme *runtime.Scheme) (Runtime, error) {
	return NewRuntime(RuntimeOptions{
		Client:       client,
		Scheme:       scheme,
		EnableEvents: false,
	})
}

// CreateDefaultRuntime creates a runtime with standard configuration
func CreateDefaultRuntime(client client.Client, scheme *runtime.Scheme, component string) (Runtime, error) {
	return NewRuntime(RuntimeOptions{
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
