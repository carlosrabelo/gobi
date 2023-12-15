package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func TestHistoryTypeSuspendTYPE(t *testing.T) {
	dir := t.TempDir()
	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	mux := NewMux()
	hist := NewHistory(dir)
	hist.Cap = 20
	mux.hist = hist
	_ = mux.Dispatch(ctx, command.Split("SET HISTORY TO 2"))
	if hist.Cap != 2 {
		t.Fatalf("cap %d", hist.Cap)
	}
	hist.Add("A")
	hist.Add("B")
	hist.Add("C")
	if hist.Len() != 2 {
		t.Fatalf("len %d", hist.Len())
	}
	_ = mux.Dispatch(ctx, command.Split("SET DOHISTORY ON"))
	_ = os.WriteFile(filepath.Join(dir, "h.prg"), []byte("STORE 1 TO Z\n"), 0o644)
	if err := mux.Dispatch(ctx, command.Split("DO h")); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, l := range hist.Lines() {
		if l == "STORE 1 TO Z" {
			found = true
		}
	}
	if !found {
		t.Fatalf("DOHISTORY missing line: %v", hist.Lines())
	}
	var hout bytes.Buffer
	ctx.Stdout = &hout
	if err := mux.Dispatch(ctx, command.Split("DISPLAY HISTORY")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(hout.String(), "STORE 1 TO Z") {
		t.Fatalf("DISPLAY HISTORY: %q", hout.String())
	}

	_ = mux.Dispatch(ctx, command.Split("SET TYPEAHEAD TO 2"))
	ctx.PushTypeahead('A', 'B', 'C')
	if len(ctx.TypeaheadBuf) != 2 || ctx.TypeaheadBuf[0] != 'B' {
		t.Fatalf("typeahead cap: %v", ctx.TypeaheadBuf)
	}
	_ = mux.Dispatch(ctx, command.Split("CLEAR TYPEAHEAD"))
	if len(ctx.TypeaheadBuf) != 0 {
		t.Fatal("typeahead not cleared")
	}
	ctx.PushTypeahead('Z')
	ink, err := expr.Eval(mustParse(t, "INKEY()"), newEnv(ctx))
	if err != nil || ink.Num != float64('Z') {
		t.Fatalf("INKEY from typeahead: %v %v", ink, err)
	}
	if len(ctx.TypeaheadBuf) != 0 {
		t.Fatal("typeahead not drained")
	}

	prg := "SUSPEND\nX = 1\nRETURN\n"
	if err := os.WriteFile(filepath.Join(dir, "s.prg"), []byte(prg), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("DO s")); err != nil {
		t.Fatal(err)
	}
	if !ctx.Suspended {
		t.Fatal("expected suspended")
	}
	if err := mux.Dispatch(ctx, command.Split("RESUME")); err != nil {
		t.Fatal(err)
	}
	v, ok := ctx.Symbols.Get("X")
	if !ok || v.Num != 1 {
		t.Fatalf("X after RESUME %v", v)
	}

	if err := os.WriteFile(filepath.Join(dir, "t.txt"), []byte("L1\nL2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	ctx.Stdout = out
	if err := mux.Dispatch(ctx, command.Split("TYPE t.txt")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "L1") || !strings.Contains(out.String(), "L2") {
		t.Fatalf("%q", out.String())
	}

	_ = mux.Dispatch(ctx, command.Split("SET MEMOWIDTH TO 10"))
	if ctx.Config.MemoWidth != 10 {
		t.Fatalf("memowidth %d", ctx.Config.MemoWidth)
	}
}
