package ndx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenIndexReadsExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "people.ndx")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := CreatePageManager(file, newTestHeader()); err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	if err := file.Close(); err != nil {
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
