package ndx

import (
	"encoding/binary"
	"fmt"
)

func pageNodeKind(h *Header, page []byte) NodeKind {
	count := int(binary.LittleEndian.Uint16(page[0:2]))
	offset := 2 + count*entrySize(h)
	if offset+2 <= PageSize && binary.LittleEndian.Uint16(page[offset:offset+2]) != 0 {
		return NodeKindInternal
	}
	return NodeKindLeaf
}

type treePathFrame struct {
	pageID   uint16
	childIdx int
}

func internalChildRoute(h *Header, node *Node, key Key) (int, uint16, error) {
	if node == nil || node.Kind != NodeKindInternal {
		return 0, 0, fmt.Errorf("ndx: not an internal node")
	}

	norm, err := normalizeKey(h, key)
	if err != nil {
		return 0, 0, err
	}

	next := node.RightChild
	childIdx := len(node.Internal)
	for i, entry := range node.Internal {
		cmp, err := CompareKeys(h, norm, entry.Key)
		if err != nil {
			return 0, 0, err
		}
		if cmp <= 0 {
			next = entry.ChildPageID
			childIdx = i
			break
		}
	}
	if next == 0 {
		return 0, 0, fmt.Errorf("ndx: invalid child page id")
	}
	return childIdx, next, nil
}

func (pm *PageManager) descendToLeaf(pageID uint16, key Key) ([]treePathFrame, *Node, error) {
	var path []treePathFrame
	for {
		page, err := pm.ReadPage(pageID)
		if err != nil {
			return nil, nil, err
		}

		switch pageNodeKind(pm.header, page[:]) {
		case NodeKindLeaf:
			node, err := ParseNodePage(pm.header, NodeKindLeaf, page[:])
			if err != nil {
				return nil, nil, err
			}
			node.PageID = pageID
			return path, node, nil
		case NodeKindInternal:
			node, err := ParseNodePage(pm.header, NodeKindInternal, page[:])
			if err != nil {
				return nil, nil, err
			}
			childIdx, childPage, err := internalChildRoute(pm.header, node, key)
			if err != nil {
				return nil, nil, err
			}
			path = append(path, treePathFrame{pageID: pageID, childIdx: childIdx})
			pageID = childPage
		default:
			return nil, nil, fmt.Errorf("ndx: invalid page kind at page %d", pageID)
		}
	}
}
