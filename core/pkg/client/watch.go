package client

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
)

// watchClient implements the WithWatch interface.
type watchClient struct {
	client *client
}

// NewWatchClient creates a new client that supports watch operations.
func NewWatchClient(c Client) (WithWatch, error) {
	clientImpl, ok := c.(*client)
	if !ok {
		return nil, fmt.Errorf("client does not support watch operations")
	}

	return &watchClient{client: clientImpl}, nil
}

// Implement Client interface methods by delegating to embedded client
func (w *watchClient) Get(ctx context.Context, key ObjectKey, obj Object, opts ...GetOption) error {
	return w.client.Get(ctx, key, obj, opts...)
}

func (w *watchClient) List(ctx context.Context, list ObjectList, opts ...ListOption) error {
	return w.client.List(ctx, list, opts...)
}

func (w *watchClient) Create(ctx context.Context, obj Object, opts ...CreateOption) error {
	return w.client.Create(ctx, obj, opts...)
}

func (w *watchClient) Delete(ctx context.Context, obj Object, opts ...DeleteOption) error {
	return w.client.Delete(ctx, obj, opts...)
}

func (w *watchClient) Update(ctx context.Context, obj Object, opts ...UpdateOption) error {
	return w.client.Update(ctx, obj, opts...)
}

func (w *watchClient) Patch(ctx context.Context, obj Object, patch Patch, opts ...PatchOption) error {
	return w.client.Patch(ctx, obj, patch, opts...)
}

func (w *watchClient) Status() StatusWriter {
	return w.client.Status()
}

func (w *watchClient) Scheme() *runtime.Scheme {
	return w.client.Scheme()
}

func (w *watchClient) RESTMapper() meta.RESTMapper {
	return w.client.RESTMapper()
}

// Watch watches objects of the given type.
func (w *watchClient) Watch(ctx context.Context, obj ObjectList, opts ...WatchOption) (watch.Interface, error) {
	options := &WatchOptions{}
	for _, opt := range opts {
		opt.ApplyToWatch(options)
	}

	gvk, err := w.client.getGVKForObjectList(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to get GVK for object list: %w", err)
	}

	gvr, err := w.client.registry.GetGVRForGVK(gvk)
	if err != nil {
		return nil, fmt.Errorf("failed to get GVR for GVK %s: %w", gvk, err)
	}

	storageKey := w.client.buildListStorageKey(gvr, options.Namespace)

	// Build storage list options for watching
	listOpts := storage.ListOptions{
		ResourceVersion: "0",
		Recursive:       true,
	}

	if options.Raw != nil {
		if options.Raw.ResourceVersion != "" {
			listOpts.ResourceVersion = options.Raw.ResourceVersion
		}
		// Note: storage.ListOptions doesn't have TimeoutSeconds field
		// Timeout would be handled by the context or storage implementation
	}

	// Apply label and field selectors if specified
	// Note: For now, server-side filtering is not implemented at the storage level
	// We rely on client-side filtering in the filteringWatcher below
	hasLabelFilters := len(options.LabelSelector.MatchLabels) > 0 || len(options.LabelSelector.MatchExpressions) > 0
	hasFieldFilters := options.FieldSelector != ""

	// Storage-level filtering would be implemented here in the future
	_ = hasLabelFilters
	_ = hasFieldFilters

	watcher, err := w.client.storage.Watch(ctx, storageKey, listOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to start watch: %w", err)
	}

	// If we have selectors, wrap the watcher with filtering
	if w.needsFiltering(options) {
		return w.newFilteringWatcher(watcher, options), nil
	}

	return watcher, nil
}

// needsFiltering determines if we need to apply client-side filtering.
func (w *watchClient) needsFiltering(options *WatchOptions) bool {
	return len(options.LabelSelector.MatchLabels) > 0 ||
		len(options.LabelSelector.MatchExpressions) > 0 ||
		options.FieldSelector != ""
}

// newFilteringWatcher creates a watcher that applies client-side filtering.
func (w *watchClient) newFilteringWatcher(watcher watch.Interface, options *WatchOptions) watch.Interface {
	return &filteringWatcher{
		watcher: watcher,
		options: options,
	}
}

// filteringWatcher wraps a watch.Interface and applies client-side filtering.
type filteringWatcher struct {
	watcher watch.Interface
	options *WatchOptions
}

// Stop stops the underlying watcher.
func (fw *filteringWatcher) Stop() {
	fw.watcher.Stop()
}

// ResultChan returns the filtered result channel.
func (fw *filteringWatcher) ResultChan() <-chan watch.Event {
	input := fw.watcher.ResultChan()
	output := make(chan watch.Event)

	go func() {
		defer close(output)
		for event := range input {
			if fw.shouldIncludeEvent(event) {
				output <- event
			}
		}
	}()

	return output
}

// shouldIncludeEvent determines if an event should be included based on the filtering options.
func (fw *filteringWatcher) shouldIncludeEvent(event watch.Event) bool {
	obj, ok := event.Object.(Object)
	if !ok {
		// If we can't cast to Object, include it (might be an error event)
		return true
	}

	// Apply label selector filtering
	if len(fw.options.LabelSelector.MatchLabels) > 0 {
		objLabels := obj.GetLabels()
		if objLabels == nil {
			return false
		}

		for key, value := range fw.options.LabelSelector.MatchLabels {
			if objLabels[key] != value {
				return false
			}
		}
	}

	// Apply match expressions (simplified implementation)
	if len(fw.options.LabelSelector.MatchExpressions) > 0 {
		objLabels := obj.GetLabels()
		if objLabels == nil {
			return false
		}

		for _, expr := range fw.options.LabelSelector.MatchExpressions {
			switch expr.Operator {
			case "In":
				value, exists := objLabels[expr.Key]
				if !exists {
					return false
				}
				found := false
				for _, v := range expr.Values {
					if v == value {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			case "NotIn":
				value, exists := objLabels[expr.Key]
				if exists {
					for _, v := range expr.Values {
						if v == value {
							return false
						}
					}
				}
			case "Exists":
				if _, exists := objLabels[expr.Key]; !exists {
					return false
				}
			case "DoesNotExist":
				if _, exists := objLabels[expr.Key]; exists {
					return false
				}
			}
		}
	}

	// Apply field selector filtering (simplified implementation)
	if fw.options.FieldSelector != "" {
		// For now, we only support basic field selectors like "metadata.name=value"
		// A full implementation would parse and evaluate complex field selectors
		// This is a placeholder for future field selector implementation
		return fw.evaluateFieldSelector(obj)
	}

	return true
}

// evaluateFieldSelector evaluates basic field selectors.
// This is a simplified implementation for basic field selector patterns.
func (fw *filteringWatcher) evaluateFieldSelector(obj Object) bool {
	// For now, just return true - in a full implementation this would
	// parse and evaluate field selectors like "metadata.name=value"
	// TODO: Implement proper field selector parsing and evaluation
	return true
}
