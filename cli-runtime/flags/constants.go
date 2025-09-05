package flags

// Flag name constants for kubectl-compatible CLI flags.
// These constants ensure consistency and prevent typos in flag usage.

// Output formatting flags
const (
	// FlagOutput specifies the output format (table, json, yaml, name, wide)
	FlagOutput = "output"
	// FlagOutputShort is the short form of output flag
	FlagOutputShort = "o"
	// FlagNoHeaders disables header printing in table output
	FlagNoHeaders = "no-headers"
	// FlagShowLabels shows all labels as the last column
	FlagShowLabels = "show-labels"
	// FlagSortBy sorts list of resources using specified field
	FlagSortBy = "sort-by"
)

// Resource selection flags
const (
	// FlagSelector specifies label selector for filtering
	FlagSelector = "selector"
	// FlagSelectorShort is the short form of selector flag
	FlagSelectorShort = "l"
	// FlagFieldSelector specifies field selector for filtering
	FlagFieldSelector = "field-selector"
	// FlagAllNamespaces lists resources across all namespaces
	FlagAllNamespaces = "all-namespaces"
	// FlagNamespace specifies the namespace scope
	FlagNamespace = "namespace"
	// FlagNamespaceShort is the short form of namespace flag
	FlagNamespaceShort = "n"
)

// Common operation flags
const (
	// FlagDryRun only prints what would be sent without sending
	FlagDryRun = "dry-run"
	// FlagFieldManager specifies the field manager name
	FlagFieldManager = "field-manager"
	// FlagForce immediately removes resources bypassing graceful deletion
	FlagForce = "force"
)

// Watch operation flags
const (
	// FlagWatch watches for changes after listing/getting
	FlagWatch = "watch"
	// FlagWatchShort is the short form of watch flag
	FlagWatchShort = "w"
	// FlagWatchOnly watches for changes without listing/getting first
	FlagWatchOnly = "watch-only"
)

// Create operation flags
const (
	// FlagSaveConfig saves configuration in object annotation
	FlagSaveConfig = "save-config"
	// FlagFilename specifies a single filename
	FlagFilename = "filename"
	// FlagFilenameShort is the short form of filename flag
	FlagFilenameShort = "f"
	// FlagFilenames specifies multiple filenames
	FlagFilenames = "filenames"
	// FlagRecursive processes directories recursively
	FlagRecursive = "recursive"
	// FlagRecursiveShort is the short form of recursive flag
	FlagRecursiveShort = "R"
)

// Delete operation flags
const (
	// FlagCascade cascades deletion of managed resources
	FlagCascade = "cascade"
	// FlagGracePeriod specifies termination grace period
	FlagGracePeriod = "grace-period"
	// FlagIgnoreNotFound returns success even if object doesn't exist
	FlagIgnoreNotFound = "ignore-not-found"
	// FlagNow signals immediate shutdown
	FlagNow = "now"
	// FlagTimeout specifies wait timeout for deletion
	FlagTimeout = "timeout"
	// FlagWait waits for resources to be gone before returning
	FlagWait = "wait"
)

// Apply operation flags
const (
	// FlagServerSide runs apply on server instead of client
	FlagServerSide = "server-side"
	// FlagForceConflicts forces changes against conflicts in server-side apply
	FlagForceConflicts = "force-conflicts"
	// FlagPrune automatically deletes resources not in configs
	FlagPrune = "prune"
)

// Default values for commonly used flags
const (
	// DefaultOutputFormat is the default output format
	DefaultOutputFormat = "table"
	// DefaultNamespace is the default namespace when none specified
	DefaultNamespace = "default"
	// DefaultTimeout is the default timeout value
	DefaultTimeout = "0s"
)
