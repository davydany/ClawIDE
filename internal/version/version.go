package version

import "fmt"

// Build-time variables injected via -ldflags -X.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// Info holds structured version information.
type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

// Get returns the current version information.
func Get() Info {
	return Info{
		Version: Version,
		Commit:  Commit,
		Date:    Date,
	}
}

// String returns a human-readable version string.
func String() string {
	return fmt.Sprintf("ClawIDE %s (commit: %s, built: %s)", Version, Commit, Date)
}
