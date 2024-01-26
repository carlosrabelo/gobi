package term

import (
	"fmt"
	"io"
	"os"
	"syscall"
	"unsafe"
)

type termiosState struct {
	termios syscall.Termios
}

// IsTerminal reports whether file refers to a terminal device.
func IsTerminal(file *os.File) bool {
	if file == nil {
		return false
	}
	return isTerminalFd(file.Fd())
}

func isTerminalFd(fd uintptr) bool {
	var termios syscall.Termios
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, fd, syscall.TCGETS, uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
	return err == 0
}

func enableRawMode(fd uintptr) (*termiosState, error) {
	var oldState syscall.Termios
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, fd, syscall.TCGETS, uintptr(unsafe.Pointer(&oldState)), 0, 0, 0); err != 0 {
		return nil, fmt.Errorf("term: tcgets: %v", err)
	}

	newState := oldState
	newState.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	newState.Oflag &^= syscall.OPOST
	newState.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	newState.Cflag &^= syscall.CSIZE | syscall.PARENB
	newState.Cflag |= syscall.CS8
	newState.Cc[syscall.VMIN] = 1
	newState.Cc[syscall.VTIME] = 0

	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, fd, syscall.TCSETS, uintptr(unsafe.Pointer(&newState)), 0, 0, 0); err != 0 {
		return nil, fmt.Errorf("term: tcsets: %v", err)
	}

	return &termiosState{termios: oldState}, nil
}

func restoreRawMode(fd uintptr, state *termiosState) {
	if state == nil {
		return
	}
	syscall.Syscall6(syscall.SYS_IOCTL, fd, syscall.TCSETS, uintptr(unsafe.Pointer(&state.termios)), 0, 0, 0)
}

// ErrNotTerminal indicates raw mode is unavailable for the input file.
var ErrNotTerminal = fmt.Errorf("term: not a terminal")

// RawMode captures keyboard input byte-at-a-time without line buffering.
type RawMode struct {
	in    *os.File
	state *termiosState
}

// EnterRawMode enables raw mode on in and returns a restore handle.
func EnterRawMode(in *os.File) (*RawMode, error) {
	if !IsTerminal(in) {
		return nil, ErrNotTerminal
	}
	state, err := enableRawMode(in.Fd())
	if err != nil {
		return nil, err
	}
	return &RawMode{in: in, state: state}, nil
}

// Close restores the previous terminal attributes.
func (m *RawMode) Close() error {
	if m == nil {
		return nil
	}
	restoreRawMode(m.in.Fd(), m.state)
	m.state = nil
	return nil
}

// Keyboard reads decoded keys from a raw terminal input stream.
type Keyboard struct {
	in     io.Reader
	buffer [1]byte
}

// NewKeyboard returns a keyboard adapter for in.
func NewKeyboard(in io.Reader) *Keyboard {
	return &Keyboard{in: in}
}

// KeyKind classifies a decoded terminal key event.
type KeyKind int

const (
	KeyByte KeyKind = iota
	KeyEnter
	KeyBackspace
	KeyEscape
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyShiftTab
)

// Key is one decoded terminal key press.
type Key struct {
	Kind KeyKind
	Byte byte
}

// ReadKey reads and decodes the next key from the input stream.
func (k *Keyboard) ReadKey() (Key, error) {
	ch, err := k.readByte()
	if err != nil {
		return Key{}, err
	}

	switch ch {
	case 10, 13:
		return Key{Kind: KeyEnter}, nil
	case 8, 127:
		return Key{Kind: KeyBackspace}, nil
	case 27:
		seq, err := k.readEscapeSequence()
		if err != nil {
			return Key{Kind: KeyEscape}, nil
		}
		switch seq {
		case "[A":
			return Key{Kind: KeyUp}, nil
		case "[B":
			return Key{Kind: KeyDown}, nil
		case "[C":
			return Key{Kind: KeyRight}, nil
		case "[D":
			return Key{Kind: KeyLeft}, nil
		case "[Z":
			return Key{Kind: KeyShiftTab}, nil
		default:
			return Key{Kind: KeyEscape}, nil
		}
	default:
		return Key{Kind: KeyByte, Byte: ch}, nil
	}
}

func (k *Keyboard) readByte() (byte, error) {
	n, err := k.in.Read(k.buffer[:])
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, io.EOF
	}
	return k.buffer[0], nil
}

func (k *Keyboard) readEscapeSequence() (string, error) {
	seq := make([]byte, 0, 2)
	for i := 0; i < 2; i++ {
		ch, err := k.readByte()
		if err != nil {
			return string(seq), err
		}
		seq = append(seq, ch)
	}
	return string(seq), nil
}
