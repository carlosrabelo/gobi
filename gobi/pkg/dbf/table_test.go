package dbf

import (
	"encoding/binary"
	"testing"
)

func TestCreateAppendUpdate(t *testing.T) {
	fields := []Field{
		{Name: "NAME", Type: 'C', Length: 4},
		{Name: "AMT", Type: 'N', Length: 6, Decimals: 1},
	}
	mem := &MemFile{}
	tab, err := Create(mem, fields, 86, 7, 4)
	if err != nil {
		t.Fatal(err)
	}
	if tab.Header.Signature != 0x03 {
		t.Fatalf("sig 0x%02X", tab.Header.Signature)
	}
	if tab.Header.RecordCount != 0 {
		t.Fatalf("count %d", tab.Header.RecordCount)
	}

	r1 := Record{Values: []Value{{Str: "AA"}, {Num: 1.5}}}
	r2 := Record{Values: []Value{{Str: "BB"}, {Num: 2.5}}}
	if err := tab.Append(r1); err != nil {
		t.Fatal(err)
	}
	if err := tab.Append(r2); err != nil {
		t.Fatal(err)
	}
	if tab.Header.RecordCount != 2 {
		t.Fatalf("count %d", tab.Header.RecordCount)
	}
	// Header count on disk
	count := binary.LittleEndian.Uint32(mem.Bytes()[4:8])
	if count != 2 {
		t.Fatalf("disk count %d", count)
	}

	got, err := tab.Read(2)
	if err != nil {
		t.Fatal(err)
	}
	if got.Values[0].Str != "BB" || got.Values[1].Num != 2.5 {
		t.Fatalf("got %+v", got.Values)
	}

	upd := Record{Values: []Value{{Str: "ZZ"}, {Num: 9.0}}}
	if err := tab.Write(1, upd); err != nil {
		t.Fatal(err)
	}
	g1, err := tab.Read(1)
	if err != nil {
		t.Fatal(err)
	}
	if g1.Values[0].Str != "ZZ" {
		t.Fatalf("rec1 %q", g1.Values[0].Str)
	}
	g2, err := tab.Read(2)
	if err != nil {
		t.Fatal(err)
	}
	if g2.Values[0].Str != "BB" {
		t.Fatalf("rec2 changed: %q", g2.Values[0].Str)
	}

	if err := tab.Close(); err != nil {
		t.Fatal(err)
	}

	// Re-open and verify
	mem.Seek(0, 0)
	tab2, err := Open(mem)
	if err != nil {
		t.Fatal(err)
	}
	if tab2.Header.RecordCount != 2 {
		t.Fatalf("reopen count %d", tab2.Header.RecordCount)
	}
	g1, err = tab2.Read(1)
	if err != nil {
		t.Fatal(err)
	}
	if g1.Values[0].Str != "ZZ" {
		t.Fatalf("reopen rec1 %q", g1.Values[0].Str)
	}
}

func TestWriteOutOfRange(t *testing.T) {
	mem := &MemFile{}
	tab, err := Create(mem, []Field{{Name: "A", Type: 'C', Length: 1}}, 86, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	err = tab.Write(1, Record{Values: []Value{{Str: "X"}}})
	if err == nil {
		t.Fatal("expected out of range")
	}
}

func TestEncodeNOverflow(t *testing.T) {
	fields := []Field{{Name: "N", Type: 'N', Length: 3, Decimals: 0}}
	raw, err := EncodeRecord(fields, Record{Values: []Value{{Num: 12345}}})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw[1:]) != "***" {
		t.Fatalf("got %q", raw[1:])
	}
}
