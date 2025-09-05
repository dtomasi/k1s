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

	flags.StringP(FlagOutput, FlagOutputShort, DefaultOutputFormat, "Output format. One of: table|json|yaml|name|wide")
	flags.Bool(FlagNoHeaders, false, "Don't print headers (default print headers)")
	flags.Bool(FlagShowLabels, false, "When printing, show all labels as the last column (default hide labels column)")
	flags.String(FlagSortBy, "", "Sort list of resources using specified field")

	return flags
}

// SelectorFlags returns flags for resource selection.
func SelectorFlags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("selector", pflag.ContinueOnError)

	flags.StringP(FlagSelector, FlagSelectorShort, "", "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	flags.String(FlagFieldSelector, "", "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2)")
	flags.Bool(FlagAllNamespaces, false, "If present, list the requested object(s) across all namespaces")
	flags.StringP(FlagNamespace, FlagNamespaceShort, "", "If present, the namespace scope for this CLI request")

	return flags
}

// CommonFlags returns flags common to most operations.
func CommonFlags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("common", pflag.ContinueOnError)

	flags.Bool(FlagDryRun, false, "If true, only print the object that would be sent, without sending it")
	flags.String(FlagFieldManager, "", "Name of the manager used to track field ownership")
	flags.Bool(FlagForce, false, "If true, immediately remove resources from API and bypass graceful deletion")

	return flags
}

// WatchFlags returns flags for watch operations.
func WatchFlags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("watch", pflag.ContinueOnError)

	flags.BoolP(FlagWatch, FlagWatchShort, false, "After listing/getting the requested object, watch for changes")
	flags.Bool(FlagWatchOnly, false, "Watch for changes to the requested object(s), without listing/getting first")

	return flags
}

// CreateFlags returns flags specific to create operations.
func CreateFlags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("create", pflag.ContinueOnError)

	flags.Bool(FlagSaveConfig, false, "If true, the configuration of current object will be saved in its annotation")
	flags.StringP(FlagFilename, FlagFilenameShort, "", "Filename to use to create the resource")
	flags.StringSlice(FlagFilenames, []string{}, "Filenames to use to create the resources")
	flags.BoolP(FlagRecursive, FlagRecursiveShort, false, "Process the directory used in -f, --filename recursively")

	return flags
}

// DeleteFlags returns flags specific to delete operations.
func DeleteFlags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("delete", pflag.ContinueOnError)

	flags.Bool(FlagCascade, true, "If true, cascade the deletion of the resources managed by this resource")
	flags.Int64(FlagGracePeriod, -1, "Period of time in seconds given to the resource to terminate gracefully")
	flags.Bool(FlagIgnoreNotFound, false, "If the requested object does not exist the command will return exit code 0")
	flags.Bool(FlagNow, false, "If true, resources are signaled for immediate shutdown")
	flags.String(FlagTimeout, DefaultTimeout, "The length of time to wait before giving up on a delete")
	flags.Bool(FlagWait, true, "If true, wait for resources to be gone before returning")

	return flags
}

// GetFlags returns flags specific to get operations.
func GetFlags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("get", pflag.ContinueOnError)

	flags.BoolP(FlagWatch, FlagWatchShort, false, "After listing/getting the requested object, watch for changes")
	flags.Bool(FlagWatchOnly, false, "Watch for changes to the requested object(s), without listing/getting first")
	flags.String(FlagSortBy, "", "Sort list of resources using specified field")
	flags.Bool(FlagIgnoreNotFound, false, "If the requested object does not exist the command will return exit code 0")

	return flags
}

// ApplyFlags returns flags specific to apply operations.
func ApplyFlags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("apply", pflag.ContinueOnError)

	flags.Bool(FlagServerSide, false, "If true, apply runs in the server instead of the client")
	flags.Bool(FlagForceConflicts, false, "If true, server-side apply will force the changes against conflicts")
	flags.StringP(FlagFilename, FlagFilenameShort, "", "Filename to use to apply the resource")
	flags.StringSlice(FlagFilenames, []string{}, "Filenames to use to apply the resources")
	flags.BoolP(FlagRecursive, FlagRecursiveShort, false, "Process the directory used in -f, --filename recursively")
	flags.Bool(FlagPrune, false, "Automatically delete resource objects that do not appear in the configs")

	return flags
}
