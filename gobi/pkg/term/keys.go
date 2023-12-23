package term

import (
	"io"
)

// Key codes matching the constitution / INKEY values.
const (
	KeyCtrlQ     = 17
	KeyCtrlU     = 21
	KeyCtrlW     = 23
	KeyCtrlX     = 24 // also down arrow alias in INKEY is 24
	KeyCtrlY     = 25
	KeyCtrlR     = 18
	KeyTab       = 9
	KeyCtrlC     = 3 // cancel; must not kill the process at the prompt
	KeyEnter     = 13
	KeyEsc       = 27
	KeyBackspace = 127
	KeyUp        = 5
	KeyDown      = 24
	KeyLeft      = 19
	KeyRight     = 4
	KeyF1        = 315
	KeyF12       = 326
)

// Reader reads keys from a byte stream (tests inject bytes; terminals use raw mode).
type Reader struct {
	r   io.Reader
	buf []byte
}

// NewReader wraps r.
func NewReader(r io.Reader) *Reader {
	return &Reader{r: r}
}

// ReadKey returns the next logical key as an INKEY-style number.
func (kr *Reader) ReadKey() (int, error) {
	b, err := kr.readByte()
	if err != nil {
		return 0, err
	}
	if b != 0x1b {
		switch b {
		case '\n':
			// Unix cooked Enter is LF; treat as KeyEnter (CR).
			return KeyEnter, nil
		case KeyEnter: // '\r'
			// CRLF: consume the following LF so it is not a second Enter.
			if next, err := kr.readByte(); err == nil && next != '\n' {
				kr.unread(next)
			}
			return KeyEnter, nil
		case KeyTab, KeyEsc, KeyBackspace,
			KeyCtrlQ, KeyCtrlW, KeyCtrlU, KeyCtrlY, KeyCtrlR, KeyCtrlX:
			return int(b), nil
		default:
			return int(b), nil
		}
	}
	b2, err := kr.readByte()
	if err != nil {
		return KeyEsc, nil
	}
	if b2 == 'O' {
		// ESC O P–S = F1–F4
		b3, err := kr.readByte()
		if err != nil {
			return KeyEsc, nil
		}
		if b3 >= 'P' && b3 <= 'S' {
			return KeyF1 + int(b3-'P'), nil
		}
		return KeyEsc, nil
	}
	if b2 != '[' {
		kr.unread(b2)
		return KeyEsc, nil
	}
	b3, err := kr.readByte()
	if err != nil {
		return KeyEsc, nil
	}
	switch b3 {
	case 'A':
		return KeyUp, nil
	case 'B':
		return KeyDown, nil
	case 'C':
		return KeyRight, nil
	case 'D':
		return KeyLeft, nil
	default:
		// ESC [ 1 1 ~ … ESC [ 2 4 ~ = F1–F12
		if b3 >= '0' && b3 <= '9' {
			num := int(b3 - '0')
			for {
				bn, err := kr.readByte()
				if err != nil {
					return KeyEsc, nil
				}
				if bn >= '0' && bn <= '9' {
					num = num*10 + int(bn-'0')
					continue
				}
				if bn == '~' {
					if n, ok := fkeyFromCSI(num); ok {
						return n, nil
					}
				}
				return KeyEsc, nil
			}
		}
		return KeyEsc, nil
	}
}

func fkeyFromCSI(n int) (int, bool) {
	// xterm: 11–15 F1–F5, 17–21 F6–F10, 23–24 F11–F12
	switch n {
	case 11, 12, 13, 14, 15:
		return KeyF1 + (n - 11), true
	case 17, 18, 19, 20, 21:
		return KeyF1 + (n - 17 + 5), true
	case 23, 24:
		return KeyF1 + (n - 23 + 10), true
	default:
		return 0, false
	}
}

// Inkey returns the next waiting key without blocking. If the underlying reader
// is not ready, returns 0. For test streams with buffered unread data, consumes one key.
func (kr *Reader) Inkey() int {
	if len(kr.buf) > 0 {
		k, err := kr.ReadKey()
		if err != nil {
			return 0
		}
		return k
	}
	// Non-blocking: try a one-byte read via type assert when possible.
	type sized interface{ Len() int }
	if s, ok := kr.r.(sized); ok && s.Len() == 0 {
		return 0
	}
	// Best effort: if Read would block we cannot know; tests preload via strings.Reader.
	if s, ok := kr.r.(interface{ Len() int }); ok && s.Len() > 0 {
		k, err := kr.ReadKey()
		if err != nil {
			return 0
		}
		return k
	}
	return 0
}

func (kr *Reader) readByte() (byte, error) {
	if len(kr.buf) > 0 {
		b := kr.buf[0]
		kr.buf = kr.buf[1:]
		return b, nil
	}
	var one [1]byte
	_, err := kr.r.Read(one[:])
	return one[0], err
}

func (kr *Reader) unread(b byte) {
	kr.buf = append([]byte{b}, kr.buf...)
}
