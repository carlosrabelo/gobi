package repl

import (
	"fmt"
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
		end, err := findEndFromInside(fr.Lines, fr.PC-1, "IF", "ENDIF")
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		fr.PC = end + 1
		return nil
	case "ENDIF":
		return nil
	case "DO":
		if strings.EqualFold(strings.TrimSpace(cmd.Args), "CASE") {
			return runCASE(ctx, fr)
		}
		return runWHILE(ctx, fr, cmd)
	case "ENDDO":
		return handleENDDO(ctx, fr)
	case "LOOP":
		return handleLOOP(ctx, fr)
	case "EXIT":
		return handleEXIT(ctx, fr)
	case "CASE", "OTHERWISE":
		// Skip to ENDCASE when falling through a taken branch's end.
		end, err := findEndFromInside(fr.Lines, fr.PC-1, "DO", "ENDCASE")
		if err != nil {
			// DO CASE uses Verb DO — find matching ENDCASE by scanning
			end, err = findEndCASE(fr.Lines, fr.PC-1)
		}
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		fr.PC = end + 1
		return nil
	case "ENDCASE":
		return nil
	case "TEXT":
		return runTEXT(ctx, fr)
	case "ENDTEXT":
		return nil
	default:
		return fail(ctx, "unknown control %s", cmd.Verb)
	}
}

func runTEXT(ctx *context.Context, fr *Frame) error {
	for fr.PC < len(fr.Lines) {
		line := fr.Lines[fr.PC]
		fr.PC++
		c := command.Split(line)
		if c.Verb == "ENDTEXT" {
			return nil
		}
		fmt.Fprintln(ctx.Stdout, line)
	}
	return fail(ctx, "TEXT without ENDTEXT")
}

func runCASE(ctx *context.Context, fr *Frame) error {
	caseLine := fr.PC - 1
	endLine, err := findEndCASE(fr.Lines, caseLine)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	i := caseLine + 1
	matched := false
	for i < endLine {
		if isCommentLine(fr.Lines[i]) {
			i++
			continue
		}
		c := command.Split(fr.Lines[i])
		switch c.Verb {
		case "CASE":
			if matched {
				fr.PC = endLine + 1
				return nil
			}
			ok, err := evalCond(ctx, c.Args)
			if err != nil {
				return fail(ctx, "%s", err.Error())
			}
			i++
			if ok {
				matched = true
				fr.PC = i
				return nil
			}
			// skip body until next CASE/OTHERWISE/ENDCASE
			i = skipCaseBody(fr.Lines, i, endLine)
		case "OTHERWISE":
			if matched {
				fr.PC = endLine + 1
				return nil
			}
			fr.PC = i + 1
			return nil
		case "ENDCASE":
			fr.PC = endLine + 1
			return nil
		default:
			i++
		}
	}
	fr.PC = endLine + 1
	return nil
}

func skipCaseBody(lines []string, from, endLine int) int {
	depth := 0
	for i := from; i < endLine; i++ {
		if isCommentLine(lines[i]) {
			continue
		}
		c := command.Split(lines[i])
		if c.Verb == "DO" && strings.EqualFold(strings.TrimSpace(c.Args), "CASE") {
			depth++
		}
		if c.Verb == "ENDCASE" {
			if depth == 0 {
				return i
			}
			depth--
		}
		if depth == 0 && (c.Verb == "CASE" || c.Verb == "OTHERWISE") {
			return i
		}
	}
	return endLine
}

func findEndCASE(lines []string, startIdx int) (int, error) {
	depth := 0
	for i := startIdx; i < len(lines); i++ {
		if isCommentLine(lines[i]) {
			continue
		}
		c := command.Split(lines[i])
		if c.Verb == "DO" && strings.EqualFold(strings.TrimSpace(c.Args), "CASE") {
			depth++
		}
		if c.Verb == "ENDCASE" {
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return 0, errString("DO CASE without ENDCASE")
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
			if startVerb == "DO" && !isDoWhile(c) {
				continue
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
			if startVerb == "DO" && !isDoWhile(c) {
				continue
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

func isDoWhile(cmd command.Command) bool {
	if cmd.Verb != "DO" {
		return false
	}
	if _, ok := cmd.Clauses["WHILE"]; ok {
		return true
	}
	return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(cmd.Args)), "WHILE")
}

func whileCond(cmd command.Command) string {
	if w := strings.TrimSpace(cmd.Clauses["WHILE"]); w != "" {
		return w
	}
	args := strings.TrimSpace(cmd.Args)
	if strings.HasPrefix(strings.ToUpper(args), "WHILE") {
		return strings.TrimSpace(args[len("WHILE"):])
	}
	return ""
}

func runWHILE(ctx *context.Context, fr *Frame, cmd command.Command) error {
	cond := whileCond(cmd)
	ok, err := evalCond(ctx, cond)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	whileLine := fr.PC - 1
	endLine, err := findMatching(fr.Lines, whileLine, "DO", "ENDDO")
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if !ok {
		fr.PC = endLine + 1
		return nil
	}
	// Enter body; on ENDDO we jump back (handled in runControl).
	return nil
}

func handleENDDO(ctx *context.Context, fr *Frame) error {
	whileLine, err := findOpenWHILE(fr.Lines, fr.PC-1)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	fr.PC = whileLine // re-evaluate DO WHILE
	return nil
}

func handleLOOP(ctx *context.Context, fr *Frame) error {
	whileLine, err := findOpenWHILE(fr.Lines, fr.PC-1)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	fr.PC = whileLine
	return nil
}

func handleEXIT(ctx *context.Context, fr *Frame) error {
	whileLine, err := findOpenWHILE(fr.Lines, fr.PC-1)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	endLine, err := findMatching(fr.Lines, whileLine, "DO", "ENDDO")
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	fr.PC = endLine + 1
	return nil
}

// findOpenWHILE walks backward to the innermost DO WHILE that contains fromIdx.
// fromIdx is the LOOP, EXIT, or ENDDO line; scanning starts at the line above it.
func findOpenWHILE(lines []string, fromIdx int) (int, error) {
	depth := 0
	for i := fromIdx - 1; i >= 0; i-- {
		if isCommentLine(lines[i]) {
			continue
		}
		c := command.Split(lines[i])
		switch c.Verb {
		case "ENDDO":
			depth++
		case "DO":
			if isDoWhile(c) {
				if depth == 0 {
					return i, nil
				}
				depth--
			}
		}
	}
	return 0, errString("LOOP/EXIT/ENDDO without DO WHILE")
}
