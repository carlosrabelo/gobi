package repl

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
	"github.com/carlosrabelo/gobi/gobi/pkg/help"
)

func init() {
	help.Register("PROCEDURE", "PROCEDURE name — start a procedure in a command file.")
	help.Register("PARAMETERS", "PARAMETERS a, b — receive DO … WITH arguments.")
	help.Register("PUBLIC", "PUBLIC name — declare a public memory variable.")
	help.Register("PRIVATE", "PRIVATE name — hide outer names in this procedure.")
}

// ProcFrame is one procedure/file call with locals.
type ProcFrame struct {
	Locals   map[string]expr.Value
	Private  map[string]bool // names hidden from outer lookup
	Publics  map[string]bool
	Params   []string
	Args     []expr.Value
	ArgNames []string // caller memory names for by-ref, empty if expression
}

func (m *Mux) ensureProg() {
	if m.prog == nil {
		m.prog = NewProgram()
	}
	if m.procFile == nil {
		m.procFile = map[string][]string{}
	}
}

// Mux fields for procedures — added via this package's extension of Mux.
// We store procedure library on Mux via existing fields where possible.
var _ = help.Register

func parseProcedures(lines []string) map[string]int {
	out := map[string]int{}
	for i, line := range lines {
		if isCommentLine(line) {
			continue
		}
		cmd := command.Split(line)
		if cmd.Verb == "PROCEDURE" {
			fields := strings.Fields(cmd.Args)
			if len(fields) == 0 {
				continue
			}
			name := strings.ToUpper(fields[0])
			out[name] = i + 1 // start after PROCEDURE line
		}
	}
	return out
}

func procEnd(lines []string, start int) int {
	for i := start; i < len(lines); i++ {
		if isCommentLine(lines[i]) {
			continue
		}
		cmd := command.Split(lines[i])
		if cmd.Verb == "PROCEDURE" {
			return i
		}
	}
	return len(lines)
}

func (m *Mux) findProcedure(ctx *context.Context, name string) (src []string, body []string, ok bool) {
	name = strings.ToUpper(name)
	if m.prog != nil {
		for i := len(m.prog.Stack) - 1; i >= 0; i-- {
			fr := m.prog.Stack[i]
			src = fr.SourceLines
			if src == nil {
				src = fr.Lines
			}
			procs := parseProcedures(src)
			if s, found := procs[name]; found {
				end := procEnd(src, s)
				return src, src[s:end], true
			}
		}
	}
	if m.procLines != nil {
		procs := parseProcedures(m.procLines)
		if s, found := procs[name]; found {
			end := procEnd(m.procLines, s)
			return m.procLines, m.procLines[s:end], true
		}
	}
	return nil, nil, false
}

func cmdPUBLIC(ctx *context.Context, cmd command.Command) error {
	names := splitNames(cmd.Args)
	if len(names) == 0 {
		return fail(ctx, "PUBLIC requires a name")
	}
	for _, n := range names {
		if _, ok := ctx.Symbols.Get(n); !ok {
			if err := ctx.Symbols.Set(n, expr.LogicalVal(false)); err != nil {
				return fail(ctx, "%s", err.Error())
			}
		}
		if ctx.Publics == nil {
			ctx.Publics = map[string]bool{}
		}
		ctx.Publics[strings.ToUpper(n)] = true
	}
	return nil
}

func cmdPRIVATE(ctx *context.Context, cmd command.Command) error {
	args := strings.TrimSpace(cmd.Args)
	up := strings.ToUpper(args)
	fields := strings.Fields(up)
	if len(fields) == 0 {
		return fail(ctx, "PRIVATE requires a name")
	}
	if ctx.Privates == nil {
		ctx.Privates = map[string]bool{}
	}
	if fields[0] == "ALL" {
		if len(fields) >= 3 && fields[1] == "LIKE" {
			for _, name := range ctx.Symbols.Names() {
				if matchLike(fields[2], name) {
					ctx.Privates[name] = true
				}
			}
			return nil
		}
		if len(fields) >= 3 && fields[1] == "EXCEPT" {
			for _, name := range ctx.Symbols.Names() {
				if !matchLike(fields[2], name) {
					ctx.Privates[name] = true
				}
			}
			return nil
		}
	}
	for _, n := range splitNames(args) {
		ctx.Privates[strings.ToUpper(n)] = true
	}
	return nil
}

func cmdPARAMETERS(ctx *context.Context, cmd command.Command) error {
	names := splitNames(cmd.Args)
	if ctx.PendingArgs == nil {
		return fail(ctx, "PARAMETERS without DO WITH")
	}
	if len(names) != len(ctx.PendingArgs) {
		return fail(ctx, "parameter count mismatch")
	}
	for i, n := range names {
		if err := ctx.Symbols.Set(n, ctx.PendingArgs[i]); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		if ctx.ParamLocals == nil {
			ctx.ParamLocals = map[string]bool{}
		}
		ctx.ParamLocals[strings.ToUpper(n)] = true
	}
	ctx.PendingArgs = nil
	return nil
}

func splitNames(s string) []string {
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, strings.Fields(p)[0])
		}
	}
	return out
}

func matchLike(pat, name string) bool {
	return strings.EqualFold(pat, name) || // simple; reuse symbols.Match
		matchStar(strings.ToUpper(pat), strings.ToUpper(name))
}

func matchStar(pat, s string) bool {
	// delegate to symbols via identical logic — import cycle avoided with copy
	for len(pat) > 0 {
		switch pat[0] {
		case '*':
			pat = pat[1:]
			if pat == "" {
				return true
			}
			for i := 0; i <= len(s); i++ {
				if matchStar(pat, s[i:]) {
					return true
				}
			}
			return false
		case '?':
			if s == "" {
				return false
			}
			pat, s = pat[1:], s[1:]
		default:
			if s == "" || s[0] != pat[0] {
				return false
			}
			pat, s = pat[1:], s[1:]
		}
	}
	return s == ""
}

