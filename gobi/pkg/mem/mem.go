package mem

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

const headerSize = 32

// Entry is one variable in a .mem file.
type Entry struct {
	Name  string
	Value expr.Value
}

// WriteAll writes entries to w in .mem format (types C, N, L only).
func WriteAll(w io.Writer, entries []Entry) error {
	for _, e := range entries {
		if err := writeEntry(w, e); err != nil {
			return err
		}
	}
	return nil
}

func writeEntry(w io.Writer, e Entry) error {
	var hdr [headerSize]byte
	name := strings.ToUpper(e.Name)
	if len(name) > 10 {
		return fmt.Errorf("name too long: %s", name)
	}
	copy(hdr[0:], name)
	// NUL already at remaining bytes

	switch e.Value.Kind {
	case expr.KindString:
		hdr[11] = 'C'
		n := len(e.Value.Str)
		if n > 255 {
			return fmt.Errorf("character value too long")
		}
		hdr[16] = byte(n)
		if _, err := w.Write(hdr[:]); err != nil {
			return err
		}
		if n > 0 {
			_, err := w.Write([]byte(e.Value.Str))
			return err
		}
		return nil
	case expr.KindNumber:
		hdr[11] = 'N'
		binary.LittleEndian.PutUint64(hdr[18:], math.Float64bits(e.Value.Num))
		_, err := w.Write(hdr[:])
		return err
	case expr.KindLogical:
		hdr[11] = 'L'
		if e.Value.Log {
			hdr[18] = 1
		}
		_, err := w.Write(hdr[:])
		return err
	default:
		return fmt.Errorf("unsupported value kind")
	}
}

// ReadAll reads every entry until EOF. A type D header returns an error.
func ReadAll(r io.Reader) ([]Entry, error) {
	var out []Entry
	for {
		var hdr [headerSize]byte
		n, err := io.ReadFull(r, hdr[:])
		if err == io.EOF || (err == io.ErrUnexpectedEOF && n == 0) {
			return out, nil
		}
		if err == io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("truncated .mem header")
		}
		if err != nil {
			return nil, err
		}
		e, err := decodeHeader(hdr)
		if err != nil {
			return nil, err
		}
		if hdr[11] == 'C' {
			length := int(hdr[16])
			if length > 0 {
				buf := make([]byte, length)
				if _, err := io.ReadFull(r, buf); err != nil {
					return nil, fmt.Errorf("truncated character value: %w", err)
				}
				e.Value = expr.StringVal(string(buf))
			} else {
				e.Value = expr.StringVal("")
			}
		}
		out = append(out, e)
	}
}

func decodeHeader(hdr [headerSize]byte) (Entry, error) {
	nameBytes := hdr[0:11]
	end := 0
	for end < len(nameBytes) && nameBytes[end] != 0 {
		end++
	}
	name := string(nameBytes[:end])
	typ := hdr[11]
	switch typ {
	case 'C':
		return Entry{Name: name}, nil
	case 'N':
		bits := binary.LittleEndian.Uint64(hdr[18:])
		return Entry{Name: name, Value: expr.NumberVal(math.Float64frombits(bits))}, nil
	case 'L':
		return Entry{Name: name, Value: expr.LogicalVal(hdr[18] != 0)}, nil
	case 'D':
		return Entry{}, fmt.Errorf("date memory variables are not supported yet")
	default:
		return Entry{}, fmt.Errorf("unknown .mem type %q", typ)
	}
}

// WriteFile creates or truncates path and writes entries.
func WriteFile(path string, entries []Entry) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return WriteAll(f, entries)
}

// ReadFile opens path and reads all entries.
func ReadFile(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadAll(f)
}
