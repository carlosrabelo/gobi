package repl

import (
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestRecordToEditValues(t *testing.T) {
	tempDir := t.TempDir()
	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	dbfPath := createTempDBFWithRecords(t, tempDir, "editdb.dbf", [][]byte{rec})

	ctx := testCtx()
	if err := commandMux.Dispatch(ctx, Command{Verb: "USE", Args: dbfPath}); err != nil {
		t.Fatalf("open table: %v", err)
	}

	area := ctx.GetActiveArea()
	values, err := recordToEditValues(area.Table, area.ActiveRecord)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if values[0] != "Alice" {
		t.Fatalf("NAME = %q, want Alice", values[0])
	}
	if values[1] != "25" {
		t.Fatalf("AGE = %q, want 25", values[1])
	}
}

func TestEditSessionInsertCharRespectsWidth(t *testing.T) {
	s := &editSession{
		tbl: &dbf.Table{
			Header: &dbf.Header{RecordLen: 4},
			Fields: []dbf.FieldDescriptor{
				{Name: "CODE", Type: dbf.FieldTypeChar, Length: 3},
			},
			Offset: []int{0, 3},
		},
		values:    []string{"AB"},
		fieldIdx:  0,
		cursorPos: 2,
	}

	if err := s.insertChar('C'); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.values[0] != "ABC" {
		t.Fatalf("value = %q, want ABC", s.values[0])
	}

	if err := s.insertChar('D'); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.values[0] != "ABC" {
		t.Fatalf("value = %q, want ABC after overflow", s.values[0])
	}
}

func TestEditSessionLogicalField(t *testing.T) {
	s := &editSession{
		tbl: &dbf.Table{
			Header: &dbf.Header{RecordLen: 2},
			Fields: []dbf.FieldDescriptor{
				{Name: "ACTIVE", Type: dbf.FieldTypeLogical, Length: 1},
			},
			Offset: []int{0, 1},
		},
		values:    []string{"F"},
		fieldIdx:  0,
		cursorPos: 0,
	}

	if err := s.insertChar('t'); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.values[0] != "T" {
		t.Fatalf("value = %q, want T", s.values[0])
	}
}
