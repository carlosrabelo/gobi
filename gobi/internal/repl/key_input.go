package repl

import (
	"io"

	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

const (
	replKeyUp byte = 0x80 + iota
	replKeyDown
	replKeyLeft
	replKeyRight
)

func replKeyByte(k term.Key) (byte, error) {
	switch k.Kind {
	case term.KeyByte:
		return k.Byte, nil
	case term.KeyEnter:
		return editKeyEnter, nil
	case term.KeyBackspace:
		return editKeyDel, nil
	case term.KeyShiftTab:
		return editKeyCtrlK, nil
	case term.KeyEscape:
		return editKeyEsc, nil
	case term.KeyUp:
		return replKeyUp, nil
	case term.KeyDown:
		return replKeyDown, nil
	case term.KeyLeft:
		return replKeyLeft, nil
	case term.KeyRight:
		return replKeyRight, nil
	default:
		return editKeyEsc, nil
	}
}

func readReplKey(kbd *term.Keyboard, raw bool, in io.Reader) (byte, error) {
	if raw && kbd != nil {
		key, err := kbd.ReadKey()
		if err != nil {
			return 0, err
		}
		return replKeyByte(key)
	}

	var buf [1]byte
	if _, err := io.ReadFull(in, buf[:]); err != nil {
		return 0, err
	}
	return buf[0], nil
}
