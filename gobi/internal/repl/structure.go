package repl

import (
	"fmt"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

func displayStructure(ctx *context.Context) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	fmt.Fprintf(ctx.Out(), "File: %s\n", a.Path)
	fmt.Fprintf(ctx.Out(), "Records: %d\n", a.RecCount())
	for _, f := range a.Table.Header.Fields {
		fmt.Fprintf(ctx.Out(), "%-10s %c %3d %3d\n", f.Name, f.Type, f.Length, f.Decimals)
	}
	return nil
}
