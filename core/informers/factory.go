package informers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/dtomasi/k1s/core/client"
)

// SharedInformerFactory provides shared informers for k1s resources.
// It implements a controller-runtime compatible interface while being optimized for CLI applications.
type SharedInformerFactory interface {
	// ForResource returns a GenericInformer for the given resource.
	ForResource(gvr schema.GroupVersionResource) GenericInformer

	// Start starts all informers that have been requested.
	// This is optimized for CLI applications and only starts informers on-demand.
	Start(stopCh <-chan struct{})

	// Shutdown gracefully stops all informers.
	Shutdown()

	// WaitForCacheSync waits for all started informers to sync their caches.
	WaitForCacheSync(stopCh <-chan struct{}) map[schema.GroupVersionResource]bool

	// InformerFor returns a SharedIndexInformer for the given resource.
	InformerFor(gvr schema.GroupVersionResource) cache.SharedIndexInformer
}

// GenericInformer provides access to a SharedIndexInformer and GenericLister.
type GenericInformer interface {
	// Informer returns the underlying SharedIndexInformer.
	Informer() cache.SharedIndexInformer

	// Lister returns a GenericLister for this resource.
	Lister() cache.GenericLister
}

// sharedInformerFactory implements SharedInformerFactory for k1s.
type sharedInformerFactory struct {
	client client.WithWatch

	// defaultResync is the default resync period for informers
	defaultResync time.Duration

	// customResync allows setting custom resync periods per resource
	customResync map[schema.GroupVersionResource]time.Duration

	// informers tracks all created informers
	informers map[schema.GroupVersionResource]cache.SharedIndexInformer

	// genericInformers tracks all created generic informers
	genericInformers map[schema.GroupVersionResource]GenericInformer

	// started tracks which informers have been started
	started map[schema.GroupVersionResource]bool

	// startedLock protects the started map
	startedLock sync.Mutex

	// lock protects the maps
	lock sync.RWMutex

	// namespace restricts informers to a specific namespace (empty = all namespaces)
	namespace string

	// ctx is the factory context
	ctx    context.Context
	cancel context.CancelFunc

	// shutdownOnce ensures shutdown is called only once
	shutdownOnce sync.Once
}

