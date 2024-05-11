package repl

import (
	"bytes"
	"testing"
)

func TestRedrawLineErasesLeftovers(t *testing.T) {
	out := &bytes.Buffer{}
	r := newTerminalReader(nil, out, ". ")

	r.redrawLine([]byte("USE"), 3)

	got := out.String()
	want := "\r. USE\x1b[K"
	if got != want {
		t.Fatalf("redraw output = %q, want %q", got, want)
	}
}

func TestRedrawLineEmptyBuffer(t *testing.T) {
	out := &bytes.Buffer{}
	r := newTerminalReader(nil, out, ". ")

	r.redrawLine(nil, 0)

	got := out.String()
	want := "\r. \x1b[K"
	if got != want {
		t.Fatalf("redraw output = %q, want %q", got, want)
	}
}

func TestRedrawLinePositionsCursorMidLine(t *testing.T) {
	out := &bytes.Buffer{}
	r := newTerminalReader(nil, out, ". ")

	r.redrawLine([]byte("LIST"), 1)

	got := out.String()
	want := "\r. LIST\x1b[K\x1b[3D"
	if got != want {
		t.Fatalf("redraw output = %q, want %q", got, want)
	}
}
