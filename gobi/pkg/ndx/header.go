// Package ndx implements the dBase II NDX index file format.
package ndx

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

const (
	// PageSize is the fixed size of every NDX page in bytes.
	PageSize = 512

	// MaxKeyLength is the maximum indexed key width in bytes.
	MaxKeyLength = 100

	expressionOffset = 10
	expressionSize   = 400
	paddingOffset    = 410
	paddingSize      = 102
)

// KeyType identifies the indexed key representation.
type KeyType uint16

const (
	KeyTypeCharacter KeyType = 0
	KeyTypeNumeric   KeyType = 1
)

// Header describes page 0 metadata for an NDX file.
type Header struct {
	RootPageID     uint16
	PageCount      uint16
	KeyLength      uint16
	MaxKeysPerPage uint16
	KeyType        KeyType
	Expression     string
}

// ReadHeader reads the 512-byte page 0 header from r.
func ReadHeader(r io.Reader) (*Header, error) {
	var page [PageSize]byte
	if _, err := io.ReadFull(r, page[:]); err != nil {
		return nil, fmt.Errorf("ndx: reading header page: %w", err)
	}
	return parseHeaderPage(page[:])
}

// WriteHeader writes h as the 512-byte page 0 header to w.
func WriteHeader(w io.Writer, h *Header) error {
	page, err := marshalHeaderPage(h)
	if err != nil {
		return err
	}
	if _, err := w.Write(page[:]); err != nil {
		return fmt.Errorf("ndx: writing header page: %w", err)
	}
	return nil
}

func parseHeaderPage(page []byte) (*Header, error) {
	if len(page) != PageSize {
		return nil, fmt.Errorf("ndx: header page must be %d bytes", PageSize)
	}

	h := &Header{
		RootPageID:     binary.LittleEndian.Uint16(page[0:2]),
		PageCount:      binary.LittleEndian.Uint16(page[2:4]),
		KeyLength:      binary.LittleEndian.Uint16(page[4:6]),
		MaxKeysPerPage: binary.LittleEndian.Uint16(page[6:8]),
		KeyType:        KeyType(binary.LittleEndian.Uint16(page[8:10])),
		Expression:     readExpression(page[expressionOffset : expressionOffset+expressionSize]),
	}

	if err := validateHeaderFields(h); err != nil {
		return nil, err
	}
	return h, nil
}

func marshalHeaderPage(h *Header) ([PageSize]byte, error) {
	var page [PageSize]byte
	if err := validateHeader(h); err != nil {
		return page, err
	}

	binary.LittleEndian.PutUint16(page[0:2], h.RootPageID)
	binary.LittleEndian.PutUint16(page[2:4], h.PageCount)
	binary.LittleEndian.PutUint16(page[4:6], h.KeyLength)
	binary.LittleEndian.PutUint16(page[6:8], h.MaxKeysPerPage)
	binary.LittleEndian.PutUint16(page[8:10], uint16(h.KeyType))
	writeExpression(page[expressionOffset:expressionOffset+expressionSize], h.Expression)
	// bytes [410:512) remain zero padding

	return page, nil
}

func validateHeader(h *Header) error {
	if err := validateHeaderFields(h); err != nil {
		return err
	}
	if len(h.Expression) >= expressionSize {
		return fmt.Errorf("ndx: expression exceeds %d bytes", expressionSize-1)
	}
	return nil
}

func validateHeaderFields(h *Header) error {
	if h == nil {
		return fmt.Errorf("ndx: nil header")
	}
	if h.KeyLength > MaxKeyLength {
		return fmt.Errorf("ndx: key length %d exceeds maximum %d", h.KeyLength, MaxKeyLength)
	}
	if h.KeyType != KeyTypeCharacter && h.KeyType != KeyTypeNumeric {
		return fmt.Errorf("ndx: invalid key type %d", h.KeyType)
	}
	return nil
}

func readExpression(field []byte) string {
	if idx := strings.IndexByte(string(field), 0); idx >= 0 {
		return string(field[:idx])
	}
	return strings.TrimRight(string(field), "\x00")
}

func writeExpression(field []byte, expression string) {
	copy(field, []byte(expression))
}
