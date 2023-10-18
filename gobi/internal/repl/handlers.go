package repl

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func handleQuit(ctx *context.Context, cmd Command) error {
	return errQuit
}

func handleSelect(ctx *context.Context, cmd Command) error {
	arg := strings.ToUpper(strings.TrimSpace(cmd.Args))
	if arg == "" {
		return fmt.Errorf("*** SELECT requires PRIMARY or SECONDARY")
	}

	var msg string
	switch arg {
	case "PRIMARY":
		if err := ctx.SelectArea(context.Primary); err != nil {
			return err
		}
		msg = "Primary work area selected"
	case "SECONDARY":
		if err := ctx.SelectArea(context.Secondary); err != nil {
			return err
		}
		msg = "Secondary work area selected"
	default:
		return fmt.Errorf("*** Unrecognized SELECT option: %s", arg)
	}

	fmt.Fprintln(ctx.Stdout, msg)
	return nil
}

func closeWorkAreaDatabase(area *context.WorkArea, defaultAlias string) error {
	if area.Table == nil {
		return nil
	}
	if err := area.Table.Close(); err != nil {
		return fmt.Errorf("*** Error closing database: %w", err)
	}
	area.Table = nil
	area.RecordNo = 0
	area.ActiveRecord = nil
	area.Alias = defaultAlias
	clearLocateState(area)
	return nil
}

func handleClose(ctx *context.Context, cmd Command) error {
	arg := strings.ToUpper(strings.TrimSpace(cmd.Args))
	switch arg {
	case "", "DATABASES":
		for name, area := range ctx.WorkAreas {
			if err := closeWorkAreaDatabase(area, string(name)); err != nil {
				return err
			}
		}
		fmt.Fprintln(ctx.Stdout, "Database area closed")
	case "INDEX":
		area := ctx.GetActiveArea()
		closeOpenIndexes(area)
		fmt.Fprintln(ctx.Stdout, "Indexes closed")
	case "ALL":
		for name, area := range ctx.WorkAreas {
			if err := closeWorkAreaDatabase(area, string(name)); err != nil {
				return err
			}
			closeOpenIndexes(area)
		}
		fmt.Fprintln(ctx.Stdout, "All files closed")
	default:
		return fmt.Errorf("*** Unrecognized CLOSE option: %s", arg)
	}
	return nil
}

func handleDisplay(ctx *context.Context, cmd Command) error {
	arg := strings.ToUpper(strings.TrimSpace(cmd.Args))
	if arg == "STRUCTURE" {
		return displayStructure(ctx)
	}
	return fmt.Errorf("*** DISPLAY: feature not yet implemented")
}

func handleList(ctx *context.Context, cmd Command) error {
	argUpper := strings.ToUpper(strings.TrimSpace(cmd.Args))
	if argUpper == "STRUCTURE" {
		return displayStructure(ctx)
	}
	return fmt.Errorf("*** LIST: feature not yet implemented")
}

func displayStructure(ctx *context.Context) error {
	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	tbl := area.Table
	fmt.Fprintf(ctx.Stdout, "STRUCTURE FOR FILE:  %s.DBF\r\n", area.Alias)
	fmt.Fprintf(ctx.Stdout, "NUMBER OF RECORDS:   %05d\r\n", tbl.Header.RecordCount)
	fmt.Fprintf(ctx.Stdout, "DATE OF LAST UPDATE: 00/00/00\r\n")
	fmt.Fprintf(ctx.Stdout, "FLD       NAME       TYPE WIDTH DEC\r\n")

	totalWidth := 1
	for i, fd := range tbl.Fields {
		decStr := ""
		if fd.Type == dbf.FieldTypeNumeric {
			decStr = fmt.Sprintf("%03d", fd.DecimalCount)
		}
		fmt.Fprintf(ctx.Stdout, "%03d       %-10s  %c   %03d   %3s\r\n",
			i+1, fd.Name, fd.Type, fd.Length, decStr)
		totalWidth += int(fd.Length)
	}
	fmt.Fprintf(ctx.Stdout, "** TOTAL **                %05d\r\n", totalWidth)
	return nil
}

func handleGo(ctx *context.Context, cmd Command) error {
	arg := strings.ToUpper(strings.TrimSpace(cmd.Args))
	parts := strings.Fields(arg)
	if len(parts) == 2 && parts[0] == "TO" {
		return gotoRecordFromArgs(ctx, parts[1])
	}
	return fmt.Errorf("*** Unrecognized GO option: %s", strings.TrimSpace(cmd.Args))
}

func handleGoto(ctx *context.Context, cmd Command) error {
	return gotoRecordFromArgs(ctx, strings.TrimSpace(cmd.Args))
}

func gotoRecordFromArgs(ctx *context.Context, arg string) error {
	if arg == "" {
		return fmt.Errorf("*** GOTO requires a record number")
	}
	userRecNo, err := strconv.Atoi(arg)
	if err != nil {
		return fmt.Errorf("*** Invalid record number: %s", arg)
	}
	return goToRecord(ctx, userRecNo)
}

func goToRecord(ctx *context.Context, userRecNo int) error {
	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}
	if userRecNo < 1 {
		return fmt.Errorf("*** Record number out of range")
	}

	recCount := int(area.Table.Header.RecordCount)
	recIdx := userRecNo - 1

	if recIdx >= recCount {
		area.RecordNo = recIdx
		area.ActiveRecord = nil
		return nil
	}

	rseeker, ok := area.Table.Underlying().(io.ReadSeeker)
	if !ok {
		return fmt.Errorf("*** Underlying database stream is not seekable")
	}

	rec, err := area.Table.ReadRecordAt(rseeker, recIdx)
	if err != nil {
		return fmt.Errorf("*** Error reading record %d: %w", userRecNo, err)
	}

	area.RecordNo = recIdx
	area.ActiveRecord = rec
	return nil
}
