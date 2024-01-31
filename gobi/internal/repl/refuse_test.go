package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

func TestPrinterAndColorHostExchange(t *testing.T) {
	dir := t.TempDir()
	ctx := testCtx(t, dir)
	var errBuf, out bytes.Buffer
	ctx.Stderr = &errBuf
	ctx.Stdout = &out
	mux := NewMux()

	prn := filepath.Join(dir, "out.prn")
	if err := mux.Dispatch(ctx, command.Split("SET PRINTER TO "+prn)); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("EJECT")); err != nil {
		t.Fatal(err)
	}
	if ctx.PrinterFile != nil {
		_ = ctx.PrinterFile.Close()
		ctx.PrinterFile = nil
	}
	data, err := os.ReadFile(prn)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte{0x0c}) {
		t.Fatalf("printer file missing form feed: %q", data)
	}
	if err := mux.Dispatch(ctx, command.Split("SET PRINTER TO")); err != nil {
		t.Fatal(err)
	}
	if err := mux.Dispatch(ctx, command.Split("SET COLOR ON")); err != nil {
		t.Fatal(err)
	}
	if !ctx.Config.ColorOn {
		t.Fatal("ColorOn")
	}
	if err := mux.Dispatch(ctx, command.Split("SET COLOR OFF")); err != nil {
		t.Fatal(err)
	}

	out.Reset()
	if err := mux.Dispatch(ctx, command.Split("HELP ASSIST")); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "not available") || !strings.Contains(out.String(), "ASSIST") {
		t.Fatalf("HELP ASSIST should describe the menu: %q", out.String())
	}
}
