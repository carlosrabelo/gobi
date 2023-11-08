package repl

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func handleModify(ctx *context.Context, cmd Command) error {
	if strings.ToUpper(strings.TrimSpace(cmd.Args)) != "STRUCTURE" {
		return fmt.Errorf("*** MODIFY: feature not yet implemented")
	}

	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	wseeker, ok := area.Table.Underlying().(io.ReadWriteSeeker)
	if !ok {
		return fmt.Errorf("*** Database file is not writable")
	}

	if area.Table.Header.RecordCount > 0 {
		ok, err := confirmDataLoss(ctx, ctx.StdinReader())
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}

	return runModifyStructureLineMode(ctx, area, wseeker)
}

func runModifyStructureLineMode(ctx *context.Context, area *context.WorkArea, wseeker io.ReadWriteSeeker) error {
	fmt.Fprintln(ctx.Stdout, "ENTER RECORD STRUCTURE AS FOLLOWS:")
	fmt.Fprintln(ctx.Stdout, "FIELD NAME,TYPE,WIDTH,DECIMAL PLACES")
	for i, fd := range area.Table.Fields {
		fmt.Fprintf(ctx.Stdout, "%03d %s\r\n", i+1, fieldToStructureLine(fd))
	}

	fields, err := readCreateFieldDefinitions(ctx, ctx.StdinReader())
	if err != nil {
		return err
	}

	if _, err := dbf.RewriteStructure(wseeker, fields); err != nil {
		return fmt.Errorf("*** Error modifying structure: %w", err)
	}
	if _, err := wseeker.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("*** Error seeking database: %w", err)
	}
	newTbl, err := dbf.Open(wseeker)
	if err != nil {
		return fmt.Errorf("*** Error reopening database: %w", err)
	}

	area.Table = newTbl
	area.RecordNo = 0
	area.ActiveRecord = nil
	return nil
}

func fieldToStructureLine(fd dbf.FieldDescriptor) string {
	return fmt.Sprintf("%s,%c,%d,%d", fd.Name, fd.Type, fd.Length, fd.DecimalCount)
}

func confirmDataLoss(ctx *context.Context, reader *bufio.Reader) (bool, error) {
	line, err := readAppendLine(ctx, reader, "ALL DATA WILL BE LOST. CONTINUE? (Y/N) ")
	if err != nil {
		if err == io.EOF {
			return false, nil
		}
		return false, fmt.Errorf("*** Error reading input: %w", err)
	}

	switch strings.ToUpper(strings.TrimSpace(line)) {
	case "Y":
		return true, nil
	case "N", "":
		return false, nil
	default:
		return false, fmt.Errorf("*** Invalid response")
	}
}
