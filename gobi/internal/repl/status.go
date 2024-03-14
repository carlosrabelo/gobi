package repl

import (
	"fmt"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

// outputStatus implements DISPLAY STATUS / LIST STATUS: it reports the
// selected work area, the database and index files open in each area, and
// the current SET option values.
func outputStatus(ctx *context.Context) error {
	fmt.Fprintf(ctx.Stdout, "CURRENTLY SELECTED DATABASE: %s\r\n\r\n", ctx.ActiveArea)

	for _, name := range []context.AreaName{context.Primary, context.Secondary} {
		printAreaStatus(ctx, name)
	}

	fmt.Fprint(ctx.Stdout, "\r\n")
	fmt.Fprintf(ctx.Stdout, "SET TALK       %s\r\n", onOff(ctx.Config.Talk))
	fmt.Fprintf(ctx.Stdout, "SET INTENSITY  %s\r\n", onOff(ctx.Config.Intensity))
	fmt.Fprintf(ctx.Stdout, "SET BELL       %s\r\n", onOff(ctx.Config.Bell))
	fmt.Fprintf(ctx.Stdout, "SET EXACT      %s\r\n", onOff(ctx.Config.Exact))
	fmt.Fprintf(ctx.Stdout, "SET DELETED    %s\r\n", onOff(ctx.Config.Deleted))
	fmt.Fprintf(ctx.Stdout, "SET SCREEN     %s\r\n", screenMode(ctx.Config.ScreenAuto))
	fmt.Fprintf(ctx.Stdout, "SET DEFAULT TO %s\r\n", ctx.Config.DefaultDir)
	return nil
}

// printAreaStatus reports the database and indexes bound to one work area.
func printAreaStatus(ctx *context.Context, name context.AreaName) {
	area := ctx.WorkAreas[name]
	if area == nil || area.Table == nil {
		fmt.Fprintf(ctx.Stdout, "%s DATABASE IN USE: NONE\r\n", name)
		return
	}

	path, ok := tableFilePath(area.Table)
	if !ok {
		path = area.Alias
	}
	fmt.Fprintf(ctx.Stdout, "%s DATABASE IN USE: %s  ALIAS: %s\r\n", name, path, area.Alias)

	for _, idx := range area.Indexes {
		if idx == nil {
			continue
		}
		key := ""
		if idx.Manager() != nil {
			key = idx.Manager().Header().Expression
		}
		fmt.Fprintf(ctx.Stdout, "    INDEX FILE: %s  KEY: %s\r\n", idx.Path, key)
	}
}

func onOff(v bool) string {
	if v {
		return "ON"
	}
	return "OFF"
}

func screenMode(auto bool) string {
	if auto {
		return "AUTO"
	}
	return "DEFAULT"
}
