package flags

import (
	"time"

	"github.com/spf13/pflag"
)

// OutputConfig contains output formatting configuration
type OutputConfig struct {
	Output     string
	NoHeaders  bool
	ShowLabels bool
	SortBy     string
}

// NewOutputConfig creates a new OutputConfig with defaults
func NewOutputConfig() *OutputConfig {
	return &OutputConfig{
		Output:     DefaultOutputFormat,
		NoHeaders:  false,
		ShowLabels: false,
		SortBy:     "",
	}
}

// SelectorConfig contains resource selection configuration
type SelectorConfig struct {
	Selector      string
	FieldSelector string
	AllNamespaces bool
	Namespace     string
}

// NewSelectorConfig creates a new SelectorConfig with defaults
func NewSelectorConfig() *SelectorConfig {
	return &SelectorConfig{
		Selector:      "",
		FieldSelector: "",
		AllNamespaces: false,
		Namespace:     "",
	}
}

// CommonConfig contains common operation configuration
type CommonConfig struct {
	DryRun       bool
	FieldManager string
	Force        bool
}

// NewCommonConfig creates a new CommonConfig with defaults
func NewCommonConfig() *CommonConfig {
	return &CommonConfig{
		DryRun:       false,
		FieldManager: "",
		Force:        false,
	}
}

// WatchConfig contains watch operation configuration
type WatchConfig struct {
	Watch     bool
	WatchOnly bool
}

// NewWatchConfig creates a new WatchConfig with defaults
func NewWatchConfig() *WatchConfig {
	return &WatchConfig{
		Watch:     false,
		WatchOnly: false,
	}
}

// CreateConfig contains create operation configuration
type CreateConfig struct {
	SaveConfig bool
	Filename   string
	Filenames  []string
	Recursive  bool
}

// NewCreateConfig creates a new CreateConfig with defaults
func NewCreateConfig() *CreateConfig {
	return &CreateConfig{
		SaveConfig: false,
		Filename:   "",
		Filenames:  []string{},
		Recursive:  false,
	}
}

// DeleteConfig contains delete operation configuration
type DeleteConfig struct {
	Cascade        bool
	GracePeriod    int64
	IgnoreNotFound bool
	Now            bool
	Timeout        time.Duration
	Wait           bool
}

// NewDeleteConfig creates a new DeleteConfig with defaults
func NewDeleteConfig() *DeleteConfig {
	return &DeleteConfig{
		Cascade:        true,
		GracePeriod:    -1,
		IgnoreNotFound: false,
		Now:            false,
		Timeout:        0,
		Wait:           true,
	}
}

// GetConfig contains get operation configuration
type GetConfig struct {
	Watch          bool
	WatchOnly      bool
	SortBy         string
	IgnoreNotFound bool
}

// NewGetConfig creates a new GetConfig with defaults
func NewGetConfig() *GetConfig {
	return &GetConfig{
		Watch:          false,
		WatchOnly:      false,
		SortBy:         "",
		IgnoreNotFound: false,
	}
}

// ApplyConfig contains apply operation configuration
type ApplyConfig struct {
	ServerSide     bool
	ForceConflicts bool
	Filename       string
	Filenames      []string
	Recursive      bool
	Prune          bool
}

// NewApplyConfig creates a new ApplyConfig with defaults
func NewApplyConfig() *ApplyConfig {
	return &ApplyConfig{
		ServerSide:     false,
		ForceConflicts: false,
		Filename:       "",
		Filenames:      []string{},
		Recursive:      false,
		Prune:          false,
	}
}

// OutputFlagsVar returns flags for output formatting bound to a config struct
func OutputFlagsVar(config *OutputConfig) *pflag.FlagSet {
	flags := pflag.NewFlagSet("output", pflag.ContinueOnError)

	flags.StringVarP(&config.Output, FlagOutput, FlagOutputShort, config.Output, "Output format. One of: table|json|yaml|name|wide")
	flags.BoolVar(&config.NoHeaders, FlagNoHeaders, config.NoHeaders, "Don't print headers (default print headers)")
	flags.BoolVar(&config.ShowLabels, FlagShowLabels, config.ShowLabels, "When printing, show all labels as the last column (default hide labels column)")
	flags.StringVar(&config.SortBy, FlagSortBy, config.SortBy, "Sort list of resources using specified field")

	return flags
}

// SelectorFlagsVar returns flags for resource selection bound to a config struct
func SelectorFlagsVar(config *SelectorConfig) *pflag.FlagSet {
	flags := pflag.NewFlagSet("selector", pflag.ContinueOnError)

	flags.StringVarP(&config.Selector, FlagSelector, FlagSelectorShort, config.Selector, "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	flags.StringVar(&config.FieldSelector, FlagFieldSelector, config.FieldSelector, "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2)")
	flags.BoolVar(&config.AllNamespaces, FlagAllNamespaces, config.AllNamespaces, "If present, list the requested object(s) across all namespaces")
	flags.StringVarP(&config.Namespace, FlagNamespace, FlagNamespaceShort, config.Namespace, "If present, the namespace scope for this CLI request")

	return flags
}

