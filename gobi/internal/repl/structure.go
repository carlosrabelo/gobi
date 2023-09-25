package repl

import (
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

func cmdDISPLAY(ctx *context.Context, cmd command.Command) error {
	return structureOrReject(ctx, cmd, "DISPLAY")
}

func cmdLIST(ctx *context.Context, cmd command.Command) error {
	return structureOrReject(ctx, cmd, "LIST")
}

func structureOrReject(ctx *context.Context, cmd command.Command, verb string) error {
	if !strings.EqualFold(strings.TrimSpace(cmd.Args), "STRUCTURE") {
		return fail(ctx, "%s STRUCTURE required", verb)
	}
	return displayStructure(ctx)
}

func displayStructure(ctx *context.Context) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	fmt.Fprintf(ctx.Stdout, "File: %s\n", a.Path)
	fmt.Fprintf(ctx.Stdout, "Records: %d\n", a.RecCount())
	for _, f := range a.Table.Header.Fields {
		fmt.Fprintf(ctx.Stdout, "%-10s %c %3d %3d\n", f.Name, f.Type, f.Length, f.Decimals)
	}
	return nil
}
