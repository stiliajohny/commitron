package version

import "fmt"

// These variables will be set at build time using ldflags
var (
	Version   = "dev"
	CommitSHA = "unknown"
	BuildDate = "unknown"
)

// GetVersion returns the full version string
func GetVersion() string {
	return fmt.Sprintf("v%s", Version)
}

// GetFullVersion returns detailed version information
func GetFullVersion() string {
	return fmt.Sprintf("v%s (commit: %s, built: %s)", Version, CommitSHA, BuildDate)
}

// IsDev returns true if this is a development build
func IsDev() bool {
	return Version == "dev"
}
