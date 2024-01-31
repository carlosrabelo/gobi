package term

import (
	"errors"
	"io"
	"os"
	"time"
)

// Key codes for ReadKey. Arrow codes are distinct from Ctrl letters so full-screen
// forms can tell ↓ from Ctrl-X; INKEY() folds them to classic III Plus numbers
// via InkeyNumber.
const (
	KeyCtrlQ     = 17
	KeyCtrlU     = 21
	KeyCtrlW     = 23
	KeyCtrlX     = 24
	KeyCtrlY     = 25
	KeyCtrlR     = 18
	KeyTab       = 9
	KeyCtrlC     = 3 // cancel; must not kill the process at the prompt
	KeyEnter     = 13
	KeyEsc       = 27
	KeyBackspace = 127
	KeyUp        = 327 // CSI/SS3; INKEY reports 5
	KeyDown      = 328 // CSI/SS3; INKEY reports 24 (same as Ctrl-X)
	KeyLeft      = 329 // CSI/SS3; INKEY reports 19
	KeyRight     = 330 // CSI/SS3; INKEY reports 4
	KeyPgUp      = 29
	KeyPgDn      = 30
	KeyF1        = 315
	KeyF12       = 326
)

// IsInputByte reports whether k is a host text byte (ASCII printable or a
// UTF-8 octet). Special keys (arrows, F-keys, Ctrl letters already matched)
// fall outside this range. Backspace (127) is excluded.
func IsInputByte(k int) bool {
	return (k >= 32 && k < 127) || (k >= 128 && k <= 255)
}

// InkeyNumber maps a ReadKey code to the classic INKEY() integer.
func InkeyNumber(k int) int {
	switch k {
	case KeyUp:
		return 5
	case KeyDown:
		return 24
	case KeyLeft:
		return 19
	case KeyRight:
		return 4
	default:
		return k
	}
}

// How long to wait for the next byte after ESC before treating it as bare Esc.
// Modern terminals send CSI / SS3 immediately; keep this short so ASSIST Esc
// leave feels instant (III Plus left the bar on the first Esc).
const escInterByte = 25 * time.Millisecond

// Reader reads keys from a byte stream (tests inject bytes; terminals use raw mode).
type Reader struct {
	r    io.Reader
	buf  []byte
	file *os.File // optional; enables Esc inter-byte deadline / poll
}

// NewReader wraps r.
func NewReader(r io.Reader) *Reader {
	kr := &Reader{r: r}
	if f, ok := r.(*os.File); ok {
		kr.file = f
	}
	return kr
}

