package repl

import (
	"fmt"
	"io"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

const (
	appendTitleRow      = 0
	appendTitleCol      = 2
	appendFirstFieldRow = 2
	appendLabelCol      = 2
	appendGetCol        = 16
)

func appendScreenAvailable(ctx *context.Context) bool {
	return interactiveTerminal(ctx)
}

func runAppendScreen(ctx *context.Context) error {
	area := ctx.GetActiveArea()
	tbl := area.Table
	wseeker, ok := tbl.Underlying().(io.ReadWriteSeeker)
	if !ok {
		return fmt.Errorf("*** Database file is not writable")
	}

	for {
		if err := buildAppendScreen(ctx, tbl); err != nil {
			return err
		}
		if err := ctx.Screen.Present(ctx.Stdout); err != nil {
			return err
		}
		if err := ctx.Screen.MoveToCommandLine(ctx.Stdout); err != nil {
			return err
		}

		if err := newBlankActiveRecord(area, tbl); err != nil {
			return fmt.Errorf("*** Error preparing append record: %w", err)
		}

		if err := runReadForm(ctx); err != nil {
			if err == errReadAbort {
				return nil
			}
			return err
		}

		blank, err := isBlankAppendRecord(tbl, area.ActiveRecord)
		if err != nil {
			return fmt.Errorf("*** Error validating append record: %w", err)
		}
		if blank {
			return nil
		}

		recNo, err := tbl.AppendRecord(wseeker, area.ActiveRecord)
		if err != nil {
			return fmt.Errorf("*** Error appending record: %w", err)
		}

		area.RecordNo = recNo
		if err := syncOpenIndexesAfterAppend(ctx, area, recNo); err != nil {
			return err
		}
		talkPrint(ctx, "New record added\r\n")
	}
}

func buildAppendScreen(ctx *context.Context, tbl *dbf.Table) error {
	if ctx.Screen == nil {
		return fmt.Errorf("*** Screen buffer not initialized")
	}

	ctx.Screen.Clear()
	alias := ctx.GetActiveArea().Alias
	ctx.Screen.WriteAt(appendTitleRow, appendTitleCol, "APPEND - "+alias)

	lastRow := ctx.Screen.CommandLineRow() - 1
	row := appendFirstFieldRow
	for _, fd := range tbl.Fields {
		if row > lastRow {
			break
		}

		label := fd.Name + ":"
		ctx.Screen.WriteAt(row, appendLabelCol, label)

		picture := fieldPicture(fd)
		ctx.Screen.RegisterGet(row, appendGetCol, fd.Name, picture)
		ctx.Screen.WriteAt(row, appendGetCol, strings.Repeat(" ", pictureDisplayWidth(picture, fd)))
		row++
	}

	return nil
}

func fieldPicture(fd dbf.FieldDescriptor) string {
	switch fd.Type {
	case dbf.FieldTypeChar:
		return strings.Repeat("X", int(fd.Length))
	case dbf.FieldTypeNumeric:
		if fd.DecimalCount > 0 {
			whole := int(fd.Length) - int(fd.DecimalCount) - 1
			if whole < 1 {
				whole = 1
			}
			return strings.Repeat("9", whole) + "." + strings.Repeat("9", int(fd.DecimalCount))
		}
		return strings.Repeat("9", int(fd.Length))
	case dbf.FieldTypeLogical:
		return "L"
	default:
		return strings.Repeat("X", int(fd.Length))
	}
}

func pictureDisplayWidth(picture string, fd dbf.FieldDescriptor) int {
	if picture != "" {
		return len(picture)
	}
	return int(fd.Length)
}

func newBlankActiveRecord(area *context.WorkArea, tbl *dbf.Table) error {
	values := make([]interface{}, len(tbl.Fields))
	for i, fd := range tbl.Fields {
		switch fd.Type {
		case dbf.FieldTypeChar:
			values[i] = ""
		case dbf.FieldTypeNumeric:
			values[i] = float64(0)
		case dbf.FieldTypeLogical:
			values[i] = false
		default:
			values[i] = ""
		}
	}

	rec, err := dbf.NewRecord(tbl, false, values)
	if err != nil {
		return err
	}
	area.ActiveRecord = rec
	return nil
}

func isBlankAppendRecord(tbl *dbf.Table, rec *dbf.Record) (bool, error) {
	if rec == nil {
		return true, nil
	}

	for i, fd := range tbl.Fields {
		val, err := rec.DecodeField(tbl, i)
		if err != nil {
			return false, err
		}

		switch fd.Type {
		case dbf.FieldTypeChar:
			if strings.TrimSpace(val.(string)) != "" {
				return false, nil
			}
		case dbf.FieldTypeNumeric:
			if val.(float64) != 0 {
				return false, nil
			}
		case dbf.FieldTypeLogical:
			if val.(bool) {
				return false, nil
			}
		}
	}

	return true, nil
}
