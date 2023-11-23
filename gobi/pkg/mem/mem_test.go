package mem

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func TestRoundTripCNL(t *testing.T) {
	entries := []Entry{
		{Name: "MSG", Value: expr.StringVal("hello")},
		{Name: "N", Value: expr.NumberVal(3.5)},
		{Name: "FLAG", Value: expr.LogicalVal(true)},
	}
	var buf bytes.Buffer
	if err := WriteAll(&buf, entries); err != nil {
		t.Fatal(err)
	}
	got, err := ReadAll(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len %d", len(got))
	}
	if got[0].Name != "MSG" || got[0].Value.Str != "hello" {
		t.Fatalf("C: %+v", got[0])
	}
	if got[1].Name != "N" || got[1].Value.Num != 3.5 {
		t.Fatalf("N: %+v", got[1])
	}
	if got[2].Name != "FLAG" || !got[2].Value.Log {
		t.Fatalf("L: %+v", got[2])
	}
}

func TestRoundTripDate(t *testing.T) {
	d := expr.DateVal(time.Date(1986, 7, 4, 0, 0, 0, 0, time.Local))
	var buf bytes.Buffer
	if err := WriteAll(&buf, []Entry{{Name: "WHEN", Value: d}}); err != nil {
		t.Fatal(err)
	}
	raw := buf.Bytes()
	if raw[11] != 'D' {
		t.Fatalf("type %c", raw[11])
	}
	if string(raw[18:26]) != "19860704" {
		t.Fatalf("digits %q", raw[18:26])
	}
	got, err := ReadAll(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Value.Kind != expr.KindDate || got[0].Value.YYYYMMDD() != "19860704" {
		t.Fatalf("%+v", got)
	}
}

func TestFileEOF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t.mem")
	if err := WriteFile(path, []Entry{{Name: "A", Value: expr.NumberVal(1)}}); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Value.Num != 1 {
		t.Fatalf("%+v", got)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Size() != 32 {
		t.Fatalf("size %d", st.Size())
	}
}
