package dbf

import (
	"fmt"
	"io"
	"time"
)

// Create writes a new empty DBF with the given field descriptors.
func Create(w io.WriteSeeker, fields []FieldDescriptor) (*Table, error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("dbf: at least one field required")
	}
	if len(fields) > maxFieldCount {
		return nil, fmt.Errorf("dbf: too many fields (max %d)", maxFieldCount)
	}

	recLen := uint16(1)
	for _, fd := range fields {
		if err := validateFieldDescriptor(fd); err != nil {
			return nil, err
		}
		recLen += uint16(fd.Length)
	}
	if recLen > 1000 {
		return nil, fmt.Errorf("dbf: record length %d exceeds maximum 1000", recLen)
	}

	now := time.Now()
	var header [8]byte
	header[0] = SignatureStd
	header[3] = byte(now.Year() - 1900)
	header[4] = byte(now.Month())
	header[5] = byte(now.Day())
	header[6] = byte(recLen)
	header[7] = byte(recLen >> 8)

	if _, err := w.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("dbf: seeking to start: %w", err)
	}
	if _, err := w.Write(header[:]); err != nil {
		return nil, fmt.Errorf("dbf: writing header: %w", err)
	}

	for _, fd := range fields {
		if _, err := w.Write(encodeFieldDescriptor(fd)); err != nil {
			return nil, fmt.Errorf("dbf: writing field descriptor: %w", err)
		}
	}
	if _, err := w.Write([]byte{headerTerminator}); err != nil {
		return nil, fmt.Errorf("dbf: writing header terminator: %w", err)
	}
	if err := WriteEOF(w); err != nil {
		return nil, fmt.Errorf("dbf: writing EOF marker: %w", err)
	}

	offsets := make([]int, len(fields)+1)
	for i, f := range fields {
		offsets[i+1] = offsets[i] + int(f.Length)
	}

	return &Table{
		Header: &Header{
			Signature:   SignatureStd,
			RecordCount: 0,
			RecordLen:   recLen,
		},
		Fields: fields,
		Offset: offsets,
	}, nil
}

func encodeFieldDescriptor(fd FieldDescriptor) []byte {
	fb := make([]byte, fieldDescriptorSize)
	nameBytes := []byte(fd.Name)
	for i := 0; i < fieldNameLen; i++ {
		if i < len(nameBytes) {
			fb[i] = nameBytes[i]
		}
	}
	fb[10] = byte(fd.Type)
	fb[11] = fd.Length
	fb[14] = fd.DecimalCount
	return fb
}

func validateFieldDescriptor(fd FieldDescriptor) error {
	if err := ValidateFieldName(fd.Name); err != nil {
		return err
	}
	if !validFieldTypes[fd.Type] {
		return fmt.Errorf("dbf: invalid field type %q for field %q", byte(fd.Type), fd.Name)
	}
	if fd.Length == 0 {
		return fmt.Errorf("dbf: field %q width must be greater than zero", fd.Name)
	}

	switch fd.Type {
	case FieldTypeChar:
		if fd.Length > 254 {
			return fmt.Errorf("dbf: field %q width exceeds maximum 254", fd.Name)
		}
	case FieldTypeNumeric:
		if fd.DecimalCount > fd.Length-2 && fd.Length >= 2 {
			return fmt.Errorf("dbf: field %q decimal places too large for width", fd.Name)
		}
	case FieldTypeLogical:
		if fd.Length != 1 {
			return fmt.Errorf("dbf: logical field %q width must be 1", fd.Name)
		}
	}

	return nil
}

// ValidateFieldName checks dBase II field naming rules.
func ValidateFieldName(name string) error {
	if name == "" {
		return fmt.Errorf("dbf: field name required")
	}
	if len(name) > fieldNameLen {
		return fmt.Errorf("dbf: field name %q exceeds maximum length %d", name, fieldNameLen)
	}
	if !isASCIILetter(name[0]) {
		return fmt.Errorf("dbf: field name %q must start with a letter", name)
	}
	for i := 1; i < len(name); i++ {
		c := name[i]
		if !isASCIILetter(c) && !isASCIIDigit(c) && c != ':' {
			return fmt.Errorf("dbf: invalid character %q in field name %q", c, name)
		}
	}
	return nil
}

func isASCIILetter(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func isASCIIDigit(c byte) bool {
	return c >= '0' && c <= '9'
}
