package ndx

import (
	"errors"
	"fmt"
)

// InsertMapping adds recNo/key to the index, splitting the root leaf when needed.
func (pm *PageManager) InsertMapping(recNo uint16, key Key) error {
	if pm == nil || pm.header == nil {
		return fmt.Errorf("ndx: nil page manager")
	}

	norm, err := normalizeKey(pm.header, key)
	if err != nil {
		return err
	}

	if pm.header.RootPageID == 0 {
		return pm.CreateLeafMapping(recNo, norm)
	}

	page, err := pm.ReadPage(pm.header.RootPageID)
	if err != nil {
		return err
	}

	switch pageNodeKind(pm.header, page[:]) {
	case NodeKindLeaf:
		return pm.insertMappingLeafRoot(recNo, norm)
	case NodeKindInternal:
		_, leaf, err := pm.descendToLeaf(pm.header.RootPageID, norm)
		if err != nil {
			return err
		}
		if err := InsertLeafEntry(pm.header, leaf, recNo, norm); err != nil {
			if errors.Is(err, ErrLeafFull) {
				return fmt.Errorf("ndx: leaf split propagation not implemented")
			}
			return err
		}
		return pm.WriteNode(leaf)
	default:
		return fmt.Errorf("ndx: invalid root page kind")
	}
}

func (pm *PageManager) insertMappingLeafRoot(recNo uint16, key Key) error {
	node, err := pm.ReadNode(pm.header.RootPageID, NodeKindLeaf)
	if err != nil {
		return err
	}
	if !LeafNodeFull(pm.header, node) {
		return pm.InsertLeafMapping(pm.header.RootPageID, recNo, key)
	}

	result, err := SplitLeafNode(pm.header, node)
	if err != nil {
		return err
	}

	target, err := leafInsertTarget(pm.header, result, key)
	if err != nil {
		return err
	}
	if err := InsertLeafEntry(pm.header, target, recNo, key); err != nil {
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
