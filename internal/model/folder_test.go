package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateFolderDepth(t *testing.T) {
	now := time.Now()

	t.Run("root level always valid", func(t *testing.T) {
		err := ValidateFolderDepth("", nil)
		assert.NoError(t, err)
	})

	t.Run("single level parent valid", func(t *testing.T) {
		folders := []Folder{
			{ID: "f1", Name: "Level 1", CreatedAt: now, UpdatedAt: now},
		}
		err := ValidateFolderDepth("f1", folders)
		assert.NoError(t, err)
	})

	t.Run("4 levels valid (new would be 5th)", func(t *testing.T) {
		folders := []Folder{
			{ID: "f1", Name: "Level 1", CreatedAt: now, UpdatedAt: now},
			{ID: "f2", Name: "Level 2", ParentID: "f1", CreatedAt: now, UpdatedAt: now},
			{ID: "f3", Name: "Level 3", ParentID: "f2", CreatedAt: now, UpdatedAt: now},
			{ID: "f4", Name: "Level 4", ParentID: "f3", CreatedAt: now, UpdatedAt: now},
		}
		// Adding under f4 would make depth = 5 (f1 > f2 > f3 > f4 > new)
		err := ValidateFolderDepth("f4", folders)
		assert.NoError(t, err)
	})

	t.Run("5 levels exceeds max", func(t *testing.T) {
		folders := []Folder{
			{ID: "f1", Name: "Level 1", CreatedAt: now, UpdatedAt: now},
			{ID: "f2", Name: "Level 2", ParentID: "f1", CreatedAt: now, UpdatedAt: now},
			{ID: "f3", Name: "Level 3", ParentID: "f2", CreatedAt: now, UpdatedAt: now},
			{ID: "f4", Name: "Level 4", ParentID: "f3", CreatedAt: now, UpdatedAt: now},
			{ID: "f5", Name: "Level 5", ParentID: "f4", CreatedAt: now, UpdatedAt: now},
		}
		// Adding under f5 would make depth = 6
		err := ValidateFolderDepth("f5", folders)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum depth")
	})

	t.Run("missing parent returns error", func(t *testing.T) {
		err := ValidateFolderDepth("nonexistent", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("circular reference detected", func(t *testing.T) {
		folders := []Folder{
			{ID: "f1", Name: "A", ParentID: "f2", CreatedAt: now, UpdatedAt: now},
			{ID: "f2", Name: "B", ParentID: "f1", CreatedAt: now, UpdatedAt: now},
		}
		err := ValidateFolderDepth("f1", folders)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "circular")
	})
}
