package ndx

import (
	"encoding/binary"
	"fmt"
	"strings"
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

// KeyHasPrefix reports whether key begins with prefix for character indexes.
func KeyHasPrefix(h *Header, key, prefix Key) (bool, error) {
	if err := validateHeader(h); err != nil {
		return false, err
	}
	if h.KeyType != KeyTypeCharacter {
		return false, fmt.Errorf("ndx: prefix search requires character keys")
	}
	prefixText := strings.TrimRight(string(prefix), " ")
	if prefixText == "" {
		return true, nil
	}
	keyText := strings.TrimRight(string(key), " ")
	return strings.HasPrefix(keyText, prefixText), nil
}

func leafLowerBound(h *Header, entries []LeafEntry, key Key) (int, error) {
	for i, entry := range entries {
		cmp, err := CompareKeys(h, entry.Key, key)
		if err != nil {
			return 0, err
		}
		if cmp >= 0 {
			return i, nil
		}
	}
	return len(entries), nil
}

// SearchLeafPrefix returns the first leaf mapping whose key starts with prefix.
func SearchLeafPrefix(h *Header, node *Node, prefix Key) (SearchResult, bool, error) {
	if err := validateHeader(h); err != nil {
		return SearchResult{}, false, err
	}
	if node == nil || node.Kind != NodeKindLeaf {
		return SearchResult{}, false, fmt.Errorf("ndx: not a leaf node")
	}

	norm, err := normalizeKey(h, prefix)
	if err != nil {
		return SearchResult{}, false, err
	}
	start, err := leafLowerBound(h, node.Leaf, norm)
	if err != nil {
		return SearchResult{}, false, err
	}

	for i := start; i < len(node.Leaf); i++ {
		entry := node.Leaf[i]
		match, err := KeyHasPrefix(h, entry.Key, prefix)
		if err != nil {
			return SearchResult{}, false, err
		}
		if match {
			return SearchResult{RecordNumber: entry.RecordNumber, Key: entry.Key}, true, nil
		}
	}
	return SearchResult{}, false, nil
}

// SearchPrefix locates the first indexed key that starts with prefix.
func (pm *PageManager) SearchPrefix(prefix Key) (SearchResult, bool, error) {
	if pm.header.RootPageID == 0 {
		return SearchResult{}, false, nil
	}

	path, leaf, err := pm.descendToLeaf(pm.header.RootPageID, prefix)
	if err != nil {
		return SearchResult{}, false, err
	}

	for {
		result, found, err := SearchLeafPrefix(pm.header, leaf, prefix)
		if err != nil || found {
			return result, found, err
		}

		nextPageID, nextPath, ok, err := pm.nextLeafPage(path)
		if err != nil || !ok {
			return SearchResult{}, false, err
		}
		path = nextPath
		leaf, err = pm.readLeafPage(nextPageID)
		if err != nil {
			return SearchResult{}, false, err
		}
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

func (pm *PageManager) readLeafPage(pageID uint16) (*Node, error) {
	page, err := pm.ReadPage(pageID)
	if err != nil {
		return nil, err
	}
	if pageNodeKind(pm.header, page[:]) != NodeKindLeaf {
		return nil, fmt.Errorf("ndx: page %d is not a leaf", pageID)
	}
	node, err := ParseNodePage(pm.header, NodeKindLeaf, page[:])
	if err != nil {
		return nil, err
	}
	node.PageID = pageID
	return node, nil
}

func (pm *PageManager) leftmostLeaf(pageID uint16) (uint16, error) {
	for {
		page, err := pm.ReadPage(pageID)
		if err != nil {
			return 0, err
		}
		if pageNodeKind(pm.header, page[:]) == NodeKindLeaf {
			return pageID, nil
		}
		node, err := ParseNodePage(pm.header, NodeKindInternal, page[:])
		if err != nil {
			return 0, err
		}
		if len(node.Internal) == 0 {
			return 0, fmt.Errorf("ndx: empty internal node at page %d", pageID)
		}
		pageID = node.Internal[0].ChildPageID
	}
}

func internalNextChild(node *Node, childIdx int) (uint16, bool) {
	if childIdx < len(node.Internal) {
		if childIdx+1 < len(node.Internal) {
			return node.Internal[childIdx+1].ChildPageID, true
		}
		return node.RightChild, true
	}
	return 0, false
}

func (pm *PageManager) nextLeafPage(path []treePathFrame) (uint16, []treePathFrame, bool, error) {
	for len(path) > 0 {
		frame := path[len(path)-1]
		path = path[:len(path)-1]

		node, err := pm.ReadNode(frame.pageID, NodeKindInternal)
		if err != nil {
			return 0, nil, false, err
		}
		nextChild, ok := internalNextChild(node, frame.childIdx)
		if !ok {
			continue
		}

		leafPage, err := pm.leftmostLeaf(nextChild)
		if err != nil {
			return 0, nil, false, err
		}
		return leafPage, append(path, treePathFrame{pageID: frame.pageID, childIdx: frame.childIdx + 1}), true, nil
	}
	return 0, nil, false, nil
}
