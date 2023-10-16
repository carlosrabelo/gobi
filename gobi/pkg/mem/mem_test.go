package mem

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

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

func TestRejectDateLeavesCaller(t *testing.T) {
	var hdr [32]byte
	copy(hdr[0:], "WHEN")
	hdr[11] = 'D'
	copy(hdr[18:], "20200101")
	_, err := ReadAll(bytes.NewReader(hdr[:]))
	if err == nil {
		t.Fatal("expected D rejection")
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
