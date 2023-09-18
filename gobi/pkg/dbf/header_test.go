package dbf

import (
	"bytes"
	"testing"
)

func TestWriteReadHeader(t *testing.T) {
	fields := []Field{{Name: "NAME", Type: 'C', Length: 10}}
	h := Header{Year: 86, Month: 7, Day: 4, Fields: fields}
	var buf bytes.Buffer
	if err := WriteHeader(&buf, h); err != nil {
		t.Fatal(err)
	}
	got, err := ReadHeader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if got.Signature != 0x03 {
		t.Fatalf("sig: 0x%02X", got.Signature)
	}
	if got.RecordCount != 0 {
		t.Fatalf("count: %d", got.RecordCount)
	}
	if got.HeaderLen != 32+32+1 {
		t.Fatalf("header len: %d", got.HeaderLen)
	}
	if got.RecordLen != 1+10 {
		t.Fatalf("rec len: %d", got.RecordLen)
	}
	if len(got.Fields) != 1 || got.Fields[0].Name != "NAME" || got.Fields[0].Type != 'C' {
		t.Fatalf("fields: %+v", got.Fields)
	}
}

func TestRejectSignature02(t *testing.T) {
	raw := make([]byte, 32)
	raw[0] = 0x02
	raw[8] = 33 // header len
	raw[10] = 1 // record len
	_, err := ReadHeader(bytes.NewReader(raw))
	if err == nil {
		t.Fatal("expected error for 0x02")
	}
}

func TestReject129thField(t *testing.T) {
	fields := make([]Field, 129)
	for i := range fields {
		fields[i] = Field{Name: "F", Type: 'C', Length: 1}
	}
	if _, err := ValidateFields(fields); err == nil {
		t.Fatal("expected error for 129 fields")
	}
}

func TestReject129OnWrite(t *testing.T) {
	fields := make([]Field, 129)
	for i := range fields {
		fields[i] = Field{Name: "F", Type: 'C', Length: 1}
	}
	var buf bytes.Buffer
	if err := WriteHeader(&buf, Header{Fields: fields}); err == nil {
		t.Fatal("expected error")
	}
}

func TestMemoSignature(t *testing.T) {
	fields := []Field{{Name: "NOTE", Type: 'M', Length: 10}}
	var buf bytes.Buffer
	if err := WriteHeader(&buf, Header{Fields: fields}); err != nil {
		t.Fatal(err)
	}
	got, err := ReadHeader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if got.Signature != 0x83 {
		t.Fatalf("sig: 0x%02X", got.Signature)
	}
}

func TestMissingTerminator(t *testing.T) {
	fields := []Field{{Name: "A", Type: 'C', Length: 1}}
	var buf bytes.Buffer
	_ = WriteHeader(&buf, Header{Fields: fields})
	b := buf.Bytes()
	b[len(b)-1] = 0x00
	_, err := ReadHeader(bytes.NewReader(b))
	if err == nil {
		t.Fatal("expected missing terminator")
	}
}
