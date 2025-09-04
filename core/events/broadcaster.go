package events

import (
	"context"
	"sync"
	"sync/atomic"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

// eventBroadcaster implements the EventBroadcaster interface
type eventBroadcaster struct {
	mu        sync.RWMutex
	sinks     map[int]sinkRegistration
	watchers  map[int]watcherRegistration
	nextID    int
	eventChan chan *corev1.Event
	stopCh    chan struct{}
	metrics   *EventMetrics
	options   EventBroadcasterOptions
	started   bool
	ctx       context.Context
	cancel    context.CancelFunc
}

// sinkRegistration represents a registered event sink
type sinkRegistration struct {
	id      int
	sink    EventSink
	stopCh  chan struct{}
	watcher *sinkWatcher
}

// watcherRegistration represents a registered event watcher
type watcherRegistration struct {
	id      int
	handler func(*corev1.Event)
	stopCh  chan struct{}
	watcher *eventWatcher
}

// sinkWatcher implements watch.Interface for event sinks
type sinkWatcher struct {
	id          int
	stopCh      chan struct{}
	resultCh    chan watch.Event
	broadcaster *eventBroadcaster
}

// eventWatcher implements watch.Interface for event watchers
type eventWatcher struct {
	id          int
	stopCh      chan struct{}
	resultCh    chan watch.Event
	broadcaster *eventBroadcaster
}

// NewEventBroadcaster creates a new EventBroadcaster instance
func NewEventBroadcaster(options EventBroadcasterOptions) EventBroadcaster {
	if options.QueueSize <= 0 {
		options.QueueSize = DefaultEventQueueSize
	}
	if options.Clock == nil {
		options.Clock = RealClock{}
	}

	ctx, cancel := context.WithCancel(context.Background())

	broadcaster := &eventBroadcaster{
		sinks:     make(map[int]sinkRegistration),
		watchers:  make(map[int]watcherRegistration),
		eventChan: make(chan *corev1.Event, options.QueueSize),
		stopCh:    make(chan struct{}),
		metrics:   &EventMetrics{},
		options:   options,
		ctx:       ctx,
		cancel:    cancel,
	}

	return broadcaster
}

// StartRecordingToSink begins recording events to the given sink
func (b *eventBroadcaster) StartRecordingToSink(sink EventSink) watch.Interface {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := b.nextID
	b.nextID++

	stopCh := make(chan struct{})
	watcher := &sinkWatcher{
		id:          id,
		stopCh:      stopCh,
		resultCh:    make(chan watch.Event),
		broadcaster: b,
	}

	registration := sinkRegistration{
		id:      id,
		sink:    sink,
		stopCh:  stopCh,
		watcher: watcher,
	}

	b.sinks[id] = registration
	atomic.AddInt32(&b.metrics.SinksActive, 1)

	// Start the broadcaster if this is the first sink
	if !b.started {
		b.start()
	}

	return watcher
}

// StartEventWatcher begins watching events and calling the provided handler
func (b *eventBroadcaster) StartEventWatcher(eventHandler func(*corev1.Event)) watch.Interface {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := b.nextID
	b.nextID++

	stopCh := make(chan struct{})
	watcher := &eventWatcher{
		id:          id,
		stopCh:      stopCh,
		resultCh:    make(chan watch.Event),
		broadcaster: b,
	}

	registration := watcherRegistration{
		id:      id,
		handler: eventHandler,
		stopCh:  stopCh,
		watcher: watcher,
	}

	b.watchers[id] = registration
	atomic.AddInt32(&b.metrics.WatchersActive, 1)

	// Start the broadcaster if this is the first watcher
	if !b.started {
		b.start()
	}

	// Start a goroutine to handle the watcher
	go b.runWatcher(registration)

	return watcher
}

// NewRecorder returns an EventRecorder that records to this broadcaster
func (b *eventBroadcaster) NewRecorder(scheme *runtime.Scheme, source corev1.EventSource) EventRecorder {
	options := EventRecorderOptions{
		Scheme: scheme,
		Source: source,
		Clock:  b.options.Clock,
	}
	return NewEventRecorder(b, options)
}

// Shutdown gracefully shuts down the broadcaster and all its watchers
func (b *eventBroadcaster) Shutdown() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.started {
		return
	}

	// Stop all sinks
	for _, sink := range b.sinks {
		select {
		case <-sink.stopCh:
			// Already closed
		default:
			close(sink.stopCh)
		}
	}

	// Stop all watchers
	for _, watcher := range b.watchers {
		select {
		case <-watcher.stopCh:
			// Already closed
		default:
			close(watcher.stopCh)
		}
	}

	// Stop the broadcaster
	b.cancel()
	select {
	case <-b.stopCh:
		// Already closed
	default:
		close(b.stopCh)
	}
	b.started = false

	// Reset metrics
	atomic.StoreInt32(&b.metrics.SinksActive, 0)
	atomic.StoreInt32(&b.metrics.WatchersActive, 0)
}

