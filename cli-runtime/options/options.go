// Package options provides utilities for parsing command-line flags into
// structured options that can be used with handlers and other CLI components.
package options

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"

	"github.com/dtomasi/k1s/cli-runtime/handlers"
	"github.com/dtomasi/k1s/core/client"
)

// OutputOptions contains parsed output formatting options.
type OutputOptions struct {
	Format        string
	NoHeaders     bool
	ShowLabels    bool
	Wide          bool
	SortBy        string
	CustomColumns []string
}

// SelectorOptions contains parsed selector options.
type SelectorOptions struct {
	LabelSelector string
	FieldSelector string
	AllNamespaces bool
	Namespace     string
}

// CommonOptions contains parsed common options.
type CommonOptions struct {
	DryRun       bool
	FieldManager string
	Force        bool
}

// WatchOptions contains parsed watch options.
type WatchOptions struct {
	Watch     bool
	WatchOnly bool
}

// ParseOutputOptions parses output flags into OutputOptions.
func ParseOutputOptions(flags *pflag.FlagSet) (*OutputOptions, error) {
	if flags == nil {
		return &OutputOptions{Format: "table"}, nil
	}

	options := &OutputOptions{}

	if format, err := flags.GetString("output"); err == nil {
		options.Format = format
	} else {
		options.Format = "table"
	}

	if noHeaders, err := flags.GetBool("no-headers"); err == nil {
		options.NoHeaders = noHeaders
	}

	if showLabels, err := flags.GetBool("show-labels"); err == nil {
		options.ShowLabels = showLabels
	}

	if sortBy, err := flags.GetString("sort-by"); err == nil {
		options.SortBy = sortBy
	}

	// Handle wide format
	if options.Format == "wide" {
		options.Wide = true
		options.Format = "table"
	}

	return options, nil
}

// ParseSelectorOptions parses selector flags into SelectorOptions.
func ParseSelectorOptions(flags *pflag.FlagSet) (*SelectorOptions, error) {
	if flags == nil {
		return &SelectorOptions{}, nil
	}

	options := &SelectorOptions{}

	if selector, err := flags.GetString("selector"); err == nil {
		options.LabelSelector = selector
	}

	if fieldSelector, err := flags.GetString("field-selector"); err == nil {
		options.FieldSelector = fieldSelector
	}

	if allNamespaces, err := flags.GetBool("all-namespaces"); err == nil {
		options.AllNamespaces = allNamespaces
	}

	if namespace, err := flags.GetString("namespace"); err == nil {
		options.Namespace = namespace
	}

	return options, nil
}

// ParseCommonOptions parses common flags into CommonOptions.
func ParseCommonOptions(flags *pflag.FlagSet) (*CommonOptions, error) {
	if flags == nil {
		return &CommonOptions{}, nil
	}

	options := &CommonOptions{}

	if dryRun, err := flags.GetBool("dry-run"); err == nil {
		options.DryRun = dryRun
	}

	if fieldManager, err := flags.GetString("field-manager"); err == nil {
		options.FieldManager = fieldManager
	}

	if force, err := flags.GetBool("force"); err == nil {
		options.Force = force
	}

	return options, nil
}

// ParseWatchOptions parses watch flags into WatchOptions.
func ParseWatchOptions(flags *pflag.FlagSet) (*WatchOptions, error) {
	if flags == nil {
		return &WatchOptions{}, nil
	}

	options := &WatchOptions{}

	if watch, err := flags.GetBool("watch"); err == nil {
		options.Watch = watch
	}

	if watchOnly, err := flags.GetBool("watch-only"); err == nil {
		options.WatchOnly = watchOnly
	}

	return options, nil
}

// ToHandlerOutputOptions converts OutputOptions to handlers.OutputOptions.
func (o *OutputOptions) ToHandlerOutputOptions() *handlers.OutputOptions {
	return &handlers.OutputOptions{
		Format:        o.Format,
		NoHeaders:     o.NoHeaders,
		ShowLabels:    o.ShowLabels,
		Wide:          o.Wide,
		CustomColumns: o.CustomColumns,
	}
}

// ToListOptions converts SelectorOptions to client.ListOption slice.
func (s *SelectorOptions) ToListOptions() []client.ListOption {
	var opts []client.ListOption

	// Add label selector
	if s.LabelSelector != "" {
		labelMap, err := parseLabelSelector(s.LabelSelector)
		if err == nil && len(labelMap) > 0 {
			opts = append(opts, client.MatchingLabels(labelMap))
		}
	}

	// Add field selector
	if s.FieldSelector != "" {
		fieldMap, err := parseFieldSelector(s.FieldSelector)
		if err == nil && len(fieldMap) > 0 {
			opts = append(opts, client.MatchingFields(fieldMap))
		}
	}

	// Add namespace selector
	if s.Namespace != "" && !s.AllNamespaces {
		opts = append(opts, client.InNamespace(s.Namespace))
	}

	return opts
}

// ToCreateOptions converts CommonOptions to client.CreateOption slice.
func (c *CommonOptions) ToCreateOptions() []client.CreateOption {
	var opts []client.CreateOption

	// For now, return empty slice - options will be added as needed
	// This is a placeholder for future expansion

	return opts
}

// ToDeleteOptions converts CommonOptions to client.DeleteOption slice.
func (c *CommonOptions) ToDeleteOptions() []client.DeleteOption {
	var opts []client.DeleteOption

	// For now, return empty slice - options will be added as needed
	// This is a placeholder for future expansion

	return opts
}

// parseLabelSelector parses a label selector string into a map.
func parseLabelSelector(selector string) (map[string]string, error) {
	if selector == "" {
		return nil, nil
	}

	labelMap := make(map[string]string)

	// Split by comma
	pairs := strings.Split(selector, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		// Split by equals
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid label selector format: %s", pair)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		labelMap[key] = value
	}

	return labelMap, nil
}

// parseFieldSelector parses a field selector string into a map.
func parseFieldSelector(selector string) (map[string]string, error) {
	if selector == "" {
		return nil, nil
	}

	fieldMap := make(map[string]string)

	// Split by comma
	pairs := strings.Split(selector, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		// Split by equals
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid field selector format: %s", pair)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		fieldMap[key] = value
	}

	return fieldMap, nil
}
