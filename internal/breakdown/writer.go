package breakdown

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MaxChecklistBytes caps the generated checklist body. Realistic 3–7 item checklists are
// well under 1 KB; 64 KB is a paranoia bound against a runaway model.
const MaxChecklistBytes = 64 * 1024

// taskIDHeaderPrefix opens the task-id header we write at the top of every generated subtask
// file. The parser only checks the prefix; the value after the colon is the owning task's ID.
const taskIDHeaderPrefix = "<!-- clawide:task-id:"

// writeFileAtomic writes data to path by creating a sibling temp file and renaming it into
// place, so a crash during write never leaves a half-written file for readers to find.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		cleanup()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		cleanup()
		return fmt.Errorf("sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		cleanup()
		return fmt.Errorf("chmod temp: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return fmt.Errorf("rename temp: %w", err)
	}
	return nil
}

// WriteSubtaskFile writes the checklist for a task into <worktreePath>/tasks/<slug>.md.
// The file gets a clawide:task-id header at the top so future runs can detect ownership and
// handle slug collisions. If a file already exists at the target path owned by a *different*
// task ID, the slug is suffixed with -<taskID[:6]> and we retry once. If overwrite is false
// and an owned file already exists, ErrExists is returned so the handler can surface 409.
func WriteSubtaskFile(worktreePath, slug, taskID, taskTitle, checklist string, overwrite bool) (absPath string, err error) {
	if len(checklist) > MaxChecklistBytes {
		return "", fmt.Errorf("checklist exceeds %d bytes", MaxChecklistBytes)
	}
	if strings.TrimSpace(checklist) == "" {
		return "", fmt.Errorf("checklist is empty")
	}

	finalSlug, err := resolveSlugCollision(worktreePath, slug, taskID, overwrite)
	if err != nil {
		return "", err
	}

	target := filepath.Join(worktreePath, "tasks", finalSlug+".md")
	body := buildSubtaskBody(taskID, taskTitle, checklist)
	if err := writeFileAtomic(target, []byte(body), 0644); err != nil {
		return "", err
	}
	return target, nil
}

// ErrExists is returned when overwrite is false and the target subtask file already exists
// and is owned by the task being broken down.
var ErrExists = fmt.Errorf("subtask file already exists")

// resolveSlugCollision returns the slug to use. Rules:
//   - If no file exists at the target path, use the slug as-is.
//   - If a file exists and its clawide:task-id header matches the current taskID, return the
//     slug as-is when overwrite is true, or ErrExists when it's false.
//   - If a file exists but its header names a different task, append "-<taskID[:6]>" and
//     retry. If that also collides with a different owner, surface a descriptive error.
func resolveSlugCollision(worktreePath, slug, taskID string, overwrite bool) (string, error) {
	tryPath := filepath.Join(worktreePath, "tasks", slug+".md")
	owner, exists, err := readTaskIDHeader(tryPath)
	if err != nil {
		return "", err
	}
	if !exists {
		return slug, nil
	}
	if owner == taskID {
		if overwrite {
			return slug, nil
		}
		return "", ErrExists
	}

	// Different owner — try a disambiguated slug.
	suffix := taskID
	if len(suffix) > 6 {
		suffix = suffix[:6]
	}
	alt := slug + "-" + suffix
	altPath := filepath.Join(worktreePath, "tasks", alt+".md")
	altOwner, altExists, err := readTaskIDHeader(altPath)
	if err != nil {
		return "", err
	}
	if !altExists {
		return alt, nil
	}
	if altOwner == taskID {
		if overwrite {
			return alt, nil
		}
		return "", ErrExists
	}
	return "", fmt.Errorf("slug %q collides with another task (%s)", slug, owner)
}

// readTaskIDHeader opens path, reads the first 4 KB looking for the clawide:task-id header,
// and returns the owner task ID. Returns (ownerID, true, nil) if the header is found,
// ("", true, nil) if the file exists but has no header, ("", false, nil) if the file is absent.
func readTaskIDHeader(path string) (string, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	head := data
	if len(head) > 4096 {
		head = head[:4096]
	}
	for _, line := range strings.Split(string(head), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, taskIDHeaderPrefix) {
			continue
		}
		rest := strings.TrimPrefix(line, taskIDHeaderPrefix)
		rest = strings.TrimSuffix(rest, "-->")
		return strings.TrimSpace(rest), true, nil
	}
	return "", true, nil
}

func buildSubtaskBody(taskID, taskTitle, checklist string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s -->\n", taskIDHeaderPrefix, taskID)
	fmt.Fprintf(&b, "# %s\n\n", taskTitle)
	b.WriteString(strings.TrimRight(checklist, "\n"))
	b.WriteString("\n")
	return b.String()
}
