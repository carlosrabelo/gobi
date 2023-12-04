package repl

import (
	"bytes"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

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
