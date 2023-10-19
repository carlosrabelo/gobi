package repl

import (
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func runControl(ctx *context.Context, m *Mux, fr *Frame, cmd command.Command) error {
	switch cmd.Verb {
	case "IF":
		return runIF(ctx, fr, cmd)
	case "ELSE":
		// Jump past the matching ENDIF (we are inside a true IF branch).
		end, err := findEndFromInside(fr.Lines, fr.PC-1, "IF", "ENDIF")
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		fr.PC = end + 1
		return nil
	case "ENDIF":
		return nil
	case "DO":
		return runWHILE(ctx, fr, cmd)
	case "ENDDO":
		return fail(ctx, "ENDDO without DO WHILE")
	case "LOOP", "EXIT":
		return fail(ctx, "%s outside DO WHILE", cmd.Verb)
	default:
		return fail(ctx, "unknown control %s", cmd.Verb)
	}
}

func runIF(ctx *context.Context, fr *Frame, cmd command.Command) error {
	ok, err := evalCond(ctx, cmd.Args)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	ifLine := fr.PC - 1
	elseLine, hasElse, err := findElse(fr.Lines, ifLine)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	endLine, err := findMatching(fr.Lines, ifLine, "IF", "ENDIF")
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if ok {
		// Continue into true branch; PC already past IF.
		return nil
	}
	if hasElse {
		fr.PC = elseLine + 1
		return nil
	}
	fr.PC = endLine + 1
	return nil
}

func evalCond(ctx *context.Context, src string) (bool, error) {
	src = strings.TrimSpace(src)
	tree, errs := expr.Parse(src)
	if len(errs) > 0 || tree == nil {
		msg := "invalid condition"
		if len(errs) > 0 {
			msg = errs[0]
		}
		return false, errString(msg)
	}
	v, err := expr.Eval(tree, newEnv(ctx))
	if err != nil {
		return false, err
	}
	if v.Kind != expr.KindLogical {
		return false, errString("condition must be logical")
	}
	return v.Log, nil
}

// findMatching returns the line index of the matching end verb for start at startIdx.
func findMatching(lines []string, startIdx int, startVerb, endVerb string) (int, error) {
	depth := 0
	for i := startIdx; i < len(lines); i++ {
		if isCommentLine(lines[i]) {
			continue
		}
		c := command.Split(lines[i])
		switch c.Verb {
		case startVerb:
			if startVerb == "DO" {
				if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(c.Args)), "WHILE") {
					continue
				}
			}
			depth++
		case endVerb:
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return 0, errString(startVerb + " without " + endVerb)
}

// findEndFromInside finds endVerb while already inside one startVerb (depth starts at 1).
func findEndFromInside(lines []string, fromIdx int, startVerb, endVerb string) (int, error) {
	depth := 1
	for i := fromIdx + 1; i < len(lines); i++ {
		if isCommentLine(lines[i]) {
			continue
		}
		c := command.Split(lines[i])
		switch c.Verb {
		case startVerb:
			if startVerb == "DO" {
				if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(c.Args)), "WHILE") {
					continue
				}
			}
			depth++
		case endVerb:
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return 0, errString(startVerb + " without " + endVerb)
}

func findElse(lines []string, ifIdx int) (elseIdx int, found bool, err error) {
	depth := 0
	for i := ifIdx; i < len(lines); i++ {
		if isCommentLine(lines[i]) {
			continue
		}
		c := command.Split(lines[i])
		switch c.Verb {
		case "IF":
			depth++
		case "ENDIF":
			depth--
			if depth == 0 {
				return 0, false, nil
			}
		case "ELSE":
			if depth == 1 {
				return i, true, nil
			}
		}
	}
	return 0, false, errString("IF without ENDIF")
}

func runWHILE(ctx *context.Context, fr *Frame, cmd command.Command) error {
	_ = ctx
	_ = fr
	_ = cmd
	return fail(ctx, "DO WHILE is not supported yet")
}
