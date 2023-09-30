package dbf

import (
	"bytes"
	"fmt"
	"io"
)

const (
	fieldDescriptorSize = 16
	maxFieldCount       = 32
	fieldNameLen        = 10
	headerTerminator    = 0x0D
)

type FieldType byte

const (
	FieldTypeChar    FieldType = 'C'
	FieldTypeNumeric FieldType = 'N'
	FieldTypeLogical FieldType = 'L'
)

var validFieldTypes = map[FieldType]bool{
	FieldTypeChar:    true,
	FieldTypeNumeric: true,
	FieldTypeLogical: true,
}

type FieldDescriptor struct {
	Name         string
	Type         FieldType
	Length       byte
	DecimalCount byte
}

func readFieldDescriptor(buf []byte) (*FieldDescriptor, error) {
	name := string(bytes.TrimRight(buf[:fieldNameLen], "\x00"))
	ft := FieldType(buf[10])
	if !validFieldTypes[ft] {
		return nil, fmt.Errorf("dbf: invalid field type %q for field %q", byte(ft), name)
	}
	return &FieldDescriptor{
		Name:         name,
		Type:         ft,
		Length:       buf[11],
		DecimalCount: buf[14],
	}, nil
}

func readFieldDescriptors(r io.Reader) ([]FieldDescriptor, error) {
	var fields []FieldDescriptor
	var buf [fieldDescriptorSize]byte

	for i := 0; i < maxFieldCount; i++ {
		var first [1]byte
		if _, err := io.ReadFull(r, first[:]); err != nil {
			return nil, fmt.Errorf("dbf: reading field descriptor %d: %w", i, err)
		}

		if first[0] == headerTerminator {
			return fields, nil
		}

		buf[0] = first[0]
		if _, err := io.ReadFull(r, buf[1:]); err != nil {
			return nil, fmt.Errorf("dbf: reading field descriptor %d: %w", i, err)
		}

		fd, err := readFieldDescriptor(buf[:])
		if err != nil {
			return nil, err
		}
		fields = append(fields, *fd)
	}

	return fields, nil
}
