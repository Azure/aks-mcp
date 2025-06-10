package version

import (
	"fmt"
	"runtime"
)

// Version information
var (
	// GitVersion is the git tag version
	GitVersion = "1"
	// BuildMetadata is extra build time data
	BuildMetadata = ""
	// GitCommit is the git sha1
	GitCommit = ""
	// GitTreeState describes the state of the git tree
	GitTreeState = ""
)

// GetVersion returns the version string
func GetVersion() string {
	var version string
	if BuildMetadata != "" {
		version = fmt.Sprintf("%s+%s", GitVersion, BuildMetadata)
	} else {
		version = GitVersion
	}
	return version
}

// GetVersionInfo returns a map with all version information
func GetVersionInfo() map[string]string {
	return map[string]string{
		"version":      GetVersion(),
		"gitCommit":    GitCommit,
		"gitTreeState": GitTreeState,
		"goVersion":    runtime.Version(),
		"platform":     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}
