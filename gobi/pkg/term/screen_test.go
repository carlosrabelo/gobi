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

func TestResizePreservesOverlap(t *testing.T) {
	s := NewSize(80, 24)
	s.Put(0, 0, 'A', true)
	s.Put(5, 10, 'B', false)
	s.Resize(120, 40)
	if s.Cols != 120 || s.Rows != 40 {
		t.Fatalf("size %dx%d", s.Cols, s.Rows)
	}
	if s.CellAt(0, 0).Ch != 'A' || !s.CellAt(0, 0).Reverse {
		t.Fatalf("lost 0,0: %+v", s.CellAt(0, 0))
	}
	if s.CellAt(5, 10).Ch != 'B' {
		t.Fatalf("lost 5,10: %+v", s.CellAt(5, 10))
	}
	if s.CellAt(39, 119).Ch != ' ' {
		t.Fatal("new cell should be space")
	}
}

func TestPresentClearHomesCursor(t *testing.T) {
	s := NewSize(40, 10)
	s.Put(0, 0, 'X', false)
	var out bytes.Buffer
	if err := s.PresentClear(&out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.HasSuffix(got, "\x1b[H") {
		t.Fatalf("expected home after clear: %q", got)
	}
	if s.NextRow != 0 || s.NextCol != 0 {
		t.Fatalf("cursor %d,%d", s.NextRow, s.NextCol)
	}
}
