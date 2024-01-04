package ndx

import (
	"errors"
	"fmt"
	"testing"
)

func compactTestHeader() *Header {
	return &Header{
		KeyLength:      100,
		MaxKeysPerPage: 5,
		KeyType:        KeyTypeCharacter,
		Expression:     "NAME",
	}
}

func makeFullInternalNode(h *Header, pageID uint16, count int) *Node {
	node := &Node{
		PageID:     pageID,
		Kind:       NodeKindInternal,
		RightChild: 99,
	}
	for i := 0; i < count; i++ {
		node.Internal = append(node.Internal, InternalEntry{
			ChildPageID: uint16(i + 1),
			Key:         Key(fmt.Sprintf("K%02d", i)),
		})
	}
	return node
}

func TestInternalNodeFull(t *testing.T) {
	h := compactTestHeader()
	node := makeFullInternalNode(h, 2, maxInternalKeys(h)-1)
	if InternalNodeFull(h, node) {
		t.Fatal("expected node below capacity not to be full")
	}

	node.Internal = append(node.Internal, InternalEntry{ChildPageID: 9, Key: Key("K99")})
	if !InternalNodeFull(h, node) {
		t.Fatal("expected full internal node")
	}
}

func TestSplitInternalNodeDividesKeys(t *testing.T) {
	h := compactTestHeader()
	full := makeFullInternalNode(h, 5, maxInternalKeys(h))

	result, err := SplitInternalNode(h, full)
	if err != nil {
		t.Fatalf("SplitInternalNode: %v", err)
	}

	if len(result.Left.Internal) != 2 {
		t.Fatalf("left key count = %d, want 2", len(result.Left.Internal))
	}
	if len(result.Right.Internal) != 1 {
		t.Fatalf("right key count = %d, want 1", len(result.Right.Internal))
	}
	if result.Left.PageID != 5 {
		t.Fatalf("left page id = %d, want 5", result.Left.PageID)
	}
	if result.Left.RightChild != full.Internal[2].ChildPageID {
		t.Fatalf("left right child = %d, want %d", result.Left.RightChild, full.Internal[2].ChildPageID)
	}
	if result.Right.RightChild != full.RightChild {
		t.Fatalf("right right child = %d, want %d", result.Right.RightChild, full.RightChild)
	}
	if string(result.Promoted.Key) != string(full.Internal[2].Key) {
		t.Fatalf("promoted key = %q, want %q", result.Promoted.Key, full.Internal[2].Key)
	}
}

func TestSplitInternalNodePreservesAllKeys(t *testing.T) {
	h := compactTestHeader()
	full := makeFullInternalNode(h, 5, maxInternalKeys(h))

	result, err := SplitInternalNode(h, full)
	if err != nil {
		t.Fatalf("SplitInternalNode: %v", err)
	}

	var got []string
	for _, entry := range result.Left.Internal {
		got = append(got, string(entry.Key))
	}
	got = append(got, string(result.Promoted.Key))
	for _, entry := range result.Right.Internal {
		got = append(got, string(entry.Key))
	}

	for i, entry := range full.Internal {
		if got[i] != string(entry.Key) {
			t.Fatalf("key order mismatch at %d: got %q want %q", i, got[i], entry.Key)
		}
	}
}

func TestSplitInternalNodeRejectsNonFullNode(t *testing.T) {
	h := compactTestHeader()
	node := makeFullInternalNode(h, 2, maxInternalKeys(h)-1)
	if _, err := SplitInternalNode(h, node); err == nil {
		t.Fatal("expected not full error")
	}
}

func TestSplitInternalNodeRejectsLeafNode(t *testing.T) {
	h := compactTestHeader()
	node := &Node{PageID: 2, Kind: NodeKindLeaf, Leaf: []LeafEntry{{RecordNumber: 1, Key: Key("A")}}}
	if _, err := SplitInternalNode(h, node); err == nil {
		t.Fatal("expected non-internal error")
	}
}