func cmdSETPROCEDURE(ctx *context.Context, m *Mux, to string) error {
	m.ensureProg()
	if to == "" {
		m.procLines = nil
		m.procPath = ""
		ctx.Procedure = false
		return nil
	}
	path, err := prgPath(ctx, to)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	lines, err := loadPrg(path)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	m.procLines = lines
	m.procPath = path
	ctx.Procedure = true
	return nil
}

func cmdCLOSEPROCEDURE(ctx *context.Context, m *Mux) error {
	return cmdSETPROCEDURE(ctx, m, "")
}

// resolveDO opens a procedure or .prg for DO name [WITH …].
func (m *Mux) resolveDO(ctx *context.Context, raw string) error {
	m.ensureProg()
	raw = strings.TrimSpace(raw)
	withIdx := strings.Index(strings.ToUpper(raw), " WITH ")
	var withPart string
	name := raw
	if withIdx >= 0 {
		name = strings.TrimSpace(raw[:withIdx])
		withPart = strings.TrimSpace(raw[withIdx+6:])
	}
	if i := strings.IndexAny(name, " \t"); i >= 0 {
		name = name[:i]
	}
	if name == "" {
		return fail(ctx, "DO requires a file name")
	}

	var args []expr.Value
	if withPart != "" {
		parts := splitWithArgs(withPart)
		env := newEnv(ctx)
		for _, p := range parts {
			tree, errs := expr.Parse(p)
			if len(errs) > 0 || tree == nil {
				return fail(ctx, "invalid WITH expression")
			}
			v, err := expr.Eval(tree, env)
			if err != nil {
				return fail(ctx, "%s", err.Error())
			}
			args = append(args, v)
		}
		ctx.PendingArgs = args
	}

	if src, body, ok := m.findProcedure(ctx, name); ok {
		snapshot := snapshotLocals(ctx)
		m.prog.Stack = append(m.prog.Stack, Frame{
			Path: name, Lines: body, SourceLines: src, PC: 0, onReturn: snapshot,
		})
		if len(m.prog.Stack) == 1 {
			return runProgram(ctx, m)
		}
		return nil
	}

	path, err := prgPath(ctx, name)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if _, err := os.Stat(path); err != nil {
		ctx.PendingArgs = nil
		return fail(ctx, "%s", err.Error())
	}
	lines, err := loadPrg(path)
	if err != nil {
		ctx.PendingArgs = nil
		return fail(ctx, "%s", err.Error())
	}
	if err := validateBlocks(lines); err != nil {
		ctx.PendingArgs = nil
		return fail(ctx, "%s", err.Error())
	}
	snapshot := snapshotLocals(ctx)
	m.prog.Stack = append(m.prog.Stack, Frame{
		Path: path, Lines: lines, SourceLines: lines, PC: 0, onReturn: snapshot,
	})
	if len(m.prog.Stack) == 1 {
		return runProgram(ctx, m)
	}
	return nil
}

func splitWithArgs(s string) []string {
	var out []string
	var b strings.Builder
	depth := 0
	inQ := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQ != 0 {
			b.WriteByte(c)
			if c == inQ {
				inQ = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			inQ = c
			b.WriteByte(c)
		case '(':
			depth++
			b.WriteByte(c)
		case ')':
			depth--
			b.WriteByte(c)
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(b.String()))
				b.Reset()
			} else {
				b.WriteByte(c)
			}
		default:
			b.WriteByte(c)
		}
	}
	if t := strings.TrimSpace(b.String()); t != "" {
		out = append(out, t)
	}
	return out
}

type localSnapshot struct {
	names      []string
	privates   map[string]bool
	paramLocal map[string]bool
}

func snapshotLocals(ctx *context.Context) *localSnapshot {
	s := &localSnapshot{
		privates:   copyBoolMap(ctx.Privates),
		paramLocal: copyBoolMap(ctx.ParamLocals),
	}
	if ctx.ParamLocals != nil {
		for n := range ctx.ParamLocals {
			s.names = append(s.names, n)
		}
	}
	return s
}

func restoreLocals(ctx *context.Context, s *localSnapshot) {
	if s == nil {
		return
	}
	for _, n := range s.names {
		ctx.Symbols.Release(n)
	}
	// also drop non-public names created in procedure that were marked private
	if ctx.Privates != nil {
		for n := range ctx.Privates {
			if ctx.Publics == nil || !ctx.Publics[n] {
				ctx.Symbols.Release(n)
			}
		}
	}
	ctx.Privates = s.privates
	ctx.ParamLocals = s.paramLocal
}

func copyBoolMap(m map[string]bool) map[string]bool {
	if m == nil {
		return nil
	}
	out := make(map[string]bool, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func doReturn(ctx *context.Context, m *Mux, toMaster bool) {
	if m.prog == nil || len(m.prog.Stack) == 0 {
		return
	}
	if toMaster {
		for len(m.prog.Stack) > 0 {
			fr := m.prog.Stack[len(m.prog.Stack)-1]
			restoreLocals(ctx, fr.onReturn)
			m.prog.Stack = m.prog.Stack[:len(m.prog.Stack)-1]
		}
		return
	}
	fr := m.prog.Stack[len(m.prog.Stack)-1]
	restoreLocals(ctx, fr.onReturn)
	m.prog.Stack = m.prog.Stack[:len(m.prog.Stack)-1]
}

// unused import guard
var _ = filepath.Join
