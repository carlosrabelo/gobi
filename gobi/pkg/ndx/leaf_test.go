package ndx

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func makeLeafNode(h *Header, pageID uint16, entries []LeafEntry) *Node {
	node := &Node{
		PageID: pageID,
		Kind:   NodeKindLeaf,
		Leaf:   cloneLeafEntries(entries),
	}
	return node
}

func TestLeafNodeFull(t *testing.T) {
	h := compactTestHeader()
	node := makeLeafNode(h, 2, nil)
	for i := 0; i < maxLeafKeys(h)-1; i++ {
		node.Leaf = append(node.Leaf, LeafEntry{RecordNumber: uint16(i + 1), Key: Key(fmt.Sprintf("K%d", i))})
	}
	if LeafNodeFull(h, node) {
		t.Fatal("expected leaf below capacity")
	}
	node.Leaf = append(node.Leaf, LeafEntry{RecordNumber: 99, Key: Key("K99")})
	if !LeafNodeFull(h, node) {
		t.Fatal("expected full leaf")
	}
}

func TestInsertLeafEntryKeepsSortedOrder(t *testing.T) {
	h := compactTestHeader()
	node := makeLeafNode(h, 2, []LeafEntry{
		{RecordNumber: 1, Key: Key("Alice")},
		{RecordNumber: 3, Key: Key("Charlie")},
	})

	if err := InsertLeafEntry(h, node, 2, Key("Bob")); err != nil {
		t.Fatalf("InsertLeafEntry: %v", err)
	}
	if len(node.Leaf) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(node.Leaf))
	}
	if strings.TrimRight(string(node.Leaf[1].Key), " ") != "Bob" {
		t.Fatalf("middle key = %q", node.Leaf[1].Key)
	}
	if node.Leaf[1].RecordNumber != 2 {
		t.Fatalf("middle record = %d, want 2", node.Leaf[1].RecordNumber)
	}
}

func TestInsertLeafEntryDuplicateKeys(t *testing.T) {
	h := compactTestHeader()
	node := makeLeafNode(h, 2, []LeafEntry{
		{RecordNumber: 1, Key: Key("Alice")},
		{RecordNumber: 2, Key: Key("Bob")},
	})
	if err := InsertLeafEntry(h, node, 4, Key("Alice")); err != nil {
		t.Fatalf("InsertLeafEntry: %v", err)
	}
	if node.Leaf[1].RecordNumber != 4 || strings.TrimRight(string(node.Leaf[1].Key), " ") != "Alice" {
		t.Fatalf("duplicate key not inserted after equal keys: %#v", node.Leaf[1])
	}
}

func TestInsertLeafEntryRejectsFullNode(t *testing.T) {
	h := compactTestHeader()
	node := makeFullLeafNode(h, 2)
	err := InsertLeafEntry(h, node, 9, Key("ZZZ"))
	if !errors.Is(err, ErrLeafFull) {
		t.Fatalf("expected ErrLeafFull, got %v", err)
	}
}

func TestSplitLeafNodePreservesMappings(t *testing.T) {
	h := compactTestHeader()
	full := makeFullLeafNode(h, 5)

	result, err := SplitLeafNode(h, full)
	if err != nil {
		t.Fatalf("SplitLeafNode: %v", err)
	}
	if len(result.Left.Leaf) != 2 {
		t.Fatalf("left count = %d, want 2", len(result.Left.Leaf))
	}
	if len(result.Right.Leaf) != 2 {
		t.Fatalf("right count = %d, want 2", len(result.Right.Leaf))
	}
	if string(result.Promoted) != string(full.Leaf[2].Key) {
		t.Fatalf("promoted = %q, want %q", result.Promoted, full.Leaf[2].Key)
	}

	var got []string
	for _, entry := range result.Left.Leaf {
		got = append(got, string(entry.Key))
	}
	got = append(got, string(result.Promoted))
	for _, entry := range result.Right.Leaf {
		got = append(got, string(entry.Key))
	}
	for i, entry := range full.Leaf {
		if got[i] != string(entry.Key) {
			t.Fatalf("key order mismatch at %d: got %q want %q", i, got[i], entry.Key)
		}
	}
}

func TestLeafEntryForKey(t *testing.T) {
	h := compactTestHeader()
	node := makeLeafNode(h, 2, []LeafEntry{
		{RecordNumber: 10, Key: Key("Alice")},
		{RecordNumber: 20, Key: Key("Charlie")},
	})

	entry, ok, err := LeafEntryForKey(h, node, Key("Alice"))
	if err != nil || !ok {
		t.Fatalf("expected Alice mapping, ok=%v err=%v", ok, err)
	}
	if entry.RecordNumber != 10 {
		t.Fatalf("record = %d, want 10", entry.RecordNumber)
	}

	_, ok, err = LeafEntryForKey(h, node, Key("Bob"))
	if err != nil || ok {
		t.Fatalf("expected missing Bob, ok=%v err=%v", ok, err)
	}
}

