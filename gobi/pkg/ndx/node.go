package ndx

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// NodeKind identifies whether a page stores internal separators or leaf mappings.
type NodeKind int

const (
	// NodeKindInternal marks a page containing child page pointers and separator keys.
	NodeKindInternal NodeKind = iota
	// NodeKindLeaf marks a page mapping indexed keys to DBF record numbers.
	NodeKindLeaf
)

// Key is a fixed-width indexed value stored in NDX pages.
type Key []byte

// InternalEntry is one separator in an internal B-Tree node.
type InternalEntry struct {
	ChildPageID uint16
	Key         Key
}

// LeafEntry maps an indexed key to a DBF record number.
type LeafEntry struct {
	RecordNumber uint16
	Key          Key
}

// Node is an in-memory B-Tree page.
type Node struct {
	PageID     uint16
	Kind       NodeKind
	Internal   []InternalEntry
	RightChild uint16
	Leaf       []LeafEntry
}

// ReadNode reads a 512-byte node page of kind from r.
func ReadNode(r io.Reader, h *Header, kind NodeKind) (*Node, error) {
	var page [PageSize]byte
	if _, err := io.ReadFull(r, page[:]); err != nil {
		return nil, fmt.Errorf("ndx: reading node page: %w", err)
	}
	return ParseNodePage(h, kind, page[:])
}

// WriteNode writes node as a 512-byte page to w.
func WriteNode(w io.Writer, h *Header, node *Node) error {
	page, err := MarshalNodePage(h, node)
	if err != nil {
		return err
	}
	if _, err := w.Write(page[:]); err != nil {
		return fmt.Errorf("ndx: writing node page: %w", err)
	}
	return nil
}

// ParseNodePage decodes a 512-byte page into an in-memory node.
func ParseNodePage(h *Header, kind NodeKind, page []byte) (*Node, error) {
	if err := validateHeader(h); err != nil {
		return nil, err
	}
	if len(page) != PageSize {
		return nil, fmt.Errorf("ndx: node page must be %d bytes", PageSize)
	}

	count := int16(binary.LittleEndian.Uint16(page[0:2]))
	if count < 0 {
		return nil, fmt.Errorf("ndx: negative key count %d", count)
	}

	node := &Node{Kind: kind}
	offset := 2

	switch kind {
	case NodeKindInternal:
		if int(count) > maxInternalKeys(h) {
			return nil, fmt.Errorf("ndx: internal key count %d exceeds page capacity", count)
		}
		node.Internal = make([]InternalEntry, count)
		for i := 0; i < int(count); i++ {
			node.Internal[i], offset = readInternalEntry(h, page, offset)
		}
		node.RightChild = binary.LittleEndian.Uint16(page[offset : offset+2])
	case NodeKindLeaf:
		if int(count) > maxLeafKeys(h) {
			return nil, fmt.Errorf("ndx: leaf key count %d exceeds page capacity", count)
		}
		node.Leaf = make([]LeafEntry, count)
		for i := 0; i < int(count); i++ {
			node.Leaf[i], offset = readLeafEntry(h, page, offset)
		}
	default:
		return nil, fmt.Errorf("ndx: invalid node kind %d", kind)
	}

	if err := validateNodeAgainstHeader(h, node); err != nil {
		return nil, err
	}
	return node, nil
}

// MarshalNodePage encodes node into a 512-byte page.
func MarshalNodePage(h *Header, node *Node) ([PageSize]byte, error) {
	var page [PageSize]byte
	if err := validateHeader(h); err != nil {
		return page, err
	}
	if node == nil {
		return page, fmt.Errorf("ndx: nil node")
	}
	if err := validateNodeAgainstHeader(h, node); err != nil {
		return page, err
	}

	switch node.Kind {
	case NodeKindInternal:
		count := len(node.Internal)
		binary.LittleEndian.PutUint16(page[0:2], uint16(count))
		offset := 2
		for i, entry := range node.Internal {
			if err := writeInternalEntry(h, page[:], offset, entry); err != nil {
				return page, fmt.Errorf("ndx: internal entry %d: %w", i, err)
			}
			offset += entrySize(h)
		}
		binary.LittleEndian.PutUint16(page[offset:offset+2], node.RightChild)
	case NodeKindLeaf:
		count := len(node.Leaf)
		binary.LittleEndian.PutUint16(page[0:2], uint16(count))
		offset := 2
		for i, entry := range node.Leaf {
			if err := writeLeafEntry(h, page[:], offset, entry); err != nil {
				return page, fmt.Errorf("ndx: leaf entry %d: %w", i, err)
			}
			offset += entrySize(h)
		}
	default:
		return page, fmt.Errorf("ndx: invalid node kind %d", node.Kind)
	}

	return page, nil
}

