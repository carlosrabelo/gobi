package repl

import (
	"fmt"
	"io"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/internal/symbols"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

// handleInput implements INPUT ['prompt'] TO <memvar>, reading a line from
// the console, evaluating it as a dBase II expression, and storing the
// resulting value (numeric, logical, or string) in a memory variable.
func handleInput(ctx *context.Context, cmd Command) error {
	target := strings.ToUpper(strings.TrimSpace(cmd.ToClause))
	if target == "" {
		return fmt.Errorf("*** INPUT requires TO <variable>")
	}
	if err := symbols.ValidateName(target); err != nil {
		return fmt.Errorf("*** Invalid INPUT target %s: %v", target, err)
	}

	prompt, err := parseConsolePrompt("INPUT", cmd.Args)
	if err != nil {
		return err
	}

	line, err := readAppendLine(ctx, ctx.StdinReader(), consolePromptText(prompt))
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return fmt.Errorf("*** Error reading INPUT input: %w", err)
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return fmt.Errorf("*** INPUT requires an expression")
	}

	exp, err := parseReplExpression(line)
	if err != nil {
		return err
	}
	if exp == nil {
		return fmt.Errorf("*** INPUT requires an expression")
	}

	env := newReplEnvironment(ctx)
	obj, err := expr.Eval(exp, env)
	if err != nil {
		return fmt.Errorf("*** Evaluation error in INPUT expression: %w", err)
	}

	value, err := objectToStoredValue(obj)
	if err != nil {
		return fmt.Errorf("*** INPUT expression returned unsupported type %T", obj)
	}

	if err := ctx.Variables.Set(target, value); err != nil {
		return fmt.Errorf("*** Error storing %s: %v", target, err)
	}
	return nil
}