func TestPageManagerCreateLeafMapping(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	if err := pm.CreateLeafMapping(3, Key("Alice")); err != nil {
		t.Fatalf("CreateLeafMapping: %v", err)
	}
	if pm.header.RootPageID == 0 {
		t.Fatal("expected root page")
	}

	root, err := pm.ReadNode(pm.header.RootPageID, NodeKindLeaf)
	if err != nil {
		t.Fatalf("ReadNode: %v", err)
	}
	entry, ok, err := LeafEntryForKey(pm.header, root, Key("Alice"))
	if err != nil || !ok {
		t.Fatalf("expected mapping, ok=%v err=%v", ok, err)
	}
	if entry.RecordNumber != 3 {
		t.Fatalf("record = %d, want 3", entry.RecordNumber)
	}
}

func TestPageManagerInsertLeafMapping(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	if err := pm.CreateLeafMapping(1, Key("Alice")); err != nil {
		t.Fatalf("CreateLeafMapping: %v", err)
	}
	rootID := pm.header.RootPageID

	if err := pm.InsertLeafMapping(rootID, 2, Key("Bob")); err != nil {
		t.Fatalf("InsertLeafMapping: %v", err)
	}

	root, err := pm.ReadNode(rootID, NodeKindLeaf)
	if err != nil {
		t.Fatalf("ReadNode: %v", err)
	}
	if len(root.Leaf) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(root.Leaf))
	}
}

func TestPageManagerAppendLeafMappingSplitsFullLeaf(t *testing.T) {
	h := compactTestHeader()
	file := &ndxFile{}
	pm, err := CreatePageManager(file, h)
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	pageID, err := pm.AllocatePage()
	if err != nil {
		t.Fatalf("AllocatePage: %v", err)
	}
	pm.header.RootPageID = pageID
	if err := pm.SyncHeader(); err != nil {
		t.Fatalf("SyncHeader: %v", err)
	}

	node := makeFullLeafNode(h, pageID)
	if err := pm.WriteNode(node); err != nil {
		t.Fatalf("WriteNode: %v", err)
	}

	split, err := pm.AppendLeafMapping(pageID, 99, Key("K99"))
	if err != nil {
		t.Fatalf("AppendLeafMapping: %v", err)
	}
	if split == nil {
		t.Fatal("expected split result")
	}

	left, err := pm.ReadNode(split.Left.PageID, NodeKindLeaf)
	if err != nil {
		t.Fatalf("ReadNode left: %v", err)
	}
	right, err := pm.ReadNode(split.Right.PageID, NodeKindLeaf)
	if err != nil {
		t.Fatalf("ReadNode right: %v", err)
	}
	if len(left.Leaf)+len(right.Leaf) != maxLeafKeys(h) {
		t.Fatalf("unexpected entry count after split insert: left=%d right=%d", len(left.Leaf), len(right.Leaf))
	}
}

func TestPageManagerSplitLeafRoot(t *testing.T) {
	h := compactTestHeader()
	file := &ndxFile{}
	pm, err := CreatePageManager(file, h)
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	rootID, err := pm.AllocatePage()
	if err != nil {
		t.Fatalf("AllocatePage: %v", err)
	}
	pm.header.RootPageID = rootID
	if err := pm.SyncHeader(); err != nil {
		t.Fatalf("SyncHeader: %v", err)
	}

	full := makeFullLeafNode(h, rootID)
	if err := pm.WriteNode(full); err != nil {
		t.Fatalf("WriteNode: %v", err)
	}

	if err := pm.SplitLeafRoot(full); err != nil {
		t.Fatalf("SplitLeafRoot: %v", err)
	}
	if pm.header.RootPageID == rootID {
		t.Fatal("expected new internal root")
	}

	root, err := pm.ReadNode(pm.header.RootPageID, NodeKindInternal)
	if err != nil {
		t.Fatalf("ReadNode root: %v", err)
	}
	if len(root.Internal) != 1 {
		t.Fatalf("expected one separator, got %d", len(root.Internal))
	}
	if root.Internal[0].ChildPageID != rootID {
		t.Fatalf("left child = %d, want %d", root.Internal[0].ChildPageID, rootID)
	}
}

func makeFullLeafNode(h *Header, pageID uint16) *Node {
	node := &Node{PageID: pageID, Kind: NodeKindLeaf}
	for i := 0; i < maxLeafKeys(h); i++ {
		node.Leaf = append(node.Leaf, LeafEntry{
			RecordNumber: uint16(i + 1),
			Key:          Key(fmt.Sprintf("K%02d", i)),
		})
	}
	return node
}

