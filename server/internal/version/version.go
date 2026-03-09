// Package version provides version information for AI-Trace.
package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the current version of AI-Trace.
	// This is set during build using ldflags.
	Version = "0.2.0"

	// GitCommit is the git commit hash.
	// This is set during build using ldflags.
	GitCommit = "unknown"

	// BuildTime is the build timestamp.
	// This is set during build using ldflags.
	BuildTime = "unknown"
)

// Info holds complete version information.
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// Get returns the version information.
func Get() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns a formatted version string.
func (i Info) String() string {
	return fmt.Sprintf("AI-Trace %s (%s) built at %s with %s for %s",
		i.Version, i.GitCommit, i.BuildTime, i.GoVersion, i.Platform)
}

// Short returns a short version string.
func Short() string {
	return Version
}

// Full returns the full version information as a formatted string.
func Full() string {
	return Get().String()
}
