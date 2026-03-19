package migration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/davydany/ClawIDE/internal/model"
	"github.com/davydany/ClawIDE/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initGitRepo(t *testing.T, branchName string) string {
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
	run("init", "-b", branchName)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test"), 0644))
	run("add", ".")
	run("commit", "-m", "initial")
	return dir
}

func TestBackfillActiveBranch(t *testing.T) {
	storeDir := t.TempDir()
	st, err := store.New(filepath.Join(storeDir, "state.json"))
	require.NoError(t, err)

	// Create a project with a git repo using "main"
	repoDir := initGitRepo(t, "main")
	now := time.Now()
	p := model.Project{
		ID:        "p1",
		Name:      "Test Project",
		Path:      repoDir,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, st.AddProject(p))

	// Backfill should set ActiveBranch
	BackfillActiveBranch(st)

	updated, ok := st.GetProject("p1")
	require.True(t, ok)
	assert.Equal(t, "main", updated.ActiveBranch)
}

func TestBackfillActiveBranch_SkipsAlreadySet(t *testing.T) {
	storeDir := t.TempDir()
	st, err := store.New(filepath.Join(storeDir, "state.json"))
	require.NoError(t, err)

	repoDir := initGitRepo(t, "main")
	now := time.Now()
	p := model.Project{
		ID:           "p1",
		Name:         "Test",
		Path:         repoDir,
		ActiveBranch: "develop",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	require.NoError(t, st.AddProject(p))

	BackfillActiveBranch(st)

	updated, ok := st.GetProject("p1")
	require.True(t, ok)
	assert.Equal(t, "develop", updated.ActiveBranch, "should not overwrite existing ActiveBranch")
}

func TestBackfillActiveBranch_SkipsNonGitProject(t *testing.T) {
	storeDir := t.TempDir()
	st, err := store.New(filepath.Join(storeDir, "state.json"))
	require.NoError(t, err)

	now := time.Now()
	p := model.Project{
		ID:        "p1",
		Name:      "No Git",
		Path:      t.TempDir(), // not a git repo
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, st.AddProject(p))

	BackfillActiveBranch(st)

	updated, ok := st.GetProject("p1")
	require.True(t, ok)
	assert.Empty(t, updated.ActiveBranch, "should not set ActiveBranch for non-git project")
}

func TestBackfillActiveBranch_MasterBranch(t *testing.T) {
	storeDir := t.TempDir()
	st, err := store.New(filepath.Join(storeDir, "state.json"))
	require.NoError(t, err)

	repoDir := initGitRepo(t, "master")
	now := time.Now()
	p := model.Project{
		ID:        "p1",
		Name:      "Master Project",
		Path:      repoDir,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, st.AddProject(p))

	BackfillActiveBranch(st)

	updated, ok := st.GetProject("p1")
	require.True(t, ok)
	assert.Equal(t, "master", updated.ActiveBranch)
}