func TestInsertLeafEntryRejectsInternalNode(t *testing.T) {
	h := compactTestHeader()
	node := &Node{PageID: 2, Kind: NodeKindInternal}
	if err := InsertLeafEntry(h, node, 1, Key("A")); err == nil {
		t.Fatal("expected non-leaf error")
	}
}

func TestSplitLeafNodeRejectsNonFullNode(t *testing.T) {
	h := compactTestHeader()
	node := makeLeafNode(h, 2, []LeafEntry{{RecordNumber: 1, Key: Key("A")}})
	if _, err := SplitLeafNode(h, node); err == nil {
		t.Fatal("expected not full error")
	}
}

func TestCreateLeafMappingRejectsExistingRoot(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	if err := pm.CreateLeafMapping(1, Key("A")); err != nil {
		t.Fatalf("CreateLeafMapping: %v", err)
	}
	if err := pm.CreateLeafMapping(2, Key("B")); err == nil {
		t.Fatal("expected existing root error")
	}
}

func TestLeafNodeFullNilInputs(t *testing.T) {
	h := compactTestHeader()
	if LeafNodeFull(nil, &Node{Kind: NodeKindLeaf}) {
		t.Fatal("expected false for nil header")
	}
	if LeafNodeFull(h, nil) {
		t.Fatal("expected false for nil node")
	}
}

func TestDeleteLeafEntryRemovesFirstMatch(t *testing.T) {
	h := compactTestHeader()
	node := makeLeafNode(h, 2, []LeafEntry{
		{RecordNumber: 1, Key: Key("Alice")},
		{RecordNumber: 2, Key: Key("Bob")},
		{RecordNumber: 3, Key: Key("Charlie")},
	})

	removed, err := DeleteLeafEntry(h, node, Key("Bob"))
	if err != nil {
		t.Fatalf("DeleteLeafEntry: %v", err)
	}
	if removed.RecordNumber != 2 {
		t.Fatalf("removed record = %d, want 2", removed.RecordNumber)
	}
	if len(node.Leaf) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(node.Leaf))
	}
	if strings.TrimRight(string(node.Leaf[0].Key), " ") != "Alice" {
		t.Fatalf("first key = %q", node.Leaf[0].Key)
	}
	if strings.TrimRight(string(node.Leaf[1].Key), " ") != "Charlie" {
		t.Fatalf("second key = %q", node.Leaf[1].Key)
	}
}

func TestDeleteLeafEntryDuplicateKeysRemovesFirstOnly(t *testing.T) {
	h := compactTestHeader()
	node := makeLeafNode(h, 2, []LeafEntry{
		{RecordNumber: 1, Key: Key("Alice")},
		{RecordNumber: 2, Key: Key("Alice")},
		{RecordNumber: 3, Key: Key("Bob")},
	})

	removed, err := DeleteLeafEntry(h, node, Key("Alice"))
	if err != nil {
		t.Fatalf("DeleteLeafEntry: %v", err)
	}
	if removed.RecordNumber != 1 {
		t.Fatalf("removed record = %d, want 1", removed.RecordNumber)
	}
	if len(node.Leaf) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(node.Leaf))
	}
	if node.Leaf[0].RecordNumber != 2 {
		t.Fatalf("remaining Alice record = %d, want 2", node.Leaf[0].RecordNumber)
	}
}

func TestDeleteLeafEntryMissingKey(t *testing.T) {
	h := compactTestHeader()
	node := makeLeafNode(h, 2, []LeafEntry{{RecordNumber: 1, Key: Key("Alice")}})
	_, err := DeleteLeafEntry(h, node, Key("Bob"))
	if !errors.Is(err, ErrLeafKeyNotFound) {
		t.Fatalf("expected ErrLeafKeyNotFound, got %v", err)
	}
}

func TestDeleteLeafEntryRejectsInternalNode(t *testing.T) {
	h := compactTestHeader()
	_, err := DeleteLeafEntry(h, &Node{PageID: 2, Kind: NodeKindInternal}, Key("A"))
	if err == nil {
		t.Fatal("expected non-leaf error")
	}
}

func TestPageManagerDeleteLeafMapping(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	if err := pm.CreateLeafMapping(1, Key("Alice")); err != nil {
		t.Fatalf("CreateLeafMapping: %v", err)
	}
	if err := pm.InsertLeafMapping(pm.header.RootPageID, 2, Key("Bob")); err != nil {
		t.Fatalf("InsertLeafMapping: %v", err)
	}

	removed, err := pm.DeleteLeafMapping(pm.header.RootPageID, Key("Alice"))
	if err != nil {
		t.Fatalf("DeleteLeafMapping: %v", err)
	}
	if removed.RecordNumber != 1 {
		t.Fatalf("removed record = %d, want 1", removed.RecordNumber)
	}

	root, err := pm.ReadNode(pm.header.RootPageID, NodeKindLeaf)
	if err != nil {
		t.Fatalf("ReadNode: %v", err)
	}
	if len(root.Leaf) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(root.Leaf))
	}
}

