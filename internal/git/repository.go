package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Branch represents a git branch (local or remote).
type Branch struct {
	Name      string `json:"name"`
	IsCurrent bool   `json:"is_current"`
	IsRemote  bool   `json:"is_remote"`
}

// IsGitRepo checks whether the given path contains a git repository
// by looking for a .git directory or file (worktrees use a .git file).
func IsGitRepo(path string) bool {
	info, err := os.Stat(filepath.Join(path, ".git"))
	if err != nil {
		return false
	}
	// .git can be a directory (normal repo) or a file (worktree link)
	return info.IsDir() || info.Mode().IsRegular()
}

// CurrentBranch returns the name of the currently checked-out branch
// in the repository at repoPath. Returns an error if the command fails
// or the repo is in a detached HEAD state with no branch name.
func CurrentBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	branch := strings.TrimSpace(string(out))
	if branch == "HEAD" {
		// Detached HEAD -- return empty string, no error
		return "", nil
	}
	return branch, nil
}

// CheckoutBranch switches the repository at repoPath to the specified branch.
func CheckoutBranch(repoPath, branch string) error {
	cmd := exec.Command("git", "checkout", branch)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout %s: %s: %w", branch, strings.TrimSpace(string(output)), err)
	}
	return nil
}

// CreateBranch creates a new branch at repoPath based on the given base branch
// and switches to it. Equivalent to `git checkout -b <name> <base>`.
func CreateBranch(repoPath, name, base string) error {
	args := []string{"checkout", "-b", name}
	if base != "" {
		args = append(args, base)
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout -b %s: %s: %w", name, strings.TrimSpace(string(output)), err)
	}
	return nil
}

// DetectMainBranch returns the name of the main integration branch (either
// "main" or "master") by checking which exists locally.
func DetectMainBranch(repoPath string) (string, error) {
	for _, candidate := range []string{"main", "master"} {
		cmd := exec.Command("git", "rev-parse", "--verify", candidate)
		cmd.Dir = repoPath
		if err := cmd.Run(); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("neither main nor master branch found")
}

// ListBranches returns all local and remote branches for the repository
// at repoPath by running `git branch -a`.
func ListBranches(repoPath string) ([]Branch, error) {
	cmd := exec.Command("git", "branch", "-a")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var branches []Branch
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		isCurrent := strings.HasPrefix(line, "* ")
		// Strip leading whitespace and the "* " marker
		name := strings.TrimSpace(line)
		if isCurrent {
			name = strings.TrimPrefix(name, "* ")
		}

		// Skip detached HEAD entries like "* (HEAD detached at abc1234)"
		if strings.HasPrefix(name, "(") {
			continue
		}

		isRemote := false
		if strings.HasPrefix(name, "remotes/") {
			isRemote = true
			name = strings.TrimPrefix(name, "remotes/")

			// Skip symbolic refs like origin/HEAD -> origin/main
			if strings.Contains(name, " -> ") {
				continue
			}
		}

		branches = append(branches, Branch{
			Name:      name,
			IsCurrent: isCurrent,
			IsRemote:  isRemote,
		})
	}

	return branches, nil
}
