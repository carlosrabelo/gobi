package repl

import (
	"bytes"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func TestPCOLPROWAdvanceOnPrint(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	ctx.Stdout = &bytes.Buffer{}
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("SET DEVICE TO PRINT"))
	if err := mux.Dispatch(ctx, command.Split(`@ 0, 0 SAY "Hi"`)); err != nil {
		t.Fatal(err)
	}
	if ctx.PrintCol != 2 {
		t.Fatalf("PCOL after Hi: %d", ctx.PrintCol)
	}
	if err := mux.Dispatch(ctx, command.Split("EJECT")); err != nil {
		t.Fatal(err)
	}
	if ctx.PrintRow != 0 || ctx.PrintCol != 0 {
		t.Fatalf("after EJECT row=%d col=%d", ctx.PrintRow, ctx.PrintCol)
	}
	_ = mux.Dispatch(ctx, command.Split("SET DEVICE TO SCREEN"))
	_ = mux.Dispatch(ctx, command.Split("SET PRINT ON"))
	_ = mux.Dispatch(ctx, command.Split("? \"AB\""))
	if ctx.PrintCol != 0 || ctx.PrintRow != 1 {
		// "? AB\n" → two chars then newline → row 1 col 0
		t.Fatalf("after ?: row=%d col=%d", ctx.PrintRow, ctx.PrintCol)
	}
}

func TestSessionProgramFns(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	ctx.Stdout = &bytes.Buffer{}
	mux := NewMux()
	env := newEnv(ctx)
	got, err := expr.Eval(mustParse(t, "SELECT()"), env)
	if err != nil || got.Num != 1 {
		t.Fatalf("SELECT %v %v", got, err)
	}
	got, err = expr.Eval(mustParse(t, "UPDATED()"), env)
	if err != nil || got.Log {
		t.Fatalf("UPDATED %v", got)
	}
	if err := mux.Dispatch(ctx, command.Split(`SET FUNCTION 2 TO "DIR;"`)); err != nil {
		t.Fatal(err)
	}
	if ctx.FuncKeys[2] != "DIR;" {
		t.Fatalf("F2 %q", ctx.FuncKeys[2])
	}
}
