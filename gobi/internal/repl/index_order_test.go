package repl

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestIndexOrderNavigation(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "peeps.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "CYN"}}},
		{Values: []dbf.Value{{Str: "ANN"}}},
		{Values: []dbf.Value{{Str: "BOB"}}},
	})
	ctx := testCtx(t, dir)
	ctx.Config.Safety = false
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("USE peeps"))
	_ = mux.Dispatch(ctx, command.Split("INDEX ON NAME TO byname"))
	_ = mux.Dispatch(ctx, command.Split("GO TOP"))
	if strings.TrimSpace(ctx.Active().Record.Values[0].Str) != "ANN" {
		t.Fatalf("GO TOP: %q", ctx.Active().Record.Values[0].Str)
	}
	_ = mux.Dispatch(ctx, command.Split("SKIP"))
	if strings.TrimSpace(ctx.Active().Record.Values[0].Str) != "BOB" {
		t.Fatalf("SKIP: %q", ctx.Active().Record.Values[0].Str)
	}
	out.Reset()
	_ = mux.Dispatch(ctx, command.Split("LIST NAME"))
	got := out.String()
	ai := strings.Index(got, "ANN")
	bi := strings.Index(got, "BOB")
	ci := strings.Index(got, "CYN")
	if !(ai >= 0 && bi > ai && ci > bi) {
		t.Fatalf("LIST order: %q", got)
	}
	_ = mux.Dispatch(ctx, command.Split("SET INDEX TO"))
	_ = mux.Dispatch(ctx, command.Split("GO TOP"))
	if strings.TrimSpace(ctx.Active().Record.Values[0].Str) != "CYN" {
		t.Fatalf("physical TOP after clear: %q", ctx.Active().Record.Values[0].Str)
	}
}
