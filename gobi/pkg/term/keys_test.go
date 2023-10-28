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
