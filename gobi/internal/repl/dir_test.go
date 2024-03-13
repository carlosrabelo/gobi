package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

func dirTestCtx(t *testing.T) *context.Context {
	t.Helper()
	tempDir := t.TempDir()

	rec := append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...)
	createTempDBFWithRecords(t, tempDir, "people.dbf", [][]byte{rec, rec})

	if err := os.WriteFile(filepath.Join(tempDir, "note.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ctx := testCtx()
	ctx.Config.DefaultDir = tempDir
	ctx.Stdout = &bytes.Buffer{}
	return ctx
}

func TestDispatchDisplayFilesListsDatabases(t *testing.T) {
	ctx := dirTestCtx(t)

	if err := commandMux.Dispatch(ctx, Command{Verb: "DISPLAY", Args: "FILES"}); err != nil {
		t.Fatalf("DISPLAY FILES: %v", err)
	}

	out := ctx.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(out, "DATABASE FILES") {
		t.Fatalf("expected header, got %q", out)
	}
	if !strings.Contains(out, "PEOPLE.DBF") {
		t.Fatalf("expected PEOPLE.DBF, got %q", out)
	}
	if !strings.Contains(out, "2") {
		t.Fatalf("expected record count 2, got %q", out)
	}
	if strings.Contains(out, "NOTE.TXT") {
		t.Fatalf("expected only databases in default listing, got %q", out)
	}
	if !strings.Contains(out, "1 FILE(S)") {
		t.Fatalf("expected file count, got %q", out)
	}
}

func TestDispatchListFilesLikePattern(t *testing.T) {
	ctx := dirTestCtx(t)

	if err := commandMux.Dispatch(ctx, Command{Verb: "LIST", Args: "FILES LIKE *.txt"}); err != nil {
		t.Fatalf("LIST FILES LIKE: %v", err)
	}

	out := ctx.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(out, "NOTE.TXT") {
		t.Fatalf("expected NOTE.TXT, got %q", out)
	}
	if strings.Contains(out, "PEOPLE.DBF") {
		t.Fatalf("expected only matching files, got %q", out)
	}
	if strings.Contains(out, "DATABASE FILES") {
		t.Fatalf("expected plain listing for LIKE pattern, got %q", out)
	}
}

func TestDispatchDirAlias(t *testing.T) {
	ctx := dirTestCtx(t)

	if err := commandMux.Dispatch(ctx, Command{Verb: "DIR", Args: "*.txt"}); err != nil {
		t.Fatalf("DIR: %v", err)
	}

	out := ctx.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(out, "NOTE.TXT") {
		t.Fatalf("expected NOTE.TXT, got %q", out)
	}
}

func TestDispatchDirDefaultListsDatabases(t *testing.T) {
	ctx := dirTestCtx(t)

	if err := commandMux.Dispatch(ctx, Command{Verb: "DIR"}); err != nil {
		t.Fatalf("DIR: %v", err)
	}

	out := ctx.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(out, "DATABASE FILES") || !strings.Contains(out, "PEOPLE.DBF") {
		t.Fatalf("expected database listing, got %q", out)
	}
}

func TestDispatchDisplayFilesLikeRequiresPattern(t *testing.T) {
	ctx := dirTestCtx(t)

	err := commandMux.Dispatch(ctx, Command{Verb: "DISPLAY", Args: "FILES LIKE"})
	if err == nil || !strings.Contains(err.Error(), "LIKE requires a file pattern") {
		t.Fatalf("expected pattern error, got %v", err)
	}
}

func TestParseFilesPattern(t *testing.T) {
	tests := []struct {
		args     string
		pattern  string
		explicit bool
		bad      bool
	}{
		{"", "*.dbf", false, false},
		{"LIKE *.prg", "*.prg", true, false},
		{"like *.txt", "*.txt", true, false},
		{"*.ndx", "*.ndx", true, false},
		{"LIKE", "", false, true},
	}
	for _, tt := range tests {
		pattern, explicit, err := parseFilesPattern(tt.args)
		if tt.bad {
			if err == nil {
				t.Errorf("parseFilesPattern(%q): expected error", tt.args)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseFilesPattern(%q): %v", tt.args, err)
			continue
		}
		if pattern != tt.pattern || explicit != tt.explicit {
			t.Errorf("parseFilesPattern(%q) = (%q, %v), want (%q, %v)",
				tt.args, pattern, explicit, tt.pattern, tt.explicit)
		}
	}
}
