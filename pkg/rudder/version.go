package rudder

import "fmt"

// Version is the current release version. Overridden at build time via ldflags.
var Version = "0.0.1-dev"

// Commit is the git commit hash at build time.
var Commit = "unknown"

// BuildDate is the UTC build timestamp.
var BuildDate = "unknown"

// BuildInfo returns a formatted multi-line build information string.
func BuildInfo() string {
	return fmt.Sprintf("Version:    %s\nCommit:     %s\nBuilt:      %s", Version, Commit, BuildDate)
}
