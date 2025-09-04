package version

import (
	"fmt"
	"runtime"
)

// These variables are set at build time via ldflags
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
	BuiltBy = "unknown"
	Arch    = runtime.GOARCH
	Os      = runtime.GOOS
)

// Info represents the version information
type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	BuiltBy string `json:"builtBy"`
	Arch    string `json:"arch"`
	Os      string `json:"os"`
}

// GetVersionInfo returns the version information
func GetVersionInfo() Info {
	return Info{
		Version: Version,
		Commit:  Commit,
		Date:    Date,
		BuiltBy: BuiltBy,
		Arch:    Arch,
		Os:      Os,
	}
}

// String returns a formatted version string
func (i Info) String() string {
	return fmt.Sprintf("cli-gen version %s (%s) built on %s by %s for %s/%s",
		i.Version, i.Commit[:min(len(i.Commit), 8)], i.Date, i.BuiltBy, i.Os, i.Arch)
}

// Short returns a short version string
func (i Info) Short() string {
	return fmt.Sprintf("cli-gen %s", i.Version)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
