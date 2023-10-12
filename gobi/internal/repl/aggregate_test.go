package repl

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestCOUNTForTalkAndCursor(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "peeps.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
		{Name: "AGE", Type: 'N', Length: 3, Decimals: 0},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "ANN"}, {Num: 10}}},
		{Values: []dbf.Value{{Str: "BOB"}, {Num: 20}}},
		{Values: []dbf.Value{{Str: "CYN"}, {Num: 20}}},
		{Values: []dbf.Value{{Str: "DAN"}, {Num: 40}}},
	})

	ctx := testCtx(t, dir)
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE peeps")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("GOTO 2")); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("COUNT FOR AGE = 20")); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "2" {
		t.Fatalf("COUNT print: %q", out.String())
	}
	if ctx.Active().Cursor != 2 {
		t.Fatalf("cursor restored: %d", ctx.Active().Cursor)
	}
	if len(ctx.PendingTO) != 0 || len(ctx.PendingVals) != 1 || ctx.PendingVals[0] != 2 {
		t.Fatalf("pending: TO=%v vals=%v", ctx.PendingTO, ctx.PendingVals)
	}

	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("COUNT FOR AGE = 20 TO N")); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "2" {
		t.Fatalf("COUNT TO print: %q", out.String())
	}
	if len(ctx.PendingTO) != 1 || ctx.PendingTO[0] != "N" || ctx.PendingVals[0] != 2 {
		t.Fatalf("pending TO: %v %v", ctx.PendingTO, ctx.PendingVals)
	}

	ctx.Config.Talk = false
	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("COUNT")); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("TALK off should print nothing: %q", out.String())
	}
	if len(ctx.PendingVals) != 1 || ctx.PendingVals[0] != 4 {
		t.Fatalf("count all vals: %v", ctx.PendingVals)
	}
}

func TestSUMAndAVERAGE(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "peeps.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
		{Name: "AGE", Type: 'N', Length: 3, Decimals: 0},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "ANN"}, {Num: 10}}},
		{Values: []dbf.Value{{Str: "BOB"}, {Num: 20}}},
		{Values: []dbf.Value{{Str: "CYN"}, {Num: 20}}},
		{Values: []dbf.Value{{Str: "DAN"}, {Num: 40}}},
	})

	ctx := testCtx(t, dir)
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE peeps")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("GOTO 3")); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("SUM AGE")); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "90" {
		t.Fatalf("SUM: %q", out.String())
	}
	if ctx.Active().Cursor != 3 {
		t.Fatalf("cursor after SUM: %d", ctx.Active().Cursor)
	}

	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("SUM AGE TO S")); err != nil {
		t.Fatal(err)
	}
	if len(ctx.PendingTO) != 1 || ctx.PendingTO[0] != "S" || ctx.PendingVals[0] != 90 {
		t.Fatalf("SUM TO: %v %v", ctx.PendingTO, ctx.PendingVals)
	}

	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("AVERAGE AGE")); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "22.5" {
		t.Fatalf("AVERAGE: %q", out.String())
	}

	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("SUM AGE FOR AGE > 100")); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "0" {
		t.Fatalf("SUM zero matches: %q", out.String())
	}
	if ctx.PendingVals[0] != 0 {
		t.Fatalf("zero match val: %v", ctx.PendingVals)
	}

	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("AVERAGE AGE FOR AGE > 100")); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "0" {
		t.Fatalf("AVERAGE zero: %q", out.String())
	}
}
