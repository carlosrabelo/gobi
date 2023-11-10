package repl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestScopeNEXTAndAppendBlankInsertClearErase(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "N", Type: 'N', Length: 3, Decimals: 0},
	}, []dbf.Record{
		{Values: []dbf.Value{{Num: 1}}},
		{Values: []dbf.Value{{Num: 2}}},
		{Values: []dbf.Value{{Num: 3}}},
	})
	ctx := testCtx(t, dir)
	ctx.Config.Safety = false
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("USE t"))
	_ = mux.Dispatch(ctx, command.Split("GOTO 2"))
	if err := mux.Dispatch(ctx, command.Split("DELETE NEXT 1")); err != nil {
		t.Fatal(err)
	}
	r2, _ := ctx.Active().Table.Read(2)
	r3, _ := ctx.Active().Table.Read(3)
	if !r2.Deleted || r3.Deleted {
		t.Fatalf("NEXT 1: r2=%v r3=%v", r2.Deleted, r3.Deleted)
	}

	if err := mux.Dispatch(ctx, command.Split("APPEND BLANK")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().RecCount() != 4 || !ctx.Active().Landed {
		t.Fatalf("APPEND BLANK count=%d landed=%v", ctx.Active().RecCount(), ctx.Active().Landed)
	}

	_ = mux.Dispatch(ctx, command.Split("GOTO 2"))
	if err := mux.Dispatch(ctx, command.Split("INSERT BEFORE BLANK")); err != nil {
		t.Fatal(err)
	}
	if ctx.Active().RecCount() != 5 {
		t.Fatalf("after INSERT count %d", ctx.Active().RecCount())
	}

	_ = mux.Dispatch(ctx, command.Split(`STORE 1 TO V`))
	if err := mux.Dispatch(ctx, command.Split("CLEAR ALL")); err != nil {
		t.Fatal(err)
	}
	if ctx.Symbols.Len() != 0 || ctx.Active().HasTable() {
		t.Fatal("CLEAR ALL")
	}

	path := filepath.Join(dir, "gone.txt")
	_ = os.WriteFile(path, []byte("x"), 0o644)
	ctx.Config.Default = dir
	if err := mux.Dispatch(ctx, command.Split("ERASE gone.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("ERASE")
	}
	_ = strings.TrimSpace
}
