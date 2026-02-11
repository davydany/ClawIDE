package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLeafPane(t *testing.T) {
	p := NewLeafPane("abc-123")

	assert.Equal(t, "leaf", p.Type)
	assert.Equal(t, "abc-123", p.PaneID)
	assert.Equal(t, "clawide-abc-123", p.TmuxName)
	assert.Nil(t, p.First)
	assert.Nil(t, p.Second)
	assert.Empty(t, p.Direction)
	assert.Zero(t, p.Ratio)
}

func TestNewLeafPaneWithID(t *testing.T) {
	p := NewLeafPaneWithID()

	assert.Equal(t, "leaf", p.Type)
	assert.NotEmpty(t, p.PaneID)
	assert.Equal(t, "clawide-"+p.PaneID, p.TmuxName)

	// UUIDs should be unique
	p2 := NewLeafPaneWithID()
	assert.NotEqual(t, p.PaneID, p2.PaneID)
}

func TestFindPane(t *testing.T) {
	tests := []struct {
		name       string
		tree       *PaneNode
		searchID   string
		wantFound  bool
		wantParent bool
	}{
		{
			name:       "root match",
			tree:       NewLeafPane("root"),
			searchID:   "root",
			wantFound:  true,
			wantParent: false,
		},
		{
			name: "first child",
			tree: &PaneNode{
				Type:   "split",
				First:  NewLeafPane("first"),
				Second: NewLeafPane("second"),
			},
			searchID:   "first",
			wantFound:  true,
			wantParent: true,
		},
		{
			name: "second child",
			tree: &PaneNode{
				Type:   "split",
				First:  NewLeafPane("first"),
				Second: NewLeafPane("second"),
			},
			searchID:   "second",
			wantFound:  true,
			wantParent: true,
		},
		{
			name: "nested tree",
			tree: &PaneNode{
				Type: "split",
				First: &PaneNode{
					Type:   "split",
					First:  NewLeafPane("a"),
					Second: NewLeafPane("b"),
				},
				Second: NewLeafPane("c"),
			},
			searchID:   "b",
			wantFound:  true,
			wantParent: true,
		},
		{
			name: "not found",
			tree: &PaneNode{
				Type:   "split",
				First:  NewLeafPane("first"),
				Second: NewLeafPane("second"),
			},
			searchID:   "missing",
			wantFound:  false,
			wantParent: false,
		},
		{
			name:       "nil node",
			tree:       nil,
			searchID:   "any",
			wantFound:  false,
			wantParent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, parent := tt.tree.FindPane(tt.searchID)
			if tt.wantFound {
				require.NotNil(t, target)
				assert.Equal(t, tt.searchID, target.PaneID)
			} else {
				assert.Nil(t, target)
			}
			if tt.wantParent {
				assert.NotNil(t, parent)
			} else {
				assert.Nil(t, parent)
			}
		})
	}
}

func TestCollectLeaves(t *testing.T) {
	tests := []struct {
		name string
		tree *PaneNode
		want []string
	}{
		{
			name: "single leaf",
			tree: NewLeafPane("only"),
			want: []string{"only"},
		},
		{
			name: "split tree",
			tree: &PaneNode{
				Type:   "split",
				First:  NewLeafPane("a"),
				Second: NewLeafPane("b"),
			},
			want: []string{"a", "b"},
		},
		{
			name: "deeply nested",
			tree: &PaneNode{
				Type: "split",
				First: &PaneNode{
					Type:   "split",
					First:  NewLeafPane("x"),
					Second: NewLeafPane("y"),
				},
				Second: NewLeafPane("z"),
			},
			want: []string{"x", "y", "z"},
		},
		{
			name: "nil node",
			tree: nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tree.CollectLeaves()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClone(t *testing.T) {
	original := &PaneNode{
		Type:      "split",
		Direction: "horizontal",
		Ratio:     0.5,
		First:     NewLeafPane("a"),
		Second:    NewLeafPane("b"),
	}

	clone := original.Clone()

	// Deep copy fidelity
	assert.Equal(t, original.Type, clone.Type)
	assert.Equal(t, original.Direction, clone.Direction)
	assert.Equal(t, original.Ratio, clone.Ratio)
	assert.Equal(t, original.First.PaneID, clone.First.PaneID)
	assert.Equal(t, original.Second.PaneID, clone.Second.PaneID)

	// Original unaffected by clone mutation
	clone.First.PaneID = "mutated"
	assert.Equal(t, "a", original.First.PaneID)

	// Nil clone
	var nilNode *PaneNode
	assert.Nil(t, nilNode.Clone())
}

func TestReplaceChild(t *testing.T) {
	t.Run("replace first", func(t *testing.T) {
		first := NewLeafPane("old")
		second := NewLeafPane("keep")
		parent := &PaneNode{Type: "split", First: first, Second: second}

		replacement := NewLeafPane("new")
		parent.ReplaceChild(first, replacement)

		assert.Equal(t, replacement, parent.First)
		assert.Equal(t, second, parent.Second)
	})

	t.Run("replace second", func(t *testing.T) {
		first := NewLeafPane("keep")
		second := NewLeafPane("old")
		parent := &PaneNode{Type: "split", First: first, Second: second}

		replacement := NewLeafPane("new")
		parent.ReplaceChild(second, replacement)

		assert.Equal(t, first, parent.First)
		assert.Equal(t, replacement, parent.Second)
	})

	t.Run("non-split node no-op", func(t *testing.T) {
		leaf := NewLeafPane("leaf")
		replacement := NewLeafPane("new")
		leaf.ReplaceChild(leaf, replacement) // should not panic
		assert.Equal(t, "leaf", leaf.Type)
	})

	t.Run("mismatch no-op", func(t *testing.T) {
		first := NewLeafPane("a")
		second := NewLeafPane("b")
		parent := &PaneNode{Type: "split", First: first, Second: second}

		unrelated := NewLeafPane("unrelated")
		replacement := NewLeafPane("new")
		parent.ReplaceChild(unrelated, replacement)

		assert.Equal(t, first, parent.First)
		assert.Equal(t, second, parent.Second)
	})
}

func TestHasPane(t *testing.T) {
	tree := &PaneNode{
		Type:   "split",
		First:  NewLeafPane("exists"),
		Second: NewLeafPane("other"),
	}

	assert.True(t, tree.HasPane("exists"))
	assert.True(t, tree.HasPane("other"))
	assert.False(t, tree.HasPane("missing"))
}
