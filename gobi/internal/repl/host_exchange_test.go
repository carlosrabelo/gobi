package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/hostmod"
)

func TestExportImportRoundTrip(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "ANN "}}},
		{Values: []dbf.Value{{Str: "BOB "}}},
	})
	ctx := testCtx(t, dir)
	ctx.Config.Safety = false
	ctx.Stdout = &bytes.Buffer{}
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE t")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("EXPORT TO out.txt DELIMITED")); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "out.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "ANN") {
		t.Fatalf("export: %q", raw)
	}
	if err := mux.Dispatch(ctx, command.Split("IMPORT FROM out.txt DELIMITED")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "out.dbf")); err != nil {
		t.Fatal(err)
	}
}

func TestLoadCallReleaseModule(t *testing.T) {
	hostmod.ClearBuiltins()
	defer hostmod.ClearBuiltins()
	var calls atomic.Int32
	hostmod.RegisterBuiltin("TESTMOD", func(args []string) error {
		calls.Add(1)
		return nil
	})
	ctx := testCtx(t, t.TempDir())
	ctx.Stdout = &bytes.Buffer{}
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("LOAD TESTMOD")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("CALL TESTMOD")); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls %d", calls.Load())
	}
	if err := mux.Dispatch(ctx, command.Split("RELEASE MODULE TESTMOD")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("CALL TESTMOD")); err == nil {
		t.Fatal("expected not loaded")
	}
}

func TestFlockUnlockAndLocalNoops(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "ANN "}}},
	})
	ctx := testCtx(t, dir)
	out := &bytes.Buffer{}
	ctx.Stdout = out
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE t")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("? FLOCK()")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), ".T.") {
		t.Fatalf("FLOCK: %q", out.String())
	}
	if err := mux.Dispatch(ctx, command.Split("UNLOCK")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("SET EXCLUSIVE ON")); err != nil {
		t.Fatal(err)
	}
	if !ctx.Config.Exclusive {
		t.Fatal("Exclusive")
	}
	if err := mux.Dispatch(ctx, command.Split("SET ENCRYPTION ON")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("DISPLAY USERS")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("LOGOUT")); err != nil {
		t.Fatal(err)
	}
	if !ctx.Active().HasTable() {
		t.Fatal("LOGOUT should leave table open")
	}
}
