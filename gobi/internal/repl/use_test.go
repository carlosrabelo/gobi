package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestUSEEmptyAndClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.dbf")
	mustCreateDBF(t, path, []dbf.Field{{Name: "NAME", Type: 'C', Length: 4}}, nil)

	ctx := testCtx(t, dir)
	mux := NewMux()

	if err := mux.Dispatch(ctx, command.Split("USE empty")); err != nil {
		t.Fatal(err)
	}
	a := ctx.Active()
	if !a.HasTable() {
		t.Fatal("expected open table")
	}
	if a.Alias != "EMPTY" {
		t.Fatalf("alias %q", a.Alias)
	}
	if a.Cursor != 1 || a.Landed {
		t.Fatalf("empty: cursor=%d landed=%v", a.Cursor, a.Landed)
	}
	if a.RecCount() != 0 {
		t.Fatalf("count %d", a.RecCount())
	}
	if a.BOF {
		t.Fatal("BOF should be false after USE")
	}

	if err := mux.Dispatch(ctx, command.Split("USE")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().HasTable() {
		t.Fatal("USE with no name should close")
	}
	if ctx.OpenFileCount() != 0 {
		t.Fatalf("open files %d", ctx.OpenFileCount())
	}
}

func TestUSEMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keep.dbf")
	mustCreateDBF(t, path, []dbf.Field{{Name: "A", Type: 'C', Length: 1}}, []dbf.Record{
		{Values: []dbf.Value{{Str: "X"}}},
	})

	ctx := testCtx(t, dir)
	var errBuf bytes.Buffer
	ctx.Stderr = &errBuf
	mux := NewMux()

	if err := mux.Dispatch(ctx, command.Split("USE keep")); err != nil {
		t.Fatal(err)
	}
	err := mux.Dispatch(ctx, command.Split("USE nosuch"))
	if err == nil {
		t.Fatal("expected missing-file error")
	}
	if !strings.Contains(errBuf.String(), "*** ") {
		t.Fatalf("stderr %q", errBuf.String())
	}
	if ctx.Active().HasTable() {
		t.Fatal("missing USE must leave area empty after closing previous")
	}
}

func TestCLOSEDatabases(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "a.dbf"), []dbf.Field{{Name: "A", Type: 'C', Length: 1}}, nil)

	ctx := testCtx(t, dir)
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE a")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("CLOSE DATABASES")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().HasTable() || ctx.ActiveArea != 1 {
		t.Fatalf("area=%d table=%v", ctx.ActiveArea, ctx.Active().HasTable())
	}
}

func testCtx(t *testing.T, defaultDir string) *context.Context {
	t.Helper()
	ctx := context.New()
	ctx.Config.Default = defaultDir
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stderr = &bytes.Buffer{}
	return ctx
}

func mustCreateDBF(t *testing.T, path string, fields []dbf.Field, recs []dbf.Record) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	tab, err := dbf.Create(f, fields, 86, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range recs {
		if err := tab.Append(r); err != nil {
			t.Fatal(err)
		}
	}
	if err := tab.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}
