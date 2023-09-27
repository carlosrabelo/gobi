package repl

import (
	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

// sessionEnv resolves fields and built-ins against the session.
type sessionEnv struct {
	ctx *context.Context
}

func newEnv(ctx *context.Context) *sessionEnv {
	return &sessionEnv{ctx: ctx}
}

func (e *sessionEnv) Memory(string) (expr.Value, bool) {
	return expr.Value{}, false
}

func (e *sessionEnv) Field(alias, name string) (expr.Value, bool) {
	return e.ctx.FieldValue(alias, name)
}

func (e *sessionEnv) Call(name string, args []expr.Value) (expr.Value, error) {
	return expr.CallBuiltin(name, args, e.ctx.ExprArea())
}

func (e *sessionEnv) Exact() (on bool, ok bool) {
	return e.ctx.Config.Exact, true
}
