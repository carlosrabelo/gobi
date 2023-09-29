package dbf

import (
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

type FieldDescriptor struct {
	Name         string
	Type         FieldType
	Length       byte
	DecimalCount byte
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
		fields = append(fields, FieldDescriptor{})
	}
	return fields, nil
}
