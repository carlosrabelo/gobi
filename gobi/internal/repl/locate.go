package repl

import (
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

func cmdLOCATE(ctx *context.Context, cmd command.Command) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	forSrc := strings.TrimSpace(cmd.Clauses["FOR"])
	whileSrc := strings.TrimSpace(cmd.Clauses["WHILE"])
	if forSrc == "" && whileSrc == "" {
		return fail(ctx, "LOCATE requires FOR or WHILE")
	}
	ctx.LocateActive = true
	ctx.LocateFOR = forSrc
	ctx.LocateWHILE = whileSrc

	start := 1
	if whileSrc != "" {
		start = a.Cursor
		if start < 1 {
			start = 1
		}
	}
	return runLocate(ctx, start)
}

func cmdCONTINUE(ctx *context.Context, cmd command.Command) error {
	if !ctx.LocateActive {
		return fail(ctx, "no previous LOCATE")
	}
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	return runLocate(ctx, a.Cursor+1)
}

func runLocate(ctx *context.Context, start int) error {
	a := ctx.Active()
	forExpr, err := parseOptionalLogic(ctx.LocateFOR)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	whileExpr, err := parseOptionalLogic(ctx.LocateWHILE)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	env := newEnv(ctx)
	count := a.RecCount()
	for n := start; n <= count; n++ {
		a.Cursor = n
		a.BOF = false
		if err := a.LoadCurrent(); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		if ctx.Config.Deleted && a.Record.Deleted {
			continue
		}
		if whileExpr != nil {
			ok, err := evalLogic(whileExpr, env)
			if err != nil {
				return fail(ctx, "%s", err.Error())
			}
			if !ok {
				break
			}
		}
		if forExpr != nil {
			ok, err := evalLogic(forExpr, env)
			if err != nil {
				return fail(ctx, "%s", err.Error())
			}
			if !ok {
				continue
			}
		}
		a.Found = true
		return nil
	}
	a.Found = false
	a.Cursor = count + 1
	a.BOF = false
	_ = a.LoadCurrent()
	return nil
}
