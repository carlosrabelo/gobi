package mem

import (
	"bytes"
	"math"
	"testing"
)

func TestWriteReadRoundTrip(t *testing.T) {
	input := []Variable{
		{Name: "ONE", Value: float64(1)},
		{Name: "ALFABET", Value: "abcdefghijkl"},
		{Name: "CHARS", Value: "abcdefghijkl new stuff"},
		{Name: "FLAG", Value: true},
		{Name: "EMPTY", Value: false},
	}

	var buf bytes.Buffer
	if err := Write(&buf, input); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if buf.Bytes()[len(buf.Bytes())-1] != eofMarker {
		t.Fatal("expected EOF marker at end of file")
	}

	got, err := Read(&buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got) != len(input) {
		t.Fatalf("expected %d variables, got %d", len(input), len(got))
	}

	for i, want := range input {
		if got[i].Name != want.Name {
			t.Fatalf("var %d: expected name %q, got %q", i, want.Name, got[i].Name)
		}
		if !valuesEqual(got[i].Value, want.Value) {
			t.Fatalf("var %q: expected %#v, got %#v", want.Name, want.Value, got[i].Value)
		}
	}
}

func TestWriteEmptyRegistry(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, nil); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), []byte{eofMarker}) {
		t.Fatalf("expected only EOF marker, got % x", buf.Bytes())
	}

	got, err := Read(&buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no variables, got %#v", got)
	}
}

func TestWriteCharacterHeaderLayout(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, []Variable{{Name: "MSG", Value: "hi"}}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	data := buf.Bytes()
	if data[0] != 'M' || data[1] != 'S' || data[2] != 'G' {
		t.Fatalf("expected name MSG at start, got %q", string(data[:10]))
	}
	if data[11] != typeChar {
		t.Fatalf("expected character type 0xC3, got 0x%02X", data[11])
	}
	if data[12] != 2 {
		t.Fatalf("expected value length 2, got %d", data[12])
	}
	if data[15] != markerByte {
		t.Fatalf("expected marker 'E' at byte 15, got 0x%02X", data[15])
	}
	if data[16] != 2 {
		t.Fatalf("expected field length 2, got %d", data[16])
	}
	if string(data[19:21]) != "hi" {
		t.Fatalf("expected value hi at offset 19, got %q", string(data[19:21]))
	}
}

func TestWriteNumericHeaderLayout(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, []Variable{{Name: "NUM", Value: float64(42)}}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	data := buf.Bytes()
	if data[11] != typeNumeric {
		t.Fatalf("expected numeric type 0xCE, got 0x%02X", data[11])
	}
	if data[12] != numericFieldLen {
		t.Fatalf("expected value length %d, got %d", numericFieldLen, data[12])
	}
	if data[15] != markerByte {
		t.Fatalf("expected marker 'E' at byte 15, got 0x%02X", data[15])
	}
	if data[16] != numericFieldLen {
		t.Fatalf("expected field length %d, got %d", numericFieldLen, data[16])
	}
	if data[17] != 0 {
		t.Fatalf("expected decimal count 0, got %d", data[17])
	}
}

func TestWriteLogicalValueLayout(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, []Variable{{Name: "OK", Value: true}}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	data := buf.Bytes()
	if data[11] != typeLogical {
		t.Fatalf("expected logical type 0xCC, got 0x%02X", data[11])
	}
	if data[12] != logicalValueSize {
		t.Fatalf("expected value length %d, got %d", logicalValueSize, data[12])
	}
	if data[19+logicalValueSize-1] != 1 {
		t.Fatalf("expected true marker in last value byte, got 0x%02X", data[19+logicalValueSize-1])
	}
}

func TestWriteTruncatesLongNames(t *testing.T) {
	var buf bytes.Buffer
	longName := "VERYLONGNAME"
	if err := Write(&buf, []Variable{{Name: longName, Value: float64(1)}}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := Read(&buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got[0].Name != "VERYLONGN" {
		t.Fatalf("expected truncated name VERYLONGN, got %q", got[0].Name)
	}
}

func TestReadDecimalNumeric(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, []Variable{{Name: "PAY", Value: 17.35}}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := Read(&buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	val, ok := got[0].Value.(float64)
	if !ok || math.Abs(val-17.35) > 0.001 {
		t.Fatalf("expected 17.35, got %#v", got[0].Value)
	}
}

func valuesEqual(a, b interface{}) bool {
	switch av := a.(type) {
	case float64:
		bv, ok := b.(float64)
		return ok && av == bv
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	default:
		return false
	}
}
