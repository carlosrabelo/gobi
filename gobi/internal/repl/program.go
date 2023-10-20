package repl

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

// Frame is one running .prg on the call stack.
type Frame struct {
	Path  string
	Lines []string // original text lines (0-based)
	PC    int      // next line index to consider
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
	if m.prog == nil {
		m.prog = NewProgram()
	}
	name := strings.TrimSpace(cmd.Args)
	if name == "" {
		return fail(ctx, "DO requires a file name")
	}
	if i := strings.IndexAny(name, " \t"); i >= 0 {
		name = name[:i]
	}
	path, err := prgPath(ctx, name)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	lines, err := loadPrg(path)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if err := validateBlocks(lines); err != nil {
		return fail(ctx, "%s", err.Error())
	}
	m.prog.Stack = append(m.prog.Stack, Frame{Path: path, Lines: lines, PC: 0})
	if len(m.prog.Stack) == 1 {
		return runProgram(ctx, m)
	}
	return nil
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
	var ifDepth, whileDepth int
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
			}
		case "ENDDO":
			if whileDepth == 0 {
				return errString("ENDDO without DO WHILE")
			}
			whileDepth--
		}
	}
	if ifDepth != 0 {
		return errString("IF without ENDIF")
	}
	if whileDepth != 0 {
		return errString("DO WHILE without ENDDO")
	}
	return nil
}

func runProgram(ctx *context.Context, m *Mux) error {
	for len(m.prog.Stack) > 0 {
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
		cmd := command.Split(line)
		switch cmd.Verb {
		case "RETURN":
			m.prog.Stack = m.prog.Stack[:len(m.prog.Stack)-1]
			continue
		case "CANCEL":
			m.prog.Stack = nil
			return nil
		case "IF", "ELSE", "ENDIF", "ENDDO", "LOOP", "EXIT":
			if err := runControl(ctx, m, fr, cmd); err != nil {
				return err
			}
			continue
		case "DO":
			if isDoWhile(cmd) {
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
		if err := m.Dispatch(ctx, cmd); err != nil {
			return err
		}
	}
	return nil
}
