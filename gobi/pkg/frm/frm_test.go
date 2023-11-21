package frm

import (
	"strings"
	"testing"
)

func TestParseTitleColumnsGroup(t *testing.T) {
	src := "Sales\nGROUP DEPT\nNAME 10\nAGE 3\n\n"
	f, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if f.Title != "Sales" || f.Group != "DEPT" || len(f.Columns) != 2 {
		t.Fatalf("%+v", f)
	}
	if f.Columns[0].Field != "NAME" || f.Columns[0].Width != 10 {
		t.Fatalf("col0 %+v", f.Columns[0])
	}
}
