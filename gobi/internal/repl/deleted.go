package repl

import (
	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func deletedHidden(ctx *context.Context, rec *dbf.Record) bool {
	return ctx != nil && ctx.Config != nil && ctx.Config.Deleted && rec != nil && rec.Deleted
}