func TestPageManagerSplitInternalNodePersistsPages(t *testing.T) {
	h := newTestHeader()
	file := &ndxFile{}
	pm, err := CreatePageManager(file, h)
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	pageID, err := pm.AllocatePage()
	if err != nil {
		t.Fatalf("AllocatePage: %v", err)
	}

	full := makeFullInternalNode(h, pageID, maxInternalKeys(h))
	if err := pm.WriteNode(full); err != nil {
		t.Fatalf("WriteNode: %v", err)
	}

	result, err := pm.SplitInternalNode(full)
	if err != nil {
		t.Fatalf("SplitInternalNode: %v", err)
	}
	if result.Promoted.ChildPageID != pageID {
		t.Fatalf("promoted child = %d, want %d", result.Promoted.ChildPageID, pageID)
	}

	left, err := pm.ReadNode(pageID, NodeKindInternal)
	if err != nil {
		t.Fatalf("ReadNode left: %v", err)
	}
	if len(left.Internal) != maxInternalKeys(h)/2 {
		t.Fatalf("left keys = %d, want %d", len(left.Internal), maxInternalKeys(h)/2)
	}

	right, err := pm.ReadNode(result.Right.PageID, NodeKindInternal)
	if err != nil {
		t.Fatalf("ReadNode right: %v", err)
	}
	if len(left.Internal)+1+len(right.Internal) != maxInternalKeys(h) {
		t.Fatalf("split key count mismatch")
	}
}

func TestPageManagerSplitInternalRootInstallsNewRoot(t *testing.T) {
	h := newTestHeader()
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

	full := makeFullInternalNode(h, rootID, maxInternalKeys(h))
	if err := pm.WriteNode(full); err != nil {
		t.Fatalf("WriteNode: %v", err)
	}

	oldRoot := rootID
	if err := pm.SplitInternalRoot(full); err != nil {
		t.Fatalf("SplitInternalRoot: %v", err)
	}
	if pm.header.RootPageID == oldRoot {
		t.Fatal("expected new root page id")
	}

	root, err := pm.ReadNode(pm.header.RootPageID, NodeKindInternal)
	if err != nil {
		t.Fatalf("ReadNode root: %v", err)
	}
	if len(root.Internal) != 1 {
		t.Fatalf("expected single separator in new root, got %d", len(root.Internal))
	}
	if root.Internal[0].ChildPageID != oldRoot {
		t.Fatalf("root left child = %d, want %d", root.Internal[0].ChildPageID, oldRoot)
	}
	if root.RightChild == 0 || root.RightChild == oldRoot {
		t.Fatalf("unexpected root right child %d", root.RightChild)
	}
}

func TestInternalNodeFullNilInputs(t *testing.T) {
	h := compactTestHeader()
	if InternalNodeFull(nil, &Node{Kind: NodeKindInternal}) {
		t.Fatal("expected false for nil header")
	}
	if InternalNodeFull(h, nil) {
		t.Fatal("expected false for nil node")
	}
	if InternalNodeFull(h, &Node{Kind: NodeKindLeaf}) {
		t.Fatal("expected false for leaf node")
	}
}

func TestPageManagerSplitInternalNodeErrors(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	if _, err := pm.SplitInternalNode(&Node{PageID: 1, Kind: NodeKindInternal}); err == nil {
		t.Fatal("expected not full error")
	}
}

func TestPageManagerSplitInternalRootWriteError(t *testing.T) {
	h := newTestHeader()
	pm := &PageManager{
		file:   failingWriteSeeker{writeErr: errors.New("write failed")},
		header: h,
	}
	full := makeFullInternalNode(h, 1, maxInternalKeys(h))
	if err := pm.SplitInternalRoot(full); err == nil {
		t.Fatal("expected split root error")
	}
}

func TestSplitInternalNodeNilNode(t *testing.T) {
	_, err := SplitInternalNode(compactTestHeader(), nil)
	if err == nil {
		t.Fatal("expected nil node error")
	}
}

func TestSplitInternalNodeNilHeader(t *testing.T) {
	node := makeFullInternalNode(compactTestHeader(), 1, 4)
	_, err := SplitInternalNode(nil, node)
	if err == nil {
		t.Fatal("expected nil header error")
	}
}