// NewSharedInformerFactory creates a new SharedInformerFactory for k1s.
func NewSharedInformerFactory(clientImpl client.Client, defaultResync time.Duration) SharedInformerFactory {
	watchClient, err := client.NewWatchClient(clientImpl)
	if err != nil {
		panic(fmt.Errorf("failed to create watch client: %w", err))
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &sharedInformerFactory{
		client:           watchClient,
		defaultResync:    defaultResync,
		customResync:     make(map[schema.GroupVersionResource]time.Duration),
		informers:        make(map[schema.GroupVersionResource]cache.SharedIndexInformer),
		genericInformers: make(map[schema.GroupVersionResource]GenericInformer),
		started:          make(map[schema.GroupVersionResource]bool),
		ctx:              ctx,
		cancel:           cancel,
	}
}

// NewSharedInformerFactoryWithOptions creates a SharedInformerFactory with options.
func NewSharedInformerFactoryWithOptions(clientImpl client.Client, defaultResync time.Duration, options ...SharedInformerFactoryOption) SharedInformerFactory {
	factory := NewSharedInformerFactory(clientImpl, defaultResync).(*sharedInformerFactory)

	for _, opt := range options {
		opt.Apply(factory)
	}

	return factory
}

// SharedInformerFactoryOption configures a SharedInformerFactory.
type SharedInformerFactoryOption interface {
	Apply(*sharedInformerFactory)
}

// WithNamespace creates an option to restrict informers to a specific namespace.
func WithNamespace(namespace string) SharedInformerFactoryOption {
	return &namespaceOption{namespace: namespace}
}

type namespaceOption struct {
	namespace string
}

func (o *namespaceOption) Apply(factory *sharedInformerFactory) {
	factory.namespace = o.namespace
}

// WithCustomResync creates an option to set custom resync periods for specific resources.
func WithCustomResync(resync map[schema.GroupVersionResource]time.Duration) SharedInformerFactoryOption {
	return &resyncOption{resync: resync}
}

type resyncOption struct {
	resync map[schema.GroupVersionResource]time.Duration
}

func (o *resyncOption) Apply(factory *sharedInformerFactory) {
	for gvr, resync := range o.resync {
		factory.customResync[gvr] = resync
	}
}

// ForResource returns a GenericInformer for the given resource.
func (f *sharedInformerFactory) ForResource(gvr schema.GroupVersionResource) GenericInformer {
	f.lock.Lock()
	defer f.lock.Unlock()

	if gi, exists := f.genericInformers[gvr]; exists {
		return gi
	}

	// Create the informer
	informer := f.createInformer(gvr)
	f.informers[gvr] = informer

	// Create the generic informer wrapper
	gr := schema.GroupResource{Group: gvr.Group, Resource: gvr.Resource}
	gi := &genericInformer{
		informer: informer,
		lister:   cache.NewGenericLister(informer.GetIndexer(), gr),
	}

	f.genericInformers[gvr] = gi
	return gi
}

// InformerFor returns a SharedIndexInformer for the given resource.
func (f *sharedInformerFactory) InformerFor(gvr schema.GroupVersionResource) cache.SharedIndexInformer {
	f.lock.Lock()
	defer f.lock.Unlock()

	if informer, exists := f.informers[gvr]; exists {
		return informer
	}

	informer := f.createInformer(gvr)
	f.informers[gvr] = informer
	return informer
}

// createInformer creates a SharedIndexInformer for the given resource.
func (f *sharedInformerFactory) createInformer(gvr schema.GroupVersionResource) cache.SharedIndexInformer {
	resyncPeriod := f.defaultResync
	if customResync, exists := f.customResync[gvr]; exists {
		resyncPeriod = customResync
	}

	// Create list/watch functions
	listWatch := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			listOpts := []client.ListOption{}

			if f.namespace != "" {
				listOpts = append(listOpts, client.InNamespace(f.namespace))
			}

			if options.LabelSelector != "" {
				// Parse label selector - simplified implementation
				labels := parseLabelSelector(options.LabelSelector)
				listOpts = append(listOpts, client.MatchingLabels(labels))
			}

			if options.FieldSelector != "" {
				// Parse field selector - simplified implementation
				fields := parseFieldSelector(options.FieldSelector)
				listOpts = append(listOpts, client.MatchingFields(fields))
			}

			// Get the list type for this GVR
			listObj, err := f.getListObjectForGVR(gvr)
			if err != nil {
				return nil, fmt.Errorf("failed to get list object for %s: %w", gvr, err)
			}

			if err := f.client.List(f.ctx, listObj, listOpts...); err != nil {
				return nil, fmt.Errorf("failed to list %s: %w", gvr, err)
			}

			return listObj, nil
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			watchOpts := []client.WatchOption{}

			if f.namespace != "" {
				watchOpts = append(watchOpts, client.InNamespace(f.namespace))
			}

			if options.LabelSelector != "" {
				labels := parseLabelSelector(options.LabelSelector)
				watchOpts = append(watchOpts, client.MatchingLabels(labels))
			}

			if options.FieldSelector != "" {
				fields := parseFieldSelector(options.FieldSelector)
				watchOpts = append(watchOpts, client.MatchingFields(fields))
			}

			// Get the list type for this GVR
			listObj, err := f.getListObjectForGVR(gvr)
			if err != nil {
				return nil, fmt.Errorf("failed to get list object for %s: %w", gvr, err)
			}

			return f.client.Watch(f.ctx, listObj, watchOpts...)
		},
	}

	// Create the SharedIndexInformer
	informer := cache.NewSharedIndexInformer(
		listWatch,
		f.getObjectForGVR(gvr),
		resyncPeriod,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)

	return informer
}

