package hostterm

import (
	"bytes"
	"io"
)

// Key codes shared with gobi/pkg/term ReadKey (spec 003). Arrows are distinct
// from Ctrl letters; INKEY folds via term.InkeyNumber.
const (
	KeyEnter     = 13
	KeyEsc       = 27
	KeyTab       = 9
	KeyBackspace = 127
	KeyUp        = 327
	KeyDown      = 328
	KeyLeft      = 329
	KeyRight     = 330
	KeyCtrlQ     = 17
	KeyCtrlW     = 23
	KeyCtrlU     = 21
	KeyCtrlY     = 25
	KeyPgUp      = 29
	KeyPgDn      = 30
)

// DecodeANSIKey reads one logical key from r (POSIX CSI/SS3 fixtures).
func DecodeANSIKey(r io.Reader) (int, error) {
	var one [1]byte
	if _, err := io.ReadFull(r, one[:]); err != nil {
		return 0, err
	}
	b := one[0]
	if b != 0x1b {
		switch b {
		case '\n', '\r':
			return KeyEnter, nil
		default:
			return int(b), nil
		}
	}
	if _, err := io.ReadFull(r, one[:]); err != nil {
		return KeyEsc, nil
	}
	b2 := one[0]
	if b2 == 'O' {
		if _, err := io.ReadFull(r, one[:]); err != nil {
			return KeyEsc, nil
		}
		switch one[0] {
		case 'A':
			return KeyUp, nil
		case 'B':
			return KeyDown, nil
		case 'C':
			return KeyRight, nil
		case 'D':
			return KeyLeft, nil
		default:
			return KeyEsc, nil
		}
	}
	if b2 != '[' {
		return KeyEsc, nil
	}
	if _, err := io.ReadFull(r, one[:]); err != nil {
		return KeyEsc, nil
	}
	b3 := one[0]
	switch b3 {
	case 'A':
		return KeyUp, nil
	case 'B':
		return KeyDown, nil
	case 'C':
		return KeyRight, nil
	case 'D':
		return KeyLeft, nil
	}
	if b3 < '0' || b3 > '9' {
		return KeyEsc, nil
	}
	num := int(b3 - '0')
	for {
		if _, err := io.ReadFull(r, one[:]); err != nil {
			return KeyEsc, nil
		}
		bn := one[0]
		if bn >= '0' && bn <= '9' {
			num = num*10 + int(bn-'0')
			continue
		}
		if bn == '~' {
			switch num {
			case 5:
				return KeyPgUp, nil
			case 6:
				return KeyPgDn, nil
			default:
				return KeyEsc, nil
			}
		}
		switch bn {
		case 'A':
			return KeyUp, nil
		case 'B':
			return KeyDown, nil
		case 'C':
			return KeyRight, nil
		case 'D':
			return KeyLeft, nil
		default:
			return KeyEsc, nil
		}
	}
}

// DecodeANSIKeyBytes decodes one key from a byte slice (test helper).
func DecodeANSIKeyBytes(b []byte) (int, error) {
	return DecodeANSIKey(bytes.NewReader(b))
}
