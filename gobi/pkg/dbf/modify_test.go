package dbf

import (
	"bytes"
	"testing"
)

func TestRewriteStructure(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 10},
	}
	recData := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	data, _ := buildDBFFile(fields, [][]byte{recData})

	pf := &packFile{data: make([]byte, len(data))}
	copy(pf.data, data)

	newFields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 10},
		{Name: "PHONE", Type: FieldTypeChar, Length: 12},
	}

	tbl, err := RewriteStructure(pf, newFields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tbl.Header.RecordCount != 0 {
		t.Fatalf("record count = %d, want 0", tbl.Header.RecordCount)
	}
	if len(tbl.Fields) != 2 {
		t.Fatalf("field count = %d, want 2", len(tbl.Fields))
	}

	wantSize := tbl.HeaderSize() + 1
	if len(pf.data) != wantSize {
		t.Fatalf("file size = %d, want %d", len(pf.data), wantSize)
	}

	reopened, err := Open(bytes.NewReader(pf.data))
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if reopened.Header.RecordLen != 23 {
		t.Fatalf("record length = %d, want 23", reopened.Header.RecordLen)
	}
}
