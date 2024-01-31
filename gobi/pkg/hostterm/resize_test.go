package hostterm

import (
	"sync/atomic"
	"testing"
)

func TestOnResizeNotify(t *testing.T) {
	var gotCols, gotRows atomic.Int32
	OnResize(0, func(c, r int) {
		gotCols.Store(int32(c))
		gotRows.Store(int32(r))
	})
	NotifyResizeForTest(120, 40)
	if gotCols.Load() != 120 || gotRows.Load() != 40 {
		t.Fatalf("got %dx%d", gotCols.Load(), gotRows.Load())
	}
	NotifyResizeForTest(20, 8)
	if gotCols.Load() != MinCols || gotRows.Load() != MinRows {
		t.Fatalf("floors: got %dx%d", gotCols.Load(), gotRows.Load())
	}
}
