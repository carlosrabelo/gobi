package ndx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSampleNDXFixture(t *testing.T) {
	path := filepath.Join("testdata", "sample.ndx")
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	idx, err := Open(f)
	if err != nil {
		t.Fatal(err)
	}
	h := idx.Header()
	if h.Type != KeyChar || h.KeyLen != 10 {
		t.Fatalf("hdr %+v", h)
	}
	rec, ok, err := idx.Search(EncodeKeyChar("ALPHA", 10))
	if err != nil || !ok || rec != 1 {
		t.Fatalf("seek ALPHA: rec=%d ok=%v err=%v", rec, ok, err)
	}
}
