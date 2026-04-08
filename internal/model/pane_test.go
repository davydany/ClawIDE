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

func TestNewAgentPane(t *testing.T) {
	p := NewAgentPane("abc-123")

	assert.Equal(t, "leaf", p.Type)
	assert.Equal(t, "abc-123", p.PaneID)
	assert.Equal(t, "clawide-abc-123", p.TmuxName)
	assert.Equal(t, PaneTypeAgent, p.PaneType)
	assert.Nil(t, p.First)
	assert.Nil(t, p.Second)
}

func TestNewAgentPaneWithID(t *testing.T) {
	p := NewAgentPaneWithID()

	assert.Equal(t, "leaf", p.Type)
	assert.NotEmpty(t, p.PaneID)
	assert.Equal(t, "clawide-"+p.PaneID, p.TmuxName)
	assert.Equal(t, PaneTypeAgent, p.PaneType)

	p2 := NewAgentPaneWithID()
	assert.NotEqual(t, p.PaneID, p2.PaneID)
}

func TestEffectivePaneType(t *testing.T) {
	tests := []struct {
		name     string
		paneType string
		want     string
	}{
		{"empty defaults to shell", "", PaneTypeShell},
		{"agent stays agent", PaneTypeAgent, PaneTypeAgent},
		{"shell stays shell", PaneTypeShell, PaneTypeShell},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PaneNode{Type: "leaf", PaneID: "test", PaneType: tt.paneType}
			assert.Equal(t, tt.want, p.EffectivePaneType())
		})
	}
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
		First:     NewAgentPane("a"),
		Second:    NewLeafPane("b"),
	}

	clone := original.Clone()

	// Deep copy fidelity
	assert.Equal(t, original.Type, clone.Type)
	assert.Equal(t, original.Direction, clone.Direction)
	assert.Equal(t, original.Ratio, clone.Ratio)
	assert.Equal(t, original.First.PaneID, clone.First.PaneID)
	assert.Equal(t, original.Second.PaneID, clone.Second.PaneID)

	// PaneType preserved
	assert.Equal(t, PaneTypeAgent, clone.First.PaneType)
	assert.Empty(t, clone.Second.PaneType)

	// Original unaffected by clone mutation
	clone.First.PaneID = "mutated"
	assert.Equal(t, "a", original.First.PaneID)

	// Nil clone
	var nilNode *PaneNode
	assert.Nil(t, nilNode.Clone())
}

