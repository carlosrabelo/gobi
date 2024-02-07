package repl

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

func (m *Mux) cmdCLEAR(ctx *context.Context, cmd command.Command) error {
	args := strings.ToUpper(strings.TrimSpace(cmd.Args))
	switch {
	case args == "" || args == "SCREEN":
		scr := ensureScreen(ctx)
		scr.Clear()
		_ = flushScreenClear(ctx)
		return nil
	case args == "ALL":
		_ = ctx.CloseAll()
		ctx.Symbols.ReleaseAll()
		m.gets = nil
		_ = ctx.Select("1")
		return nil
	case args == "MEMORY":
		ctx.Symbols.ReleaseAll()
		return nil
	case args == "GETS":
		m.gets = nil
		return nil
	case args == "FIELDS":
		return clearFields(ctx)
	case args == "TYPEAHEAD":
		ctx.TypeaheadBuf = nil
		return nil
	default:
		return fail(ctx, "invalid CLEAR")
	}
}

func cmdERASE(ctx *context.Context, cmd command.Command) error {
	name := strings.TrimSpace(cmd.Args)
	if name == "" {
		return fail(ctx, "ERASE requires a name")
	}
	path, err := resolveAny(ctx, strings.Fields(name)[0])
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if err := os.Remove(path); err != nil {
		return fail(ctx, "%s", err.Error())
	}
	return nil
}

func cmdCHANGE(ctx *context.Context, cmd command.Command) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	scope := parseScope(cmd)
	forExpr, err := parseOptionalLogic(cmd.Clauses["FOR"])
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	whileExpr, err := parseOptionalLogic(cmd.Clauses["WHILE"])
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	args := strings.TrimSpace(scope.Rest)
	up := strings.ToUpper(args)
	if !strings.HasPrefix(up, "FIELD ") {
		return fail(ctx, "CHANGE requires FIELD")
	}
	rest := strings.TrimSpace(args[6:])
	names := splitCommaExprs(rest)
	if len(names) == 0 {
		return fail(ctx, "CHANGE requires a field")
	}
	count := a.RecCount()
	start, end := scopeRange(count, a.Cursor, scope)
	if scope.Kind == "" {
		if forExpr != nil || whileExpr != nil {
			start, end = 1, count
		} else {
			if !a.Landed {
				return fail(ctx, "no current record")
			}
			start, end = a.Cursor, a.Cursor
		}
	}
	if end < start {
		return nil
	}
	env := newEnv(ctx)
	for n := start; n <= end; n++ {
		a.Cursor = n
		a.BOF = false
		if err := a.LoadCurrent(); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		if ctx.Config.Deleted && a.Record.Deleted {
			continue
		}
		if whileExpr != nil {
			ok, e := evalLogic(whileExpr, env)
			if e != nil {
				return fail(ctx, "%s", e.Error())
			}
			if !ok {
				break
			}
		}
		if forExpr != nil {
			ok, e := evalLogic(forExpr, env)
			if e != nil {
				return fail(ctx, "%s", e.Error())
			}
			if !ok {
				continue
			}
		}
		for _, name := range names {
			fi := fieldIndex(a, strings.TrimSpace(name))
			if fi < 0 {
				return fail(ctx, "unknown field: %s", name)
			}
			f := a.Table.Header.Fields[fi]
			cur := fieldDisplay(a.Record.Values[fi], f)
			fmt.Fprintln(ctx.Out(), cur)
			fmt.Fprint(ctx.Out(), "CHANGE? ")
			from, err := readInteractiveLine(ctx)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil // leave CHANGE quietly (batch / redirected stdin)
				}
				return fail(ctx, "%s", err.Error())
			}
			if from == "" {
				continue
			}
			fmt.Fprint(ctx.Out(), "TO? ")
			to, err := readInteractiveLine(ctx)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return fail(ctx, "%s", err.Error())
			}
			newVal := strings.Replace(cur, from, to, 1)
			v, err := parseFieldInput(f, newVal)
			if err != nil {
				return fail(ctx, "%s", err.Error())
			}
			rec := a.Record
			rec.Values[fi] = v
			if err := a.Table.Write(a.Cursor, rec); err != nil {
				return fail(ctx, "%s", err.Error())
			}
			a.Record = rec
		}
	}
	return syncAllIndexes(ctx)
}
