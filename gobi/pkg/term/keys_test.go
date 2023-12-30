package term

import (
	"strings"
	"testing"
)

func TestReadKeyArrowsAndControls(t *testing.T) {
	r := NewReader(strings.NewReader("\x11\x17\x1b[A\x1b[B\x1b[C\x1b[D\r\t\x1b"))
	want := []int{KeyCtrlQ, KeyCtrlW, KeyUp, KeyDown, KeyRight, KeyLeft, KeyEnter, KeyTab, KeyEsc}
	for i, w := range want {
		got, err := r.ReadKey()
		if err != nil {
			t.Fatalf("%d: %v", i, err)
		}
		if got != w {
			t.Fatalf("%d: got %d want %d", i, got, w)
		}
	}
}

func TestReadKeyLFAndCRLFAsEnter(t *testing.T) {
	r := NewReader(strings.NewReader("a\nb\r\nc\r"))
	want := []int{int('a'), KeyEnter, int('b'), KeyEnter, int('c'), KeyEnter}
	for i, w := range want {
		got, err := r.ReadKey()
		if err != nil {
			t.Fatalf("%d: %v", i, err)
		}
		if got != w {
			t.Fatalf("%d: got %d want %d", i, got, w)
		}
	}
}

func TestReadKeySS3ArrowsAndModifiedCSI(t *testing.T) {
	// ESC O A–D = application cursor keys; ESC [ 1 ; 5 A = Ctrl+Up, etc.
	r := NewReader(strings.NewReader("\x1bOA\x1bOB\x1bOC\x1bOD\x1b[1;5A\x1b[1;5B\x1b[1;5C\x1b[1;5D"))
	want := []int{KeyUp, KeyDown, KeyRight, KeyLeft, KeyUp, KeyDown, KeyRight, KeyLeft}
	for i, w := range want {
		got, err := r.ReadKey()
		if err != nil {
			t.Fatalf("%d: %v", i, err)
		}
		if got != w {
			t.Fatalf("%d: got %d want %d", i, got, w)
		}
	}
}

func TestReadKeyBareEsc(t *testing.T) {
	r := NewReader(strings.NewReader("\x1b"))
	got, err := r.ReadKey()
	if err != nil {
		t.Fatal(err)
	}
	if got != KeyEsc {
		t.Fatalf("got %d want %d", got, KeyEsc)
	}
}
