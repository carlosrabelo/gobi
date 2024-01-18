package ndx

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenIndexReadsExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "people.ndx")
	header := NewHeaderForExpression("NAME", KeyTypeCharacter, 10)
	entries := []LeafEntry{
		{RecordNumber: 1, Key: Key("Alice")},
		{RecordNumber: 2, Key: Key("Bob")},
	}

	created, err := CreateIndexFile(path, header, entries)
	if err != nil {
		t.Fatalf("CreateIndexFile: %v", err)
	}
	if err := created.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	idx, err := OpenIndex(path)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer idx.Close()

	if idx.Path != path {
		t.Fatalf("Path = %q, want %q", idx.Path, path)
	}
	got := idx.Manager().Header()
	if got.Expression != "NAME" || got.KeyType != KeyTypeCharacter {
		t.Fatalf("unexpected header: %#v", got)
	}

	result, found, err := idx.Manager().SearchPrefix(Key("Bob"))
	if err != nil {
		t.Fatalf("SearchPrefix: %v", err)
	}
	if !found || result.RecordNumber != 2 {
		t.Fatalf("expected record 2, got found=%v result=%#v", found, result)
	}
}

func TestOpenIndexMissingFileFails(t *testing.T) {
	_, err := OpenIndex(filepath.Join(t.TempDir(), "missing.ndx"))
	if err == nil {
		t.Fatal("expected error for missing index file")
	}
	if !strings.Contains(err.Error(), "opening index file") {
		t.Fatalf("unexpected error: %v", err)
	}
}
