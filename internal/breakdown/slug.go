// Package breakdown turns an AI-generated task breakdown into a markdown checklist file
// inside the linked worktree, and maintains a managed region in that worktree's CLAUDE.md
// that points at the generated files.
package breakdown

import "strings"

// slugMaxLen caps generated basenames at a length safely under filesystem limits on macOS
// (HFS+: 255 bytes per name) and eCryptfs (143 chars). 60 chars leaves room for a
// "-<taskID[:6]>" collision suffix plus the ".md" extension.
const slugMaxLen = 60

// TaskSlug turns a task title into a filesystem-safe basename in the charset [a-z0-9-].
// Runs of non-alphanumeric characters collapse to a single "-". Leading/trailing dashes
// are trimmed. If the result is empty (e.g. title was all punctuation or pure CJK), the
// fallback "task-<first 8 of taskID>" is returned so the filename is always non-empty.
// Output is capped at slugMaxLen, truncated at the last "-" boundary so it stays readable.
func TaskSlug(title, taskID string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	var b strings.Builder
	b.Grow(len(s))
	prevDash := false
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	out := strings.TrimRight(b.String(), "-")
	if out == "" {
		return fallbackSlug(taskID)
	}
	if len(out) > slugMaxLen {
		// Truncate at the last "-" boundary within the cap so we don't cut mid-word.
		cut := strings.LastIndex(out[:slugMaxLen], "-")
		if cut <= 0 {
			cut = slugMaxLen
		}
		out = strings.TrimRight(out[:cut], "-")
		if out == "" {
			return fallbackSlug(taskID)
		}
	}
	return out
}

// fallbackSlug derives a predictable slug from a task ID when the title produces nothing usable.
// Strips dashes so the output is stable regardless of UUID punctuation position — "abc-12345" and
// "abc12345" both produce "task-abc12345", keyed off the first 8 hex chars.
func fallbackSlug(taskID string) string {
	clean := strings.ToLower(strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, taskID))
	if len(clean) > 8 {
		clean = clean[:8]
	}
	if clean == "" {
		clean = "unnamed"
	}
	return "task-" + clean
}