// DrainPendingLineEnds discards CR/LF still sitting on the input after the
// prompt consumed Enter. Full-screen UIs (ASSIST) must call this on entry so a
// leftover LF is not read as KeyEnter and does not open a submenu.
func (kr *Reader) DrainPendingLineEnds() {
	for {
		b, ok := kr.tryNonblockByte()
		if !ok {
			if kr.file == nil {
				return
			}
			// Brief poll only — never block (TTY SetReadDeadline often fails).
			if !kr.waitReadable(5 * time.Millisecond) {
				return
			}
			var one [1]byte
			_, err := kr.file.Read(one[:])
			if err != nil {
				return
			}
			b = one[0]
		}
		if b == '\r' || b == '\n' {
			continue
		}
		kr.unread(b)
		return
	}
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
			// CRLF: consume a following LF only if already available. Never
			// block — raw Enter is a lone CR, and a blocking peek would steal
			// the next keystroke (or lose it when the Reader is replaced).
			if next, ok := kr.tryNonblockByte(); ok && next != '\n' {
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
	b2, err := kr.readByteDeadline(escInterByte)
	if err != nil {
		return KeyEsc, nil
	}
	if b2 == 'O' {
		// SS3: ESC O A–D = arrows (application cursor keys);
		// ESC O P–S = F1–F4.
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
		case 'P', 'Q', 'R', 'S':
			return KeyF1 + int(b3-'P'), nil
		default:
			return KeyEsc, nil
		}
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
		// ESC [ n ~ = F-keys; ESC [ 1 ; mod A–D = modified arrows.
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
				if bn == ';' {
					// Skip modifier parameters until a final byte.
					for {
						bm, err := kr.readByte()
						if err != nil {
							return KeyEsc, nil
						}
						if (bm >= '0' && bm <= '9') || bm == ';' {
							continue
						}
						bn = bm
						break
					}
				}
				switch bn {
				case '~':
					// Bracketed paste (xterm): ESC [ 200 ~ … ESC [ 201 ~
					if num == 200 {
						return kr.consumeBracketedPaste()
					}
					if num == 201 {
						return kr.ReadKey() // stray end — ignore
					}
					if n, ok := fkeyFromCSI(num); ok {
						return n, nil
					}
					return KeyEsc, nil
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
		return KeyEsc, nil
	}
}

func fkeyFromCSI(n int) (int, bool) {
	// xterm: 5/6 = PgUp/PgDn; 11–15 F1–F5, 17–21 F6–F10, 23–24 F11–F12
	switch n {
	case 5:
		return KeyPgUp, true
	case 6:
		return KeyPgDn, true
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
		return InkeyNumber(k)
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
		return InkeyNumber(k)
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

// tryNonblockByte returns the next byte only if it will not block.
func (kr *Reader) tryNonblockByte() (byte, bool) {
	if len(kr.buf) > 0 {
		b := kr.buf[0]
		kr.buf = kr.buf[1:]
		return b, true
	}
	if s, ok := kr.r.(interface{ Len() int }); ok {
		if s.Len() == 0 {
			return 0, false
		}
		b, err := kr.readByte()
		return b, err == nil
	}
	if kr.file != nil {
		if !kr.waitReadable(0) {
			return 0, false
		}
		b, err := kr.readByte()
		return b, err == nil
	}
	return 0, false
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "deadline exceeded" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

var errReadTimeout = timeoutError{}

// readByteDeadline reads one byte, timing out after d. TTYs often reject
// os.File.SetReadDeadline ("file type does not support deadline"); then we
// poll(2) so bare Esc does not block until the next keystroke.
func (kr *Reader) readByteDeadline(d time.Duration) (byte, error) {
	if len(kr.buf) > 0 {
		return kr.readByte()
	}
	if kr.file != nil && d > 0 {
		if err := kr.file.SetReadDeadline(time.Now().Add(d)); err == nil {
			defer func() { _ = kr.file.SetReadDeadline(time.Time{}) }()
			b, err := kr.readByte()
			if err != nil && isTimeout(err) {
				return 0, err
			}
			return b, err
		}
		if !kr.waitReadable(d) {
			return 0, errReadTimeout
		}
	}
	return kr.readByte()
}

func isTimeout(err error) bool {
	var to interface{ Timeout() bool }
	return errors.As(err, &to) && to.Timeout()
}

func (kr *Reader) unread(b byte) {
	kr.buf = append([]byte{b}, kr.buf...)
}

// consumeBracketedPaste reads until ESC [ 201 ~ and queues the pasted bytes
// (newlines → spaces so a form field is not split). Then returns the next key.
func (kr *Reader) consumeBracketedPaste() (int, error) {
	var pasted []byte
	for {
		b, err := kr.readByte()
		if err != nil {
			break
		}
		if b != 0x1b {
			if b == '\r' || b == '\n' {
				pasted = append(pasted, ' ')
				continue
			}
			if b >= 32 || b == '\t' {
				if b == '\t' {
					b = ' '
				}
				pasted = append(pasted, b)
			}
			continue
		}
		b2, err := kr.readByte()
		if err != nil {
			break
		}
		if b2 != '[' {
			kr.unread(b2)
			continue
		}
		num := 0
		ended := false
		for {
			bn, err := kr.readByte()
			if err != nil {
				ended = true
				break
			}
			if bn >= '0' && bn <= '9' {
				num = num*10 + int(bn-'0')
				continue
			}
			if bn == '~' && num == 201 {
				ended = true
			}
			break
		}
		if ended {
			break
		}
	}
	for i := len(pasted) - 1; i >= 0; i-- {
		kr.unread(pasted[i])
	}
	if len(pasted) == 0 {
		return kr.ReadKey()
	}
	return kr.ReadKey()
}
