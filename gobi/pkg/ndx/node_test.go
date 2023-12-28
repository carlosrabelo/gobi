package ndx

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

func nodeHeader() *Header {
	return &Header{
		RootPageID:     2,
		PageCount:      4,
		KeyLength:      10,
		MaxKeysPerPage: 42,
		KeyType:        KeyTypeCharacter,
		Expression:     "NAME",
	}
}

func TestInternalNodeRoundTrip(t *testing.T) {
	h := nodeHeader()
	want := &Node{
		PageID: 2,
		Kind:   NodeKindInternal,
		Internal: []InternalEntry{
			{ChildPageID: 3, Key: Key("Alice")},
			{ChildPageID: 4, Key: Key("Mallory")},
		},
		RightChild: 5,
	}

	page, err := MarshalNodePage(h, want)
	if err != nil {
		t.Fatalf("MarshalNodePage: %v", err)
	}
	if got := binary.LittleEndian.Uint16(page[0:2]); got != 2 {
		t.Fatalf("key count = %d, want 2", got)
	}

	got, err := ParseNodePage(h, NodeKindInternal, page[:])
	if err != nil {
		t.Fatalf("ParseNodePage: %v", err)
	}
	if len(got.Internal) != 2 {
		t.Fatalf("expected 2 internal entries, got %d", len(got.Internal))
	}
	if got.Internal[0].ChildPageID != 3 || string(got.Internal[0].Key) != "Alice     " {
		t.Fatalf("first entry = %#v", got.Internal[0])
	}
	if got.Internal[1].ChildPageID != 4 || string(got.Internal[1].Key) != "Mallory   " {
		t.Fatalf("second entry = %#v", got.Internal[1])
	}
	if got.RightChild != 5 {
		t.Fatalf("right child = %d, want 5", got.RightChild)
	}
}

func TestLeafNodeRoundTrip(t *testing.T) {
	h := nodeHeader()
	want := &Node{
		PageID: 3,
		Kind:   NodeKindLeaf,
		Leaf: []LeafEntry{
			{RecordNumber: 1, Key: Key("Alice")},
			{RecordNumber: 5, Key: Key("Zed")},
		},
	}

	page, err := MarshalNodePage(h, want)
	if err != nil {
		t.Fatalf("MarshalNodePage: %v", err)
	}

	got, err := ParseNodePage(h, NodeKindLeaf, page[:])
	if err != nil {
		t.Fatalf("ParseNodePage: %v", err)
	}
	if len(got.Leaf) != 2 {
		t.Fatalf("expected 2 leaf entries, got %d", len(got.Leaf))
	}
	if got.Leaf[0].RecordNumber != 1 || string(got.Leaf[0].Key) != "Alice     " {
		t.Fatalf("first leaf = %#v", got.Leaf[0])
	}
	if got.Leaf[1].RecordNumber != 5 || string(got.Leaf[1].Key) != "Zed       " {
		t.Fatalf("second leaf = %#v", got.Leaf[1])
	}
}

func TestWriteReadNodeRoundTrip(t *testing.T) {
	h := nodeHeader()
	want := &Node{
		Kind:       NodeKindInternal,
		Internal:   []InternalEntry{{ChildPageID: 2, Key: Key("root")}},
		RightChild: 3,
	}

	var buf bytes.Buffer
	if err := WriteNode(&buf, h, want); err != nil {
		t.Fatalf("WriteNode: %v", err)
	}
	if buf.Len() != PageSize {
		t.Fatalf("expected %d-byte page, got %d", PageSize, buf.Len())
	}

	got, err := ReadNode(&buf, h, NodeKindInternal)
	if err != nil {
		t.Fatalf("ReadNode: %v", err)
	}
	if len(got.Internal) != 1 || got.Internal[0].ChildPageID != 2 || got.RightChild != 3 {
		t.Fatalf("unexpected node: %#v", got)
	}
}

func TestEmptyInternalNode(t *testing.T) {
	h := nodeHeader()
	node := &Node{Kind: NodeKindInternal, RightChild: 7}

	page, err := MarshalNodePage(h, node)
	if err != nil {
		t.Fatalf("MarshalNodePage: %v", err)
	}
	if binary.LittleEndian.Uint16(page[0:2]) != 0 {
		t.Fatal("expected zero key count")
	}
	if binary.LittleEndian.Uint16(page[2:4]) != 7 {
		t.Fatal("expected trailing child at offset 2")
	}

	got, err := ParseNodePage(h, NodeKindInternal, page[:])
	if err != nil {
		t.Fatalf("ParseNodePage: %v", err)
	}
	if len(got.Internal) != 0 || got.RightChild != 7 {
		t.Fatalf("unexpected empty internal node: %#v", got)
	}
}

