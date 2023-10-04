package dbf

import (
	"bytes"
	"testing"
)

func buildDBFWithRecords(fields []FieldDescriptor, records [][]byte) []byte {
	recLen := 1
	for _, f := range fields {
		recLen += int(f.Length)
	}
	data := buildDBF(SignatureStd, uint16(len(records)), uint16(recLen), fields, true)
	for _, rec := range records {
		data = append(data, rec...)
	}
	data = append(data, 0x1A)
	return data
}

func TestReadRecordActive(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'H', 'E', 'L', 'L', 'O'},
	}
	data := buildDBFWithRecords(fields, records)

	tbl, err := Open(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec, err := tbl.ReadRecord(bytes.NewReader(records[0]))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Deleted {
		t.Error("record should not be marked as deleted")
	}
	if string(rec.Data) != string(records[0]) {
		t.Errorf("data = %q, want %q", string(rec.Data), string(records[0]))
	}
}

func TestReadRecordDeleted(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x2A, 'W', 'O', 'R', 'L', 'D'},
	}
	data := buildDBFWithRecords(fields, records)

	tbl, err := Open(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec, err := tbl.ReadRecord(bytes.NewReader(records[0]))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rec.Deleted {
		t.Error("record should be marked as deleted")
	}
}

func TestReadRecordInvalidDeletionFlag(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0xFF, 'A', 'B', 'C', 'D', 'E'},
	}
	data := buildDBFWithRecords(fields, records)

	tbl, err := Open(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = tbl.ReadRecord(bytes.NewReader(records[0]))
	if err == nil {
		t.Fatal("expected error for invalid deletion flag")
	}
}

func TestReadRecordTruncated(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	data := buildDBFWithRecords(fields, nil)
	tbl, err := Open(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = tbl.ReadRecord(bytes.NewReader([]byte{0x20, 'A', 'B'}))
	if err == nil {
		t.Fatal("expected error for truncated record")
	}
}

func TestReadMultipleRecords(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
		{Name: "AGE", Type: FieldTypeNumeric, Length: 3},
	}
	records := [][]byte{
		{0x20, 'H', 'E', 'L', 'L', 'O', ' ', '2', '5'},
		{0x2A, 'W', 'O', 'R', 'L', 'D', ' ', '3', '0'},
		{0x20, 'F', 'O', 'O', ' ', ' ', ' ', '1', '8'},
	}
	all := append([]byte{}, records[0]...)
	all = append(all, records[1]...)
	all = append(all, records[2]...)
	all = append(all, 0x1A)

	data := buildDBFWithRecords(fields, records)
	tbl, err := Open(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := bytes.NewReader(all)
	rec0, err := tbl.ReadRecord(r)
	if err != nil {
		t.Fatalf("record 0: %v", err)
	}
	if rec0.Deleted {
		t.Error("record 0 should not be deleted")
	}

	rec1, err := tbl.ReadRecord(r)
	if err != nil {
		t.Fatalf("record 1: %v", err)
	}
	if !rec1.Deleted {
		t.Error("record 1 should be deleted")
	}

	rec2, err := tbl.ReadRecord(r)
	if err != nil {
		t.Fatalf("record 2: %v", err)
	}
	if rec2.Deleted {
		t.Error("record 2 should not be deleted")
	}

	_, err = tbl.ReadRecord(r)
	if err == nil {
		t.Fatal("expected EOF after all records")
	}
}

func TestRecordFieldData(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
		{Name: "AGE", Type: FieldTypeNumeric, Length: 3},
	}
	records := [][]byte{
		{0x20, 'H', 'E', 'L', 'L', 'O', ' ', '2', '5'},
	}
	data := buildDBFWithRecords(fields, records)

	tbl, err := Open(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec, err := tbl.ReadRecord(bytes.NewReader(records[0]))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nameData := rec.FieldData(tbl, 0)
	if string(nameData) != "HELLO" {
		t.Errorf("field 0 = %q, want %q", string(nameData), "HELLO")
	}

	ageData := rec.FieldData(tbl, 1)
	if string(ageData) != " 25" {
		t.Errorf("field 1 = %q, want %q", string(ageData), " 25")
	}
}

func TestRecordFieldDataOutOfBounds(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'H', 'E', 'L', 'L', 'O'},
	}
	data := buildDBFWithRecords(fields, records)

	tbl, err := Open(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec, err := tbl.ReadRecord(bytes.NewReader(records[0]))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.FieldData(tbl, -1) != nil {
		t.Error("expected nil for negative index")
	}
	if rec.FieldData(tbl, 1) != nil {
		t.Error("expected nil for out-of-bounds index")
	}
}
