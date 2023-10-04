package dbf

import (
	"bytes"
	"strings"
	"testing"
)

func buildDBF(sig byte, recCount uint16, recLen uint16, fields []FieldDescriptor, term bool) []byte {
	var buf []byte
	buf = append(buf, sig)
	buf = append(buf, byte(recCount), byte(recCount>>8))
	buf = append(buf, 0x50, 0x06, 0x01)
	buf = append(buf, byte(recLen), byte(recLen>>8))
	for _, f := range fields {
		fb := make([]byte, fieldDescriptorSize)
		nameBytes := []byte(f.Name)
		for i := 0; i < fieldNameLen; i++ {
			if i < len(nameBytes) {
				fb[i] = nameBytes[i]
			} else {
				fb[i] = 0x00
			}
		}
		fb[10] = byte(f.Type)
		fb[11] = f.Length
		fb[14] = f.DecimalCount
		buf = append(buf, fb...)
	}
	if term {
		buf = append(buf, 0x0D)
	}
	return buf
}

func TestOpenValidTable(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 20},
		{Name: "AGE", Type: FieldTypeNumeric, Length: 3},
	}
	data := buildDBF(SignatureStd, 5, 24, fields, true)

	tbl, err := Open(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tbl.Header.Signature != SignatureStd {
		t.Errorf("signature = 0x%02X, want 0x%02X", tbl.Header.Signature, SignatureStd)
	}
	if tbl.Header.RecordCount != 5 {
		t.Errorf("record count = %d, want 5", tbl.Header.RecordCount)
	}
	if len(tbl.Fields) != 2 {
		t.Fatalf("field count = %d, want 2", len(tbl.Fields))
	}
	if tbl.Fields[0].Name != "NAME" {
		t.Errorf("fields[0].Name = %q, want %q", tbl.Fields[0].Name, "NAME")
	}
	if tbl.Fields[1].Name != "AGE" {
		t.Errorf("fields[1].Name = %q, want %q", tbl.Fields[1].Name, "AGE")
	}
	wantOffsets := []int{0, 20, 23}
	for i, wo := range wantOffsets {
		if tbl.Offset[i] != wo {
			t.Errorf("offset[%d] = %d, want %d", i, tbl.Offset[i], wo)
		}
	}
}

func TestOpenNoFields(t *testing.T) {
	buf := []byte{
		0x02,
		0x00, 0x00,
		0x50, 0x06, 0x01,
		0x01, 0x00,
		0x0D,
	}
	_, err := Open(bytes.NewReader(buf))
	if err == nil {
		t.Fatal("expected error for no fields")
	}
	if !strings.Contains(err.Error(), "no field descriptors") {
		t.Errorf("error = %v, want no field descriptors error", err)
	}
}

func TestOpenRecordLengthMismatch(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 20},
	}
	data := buildDBF(SignatureStd, 1, 99, fields, true)
	_, err := Open(bytes.NewReader(data))
	if err == nil {
		t.Fatal("expected error for record length mismatch")
	}
	if !strings.Contains(err.Error(), "record length") {
		t.Errorf("error = %v, want record length error", err)
	}
}

func TestOpenMemoSignature(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "VAL", Type: FieldTypeNumeric, Length: 5},
	}
	data := buildDBF(SignatureMemo, 1, 6, fields, true)

	tbl, err := Open(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tbl.Header.Signature != SignatureMemo {
		t.Errorf("signature = 0x%02X, want 0x%02X", tbl.Header.Signature, SignatureMemo)
	}
}

func TestOpenInvalidSignature(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "X", Type: FieldTypeChar, Length: 1},
	}
	data := buildDBF(0xFF, 0, 2, fields, true)
	_, err := Open(bytes.NewReader(data))
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
}

func TestFieldByNameFound(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 20},
		{Name: "AGE", Type: FieldTypeNumeric, Length: 3},
	}
	data := buildDBF(SignatureStd, 1, 24, fields, true)

	tbl, err := Open(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fd, idx := tbl.FieldByName("AGE")
	if fd == nil {
		t.Fatal("expected to find field AGE")
	}
	if fd.Name != "AGE" {
		t.Errorf("name = %q, want %q", fd.Name, "AGE")
	}
	if idx != 1 {
		t.Errorf("index = %d, want 1", idx)
	}
}

func TestFieldByNameNotFound(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 20},
	}
	data := buildDBF(SignatureStd, 1, 21, fields, true)

	tbl, err := Open(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fd, idx := tbl.FieldByName("MISSING")
	if fd != nil {
		t.Error("expected nil for missing field")
	}
	if idx != -1 {
		t.Errorf("index = %d, want -1", idx)
	}
}

func TestTableUnderlying(t *testing.T) {
	reader := strings.NewReader("dummy")
	tbl := &Table{underlying: reader}
	if tbl.Underlying() != reader {
		t.Errorf("expected Underlying() to return the same reader")
	}
}
