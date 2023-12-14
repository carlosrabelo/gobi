package script

import "github.com/carlosrabelo/gobi/gobi/pkg/command"

// LineKind classifies a parsed PRG source line.
type LineKind int

const (
	// LineEmpty marks a blank source line.
	LineEmpty LineKind = iota
	// LineRemark marks a silent comment line (asterisk or NOTE).
	LineRemark
	// LineCommand marks an executable dBase II command line.
	LineCommand
	// LineText marks a literal body line of a TEXT ... ENDTEXT block
	// (including the closing ENDTEXT), preserved verbatim and never executed.
	LineText
)

// Line is one parsed line from a command file.
type Line struct {
	Number  int
	Source  string
	Kind    LineKind
	Remark  string
	Command command.Command
	Text    []string // Literal block lines attached to a TEXT command
}

// Commands returns the executable command lines in source order.
func (p *Program) Commands() []Line {
	if p == nil {
		return nil
	}
	out := make([]Line, 0, len(p.Lines))
	for _, line := range p.Lines {
		if line.Kind == LineCommand {
			out = append(out, line)
		}
	}
	return out
}
