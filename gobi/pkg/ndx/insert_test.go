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

	node, err := pm.ReadNode(pm.Header().RootPageID, NodeKindLeaf)
	if err != nil {
		t.Fatalf("ReadNode: %v", err)
	}
	entry, found, err := LeafEntryForKey(pm.Header(), node, Key("Alice"))
	if err != nil || !found || entry.RecordNumber != 1 {
		t.Fatalf("expected Alice rec 1, found=%v entry=%#v err=%v", found, entry, err)
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

	node, err := pm.ReadNode(pm.Header().RootPageID, NodeKindLeaf)
	if err != nil {
		t.Fatalf("ReadNode: %v", err)
	}
	entry, found, err := LeafEntryForKey(pm.Header(), node, Key("Bob"))
	if err != nil || !found || entry.RecordNumber != 2 {
		t.Fatalf("expected Bob rec 2, found=%v entry=%#v err=%v", found, entry, err)
	}
}
