package version

import (
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
)

// Version string is set during build via ldflags
var Version string

var (
	once          sync.Once
	versionString string
)

// For testing
var buildInfoFunc = debug.ReadBuildInfo

// Get returns the version string, determining it on first call
func Get() string {
	once.Do(func() {
		versionString = determineVersion()
	})
	return versionString
}

// determineVersion returns the version string, falling back to build info for development builds
func determineVersion() string {
	// If version was set by ldflags (during release builds), use it
	if Version != "" {
		return Version
	}

	// Try to get version from build info (works for go install)
	if info, ok := buildInfoFunc(); ok {
		// First check the main module version
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return strings.TrimPrefix(info.Main.Version, "v")
		}

		// Check for vcs information
		var revision, time string
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				revision = setting.Value[:7] // Short SHA
			case "vcs.time":
				time = setting.Value
			}
		}
		if revision != "" && time != "" {
			return fmt.Sprintf("dev-%s-%s", revision, time)
		}
	}

	return "0.0.0-dev"
}
