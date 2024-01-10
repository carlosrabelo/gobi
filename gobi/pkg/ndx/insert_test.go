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
