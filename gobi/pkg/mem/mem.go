// Package mem implements the dBase II binary memory variable file format (.mem).
package mem

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

const (
	headerSize = 19
	eofMarker  = 0x1A

	typeChar    = 0xC3
	typeNumeric = 0xCE
	typeLogical = 0xCC

	markerByte       = 'E'
	numericFieldLen  = 16
	logicalValueSize = 17
	maxNameLen       = 9
)

// Variable is a named memory variable value.
type Variable struct {
	Name  string
	Value interface{}
}

// Write serializes vars to w in dBase II .mem format and appends the EOF marker.
func Write(w io.Writer, vars []Variable) error {
	for _, v := range vars {
		if err := writeEntry(w, v.Name, v.Value); err != nil {
			return err
		}
	}
	_, err := w.Write([]byte{eofMarker})
	return err
}

// Read loads all variables from r until the dBase II EOF marker (0x1A).
func Read(r io.Reader) ([]Variable, error) {
	var vars []Variable
	for {
		hdr := make([]byte, headerSize)
		n, err := io.ReadFull(r, hdr)
		if err == io.EOF {
			return vars, nil
		}
		if err == io.ErrUnexpectedEOF {
			if n == 1 && hdr[0] == eofMarker {
				return vars, nil
			}
			return nil, fmt.Errorf("mem: truncated header")
		}
		if err != nil {
			return nil, fmt.Errorf("mem: reading header: %w", err)
		}
		if hdr[0] == eofMarker {
			return vars, nil
		}

		name := strings.TrimRight(string(hdr[:10]), "\x00")
		name = strings.TrimRight(name, "\x00")
		name = strings.TrimSpace(name)

		valueLen := int(hdr[12])
		if valueLen < 0 {
			return nil, fmt.Errorf("mem: invalid value length for %q", name)
		}

		valueData := make([]byte, valueLen)
		if _, err := io.ReadFull(r, valueData); err != nil {
			return nil, fmt.Errorf("mem: reading value for %q: %w", name, err)
		}

		val, err := decodeValue(hdr[11], hdr[16], hdr[17], valueData)
		if err != nil {
			return nil, fmt.Errorf("mem: decoding %q: %w", name, err)
		}

		vars = append(vars, Variable{Name: name, Value: val})
	}
}

func writeEntry(w io.Writer, name string, val interface{}) error {
	hdr := make([]byte, headerSize)

	nameBytes := []byte(strings.ToUpper(strings.TrimSpace(name)))
	if len(nameBytes) > maxNameLen {
		nameBytes = nameBytes[:maxNameLen]
	}
	copy(hdr[0:], nameBytes)

	var (
		typeByte     byte
		valueData    []byte
		fieldLen     byte
		decimalCount byte
		valueLen     byte
	)

	switch v := val.(type) {
	case string:
		typeByte = typeChar
		fieldLen = byte(len(v))
		valueLen = fieldLen
		valueData = encodeChar(fieldLen, v)
	case float64:
		typeByte = typeNumeric
		fieldLen = numericFieldLen
		decimalCount = numericDecimals(v)
		encoded, err := encodeNumeric(fieldLen, decimalCount, v)
		if err != nil {
			return fmt.Errorf("mem: encoding numeric %q: %w", name, err)
		}
		valueData = encoded
		valueLen = fieldLen
	case int:
		typeByte = typeNumeric
		fieldLen = numericFieldLen
		decimalCount = 0
		encoded, err := encodeNumeric(fieldLen, 0, float64(v))
		if err != nil {
			return fmt.Errorf("mem: encoding numeric %q: %w", name, err)
		}
		valueData = encoded
		valueLen = fieldLen
	case bool:
		typeByte = typeLogical
		fieldLen = 1
		valueLen = logicalValueSize
		valueData = encodeLogical(v)
	default:
		return fmt.Errorf("mem: unsupported type %T for %q", val, name)
	}

	hdr[11] = typeByte
	hdr[12] = valueLen
	hdr[15] = markerByte
	hdr[16] = fieldLen
	hdr[17] = decimalCount

	if _, err := w.Write(hdr); err != nil {
		return err
	}
	_, err := w.Write(valueData)
	return err
}

func encodeChar(fieldLen byte, val string) []byte {
	if fieldLen == 0 {
		return nil
	}
	buf := make([]byte, fieldLen)
	copy(buf[fieldLen-byte(len(val)):], []byte(val))
	return buf
}

func encodeNumeric(length byte, decimalCount byte, val float64) ([]byte, error) {
	var s string
	if decimalCount > 0 {
		fmtStr := fmt.Sprintf("%%%d.%df", length, decimalCount)
		s = fmt.Sprintf(fmtStr, val)
	} else {
		fmtStr := fmt.Sprintf("%%%dd", length)
		s = fmt.Sprintf(fmtStr, int(val))
	}

	if len(s) > int(length) {
		s = strings.ReplaceAll(s, "-", "")
	}
	if len(s) > int(length) {
		s = strings.Repeat("*", int(length))
	}

	return []byte(s), nil
}

func encodeLogical(val bool) []byte {
	buf := make([]byte, logicalValueSize)
	if val {
		buf[logicalValueSize-1] = 1
	}
	return buf
}

func numericDecimals(val float64) byte {
	if math.Mod(val, 1) == 0 {
		return 0
	}
	s := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.10f", val), "0"), ".")
	if idx := strings.IndexByte(s, '.'); idx >= 0 {
		return byte(len(s) - idx - 1)
	}
	return 0
}

func decodeValue(typeByte, _, decimalCount byte, data []byte) (interface{}, error) {
	switch typeByte {
	case typeChar:
		return strings.TrimLeft(string(data), "\x00"), nil
	case typeNumeric:
		s := strings.TrimSpace(string(data))
		if s == "" || strings.IndexFunc(s, func(r rune) bool {
			return r != ' ' && r != '*'
		}) < 0 {
			return float64(0), nil
		}
		if decimalCount > 0 {
			f, err := strconv.ParseFloat(s, 64)
			return f, err
		}
		i, err := strconv.ParseInt(s, 10, 64)
		return float64(i), err
	case typeLogical:
		return data[len(data)-1] != 0, nil
	default:
		return nil, fmt.Errorf("unknown type byte 0x%02X", typeByte)
	}
}