// CommonFlagsVar returns flags common to most operations bound to a config struct
func CommonFlagsVar(config *CommonConfig) *pflag.FlagSet {
	flags := pflag.NewFlagSet("common", pflag.ContinueOnError)

	flags.BoolVar(&config.DryRun, FlagDryRun, config.DryRun, "If true, only print the object that would be sent, without sending it")
	flags.StringVar(&config.FieldManager, FlagFieldManager, config.FieldManager, "Name of the manager used to track field ownership")
	flags.BoolVar(&config.Force, FlagForce, config.Force, "If true, immediately remove resources from API and bypass graceful deletion")

	return flags
}

// WatchFlagsVar returns flags for watch operations bound to a config struct
func WatchFlagsVar(config *WatchConfig) *pflag.FlagSet {
	flags := pflag.NewFlagSet("watch", pflag.ContinueOnError)

	flags.BoolVarP(&config.Watch, FlagWatch, FlagWatchShort, config.Watch, "After listing/getting the requested object, watch for changes")
	flags.BoolVar(&config.WatchOnly, FlagWatchOnly, config.WatchOnly, "Watch for changes to the requested object(s), without listing/getting first")

	return flags
}

// CreateFlagsVar returns flags specific to create operations bound to a config struct
func CreateFlagsVar(config *CreateConfig) *pflag.FlagSet {
	flags := pflag.NewFlagSet("create", pflag.ContinueOnError)

	flags.BoolVar(&config.SaveConfig, FlagSaveConfig, config.SaveConfig, "If true, the configuration of current object will be saved in its annotation")
	flags.StringVarP(&config.Filename, FlagFilename, FlagFilenameShort, config.Filename, "Filename to use to create the resource")
	flags.StringSliceVar(&config.Filenames, FlagFilenames, config.Filenames, "Filenames to use to create the resources")
	flags.BoolVarP(&config.Recursive, FlagRecursive, FlagRecursiveShort, config.Recursive, "Process the directory used in -f, --filename recursively")

	return flags
}

// DeleteFlagsVar returns flags specific to delete operations bound to a config struct
func DeleteFlagsVar(config *DeleteConfig) *pflag.FlagSet {
	flags := pflag.NewFlagSet("delete", pflag.ContinueOnError)

	flags.BoolVar(&config.Cascade, FlagCascade, config.Cascade, "If true, cascade the deletion of the resources managed by this resource")
	flags.Int64Var(&config.GracePeriod, FlagGracePeriod, config.GracePeriod, "Period of time in seconds given to the resource to terminate gracefully")
	flags.BoolVar(&config.IgnoreNotFound, FlagIgnoreNotFound, config.IgnoreNotFound, "If the requested object does not exist the command will return exit code 0")
	flags.BoolVar(&config.Now, FlagNow, config.Now, "If true, resources are signaled for immediate shutdown")
	flags.DurationVar(&config.Timeout, FlagTimeout, config.Timeout, "The length of time to wait before giving up on a delete")
	flags.BoolVar(&config.Wait, FlagWait, config.Wait, "If true, wait for resources to be gone before returning")

	return flags
}

// GetFlagsVar returns flags specific to get operations bound to a config struct
func GetFlagsVar(config *GetConfig) *pflag.FlagSet {
	flags := pflag.NewFlagSet("get", pflag.ContinueOnError)

	flags.BoolVarP(&config.Watch, FlagWatch, FlagWatchShort, config.Watch, "After listing/getting the requested object, watch for changes")
	flags.BoolVar(&config.WatchOnly, FlagWatchOnly, config.WatchOnly, "Watch for changes to the requested object(s), without listing/getting first")
	flags.StringVar(&config.SortBy, FlagSortBy, config.SortBy, "Sort list of resources using specified field")
	flags.BoolVar(&config.IgnoreNotFound, FlagIgnoreNotFound, config.IgnoreNotFound, "If the requested object does not exist the command will return exit code 0")

	return flags
}

// ApplyFlagsVar returns flags specific to apply operations bound to a config struct
func ApplyFlagsVar(config *ApplyConfig) *pflag.FlagSet {
	flags := pflag.NewFlagSet("apply", pflag.ContinueOnError)

	flags.BoolVar(&config.ServerSide, FlagServerSide, config.ServerSide, "If true, apply runs in the server instead of the client")
	flags.BoolVar(&config.ForceConflicts, FlagForceConflicts, config.ForceConflicts, "If true, server-side apply will force the changes against conflicts")
	flags.StringVarP(&config.Filename, FlagFilename, FlagFilenameShort, config.Filename, "Filename to use to apply the resource")
	flags.StringSliceVar(&config.Filenames, FlagFilenames, config.Filenames, "Filenames to use to apply the resources")
	flags.BoolVarP(&config.Recursive, FlagRecursive, FlagRecursiveShort, config.Recursive, "Process the directory used in -f, --filename recursively")
	flags.BoolVar(&config.Prune, FlagPrune, config.Prune, "Automatically delete resource objects that do not appear in the configs")

	return flags
}
