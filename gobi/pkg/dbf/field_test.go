package dbf

import (
	"bytes"
	"testing"
)

func makeFieldBlock(name string, ftype byte, length byte, decimal byte) []byte {
	b := make([]byte, fieldDescriptorSize)
	copy(b, name)
	for i := len(name); i < fieldNameLen; i++ {
		b[i] = 0x00
	}
	b[10] = ftype
	b[11] = length
	b[12] = 0x00
	b[13] = 0x00
	b[14] = decimal
	b[15] = 0x00
	return b
}

func TestReadFieldDescriptorsSingleChar(t *testing.T) {
	name := makeFieldBlock("NAME", 'C', 20, 0)
	term := []byte{0x0D}
	data := append(name, term...)

	fields, err := readFieldDescriptors(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fields) != 1 {
		t.Fatalf("field count = %d, want 1", len(fields))
	}
	if fields[0].Name != "NAME" {
		t.Errorf("name = %q, want %q", fields[0].Name, "NAME")
	}
	if fields[0].Type != FieldTypeChar {
		t.Errorf("type = %c, want C", fields[0].Type)
	}
	if fields[0].Length != 20 {
		t.Errorf("length = %d, want 20", fields[0].Length)
	}
	if fields[0].DecimalCount != 0 {
		t.Errorf("decimal count = %d, want 0", fields[0].DecimalCount)
	}
}

func TestReadFieldDescriptorsMultiple(t *testing.T) {
	f1 := makeFieldBlock("NAME", 'C', 25, 0)
	f2 := makeFieldBlock("AGE", 'N', 3, 0)
	f3 := makeFieldBlock("ACTIVE", 'L', 1, 0)
	term := []byte{0x0D}

	var data []byte
	data = append(data, f1...)
	data = append(data, f2...)
	data = append(data, f3...)
	data = append(data, term...)

	fields, err := readFieldDescriptors(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fields) != 3 {
		t.Fatalf("field count = %d, want 3", len(fields))
	}

	want := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 25, DecimalCount: 0},
		{Name: "AGE", Type: FieldTypeNumeric, Length: 3, DecimalCount: 0},
		{Name: "ACTIVE", Type: FieldTypeLogical, Length: 1, DecimalCount: 0},
	}
	for i, w := range want {
		if fields[i].Name != w.Name {
			t.Errorf("fields[%d].Name = %q, want %q", i, fields[i].Name, w.Name)
		}
		if fields[i].Type != w.Type {
			t.Errorf("fields[%d].Type = %c, want %c", i, fields[i].Type, w.Type)
		}
		if fields[i].Length != w.Length {
			t.Errorf("fields[%d].Length = %d, want %d", i, fields[i].Length, w.Length)
		}
		if fields[i].DecimalCount != w.DecimalCount {
			t.Errorf("fields[%d].DecimalCount = %d, want %d", i, fields[i].DecimalCount, w.DecimalCount)
		}
	}
}

func TestReadFieldDescriptorsNumericWithDecimal(t *testing.T) {
	f1 := makeFieldBlock("PRICE", 'N', 10, 2)
	term := []byte{0x0D}
	data := append(f1, term...)

	fields, err := readFieldDescriptors(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fields[0].DecimalCount != 2 {
		t.Errorf("decimal count = %d, want 2", fields[0].DecimalCount)
	}
}

func TestReadFieldDescriptorsEmpty(t *testing.T) {
	data := []byte{0x0D}
	fields, err := readFieldDescriptors(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fields) != 0 {
		t.Errorf("field count = %d, want 0", len(fields))
	}
}

func TestReadFieldDescriptorsInvalidType(t *testing.T) {
	f1 := makeFieldBlock("BAD", 'X', 10, 0)
	term := []byte{0x0D}
	data := append(f1, term...)

	_, err := readFieldDescriptors(bytes.NewReader(data))
	if err == nil {
		t.Fatal("expected error for invalid field type")
	}
}

func TestReadFieldDescriptorsTruncated(t *testing.T) {
	f1 := makeFieldBlock("NAME", 'C', 10, 0)
	truncated := f1[:8]
	_, err := readFieldDescriptors(bytes.NewReader(truncated))
	if err == nil {
		t.Fatal("expected error for truncated field descriptor")
	}
}

func TestReadFieldDescriptorsNameNullPadded(t *testing.T) {
	b := make([]byte, fieldDescriptorSize)
	copy(b, "A")
	for i := 1; i < fieldNameLen; i++ {
		b[i] = 0x00
	}
	b[10] = 'C'
	b[11] = 10
	term := []byte{0x0D}
	data := append(b, term...)

	fields, err := readFieldDescriptors(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fields[0].Name != "A" {
		t.Errorf("name = %q, want %q", fields[0].Name, "A")
	}
}

