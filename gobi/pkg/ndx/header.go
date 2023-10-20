package ndx

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

const (
	// PageSize is the fixed NDX page length.
	PageSize = 512
	// MaxKeyLen is the maximum key length in bytes.
	MaxKeyLen = 100
	// MaxExpr is the maximum expression length in the header.
	MaxExpr = 100
)

// KeyType is the header key-type field.
type KeyType uint16

const (
	KeyChar KeyType = 0
	KeyNum  KeyType = 1
)

// Header is page 0 of an NDX file.
type Header struct {
	Root      uint32
	PageCount uint32
	KeyLen    uint16
	MaxKeys   uint16
	Type      KeyType
	EntrySize uint32
	Unique    bool
	Expr      string
}

// EntrySizeOf returns 8 + keyLen rounded up to a multiple of 4.
func EntrySizeOf(keyLen int) int {
	n := 8 + keyLen
	if rem := n % 4; rem != 0 {
		n += 4 - rem
	}
	return n
}

// MaxKeysOf returns how many entries fit on a page.
func MaxKeysOf(entrySize int) int {
	if entrySize <= 0 {
		return 0
	}
	return (PageSize - 4) / entrySize
}

// ReadHeader reads page 0 from r (at current offset; typically start of file).
func ReadHeader(r io.Reader) (Header, error) {
	var buf [PageSize]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return Header{}, err
	}
	return DecodeHeader(buf[:])
}

// DecodeHeader parses a 512-byte page 0.
func DecodeHeader(buf []byte) (Header, error) {
	if len(buf) < PageSize {
		return Header{}, fmt.Errorf("short NDX header")
	}
	h := Header{
		Root:      binary.LittleEndian.Uint32(buf[0:4]),
		PageCount: binary.LittleEndian.Uint32(buf[4:8]),
		KeyLen:    binary.LittleEndian.Uint16(buf[12:14]),
		MaxKeys:   binary.LittleEndian.Uint16(buf[14:16]),
		Type:      KeyType(binary.LittleEndian.Uint16(buf[16:18])),
		EntrySize: binary.LittleEndian.Uint32(buf[18:22]),
		Unique:    buf[23] != 0,
	}
	if h.Type != KeyChar && h.Type != KeyNum {
		return Header{}, fmt.Errorf("invalid key type %d", h.Type)
	}
	if h.KeyLen == 0 || h.KeyLen > MaxKeyLen {
		return Header{}, fmt.Errorf("invalid key length %d", h.KeyLen)
	}
	if h.Type == KeyNum && h.KeyLen != 8 {
		return Header{}, fmt.Errorf("numeric key length must be 8")
	}
	wantEntry := uint32(EntrySizeOf(int(h.KeyLen)))
	if h.EntrySize != 0 && h.EntrySize != wantEntry {
		return Header{}, fmt.Errorf("entry size %d, want %d", h.EntrySize, wantEntry)
	}
	if h.EntrySize == 0 {
		h.EntrySize = wantEntry
	}
	wantMax := uint16(MaxKeysOf(int(h.EntrySize)))
	if h.MaxKeys != 0 && h.MaxKeys != wantMax {
		return Header{}, fmt.Errorf("max keys %d, want %d", h.MaxKeys, wantMax)
	}
	if h.MaxKeys == 0 {
		h.MaxKeys = wantMax
	}
	expr := buf[24 : 24+MaxExpr]
	end := 0
	for end < len(expr) && expr[end] != 0 {
		end++
	}
	h.Expr = string(expr[:end])
	return h, nil
}

// EncodeHeader writes page 0 into a 512-byte buffer.
func EncodeHeader(h Header) ([]byte, error) {
	if h.Type != KeyChar && h.Type != KeyNum {
		return nil, fmt.Errorf("invalid key type %d", h.Type)
	}
	if h.KeyLen == 0 || h.KeyLen > MaxKeyLen {
		return nil, fmt.Errorf("invalid key length %d", h.KeyLen)
	}
	if h.Type == KeyNum && h.KeyLen != 8 {
		return nil, fmt.Errorf("numeric key length must be 8")
	}
	expr := strings.ToLower(h.Expr)
	if len(expr) > MaxExpr {
		return nil, fmt.Errorf("expression longer than %d", MaxExpr)
	}
	h.EntrySize = uint32(EntrySizeOf(int(h.KeyLen)))
	h.MaxKeys = uint16(MaxKeysOf(int(h.EntrySize)))
	buf := make([]byte, PageSize)
	binary.LittleEndian.PutUint32(buf[0:4], h.Root)
	binary.LittleEndian.PutUint32(buf[4:8], h.PageCount)
	binary.LittleEndian.PutUint16(buf[12:14], h.KeyLen)
	binary.LittleEndian.PutUint16(buf[14:16], h.MaxKeys)
	binary.LittleEndian.PutUint16(buf[16:18], uint16(h.Type))
	binary.LittleEndian.PutUint32(buf[18:22], h.EntrySize)
	if h.Unique {
		buf[23] = 1
	}
	copy(buf[24:], expr)
	return buf, nil
}

// WriteHeader writes page 0 at the start of w (Seek 0 first if needed by caller).
func WriteHeader(w io.Writer, h Header) error {
	buf, err := EncodeHeader(h)
	if err != nil {
		return err
	}
	_, err = w.Write(buf)
	return err
}