func TestCloneWithName(t *testing.T) {
	original := &PaneNode{
		Type:      "split",
		Direction: "horizontal",
		Ratio:     0.5,
		First: &PaneNode{
			Type:     "leaf",
			PaneID:   "a",
			TmuxName: "clawide-a",
			Name:     "Server",
			PaneType: PaneTypeAgent,
		},
		Second: &PaneNode{
			Type:     "leaf",
			PaneID:   "b",
			TmuxName: "clawide-b",
			Name:     "Client",
			PaneType: PaneTypeShell,
		},
	}

	clone := original.Clone()

	assert.Equal(t, "Server", clone.First.Name)
	assert.Equal(t, "Client", clone.Second.Name)
	assert.Equal(t, PaneTypeAgent, clone.First.PaneType)
	assert.Equal(t, PaneTypeShell, clone.Second.PaneType)

	// Mutation independence
	clone.First.Name = "Mutated"
	assert.Equal(t, "Server", original.First.Name)
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

func TestDetachPane(t *testing.T) {
	t.Run("detach from 2-pane tree", func(t *testing.T) {
		tree := &PaneNode{
			Type:      "split",
			Direction: "horizontal",
			Ratio:     0.5,
			First:     NewLeafPane("a"),
			Second:    NewLeafPane("b"),
		}

		detached, newRoot := tree.DetachPane("a")
		require.NotNil(t, detached)
		assert.Equal(t, "a", detached.PaneID)
		require.NotNil(t, newRoot)
		assert.Equal(t, "leaf", newRoot.Type)
		assert.Equal(t, "b", newRoot.PaneID)
	})

	t.Run("detach second from 2-pane tree", func(t *testing.T) {
		tree := &PaneNode{
			Type:      "split",
			Direction: "horizontal",
			Ratio:     0.5,
			First:     NewLeafPane("a"),
			Second:    NewLeafPane("b"),
		}

		detached, newRoot := tree.DetachPane("b")
		require.NotNil(t, detached)
		assert.Equal(t, "b", detached.PaneID)
		require.NotNil(t, newRoot)
		assert.Equal(t, "leaf", newRoot.Type)
		assert.Equal(t, "a", newRoot.PaneID)
	})

	t.Run("detach from nested tree", func(t *testing.T) {
		tree := &PaneNode{
			Type:      "split",
			Direction: "horizontal",
			Ratio:     0.5,
			First: &PaneNode{
				Type:      "split",
				Direction: "vertical",
				Ratio:     0.5,
				First:     NewLeafPane("a"),
				Second:    NewLeafPane("b"),
			},
			Second: NewLeafPane("c"),
		}

		detached, newRoot := tree.DetachPane("a")
		require.NotNil(t, detached)
		assert.Equal(t, "a", detached.PaneID)
		// The inner split should collapse, leaving b promoted
		require.NotNil(t, newRoot)
		assert.Equal(t, "split", newRoot.Type)
		assert.Equal(t, "b", newRoot.First.PaneID)
		assert.Equal(t, "c", newRoot.Second.PaneID)
	})

	t.Run("detach root leaf returns nil", func(t *testing.T) {
		tree := NewLeafPane("only")
		detached, newRoot := tree.DetachPane("only")
		assert.Nil(t, detached)
		assert.Equal(t, "only", newRoot.PaneID)
	})

	t.Run("detach nonexistent returns nil", func(t *testing.T) {
		tree := &PaneNode{
			Type:   "split",
			First:  NewLeafPane("a"),
			Second: NewLeafPane("b"),
		}
		detached, newRoot := tree.DetachPane("missing")
		assert.Nil(t, detached)
		assert.NotNil(t, newRoot)
	})

	t.Run("detach from nil returns nil", func(t *testing.T) {
		var tree *PaneNode
		detached, newRoot := tree.DetachPane("any")
		assert.Nil(t, detached)
		assert.Nil(t, newRoot)
	})
}

func TestInsertPaneAt(t *testing.T) {
	t.Run("insert left of root leaf", func(t *testing.T) {
		tree := NewLeafPane("target")
		source := NewLeafPane("source")

		newRoot := tree.InsertPaneAt(source, "target", "left")
		require.NotNil(t, newRoot)
		assert.Equal(t, "split", newRoot.Type)
		assert.Equal(t, "horizontal", newRoot.Direction)
		assert.Equal(t, "source", newRoot.First.PaneID)
		assert.Equal(t, "target", newRoot.Second.PaneID)
	})

	t.Run("insert right of root leaf", func(t *testing.T) {
		tree := NewLeafPane("target")
		source := NewLeafPane("source")

		newRoot := tree.InsertPaneAt(source, "target", "right")
		require.NotNil(t, newRoot)
		assert.Equal(t, "horizontal", newRoot.Direction)
		assert.Equal(t, "target", newRoot.First.PaneID)
		assert.Equal(t, "source", newRoot.Second.PaneID)
	})

	t.Run("insert top of leaf in split", func(t *testing.T) {
		tree := &PaneNode{
			Type:      "split",
			Direction: "horizontal",
			Ratio:     0.5,
			First:     NewLeafPane("a"),
			Second:    NewLeafPane("b"),
		}
		source := NewLeafPane("source")

		newRoot := tree.InsertPaneAt(source, "b", "top")
		require.NotNil(t, newRoot)
		assert.Equal(t, "split", newRoot.Type)
		// Second child should now be a split
		assert.Equal(t, "split", newRoot.Second.Type)
		assert.Equal(t, "vertical", newRoot.Second.Direction)
		assert.Equal(t, "source", newRoot.Second.First.PaneID)
		assert.Equal(t, "b", newRoot.Second.Second.PaneID)
	})

	t.Run("insert bottom", func(t *testing.T) {
		tree := NewLeafPane("target")
		source := NewLeafPane("source")

		newRoot := tree.InsertPaneAt(source, "target", "bottom")
		require.NotNil(t, newRoot)
		assert.Equal(t, "vertical", newRoot.Direction)
		assert.Equal(t, "target", newRoot.First.PaneID)
		assert.Equal(t, "source", newRoot.Second.PaneID)
	})

	t.Run("insert at nonexistent target returns unchanged", func(t *testing.T) {
		tree := NewLeafPane("a")
		source := NewLeafPane("source")

		newRoot := tree.InsertPaneAt(source, "missing", "left")
		assert.Equal(t, "a", newRoot.PaneID)
	})

	t.Run("invalid position returns unchanged", func(t *testing.T) {
		tree := NewLeafPane("a")
		source := NewLeafPane("source")

		newRoot := tree.InsertPaneAt(source, "a", "invalid")
		assert.Equal(t, "a", newRoot.PaneID)
	})
}

func TestDetachThenInsert(t *testing.T) {
	// Simulate a full move: detach pane "a" and insert it to the right of "c"
	tree := &PaneNode{
		Type:      "split",
		Direction: "horizontal",
		Ratio:     0.5,
		First: &PaneNode{
			Type:      "split",
			Direction: "vertical",
			Ratio:     0.5,
			First:     NewLeafPane("a"),
			Second:    NewLeafPane("b"),
		},
		Second: NewLeafPane("c"),
	}

	detached, newRoot := tree.DetachPane("a")
	require.NotNil(t, detached)
	assert.Equal(t, "a", detached.PaneID)

	// After detach: tree should be split(b, c)
	assert.Equal(t, "split", newRoot.Type)
	assert.Equal(t, "b", newRoot.First.PaneID)
	assert.Equal(t, "c", newRoot.Second.PaneID)

	// Insert "a" to the right of "c"
	finalRoot := newRoot.InsertPaneAt(detached, "c", "right")
	require.NotNil(t, finalRoot)

	// Result: split(b, split(c, a))
	assert.Equal(t, "split", finalRoot.Type)
	assert.Equal(t, "b", finalRoot.First.PaneID)
	assert.Equal(t, "split", finalRoot.Second.Type)
	assert.Equal(t, "c", finalRoot.Second.First.PaneID)
	assert.Equal(t, "a", finalRoot.Second.Second.PaneID)

	// All leaves present
	leaves := finalRoot.CollectLeaves()
	assert.ElementsMatch(t, []string{"a", "b", "c"}, leaves)
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
