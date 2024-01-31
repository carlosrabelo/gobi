package hostterm

import (
	"bytes"
	"strings"
	"testing"
)

func TestPresentNoopWhenCapsFalse(t *testing.T) {
	var buf bytes.Buffer
	caps := Caps{}
	if err := MoveCursor(&buf, caps, 1, 2); err != nil {
		t.Fatal(err)
	}
	if err := Clear(&buf, caps); err != nil {
		t.Fatal(err)
	}
	if err := SetReverse(&buf, caps, true); err != nil {
		t.Fatal(err)
	}
	if err := SetColor(&buf, caps, 1, 2); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected no output, got %q", buf.String())
	}
}

func TestPresentANSIWhenCapsTrue(t *testing.T) {
	var buf bytes.Buffer
	caps := Caps{CursorAddress: true, Clear: true, Reverse: true, Color: true}
	_ = MoveCursor(&buf, caps, 0, 0)
	_ = Clear(&buf, caps)
	_ = SetReverse(&buf, caps, true)
	_ = SetColor(&buf, caps, 1, ColorDefault)
	s := buf.String()
	if !strings.Contains(s, "\x1b[1;1H") || !strings.Contains(s, "\x1b[2J") || !strings.Contains(s, "\x1b[7m") {
		t.Fatalf("ansi: %q", s)
	}
}
