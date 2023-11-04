package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestETwoCommands(t *testing.T) {
	code := run([]string{"-e", `STORE 1 TO X; STORE 2 TO Y`})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
}

func TestEWithFileRejected(t *testing.T) {
	code := run([]string{"-e", "QUIT", "x.prg"})
	if code != 1 {
		t.Fatalf("exit %d", code)
	}
}

func TestUnknownVerbExit1(t *testing.T) {
	code := run([]string{"-e", "NOSUCH"})
	if code != 1 {
		t.Fatalf("exit %d", code)
	}
}

func TestScriptArg(t *testing.T) {
	dir := t.TempDir()
	prg := filepath.Join(dir, "t.prg")
	if err := os.WriteFile(prg, []byte("STORE 1 TO A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// DO needs DEFAULT; set via -e first is separate. Script alone uses cwd.
	old, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer os.Chdir(old)
	code := run([]string{"t.prg"})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
}
