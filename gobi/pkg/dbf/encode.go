package dbf

import (
	"fmt"
	"strconv"
	"strings"
)

// EncodeRecord packs a record into RecordLen bytes for the given fields.
func EncodeRecord(fields []Field, rec Record) ([]byte, error) {
	recLen, err := ValidateFields(fields)
	if err != nil {
		return nil, err
	}
	if len(rec.Values) != len(fields) {
		return nil, fmt.Errorf("value count %d != field count %d", len(rec.Values), len(fields))
	}
	raw := make([]byte, recLen)
	if rec.Deleted {
		raw[0] = markDeleted
	} else {
		raw[0] = markActive
	}
	off := 1
	for i, f := range fields {
		chunk, err := encodeField(f, rec.Values[i])
		if err != nil {
			return nil, err
		}
		copy(raw[off:off+int(f.Length)], chunk)
		off += int(f.Length)
	}
	return raw, nil
}

func encodeField(f Field, v Value) ([]byte, error) {
	n := int(f.Length)
	out := make([]byte, n)
	switch f.Type {
	case 'C':
		s := v.Str
		if len(s) > n {
			s = s[:n]
		}
		copy(out, s)
		for i := len(s); i < n; i++ {
			out[i] = ' '
		}
	case 'N':
		var s string
		if f.Decimals == 0 {
			s = strconv.FormatFloat(v.Num, 'f', 0, 64)
		} else {
			s = strconv.FormatFloat(v.Num, 'f', int(f.Decimals), 64)
		}
		if len(s) > n {
			for i := range out {
				out[i] = '*'
			}
			return out, nil
		}
		pad := n - len(s)
		for i := 0; i < pad; i++ {
			out[i] = ' '
		}
		copy(out[pad:], s)
	case 'L':
		if v.Log {
			out[0] = 'T'
		} else {
			out[0] = 'F'
		}
	case 'D':
		if v.Empty || v.Str == "" {
			for i := range out {
				out[i] = ' '
			}
		} else {
			s := v.Str
			if len(s) != 8 {
				return nil, fmt.Errorf("date must be YYYYMMDD")
			}
			copy(out, s)
		}
	case 'M':
		if v.Empty || v.Str == "" {
			for i := range out {
				out[i] = ' '
			}
		} else {
			s := strings.TrimSpace(v.Str)
			if len(s) > n {
				return nil, fmt.Errorf("memo block too long")
			}
			pad := n - len(s)
			for i := 0; i < pad; i++ {
				out[i] = ' '
			}
			copy(out[pad:], s)
		}
	default:
		return nil, fmt.Errorf("bad type %c", f.Type)
	}
	return out, nil
}
