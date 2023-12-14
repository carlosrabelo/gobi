package repl

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

// Frame is one running .prg on the call stack.
type Frame struct {
	Path        string
	Lines       []string // lines being executed
	SourceLines []string // full file for PROCEDURE lookup; nil = Lines
	PC          int      // next line index to consider
	onReturn    *localSnapshot
}

// Program holds the DO call stack.
type Program struct {
	Stack []Frame
}

// NewProgram returns an empty call stack.
func NewProgram() *Program {
	return &Program{}
}

func (m *Mux) cmdDO(ctx *context.Context, cmd command.Command) error {
	return m.resolveDO(ctx, cmd.Args)
}

func prgPath(ctx *context.Context, name string) (string, error) {
	if filepath.Ext(name) == "" {
		name += ".prg"
	}
	if filepath.IsAbs(name) {
		return name, nil
	}
	base := ctx.Config.Default
	if base == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		base = wd
	}
	return filepath.Join(base, name), nil
}

func loadPrg(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, sc.Err()
}

func isCommentLine(line string) bool {
	trim := strings.TrimLeftFunc(line, unicode.IsSpace)
	if trim == "" {
		return true
	}
	if strings.HasPrefix(trim, "*") {
		return true
	}
	cmd := command.Split(trim)
	return cmd.Verb == "NOTE"
}

func validateBlocks(lines []string) error {
	var ifDepth, whileDepth, caseDepth, textDepth int
	for _, line := range lines {
		if isCommentLine(line) {
			continue
		}
		cmd := command.Split(line)
		switch cmd.Verb {
		case "IF":
			ifDepth++
		case "ELSE":
			if ifDepth == 0 {
				return errString("ELSE without IF")
			}
		case "ENDIF":
			if ifDepth == 0 {
				return errString("ENDIF without IF")
			}
			ifDepth--
		case "DO":
			if isDoWhile(cmd) {
				whileDepth++
			} else if strings.EqualFold(strings.TrimSpace(cmd.Args), "CASE") {
				caseDepth++
			}
		case "ENDDO":
			if whileDepth == 0 {
				return errString("ENDDO without DO WHILE")
			}
			whileDepth--
		case "ENDCASE":
			if caseDepth == 0 {
				return errString("ENDCASE without DO CASE")
			}
			caseDepth--
		case "TEXT":
			textDepth++
		case "ENDTEXT":
			if textDepth == 0 {
				return errString("ENDTEXT without TEXT")
			}
			textDepth--
		}
	}
	if ifDepth != 0 {
		return errString("IF without ENDIF")
	}
	if whileDepth != 0 {
		return errString("DO WHILE without ENDDO")
	}
	if caseDepth != 0 {
		return errString("DO CASE without ENDCASE")
	}
	if textDepth != 0 {
		return errString("TEXT without ENDTEXT")
	}
	return nil
}

func runProgram(ctx *context.Context, m *Mux) error {
	for len(m.prog.Stack) > 0 {
		if stop, err := checkProgramEscape(ctx, m); err != nil {
			return err
		} else if stop {
			return nil
		}
		fr := &m.prog.Stack[len(m.prog.Stack)-1]
		if fr.PC >= len(fr.Lines) {
			m.prog.Stack = m.prog.Stack[:len(m.prog.Stack)-1]
			continue
		}
		line := fr.Lines[fr.PC]
		fr.PC++
		if isCommentLine(line) {
			continue
		}
		expanded, err := expandLineMacros(ctx, line)
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		line = expanded
		if ctx.Config.Echo {
			echoProgramLine(ctx, line)
		} else if ctx.Config.Step && ctx.Config.Debug {
			fmt.Fprintln(printSink(ctx), line)
		}
		if stop, err := stepProgramLine(ctx, m); err != nil {
			return err
		} else if stop {
			return nil
		}
		cmd := command.Split(line)
		switch cmd.Verb {
		case "RETURN":
			toMaster := strings.Contains(strings.ToUpper(cmd.Args), "TO MASTER")
			doReturn(ctx, m, toMaster)
			continue
		case "CANCEL":
			m.prog.Stack = nil
			ctx.Privates = nil
			ctx.ParamLocals = nil
			ctx.PendingArgs = nil
			return nil
		case "PROCEDURE":
			// skip to next PROCEDURE or end when falling through main
			fr.PC = procEnd(fr.Lines, fr.PC)
			continue
		case "IF", "ELSE", "ENDIF", "ENDDO", "LOOP", "EXIT", "CASE", "OTHERWISE", "ENDCASE", "TEXT", "ENDTEXT":
			if err := runControl(ctx, m, fr, cmd); err != nil {
				return err
			}
			continue
		case "DO":
			if isDoWhile(cmd) || strings.EqualFold(strings.TrimSpace(cmd.Args), "CASE") {
				if err := runControl(ctx, m, fr, cmd); err != nil {
					return err
				}
				continue
			}
			if err := m.cmdDO(ctx, cmd); err != nil {
				return err
			}
			continue
		}
		if err := m.dispatchLine(ctx, cmd, line); err != nil {
			if err == errSuspend {
				return nil
			}
			return err
		}
	}
	return nil
}

// echoProgramLine prints a command-file line when SET ECHO is on (023.3).
// SET DEBUG ON sends ECHO/STEP lines to the print sink (023.12).
func echoProgramLine(ctx *context.Context, line string) {
	w := ctx.Out()
	if ctx.Config.Debug {
		w = printSink(ctx)
	}
	fmt.Fprintln(w, line)
}

// stepProgramLine waits for a key when SET STEP is on (023.4).
// Esc with SET ESCAPE on runs ON ESCAPE or cancels like CANCEL (031.3).
func stepProgramLine(ctx *context.Context, m *Mux) (stop bool, err error) {
	if !ctx.Config.Step {
		return false, nil
	}
	keys := ctx.Keys
	if keys == nil {
		keys = term.NewReader(ctx.Stdin)
		ctx.Keys = keys
	}
	k, err := keys.ReadKey()
	if err != nil {
		return false, fail(ctx, "%s", err.Error())
	}
	if k == term.KeyEsc && ctx.Config.Escape {
		return handleProgramEscape(ctx, m)
	}
	return false, nil
}

// checkProgramEscape looks for a waiting Esc between command-file lines (031.3).
// When SET STEP is on, Esc is handled only in stepProgramLine.
func checkProgramEscape(ctx *context.Context, m *Mux) (stop bool, err error) {
	if !ctx.Config.Escape || ctx.Config.Step || m.prog == nil || len(m.prog.Stack) == 0 {
		return false, nil
	}
	keys := ctx.Keys
	if keys == nil {
		keys = term.NewReader(ctx.Stdin)
		ctx.Keys = keys
	}
	k := keys.Inkey()
	if k == 0 {
		return false, nil
	}
	if k == term.KeyEsc {
		return handleProgramEscape(ctx, m)
	}
	// Non-Esc key stays available for ON KEY / typeahead (031.4 / 033).
	if k > 0 && k < 256 {
		ctx.TypeaheadBuf = append(ctx.TypeaheadBuf, byte(k))
	}
	return false, nil
}

func handleProgramEscape(ctx *context.Context, m *Mux) (stop bool, err error) {
	if ctx.OnEscape != "" {
		ctx.InHandler = true
		herr := m.Dispatch(ctx, command.Split(ctx.OnEscape))
		ctx.InHandler = false
		if herr != nil {
			return false, herr
		}
		return false, nil
	}
	m.prog.Stack = nil
	ctx.Privates = nil
	ctx.ParamLocals = nil
	ctx.PendingArgs = nil
	return true, nil
}
