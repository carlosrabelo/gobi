package repl

import (
	"strings"
	"testing"
)

func TestDispatchCountRejectsUnknownScope(t *testing.T) {
	ctx := testCtx()
	err := commandMux.Dispatch(ctx, Command{Verb: "COUNT", Args: "REST"})
	if err == nil || !strings.Contains(err.Error(), "COUNT scope") {
		t.Fatalf("expected COUNT scope error, got %v", err)
	}
}
