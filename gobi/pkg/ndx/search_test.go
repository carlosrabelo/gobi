package ndx

import (
	"encoding/binary"
	"testing"
)

func TestInternalChildPageRouting(t *testing.T) {
	h := compactTestHeader()
	node := &Node{
		Kind: NodeKindInternal,
		Internal: []InternalEntry{
			{ChildPageID: 10, Key: Key("M")},
			{ChildPageID: 20, Key: Key("S")},
		},
		RightChild: 30,
	}

	child, err := internalChildPage(h, node, Key("A"))
	if err != nil || child != 10 {
		t.Fatalf("expected child 10, got %d err=%v", child, err)
	}

	child, err = internalChildPage(h, node, Key("M"))
	if err != nil || child != 10 {
		t.Fatalf("expected child 10 for equal key, got %d", child)
	}

	child, err = internalChildPage(h, node, Key("R"))
	if err != nil || child != 20 {
		t.Fatalf("expected child 20, got %d", child)
	}

	child, err = internalChildPage(h, node, Key("Z"))
	if err != nil || child != 30 {
		t.Fatalf("expected right child 30, got %d", child)
	}
}

func TestPageNodeKindDetectsInternalTrailer(t *testing.T) {
	h := compactTestHeader()
	internal := &Node{
		Kind: NodeKindInternal,
		Internal: []InternalEntry{
			{ChildPageID: 2, Key: Key("A")},
		},
		RightChild: 3,
	}
	page, err := MarshalNodePage(h, internal)
	if err != nil {
		t.Fatalf("MarshalNodePage: %v", err)
	}
	if pageNodeKind(h, page[:]) != NodeKindInternal {
		t.Fatal("expected internal page kind")
	}

	leaf := &Node{
		Kind: NodeKindLeaf,
		Leaf: []LeafEntry{{RecordNumber: 5, Key: Key("A")}},
	}
	page, err = MarshalNodePage(h, leaf)
	if err != nil {
		t.Fatalf("MarshalNodePage: %v", err)
	}
	if pageNodeKind(h, page[:]) != NodeKindLeaf {
		t.Fatal("expected leaf page kind")
	}
}

func TestSearchLeafExact(t *testing.T) {
	h := compactTestHeader()
	node := makeLeafNode(h, 2, []LeafEntry{
		{RecordNumber: 7, Key: Key("Alice")},
		{RecordNumber: 9, Key: Key("Charlie")},
	})

	result, found, err := SearchLeafExact(h, node, Key("Alice"))
	if err != nil || !found {
		t.Fatalf("expected Alice, found=%v err=%v", found, err)
	}
	if result.RecordNumber != 7 {
		t.Fatalf("record = %d, want 7", result.RecordNumber)
	}

	_, found, err = SearchLeafExact(h, node, Key("Bob"))
	if err != nil || found {
		t.Fatalf("expected missing Bob, found=%v err=%v", found, err)
	}
}

func TestPageManagerSearchExactSingleLeaf(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	if err := pm.CreateLeafMapping(4, Key("Alice")); err != nil {
		t.Fatalf("CreateLeafMapping: %v", err)
	}
	if err := pm.InsertLeafMapping(pm.header.RootPageID, 6, Key("Bob")); err != nil {
		t.Fatalf("InsertLeafMapping: %v", err)
	}

	result, found, err := pm.SearchExact(Key("Bob"))
	if err != nil || !found {
		t.Fatalf("expected Bob, found=%v err=%v", found, err)
	}
	if result.RecordNumber != 6 {
		t.Fatalf("record = %d, want 6", result.RecordNumber)
	}

	_, found, err = pm.SearchExact(Key("Zed"))
	if err != nil || found {
		t.Fatalf("expected missing key, found=%v err=%v", found, err)
	}
}

func TestPageManagerSearchExactEmptyIndex(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	_, found, err := pm.SearchExact(Key("Alice"))
	if err != nil || found {
		t.Fatalf("expected empty index miss, found=%v err=%v", found, err)
	}
}

func TestPageManagerSearchExactInternalRoot(t *testing.T) {
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

	result, found, err := pm.SearchExact(Key("K01"))
	if err != nil || !found {
		t.Fatalf("expected K01 on left leaf, found=%v err=%v", found, err)
	}
	if result.RecordNumber != 2 {
		t.Fatalf("record = %d, want 2", result.RecordNumber)
	}

	result, found, err = pm.SearchExact(Key("K04"))
	if err != nil || !found {
		t.Fatalf("expected K04 on right leaf, found=%v err=%v", found, err)
	}
	if result.RecordNumber != 4 {
		t.Fatalf("record = %d, want 4", result.RecordNumber)
	}

	_, found, err = pm.SearchExact(Key("K02"))
	if err != nil || found {
		t.Fatalf("separator key should not appear in leaves, found=%v err=%v", found, err)
	}
}

func TestPageManagerSearchExactAfterSplitLeafRoot(t *testing.T) {
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

	_, found, err := pm.SearchExact(Key("K00"))
	if err != nil || !found {
		t.Fatalf("expected K00 after split, found=%v err=%v", found, err)
	}

	_, found, err = pm.SearchExact(Key("K04"))
	if err != nil || !found {
		t.Fatalf("expected K04 after split, found=%v err=%v", found, err)
	}
}

func TestInternalChildPageRejectsLeafNode(t *testing.T) {
	h := compactTestHeader()
	_, err := internalChildPage(h, &Node{Kind: NodeKindLeaf}, Key("A"))
	if err == nil {
		t.Fatal("expected non-internal error")
	}
}

func TestPageNodeKindEmptyLeaf(t *testing.T) {
	h := compactTestHeader()
	var page [PageSize]byte
	if pageNodeKind(h, page[:]) != NodeKindLeaf {
		t.Fatal("expected empty page to be treated as leaf")
	}
	if binary.LittleEndian.Uint16(page[0:2]) != 0 {
		t.Fatal("expected zero count")
	}
}

func TestSearchLeafExactRejectsInternalNode(t *testing.T) {
	h := compactTestHeader()
	_, _, err := SearchLeafExact(h, &Node{Kind: NodeKindInternal}, Key("A"))
	if err == nil {
		t.Fatal("expected non-leaf error")
	}
}
