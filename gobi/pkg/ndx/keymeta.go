package ndx

import (
	"fmt"
	"strconv"
	"strings"
)

// KeyFromText converts raw text into a normalized index key for h.
func KeyFromText(h *Header, text string) (Key, error) {
	if h == nil {
		return nil, fmt.Errorf("ndx: nil header")
	}
	switch h.KeyType {
	case KeyTypeCharacter:
		return normalizeKey(h, Key(text))
	case KeyTypeNumeric:
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			return normalizeKey(h, Key(""))
		}
		if _, err := strconv.ParseFloat(trimmed, 64); err != nil {
			return nil, fmt.Errorf("ndx: invalid numeric key %q", text)
		}
		return normalizeKey(h, Key(trimmed))
	default:
		return nil, fmt.Errorf("ndx: invalid key type %d", h.KeyType)
	}
}

// UpdateKeyMetadata tracks key width and type while scanning DBF records.
func UpdateKeyMetadata(keyType *KeyType, keyLength *uint16, text string, numeric bool) error {
	if numeric {
		*keyType = KeyTypeNumeric
	} else if *keyType != KeyTypeNumeric {
		*keyType = KeyTypeCharacter
	}

	width := uint16(len(text))
	if numeric {
		width = uint16(len(strings.TrimSpace(text)))
	}
	if width > MaxKeyLength {
		return fmt.Errorf("ndx: key length %d exceeds maximum %d", width, MaxKeyLength)
	}
	if width > *keyLength {
		*keyLength = width
	}
	if *keyLength == 0 {
		*keyLength = 1
	}
	return nil
}

// NewHeaderForExpression builds page 0 metadata for a new index file.
func NewHeaderForExpression(expression string, keyType KeyType, keyLength uint16) *Header {
	h := &Header{
		KeyLength:      keyLength,
		KeyType:        keyType,
		Expression:     expression,
		MaxKeysPerPage: uint16(maxLeafKeys(&Header{KeyLength: keyLength})),
	}
	return h
}
