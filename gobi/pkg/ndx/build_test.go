package ndx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildFromEntriesSingleLeaf(t *testing.T) {
	file := &ndxFile{}
	h := newTestHeader()
	pm, err := CreatePageManager(file, h)
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}

	entries := []LeafEntry{
		{RecordNumber: 2, Key: Key("Bob")},
		{RecordNumber: 1, Key: Key("Alice")},
	}
	if err := pm.BuildFromEntries(entries); err != nil {
		t.Fatalf("BuildFromEntries: %v", err)
	}
	if pm.header.RootPageID == 0 {
		t.Fatal("expected root page")
	}

	result, found, err := pm.SearchExact(Key("Alice"))
	if err != nil || !found || result.RecordNumber != 1 {
		t.Fatalf("Alice = %+v found=%v err=%v", result, found, err)
	}
}

func TestBuildFromEntriesInternalRoot(t *testing.T) {
	h := compactTestHeader()
	file := &ndxFile{}
	pm, err := CreatePageManager(file, h)
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}

	var entries []LeafEntry
	for i := 0; i < maxLeafKeys(h)+2; i++ {
		entries = append(entries, LeafEntry{
			RecordNumber: uint16(i + 1),
			Key:          Key(string(rune('A' + i))),
		})
	}
	if err := pm.BuildFromEntries(entries); err != nil {
		t.Fatalf("BuildFromEntries: %v", err)
	}

	page, err := pm.ReadPage(pm.header.RootPageID)
	if err != nil {
		t.Fatalf("ReadPage root: %v", err)
	}
	if pageNodeKind(h, page[:]) != NodeKindInternal {
		t.Fatal("expected internal root")
	}

	_, found, err := pm.SearchExact(Key(string(rune('A' + maxLeafKeys(h) + 1))))
	if err != nil || !found {
		t.Fatalf("expected last key, found=%v err=%v", found, err)
	}
}

func TestCreateIndexFilePersistsEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "people.ndx")
	h := NewHeaderForExpression("NAME", KeyTypeCharacter, 10)

	entries := []LeafEntry{
		{RecordNumber: 1, Key: Key("Alice")},
		{RecordNumber: 2, Key: Key("Bob")},
	}
	idx, err := CreateIndexFile(path, h, entries)
	if err != nil {
		t.Fatalf("CreateIndexFile: %v", err)
	}
	defer idx.Close()

	reopened, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer reopened.Close()

	pm, err := OpenPageManager(reopened)
	if err != nil {
		t.Fatalf("OpenPageManager: %v", err)
	}
	if pm.Header().Expression != "NAME" {
		t.Fatalf("expression = %q", pm.Header().Expression)
	}
	_, found, err := pm.SearchExact(Key("Bob"))
	if err != nil || !found {
		t.Fatalf("expected Bob in rebuilt file, found=%v err=%v", found, err)
	}
}

func TestKeyFromTextCharacterAndNumeric(t *testing.T) {
	charHeader := &Header{KeyLength: 10, KeyType: KeyTypeCharacter, Expression: "NAME"}
	key, err := KeyFromText(charHeader, "Alice")
	if err != nil {
		t.Fatalf("KeyFromText: %v", err)
	}
	if strings.TrimRight(string(key), " ") != "Alice" {
		t.Fatalf("key = %q", key)
	}

	numHeader := &Header{KeyLength: 8, KeyType: KeyTypeNumeric, Expression: "AGE"}
	key, err = KeyFromText(numHeader, " 25 ")
	if err != nil {
		t.Fatalf("KeyFromText numeric: %v", err)
	}
	if strings.TrimSpace(string(key)) != "25" {
		t.Fatalf("numeric key = %q", key)
	}
}

func TestBuildFromEntriesEmptyClearsRoot(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	pm.header.RootPageID = 3
	if err := pm.BuildFromEntries(nil); err != nil {
		t.Fatalf("BuildFromEntries: %v", err)
	}
	if pm.header.RootPageID != 0 {
		t.Fatalf("root = %d, want 0", pm.header.RootPageID)
	}
}

func TestRebuildIndexReplacesTree(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "people.ndx")
	h := NewHeaderForExpression("NAME", KeyTypeCharacter, 10)

	idx, err := CreateIndexFile(path, h, []LeafEntry{
		{RecordNumber: 1, Key: Key("Alice")},
	})
	if err != nil {
		t.Fatalf("CreateIndexFile: %v", err)
	}

	newHeader := NewHeaderForExpression("NAME", KeyTypeCharacter, 10)
	if err := RebuildIndex(idx, newHeader, []LeafEntry{
		{RecordNumber: 1, Key: Key("Alice")},
		{RecordNumber: 2, Key: Key("Bob")},
	}); err != nil {
		t.Fatalf("RebuildIndex: %v", err)
	}
	defer idx.Close()

	_, found, err := idx.Manager().SearchExact(Key("Bob"))
	if err != nil || !found {
		t.Fatalf("expected Bob after rebuild, found=%v err=%v", found, err)
	}
}
