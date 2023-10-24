package ndx

import (
	"encoding/binary"
	"fmt"
	"math"
)

// Entry is one index slot: optional child page, record number, and key bytes.
type Entry struct {
	Child uint32
	RecNo uint32
	Key   []byte // length KeyLen (padded for character keys)
}

// EncodeKeyChar space-pads s to keyLen.
func EncodeKeyChar(s string, keyLen int) []byte {
	key := make([]byte, keyLen)
	for i := range key {
		key[i] = ' '
	}
	copy(key, s)
	return key
}

// EncodeKeyNum stores n as IEEE-754 binary64.
func EncodeKeyNum(n float64) []byte {
	key := make([]byte, 8)
	binary.LittleEndian.PutUint64(key, math.Float64bits(n))
	return key
}

// DecodeKeyNum reads an IEEE-754 binary64 key.
func DecodeKeyNum(key []byte) float64 {
	return math.Float64frombits(binary.LittleEndian.Uint64(key[:8]))
}

// EncodeEntry writes one entry into dst (must be entrySize bytes).
func EncodeEntry(dst []byte, e Entry, keyLen int) error {
	size := EntrySizeOf(keyLen)
	if len(dst) < size {
		return fmt.Errorf("short entry buffer")
	}
	if len(e.Key) != keyLen {
		return fmt.Errorf("key length %d, want %d", len(e.Key), keyLen)
	}
	binary.LittleEndian.PutUint32(dst[0:4], e.Child)
	binary.LittleEndian.PutUint32(dst[4:8], e.RecNo)
	copy(dst[8:8+keyLen], e.Key)
	for i := 8 + keyLen; i < size; i++ {
		dst[i] = 0
	}
	return nil
}

// DecodeEntry reads one entry from src.
func DecodeEntry(src []byte, keyLen int) (Entry, error) {
	size := EntrySizeOf(keyLen)
	if len(src) < size {
		return Entry{}, fmt.Errorf("short entry")
	}
	e := Entry{
		Child: binary.LittleEndian.Uint32(src[0:4]),
		RecNo: binary.LittleEndian.Uint32(src[4:8]),
		Key:   make([]byte, keyLen),
	}
	copy(e.Key, src[8:8+keyLen])
	return e, nil
}

// CompareKeys compares two key byte slices for the given type.
func CompareKeys(a, b []byte, typ KeyType) int {
	if typ == KeyNum {
		na := DecodeKeyNum(a)
		nb := DecodeKeyNum(b)
		switch {
		case na < nb:
			return -1
		case na > nb:
			return 1
		default:
			return 0
		}
	}
	// Character: byte-wise compare over key length (space-padded).
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	switch {
	case len(a) < len(b):
		return -1
	case len(a) > len(b):
		return 1
	default:
		return 0
	}
}
