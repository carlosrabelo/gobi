package mem

import (
	"path/filepath"
	"testing"
)

func TestSampleMEMFixture(t *testing.T) {
	path := filepath.Join("testdata", "sample.mem")
	got, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len %d", len(got))
	}
	if got[0].Name != "MSG" || got[0].Value.Str != "hello" {
		t.Fatalf("C %+v", got[0])
	}
	if got[1].Name != "N" || got[1].Value.Num != 42 {
		t.Fatalf("N %+v", got[1])
	}
}
