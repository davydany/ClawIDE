package model

import "github.com/google/uuid"

// PaneNode represents a node in the binary tree of pane splits.
// A node is either a Leaf (single terminal pane) or a Split (two children).
type PaneNode struct {
	Type      string    `json:"type"`                 // "leaf" or "split"
	PaneID    string    `json:"pane_id,omitempty"`    // leaf only
	TmuxName  string    `json:"tmux_name,omitempty"`  // leaf only: "clawide-{PaneID}"
	Direction string    `json:"direction,omitempty"`   // split only: "horizontal" or "vertical"
	Ratio     float64   `json:"ratio,omitempty"`       // split only: 0.1-0.9
	First     *PaneNode `json:"first,omitempty"`       // split only
	Second    *PaneNode `json:"second,omitempty"`      // split only
}

// NewLeafPane creates a new leaf pane node with the given pane ID.
func NewLeafPane(paneID string) *PaneNode {
	return &PaneNode{
		Type:     "leaf",
		PaneID:   paneID,
		TmuxName: "clawide-" + paneID,
	}
}

// NewLeafPaneWithID creates a new leaf pane with an auto-generated UUID.
func NewLeafPaneWithID() *PaneNode {
	return NewLeafPane(uuid.New().String())
}

// FindPane searches the tree for a pane by ID.
// Returns (target, parent). Parent is nil if target is the root.
func (n *PaneNode) FindPane(paneID string) (target *PaneNode, parent *PaneNode) {
	if n == nil {
		return nil, nil
	}
	if n.Type == "leaf" && n.PaneID == paneID {
		return n, nil
	}
	if n.Type == "split" {
		if n.First != nil {
			if n.First.Type == "leaf" && n.First.PaneID == paneID {
				return n.First, n
			}
			if t, p := n.First.FindPane(paneID); t != nil {
				if p == nil {
					return t, n
				}
				return t, p
			}
		}
		if n.Second != nil {
			if n.Second.Type == "leaf" && n.Second.PaneID == paneID {
				return n.Second, n
			}
			if t, p := n.Second.FindPane(paneID); t != nil {
				if p == nil {
					return t, n
				}
				return t, p
			}
		}
	}
	return nil, nil
}

// CollectLeaves returns all leaf pane IDs in the tree.
func (n *PaneNode) CollectLeaves() []string {
	if n == nil {
		return nil
	}
	if n.Type == "leaf" {
		return []string{n.PaneID}
	}
	var leaves []string
	if n.First != nil {
		leaves = append(leaves, n.First.CollectLeaves()...)
	}
	if n.Second != nil {
		leaves = append(leaves, n.Second.CollectLeaves()...)
	}
	return leaves
}

// Clone returns a deep copy of the pane tree.
func (n *PaneNode) Clone() *PaneNode {
	if n == nil {
		return nil
	}
	c := &PaneNode{
		Type:      n.Type,
		PaneID:    n.PaneID,
		TmuxName:  n.TmuxName,
		Direction: n.Direction,
		Ratio:     n.Ratio,
	}
	if n.First != nil {
		c.First = n.First.Clone()
	}
	if n.Second != nil {
		c.Second = n.Second.Clone()
	}
	return c
}

// ReplaceChild swaps a child in a split node.
func (n *PaneNode) ReplaceChild(old, replacement *PaneNode) {
	if n.Type != "split" {
		return
	}
	if n.First == old {
		n.First = replacement
	} else if n.Second == old {
		n.Second = replacement
	}
}

// HasPane checks if a pane with the given ID exists in the tree.
func (n *PaneNode) HasPane(paneID string) bool {
	t, _ := n.FindPane(paneID)
	return t != nil
}
