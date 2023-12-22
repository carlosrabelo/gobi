package repl

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/help"
)

func TestRefusePrinterColorAndHelp(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	var errBuf, out bytes.Buffer
	ctx.Stderr = &errBuf
	ctx.Stdout = &out
	mux := NewMux()
	for _, line := range []string{`SET PRINTER TO LPT1`, `SET COLOR ON`, `SET COLOR OFF`} {
		errBuf.Reset()
		err := mux.Dispatch(ctx, command.Split(line))
		if err == nil || !strings.Contains(err.Error(), "not available") {
			t.Fatalf("%s: %v", line, err)
		}
		if ctx.LastError != 36 {
			t.Fatalf("%s code %d", line, ctx.LastError)
		}
	}
	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("HELP ASSIST")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "not available") {
		t.Fatalf("HELP ASSIST: %q", out.String())
	}
	for _, topic := range refusedHelpTopics() {
		text, ok := help.Text(topic)
		if !ok || !strings.Contains(text, "not available") {
			t.Fatalf("help %s: %q ok=%v", topic, text, ok)
		}
	}
	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("HELP CREATE LABEL")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "CREATE LABEL") || !strings.Contains(out.String(), "not available") {
		t.Fatalf("HELP CREATE LABEL: %q", out.String())
	}
}

func TestRefuseNetworkAndLocks(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "ANN "}}},
	})
	ctx := testCtx(t, dir)
	var errBuf bytes.Buffer
	ctx.Stderr = &errBuf
	ctx.Stdout = &bytes.Buffer{}
	mux := NewMux()
	_ = mux.Dispatch(ctx, command.Split("USE t"))
	for _, line := range []string{
		"DISPLAY USERS", "LIST USERS", "LOGOUT", "UNLOCK",
		"SET EXCLUSIVE ON", "SET ENCRYPTION ON", "? FLOCK()",
	} {
		errBuf.Reset()
		err := mux.Dispatch(ctx, command.Split(line))
		if err == nil || !strings.Contains(err.Error(), "not available") {
			t.Fatalf("%s: %v", line, err)
		}
		if ctx.LastError != 36 {
			t.Fatalf("%s code %d", line, ctx.LastError)
		}
		if !strings.Contains(errBuf.String(), "*** ") {
			t.Fatalf("%s stderr %q", line, errBuf.String())
		}
	}
	a := ctx.Active()
	if !a.HasTable() || a.RecCount() != 1 {
		t.Fatal("open table was disturbed")
	}
}

func TestRefuseHostBinaries(t *testing.T) {
	dir := t.TempDir()
	mustCreateDBF(t, filepath.Join(dir, "t.dbf"), []dbf.Field{
		{Name: "NAME", Type: 'C', Length: 4},
	}, []dbf.Record{
		{Values: []dbf.Value{{Str: "ANN "}}},
	})
	ctx := testCtx(t, dir)
	var errBuf bytes.Buffer
	ctx.Stderr = &errBuf
	ctx.Stdout = &bytes.Buffer{}
	mux := NewMux()
	if err := mux.Dispatch(ctx, command.Split("USE t")); err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"LOAD x", "CALL x", "EXPORT", "IMPORT", "RELEASE MODULE x"} {
		errBuf.Reset()
		err := mux.Dispatch(ctx, command.Split(line))
		if err == nil || !strings.Contains(err.Error(), "not available") {
			t.Fatalf("%s: %v", line, err)
		}
		if ctx.LastError != 36 {
			t.Fatalf("%s code %d", line, ctx.LastError)
		}
		if !strings.Contains(errBuf.String(), "*** ") {
			t.Fatalf("%s stderr %q", line, errBuf.String())
		}
	}
	a := ctx.Active()
	if !a.HasTable() || a.RecCount() != 1 {
		t.Fatal("open table was disturbed")
	}
	_ = a.LoadCurrent()
	if strings.TrimSpace(a.Record.Values[0].Str) != "ANN" {
		t.Fatalf("record changed: %q", a.Record.Values[0].Str)
	}
	for _, topic := range []string{"LOAD", "CALL", "EXPORT", "IMPORT", "RELEASE MODULE"} {
		text, ok := help.Text(topic)
		if !ok || !strings.Contains(text, "not available") {
			t.Fatalf("HELP %s: %q ok=%v", topic, text, ok)
		}
	}
}
