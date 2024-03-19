package ndx

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestOrderedRecordNumbersSingleLeaf(t *testing.T) {
	path := filepath.Join(t.TempDir(), "small.ndx")
	header := NewHeaderForExpression("NAME", KeyTypeCharacter, 10)
	entries := []LeafEntry{
		{RecordNumber: 1, Key: Key("Carol")},
		{RecordNumber: 2, Key: Key("Alice")},
		{RecordNumber: 3, Key: Key("Bob")},
	}

	idx, err := CreateIndexFile(path, header, entries)
	if err != nil {
		t.Fatalf("CreateIndexFile: %v", err)
	}
	defer idx.Close()

	records, err := idx.Manager().OrderedRecordNumbers()
	if err != nil {
		t.Fatalf("OrderedRecordNumbers: %v", err)
	}
	want := []uint16{2, 3, 1}
	if len(records) != len(want) {
		t.Fatalf("expected %d records, got %d", len(want), len(records))
	}
	for i, rn := range want {
		if records[i] != rn {
			t.Fatalf("records = %v, want %v", records, want)
		}
	}
}

func TestOrderedRecordNumbersMultiLevelTree(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.ndx")
	header := NewHeaderForExpression("NAME", KeyTypeCharacter, 10)

	// Insert keys in reverse order; enough entries to force internal nodes.
	const total = 300
	var entries []LeafEntry
	for i := total; i >= 1; i-- {
		entries = append(entries, LeafEntry{
			RecordNumber: uint16(i),
			Key:          Key(fmt.Sprintf("K%08d", i)),
		})
	}

	idx, err := CreateIndexFile(path, header, entries)
	if err != nil {
		t.Fatalf("CreateIndexFile: %v", err)
	}
	defer idx.Close()

	records, err := idx.Manager().OrderedRecordNumbers()
	if err != nil {
		t.Fatalf("OrderedRecordNumbers: %v", err)
	}
	if len(records) != total {
		t.Fatalf("expected %d records, got %d", total, len(records))
	}
	for i, rn := range records {
		if rn != uint16(i+1) {
			t.Fatalf("records[%d] = %d, want %d", i, rn, i+1)
		}
	}
}

func TestOrderedRecordNumbersEmptyIndex(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.ndx")
	header := NewHeaderForExpression("NAME", KeyTypeCharacter, 10)

	idx, err := CreateIndexFile(path, header, nil)
	if err != nil {
		t.Fatalf("CreateIndexFile: %v", err)
	}
	defer idx.Close()

	records, err := idx.Manager().OrderedRecordNumbers()
	if err != nil {
		t.Fatalf("OrderedRecordNumbers: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected no records, got %v", records)
	}
}
