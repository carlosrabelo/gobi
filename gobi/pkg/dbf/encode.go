package dbf

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

func NewRecord(tbl *Table, deleted bool, values []interface{}) (*Record, error) {
	if len(values) != len(tbl.Fields) {
		return nil, fmt.Errorf("dbf: expected %d field values, got %d", len(tbl.Fields), len(values))
	}

	buf := make([]byte, tbl.Header.RecordLen)
	if deleted {
		buf[0] = deletionDeleted
	} else {
		buf[0] = deletionActive
	}

	for i, fd := range tbl.Fields {
		start := tbl.Offset[i] + 1
		encoded, err := encodeField(fd, values[i])
		if err != nil {
			return nil, fmt.Errorf("dbf: encoding field %q: %w", fd.Name, err)
		}
		copy(buf[start:], encoded)
	}

	return &Record{Deleted: deleted, Data: buf}, nil
}

func encodeField(fd FieldDescriptor, val interface{}) ([]byte, error) {
	switch fd.Type {
	case FieldTypeChar:
		return encodeChar(fd.Length, val), nil
	case FieldTypeNumeric:
		return encodeNumeric(fd.Length, fd.DecimalCount, val)
	case FieldTypeLogical:
		return encodeLogical(val), nil
	default:
		return nil, fmt.Errorf("unsupported field type %c", byte(fd.Type))
	}
}

func encodeChar(length byte, val interface{}) []byte {
	s := fmt.Sprintf("%v", val)
	if len(s) > int(length) {
		s = s[:length]
	}
	return []byte(fmt.Sprintf("%-*s", length, s))
}

func encodeNumeric(length byte, decimalCount byte, val interface{}) ([]byte, error) {
	var f float64
	switch v := val.(type) {
	case float64:
		f = v
	case int:
		f = float64(v)
	case string:
		if v == "" {
			return []byte(strings.Repeat(" ", int(length))), nil
		}
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid numeric %q", v)
		}
		f = parsed
	default:
		return nil, fmt.Errorf("invalid type %T for numeric field", val)
	}

	var s string
	if decimalCount > 0 {
		fmtStr := fmt.Sprintf("%%%d.%df", length, decimalCount)
		s = fmt.Sprintf(fmtStr, f)
	} else {
		fmtStr := fmt.Sprintf("%%%dd", length)
		s = fmt.Sprintf(fmtStr, int(f))
	}

	if len(s) > int(length) {
		s = strings.ReplaceAll(s, "-", "")
	}
	if len(s) > int(length) {
		s = strings.Repeat("*", int(length))
	}

	return []byte(s), nil
}

func encodeLogical(val interface{}) []byte {
	switch v := val.(type) {
	case bool:
		if v {
			return []byte{'T'}
		}
		return []byte{'F'}
	case string:
		if len(v) > 0 {
			return []byte{strings.ToUpper(v)[0]}
		}
		return []byte{'?'}
	default:
		return []byte{'?'}
	}
}

func (tbl *Table) WriteRecord(w io.Writer, rec *Record) error {
	if len(rec.Data) != int(tbl.Header.RecordLen) {
		return fmt.Errorf("dbf: record length %d does not match table record length %d", len(rec.Data), tbl.Header.RecordLen)
	}
	_, err := w.Write(rec.Data)
	return err
}
