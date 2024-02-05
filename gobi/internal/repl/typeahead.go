package repl

import (
	"sync"
	"time"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

// startTypeaheadPump moves host keystrokes into the typeahead FIFO while a
// command runs (spec 035). Interactive readers (KeyRead) take the keyboard
// exclusively; the pump pauses then. Only terminals are pumped so scripted
// stdin is not stolen.
func startTypeaheadPump(ctx *context.Context) func() {
	if ctx == nil || ctx.Config == nil || ctx.Config.Typeahead <= 0 {
		return func() {}
	}
	if !term.IsTerminal(ctx.Stdin) {
		return func() {}
	}
	if ctx.Keys == nil {
		ctx.Keys = term.NewReader(ctx.Stdin)
	}
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		tick := time.NewTicker(8 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-done:
				return
			case <-tick.C:
				collectTypeahead(ctx)
			}
		}
	}()
	return func() {
		close(done)
		wg.Wait()
	}
}

// collectTypeahead pulls ready stdin bytes into the FIFO (testable without a TTY).
func collectTypeahead(ctx *context.Context) {
	if ctx == nil || ctx.KeyRead || ctx.Config == nil || ctx.Config.Typeahead <= 0 {
		return
	}
	if ctx.Keys == nil {
		return
	}
	for {
		b, ok := ctx.Keys.TryInputByte()
		if !ok {
			return
		}
		ctx.PushTypeahead(b)
	}
}

// drainTypeaheadIntoKeys feeds buffered typeahead into the key reader so
// READ / EDIT / WAIT / the prompt see keys typed during the previous command.
func drainTypeaheadIntoKeys(ctx *context.Context) {
	if ctx == nil || len(ctx.TypeaheadBuf) == 0 {
		return
	}
	if ctx.Keys == nil {
		ctx.Keys = term.NewReader(ctx.Stdin)
	}
	var tmp []byte
	for {
		b, ok := ctx.PopTypeahead()
		if !ok {
			break
		}
		tmp = append(tmp, b)
	}
	for i := len(tmp) - 1; i >= 0; i-- {
		ctx.Keys.PushFront(tmp[i])
	}
}
