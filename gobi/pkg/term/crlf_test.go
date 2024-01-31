package term

import (
	"bytes"
	"testing"
)

func TestCRLFWriterMapsLF(t *testing.T) {
	var buf bytes.Buffer
	w := &CRLFWriter{W: &buf}
	if _, err := w.Write([]byte("a\nb\r\nc\n")); err != nil {
		t.Fatal(err)
	}
	want := "a\r\nb\r\nc\r\n"
	if got := buf.String(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if w.Unwrap() != &buf {
		t.Fatal("Unwrap")
	}
}
