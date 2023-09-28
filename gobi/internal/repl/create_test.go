package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestCREATEOneField(t *testing.T) {
	dir := t.TempDir()
	ctx := testCtx(t, dir)
	var out bytes.Buffer
	ctx.Stdout = &out
	ctx.Stdin = strings.NewReader("NAME\nC\n10\n0\n\n")
	mux := NewMux()

	if err := mux.Dispatch(ctx, command.Split("CREATE cust")); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "cust.dbf")
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	tab, err := dbf.Open(f)
	if err != nil {
		t.Fatal(err)
	}
	if tab.Header.RecordCount != 0 {
		t.Fatalf("count %d", tab.Header.RecordCount)
	}
	if len(tab.Header.Fields) != 1 || tab.Header.Fields[0].Name != "NAME" || tab.Header.Fields[0].Type != 'C' {
		t.Fatalf("fields %+v", tab.Header.Fields)
	}
}
