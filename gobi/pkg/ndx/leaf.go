package ndx

import "fmt"

// ErrLeafFull indicates a leaf node has no room for another entry.
var ErrLeafFull = fmt.Errorf("ndx: leaf node is full")

// ErrLeafKeyNotFound indicates the requested key is absent from a leaf node.
var ErrLeafKeyNotFound = fmt.Errorf("ndx: leaf key not found")

// LeafSplitResult holds the outcome of splitting a full leaf node.
type LeafSplitResult struct {
	Left     *Node
	Right    *Node
	Promoted Key
}

// LeafNodeFull reports whether node has reached its leaf page capacity.
func LeafNodeFull(h *Header, node *Node) bool {
	if h == nil || node == nil || node.Kind != NodeKindLeaf {
		return false
	}
	return len(node.Leaf) >= maxLeafKeys(h)
}

// InsertLeafEntry inserts a sorted key/record mapping into a leaf node.
func InsertLeafEntry(h *Header, node *Node, recNo uint16, key Key) error {
	if err := validateHeader(h); err != nil {
		return err
	}
	if node == nil {
		return fmt.Errorf("ndx: nil node")
	}
	if node.Kind != NodeKindLeaf {
		return fmt.Errorf("ndx: node %d is not a leaf", node.PageID)
	}
	if LeafNodeFull(h, node) {
		return ErrLeafFull
	}

	norm, err := normalizeKey(h, key)
	if err != nil {
		return err
	}
	idx, err := leafInsertIndex(h, node.Leaf, norm)
	if err != nil {
		return err
	}

	entry := LeafEntry{
		RecordNumber: recNo,
		Key:          norm,
	}
	node.Leaf = append(node.Leaf, LeafEntry{})
	copy(node.Leaf[idx+1:], node.Leaf[idx:])
	node.Leaf[idx] = entry
	return nil
}

// SplitLeafNode splits a full leaf node into two nodes and a promoted separator key.
func SplitLeafNode(h *Header, node *Node) (*LeafSplitResult, error) {
	if err := validateHeader(h); err != nil {
		return nil, err
	}
	if node == nil {
		return nil, fmt.Errorf("ndx: nil node")
	}
	if node.Kind != NodeKindLeaf {
		return nil, fmt.Errorf("ndx: node %d is not a leaf", node.PageID)
	}

	capacity := maxLeafKeys(h)
	count := len(node.Leaf)
	if count < capacity {
		return nil, fmt.Errorf("ndx: leaf node %d is not full", node.PageID)
	}

	splitAt := count / 2
	left := &Node{
		PageID: node.PageID,
		Kind:   NodeKindLeaf,
		Leaf:   cloneLeafEntries(node.Leaf[:splitAt]),
	}
	right := &Node{
		Kind: NodeKindLeaf,
		Leaf: cloneLeafEntries(node.Leaf[splitAt+1:]),
	}

	return &LeafSplitResult{
		Left:     left,
		Right:    right,
		Promoted: cloneKey(node.Leaf[splitAt].Key),
	}, nil
}

// LeafEntryForKey returns the first mapping for key in node.
func LeafEntryForKey(h *Header, node *Node, key Key) (LeafEntry, bool, error) {
	if err := validateHeader(h); err != nil {
		return LeafEntry{}, false, err
	}
	if node == nil || node.Kind != NodeKindLeaf {
		return LeafEntry{}, false, fmt.Errorf("ndx: not a leaf node")
	}

	norm, err := normalizeKey(h, key)
	if err != nil {
		return LeafEntry{}, false, err
	}
	for _, entry := range node.Leaf {
		cmp, err := CompareKeys(h, entry.Key, norm)
		if err != nil {
			return LeafEntry{}, false, err
		}
		if cmp == 0 {
			return entry, true, nil
		}
		if cmp > 0 {
			break
		}
	}
	return LeafEntry{}, false, nil
}

// DeleteLeafEntry removes the first mapping for key from node.
func DeleteLeafEntry(h *Header, node *Node, key Key) (LeafEntry, error) {
	if err := validateHeader(h); err != nil {
		return LeafEntry{}, err
	}
	if node == nil {
		return LeafEntry{}, fmt.Errorf("ndx: nil node")
	}
	if node.Kind != NodeKindLeaf {
		return LeafEntry{}, fmt.Errorf("ndx: node %d is not a leaf", node.PageID)
	}

	idx, err := leafKeyIndex(h, node.Leaf, key)
	if err != nil {
		return LeafEntry{}, err
	}
	if idx < 0 {
		return LeafEntry{}, ErrLeafKeyNotFound
	}

	removed := node.Leaf[idx]
	node.Leaf = append(node.Leaf[:idx], node.Leaf[idx+1:]...)
	return removed, nil
}

// SplitLeafNode persists a full leaf split and returns the split outcome.
func (pm *PageManager) SplitLeafNode(node *Node) (*LeafSplitResult, error) {
	result, err := SplitLeafNode(pm.header, node)
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
	return result, nil
}

