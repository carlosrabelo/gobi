package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

func TestHistoryTypeSuspendTYPE(t *testing.T) {
	dir := t.TempDir()
	ctx := testCtx(t, dir)
	ctx.Stdout = &bytes.Buffer{}
	mux := NewMux()
	hist := NewHistory(dir)
	hist.Cap = 20
	_ = mux.Dispatch(ctx, command.Split("SET HISTORY TO 2"))
	hist.SetCap(ctx.Config.HistoryTo)
	hist.Add("A")
	hist.Add("B")
	hist.Add("C")
	if hist.Len() != 2 {
		t.Fatalf("len %d", hist.Len())
	}

	_ = mux.Dispatch(ctx, command.Split("CLEAR TYPEAHEAD"))
	ctx.TypeaheadBuf = []byte{65}
	_ = mux.Dispatch(ctx, command.Split("CLEAR TYPEAHEAD"))
	if len(ctx.TypeaheadBuf) != 0 {
		t.Fatal("typeahead not cleared")
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
