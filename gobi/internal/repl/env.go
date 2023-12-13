package repl

import (
	"fmt"
	"os"
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
	case "ERROR":
		if len(args) != 0 {
			return expr.Value{}, fmt.Errorf("ERROR: wrong number of arguments")
		}
		return expr.NumberVal(float64(e.ctx.LastError)), nil
	case "MESSAGE":
		if len(args) != 0 {
			return expr.Value{}, fmt.Errorf("MESSAGE: wrong number of arguments")
		}
		return expr.StringVal(e.ctx.LastMessage), nil
	case "ALIAS":
		if len(args) != 0 {
			return expr.Value{}, fmt.Errorf("ALIAS: wrong number of arguments")
		}
		a := e.ctx.Active()
		if !a.HasTable() {
			return expr.StringVal(""), nil
		}
		return expr.StringVal(a.Alias), nil
	case "SELECT":
		if len(args) != 0 {
			return expr.Value{}, fmt.Errorf("SELECT: wrong number of arguments")
		}
		return expr.NumberVal(float64(e.ctx.ActiveArea)), nil
	case "ISCOLOR":
		if len(args) != 0 {
			return expr.Value{}, fmt.Errorf("ISCOLOR: wrong number of arguments")
		}
		return expr.LogicalVal(e.ctx.Config.Color != ""), nil
	case "GETENV":
		if len(args) != 1 {
			return expr.Value{}, fmt.Errorf("GETENV: wrong number of arguments")
		}
		return expr.StringVal(os.Getenv(strings.ToUpper(args[0].String()))), nil
	case "PCOL":
		if len(args) != 0 {
			return expr.Value{}, fmt.Errorf("PCOL: wrong number of arguments")
		}
		return expr.NumberVal(float64(e.ctx.PrintCol)), nil
	case "PROW":
		if len(args) != 0 {
			return expr.Value{}, fmt.Errorf("PROW: wrong number of arguments")
		}
		return expr.NumberVal(float64(e.ctx.PrintRow)), nil
	case "READKEY":
		if len(args) != 0 {
			return expr.Value{}, fmt.Errorf("READKEY: wrong number of arguments")
		}
		return expr.NumberVal(float64(e.ctx.ReadKey)), nil
	case "UPDATED":
		if len(args) != 0 {
			return expr.Value{}, fmt.Errorf("UPDATED: wrong number of arguments")
		}
		return expr.LogicalVal(e.ctx.Updated), nil
	case "FKLABEL":
		if len(args) != 1 {
			return expr.Value{}, fmt.Errorf("FKLABEL: wrong number of arguments")
		}
		n := int(args[0].Num)
		if n < 1 || n > 12 {
			return expr.StringVal(""), nil
		}
		return expr.StringVal(e.ctx.FuncKeys[n]), nil
	case "FLOCK", "RLOCK", "LOCK":
		return expr.Value{}, fmt.Errorf("%s is not available in Gobi", strings.ToUpper(name))
	}
	return expr.CallBuiltin(name, args, e.ctx.ExprArea(), e)
}

func (e *sessionEnv) Exact() (on bool, ok bool) {
	return e.ctx.Config.Exact, true
}

func (e *sessionEnv) Fixed() (on bool, decimals int, ok bool) {
	return e.ctx.Config.Fixed, e.ctx.Config.Decimals, true
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