func TestEmptyLeafNode(t *testing.T) {
	h := nodeHeader()
	node := &Node{Kind: NodeKindLeaf}

	page, err := MarshalNodePage(h, node)
	if err != nil {
		t.Fatalf("MarshalNodePage: %v", err)
	}
	got, err := ParseNodePage(h, NodeKindLeaf, page[:])
	if err != nil {
		t.Fatalf("ParseNodePage: %v", err)
	}
	if len(got.Leaf) != 0 {
		t.Fatalf("expected empty leaf, got %#v", got.Leaf)
	}
}

func TestNumericKeyNormalization(t *testing.T) {
	h := nodeHeader()
	h.KeyType = KeyTypeNumeric

	key, err := NormalizeKey(h, Key("25"))
	if err != nil {
		t.Fatalf("NormalizeKey: %v", err)
	}
	if string(key) != "25        " {
		t.Fatalf("normalized key = %q", key)
	}

	node := &Node{
		Kind: NodeKindLeaf,
		Leaf: []LeafEntry{{RecordNumber: 1, Key: Key("25")}},
	}
	page, err := MarshalNodePage(h, node)
	if err != nil {
		t.Fatalf("MarshalNodePage: %v", err)
	}
	got, err := ParseNodePage(h, NodeKindLeaf, page[:])
	if err != nil {
		t.Fatalf("ParseNodePage: %v", err)
	}
	if string(got.Leaf[0].Key) != "25        " {
		t.Fatalf("leaf key = %q", got.Leaf[0].Key)
	}
}

func TestCompareKeys(t *testing.T) {
	h := nodeHeader()
	cmp, err := CompareKeys(h, Key("Alice"), Key("Bob"))
	if err != nil {
		t.Fatalf("CompareKeys: %v", err)
	}
	if cmp >= 0 {
		t.Fatalf("expected Alice < Bob, got %d", cmp)
	}
}

func TestMaxKeysCapacity(t *testing.T) {
	h := &Header{
		KeyLength:      100,
		MaxKeysPerPage: 5,
		KeyType:        KeyTypeCharacter,
	}
	if maxLeafKeys(h) != 5 {
		t.Fatalf("max leaf keys = %d, want 5", maxLeafKeys(h))
	}
	if maxInternalKeys(h) != 4 {
		t.Fatalf("max internal keys = %d, want 4", maxInternalKeys(h))
	}

	leaf := &Node{Kind: NodeKindLeaf}
	for i := 0; i < 5; i++ {
		leaf.Leaf = append(leaf.Leaf, LeafEntry{RecordNumber: uint16(i + 1), Key: Key("x")})
	}
	if _, err := MarshalNodePage(h, leaf); err != nil {
		t.Fatalf("expected 5 leaf entries to fit: %v", err)
	}

	leaf.Leaf = append(leaf.Leaf, LeafEntry{RecordNumber: 9, Key: Key("overflow")})
	if _, err := MarshalNodePage(h, leaf); err == nil {
		t.Fatal("expected overflow error for leaf node")
	}
}

func TestParseNodeRejectsNegativeCount(t *testing.T) {
	h := nodeHeader()
	var page [PageSize]byte
	binary.LittleEndian.PutUint16(page[0:2], 0xFFFF)

	_, err := ParseNodePage(h, NodeKindLeaf, page[:])
	if err == nil {
		t.Fatal("expected negative count error")
	}
}

func TestParseNodeRejectsExcessiveCount(t *testing.T) {
	h := nodeHeader()
	var page [PageSize]byte
	binary.LittleEndian.PutUint16(page[0:2], uint16(maxLeafKeys(h)+1))

	_, err := ParseNodePage(h, NodeKindLeaf, page[:])
	if err == nil {
		t.Fatal("expected excessive count error")
	}
}

func TestParseNodeRejectsHeaderMaxKeys(t *testing.T) {
	h := nodeHeader()
	h.MaxKeysPerPage = 1
	node := &Node{
		Kind: NodeKindLeaf,
		Leaf: []LeafEntry{
			{RecordNumber: 1, Key: Key("a")},
			{RecordNumber: 2, Key: Key("b")},
		},
	}
	_, err := MarshalNodePage(h, node)
	if err == nil {
		t.Fatal("expected header max keys error")
	}
}

func TestParseNodeWrongPageSize(t *testing.T) {
	h := nodeHeader()
	_, err := ParseNodePage(h, NodeKindLeaf, make([]byte, 64))
	if err == nil {
		t.Fatal("expected wrong page size error")
	}
}

func TestMarshalNodeNilNode(t *testing.T) {
	_, err := MarshalNodePage(nodeHeader(), nil)
	if err == nil {
		t.Fatal("expected nil node error")
	}
}

