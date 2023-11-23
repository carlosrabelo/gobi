package repl

import (
	"fmt"
	"strings"
	"time"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

// sessionEnv resolves fields and built-ins against the session.
type sessionEnv struct {
	ctx *context.Context
}

func newEnv(ctx *context.Context) *sessionEnv {
	return &sessionEnv{ctx: ctx}
}

func (e *sessionEnv) Memory(name string) (expr.Value, bool) {
	if e.ctx.Symbols == nil {
		return expr.Value{}, false
	}
	return e.ctx.Symbols.Get(name)
}

func (e *sessionEnv) Field(alias, name string) (expr.Value, bool) {
	return e.ctx.FieldValue(alias, name)
}

func (e *sessionEnv) Call(name string, args []expr.Value) (expr.Value, error) {
	switch strings.ToUpper(name) {
	case "ROW":
		if len(args) != 0 {
			return expr.Value{}, fmt.Errorf("ROW: wrong number of arguments")
		}
		scr := ensureScreen(e.ctx)
		return expr.NumberVal(float64(scr.NextRow)), nil
	case "COL":
		if len(args) != 0 {
			return expr.Value{}, fmt.Errorf("COL: wrong number of arguments")
		}
		scr := ensureScreen(e.ctx)
		return expr.NumberVal(float64(scr.NextCol)), nil
	case "INKEY":
		if len(args) != 0 {
			return expr.Value{}, fmt.Errorf("INKEY: wrong number of arguments")
		}
		if e.ctx.Keys == nil {
			e.ctx.Keys = term.NewReader(e.ctx.Stdin)
		}
		k := e.ctx.Keys.Inkey()
		return expr.NumberVal(float64(k)), nil
	}
	return expr.CallBuiltin(name, args, e.ctx.ExprArea(), e)
}

func (e *sessionEnv) Exact() (on bool, ok bool) {
	return e.ctx.Config.Exact, true
}

func (e *sessionEnv) Default() (path string, ok bool) {
	return e.ctx.Config.Default, true
}

func (e *sessionEnv) DateSettings() (order string, century bool, ok bool) {
	return e.ctx.Config.DateOrder, e.ctx.Config.Century, true
}

func (e *sessionEnv) Today() (time.Time, bool) {
	return e.ctx.SessionDate, true
}

func (e *sessionEnv) LUpdate() expr.Value {
	a := e.ctx.Active()
	if !a.HasTable() {
		return expr.EmptyDateVal()
	}
	h := a.Table.Header
	t := time.Date(1900+int(h.Year), time.Month(h.Month), int(h.Day), 0, 0, 0, 0, time.Local)
	return expr.DateVal(t)
}
