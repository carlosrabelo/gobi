package repl

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func handleEdit(ctx *context.Context, cmd Command) error {
	arg := strings.TrimSpace(cmd.Args)
	if arg == "" {
		return fmt.Errorf("*** EDIT requires a record number")
	}

	userRecNo, err := strconv.Atoi(arg)
	if err != nil {
		return fmt.Errorf("*** Invalid record number: %s", arg)
	}

	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	recCount := int(area.Table.Header.RecordCount)
	if userRecNo < 1 || userRecNo > recCount {
		return fmt.Errorf("*** Record number out of range")
	}

	if err := goToRecord(ctx, userRecNo); err != nil {
		return err
	}
	return editRecordLineMode(ctx)
}

func editRecordLineMode(ctx *context.Context) error {
	area := ctx.GetActiveArea()
	wseeker, ok := area.Table.Underlying().(io.ReadWriteSeeker)
	if !ok {
		return fmt.Errorf("*** Database file is not writable")
	}

	tbl := area.Table
	rec := area.ActiveRecord
	if rec == nil {
		return fmt.Errorf("*** Record number out of range")
	}

	values := make([]interface{}, len(tbl.Fields))
	reader := ctx.StdinReader()
	for i, fd := range tbl.Fields {
		current, err := rec.DecodeField(tbl, i)
		if err != nil {
			return fmt.Errorf("*** Error decoding field %s: %w", fd.Name, err)
		}
		prompt := fmt.Sprintf("%s : %s\r\n%s ? ", fd.Name, formatEditPromptValue(current), fd.Name)
		line, err := readAppendLine(ctx, reader, prompt)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("*** Error reading input: %w", err)
		}
		if strings.TrimSpace(line) == "" {
			values[i] = current
			continue
		}
		val, err := parseFieldInput(fd, line)
		if err != nil {
			return err
		}
		values[i] = val
	}

	updated, err := dbf.NewRecord(tbl, rec.Deleted, values)
	if err != nil {
		return fmt.Errorf("*** Error building record: %w", err)
	}
	if err := tbl.WriteRecordAt(wseeker, area.RecordNo, updated); err != nil {
		return fmt.Errorf("*** Error writing record: %w", err)
	}
	area.ActiveRecord = updated
	return nil
}

func formatEditPromptValue(val interface{}) string {
	if val == nil {
		return ""
	}
	return strings.TrimRight(fmt.Sprintf("%v", val), " ")
}
