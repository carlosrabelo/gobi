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
	// III Plus–like STRUCTURE: path, record count, field table, total width.
	fmt.Fprintf(ctx.Out(), "Structure for database: %s\n", a.Path)
	fmt.Fprintf(ctx.Out(), "Number of data records: %8d\n", a.RecCount())
	if t, ok := dbfHeaderDate(a.Table.Header); ok {
		fmt.Fprintf(ctx.Out(), "Date of last update   : %s\n", t.Format("01/02/06"))
	}
	fmt.Fprintf(ctx.Out(), "%5s  %-10s %-9s %5s %4s\n", "Field", "Field Name", "Type", "Width", "Dec")
	total := 1 // deletion mark
	for i, f := range a.Table.Header.Fields {
		typeName := structureTypeName(f.Type)
		fmt.Fprintf(ctx.Out(), "%5d  %-10s %-9s %5d %4d\n", i+1, f.Name, typeName, f.Length, f.Decimals)
		total += int(f.Length)
	}
	fmt.Fprintf(ctx.Out(), "** Total **                %5d\n", total)
	return nil
}

func structureTypeName(t byte) string {
	switch t {
	case 'C':
		return "Character"
	case 'N':
		return "Numeric"
	case 'L':
		return "Logical"
	case 'D':
		return "Date"
	case 'M':
		return "Memo"
	default:
		return string(t)
	}
}
