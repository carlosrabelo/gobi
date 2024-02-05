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

func TestReadKeyBracketedPaste(t *testing.T) {
	// ESC [ 200 ~ hello ESC [ 201 ~ then 'x'
	in := "\x1b[200~hello\x1b[201~x"
	r := NewReader(strings.NewReader(in))
	want := []int{int('h'), int('e'), int('l'), int('l'), int('o'), int('x')}
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

func TestReadKeyBracketedPasteNewlinesToSpaces(t *testing.T) {
	in := "\x1b[200~a\nb\x1b[201~"
	r := NewReader(strings.NewReader(in))
	want := []int{int('a'), int(' '), int('b')}
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

func TestIsInputByte(t *testing.T) {
	if !IsInputByte(int('A')) || !IsInputByte(0xc3) || IsInputByte(KeyBackspace) || IsInputByte(KeyUp) {
		t.Fatal("IsInputByte range wrong")
	}
}

func TestReadKeyInsert(t *testing.T) {
	r := NewReader(strings.NewReader("\x1b[2~"))
	got, err := r.ReadKey()
	if err != nil {
		t.Fatal(err)
	}
	if got != KeyIns {
		t.Fatalf("got %d want KeyIns", got)
	}
}

func TestDrainPendingLineEnds(t *testing.T) {
	r := NewReader(strings.NewReader("\r\n\x1b"))
	r.DrainPendingLineEnds()
	got, err := r.ReadKey()
	if err != nil {
		t.Fatal(err)
	}
	if got != KeyEsc {
		t.Fatalf("got %d want Esc; leftover line ends were not drained", got)
	}
	// Non line-ending must stay available.
	r2 := NewReader(strings.NewReader("x"))
	r2.DrainPendingLineEnds()
	got, err = r2.ReadKey()
	if err != nil {
		t.Fatal(err)
	}
	if got != int('x') {
		t.Fatalf("got %d want 'x'", got)
	}
}
