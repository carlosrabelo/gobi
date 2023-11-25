package dbf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSampleDBFFixture(t *testing.T) {
	path := filepath.Join("testdata", "sample.dbf")
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	tab, err := Open(f)
	if err != nil {
		t.Fatal(err)
	}
	if tab.Header.Signature != 0x03 {
		t.Fatalf("sig %#x", tab.Header.Signature)
	}
	if len(tab.Header.Fields) != 2 {
		t.Fatalf("fields %d", len(tab.Header.Fields))
	}
	if tab.Header.Fields[0].Type != 'C' || tab.Header.Fields[1].Type != 'D' {
		t.Fatalf("types %c %c", tab.Header.Fields[0].Type, tab.Header.Fields[1].Type)
	}
	if tab.Header.RecordCount != 2 {
		t.Fatalf("count %d", tab.Header.RecordCount)
	}
	r1, err := tab.Read(1)
	if err != nil {
		t.Fatal(err)
	}
	if r1.Deleted || r1.Values[1].Str != "19860704" {
		t.Fatalf("r1 %+v", r1)
	}
	r2, err := tab.Read(2)
	if err != nil {
		t.Fatal(err)
	}
	if !r2.Deleted {
		t.Fatal("r2 should be deleted")
	}
}
