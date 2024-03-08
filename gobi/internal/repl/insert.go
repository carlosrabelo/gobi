package repl

import (
	"fmt"
	"io"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

// handleInsert implements INSERT [BEFORE] [BLANK]: it inserts a record at the
// cursor position (after the current record, or before it with BEFORE),
// physically shifting the following records down. Without BLANK the field
// values are prompted like a single APPEND entry.
func handleInsert(ctx *context.Context, cmd Command) error {
	before, blank, err := parseInsertArgs(cmd.Args)
	if err != nil {
		return err
	}

	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	wseeker, ok := area.Table.Underlying().(io.ReadWriteSeeker)
	if !ok {
		return fmt.Errorf("*** Database file is not writable")
	}

	tbl := area.Table
	recCount := int(tbl.Header.RecordCount)

	insertAt := area.RecordNo + 1
	if before {
		insertAt = area.RecordNo
	}
	if insertAt > recCount {
		insertAt = recCount
	}
	if insertAt < 0 {
		insertAt = 0
	}

	values := make([]interface{}, len(tbl.Fields))
	for i := range values {
		values[i] = ""
	}
	if !blank {
		entered, cancelled, err := promptInsertValues(ctx, tbl)
		if err != nil {
			return err
		}
		if cancelled {
			return nil
		}
		values = entered
	}

	rec, err := dbf.NewRecord(tbl, false, values)
	if err != nil {
		return fmt.Errorf("*** Error building record: %w", err)
	}

	recNo, err := tbl.AppendRecord(wseeker, rec)
	if err != nil {
		return fmt.Errorf("*** Error appending record: %w", err)
	}

	// Shift the trailing records down to open the slot at insertAt.
	for i := recNo - 1; i >= insertAt; i-- {
		moved, err := tbl.ReadRecordAt(wseeker, i)
		if err != nil {
			return fmt.Errorf("*** Error reading record %d: %w", i+1, err)
		}
		if err := tbl.WriteRecordAt(wseeker, i+1, moved); err != nil {
			return fmt.Errorf("*** Error writing record %d: %w", i+2, err)
		}
	}
	if insertAt != recNo {
		if err := tbl.WriteRecordAt(wseeker, insertAt, rec); err != nil {
			return fmt.Errorf("*** Error writing record %d: %w", insertAt+1, err)
		}
	}

	area.RecordNo = insertAt
	area.ActiveRecord = rec

	// Physical record numbers changed; rebuild any open indexes quietly.
	if len(area.Indexes) > 0 {
		if err := reindexOpenIndexes(ctx, area, false); err != nil {
			return err
		}
	}

	talkPrint(ctx, "RECORD: %05d\r\n", insertAt+1)
	return nil
}

// parseInsertArgs interprets the optional BEFORE and BLANK keywords.
func parseInsertArgs(args string) (before, blank bool, err error) {
	for _, word := range strings.Fields(args) {
		switch strings.ToUpper(word) {
		case "BEFORE":
			before = true
		case "BLANK":
			blank = true
		default:
			return false, false, fmt.Errorf("*** Unexpected argument: %s", word)
		}
	}
	return before, blank, nil
}

// promptInsertValues asks for each field value like a single APPEND entry.
// An empty first field cancels the insertion.
func promptInsertValues(ctx *context.Context, tbl *dbf.Table) ([]interface{}, bool, error) {
	reader := ctx.StdinReader()
	values := make([]interface{}, len(tbl.Fields))

	for i, fd := range tbl.Fields {
		line, err := readAppendLine(ctx, reader, fmt.Sprintf("%s ? ", fd.Name))
		if err != nil {
			if err == io.EOF {
				return nil, true, nil
			}
			return nil, false, fmt.Errorf("*** Error reading input: %w", err)
		}

		if i == 0 && strings.TrimSpace(line) == "" {
			return nil, true, nil
		}

		val, err := parseFieldInput(fd, line)
		if err != nil {
			return nil, false, err
		}
		values[i] = val
	}
	return values, false, nil
}
