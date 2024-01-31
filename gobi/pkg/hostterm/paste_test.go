package hostterm

import (
	"bytes"
	"strings"
	"testing"
)

func TestBracketedPasteEnableDisable(t *testing.T) {
	var buf bytes.Buffer
	if err := EnableBracketedPaste(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "\x1b[?2004h") {
		t.Fatalf("enable: %q", buf.String())
	}
	buf.Reset()
	if err := DisableBracketedPaste(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "\x1b[?2004l") {
		t.Fatalf("disable: %q", buf.String())
	}
}
