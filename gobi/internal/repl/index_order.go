package repl

import (
	"fmt"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

// controllingIndexOrder returns the 0-based record indexes of the active
// table in controlling-index key order, or nil when no index is bound.
// Entries pointing past the current record count are dropped.
func controllingIndexOrder(area *context.WorkArea) ([]int, error) {
	if area == nil || area.Table == nil || len(area.Indexes) == 0 {
		return nil, nil
	}
	idx := area.Indexes[0]
	if idx == nil || idx.Manager() == nil {
		return nil, nil
	}

	records, err := idx.Manager().OrderedRecordNumbers()
	if err != nil {
		return nil, fmt.Errorf("*** Index read error: %w", err)
	}

	recCount := int(area.Table.Header.RecordCount)
	order := make([]int, 0, len(records))
	for _, rn := range records {
		recIdx := int(rn) - 1
		if recIdx >= 0 && recIdx < recCount {
			order = append(order, recIdx)
		}
	}
	return order, nil
}

// recordSequence returns the record iteration order for table scans: the
// controlling index order when one is bound, sequential otherwise.
func recordSequence(area *context.WorkArea) ([]int, error) {
	order, err := controllingIndexOrder(area)
	if err != nil {
		return nil, err
	}
	if order != nil {
		return order, nil
	}

	recCount := int(area.Table.Header.RecordCount)
	seq := make([]int, recCount)
	for i := range seq {
		seq[i] = i
	}
	return seq, nil
}

// positionInSequence finds the position of record index recNo within seq.
// Records at or past EOF (and stale positions not present in seq) map to
// len(seq), i.e. one past the last sequence entry.
func positionInSequence(seq []int, recNo, recCount int) int {
	if recNo >= recCount {
		return len(seq)
	}
	for pos, recIdx := range seq {
		if recIdx == recNo {
			return pos
		}
	}
	return len(seq)
}