func TestMarshalNodeInvalidKind(t *testing.T) {
	_, err := MarshalNodePage(nodeHeader(), &Node{Kind: NodeKind(99)})
	if err == nil {
		t.Fatal("expected invalid kind error")
	}
}

func TestNormalizeKeyRejectsLongNumeric(t *testing.T) {
	h := nodeHeader()
	h.KeyType = KeyTypeNumeric
	_, err := NormalizeKey(h, Key("12345678901"))
	if err == nil {
		t.Fatal("expected numeric key too long error")
	}
}

func TestNormalizeKeyRejectsZeroKeyLength(t *testing.T) {
	h := nodeHeader()
	h.KeyLength = 0
	_, err := NormalizeKey(h, Key("x"))
	if err == nil {
		t.Fatal("expected zero key length error")
	}
}

func TestWriteNodeWriteError(t *testing.T) {
	node := &Node{Kind: NodeKindLeaf}
	err := WriteNode(failingWriter{err: errors.New("disk full")}, nodeHeader(), node)
	if err == nil {
		t.Fatal("expected write error")
	}
}

func TestReadNodeShortPage(t *testing.T) {
	_, err := ReadNode(bytes.NewReader(make([]byte, PageSize-1)), nodeHeader(), NodeKindLeaf)
	if err == nil {
		t.Fatal("expected short page error")
	}
}

func TestCompareKeysNilHeader(t *testing.T) {
	_, err := CompareKeys(nil, Key("a"), Key("b"))
	if err == nil {
		t.Fatal("expected nil header error")
	}
}

func TestParseNodeInvalidKind(t *testing.T) {
	h := nodeHeader()
	var page [PageSize]byte
	_, err := ParseNodePage(h, NodeKind(99), page[:])
	if err == nil {
		t.Fatal("expected invalid kind error")
	}
}

func TestWriteInternalEntryOutOfRange(t *testing.T) {
	h := nodeHeader()
	page := make([]byte, PageSize)
	err := writeInternalEntry(h, page, PageSize-1, InternalEntry{ChildPageID: 1, Key: Key("x")})
	if err == nil {
		t.Fatal("expected out of range error")
	}
}

func TestWriteLeafEntryOutOfRange(t *testing.T) {
	h := nodeHeader()
	page := make([]byte, PageSize)
	err := writeLeafEntry(h, page, PageSize-1, LeafEntry{RecordNumber: 1, Key: Key("x")})
	if err == nil {
		t.Fatal("expected out of range error")
	}
}

func TestNormalizeKeyInvalidKeyType(t *testing.T) {
	h := nodeHeader()
	h.KeyType = 9
	_, err := normalizeKey(h, Key("x"))
	if err == nil {
		t.Fatal("expected invalid key type error")
	}
}

func TestValidateNodeInternalOverflow(t *testing.T) {
	h := nodeHeader()
	node := &Node{Kind: NodeKindInternal}
	for i := 0; i < maxInternalKeys(h)+1; i++ {
		node.Internal = append(node.Internal, InternalEntry{ChildPageID: 1, Key: Key("x")})
	}
	_, err := MarshalNodePage(h, node)
	if err == nil {
		t.Fatal("expected internal overflow error")
	}
}

func TestValidateNodeAgainstHeaderInvalidKind(t *testing.T) {
	h := nodeHeader()
	err := validateNodeAgainstHeader(h, &Node{Kind: NodeKind(99)})
	if err == nil {
		t.Fatal("expected invalid kind error")
	}
}

func TestCompareKeysInvalidKey(t *testing.T) {
	h := nodeHeader()
	h.KeyType = KeyTypeNumeric
	_, err := CompareKeys(h, Key("12345678901"), Key("1"))
	if err == nil {
		t.Fatal("expected invalid key error")
	}
}

func TestParseNodeInternalExcessiveCount(t *testing.T) {
	h := nodeHeader()
	var page [PageSize]byte
	binary.LittleEndian.PutUint16(page[0:2], uint16(maxInternalKeys(h)+1))
	_, err := ParseNodePage(h, NodeKindInternal, page[:])
	if err == nil {
		t.Fatal("expected excessive internal count error")
	}
}

func TestParseNodeNilHeader(t *testing.T) {
	var page [PageSize]byte
	_, err := ParseNodePage(nil, NodeKindLeaf, page[:])
	if err == nil {
		t.Fatal("expected nil header error")
	}
}

func TestMarshalNodeInternalHeaderLimit(t *testing.T) {
	h := nodeHeader()
	h.MaxKeysPerPage = 1
	node := &Node{
		Kind:     NodeKindInternal,
		Internal: []InternalEntry{{ChildPageID: 1, Key: Key("a")}, {ChildPageID: 2, Key: Key("b")}},
	}
	_, err := MarshalNodePage(h, node)
	if err == nil {
		t.Fatal("expected header max keys error")
	}
}

