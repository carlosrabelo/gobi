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
}

func TestReadFieldDescriptorsTruncated(t *testing.T) {
	f1 := makeFieldBlock("NAME", 'C', 10, 0)
	truncated := f1[:8]
	_, err := readFieldDescriptors(bytes.NewReader(truncated))
	if err == nil {
		t.Fatal("expected error for truncated field descriptor")
	}
}
