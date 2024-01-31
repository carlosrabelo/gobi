package hostterm

import (
	"bytes"
	"strings"
	"testing"
)

func TestEnterLeaveAltScreen(t *testing.T) {
	caps := Caps{Clear: true}
	var buf bytes.Buffer
	if err := EnterAltScreen(&buf, caps); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "\x1b[?1049h") {
		t.Fatalf("enter: %q", buf.String())
	}
	buf.Reset()
	if err := LeaveAltScreen(&buf, caps); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "\x1b[?1049l") {
		t.Fatalf("leave: %q", buf.String())
	}
	buf.Reset()
	if err := EnterAltScreen(&buf, Caps{}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Fatalf("no-op without caps: %q", buf.String())
	}
}