func TestNormalizeKeyEmptyNumeric(t *testing.T) {
	h := nodeHeader()
	h.KeyType = KeyTypeNumeric
	key, err := NormalizeKey(h, Key(""))
	if err != nil {
		t.Fatalf("NormalizeKey: %v", err)
	}
	if string(key) != "          " {
		t.Fatalf("expected blank numeric key, got %q", key)
	}
}

func TestWriteLeafEntryInvalidKey(t *testing.T) {
	h := nodeHeader()
	h.KeyType = KeyTypeNumeric
	page := make([]byte, PageSize)
	err := writeLeafEntry(h, page, 2, LeafEntry{RecordNumber: 1, Key: Key("12345678901")})
	if err == nil {
		t.Fatal("expected invalid key error")
	}
}

func TestCompareKeysInvalidFirstKey(t *testing.T) {
	h := nodeHeader()
	h.KeyType = KeyTypeNumeric
	_, err := CompareKeys(h, Key("12345678901"), Key("1"))
	if err == nil {
		t.Fatal("expected invalid first key error")
	}
}

func TestWriteInternalEntryInvalidKey(t *testing.T) {
	h := nodeHeader()
	h.KeyType = KeyTypeNumeric
	page := make([]byte, PageSize)
	err := writeInternalEntry(h, page, 2, InternalEntry{ChildPageID: 1, Key: Key("12345678901")})
	if err == nil {
		t.Fatal("expected invalid key error")
	}
}

func TestMarshalNodeInternalInvalidKey(t *testing.T) {
	h := nodeHeader()
	h.KeyType = KeyTypeNumeric
	node := &Node{
		Kind: NodeKindInternal,
		Internal: []InternalEntry{
			{ChildPageID: 1, Key: Key("10")},
			{ChildPageID: 2, Key: Key("12345678901")},
		},
		RightChild: 3,
	}
	_, err := MarshalNodePage(h, node)
	if err == nil {
		t.Fatal("expected marshal key error")
	}
}

func TestMarshalNodeLeafInvalidKey(t *testing.T) {
	h := nodeHeader()
	h.KeyType = KeyTypeNumeric
	node := &Node{
		Kind: NodeKindLeaf,
		Leaf: []LeafEntry{{RecordNumber: 1, Key: Key("12345678901")}},
	}
	_, err := MarshalNodePage(h, node)
	if err == nil {
		t.Fatal("expected marshal key error")
	}
}

func TestWriteNodeMarshalError(t *testing.T) {
	err := WriteNode(ioDiscard{}, nodeHeader(), nil)
	if err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestMarshalNodePageNilHeader(t *testing.T) {
	_, err := MarshalNodePage(nil, &Node{Kind: NodeKindLeaf})
	if err == nil {
		t.Fatal("expected nil header error")
	}
}

func TestCompareKeysInvalidSecondKey(t *testing.T) {
	h := nodeHeader()
	h.KeyType = KeyTypeNumeric
	_, err := CompareKeys(h, Key("1"), Key("12345678901"))
	if err == nil {
		t.Fatal("expected invalid second key error")
	}
}

func TestValidateNodeLeafOverflowWithoutHeaderMax(t *testing.T) {
	h := nodeHeader()
	h.MaxKeysPerPage = 0
	node := &Node{Kind: NodeKindLeaf}
	for i := 0; i < maxLeafKeys(h)+1; i++ {
		node.Leaf = append(node.Leaf, LeafEntry{RecordNumber: 1, Key: Key("x")})
	}
	_, err := MarshalNodePage(h, node)
	if err == nil {
		t.Fatal("expected leaf overflow error")
	}
}

func TestParseNodeRejectsHeaderMaxOnRead(t *testing.T) {
	h := nodeHeader()
	h.MaxKeysPerPage = 1
	node := &Node{
		Kind: NodeKindLeaf,
		Leaf: []LeafEntry{
			{RecordNumber: 1, Key: Key("a")},
			{RecordNumber: 2, Key: Key("b")},
		},
	}
	page, err := MarshalNodePage(h, node)
	if err == nil {
		t.Fatal("expected marshal header max error")
	}
	h.MaxKeysPerPage = 2
	page, err = MarshalNodePage(h, node)
	if err != nil {
		t.Fatalf("MarshalNodePage: %v", err)
	}
	h.MaxKeysPerPage = 1
	_, err = ParseNodePage(h, NodeKindLeaf, page[:])
	if err == nil {
		t.Fatal("expected parse header max error")
	}
}