func TestPageManagerDeleteMappingSingleLeaf(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	if err := pm.CreateLeafMapping(1, Key("Alice")); err != nil {
		t.Fatalf("CreateLeafMapping: %v", err)
	}
	if err := pm.InsertLeafMapping(pm.header.RootPageID, 2, Key("Bob")); err != nil {
		t.Fatalf("InsertLeafMapping: %v", err)
	}

	removed, err := pm.DeleteMapping(Key("Bob"))
	if err != nil {
		t.Fatalf("DeleteMapping: %v", err)
	}
	if removed.RecordNumber != 2 {
		t.Fatalf("removed record = %d, want 2", removed.RecordNumber)
	}

	_, found, err := pm.SearchExact(Key("Bob"))
	if err != nil || found {
		t.Fatalf("expected Bob removed, found=%v err=%v", found, err)
	}
	_, found, err = pm.SearchExact(Key("Alice"))
	if err != nil || !found {
		t.Fatalf("expected Alice to remain, found=%v err=%v", found, err)
	}
}

func TestPageManagerDeleteMappingClearsEmptyRoot(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	if err := pm.CreateLeafMapping(1, Key("Alice")); err != nil {
		t.Fatalf("CreateLeafMapping: %v", err)
	}

	if _, err := pm.DeleteMapping(Key("Alice")); err != nil {
		t.Fatalf("DeleteMapping: %v", err)
	}
	if pm.header.RootPageID != 0 {
		t.Fatalf("root page = %d, want 0", pm.header.RootPageID)
	}

	_, found, err := pm.SearchExact(Key("Alice"))
	if err != nil || found {
		t.Fatalf("expected empty index, found=%v err=%v", found, err)
	}
}

func TestPageManagerDeleteMappingInternalRoot(t *testing.T) {
	h := compactTestHeader()
	file := &ndxFile{}
	pm, err := CreatePageManager(file, h)
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}

	rootID, err := pm.AllocatePage()
	if err != nil {
		t.Fatalf("AllocatePage root: %v", err)
	}
	leftID, err := pm.AllocatePage()
	if err != nil {
		t.Fatalf("AllocatePage left: %v", err)
	}
	rightID, err := pm.AllocatePage()
	if err != nil {
		t.Fatalf("AllocatePage right: %v", err)
	}

	left := makeLeafNode(h, leftID, []LeafEntry{
		{RecordNumber: 1, Key: Key("K00")},
		{RecordNumber: 2, Key: Key("K01")},
	})
	right := makeLeafNode(h, rightID, []LeafEntry{
		{RecordNumber: 3, Key: Key("K03")},
		{RecordNumber: 4, Key: Key("K04")},
	})
	root := &Node{
		PageID: rootID,
		Kind:   NodeKindInternal,
		Internal: []InternalEntry{{
			ChildPageID: leftID,
			Key:         Key("K02"),
		}},
		RightChild: rightID,
	}

	for _, node := range []*Node{left, right, root} {
		if err := pm.WriteNode(node); err != nil {
			t.Fatalf("WriteNode: %v", err)
		}
	}
	pm.header.RootPageID = rootID
	if err := pm.SyncHeader(); err != nil {
		t.Fatalf("SyncHeader: %v", err)
	}

	removed, err := pm.DeleteMapping(Key("K04"))
	if err != nil {
		t.Fatalf("DeleteMapping: %v", err)
	}
	if removed.RecordNumber != 4 {
		t.Fatalf("removed record = %d, want 4", removed.RecordNumber)
	}

	_, found, err := pm.SearchExact(Key("K04"))
	if err != nil || found {
		t.Fatalf("expected K04 removed, found=%v err=%v", found, err)
	}
	_, found, err = pm.SearchExact(Key("K01"))
	if err != nil || !found {
		t.Fatalf("expected K01 to remain, found=%v err=%v", found, err)
	}
}

func TestPageManagerDeleteMappingMissingKey(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	if err := pm.CreateLeafMapping(1, Key("Alice")); err != nil {
		t.Fatalf("CreateLeafMapping: %v", err)
	}
	_, err = pm.DeleteMapping(Key("Bob"))
	if !errors.Is(err, ErrLeafKeyNotFound) {
		t.Fatalf("expected ErrLeafKeyNotFound, got %v", err)
	}
}
