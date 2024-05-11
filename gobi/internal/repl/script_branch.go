package repl

import (
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
	"github.com/carlosrabelo/gobi/gobi/pkg/script"
)

func evalLogicalExpression(ctx *context.Context, expression, label string) (bool, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return false, fmt.Errorf("*** %s requires a logical expression", label)
	}

	lexer := expr.NewLexer(expression)
	parser := expr.NewParser(lexer)
	exp := parser.ParseExpression()
	if len(parser.Errors()) > 0 {
		return false, fmt.Errorf("*** Syntax error in %s expression: %s", label, strings.Join(parser.Errors(), "; "))
	}
	if exp == nil {
		return false, fmt.Errorf("*** %s requires a logical expression", label)
	}

	res, err := expr.Eval(exp, newReplEnvironment(ctx))
	if err != nil {
		return false, fmt.Errorf("*** Evaluation error in %s expression: %w", label, err)
	}

	boolRes, ok := res.(*expr.BooleanObject)
	if !ok {
		return false, fmt.Errorf("*** %s expression must evaluate to .T. or .F.", label)
	}

	return boolRes.Value, nil
}

func finishAdvance(ctrl *script.Controller) bool {
	if ctrl.Advance() {
		return false
	}
	return ctrl.Depth() <= 1
}

// executeDoCase selects the DO CASE branch to run: the first CASE whose
// expression is true, the OTHERWISE branch when no CASE matches, or the
// command following ENDCASE when there is no branch to take. Commands between
// DO CASE and the first CASE are never executed.
func executeDoCase(ctx *context.Context, ctrl *script.Controller, prog *script.Program, line script.Line) (stop bool, err error) {
	block, ok := prog.CaseBlockAt(ctrl.Index())
	if !ok {
		return false, fmt.Errorf("*** Error in %s, line %d: unmatched DO CASE block", prog.Path, line.Number)
	}

	cmds := prog.Commands()
	target := block.EndIndex + 1
	matched := false
	for _, caseIdx := range block.CaseIndexes {
		caseLine := cmds[caseIdx]
		truthy, err := evalLogicalExpression(ctx, caseLine.Command.Args, "CASE")
		if err != nil {
			return false, fmt.Errorf("*** Error in %s, line %d: %w", prog.Path, caseLine.Number, err)
		}
		if truthy {
			target = caseIdx + 1
			matched = true
			break
		}
	}
	if !matched && block.OtherwiseIndex >= 0 {
		target = block.OtherwiseIndex + 1
	}

	if err := ctrl.SetIndex(target); err != nil {
		return false, err
	}
	return false, nil
}

