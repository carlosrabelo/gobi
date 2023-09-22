package repl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHistoryAdd(t *testing.T) {
	h := NewHistory(5)
	h.Add("USE customers")
	h.Add("LIST")
	h.Add("QUIT")

	if len(h.All()) != 3 {
		t.Fatalf("expected 3 items, got %d", len(h.All()))
	}
}

func TestHistorySkipsEmpty(t *testing.T) {
	h := NewHistory(5)
	h.Add("")
	h.Add("LIST")
	h.Add("")

	if len(h.All()) != 1 {
		t.Fatalf("expected 1 item, got %d", len(h.All()))
	}
}

func TestHistorySkipsDuplicates(t *testing.T) {
	h := NewHistory(5)
	h.Add("LIST")
	h.Add("LIST")
	h.Add("LIST")

	if len(h.All()) != 1 {
		t.Fatalf("expected 1 item, got %d", len(h.All()))
	}
}

func TestHistoryPrevNext(t *testing.T) {
	h := NewHistory(10)
	h.Add("ONE")
	h.Add("TWO")
	h.Add("THREE")

	got, ok := h.Prev()
	if !ok || got != "THREE" {
		t.Fatalf("expected THREE, got %q", got)
	}

	got, ok = h.Prev()
	if !ok || got != "TWO" {
		t.Fatalf("expected TWO, got %q", got)
	}

	got, ok = h.Next()
	if !ok || got != "THREE" {
		t.Fatalf("expected THREE, got %q", got)
	}

	got, ok = h.Next()
	if ok || got != "" {
		t.Fatalf("expected no more items, got %q", got)
	}
}

func TestHistoryPrevAtStart(t *testing.T) {
	h := NewHistory(10)
	h.Add("ONLY")

	h.Prev()
	h.Prev()

	if _, ok := h.Prev(); ok {
		t.Fatal("expected false at start")
	}
}

func TestHistoryMaxSize(t *testing.T) {
	h := NewHistory(3)
	h.Add("A")
	h.Add("B")
	h.Add("C")
	h.Add("D")

	if len(h.All()) != 3 {
		t.Fatalf("expected 3 items, got %d", len(h.All()))
	}
	if h.All()[0] != "B" {
		t.Fatalf("expected first item B, got %q", h.All()[0])
	}
}

func TestHistorySaveLoad(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)

	h := NewHistory(10)
	h.Add("USE test")
	h.Add("LIST")
	h.Add("QUIT")

	if err := h.Save(); err != nil {
		t.Fatalf("save error: %v", err)
	}

	// Verify file exists
	path := filepath.Join(dir, historyFileName)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("history file not created: %v", err)
	}

	// Load into new history
	h2 := NewHistory(10)
	if err := h2.Load(); err != nil {
		t.Fatalf("load error: %v", err)
	}

	all := h2.All()
	if len(all) != 3 {
		t.Fatalf("expected 3 items after load, got %d", len(all))
	}
	if all[0] != "USE test" {
		t.Fatalf("expected 'USE test', got %q", all[0])
	}
}

func TestHistoryLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)

	h := NewHistory(10)
	if err := h.Load(); err != nil {
		t.Fatalf("load of missing file should not error: %v", err)
	}
}

func TestHistoryReset(t *testing.T) {
	h := NewHistory(10)
	h.Add("A")
	h.Add("B")

	h.Prev()
	h.Prev()
	h.Reset()

	if _, ok := h.Next(); ok {
		t.Fatal("expected no next after reset to end")
	}
}
