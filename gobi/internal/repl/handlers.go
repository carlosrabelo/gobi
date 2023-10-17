package repl

import (
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
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
