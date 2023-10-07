package dbf

import (
	"bytes"
	"testing"
)

func TestNewRecordCharPadding(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 10},
	}
	tbl := newTestTable(fields)

	rec, err := NewRecord(tbl, false, []interface{}{"HELLO"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data := rec.FieldData(tbl, 0)
	if string(data) != "HELLO     " {
		t.Errorf("padded = %q, want %q", string(data), "HELLO     ")
	}
}

func TestNewRecordCharTruncation(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 3},
	}
	tbl := newTestTable(fields)

	rec, err := NewRecord(tbl, false, []interface{}{"HELLO"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data := rec.FieldData(tbl, 0)
	if string(data) != "HEL" {
		t.Errorf("truncated = %q, want %q", string(data), "HEL")
	}
}

func TestNewRecordNumericInteger(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "AGE", Type: FieldTypeNumeric, Length: 5},
	}
	tbl := newTestTable(fields)

	rec, err := NewRecord(tbl, false, []interface{}{float64(42)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data := rec.FieldData(tbl, 0)
	if string(data) != "   42" {
		t.Errorf("padded = %q, want %q", string(data), "   42")
	}
}

func TestNewRecordNumericDecimal(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "PRICE", Type: FieldTypeNumeric, Length: 8, DecimalCount: 2},
	}
	tbl := newTestTable(fields)

	rec, err := NewRecord(tbl, false, []interface{}{float64(12.5)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data := rec.FieldData(tbl, 0)
	if string(data) != "   12.50" {
		t.Errorf("padded = %q, want %q", string(data), "   12.50")
	}
}

func TestNewRecordNumericIntInput(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "AGE", Type: FieldTypeNumeric, Length: 5},
	}
	tbl := newTestTable(fields)

	rec, err := NewRecord(tbl, false, []interface{}{100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data := rec.FieldData(tbl, 0)
	if string(data) != "  100" {
		t.Errorf("padded = %q, want %q", string(data), "  100")
	}
}

func TestNewRecordNumericEmptyString(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "VAL", Type: FieldTypeNumeric, Length: 5},
	}
	tbl := newTestTable(fields)

	rec, err := NewRecord(tbl, false, []interface{}{""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data := rec.FieldData(tbl, 0)
	if string(data) != "     " {
		t.Errorf("empty = %q, want %q", string(data), "     ")
	}
}

func TestNewRecordLogicalTrue(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "FLAG", Type: FieldTypeLogical, Length: 1},
	}
	tbl := newTestTable(fields)

	rec, err := NewRecord(tbl, false, []interface{}{true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data := rec.FieldData(tbl, 0)
	if string(data) != "T" {
		t.Errorf("value = %q, want %q", string(data), "T")
	}
}

func TestNewRecordLogicalFalse(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "FLAG", Type: FieldTypeLogical, Length: 1},
	}
	tbl := newTestTable(fields)

	rec, err := NewRecord(tbl, false, []interface{}{false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data := rec.FieldData(tbl, 0)
	if string(data) != "F" {
		t.Errorf("value = %q, want %q", string(data), "F")
	}
}

func TestNewRecordDeleted(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	tbl := newTestTable(fields)

	rec, err := NewRecord(tbl, true, []interface{}{"TEST"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !rec.Deleted {
		t.Error("record should be marked as deleted")
	}
	if rec.Data[0] != 0x2A {
		t.Errorf("deletion flag = 0x%02X, want 0x2A", rec.Data[0])
	}
}

func TestNewRecordWrongValueCount(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	tbl := newTestTable(fields)

	_, err := NewRecord(tbl, false, []interface{}{"A", "B"})
	if err == nil {
		t.Fatal("expected error for wrong value count")
	}
}

func TestNewRecordMultipleFields(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
		{Name: "AGE", Type: FieldTypeNumeric, Length: 3},
		{Name: "ACTIVE", Type: FieldTypeLogical, Length: 1},
	}
	tbl := newTestTable(fields)

	rec, err := NewRecord(tbl, false, []interface{}{"JOHN", float64(25), true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(rec.Data) != " JOHN  25T" {
		t.Errorf("full record = %q, want %q", string(rec.Data), " JOHN  25T")
	}
}

func TestWriteRecord(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	tbl := newTestTable(fields)

	rec, _ := NewRecord(tbl, false, []interface{}{"TEST"})

	var buf bytes.Buffer
	err := tbl.WriteRecord(&buf, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	written := buf.Bytes()
	if string(written) != " TEST " {
		t.Errorf("written = %q, want %q", string(written), " TEST ")
	}
}

func TestWriteRecordLengthMismatch(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	tbl := newTestTable(fields)

	rec := &Record{Deleted: false, Data: []byte{0x20, 'A', 'B'}}

	var buf bytes.Buffer
	err := tbl.WriteRecord(&buf, rec)
	if err == nil {
		t.Fatal("expected error for length mismatch")
	}
}

func TestRoundTripRecord(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 10},
		{Name: "AGE", Type: FieldTypeNumeric, Length: 5},
		{Name: "ACTIVE", Type: FieldTypeLogical, Length: 1},
	}
	tbl := newTestTable(fields)

	rec, _ := NewRecord(tbl, false, []interface{}{"JOHN", float64(42), true})

	var buf bytes.Buffer
	tbl.WriteRecord(&buf, rec)

	readback, err := tbl.ReadRecord(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	name, _ := readback.DecodeField(tbl, 0)
	if name.(string) != "JOHN" {
		t.Errorf("name = %q, want %q", name, "JOHN")
	}

	age, _ := readback.DecodeField(tbl, 1)
	if age.(float64) != 42 {
		t.Errorf("age = %v, want 42", age)
	}

	active, _ := readback.DecodeField(tbl, 2)
	if active.(bool) != true {
		t.Errorf("active = %v, want true", active)
	}
}

func newTestTable(fields []FieldDescriptor) *Table {
	offsets := make([]int, len(fields)+1)
	for i, f := range fields {
		offsets[i+1] = offsets[i] + int(f.Length)
	}
	recLen := uint16(offsets[len(fields)] + 1)
	return &Table{
		Header: &Header{Signature: SignatureStd, RecordCount: 0, RecordLen: recLen},
		Fields: fields,
		Offset: offsets,
	}
}

func TestNewRecordEncodingError(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "AGE", Type: FieldTypeNumeric, Length: 5},
	}
	tbl := newTestTable(fields)

	_, err := NewRecord(tbl, false, []interface{}{"not-a-number"})
	if err == nil {
		t.Fatal("expected error encoding invalid numeric string")
	}
}

func TestEncodeFieldUnsupportedType(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "DUMMY", Type: 'X', Length: 5}, // unsupported type 'X'
	}
	tbl := newTestTable(fields)

	_, err := NewRecord(tbl, false, []interface{}{"hello"})
	if err == nil {
		t.Fatal("expected error for unsupported field type")
	}
}

