package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// initTestRepo creates a bare git repo with an initial commit.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v failed: %s", args, string(out))
	}
	run("init", "-b", "main")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test"), 0644))
	run("add", ".")
	run("commit", "-m", "initial")
	return dir
}

func TestListRemotes(t *testing.T) {
	dir := initTestRepo(t)

	// Initially no remotes
	remotes, err := ListRemotes(dir)
	require.NoError(t, err)
	assert.Empty(t, remotes)

	// Add a remote
	cmd := exec.Command("git", "remote", "add", "origin", "https://example.com/repo.git")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())

	remotes, err = ListRemotes(dir)
	require.NoError(t, err)
	require.Len(t, remotes, 1)
	assert.Equal(t, "origin", remotes[0].Name)
	assert.Equal(t, "https://example.com/repo.git", remotes[0].URL)
}

func TestListRemotes_Multiple(t *testing.T) {
	dir := initTestRepo(t)

	for _, r := range []struct{ name, url string }{
		{"origin", "https://example.com/origin.git"},
		{"upstream", "https://example.com/upstream.git"},
	} {
		cmd := exec.Command("git", "remote", "add", r.name, r.url)
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
	}

	remotes, err := ListRemotes(dir)
	require.NoError(t, err)
	assert.Len(t, remotes, 2)

	names := make(map[string]bool)
	for _, r := range remotes {
		names[r.Name] = true
	}
	assert.True(t, names["origin"])
	assert.True(t, names["upstream"])
}

func TestFetchAll(t *testing.T) {
	dir := initTestRepo(t)
	// FetchAll with no remotes should not error
	err := FetchAll(dir)
	assert.NoError(t, err)
}

func TestCreateTrackingBranch(t *testing.T) {
	// Set up: create a "remote" repo and clone it
	remoteDir := initTestRepo(t)
	cloneDir := t.TempDir()

	cmd := exec.Command("git", "clone", remoteDir, cloneDir)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "clone failed: %s", string(out))

	// Create a branch on the remote
	run := func(dir string, args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v failed: %s", args, string(out))
	}
	run(remoteDir, "checkout", "-b", "develop")
	require.NoError(t, os.WriteFile(filepath.Join(remoteDir, "dev.txt"), []byte("dev"), 0644))
	run(remoteDir, "add", ".")
	run(remoteDir, "commit", "-m", "dev commit")

	// Fetch in clone
	run(cloneDir, "fetch", "origin")

	// Create tracking branch
	err = CreateTrackingBranch(cloneDir, "develop", "origin/develop")
	require.NoError(t, err)

	// Verify we're on the new branch
	branch, err := CurrentBranch(cloneDir)
	require.NoError(t, err)
	assert.Equal(t, "develop", branch)
}

func TestPullFromBranch(t *testing.T) {
	// Set up: create a "remote" repo and clone it
	remoteDir := initTestRepo(t)
	cloneDir := t.TempDir()

	cmd := exec.Command("git", "clone", remoteDir, cloneDir)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "clone failed: %s", string(out))

	// Add a commit to the remote's main branch
	run := func(dir string, args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v failed: %s", args, string(out))
	}
	require.NoError(t, os.WriteFile(filepath.Join(remoteDir, "new.txt"), []byte("new"), 0644))
	run(remoteDir, "add", ".")
	run(remoteDir, "commit", "-m", "new commit")

	// Pull from branch
	err = PullFromBranch(cloneDir, "origin", "main")
	require.NoError(t, err)

	// Verify the new file is present
	_, err = os.Stat(filepath.Join(cloneDir, "new.txt"))
	assert.NoError(t, err)
}

func TestListBranches_RemoteField(t *testing.T) {
	// Set up a cloned repo with remote branches
	remoteDir := initTestRepo(t)
	cloneDir := t.TempDir()

	cmd := exec.Command("git", "clone", remoteDir, cloneDir)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "clone failed: %s", string(out))

	branches, err := ListBranches(cloneDir)
	require.NoError(t, err)

	// Should have local main + remote origin/main
	var foundRemote bool
	for _, b := range branches {
		if b.IsRemote && b.Remote == "origin" {
			foundRemote = true
			break
		}
	}
	assert.True(t, foundRemote, "expected at least one remote branch with Remote='origin'")
}
