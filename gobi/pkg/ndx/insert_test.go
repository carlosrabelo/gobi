package ndx

import "testing"

func TestInsertMappingEmptyIndex(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}

	if err := pm.InsertMapping(1, Key("Alice")); err != nil {
		t.Fatalf("InsertMapping: %v", err)
	}

	_, found, err := pm.SearchExact(Key("Alice"))
	if err != nil || !found {
		t.Fatalf("expected Alice, found=%v err=%v", found, err)
	}
}

func TestInsertMappingLeafRoot(t *testing.T) {
	file := &ndxFile{}
	pm, err := CreatePageManager(file, newTestHeader())
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}
	if err := pm.CreateLeafMapping(1, Key("Alice")); err != nil {
		t.Fatalf("CreateLeafMapping: %v", err)
	}
	if err := pm.InsertMapping(2, Key("Bob")); err != nil {
		t.Fatalf("InsertMapping: %v", err)
	}

	_, found, err := pm.SearchExact(Key("Bob"))
	if err != nil || !found {
		t.Fatalf("expected Bob, found=%v err=%v", found, err)
	}
}

func TestInsertMappingInternalRoot(t *testing.T) {
	h := compactTestHeader()
	file := &ndxFile{}
	pm, err := CreatePageManager(file, h)
	if err != nil {
		t.Fatalf("CreatePageManager: %v", err)
	}

	var entries []LeafEntry
	for i := 0; i < maxLeafKeys(h); i++ {
		entries = append(entries, LeafEntry{
			RecordNumber: uint16(i + 1),
			Key:          Key(string(rune('A' + i))),
		})
	}
	if err := pm.BuildFromEntries(entries); err != nil {
		t.Fatalf("BuildFromEntries: %v", err)
	}

	if err := pm.InsertMapping(uint16(maxLeafKeys(h)+1), Key(string(rune('A'+maxLeafKeys(h))))); err != nil {
		t.Fatalf("InsertMapping: %v", err)
	}

	_, found, err := pm.SearchExact(Key(string(rune('A' + maxLeafKeys(h)))))
	if err != nil || !found {
		t.Fatalf("expected inserted key, found=%v err=%v", found, err)
	}
}