// CompareKeys orders two fixed-width keys according to h.KeyType.
func CompareKeys(h *Header, a, b Key) (int, error) {
	if err := validateHeader(h); err != nil {
		return 0, err
	}
	ka, err := normalizeKey(h, a)
	if err != nil {
		return 0, err
	}
	kb, err := normalizeKey(h, b)
	if err != nil {
		return 0, err
	}
	return bytes.Compare(ka, kb), nil
}

// NormalizeKey returns a fixed-width copy of key padded for h.KeyType.
func NormalizeKey(h *Header, key Key) (Key, error) {
	return normalizeKey(h, key)
}

func entryPayloadSize(h *Header) int {
	return 2 + int(h.KeyLength)
}

func maxLeafKeys(h *Header) int {
	return (PageSize - 2) / entryPayloadSize(h)
}

func maxInternalKeys(h *Header) int {
	return (PageSize - 2 - 2) / entryPayloadSize(h)
}

func validateNodeAgainstHeader(h *Header, node *Node) error {
	if h.MaxKeysPerPage == 0 {
		return nil
	}
	var count int
	switch node.Kind {
	case NodeKindInternal:
		count = len(node.Internal)
		if count > maxInternalKeys(h) {
			return fmt.Errorf("ndx: internal key count %d exceeds page capacity", count)
		}
	case NodeKindLeaf:
		count = len(node.Leaf)
		if count > maxLeafKeys(h) {
			return fmt.Errorf("ndx: leaf key count %d exceeds page capacity", count)
		}
	default:
		return fmt.Errorf("ndx: invalid node kind %d", node.Kind)
	}
	if uint16(count) > h.MaxKeysPerPage {
		return fmt.Errorf("ndx: key count %d exceeds header max %d", count, h.MaxKeysPerPage)
	}
	return nil
}

func readInternalEntry(h *Header, page []byte, offset int) (InternalEntry, int) {
	entry := InternalEntry{
		ChildPageID: binary.LittleEndian.Uint16(page[offset : offset+2]),
		Key:         append(Key(nil), page[offset+2:offset+2+int(h.KeyLength)]...),
	}
	return entry, offset + entrySize(h)
}

func readLeafEntry(h *Header, page []byte, offset int) (LeafEntry, int) {
	entry := LeafEntry{
		RecordNumber: binary.LittleEndian.Uint16(page[offset : offset+2]),
		Key:          append(Key(nil), page[offset+2:offset+2+int(h.KeyLength)]...),
	}
	return entry, offset + entrySize(h)
}

func writeInternalEntry(h *Header, page []byte, offset int, entry InternalEntry) error {
	if offset+entrySize(h) > len(page) {
		return fmt.Errorf("ndx: internal entry out of range")
	}
	key, err := normalizeKey(h, entry.Key)
	if err != nil {
		return err
	}
	binary.LittleEndian.PutUint16(page[offset:offset+2], entry.ChildPageID)
	copy(page[offset+2:offset+2+int(h.KeyLength)], key)
	return nil
}

func writeLeafEntry(h *Header, page []byte, offset int, entry LeafEntry) error {
	if offset+entrySize(h) > len(page) {
		return fmt.Errorf("ndx: leaf entry out of range")
	}
	key, err := normalizeKey(h, entry.Key)
	if err != nil {
		return err
	}
	binary.LittleEndian.PutUint16(page[offset:offset+2], entry.RecordNumber)
	copy(page[offset+2:offset+2+int(h.KeyLength)], key)
	return nil
}

func normalizeKey(h *Header, key Key) (Key, error) {
	if h.KeyLength == 0 {
		return nil, fmt.Errorf("ndx: key length must be greater than zero")
	}
	out := make([]byte, h.KeyLength)
	switch h.KeyType {
	case KeyTypeCharacter:
		copy(out, key)
		for i := len(key); i < int(h.KeyLength); i++ {
			out[i] = ' '
		}
	case KeyTypeNumeric:
		trimmed := bytes.TrimSpace(key)
		if len(trimmed) == 0 {
			for i := range out {
				out[i] = ' '
			}
			return Key(out), nil
		}
		if len(trimmed) > int(h.KeyLength) {
			return nil, fmt.Errorf("ndx: numeric key exceeds key length")
		}
		copy(out, trimmed)
		for i := len(trimmed); i < int(h.KeyLength); i++ {
			out[i] = ' '
		}
	default:
		return nil, fmt.Errorf("ndx: invalid key type %d", h.KeyType)
	}
	return Key(out), nil
}

func entrySize(h *Header) int {
	return entryPayloadSize(h)
}
