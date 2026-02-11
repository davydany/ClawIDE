package tmpl

import (
	"bytes"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDictFunc(t *testing.T) {
	t.Run("valid pairs", func(t *testing.T) {
		result, err := dictFunc("key1", "value1", "key2", 42)
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"key1": "value1", "key2": 42}, result)
	})

	t.Run("empty args", func(t *testing.T) {
		result, err := dictFunc()
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("odd number of args", func(t *testing.T) {
		_, err := dictFunc("key1", "value1", "key2")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "even number")
	})

	t.Run("non-string key", func(t *testing.T) {
		_, err := dictFunc(123, "value")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key must be string")
	})
}

func TestRenderTemplateNotFound(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/base.html": &fstest.MapFile{
			Data: []byte(`{{define "base.html"}}base{{end}}`),
		},
	}

	r, err := New(fsys)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = r.Render(&buf, "nonexistent", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
