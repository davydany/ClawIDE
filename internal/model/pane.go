package model

import "github.com/google/uuid"

// PaneType constants for distinguishing agent vs shell panes.
const (
	PaneTypeAgent = "agent"
	PaneTypeShell = "shell"
)

// PaneNode represents a node in the binary tree of pane splits.
// A node is either a Leaf (single terminal pane) or a Split (two children).
type PaneNode struct {
	Type      string    `json:"type"`                 // "leaf" or "split"
	PaneID    string    `json:"pane_id,omitempty"`    // leaf only
	TmuxName  string    `json:"tmux_name,omitempty"`  // leaf only: "clawide-{PaneID}"
	Name      string    `json:"name,omitempty"`        // leaf only: user-assigned display name
	PaneType  string    `json:"pane_type,omitempty"`   // leaf only: "agent" or "shell"
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

// NewAgentPane creates a new leaf pane node that will auto-launch the configured agent command.
func NewAgentPane(paneID string) *PaneNode {
	return &PaneNode{
		Type:     "leaf",
		PaneID:   paneID,
		TmuxName: "clawide-" + paneID,
		PaneType: PaneTypeAgent,
	}
}

// NewAgentPaneWithID creates a new agent pane with an auto-generated UUID.
func NewAgentPaneWithID() *PaneNode {
	return NewAgentPane(uuid.New().String())
}

// EffectivePaneType returns the pane type, defaulting to PaneTypeShell for
// backward compatibility with existing state.json entries that lack a pane_type.
func (n *PaneNode) EffectivePaneType() string {
	if n.PaneType == "" {
		return PaneTypeShell
	}
	return n.PaneType
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
		Name:      n.Name,
		PaneType:  n.PaneType,
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

// replaceInTree walks the tree and replaces old with replacement.
func replaceInTree(root, old, replacement *PaneNode) {
	if root == nil || root.Type != "split" {
		return
	}
	if root.First == old {
		root.First = replacement
		return
	}
	if root.Second == old {
		root.Second = replacement
		return
	}
	replaceInTree(root.First, old, replacement)
	replaceInTree(root.Second, old, replacement)
}

// DetachPane removes a leaf pane from the tree by collapsing its parent split.
// Returns (detachedLeaf, newRoot). If the pane is the root (only pane), returns (nil, n).
// If the pane is not found, returns (nil, n).
func (n *PaneNode) DetachPane(paneID string) (*PaneNode, *PaneNode) {
	if n == nil {
		return nil, n
	}
	// Root is the target leaf — can't detach the only pane
	if n.Type == "leaf" && n.PaneID == paneID {
		return nil, n
	}

	target, parent := n.FindPane(paneID)
	if target == nil || parent == nil {
		return nil, n
	}

	// Determine the surviving sibling
	var sibling *PaneNode
	if parent.First == target {
		sibling = parent.Second
	} else {
		sibling = parent.First
	}

	// Replace parent with sibling in the tree
	if n == parent {
		// Parent is root — sibling becomes the new root
		return target, sibling
	}
	replaceInTree(n, parent, sibling)
	return target, n
}

// InsertPaneAt inserts sourceLeaf adjacent to the pane identified by targetPaneID.
// position must be "left", "right", "top", or "bottom".
// Returns the new root of the tree.
func (n *PaneNode) InsertPaneAt(sourceLeaf *PaneNode, targetPaneID, position string) *PaneNode {
	if n == nil || sourceLeaf == nil {
		return n
	}

	// Map position to direction and ordering
	var direction string
	var sourceIsFirst bool
	switch position {
	case "left":
		direction = "horizontal"
		sourceIsFirst = true
	case "right":
		direction = "horizontal"
		sourceIsFirst = false
	case "top":
		direction = "vertical"
		sourceIsFirst = true
	case "bottom":
		direction = "vertical"
		sourceIsFirst = false
	default:
		return n
	}

	target, parent := n.FindPane(targetPaneID)
	if target == nil {
		return n
	}

	// Create a split node wrapping source and target
	splitNode := &PaneNode{
		Type:      "split",
		Direction: direction,
		Ratio:     0.5,
	}
	if sourceIsFirst {
		splitNode.First = sourceLeaf
		splitNode.Second = target
	} else {
		splitNode.First = target
		splitNode.Second = sourceLeaf
	}

	// Replace target in tree with the new split
	if parent == nil {
		// Target is root
		return splitNode
	}
	parent.ReplaceChild(target, splitNode)
	return n
}
