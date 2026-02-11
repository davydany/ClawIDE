package handler

import (
	"testing"

	"github.com/davydany/ClawIDE/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestReplaceNodeInTree(t *testing.T) {
	t.Run("replaces first child", func(t *testing.T) {
		first := model.NewLeafPane("old")
		second := model.NewLeafPane("keep")
		root := &model.PaneNode{
			Type:   "split",
			First:  first,
			Second: second,
		}

		replacement := model.NewLeafPane("new")
		replaceNodeInTree(root, first, replacement)

		assert.Equal(t, replacement, root.First)
		assert.Equal(t, second, root.Second)
	})

	t.Run("replaces second child", func(t *testing.T) {
		first := model.NewLeafPane("keep")
		second := model.NewLeafPane("old")
		root := &model.PaneNode{
			Type:   "split",
			First:  first,
			Second: second,
		}

		replacement := model.NewLeafPane("new")
		replaceNodeInTree(root, second, replacement)

		assert.Equal(t, first, root.First)
		assert.Equal(t, replacement, root.Second)
	})

	t.Run("deep replacement", func(t *testing.T) {
		deepChild := model.NewLeafPane("deep-old")
		innerSplit := &model.PaneNode{
			Type:   "split",
			First:  deepChild,
			Second: model.NewLeafPane("sibling"),
		}
		root := &model.PaneNode{
			Type:   "split",
			First:  innerSplit,
			Second: model.NewLeafPane("other"),
		}

		replacement := model.NewLeafPane("deep-new")
		replaceNodeInTree(root, deepChild, replacement)

		assert.Equal(t, replacement, innerSplit.First)
	})

	t.Run("nil root no-op", func(t *testing.T) {
		// Should not panic
		replaceNodeInTree(nil, model.NewLeafPane("a"), model.NewLeafPane("b"))
	})

	t.Run("leaf root no-op", func(t *testing.T) {
		root := model.NewLeafPane("leaf")
		old := model.NewLeafPane("old")
		replacement := model.NewLeafPane("new")

		replaceNodeInTree(root, old, replacement)
		// leaf root remains unchanged
		assert.Equal(t, "leaf", root.PaneID)
	})
}
