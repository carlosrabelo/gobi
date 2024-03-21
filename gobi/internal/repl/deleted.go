package repl

import (
	"fmt"
	"io"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

// applySetDeleted implements SET DELETED ON/OFF. With DELETED ON, records
// marked for deletion are hidden from scans (LIST, DISPLAY, COUNT, LOCATE)
// and from record navigation (GO TOP/BOTTOM, SKIP).
func applySetDeleted(ctx *context.Context, parts []string) {
	ctx.Config.Deleted = parseOnOff(parts)
	fmt.Fprintf(ctx.Stdout, "Deleted: %s\r\n", onOffStr(ctx.Config.Deleted))
}

// deletedHidden reports whether a record must be hidden under SET DELETED ON.
func deletedHidden(ctx *context.Context, rec *dbf.Record) bool {
	return ctx.Config.Deleted && rec != nil && rec.Deleted
}

// skipVisible moves the record pointer by delta visible records, skipping
// records hidden by SET DELETED ON. Forward movement past the last visible
// record parks the pointer at EOF.
func skipVisible(ctx *context.Context, area *context.WorkArea, delta int) error {
	seq, err := recordSequence(area)
	if err != nil {
		return err
	}
	rseeker, ok := area.Table.Underlying().(io.ReadSeeker)
	if !ok {
		return fmt.Errorf("*** Underlying database stream is not seekable")
	}

	recCount := int(area.Table.Header.RecordCount)
	pos := positionInSequence(seq, area.RecordNo, recCount)
	step := 1
	remaining := delta
	if delta < 0 {
		step = -1
		remaining = -delta
	}

	for remaining > 0 {
		pos += step
		if pos < 0 {
			return fmt.Errorf("*** Record number out of range")
		}
		if pos >= len(seq) {
			return goToRecord(ctx, recCount+1)
		}
		rec, err := area.Table.ReadRecordAt(rseeker, seq[pos])
		if err != nil {
			return fmt.Errorf("*** Error reading record %d: %w", seq[pos]+1, err)
		}
		if !rec.Deleted {
			remaining--
		}
	}
	return goToRecord(ctx, seq[pos]+1)
}

// goEdgeVisible positions the pointer on the first (top) or last visible
// record under SET DELETED ON, parking at EOF when every record is hidden.
func goEdgeVisible(ctx *context.Context, area *context.WorkArea, top bool) error {
	seq, err := recordSequence(area)
	if err != nil {
		return err
	}
	rseeker, ok := area.Table.Underlying().(io.ReadSeeker)
	if !ok {
		return fmt.Errorf("*** Underlying database stream is not seekable")
	}

	recCount := int(area.Table.Header.RecordCount)
	start, step := 0, 1
	if !top {
		start, step = len(seq)-1, -1
	}

	for pos := start; pos >= 0 && pos < len(seq); pos += step {
		rec, err := area.Table.ReadRecordAt(rseeker, seq[pos])
		if err != nil {
			return fmt.Errorf("*** Error reading record %d: %w", seq[pos]+1, err)
		}
		if !rec.Deleted {
			return goToRecord(ctx, seq[pos]+1)
		}
	}
	return goToRecord(ctx, recCount+1)
}
