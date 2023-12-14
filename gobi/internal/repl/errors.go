package repl

import (
	"errors"
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/help"
)

func init() {
	help.Register("ON", "ON ERROR|ESCAPE|KEY command — trap events.")
	help.Register("RETRY", "RETRY — re-run the line that triggered ON ERROR.")
}

func cmdON(ctx *context.Context, cmd command.Command) error {
	args := strings.Fields(cmd.Args)
	if len(args) == 0 {
		return fail(ctx, "ON requires ERROR, ESCAPE, or KEY")
	}
	kind := strings.ToUpper(args[0])
	rest := strings.TrimSpace(cmd.Args[len(args[0]):])
	// Split pulls TO into Clauses; put it back so handlers like STORE … TO x work.
	if to, ok := cmd.Clauses["TO"]; ok {
		rest = strings.TrimSpace(rest + " TO " + to)
	}
	switch kind {
	case "ERROR":
		ctx.OnError = rest
	case "ESCAPE":
		ctx.OnEscape = rest
	case "KEY":
		ctx.OnKey = rest
	default:
		return fail(ctx, "ON requires ERROR, ESCAPE, or KEY")
	}
	return nil
}

func cmdRETRY(ctx *context.Context, m *Mux, cmd command.Command) error {
	if !ctx.InHandler || ctx.RetryLine == "" {
		return fail(ctx, "RETRY outside an error handler")
	}
	line := ctx.RetryLine
	ctx.InHandler = false
	expanded, err := expandLineMacros(ctx, line)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	return m.Dispatch(ctx, command.Split(expanded))
}

func classifyError(msg string) int {
	m := strings.TrimPrefix(msg, "*** ")
	switch {
	case strings.Contains(m, "end of file") || strings.Contains(m, "no current record"):
		return 4
	case strings.Contains(m, "too many files"):
		return 6
	case strings.Contains(m, "unknown name") || strings.Contains(m, "unknown field") || strings.Contains(m, "not found"):
		return 12
	case strings.Contains(m, "signature"):
		return 15
	case strings.Contains(m, "alias not open") || strings.Contains(m, "invalid work area"):
		return 24
	case strings.Contains(m, "no database in use"):
		return 26
	case strings.Contains(m, "number") && strings.Contains(m, "required"):
		return 27
	case strings.Contains(m, "Unrecognized") || strings.Contains(m, "not available"):
		return 36
	default:
		return 1
	}
}

func (m *Mux) handleError(ctx *context.Context, err error, retryLine string) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if !strings.HasPrefix(msg, "*** ") {
		msg = "*** " + msg
	}
	ctx.LastError = classifyError(msg)
	ctx.LastMessage = strings.TrimPrefix(msg, "*** ")
	if ctx.OnError == "" || ctx.InHandler {
		return err
	}
	stackBefore := 0
	if m.prog != nil {
		stackBefore = len(m.prog.Stack)
	}
	ctx.InHandler = true
	ctx.RetryLine = retryLine
	handler := ctx.OnError
	herr := m.Dispatch(ctx, command.Split(handler))
	stackAfter := 0
	if m.prog != nil {
		stackAfter = len(m.prog.Stack)
	}
	if stackAfter <= stackBefore {
		ctx.InHandler = false
	}
	if herr != nil {
		ctx.InHandler = false
		return herr
	}
	return nil // swallow original
}

func failCode(ctx *context.Context, code int, format string, args ...any) error {
	msg := "*** " + fmt.Sprintf(format, args...)
	fmt.Fprintln(ctx.Stderr, msg)
	ctx.LastError = code
	ctx.LastMessage = strings.TrimPrefix(msg, "*** ")
	return errors.New(msg)
}
