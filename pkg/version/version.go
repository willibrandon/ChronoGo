package version

import (
	"fmt"
	"runtime"
)

// These variables are populated by the build process
var (
	// Version is the version of the build
	Version = "dev"
	// BuildTime is the time when the build was created
	BuildTime = "unknown"
)

// GetVersionInfo returns a formatted string with version information
func GetVersionInfo() string {
	return fmt.Sprintf("ChronoGo v%s (built: %s, %s/%s)",
		Version,
		BuildTime,
		runtime.GOOS,
		runtime.GOARCH,
	)
}

// GetVersion returns just the version number
func GetVersion() string {
	return Version
}

// GetBuildTime returns the build timestamp
func GetBuildTime() string {
	return BuildTime
}
