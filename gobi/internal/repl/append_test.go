package repl

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestAPPENDLineMode(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
		{Name: "AGE", Type: 'N', Length: 3, Decimals: 0},
	}, nil)

	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stdin = strings.NewReader("ANN\n10\nBOB\n20\n\n")
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE t")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("APPEND")); err != nil {
		t.Fatal(err)
	}
	a := ctx.Active()
	if a.RecCount() != 2 {
		t.Fatalf("count %d", a.RecCount())
	}
	if a.Cursor != 2 || a.Record.Values[0].Str != "BOB" {
		t.Fatalf("cursor on last: %d %+v", a.Cursor, a.Record)
	}
}
