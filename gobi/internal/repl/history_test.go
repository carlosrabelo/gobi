package repl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHistoryCapDropsOldest(t *testing.T) {
	h := NewHistory("")
	h.Cap = 500
	for i := 1; i <= 501; i++ {
		h.Add(itoa(i))
	}
	if h.Len() != 500 {
		t.Fatalf("len: got %d, want 500", h.Len())
	}
	newest, ok := h.Prev()
	if !ok || newest != "501" {
		t.Fatalf("newest: got %q ok=%v, want 501", newest, ok)
	}
	if h.lines[0] != "2" {
		t.Fatalf("oldest: got %q, want 2", h.lines[0])
	}
}

func TestHistorySkipsDuplicate(t *testing.T) {
	h := NewHistory("")
	h.Add("LIST")
	h.Add("LIST")
	if h.Len() != 1 {
		t.Fatalf("len: got %d, want 1", h.Len())
	}
	h.Add("USE")
	h.Add("LIST")
	if h.Len() != 3 {
		t.Fatalf("len: got %d, want 3", h.Len())
	}
}

func TestHistoryLoadSave(t *testing.T) {
	dir := t.TempDir()
	h := NewHistory(dir)
	h.Add("LIST")
	h.Add("QUIT")
	if err := h.Save(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ".gobi_history")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	h2 := NewHistory(dir)
	if err := h2.Load(); err != nil {
		t.Fatal(err)
	}
	if h2.Len() != 2 {
		t.Fatalf("len: got %d", h2.Len())
	}
	got, ok := h2.Prev()
	if !ok || got != "QUIT" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
}

func TestHistoryMissingFile(t *testing.T) {
	h := NewHistory(t.TempDir())
	if err := h.Load(); err != nil {
		t.Fatal(err)
	}
	if h.Len() != 0 {
		t.Fatalf("len: got %d", h.Len())
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
