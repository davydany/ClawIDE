package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Worktree represents a git worktree entry parsed from
// `git worktree list --porcelain`.
type Worktree struct {
	Path   string `json:"path"`
	Branch string `json:"branch"`
	HEAD   string `json:"head"`
	IsMain bool   `json:"is_main"`
}

// ListWorktrees returns all worktrees for the repository at repoPath
// by parsing the porcelain output of `git worktree list --porcelain`.
//
// Porcelain format per worktree block:
//
//	worktree /absolute/path
//	HEAD <sha>
//	branch refs/heads/<name>
//	<blank line>
//
// The first worktree listed is always the main working tree.
func ListWorktrees(repoPath string) ([]Worktree, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var worktrees []Worktree
	blocks := splitWorktreeBlocks(string(out))

	for i, block := range blocks {
		wt := parseWorktreeBlock(block)
		if wt.Path == "" {
			continue
		}
		wt.IsMain = (i == 0)
		worktrees = append(worktrees, wt)
	}

	return worktrees, nil
}

// splitWorktreeBlocks splits the porcelain output into individual
// worktree blocks. Each block is separated by a blank line and starts
// with the "worktree " prefix.
func splitWorktreeBlocks(output string) []string {
	var blocks []string
	var current []string

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" {
			if len(current) > 0 {
				blocks = append(blocks, strings.Join(current, "\n"))
				current = nil
			}
			continue
		}
		current = append(current, line)
	}
	// Capture trailing block without a final blank line
	if len(current) > 0 {
		blocks = append(blocks, strings.Join(current, "\n"))
	}

	return blocks
}

// parseWorktreeBlock parses a single porcelain block into a Worktree.
func parseWorktreeBlock(block string) Worktree {
	var wt Worktree

	for _, line := range strings.Split(block, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			wt.Path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "HEAD "):
			wt.HEAD = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			ref := strings.TrimPrefix(line, "branch ")
			// Convert refs/heads/feature/foo to feature/foo
			wt.Branch = strings.TrimPrefix(ref, "refs/heads/")
		case line == "detached":
			wt.Branch = "(detached)"
		}
	}

	return wt
}

// WorktreeDir returns the conventional directory for a worktree given
// the project repo path and branch name. The convention is:
//
//	{parent}/{project}-worktrees/{branch}/
//
// For example, if repoPath is /home/user/projects/myapp and branch is
// "feature/auth", the worktree directory will be
// /home/user/projects/myapp-worktrees/feature/auth/
func WorktreeDir(repoPath, branch string) string {
	parent := filepath.Dir(repoPath)
	base := filepath.Base(repoPath)
	return filepath.Join(parent, base+"-worktrees", branch)
}

// CreateWorktree creates a new git worktree for the given branch. If
// targetDir is empty, the conventional directory layout is used. The
// branch must already exist (local or remote).
func CreateWorktree(repoPath, branch, targetDir string) error {
	if targetDir == "" {
		targetDir = WorktreeDir(repoPath, branch)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(targetDir), 0o755); err != nil {
		return fmt.Errorf("creating worktree parent directory: %w", err)
	}

	cmd := exec.Command("git", "worktree", "add", targetDir, branch)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree add: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// RemoveWorktree removes a worktree by its path. It uses --force to
// handle worktrees with modified or untracked files.
func RemoveWorktree(repoPath, worktreePath string) error {
	cmd := exec.Command("git", "worktree", "remove", worktreePath, "--force")
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree remove: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}
