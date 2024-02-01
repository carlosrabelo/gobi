package term

import (
	"os"
	"testing"
)

func TestSizeNilFile(t *testing.T) {
	if _, _, err := Size(nil); err != ErrNotTerminal {
		t.Fatalf("expected ErrNotTerminal, got %v", err)
	}
}

func TestSizeRejectsPipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	if _, _, err := Size(r); err != ErrNotTerminal {
		t.Fatalf("expected ErrNotTerminal, got %v", err)
	}
}

func TestSizeOnTerminalWhenAvailable(t *testing.T) {
	tty, err := os.Open("/dev/tty")
	if err != nil {
		t.Skip("no controlling terminal available")
	}
	defer tty.Close()

	cols, rows, err := Size(tty)
	if err != nil {
		t.Skipf("terminal size unavailable: %v", err)
	}
	if cols <= 0 || rows <= 0 {
		t.Fatalf("Size returned non-positive dimensions: %dx%d", cols, rows)
	}
}
