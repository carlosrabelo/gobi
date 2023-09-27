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
	fmt.Fprintf(ctx.Stdout, "File: %s\n", a.Path)
	fmt.Fprintf(ctx.Stdout, "Records: %d\n", a.RecCount())
	for _, f := range a.Table.Header.Fields {
		fmt.Fprintf(ctx.Stdout, "%-10s %c %3d %3d\n", f.Name, f.Type, f.Length, f.Decimals)
	}
	return nil
}
