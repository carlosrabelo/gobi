package term

import (
	"bytes"
	"strings"
	"testing"
)

func TestScreenDefaultAndCSI(t *testing.T) {
	s := New()
	if s.Cols != 80 || s.Rows != 24 {
		t.Fatalf("size %dx%d", s.Cols, s.Rows)
	}
	s.Put(0, 0, 'A', false)
	var out bytes.Buffer
	if err := s.Present(&out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "\x1b[1;1H") {
		t.Fatalf("CSI for 0,0 should be 1;1: %q", got)
	}
	if !strings.Contains(got, "A") {
		t.Fatalf("missing A: %q", got)
	}

	out.Reset()
	if err := s.Present(&out); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("unchanged cell should skip: %q", out.String())
	}
}

func TestResizeClamp(t *testing.T) {
	s := NewSize(10, 5)
	if s.Cols != MinCols || s.Rows != MinRows {
		t.Fatalf("clamp: %dx%d", s.Cols, s.Rows)
	}
	s.Put(-1, 0, 'X', false)
	s.Put(0, 1000, 'Y', false)
	if s.CellAt(0, 0).Ch != ' ' {
		t.Fatal("out of range write")
	}
}
