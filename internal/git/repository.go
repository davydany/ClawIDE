package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Branch represents a git branch (local or remote).
type Branch struct {
	Name      string `json:"name"`
	IsCurrent bool   `json:"is_current"`
	IsRemote  bool   `json:"is_remote"`
	Remote    string `json:"remote,omitempty"`
}

// RemoteInfo holds details about a single git remote.
type RemoteInfo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Clone clones a git repository from url into targetDir.
// It supports optional branch and depth (shallow clone) parameters.
// Returns the combined stdout/stderr output and any error.
func Clone(ctx context.Context, url, targetDir, branch string, depth int) (string, error) {
	args := []string{"clone"}
	if depth > 0 {
		args = append(args, "--depth", strconv.Itoa(depth))
	}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, "--progress", url, targetDir)

	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("git clone: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return string(output), nil
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

		b := Branch{
			Name:      name,
			IsCurrent: isCurrent,
			IsRemote:  isRemote,
		}

		// Parse remote name from remote branches (e.g. "origin/main" → Remote: "origin")
		if isRemote {
			if idx := strings.Index(name, "/"); idx > 0 {
				b.Remote = name[:idx]
			}
		}

		branches = append(branches, b)
	}

	return branches, nil
}

// ListRemotes returns all configured remotes for the repo at repoPath
// by parsing `git remote -v` output.
func ListRemotes(repoPath string) ([]RemoteInfo, error) {
	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git remote -v: %w", err)
	}

	seen := make(map[string]bool)
	var remotes []RemoteInfo
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		// Format: "origin\thttps://... (fetch)"
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		if seen[name] {
			continue
		}
		seen[name] = true
		remotes = append(remotes, RemoteInfo{
			Name: name,
			URL:  parts[1],
		})
	}

	return remotes, nil
}

// CreateTrackingBranch creates a local branch that tracks a remote branch.
// Equivalent to `git checkout -b <localName> --track <remoteBranch>`.
func CreateTrackingBranch(repoPath, localName, remoteBranch string) error {
	cmd := exec.Command("git", "checkout", "-b", localName, "--track", remoteBranch)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout -b %s --track %s: %s: %w", localName, remoteBranch, strings.TrimSpace(string(output)), err)
	}
	return nil
}
