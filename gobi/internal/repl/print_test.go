package repl

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestSETFixedDecimalsOnQuestionAndSTR(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("SET DECIMALS TO 2"))
	_ = mux.Dispatch(ctx, command.Split("SET FIXED ON"))
	if !ctx.Config.Fixed {
		t.Fatal("FIXED still off")
	}
	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("? 1.5")); err != nil {
		t.Fatal(err)
	}
	if out.String() != "1.50\n" {
		t.Fatalf("?: %q", out.String())
	}
	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("? STR(1.5)")); err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(out.String())
	if !strings.Contains(got, "1.50") {
		t.Fatalf("STR: %q", got)
	}
}

func TestSETPrintAppliesMargin(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split(`STORE 1 TO A`))
	_ = mux.Dispatch(ctx, command.Split("SET MARGIN TO 2"))
	_ = mux.Dispatch(ctx, command.Split("SET PRINT ON"))
	if !ctx.Config.Print {
		t.Fatal("PRINT still off")
	}
	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("? A")); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "  1") {
		t.Fatalf("margin missing: %q", out.String())
	}
}

func TestQuestionAndDouble(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()

	if err := mux.Dispatch(ctx, command.Split("? .T.")); err != nil {
		t.Fatal(err)
	}
	if out.String() != ".T.\n" {
		t.Fatalf("?: %q", out.String())
	}

	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("?? 1+1")); err != nil {
		t.Fatal(err)
	}
	if out.String() != "2" {
		t.Fatalf("??: %q", out.String())
	}

	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("? 1, 2")); err != nil {
		t.Fatal(err)
	}
	if out.String() != "1 2\n" {
		t.Fatalf("? commas: %q", out.String())
	}
}

func TestQuestionUsesFieldDecimals(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "c.dbf"), []dbf.Field{
		{Name: "BALANCE", Type: 'N', Length: 10, Decimals: 2},
	}, []dbf.Record{{Values: []dbf.Value{{Num: 120.5}}}})
	ctx := testCtx(t, dir)
	var out bytes.Buffer
	ctx.Stdout = &out
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("USE c"))
	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("? BALANCE")); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "120.50" {
		t.Fatalf("want 120.50 got %q", out.String())
	}
}