// SplitLeafRoot splits a full root leaf and installs a new internal root page.
func (pm *PageManager) SplitLeafRoot(node *Node) error {
	result, err := SplitLeafNode(pm.header, node)
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
			Key:         cloneKey(result.Promoted),
		}},
		RightChild: rightID,
	}
	if err := pm.WriteNode(root); err != nil {
		return err
	}

	pm.header.RootPageID = rootID
	return pm.SyncHeader()
}

// CreateLeafMapping allocates the first leaf page and stores recNo/key.
func (pm *PageManager) CreateLeafMapping(recNo uint16, key Key) error {
	if pm.header.RootPageID != 0 {
		return fmt.Errorf("ndx: index already has a root page")
	}

	pageID, err := pm.AllocatePage()
	if err != nil {
		return err
	}
	node := &Node{
		PageID: pageID,
		Kind:   NodeKindLeaf,
	}
	if err := InsertLeafEntry(pm.header, node, recNo, key); err != nil {
		return err
	}
	if err := pm.WriteNode(node); err != nil {
		return err
	}

	pm.header.RootPageID = pageID
	return pm.SyncHeader()
}

// InsertLeafMapping inserts recNo/key into the leaf at pageID.
func (pm *PageManager) InsertLeafMapping(pageID uint16, recNo uint16, key Key) error {
	node, err := pm.ReadNode(pageID, NodeKindLeaf)
	if err != nil {
		return err
	}
	if err := InsertLeafEntry(pm.header, node, recNo, key); err != nil {
		return err
	}
	return pm.WriteNode(node)
}

// AppendLeafMapping inserts into pageID, splitting the page first when it is full.
func (pm *PageManager) AppendLeafMapping(pageID uint16, recNo uint16, key Key) (*LeafSplitResult, error) {
	node, err := pm.ReadNode(pageID, NodeKindLeaf)
	if err != nil {
		return nil, err
	}
	if !LeafNodeFull(pm.header, node) {
		if err := InsertLeafEntry(pm.header, node, recNo, key); err != nil {
			return nil, err
		}
		if err := pm.WriteNode(node); err != nil {
			return nil, err
		}
		return nil, nil
	}

	result, err := pm.SplitLeafNode(node)
	if err != nil {
		return nil, err
	}

	target, err := leafInsertTarget(pm.header, result, key)
	if err != nil {
		return nil, err
	}
	if err := InsertLeafEntry(pm.header, target, recNo, key); err != nil {
		return nil, err
	}
	if err := pm.WriteNode(target); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteLeafMapping removes key from the leaf at pageID.
func (pm *PageManager) DeleteLeafMapping(pageID uint16, key Key) (LeafEntry, error) {
	node, err := pm.ReadNode(pageID, NodeKindLeaf)
	if err != nil {
		return LeafEntry{}, err
	}
	removed, err := DeleteLeafEntry(pm.header, node, key)
	if err != nil {
		return LeafEntry{}, err
	}
	if err := pm.WriteNode(node); err != nil {
		return LeafEntry{}, err
	}
	return removed, nil
}

// DeleteMapping removes the first index entry for key and clears an empty root index.
func (pm *PageManager) DeleteMapping(key Key) (LeafEntry, error) {
	if pm.header.RootPageID == 0 {
		return LeafEntry{}, ErrLeafKeyNotFound
	}

	_, leaf, err := pm.descendToLeaf(pm.header.RootPageID, key)
	if err != nil {
		return LeafEntry{}, err
	}

	removed, err := DeleteLeafEntry(pm.header, leaf, key)
	if err != nil {
		return LeafEntry{}, err
	}

	if len(leaf.Leaf) == 0 && leaf.PageID == pm.header.RootPageID {
		pm.header.RootPageID = 0
		if err := pm.SyncHeader(); err != nil {
			return LeafEntry{}, err
		}
		return removed, nil
	}

	if err := pm.WriteNode(leaf); err != nil {
		return LeafEntry{}, err
	}
	return removed, nil
}

func leafInsertIndex(h *Header, entries []LeafEntry, key Key) (int, error) {
	for i, entry := range entries {
		cmp, err := CompareKeys(h, entry.Key, key)
		if err != nil {
			return 0, err
		}
		if cmp > 0 {
			return i, nil
		}
	}
	return len(entries), nil
}

func leafKeyIndex(h *Header, entries []LeafEntry, key Key) (int, error) {
	norm, err := normalizeKey(h, key)
	if err != nil {
		return 0, err
	}
	for i, entry := range entries {
		cmp, err := CompareKeys(h, entry.Key, norm)
		if err != nil {
			return 0, err
		}
		if cmp == 0 {
			return i, nil
		}
		if cmp > 0 {
			break
		}
	}
	return -1, nil
}

func leafInsertTarget(h *Header, split *LeafSplitResult, key Key) (*Node, error) {
	cmp, err := CompareKeys(h, key, split.Promoted)
	if err != nil {
		return nil, err
	}
	if cmp <= 0 {
		return split.Left, nil
	}
	return split.Right, nil
}

func cloneLeafEntries(entries []LeafEntry) []LeafEntry {
	out := make([]LeafEntry, len(entries))
	for i, entry := range entries {
		out[i] = LeafEntry{
			RecordNumber: entry.RecordNumber,
			Key:          cloneKey(entry.Key),
		}
	}
	return out
}
