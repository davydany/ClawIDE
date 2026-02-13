package version

import (
	"fmt"
	"strconv"
	"strings"
)

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

// IsDevVersion returns true when the binary was built without version tags.
func IsDevVersion() bool {
	return Version == "dev"
}

// CompareVersions compares two semver strings (with or without "v" prefix).
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func CompareVersions(a, b string) int {
	aParts := parseSemver(a)
	bParts := parseSemver(b)

	for i := 0; i < 3; i++ {
		if aParts[i] < bParts[i] {
			return -1
		}
		if aParts[i] > bParts[i] {
			return 1
		}
	}
	return 0
}

// parseSemver extracts [major, minor, patch] from a version string.
func parseSemver(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i := 0; i < len(parts) && i < 3; i++ {
		// Strip anything after a hyphen (e.g. "1-rc1" -> "1")
		num := strings.SplitN(parts[i], "-", 2)[0]
		n, _ := strconv.Atoi(num)
		result[i] = n
	}
	return result
}
