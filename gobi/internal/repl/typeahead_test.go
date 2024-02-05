package repl

import (
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

func TestCollectAndDrainTypeahead(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	ctx.Config.Typeahead = 20
	ctx.Stdin = strings.NewReader("AB")
	ctx.Keys = term.NewReader(ctx.Stdin)
	collectTypeahead(ctx)
	if len(ctx.TypeaheadBuf) != 2 || ctx.TypeaheadBuf[0] != 'A' || ctx.TypeaheadBuf[1] != 'B' {
		t.Fatalf("collect: %v", ctx.TypeaheadBuf)
	}
	drainTypeaheadIntoKeys(ctx)
	if len(ctx.TypeaheadBuf) != 0 {
		t.Fatal("drain left typeahead")
	}
	k, err := ctx.Keys.ReadKey()
	if err != nil || k != int('A') {
		t.Fatalf("first key: %d %v", k, err)
	}
	k, err = ctx.Keys.ReadKey()
	if err != nil || k != int('B') {
		t.Fatalf("second key: %d %v", k, err)
	}
}

func TestCollectTypeaheadPausedWhenKeyRead(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	ctx.Config.Typeahead = 20
	ctx.KeyRead = true
	ctx.Stdin = strings.NewReader("Z")
	ctx.Keys = term.NewReader(ctx.Stdin)
	collectTypeahead(ctx)
	if len(ctx.TypeaheadBuf) != 0 {
		t.Fatalf("pump must pause while KeyRead: %v", ctx.TypeaheadBuf)
	}
}

func TestTypeaheadZeroDisablesCollect(t *testing.T) {
	ctx := testCtx(t, t.TempDir())
	ctx.Config.Typeahead = 0
	ctx.Stdin = strings.NewReader("Z")
	ctx.Keys = term.NewReader(ctx.Stdin)
	collectTypeahead(ctx)
	if len(ctx.TypeaheadBuf) != 0 {
		t.Fatal("TYPEAHEAD 0 must discard")
	}
}