func TestEncodeNumericInvalidType(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "VAL", Type: FieldTypeNumeric, Length: 5},
	}
	tbl := newTestTable(fields)

	_, err := NewRecord(tbl, false, []interface{}{true}) // boolean is invalid for numeric
	if err == nil {
		t.Fatal("expected error for invalid type in numeric field")
	}
}

func TestEncodeNumericOverflow(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "VAL", Type: FieldTypeNumeric, Length: 3},
	}
	tbl := newTestTable(fields)

	// A value that overflows 3 chars
	rec, err := NewRecord(tbl, false, []interface{}{12345.0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data := rec.FieldData(tbl, 0)
	if string(data) != "***" {
		t.Errorf("overflow representation = %q, want %q", string(data), "***")
	}

	// Negative value overflow test (replaces '-' and checks if it fits, if not, stars)
	rec, err = NewRecord(tbl, false, []interface{}{-12345.0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data = rec.FieldData(tbl, 0)
	if string(data) != "***" {
		t.Errorf("negative overflow representation = %q, want %q", string(data), "***")
	}
}

func TestEncodeLogicalStringAndInvalid(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "L1", Type: FieldTypeLogical, Length: 1},
		{Name: "L2", Type: FieldTypeLogical, Length: 1},
		{Name: "L3", Type: FieldTypeLogical, Length: 1},
	}
	tbl := newTestTable(fields)

	// L1: "Yes" -> 'Y'
	// L2: "" -> '?'
	// L3: float64(12) -> '?' (invalid type)
	rec, err := NewRecord(tbl, false, []interface{}{"Yes", "", float64(12)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(rec.FieldData(tbl, 0)) != "Y" {
		t.Errorf("L1 = %q, want %q", string(rec.FieldData(tbl, 0)), "Y")
	}
	if string(rec.FieldData(tbl, 1)) != "?" {
		t.Errorf("L2 = %q, want %q", string(rec.FieldData(tbl, 1)), "?")
	}
	if string(rec.FieldData(tbl, 2)) != "?" {
		t.Errorf("L3 = %q, want %q", string(rec.FieldData(tbl, 2)), "?")
	}
}

func TestEncodeNumericStringInput(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "VAL", Type: FieldTypeNumeric, Length: 5},
	}
	tbl := newTestTable(fields)

	rec, err := NewRecord(tbl, false, []interface{}{"42.5"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data := rec.FieldData(tbl, 0)
	// We expect "   42" because it truncates to int when decimalCount is 0 in encodeNumeric
	if string(data) != "   42" {
		t.Errorf("value = %q, want %q", string(data), "   42")
	}
}