func executeScriptLine(ctx *context.Context, ctrl *script.Controller, line script.Line) (stop bool, err error) {
	prog := ctrl.Program()
	if prog == nil {
		return false, fmt.Errorf("*** script: no active program")
	}

	switch line.Command.Verb {
	case "RETURN":
		if ctrl.Depth() <= 1 {
			return true, nil
		}
		popCallerFrame(ctx, ctrl)
		return false, nil
	case "CANCEL":
		return true, nil
	case "DO":
		if script.IsDoCase(line.Command) {
			return executeDoCase(ctx, ctrl, prog, line)
		}
		if line.Command.WhileClause != "" {
			block, ok := prog.WhileBlockAt(ctrl.Index())
			if !ok {
				return false, fmt.Errorf("*** Error in %s, line %d: unmatched DO WHILE block", prog.Path, line.Number)
			}

			truthy, err := evalLogicalExpression(ctx, line.Command.WhileClause, "DO WHILE")
			if err != nil {
				return false, fmt.Errorf("*** Error in %s, line %d: %w", prog.Path, line.Number, err)
			}

			if truthy {
				if finishAdvance(ctrl) {
					return true, nil
				}
			} else if err := ctrl.SetIndex(block.EndIndex + 1); err != nil {
				return false, err
			}
			return false, nil
		}

		filename := strings.TrimSpace(line.Command.Args)
		if filename == "" {
			return false, fmt.Errorf("*** Error in %s, line %d: DO requires a command file name", prog.Path, line.Number)
		}
		child, err := loadScript(ctx, filename)
		if err != nil {
			return false, fmt.Errorf("*** Error in %s, line %d: %w", prog.Path, line.Number, err)
		}
		if err := ctrl.PushFrame(child); err != nil {
			return false, fmt.Errorf("*** Error in %s, line %d: %w", prog.Path, line.Number, err)
		}
		pushScriptFrame(ctx, child.Path)
		return false, nil
	case "ENDDO":
		block, ok := prog.WhileBlockForEnd(ctrl.Index())
		if !ok {
			return false, fmt.Errorf("*** Error in %s, line %d: ENDDO without matching DO WHILE", prog.Path, line.Number)
		}

		doLine := prog.Commands()[block.DoIndex]
		truthy, err := evalLogicalExpression(ctx, doLine.Command.WhileClause, "DO WHILE")
		if err != nil {
			return false, fmt.Errorf("*** Error in %s, line %d: %w", prog.Path, line.Number, err)
		}

		if truthy {
			if err := ctrl.SetIndex(block.StartIndex); err != nil {
				return false, err
			}
		} else if finishAdvance(ctrl) {
			return true, nil
		}
		return false, nil
	case "LOOP":
		block, ok := prog.WhileEnclosingAt(ctrl.Index())
		if !ok {
			return false, fmt.Errorf("*** Error in %s, line %d: LOOP outside DO WHILE", prog.Path, line.Number)
		}
		if err := ctrl.SetIndex(block.DoIndex); err != nil {
			return false, err
		}
		return false, nil
	case "EXIT":
		block, ok := prog.WhileEnclosingAt(ctrl.Index())
		if !ok {
			return false, fmt.Errorf("*** Error in %s, line %d: EXIT outside DO WHILE", prog.Path, line.Number)
		}
		if err := ctrl.SetIndex(block.EndIndex + 1); err != nil {
			return false, err
		}
		return false, nil
	case "IF":
		block, ok := prog.IfBlockAt(ctrl.Index())
		if !ok {
			return false, fmt.Errorf("*** Error in %s, line %d: unmatched IF block", prog.Path, line.Number)
		}

		truthy, err := evalLogicalExpression(ctx, line.Command.Args, "IF")
		if err != nil {
			return false, fmt.Errorf("*** Error in %s, line %d: %w", prog.Path, line.Number, err)
		}

		if truthy {
			if finishAdvance(ctrl) {
				return true, nil
			}
		} else if block.ElseIndex >= 0 {
			if err := ctrl.SetIndex(block.ElseIndex + 1); err != nil {
				return false, err
			}
		} else if err := ctrl.SetIndex(block.EndIndex + 1); err != nil {
			return false, err
		}
		return false, nil
	case "ELSE":
		skip, ok := prog.SkipAfterElse(ctrl.Index())
		if !ok {
			return false, fmt.Errorf("*** Error in %s, line %d: ELSE without matching IF", prog.Path, line.Number)
		}
		if err := ctrl.SetIndex(skip); err != nil {
			return false, err
		}
		return false, nil
	case "ENDIF":
		if finishAdvance(ctrl) {
			return true, nil
		}
		return false, nil
	case "CASE", "OTHERWISE":
		// Reached by falling through after an executed branch body:
		// the rest of the DO CASE structure is skipped.
		skip, ok := prog.SkipAfterCaseBranch(ctrl.Index())
		if !ok {
			return false, fmt.Errorf("*** Error in %s, line %d: %s without matching DO CASE", prog.Path, line.Number, line.Command.Verb)
		}
		if err := ctrl.SetIndex(skip); err != nil {
			return false, err
		}
		return false, nil
	case "ENDCASE":
		if finishAdvance(ctrl) {
			return true, nil
		}
		return false, nil
	case "TEXT":
		for _, text := range line.Text {
			fmt.Fprintf(ctx.Stdout, "%s\r\n", text)
		}
		if finishAdvance(ctrl) {
			return true, nil
		}
		return false, nil
	}

	if err := scriptDispatch(ctx, line.Command); err != nil {
		return false, fmt.Errorf("*** Error in %s, line %d: %w", prog.Path, line.Number, err)
	}

	if finishAdvance(ctrl) {
		return true, nil
	}

	return false, nil
}
