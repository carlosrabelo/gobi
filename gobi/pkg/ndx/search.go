package ndx

import (
	"encoding/binary"
	"fmt"
)

// pageNodeKind detects whether a raw page stores internal separators or leaf mappings.
// Internal pages write a trailing RightChild pointer after their entries; leaf pages do not.
func pageNodeKind(h *Header, page []byte) NodeKind {
	count := int(binary.LittleEndian.Uint16(page[0:2]))
	offset := 2 + count*entrySize(h)
	if offset+2 <= PageSize && binary.LittleEndian.Uint16(page[offset:offset+2]) != 0 {
		return NodeKindInternal
	}
	return NodeKindLeaf
}

// SearchResult holds an exact key lookup outcome.
type SearchResult struct {
	RecordNumber uint16
	Key          Key
}

// internalChildPage returns the child page to follow for key in an internal node.
func internalChildPage(h *Header, node *Node, key Key) (uint16, error) {
	_, child, err := internalChildRoute(h, node, key)
	return child, err
}

// treePathFrame records one internal node visited while descending the B-Tree.
type treePathFrame struct {
	pageID   uint16
	childIdx int
}

// internalChildRoute returns the child index and page to follow for key.
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

// SearchLeafExact finds an exact key in a single leaf node.
func SearchLeafExact(h *Header, node *Node, key Key) (SearchResult, bool, error) {
	if err := validateHeader(h); err != nil {
		return SearchResult{}, false, err
	}
	if node == nil || node.Kind != NodeKindLeaf {
		return SearchResult{}, false, fmt.Errorf("ndx: not a leaf node")
	}
	entry, found, err := LeafEntryForKey(h, node, key)
	if err != nil || !found {
		return SearchResult{}, found, err
	}
	return SearchResult{RecordNumber: entry.RecordNumber, Key: entry.Key}, true, nil
}

// SearchExact locates key in the index and returns its record mapping.
func (pm *PageManager) SearchExact(key Key) (SearchResult, bool, error) {
	if pm.header.RootPageID == 0 {
		return SearchResult{}, false, nil
	}
	return pm.searchPage(pm.header.RootPageID, key)
}

func (pm *PageManager) searchPage(pageID uint16, key Key) (SearchResult, bool, error) {
	page, err := pm.ReadPage(pageID)
	if err != nil {
		return SearchResult{}, false, err
	}

	switch pageNodeKind(pm.header, page[:]) {
	case NodeKindLeaf:
		node, err := ParseNodePage(pm.header, NodeKindLeaf, page[:])
		if err != nil {
			return SearchResult{}, false, err
		}
		return SearchLeafExact(pm.header, node, key)
	case NodeKindInternal:
		node, err := ParseNodePage(pm.header, NodeKindInternal, page[:])
		if err != nil {
			return SearchResult{}, false, err
		}
		_, child, err := internalChildRoute(pm.header, node, key)
		if err != nil {
			return SearchResult{}, false, err
		}
		return pm.searchPage(child, key)
	default:
		return SearchResult{}, false, fmt.Errorf("ndx: invalid page kind at page %d", pageID)
	}
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
