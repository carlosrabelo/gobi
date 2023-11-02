package dbf

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestZapRemovesAllRecords(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'O', 'N', 'E', ' ', ' '},
		{0x2A, 'T', 'W', 'O', ' ', ' '},
		{0x20, 'T', 'H', 'R', 'E', 'E'},
	}
	data, tbl := buildDBFFile(fields, records)

	pf := &packFile{data: make([]byte, len(data))}
	copy(pf.data, data)

	removed, err := tbl.Zap(pf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removed != 3 {
		t.Fatalf("removed = %d, want 3", removed)
	}
	if tbl.Header.RecordCount != 0 {
		t.Fatalf("record count = %d, want 0", tbl.Header.RecordCount)
	}

	cnt := binary.LittleEndian.Uint16(pf.data[1:3])
	if cnt != 0 {
		t.Fatalf("on-disk record count = %d, want 0", cnt)
	}

	wantSize := tbl.HeaderSize() + 1
	if len(pf.data) != wantSize {
		t.Fatalf("file size = %d, want %d", len(pf.data), wantSize)
	}

	reopened, err := Open(bytes.NewReader(pf.data))
	if err != nil {
		t.Fatalf("reopen zapped file: %v", err)
	}
	if reopened.Header.RecordCount != 0 {
		t.Fatalf("reopened record count = %d, want 0", reopened.Header.RecordCount)
	}
}

func TestZapEmptyTable(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	data, tbl := buildDBFFile(fields, nil)

	pf := &packFile{data: make([]byte, len(data))}
	copy(pf.data, data)

	removed, err := tbl.Zap(pf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removed != 0 {
		t.Fatalf("removed = %d, want 0", removed)
	}
	if tbl.Header.RecordCount != 0 {
		t.Fatalf("record count = %d, want 0", tbl.Header.RecordCount)
	}
}

func TestZapRequiresTruncate(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'O', 'N', 'E', ' ', ' '},
	}
	data, tbl := buildDBFFile(fields, records)

	pf := &noTruncateFile{data: make([]byte, len(data))}
	copy(pf.data, data)

	_, err := tbl.Zap(pf)
	if err == nil {
		t.Fatal("expected error when truncate is unsupported")
	}
}
