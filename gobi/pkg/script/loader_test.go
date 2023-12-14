package script

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathAppendsPrg(t *testing.T) {
	got := ResolvePath("/tmp", "deptlist")
	want := filepath.Join("/tmp", "deptlist.prg")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolvePathKeepsExtension(t *testing.T) {
	got := ResolvePath("/tmp", "backup.bak")
	want := filepath.Join("/tmp", "backup.bak")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolvePathAbsoluteIgnoresDefaultDir(t *testing.T) {
	got := ResolvePath("/tmp", "/var/scripts/run.prg")
	if got != "/var/scripts/run.prg" {
		t.Fatalf("expected absolute path unchanged, got %q", got)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(t.TempDir(), "missing")
	if err == nil {
		t.Fatal("expected error for missing command file")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestLoadExistingFile(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "hello.prg")
	if err := os.WriteFile(path, []byte("STORE 1 TO x\r\n"), 0644); err != nil {
		t.Fatalf("write prg: %v", err)
	}

	prog, err := Load(tempDir, "hello")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if prog.Path != path {
		t.Fatalf("expected path %q, got %q", path, prog.Path)
	}
	if len(prog.Commands()) != 1 || prog.Commands()[0].Command.Verb != "STORE" {
		t.Fatalf("unexpected parsed commands: %#v", prog.Commands())
	}
}

func TestLoadBakExtension(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "backup.bak")
	if err := os.WriteFile(path, []byte("QUIT\r\n"), 0644); err != nil {
		t.Fatalf("write bak: %v", err)
	}

	prog, err := Load(tempDir, "backup.bak")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if prog.Path != path {
		t.Fatalf("expected path %q, got %q", path, prog.Path)
	}
}

func TestLoadRejectsDirectory(t *testing.T) {
	tempDir := t.TempDir()
	dirPath := filepath.Join(tempDir, "folder.prg")
	if err := os.Mkdir(dirPath, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	_, err := Load(tempDir, "folder")
	if err == nil || err.Error() != "not a command file" {
		t.Fatalf("expected not a command file error, got %v", err)
	}
}
