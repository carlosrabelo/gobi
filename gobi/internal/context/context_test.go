package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewTenAreas(t *testing.T) {
	ctx := New()
	if ctx.ActiveArea != 1 {
		t.Fatalf("ActiveArea: got %d, want 1", ctx.ActiveArea)
	}
	if ctx.Config == nil {
		t.Fatal("Config is nil")
	}
	for i := 1; i <= AreaCount; i++ {
		a := ctx.Areas[i]
		if a == nil {
			t.Fatalf("area %d is nil", i)
		}
		if a.Cursor != 0 || a.Found || a.BOF {
			t.Fatalf("area %d: cursor=%d found=%v bof=%v", i, a.Cursor, a.Found, a.BOF)
		}
	}
	if ctx.Areas[0] != nil {
		t.Fatal("area 0 should be unused")
	}
}

func TestSelectNumberAndLetter(t *testing.T) {
	ctx := New()
	if err := ctx.Select("5"); err != nil {
		t.Fatalf("Select 5: %v", err)
	}
	if ctx.ActiveArea != 5 {
		t.Fatalf("got %d, want 5", ctx.ActiveArea)
	}
	if err := ctx.Select("J"); err != nil {
		t.Fatalf("Select J: %v", err)
	}
	if ctx.ActiveArea != 10 {
		t.Fatalf("got %d, want 10", ctx.ActiveArea)
	}
	if err := ctx.Select("a"); err != nil {
		t.Fatalf("Select a: %v", err)
	}
	if ctx.ActiveArea != 1 {
		t.Fatalf("got %d, want 1", ctx.ActiveArea)
	}
}

func TestSelectRejectsEleven(t *testing.T) {
	ctx := New()
	_ = ctx.Select("3")
	err := ctx.Select("11")
	if err == nil {
		t.Fatal("expected error for area 11")
	}
	if ctx.ActiveArea != 3 {
		t.Fatalf("active area changed to %d after rejection", ctx.ActiveArea)
	}
	err = ctx.Select("K")
	if err == nil {
		t.Fatal("expected error for K")
	}
	if ctx.ActiveArea != 3 {
		t.Fatalf("active area changed to %d after K", ctx.ActiveArea)
	}
}

func TestStdinReaderSharedAndRebuilt(t *testing.T) {
	ctx := New()
	ctx.Stdin = strings.NewReader("first\nsecond\n")

	r1 := ctx.StdinReader()
	line, err := r1.ReadString('\n')
	if err != nil || line != "first\n" {
		t.Fatalf("first line: %q (%v)", line, err)
	}
	r2 := ctx.StdinReader()
	if r1 != r2 {
		t.Fatal("expected the same reader")
	}
	line, err = r2.ReadString('\n')
	if err != nil || line != "second\n" {
		t.Fatalf("second line: %q (%v)", line, err)
	}

	ctx.Stdin = strings.NewReader("new\n")
	r3 := ctx.StdinReader()
	if r3 == r1 {
		t.Fatal("expected a new reader after Stdin replace")
	}
	line, err = r3.ReadString('\n')
	if err != nil || line != "new\n" {
		t.Fatalf("new line: %q (%v)", line, err)
	}
}

func TestResolveDefaultAndPath(t *testing.T) {
	dir := t.TempDir()
	other := t.TempDir()
	mustWrite(t, filepath.Join(dir, "here.dbf"), []byte("x"))
	mustWrite(t, filepath.Join(other, "there.dbf"), []byte("y"))

	ctx := New()
	ctx.Config.Default = dir
	ctx.Config.Path = ""

	got, err := ctx.Resolve("here.dbf")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(dir, "here.dbf") {
		t.Fatalf("got %q", got)
	}

	ctx.Config.Path = other
	got, err = ctx.Resolve("there.dbf")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(other, "there.dbf") {
		t.Fatalf("got %q", got)
	}
}

func TestResolveUTF8HostPath(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "dados-ação")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(dir, "aqui.dbf"), []byte("x"))

	ctx := New()
	ctx.Config.Default = dir
	got, err := ctx.Resolve("aqui.dbf")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(dir, "aqui.dbf") {
		t.Fatalf("got %q", got)
	}
}

func mustWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
