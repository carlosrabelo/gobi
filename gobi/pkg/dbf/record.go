package dbf

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	markActive  = 0x20
	markDeleted = 0x2A
	endMarker   = 0x1A
)

// Record is one decoded table row.
type Record struct {
	Deleted bool
	Values  []Value
}

// Value holds one decoded field.
type Value struct {
	Str   string  // C (trimmed), D (YYYYMMDD or ""), M (block digits or "")
	Num   float64 // N
	Log   bool    // L
	Empty bool    // empty D/M (all spaces)
}

// DecodeRecord decodes one record of exactly Header.RecordLen bytes.
func DecodeRecord(fields []Field, raw []byte) (Record, error) {
	if len(raw) == 0 {
		return Record{}, fmt.Errorf("empty record")
	}
	rec := Record{}
	switch raw[0] {
	case markActive:
		rec.Deleted = false
	case markDeleted:
		rec.Deleted = true
	default:
		return Record{}, fmt.Errorf("bad deletion mark 0x%02X", raw[0])
	}
	off := 1
	rec.Values = make([]Value, len(fields))
	for i, f := range fields {
		n := int(f.Length)
		if off+n > len(raw) {
			return Record{}, fmt.Errorf("short record")
		}
		chunk := raw[off : off+n]
		off += n
		v, err := decodeField(f, chunk)
		if err != nil {
			return Record{}, err
		}
		rec.Values[i] = v
	}
	return rec, nil
}

func decodeField(f Field, chunk []byte) (Value, error) {
	switch f.Type {
	case 'C':
		return Value{Str: strings.TrimRight(string(chunk), " ")}, nil
	case 'N':
		s := strings.TrimSpace(string(chunk))
		if s == "" {
			return Value{Num: 0}, nil
		}
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return Value{}, fmt.Errorf("bad number %q", s)
		}
		return Value{Num: n}, nil
	case 'L':
		c := chunk[0]
		switch c {
		case 'Y', 'y', 'T', 't':
			return Value{Log: true}, nil
		case 'N', 'n', 'F', 'f':
			return Value{Log: false}, nil
		case '?', ' ':
			return Value{Log: false}, nil
		default:
			return Value{}, fmt.Errorf("bad logical %q", c)
		}
	case 'D':
		s := string(chunk)
		if s == "        " {
			return Value{Empty: true}, nil
		}
		return Value{Str: s}, nil
	case 'M':
		s := string(chunk)
		if s == "          " {
			return Value{Empty: true}, nil
		}
		return Value{Str: strings.TrimSpace(s)}, nil
	default:
		return Value{}, fmt.Errorf("bad type %c", f.Type)
	}
}

// ReadRecords reads up to count records from r after the header.
// A leading 0x1A or clean EOF ends the table. A short non-1A record is an error.
func ReadRecords(r io.Reader, h Header) ([]Record, error) {
	recLen := int(h.RecordLen)
	out := make([]Record, 0, h.RecordCount)
	for i := uint32(0); i < h.RecordCount; i++ {
		raw := make([]byte, recLen)
		n, err := io.ReadFull(r, raw)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			if n == 0 {
				return out, fmt.Errorf("short table: expected %d records, got %d", h.RecordCount, i)
			}
			if n >= 1 && raw[0] == endMarker {
				return out, fmt.Errorf("short table: expected %d records, got %d", h.RecordCount, i)
			}
			return nil, fmt.Errorf("short record at %d", i+1)
		}
		if err != nil {
			return nil, err
		}
		if raw[0] == endMarker {
			return out, fmt.Errorf("short table: expected %d records, got %d", h.RecordCount, i)
		}
		rec, err := DecodeRecord(h.Fields, raw)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, nil
}

// PeekEnd reports whether the next byte is the end marker or EOF.
func PeekEnd(r io.ByteScanner) (bool, error) {
	b, err := r.ReadByte()
	if err == io.EOF {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if b == endMarker {
		return true, nil
	}
	if err := r.UnreadByte(); err != nil {
		return false, err
	}
	return false, nil
}