// Start starts all requested informers.
// This is optimized for CLI applications - informers are started on-demand.
func (f *sharedInformerFactory) Start(stopCh <-chan struct{}) {
	f.startedLock.Lock()
	defer f.startedLock.Unlock()

	f.lock.RLock()
	defer f.lock.RUnlock()

	for gvr, informer := range f.informers {
		if !f.started[gvr] {
			go informer.Run(stopCh)
			f.started[gvr] = true
		}
	}
}

// Shutdown gracefully stops all informers.
func (f *sharedInformerFactory) Shutdown() {
	f.shutdownOnce.Do(func() {
		f.cancel()
	})
}

// WaitForCacheSync waits for all started informers to sync their caches.
func (f *sharedInformerFactory) WaitForCacheSync(stopCh <-chan struct{}) map[schema.GroupVersionResource]bool {
	f.lock.RLock()
	defer f.lock.RUnlock()

	res := make(map[schema.GroupVersionResource]bool)

	for gvr, informer := range f.informers {
		f.startedLock.Lock()
		if f.started[gvr] {
			res[gvr] = cache.WaitForCacheSync(stopCh, informer.HasSynced)
		} else {
			res[gvr] = true // Not started means considered synced
		}
		f.startedLock.Unlock()
	}

	return res
}

// Helper methods for resource type handling

func (f *sharedInformerFactory) getObjectForGVR(gvr schema.GroupVersionResource) runtime.Object {
	// Convert GVR to GVK
	var gvk schema.GroupVersionKind

	if f.client.RESTMapper() != nil {
		if k, err := f.client.RESTMapper().KindFor(gvr); err == nil {
			gvk = k
		} else {
			gvk = f.constructGVKFromGVR(gvr)
		}
	} else {
		gvk = f.constructGVKFromGVR(gvr)
	}

	obj, err := f.client.Scheme().New(gvk)
	if err != nil {
		// Return a generic object if we can't create the specific type
		return &unstructured.Unstructured{}
	}

	return obj
}

// constructGVKFromGVR constructs a GVK from a GVR using simple conventions
func (f *sharedInformerFactory) constructGVKFromGVR(gvr schema.GroupVersionResource) schema.GroupVersionKind {
	// Simple singularization: remove 's' suffix
	kind := gvr.Resource
	if len(kind) > 1 && kind[len(kind)-1] == 's' {
		kind = kind[:len(kind)-1]
	}

	// Capitalize first letter
	if len(kind) > 0 {
		kind = strings.ToUpper(kind[:1]) + kind[1:]
	}

	return schema.GroupVersionKind{
		Group:   gvr.Group,
		Version: gvr.Version,
		Kind:    kind,
	}
}

func (f *sharedInformerFactory) getListObjectForGVR(gvr schema.GroupVersionResource) (client.ObjectList, error) {
	// Convert GVR to GVK first
	var gvk schema.GroupVersionKind

	if f.client.RESTMapper() != nil {
		if k, err := f.client.RESTMapper().KindFor(gvr); err == nil {
			gvk = k
		} else {
			gvk = f.constructGVKFromGVR(gvr)
		}
	} else {
		gvk = f.constructGVKFromGVR(gvr)
	}

	// Construct list GVK
	listGVK := schema.GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind + "List",
	}

	listObj, err := f.client.Scheme().New(listGVK)
	if err != nil {
		return nil, fmt.Errorf("failed to create list object for %s: %w", listGVK, err)
	}

	clientListObj, ok := listObj.(client.ObjectList)
	if !ok {
		return nil, fmt.Errorf("object %T does not implement client.ObjectList", listObj)
	}

	return clientListObj, nil
}

// parseLabelSelector parses a label selector string into a map.
// This is a simplified implementation for basic selectors like "key=value,key2=value2".
func parseLabelSelector(selector string) map[string]string {
	labels := make(map[string]string)

	if selector == "" {
		return labels
	}

	pairs := strings.Split(selector, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			labels[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}

	return labels
}

// parseFieldSelector parses a field selector string into a map.
// This is a simplified implementation for basic selectors like "metadata.name=value".
func parseFieldSelector(selector string) map[string]string {
	fields := make(map[string]string)

	if selector == "" {
		return fields
	}

	pairs := strings.Split(selector, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			fields[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}

	return fields
}
