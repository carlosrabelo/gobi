package hostterm

import (
	"bytes"
	"io"
	"testing"
)

type unwrapW struct{ w io.Writer }

func (u unwrapW) Write(p []byte) (int, error) { return u.w.Write(p) }
func (u unwrapW) Unwrap() io.Writer           { return u.w }

func TestApplyAutoFloorsAndPassthrough(t *testing.T) {
	c, r := ApplyAuto(20, 8)
	if c != MinCols || r != MinRows {
		t.Fatalf("floors: got %dx%d, want %dx%d", c, r, MinCols, MinRows)
	}
	c, r = ApplyAuto(120, 40)
	if c != 120 || r != 40 {
		t.Fatalf("passthrough: got %dx%d", c, r)
	}
}

func TestSizeOfPipeDefaults(t *testing.T) {
	c, r := SizeOf(bytes.NewReader(nil))
	if c != DefaultCols || r != DefaultRows {
		t.Fatalf("pipe size: got %dx%d", c, r)
	}
}

func TestIsTerminalPipe(t *testing.T) {
	if IsTerminal(bytes.NewReader(nil)) {
		t.Fatal("pipe should not be a terminal")
	}
}

func TestCapsOfPipe(t *testing.T) {
	caps := CapsOf(&bytes.Buffer{})
	if caps.CursorAddress || caps.Clear || caps.Reverse || caps.Color {
		t.Fatalf("pipe caps should be false: %+v", caps)
	}
}

func TestSizeOfWriterUnwrapsCRLFStyle(t *testing.T) {
	var buf bytes.Buffer
	c, r := SizeOfWriter(unwrapW{&buf})
	if c != DefaultCols || r != DefaultRows {
		t.Fatalf("wrapped pipe: got %dx%d", c, r)
	}
	if IsTerminalWriter(unwrapW{&buf}) {
		t.Fatal("wrapped pipe must not look like a terminal")
	}
	if FileFromWriter(unwrapW{&buf}) != nil {
		t.Fatal("FileFromWriter on buffer")
	}
}
