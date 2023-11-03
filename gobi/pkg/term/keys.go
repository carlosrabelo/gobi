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
	KeyEnter     = 13
	KeyEsc       = 27
	KeyBackspace = 127
	KeyUp        = 5
	KeyDown      = 24
	KeyLeft      = 19
	KeyRight     = 4
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
		case KeyTab, KeyEnter, KeyEsc, KeyBackspace,
			KeyCtrlQ, KeyCtrlW, KeyCtrlU, KeyCtrlY, KeyCtrlR, KeyCtrlX:
			return int(b), nil
		default:
			return int(b), nil
		}
	}
	// ESC [ A/B/C/D arrows
	b2, err := kr.readByte()
	if err != nil {
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
		return KeyEsc, nil
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
