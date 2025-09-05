// Package flags provides reusable pflag.FlagSet implementations for common CLI patterns.
// These flag sets can be used by CLI applications to provide consistent kubectl-compatible
// command-line interfaces.
package flags

import (
	"github.com/spf13/pflag"
)

// OutputFlags returns flags for output formatting.
func OutputFlags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("output", pflag.ContinueOnError)

	flags.StringP("output", "o", "table", "Output format. One of: table|json|yaml|name|wide")
	flags.Bool("no-headers", false, "Don't print headers (default print headers)")
	flags.Bool("show-labels", false, "When printing, show all labels as the last column (default hide labels column)")
	flags.Bool("sort-by", false, "Sort list of resources using specified field")

	return flags
}

// SelectorFlags returns flags for resource selection.
func SelectorFlags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("selector", pflag.ContinueOnError)

	flags.StringP("selector", "l", "", "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	flags.String("field-selector", "", "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2)")
	flags.Bool("all-namespaces", false, "If present, list the requested object(s) across all namespaces")
	flags.StringP("namespace", "n", "", "If present, the namespace scope for this CLI request")

	return flags
}

// CommonFlags returns flags common to most operations.
func CommonFlags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("common", pflag.ContinueOnError)

	flags.Bool("dry-run", false, "If true, only print the object that would be sent, without sending it")
	flags.String("field-manager", "", "Name of the manager used to track field ownership")
	flags.Bool("force", false, "If true, immediately remove resources from API and bypass graceful deletion")

	return flags
}

// WatchFlags returns flags for watch operations.
func WatchFlags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("watch", pflag.ContinueOnError)

	flags.BoolP("watch", "w", false, "After listing/getting the requested object, watch for changes")
	flags.Bool("watch-only", false, "Watch for changes to the requested object(s), without listing/getting first")

	return flags
}

// CreateFlags returns flags specific to create operations.
func CreateFlags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("create", pflag.ContinueOnError)

	flags.Bool("save-config", false, "If true, the configuration of current object will be saved in its annotation")
	flags.String("filename", "", "Filename to use to create the resource")
	flags.StringSlice("filenames", []string{}, "Filenames to use to create the resources")
	flags.BoolP("recursive", "R", false, "Process the directory used in -f, --filename recursively")

	return flags
}

// DeleteFlags returns flags specific to delete operations.
func DeleteFlags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("delete", pflag.ContinueOnError)

	flags.Bool("cascade", true, "If true, cascade the deletion of the resources managed by this resource")
	flags.Int64("grace-period", -1, "Period of time in seconds given to the resource to terminate gracefully")
	flags.Bool("ignore-not-found", false, "If the requested object does not exist the command will return exit code 0")
	flags.Bool("now", false, "If true, resources are signaled for immediate shutdown")
	flags.String("timeout", "0s", "The length of time to wait before giving up on a delete")
	flags.Bool("wait", true, "If true, wait for resources to be gone before returning")

	return flags
}

// GetFlags returns flags specific to get operations.
func GetFlags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("get", pflag.ContinueOnError)

	flags.BoolP("watch", "w", false, "After listing/getting the requested object, watch for changes")
	flags.Bool("watch-only", false, "Watch for changes to the requested object(s), without listing/getting first")
	flags.String("sort-by", "", "Sort list of resources using specified field")
	flags.Bool("ignore-not-found", false, "If the requested object does not exist the command will return exit code 0")

	return flags
}

// ApplyFlags returns flags specific to apply operations.
func ApplyFlags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("apply", pflag.ContinueOnError)

	flags.Bool("server-side", false, "If true, apply runs in the server instead of the client")
	flags.Bool("force-conflicts", false, "If true, server-side apply will force the changes against conflicts")
	flags.String("filename", "", "Filename to use to apply the resource")
	flags.StringSlice("filenames", []string{}, "Filenames to use to apply the resources")
	flags.BoolP("recursive", "R", false, "Process the directory used in -f, --filename recursively")
	flags.Bool("prune", false, "Automatically delete resource objects that do not appear in the configs")

	return flags
}