// start begins the broadcaster's main event distribution loop
func (b *eventBroadcaster) start() {
	if b.started {
		return
	}
	b.started = true
	go b.run()
}

// run is the main event distribution loop
func (b *eventBroadcaster) run() {
	for {
		select {
		case event := <-b.eventChan:
			b.distributeEvent(event)
		case <-b.stopCh:
			return
		case <-b.ctx.Done():
			return
		}
	}
}

// distributeEvent sends an event to all registered sinks and watchers
func (b *eventBroadcaster) distributeEvent(event *corev1.Event) {
	b.mu.RLock()
	sinks := make([]sinkRegistration, 0, len(b.sinks))
	watchers := make([]watcherRegistration, 0, len(b.watchers))

	for _, sink := range b.sinks {
		sinks = append(sinks, sink)
	}
	for _, watcher := range b.watchers {
		watchers = append(watchers, watcher)
	}
	b.mu.RUnlock()

	// Send to sinks
	for _, sink := range sinks {
		go b.sendToSink(sink, event)
	}

	// Send to watchers
	for _, watcher := range watchers {
		go b.sendToWatcher(watcher, event)
	}
}

// sendToSink sends an event to a specific sink
func (b *eventBroadcaster) sendToSink(registration sinkRegistration, event *corev1.Event) {
	select {
	case <-registration.stopCh:
		return
	default:
	}

	// Try to create or update the event in the sink
	_, err := registration.sink.Create(event)
	if err != nil {
		// If create failed, try update (for event aggregation)
		_, err = registration.sink.Update(event)
		if err != nil {
			atomic.AddInt64(&b.metrics.EventsDropped, 1)
		}
	}
}

// sendToWatcher sends an event to a specific watcher
func (b *eventBroadcaster) sendToWatcher(registration watcherRegistration, event *corev1.Event) {
	select {
	case <-registration.stopCh:
		return
	default:
	}

	// Call the handler function
	registration.handler(event)
}

// runWatcher runs the event watcher goroutine
func (b *eventBroadcaster) runWatcher(registration watcherRegistration) {
	defer func() {
		b.mu.Lock()
		delete(b.watchers, registration.id)
		atomic.AddInt32(&b.metrics.WatchersActive, -1)
		b.mu.Unlock()
	}()

	<-registration.stopCh
}

// recordEvent is called by the EventRecorder to send events to the broadcaster
func (b *eventBroadcaster) recordEvent(event *corev1.Event) {
	select {
	case b.eventChan <- event:
		// Event queued successfully
	default:
		// Queue is full, drop the event
		atomic.AddInt64(&b.metrics.EventsDropped, 1)
	}
}

// GetMetrics returns current metrics for the broadcaster
func (b *eventBroadcaster) GetMetrics() EventMetrics {
	return EventMetrics{
		EventsRecorded: atomic.LoadInt64(&b.metrics.EventsRecorded),
		EventsDropped:  atomic.LoadInt64(&b.metrics.EventsDropped),
		SinksActive:    atomic.LoadInt32(&b.metrics.SinksActive),
		WatchersActive: atomic.LoadInt32(&b.metrics.WatchersActive),
	}
}

// sinkWatcher implementation of watch.Interface

// Stop stops the sink watcher
func (w *sinkWatcher) Stop() {
	w.broadcaster.mu.Lock()
	defer w.broadcaster.mu.Unlock()

	if registration, exists := w.broadcaster.sinks[w.id]; exists {
		close(registration.stopCh)
		delete(w.broadcaster.sinks, w.id)
		atomic.AddInt32(&w.broadcaster.metrics.SinksActive, -1)
	}
}

// ResultChan returns the channel for watch results
func (w *sinkWatcher) ResultChan() <-chan watch.Event {
	return w.resultCh
}

// eventWatcher implementation of watch.Interface

// Stop stops the event watcher
func (w *eventWatcher) Stop() {
	w.broadcaster.mu.Lock()
	defer w.broadcaster.mu.Unlock()

	if registration, exists := w.broadcaster.watchers[w.id]; exists {
		close(registration.stopCh)
		// The actual cleanup happens in runWatcher goroutine
	}
}

// ResultChan returns the channel for watch results
func (w *eventWatcher) ResultChan() <-chan watch.Event {
	return w.resultCh
}

// Ensure implementations satisfy watch.Interface
var _ watch.Interface = (*sinkWatcher)(nil)
var _ watch.Interface = (*eventWatcher)(nil)
