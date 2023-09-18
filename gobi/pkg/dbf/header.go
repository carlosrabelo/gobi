package dbf

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

const (
	sigNoMemo   = 0x03
	sigMemo     = 0x83
	sigDB2      = 0x02
	terminator  = 0x0D
	maxFields   = 128
	maxRecLen   = 4000
	headerBase  = 32
	fieldDescSz = 32
)

// Field is one column descriptor.
type Field struct {
	Name     string
	Type     byte // C, N, L, D, or M
	Length   byte
	Decimals byte
}

// Header is the DBF file header and field descriptors.
type Header struct {
	Signature   byte
	Year        byte // year - 1900
	Month       byte
	Day         byte
	RecordCount uint32
	HeaderLen   uint16
	RecordLen   uint16
	Fields      []Field
}

// ReadHeader reads the header and descriptors from r.
func ReadHeader(r io.Reader) (Header, error) {
	var buf [headerBase]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return Header{}, err
	}
	h := Header{
		Signature:   buf[0],
		Year:        buf[1],
		Month:       buf[2],
		Day:         buf[3],
		RecordCount: binary.LittleEndian.Uint32(buf[4:8]),
		HeaderLen:   binary.LittleEndian.Uint16(buf[8:10]),
		RecordLen:   binary.LittleEndian.Uint16(buf[10:12]),
	}
	if h.Signature != sigNoMemo && h.Signature != sigMemo {
		return Header{}, fmt.Errorf("unsupported DBF signature 0x%02X", h.Signature)
	}
	if h.HeaderLen < headerBase+1 {
		return Header{}, fmt.Errorf("header length %d too small", h.HeaderLen)
	}
	nFields := int(h.HeaderLen-headerBase-1) / fieldDescSz
	if nFields > maxFields {
		return Header{}, fmt.Errorf("too many fields: %d", nFields)
	}
	h.Fields = make([]Field, 0, nFields)
	for i := 0; i < nFields; i++ {
		var fd [fieldDescSz]byte
		if _, err := io.ReadFull(r, fd[:]); err != nil {
			return Header{}, err
		}
		f, err := parseField(fd[:])
		if err != nil {
			return Header{}, err
		}
		h.Fields = append(h.Fields, f)
	}
	var term [1]byte
	if _, err := io.ReadFull(r, term[:]); err != nil {
		return Header{}, err
	}
	if term[0] != terminator {
		return Header{}, fmt.Errorf("missing field terminator")
	}
	if err := checkHeaderSizes(h); err != nil {
		return Header{}, err
	}
	return h, nil
}

func parseField(fd []byte) (Field, error) {
	name := strings.TrimRight(string(fd[0:11]), "\x00")
	name = strings.ToUpper(name)
	if len(name) > 10 {
		return Field{}, fmt.Errorf("field name too long: %q", name)
	}
	f := Field{
		Name:     name,
		Type:     fd[11],
		Length:   fd[16],
		Decimals: fd[17],
	}
	if err := validateField(f); err != nil {
		return Field{}, err
	}
	return f, nil
}

// ValidateFields checks field rules and returns the record length (including delete mark).
func ValidateFields(fields []Field) (recLen uint16, err error) {
	if len(fields) > maxFields {
		return 0, fmt.Errorf("too many fields: %d", len(fields))
	}
	sum := 1 // deletion mark
	for _, f := range fields {
		if err := validateField(f); err != nil {
			return 0, err
		}
		sum += int(f.Length)
	}
	if sum > maxRecLen {
		return 0, fmt.Errorf("record length %d exceeds %d", sum, maxRecLen)
	}
	return uint16(sum), nil
}

func validateField(f Field) error {
	if f.Length == 0 {
		return fmt.Errorf("field %s: length 0", f.Name)
	}
	switch f.Type {
	case 'C', 'N', 'L', 'D', 'M':
	default:
		return fmt.Errorf("field %s: bad type %c", f.Name, f.Type)
	}
	if f.Type == 'D' && f.Length != 8 {
		return fmt.Errorf("field %s: D length must be 8", f.Name)
	}
	if f.Type == 'M' && f.Length != 10 {
		return fmt.Errorf("field %s: M length must be 10", f.Name)
	}
	if f.Type == 'L' && f.Length != 1 {
		return fmt.Errorf("field %s: L length must be 1", f.Name)
	}
	if f.Type == 'N' && int(f.Decimals) >= int(f.Length) {
		return fmt.Errorf("field %s: decimals >= length", f.Name)
	}
	if f.Type != 'N' && f.Decimals != 0 {
		return fmt.Errorf("field %s: decimals must be 0", f.Name)
	}
	return nil
}

func checkHeaderSizes(h Header) error {
	wantRec, err := ValidateFields(h.Fields)
	if err != nil {
		return err
	}
	if h.RecordLen != wantRec {
		return fmt.Errorf("record length mismatch: header %d, fields %d", h.RecordLen, wantRec)
	}
	wantHdr := uint16(headerBase + fieldDescSz*len(h.Fields) + 1)
	if h.HeaderLen != wantHdr {
		return fmt.Errorf("header length mismatch: header %d, want %d", h.HeaderLen, wantHdr)
	}
	return nil
}

// WriteHeader writes the header, descriptors, and terminator.
func WriteHeader(w io.Writer, h Header) error {
	recLen, err := ValidateFields(h.Fields)
	if err != nil {
		return err
	}
	h.RecordLen = recLen
	h.HeaderLen = uint16(headerBase + fieldDescSz*len(h.Fields) + 1)
	hasMemo := false
	for _, f := range h.Fields {
		if f.Type == 'M' {
			hasMemo = true
			break
		}
	}
	if hasMemo {
		h.Signature = sigMemo
	} else {
		h.Signature = sigNoMemo
	}
	var buf [headerBase]byte
	buf[0] = h.Signature
	buf[1] = h.Year
	buf[2] = h.Month
	buf[3] = h.Day
	binary.LittleEndian.PutUint32(buf[4:8], h.RecordCount)
	binary.LittleEndian.PutUint16(buf[8:10], h.HeaderLen)
	binary.LittleEndian.PutUint16(buf[10:12], h.RecordLen)
	if _, err := w.Write(buf[:]); err != nil {
		return err
	}
	for _, f := range h.Fields {
		var fd [fieldDescSz]byte
		name := strings.ToUpper(f.Name)
		if len(name) > 10 {
			return fmt.Errorf("field name too long: %q", name)
		}
		copy(fd[0:11], name)
		fd[11] = f.Type
		fd[16] = f.Length
		fd[17] = f.Decimals
		if _, err := w.Write(fd[:]); err != nil {
			return err
		}
	}
	_, err = w.Write([]byte{terminator})
	return err
}

// HasMemo reports whether any field is type M.
func (h Header) HasMemo() bool {
	for _, f := range h.Fields {
		if f.Type == 'M' {
			return true
		}
	}
	return false
}
