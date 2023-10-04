package dbf

import (
	"bytes"
	"testing"
)

func TestDecodeFieldChar(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 10},
	}
	records := [][]byte{
		{0x20, 'H', 'E', 'L', 'L', 'O', ' ', ' ', ' ', ' ', ' '},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	val, err := rec.DecodeField(tbl, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s, ok := val.(string)
	if !ok {
		t.Fatalf("expected string, got %T", val)
	}
	if s != "HELLO" {
		t.Errorf("value = %q, want %q", s, "HELLO")
	}
}

func TestDecodeFieldCharAllSpaces(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, ' ', ' ', ' ', ' ', ' '},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	val, err := rec.DecodeField(tbl, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := val.(string)
	if s != "" {
		t.Errorf("value = %q, want empty string", s)
	}
}

func TestDecodeFieldCharNoPadding(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'A', 'B', 'C', 'D', 'E'},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	val, err := rec.DecodeField(tbl, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := val.(string)
	if s != "ABCDE" {
		t.Errorf("value = %q, want %q", s, "ABCDE")
	}
}

func TestDecodeFieldCharPreservesLeadingSpaces(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 8},
	}
	records := [][]byte{
		{0x20, ' ', ' ', 'H', 'I', ' ', ' ', ' ', ' '},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	val, err := rec.DecodeField(tbl, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := val.(string)
	if s != "  HI" {
		t.Errorf("value = %q, want %q", s, "  HI")
	}
}

func TestDecodeFieldUnsupportedType(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'H', 'E', 'L', 'L', 'O'},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	tbl.Fields[0].Type = 'X'
	_, err := rec.DecodeField(tbl, 0)
	if err == nil {
		t.Fatal("expected error for unsupported field type")
	}
}

func TestDecodeFieldIndexOutOfRange(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'H', 'E', 'L', 'L', 'O'},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	_, err := rec.DecodeField(tbl, 5)
	if err == nil {
		t.Fatal("expected error for out-of-range index")
	}
}

func TestDecodeFieldMultipleCharFields(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "FIRST", Type: FieldTypeChar, Length: 5},
		{Name: "LAST", Type: FieldTypeChar, Length: 8},
	}
	records := [][]byte{
		{0x20, 'J', 'O', 'H', 'N', ' ', 'D', 'O', 'E', ' ', ' ', ' ', ' ', ' '},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	first, _ := rec.DecodeField(tbl, 0)
	if first.(string) != "JOHN" {
		t.Errorf("first = %q, want %q", first, "JOHN")
	}

	last, _ := rec.DecodeField(tbl, 1)
	if last.(string) != "DOE" {
		t.Errorf("last = %q, want %q", last, "DOE")
	}
}

func openTableWithRecord(t *testing.T, fields []FieldDescriptor, records [][]byte) (*Table, *Record) {
	t.Helper()
	data := buildDBFWithRecords(fields, records)
	tbl, err := Open(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("open table: %v", err)
	}
	rec, err := tbl.ReadRecord(bytes.NewReader(records[0]))
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	return tbl, rec
}

