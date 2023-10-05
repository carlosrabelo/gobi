package dbf

import (
	"bytes"
	"fmt"
	"strconv"
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
	case FieldTypeNumeric:
		return decodeNumeric(raw)
	case FieldTypeLogical:
		return decodeLogical(raw), nil
	default:
		return nil, fmt.Errorf("dbf: unsupported field type %c for field %q", byte(fd.Type), fd.Name)
	}
}

func decodeChar(raw []byte) string {
	return string(bytes.TrimRight(raw, " "))
}

func decodeNumeric(raw []byte) (float64, error) {
	s := string(bytes.TrimSpace(raw))
	if s == "" {
		return 0, nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("dbf: parsing numeric %q: %w", s, err)
	}
	return v, nil
}

func decodeLogical(raw []byte) bool {
	if len(raw) == 0 {
		return false
	}
	switch raw[0] {
	case 'T', 't', 'Y', 'y':
		return true
	default:
		return false
	}
}
