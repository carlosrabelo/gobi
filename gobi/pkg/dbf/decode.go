package dbf

import (
	"bytes"
	"fmt"
)

func (r *Record) DecodeField(tbl *Table, index int) (interface{}, error) {
	raw := r.FieldData(tbl, index)
	if raw == nil {
		return nil, fmt.Errorf("dbf: field index %d out of range", index)
	}

	fd := tbl.Fields[index]
	switch fd.Type {
	case FieldTypeChar:
		return decodeChar(raw), nil
	default:
		return nil, fmt.Errorf("dbf: unsupported field type %c for field %q", byte(fd.Type), fd.Name)
	}
}

func decodeChar(raw []byte) string {
	return string(bytes.TrimRight(raw, " "))
}
