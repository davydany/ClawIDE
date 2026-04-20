package breakdown

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Managed-region markers. These delimit a block in CLAUDE.md that breakdown owns and rewrites
// on every run — users are expected not to edit between them.
const (
	regionBegin = "<!-- clawide:subtasks:begin -->"
	regionEnd   = "<!-- clawide:subtasks:end -->"
)

// regionRe matches the entire managed region including both markers. Multiline dotall so
// ".*?" spans newlines between the markers.
var regionRe = regexp.MustCompile(`(?s)` + regexp.QuoteMeta(regionBegin) + `.*?` + regexp.QuoteMeta(regionEnd))

// regionEntryRe pulls a single "- [`tasks/<slug>.md`](tasks/<slug>.md) — <title>" entry out
// of an existing region body. Group 1 = slug, group 2 = title.
var regionEntryRe = regexp.MustCompile(`(?m)^-\s+\[` + "`" + `tasks/([^` + "`" + `]+)\.md` + "`" + `\]\(tasks/[^)]+\)\s+—\s+(.+)$`)

// UpdateClaudeMD writes (or updates) the managed region in <worktreePath>/CLAUDE.md to
// include an entry for the given slug + title. Existing entries are preserved, deduped by
// slug, and sorted by first-added order. Missing CLAUDE.md files are created.
func UpdateClaudeMD(worktreePath, slug, taskTitle string) error {
	path := filepath.Join(worktreePath, "CLAUDE.md")
	orig, err := readExistingCLAUDE(path)
	if err != nil {
		return err
	}

	entries := parseExistingEntries(orig)
	entries = upsertEntry(entries, slug, taskTitle)
	region := buildRegion(entries)

	var out string
	switch {
	case regionRe.MatchString(orig):
		out = regionRe.ReplaceAllString(orig, region)
	case orig == "":
		out = region + "\n"
	default:
		sep := "\n"
		if strings.HasSuffix(orig, "\n") {
			sep = ""
		}
		out = orig + sep + "\n" + region + "\n"
	}

	return writeFileAtomic(path, []byte(out), 0644)
}

// readExistingCLAUDE loads the file if it exists. CRLF is normalized to LF so the region
// regex matches on Windows-edited files. A missing file returns "" with no error.
func readExistingCLAUDE(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read CLAUDE.md: %w", err)
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n"), nil
}

// regionEntry is a single bullet inside the managed region.
type regionEntry struct {
	Slug  string
	Title string
}

// parseExistingEntries extracts entries from the managed region in orig, preserving their
// original order. If there is no region, returns nil.
func parseExistingEntries(orig string) []regionEntry {
	region := regionRe.FindString(orig)
	if region == "" {
		return nil
	}
	matches := regionEntryRe.FindAllStringSubmatch(region, -1)
	out := make([]regionEntry, 0, len(matches))
	for _, m := range matches {
		out = append(out, regionEntry{Slug: m[1], Title: strings.TrimSpace(m[2])})
	}
	return out
}

// upsertEntry either updates the title for an existing slug in place, or appends a new
// entry at the end. Order is preserved — the first-linked task stays first.
func upsertEntry(entries []regionEntry, slug, title string) []regionEntry {
	for i := range entries {
		if entries[i].Slug == slug {
			entries[i].Title = title
			return entries
		}
	}
	return append(entries, regionEntry{Slug: slug, Title: title})
}

// buildRegion renders the managed region block. Always ends with the close marker so
// subsequent runs can find it via regionRe.
func buildRegion(entries []regionEntry) string {
	var b strings.Builder
	b.WriteString(regionBegin)
	b.WriteString("\n## Active Subtasks\n\n")
	b.WriteString("ClawIDE-managed region — do not edit by hand. Regenerated on breakdown.\n\n")
	for _, e := range entries {
		fmt.Fprintf(&b, "- [`tasks/%s.md`](tasks/%s.md) — %s\n", e.Slug, e.Slug, e.Title)
	}
	b.WriteString(regionEnd)
	return b.String()
}
