package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/davydany/ClawIDE/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPromptForgeStoreRoundTrip exercises the disk round-trip: create a
// folder, create a prompt inside it, save a compiled version, then reload the
// store from disk and verify everything came back intact.
func TestPromptForgeStoreRoundTrip(t *testing.T) {
	base := filepath.Join(t.TempDir(), "promptforge")

	s, err := NewPromptForgeStore(base)
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Second)
	folder := model.Folder{
		ID:        uuid.New().String(),
		Name:      "Coding",
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, s.CreateFolder(folder))

	prompt := model.Prompt{
		ID:       uuid.New().String(),
		FolderID: folder.ID,
		Title:    "Refactor-Go",
		Type:     model.PromptTypeJinja,
		Variables: []model.Variable{
			{Name: "project", Type: model.VariableTypeString, Label: "Project", Required: true},
			{Name: "lines", Type: model.VariableTypeNumber, Default: "80"},
		},
		Content:   "# Refactor {{ project }}\n\nKeep functions under {{ lines }} lines.\n",
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, s.AddPrompt(prompt))

	version := model.CompiledVersion{
		ID:             uuid.New().String(),
		PromptID:       prompt.ID,
		Title:          "first compile",
		VariableValues: map[string]interface{}{"project": "acme", "lines": 80},
		Content:        "# Refactor acme\n\nKeep functions under 80 lines.\n",
		CompiledAt:     now,
	}
	require.NoError(t, s.AddVersion(version))

	// Fresh store forces a reload from disk.
	s2, err := NewPromptForgeStore(base)
	require.NoError(t, err)

	folders := s2.GetFolders()
	require.Len(t, folders, 1)
	assert.Equal(t, folder.ID, folders[0].ID)
	assert.Equal(t, "Coding", folders[0].Name)

	prompts := s2.GetAllPrompts()
	require.Len(t, prompts, 1)
	got := prompts[0]
	assert.Equal(t, prompt.ID, got.ID)
	assert.Equal(t, prompt.Title, got.Title)
	assert.Equal(t, folder.ID, got.FolderID)
	assert.Equal(t, model.PromptTypeJinja, got.Type)
	assert.Len(t, got.Variables, 2)
	assert.Equal(t, "project", got.Variables[0].Name)
	assert.Equal(t, model.VariableTypeString, got.Variables[0].Type)
	assert.True(t, got.Variables[0].Required)
	assert.Contains(t, got.Content, "{{ project }}")

	versions, err := s2.GetVersions(prompt.ID)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, "first compile", versions[0].Title)
	assert.Contains(t, versions[0].Content, "Refactor acme")

	// Rename prompt — compiled directory should move with it.
	renamed := got
	renamed.Title = "Refactor-Python"
	require.NoError(t, s2.UpdatePrompt(renamed))

	versionsAfterRename, err := s2.GetVersions(prompt.ID)
	require.NoError(t, err)
	require.Len(t, versionsAfterRename, 1, "compiled version must survive rename")

	// Duplicate title in the same folder should fail.
	dup := renamed
	dup.ID = uuid.New().String()
	assert.Error(t, s2.AddPrompt(dup))
}

func TestPromptForgeFolderDeletionRespectsCascade(t *testing.T) {
	base := filepath.Join(t.TempDir(), "promptforge")
	s, err := NewPromptForgeStore(base)
	require.NoError(t, err)

	f := model.Folder{ID: uuid.New().String(), Name: "Writing"}
	require.NoError(t, s.CreateFolder(f))
	require.NoError(t, s.AddPrompt(model.Prompt{
		ID: uuid.New().String(), FolderID: f.ID, Title: "Outline", Type: model.PromptTypePlain, Content: "body",
	}))

	// Non-cascading delete must refuse.
	err = s.DeleteFolder(f.ID, false)
	require.Error(t, err)

	// Cascading delete should succeed and remove contained prompts.
	require.NoError(t, s.DeleteFolder(f.ID, true))
	assert.Len(t, s.GetFolders(), 0)
	assert.Len(t, s.GetAllPrompts(), 0)
}
