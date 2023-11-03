package repl

import (
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

func TestROWCOLINKEY(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split(`@ 2, 5 SAY "X"`))
	env := newEnv(ctx)
	row, err := expr.Eval(mustParse(t, "ROW()"), env)
	if err != nil {
		t.Fatal(err)
	}
	col, err := expr.Eval(mustParse(t, "COL()"), env)
	if err != nil {
		t.Fatal(err)
	}
	if row.Num != 2 {
		t.Fatalf("ROW=%v", row)
	}
	if col.Num != 6 { // after "X" at col 5
		t.Fatalf("COL=%v", col)
	}
	ctx.Keys = term.NewReader(strings.NewReader("A"))
	ink, err := expr.Eval(mustParse(t, "INKEY()"), env)
	if err != nil {
		t.Fatal(err)
	}
	if ink.Num != float64('A') {
		t.Fatalf("INKEY=%v", ink)
	}
	ink, err = expr.Eval(mustParse(t, "INKEY()"), env)
	if err != nil {
		t.Fatal(err)
	}
	if ink.Num != 0 {
		t.Fatalf("empty INKEY=%v", ink)
	}
}

func mustParse(t *testing.T, s string) expr.Expression {
	t.Helper()
	tree, errs := expr.Parse(s)
	if len(errs) > 0 || tree == nil {
		t.Fatalf("parse %s: %v", s, errs)
	}
	return tree
}
