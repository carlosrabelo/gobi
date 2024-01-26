package term

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestKeyboardReadByteKey(t *testing.T) {
	kbd := NewKeyboard(bytes.NewReader([]byte{'A'}))
	key, err := kbd.ReadKey()
	if err != nil {
		t.Fatalf("ReadKey: %v", err)
	}
	if key.Kind != KeyByte || key.Byte != 'A' {
		t.Fatalf("unexpected key: %#v", key)
	}
}

func TestKeyboardReadSpecialKeys(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		kind KeyKind
	}{
		{name: "enter cr", in: []byte{13}, kind: KeyEnter},
		{name: "enter lf", in: []byte{10}, kind: KeyEnter},
		{name: "backspace del", in: []byte{127}, kind: KeyBackspace},
		{name: "backspace bs", in: []byte{8}, kind: KeyBackspace},
		{name: "up", in: []byte{27, '[', 'A'}, kind: KeyUp},
		{name: "down", in: []byte{27, '[', 'B'}, kind: KeyDown},
		{name: "right", in: []byte{27, '[', 'C'}, kind: KeyRight},
		{name: "left", in: []byte{27, '[', 'D'}, kind: KeyLeft},
		{name: "shift tab", in: []byte{27, '[', 'Z'}, kind: KeyShiftTab},
		{name: "escape", in: []byte{27}, kind: KeyEscape},
		{name: "unknown escape", in: []byte{27, '[', 'X'}, kind: KeyEscape},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			key, err := NewKeyboard(bytes.NewReader(tc.in)).ReadKey()
			if err != nil {
				t.Fatalf("ReadKey: %v", err)
			}
			if key.Kind != tc.kind {
				t.Fatalf("expected kind %v, got %#v", tc.kind, key)
			}
		})
	}
}

func TestKeyboardIncompleteEscape(t *testing.T) {
	key, err := NewKeyboard(bytes.NewReader([]byte{27, '['})).ReadKey()
	if err != nil {
		t.Fatalf("ReadKey: %v", err)
	}
	if key.Kind != KeyEscape {
		t.Fatalf("expected escape on truncated sequence, got %#v", key)
	}
}

func TestKeyboardEOF(t *testing.T) {
	_, err := NewKeyboard(bytes.NewReader(nil)).ReadKey()
	if err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestIsTerminalFalseForPipe(t *testing.T) {
	r, _, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	defer r.Close()

	if IsTerminal(r) {
		t.Fatal("expected pipe not to be a terminal")
	}
}

func TestEnterRawModeRejectsPipe(t *testing.T) {
	r, _, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	defer r.Close()

	if _, err := EnterRawMode(r); err != ErrNotTerminal {
		t.Fatalf("expected ErrNotTerminal, got %v", err)
	}
}

func TestRawModeCloseNil(t *testing.T) {
	var mode *RawMode
	if err := mode.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestIsTerminalNilFile(t *testing.T) {
	if IsTerminal(nil) {
		t.Fatal("expected nil file not to be a terminal")
	}
}
