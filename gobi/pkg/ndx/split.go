package ndx

import "fmt"

// InternalNodeFull reports whether node has reached its internal page capacity.
func InternalNodeFull(h *Header, node *Node) bool {
	if h == nil || node == nil || node.Kind != NodeKindInternal {
		return false
	}
	return len(node.Internal) >= maxInternalKeys(h)
}

// InternalSplitResult holds the outcome of splitting a full internal node.
type InternalSplitResult struct {
	Left     *Node
	Right    *Node
	Promoted InternalEntry
}

// SplitInternalNode splits a full internal node into two nodes and a promoted separator.
func SplitInternalNode(h *Header, node *Node) (*InternalSplitResult, error) {
	if err := validateHeader(h); err != nil {
		return nil, err
	}
	if node == nil {
		return nil, fmt.Errorf("ndx: nil node")
	}
	if node.Kind != NodeKindInternal {
		return nil, fmt.Errorf("ndx: node %d is not internal", node.PageID)
	}

	capacity := maxInternalKeys(h)
	count := len(node.Internal)
	if count < capacity {
		return nil, fmt.Errorf("ndx: internal node %d is not full", node.PageID)
	}

	splitAt := count / 2
	left := &Node{
		PageID:     node.PageID,
		Kind:       NodeKindInternal,
		Internal:   cloneInternalEntries(node.Internal[:splitAt]),
		RightChild: node.Internal[splitAt].ChildPageID,
	}
	right := &Node{
		Kind:       NodeKindInternal,
		Internal:   cloneInternalEntries(node.Internal[splitAt+1:]),
		RightChild: node.RightChild,
	}
	promoted := InternalEntry{
		Key: cloneKey(node.Internal[splitAt].Key),
	}

	return &InternalSplitResult{
		Left:     left,
		Right:    right,
		Promoted: promoted,
	}, nil
}

// SplitInternalNode persists a full internal node split and returns the split outcome.
// The left half remains on node.PageID; the right half is written to a newly allocated page.
func (pm *PageManager) SplitInternalNode(node *Node) (*InternalSplitResult, error) {
	result, err := SplitInternalNode(pm.header, node)
	if err != nil {
		return nil, err
	}

	rightID, err := pm.AllocatePage()
	if err != nil {
		return nil, err
	}
	result.Right.PageID = rightID

	if err := pm.WriteNode(result.Left); err != nil {
		return nil, err
	}
	if err := pm.WriteNode(result.Right); err != nil {
		return nil, err
	}

	result.Promoted.ChildPageID = result.Left.PageID
	return result, nil
}

// SplitInternalRoot splits a full root internal node and installs a new root page.
func (pm *PageManager) SplitInternalRoot(node *Node) error {
	result, err := SplitInternalNode(pm.header, node)
	if err != nil {
		return err
	}

	rightID, err := pm.AllocatePage()
	if err != nil {
		return err
	}
	rootID, err := pm.AllocatePage()
	if err != nil {
		return err
	}

	result.Right.PageID = rightID
	if err := pm.WriteNode(result.Left); err != nil {
		return err
	}
	if err := pm.WriteNode(result.Right); err != nil {
		return err
	}

	root := &Node{
		PageID: rootID,
		Kind:   NodeKindInternal,
		Internal: []InternalEntry{{
			ChildPageID: result.Left.PageID,
			Key:         cloneKey(result.Promoted.Key),
		}},
		RightChild: rightID,
	}
	if err := pm.WriteNode(root); err != nil {
		return err
	}

	pm.header.RootPageID = rootID
	return pm.SyncHeader()
}

func cloneInternalEntries(entries []InternalEntry) []InternalEntry {
	out := make([]InternalEntry, len(entries))
	for i, entry := range entries {
		out[i] = InternalEntry{
			ChildPageID: entry.ChildPageID,
			Key:         cloneKey(entry.Key),
		}
	}
	return out
}

func cloneKey(key Key) Key {
	return append(Key(nil), key...)
}
