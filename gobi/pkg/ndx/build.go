package ndx

import (
	"fmt"
	"io"
	"sort"
)

// Index is an open NDX index file bound to a DBF work area.
type Index struct {
	Path string
	file io.ReadWriteSeeker
	pm   *PageManager
}

// Manager returns the page manager for index operations.
func (idx *Index) Manager() *PageManager {
	if idx == nil {
		return nil
	}
	return idx.pm
}

// Close closes the underlying index file.
func (idx *Index) Close() error {
	if idx == nil || idx.file == nil {
		return nil
	}
	if closer, ok := idx.file.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

type builtNode struct {
	node   *Node
	minKey Key
}

// BuildFromEntries constructs a balanced B-Tree from sorted leaf mappings.
func (pm *PageManager) BuildFromEntries(entries []LeafEntry) error {
	if pm == nil || pm.header == nil {
		return fmt.Errorf("ndx: nil page manager")
	}
	if err := validateHeader(pm.header); err != nil {
		return err
	}

	normalized := make([]LeafEntry, len(entries))
	for i, entry := range entries {
		key, err := normalizeKey(pm.header, entry.Key)
		if err != nil {
			return fmt.Errorf("ndx: entry %d: %w", i, err)
		}
		normalized[i] = LeafEntry{
			RecordNumber: entry.RecordNumber,
			Key:          key,
		}
	}

	sort.SliceStable(normalized, func(i, j int) bool {
		cmp, err := CompareKeys(pm.header, normalized[i].Key, normalized[j].Key)
		if err != nil {
			return false
		}
		if cmp != 0 {
			return cmp < 0
		}
		return normalized[i].RecordNumber < normalized[j].RecordNumber
	})

	if len(normalized) == 0 {
		pm.header.RootPageID = 0
		return pm.SyncHeader()
	}

	root, err := pm.buildTree(normalized)
	if err != nil {
		return err
	}
	pm.header.RootPageID = root.node.PageID
	return pm.SyncHeader()
}

// CreateIndexFile creates path, builds entries into a new index, and returns an open handle.
func CreateIndexFile(path string, h *Header, entries []LeafEntry) (*Index, error) {
	if h == nil {
		return nil, fmt.Errorf("ndx: nil header")
	}
	if err := validateHeader(h); err != nil {
		return nil, err
	}

	header := *h
	if header.MaxKeysPerPage == 0 {
		header.MaxKeysPerPage = uint16(maxLeafKeys(&header))
	}

	file, err := openIndexFile(path)
	if err != nil {
		return nil, err
	}

	pm, err := CreatePageManager(file, &header)
	if err != nil {
		closeIndexFile(file)
		return nil, err
	}
	if err := pm.BuildFromEntries(entries); err != nil {
		closeIndexFile(file)
		return nil, err
	}

	return &Index{Path: path, file: file, pm: pm}, nil
}

// RebuildIndex truncates path and rebuilds the open index with entries.
func RebuildIndex(idx *Index, h *Header, entries []LeafEntry) error {
	if idx == nil {
		return fmt.Errorf("ndx: nil index")
	}
	if h == nil {
		return fmt.Errorf("ndx: nil header")
	}
	path := idx.Path
	if path == "" {
		return fmt.Errorf("ndx: index path is empty")
	}
	if err := idx.Close(); err != nil {
		return err
	}

	rebuilt, err := CreateIndexFile(path, h, entries)
	if err != nil {
		return err
	}

	idx.Path = rebuilt.Path
	idx.file = rebuilt.file
	idx.pm = rebuilt.pm
	return nil
}

func (pm *PageManager) buildTree(entries []LeafEntry) (*builtNode, error) {
	leafCapacity := maxLeafKeys(pm.header)
	if len(entries) <= leafCapacity {
		return pm.materializeLeaf(entries)
	}

	var leaves []*builtNode
	for start := 0; start < len(entries); {
		end := start + leafCapacity
		if end > len(entries) {
			end = len(entries)
		}
		leaf, err := pm.materializeLeaf(entries[start:end])
		if err != nil {
			return nil, err
		}
		leaves = append(leaves, leaf)
		start = end
	}
	return pm.buildInternalTree(leaves)
}

func (pm *PageManager) buildInternalTree(children []*builtNode) (*builtNode, error) {
	maxFanout := maxInternalKeys(pm.header) + 1
	if len(children) <= maxFanout {
		return pm.materializeInternal(children)
	}

	var parents []*builtNode
	for start := 0; start < len(children); {
		end := start + maxFanout
		if end > len(children) {
			end = len(children)
		}
		parent, err := pm.materializeInternal(children[start:end])
		if err != nil {
			return nil, err
		}
		parents = append(parents, parent)
		start = end
	}
	return pm.buildInternalTree(parents)
}

func (pm *PageManager) materializeLeaf(entries []LeafEntry) (*builtNode, error) {
	pageID, err := pm.AllocatePage()
	if err != nil {
		return nil, err
	}
	node := &Node{
		PageID: pageID,
		Kind:   NodeKindLeaf,
		Leaf:   cloneLeafEntries(entries),
	}
	if err := pm.WriteNode(node); err != nil {
		return nil, err
	}
	return &builtNode{node: node, minKey: cloneKey(entries[0].Key)}, nil
}

func (pm *PageManager) materializeInternal(children []*builtNode) (*builtNode, error) {
	if len(children) == 0 {
		return nil, fmt.Errorf("ndx: internal node requires children")
	}

	pageID, err := pm.AllocatePage()
	if err != nil {
		return nil, err
	}

	node := &Node{
		PageID: pageID,
		Kind:   NodeKindInternal,
	}
	for i := 0; i < len(children)-1; i++ {
		node.Internal = append(node.Internal, InternalEntry{
			ChildPageID: children[i].node.PageID,
			Key:         cloneKey(children[i+1].minKey),
		})
	}
	node.RightChild = children[len(children)-1].node.PageID
	if err := pm.WriteNode(node); err != nil {
		return nil, err
	}
	return &builtNode{node: node, minKey: cloneKey(children[0].minKey)}, nil
}
