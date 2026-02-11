package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// FileStatus represents a single entry from `git status --porcelain`.
type FileStatus struct {
	Path   string `json:"path"`
	Status string `json:"status"` // M, A, D, ?, R, etc.
	Staged bool   `json:"staged"`
}

// Status returns the working tree status for the repo at repoPath by
// parsing the output of `git status --porcelain`.
func Status(repoPath string) ([]FileStatus, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git status: %w", err)
	}

	var files []FileStatus
	for _, line := range strings.Split(string(out), "\n") {
		if len(line) < 4 {
			continue
		}

		indexStatus := line[0]
		workTreeStatus := line[1]
		path := strings.TrimSpace(line[3:])

		// Handle renames: "R  old -> new"
		if idx := strings.Index(path, " -> "); idx >= 0 {
			path = path[idx+4:]
		}

		if indexStatus != ' ' && indexStatus != '?' {
			files = append(files, FileStatus{
				Path:   path,
				Status: string(indexStatus),
				Staged: true,
			})
		}

		if workTreeStatus != ' ' {
			status := string(workTreeStatus)
			if workTreeStatus == '?' {
				status = "?"
			}
			files = append(files, FileStatus{
				Path:   path,
				Status: status,
				Staged: false,
			})
		}
	}

	return files, nil
}

// Add stages the specified files in the repo at repoPath.
func Add(repoPath string, files []string) error {
	if len(files) == 0 {
		return nil
	}
	args := append([]string{"add", "--"}, files...)
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// Commit creates a commit in the repo at repoPath with the given message.
// Only already-staged changes are committed.
func Commit(repoPath, message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// Merge merges the given branch into the currently checked-out branch in
// repoPath. If a conflict occurs, the merge is aborted and an error is
// returned.
func Merge(repoPath, branch string) error {
	cmd := exec.Command("git", "merge", branch, "--no-edit")
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		// Abort the conflicting merge
		abortCmd := exec.Command("git", "merge", "--abort")
		abortCmd.Dir = repoPath
		abortCmd.Run() // best-effort abort
		return fmt.Errorf("merge conflict: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// DeleteBranch safely deletes a local branch. Uses -d (not -D) so git
// will refuse if the branch has unmerged commits.
func DeleteBranch(repoPath, branch string) error {
	cmd := exec.Command("git", "branch", "-d", branch)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git branch -d %s: %s: %w", branch, strings.TrimSpace(string(output)), err)
	}
	return nil
}

// branchSlugRe matches characters that are not alphanumeric, hyphens, or slashes.
var branchSlugRe = regexp.MustCompile(`[^a-z0-9-]+`)

// SanitizeBranchName converts a human-readable feature name into a valid
// git branch name prefixed with "feature/". For example:
//
//	"Add OAuth2 Support" → "feature/add-oauth2-support"
func SanitizeBranchName(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = branchSlugRe.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "unnamed"
	}
	return "feature/" + slug
}
