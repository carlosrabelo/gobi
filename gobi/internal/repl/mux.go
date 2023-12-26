package repl

import (
	"errors"
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/help"
)

// ErrQuit is returned by QUIT and ends the prompt loop without failure.
var ErrQuit = errors.New("quit")

// Handler runs one registered verb.
type Handler func(ctx *context.Context, cmd command.Command) error

// Mux maps upper-case verbs to handlers.
type Mux struct {
	handlers  map[string]Handler
	prog      *Program
	gets      []PendingGET
	procLines []string
	procPath  string
	procFile  map[string][]string
	hist      *History
}

// NewMux returns a mux with the built-in verbs registered.
func NewMux() *Mux {
	m := &Mux{handlers: map[string]Handler{}}
	m.Register("QUIT", func(*context.Context, command.Command) error {
		return ErrQuit
	})
	m.Register("*", func(*context.Context, command.Command) error { return nil })
	m.Register("NOTE", func(*context.Context, command.Command) error { return nil })
	m.Register("USE", cmdUSE)
	m.Register("CLOSE", func(ctx *context.Context, cmd command.Command) error {
		return m.cmdCLOSE(ctx, cmd)
	})
	m.Register("SELECT", cmdSELECT)
	m.Register("GOTO", cmdGOTO)
	m.Register("GO", cmdGO)
	m.Register("SKIP", cmdSKIP)
	m.Register("DISPLAY", func(ctx *context.Context, cmd command.Command) error {
		return m.cmdDISPLAY(ctx, cmd)
	})
	m.Register("LIST", func(ctx *context.Context, cmd command.Command) error {
		return m.cmdLIST(ctx, cmd)
	})
	m.Register("INSERT", func(ctx *context.Context, cmd command.Command) error {
		return m.cmdINSERT(ctx, cmd)
	})
	m.Register("CLEAR", func(ctx *context.Context, cmd command.Command) error {
		return m.cmdCLEAR(ctx, cmd)
	})
	m.Register("ERASE", cmdERASE)
	m.Register("CHANGE", cmdCHANGE)
	m.Register("?", cmdQuestion)
	m.Register("??", cmdQuestionQuestion)
	m.Register("CREATE", cmdCREATE)
	m.Register("APPEND", func(ctx *context.Context, cmd command.Command) error {
		return m.cmdAPPEND(ctx, cmd)
	})
	m.Register("EDIT", func(ctx *context.Context, cmd command.Command) error {
		return m.cmdEDIT(ctx, cmd)
	})
	m.Register("READ", func(ctx *context.Context, cmd command.Command) error {
		return m.cmdREADWithFormat(ctx, cmd)
	})
	m.Register("BROWSE", cmdBROWSE)
	m.Register("RUN", cmdRUN)
	m.Register("!", cmdBang)
	m.Register("SET", func(ctx *context.Context, cmd command.Command) error {
		return m.cmdSET(ctx, cmd)
	})
	m.Register("DIR", cmdDIR)
	m.Register("RENAME", cmdRENAME)
	m.Register("HELP", func(ctx *context.Context, cmd command.Command) error {
		return m.cmdHELP(ctx, cmd)
	})
	m.Register("ACCEPT", cmdACCEPT)
	m.Register("INPUT", cmdINPUT)
	m.Register("WAIT", cmdWAIT)
	m.Register("REMARK", cmdREMARK)
	m.Register("REPLACE", cmdREPLACE)
	m.Register("DELETE", cmdDELETE)
	m.Register("RECALL", cmdRECALL)
	m.Register("PACK", cmdPACK)
	m.Register("ZAP", cmdZAP)
	m.Register("COPY", cmdCOPY)
	m.Register("UPDATE", cmdUPDATE)
	m.Register("JOIN", cmdJOIN)
	m.Register("TOTAL", cmdTOTAL)
	m.Register("MODIFY", cmdMODIFY)
	m.Register("REPORT", cmdREPORT)
	m.Register("EJECT", cmdEJECT)
	m.Register("LOCATE", cmdLOCATE)
	m.Register("CONTINUE", cmdCONTINUE)
	m.Register("COUNT", cmdCOUNT)
	m.Register("SUM", cmdSUM)
	m.Register("AVERAGE", cmdAVERAGE)
	m.Register("STORE", cmdSTORE)
	m.Register("RELEASE", cmdRELEASE)
	m.Register("SAVE", cmdSAVE)
	m.Register("RESTORE", cmdRESTORE)
	m.Register("INDEX", cmdINDEX)
	m.Register("REINDEX", cmdREINDEX)
	m.Register("FIND", cmdFIND)
	m.Register("SEEK", cmdSEEK)
	m.Register("SORT", cmdSORT)
	m.Register("@", func(ctx *context.Context, cmd command.Command) error {
		return m.cmdAt(ctx, cmd)
	})
	m.Register("DO", func(ctx *context.Context, cmd command.Command) error {
		return m.cmdDO(ctx, cmd)
	})
	m.Register("RETURN", func(ctx *context.Context, cmd command.Command) error {
		if m.prog == nil || len(m.prog.Stack) == 0 {
			return fail(ctx, "RETURN outside a program")
		}
		toMaster := strings.Contains(strings.ToUpper(cmd.Args), "TO MASTER")
		doReturn(ctx, m, toMaster)
		return nil
	})
	m.Register("CANCEL", func(ctx *context.Context, cmd command.Command) error {
		if m.prog != nil {
			m.prog.Stack = nil
		}
		ctx.Privates = nil
		ctx.ParamLocals = nil
		ctx.PendingArgs = nil
		return nil
	})
	m.Register("PUBLIC", cmdPUBLIC)
	m.Register("PRIVATE", cmdPRIVATE)
	m.Register("PARAMETERS", cmdPARAMETERS)
	m.Register("PROCEDURE", func(*context.Context, command.Command) error { return nil })
	m.Register("ON", cmdON)
	m.Register("RETRY", func(ctx *context.Context, cmd command.Command) error {
		return cmdRETRY(ctx, m, cmd)
	})
	m.Register("SUSPEND", func(ctx *context.Context, cmd command.Command) error {
		return cmdSUSPEND(ctx, m, cmd)
	})
	m.Register("RESUME", func(ctx *context.Context, cmd command.Command) error {
		return cmdRESUME(ctx, m, cmd)
	})
	m.Register("TYPE", cmdTYPE)
	m.Register("LABEL", cmdLABEL)
	registerRefused(m)
	m.Register("ASSIST", func(ctx *context.Context, cmd command.Command) error {
		return m.cmdASSIST(ctx, cmd)
	})
	return m
}

// Register binds verb to h. A second register replaces the handler.
func (m *Mux) Register(verb string, h Handler) {
	m.handlers[strings.ToUpper(verb)] = h
}

// Dispatch runs the handler for cmd.Verb.
func (m *Mux) Dispatch(ctx *context.Context, cmd command.Command) error {
	return m.dispatchLine(ctx, cmd, "")
}

func (m *Mux) dispatchLine(ctx *context.Context, cmd command.Command, retryLine string) error {
	if cmd.Verb == "" {
		return nil
	}
	if cmd.Verb == "DISPLAY" && isMemoryList(cmd) {
		return cmdDisplayMemory(ctx, cmd)
	}
	if cmd.Verb == "LIST" && isMemoryList(cmd) {
		return cmdDisplayMemory(ctx, cmd)
	}
	h, ok := m.handlers[cmd.Verb]
	if !ok {
		if handled, err := tryAssignment(ctx, cmd); handled {
			if err != nil {
				return m.handleError(ctx, err, retryLine)
			}
			return nil
		}
		msg := "*** Unrecognized command: " + cmd.Verb
		fmt.Fprintln(ctx.Stderr, msg)
		err := errors.New(msg)
		ctx.LastError = 36
		ctx.LastMessage = strings.TrimPrefix(msg, "*** ")
		if ctx.OnError != "" && !ctx.InHandler {
			return m.handleError(ctx, err, retryLine)
		}
		if m.shouldAskHelp(ctx) {
			askHelp(ctx, cmd.Verb)
		}
		return err
	}
	err := h(ctx, cmd)
	if err != nil {
		return m.handleError(ctx, err, retryLine)
	}
	if ctx.OnKey != "" && !ctx.InHandler && ctx.Keys != nil {
		if k := ctx.Keys.Inkey(); k != 0 {
			// put back conceptually by leaving in typeahead later; run handler once
			_ = m.Dispatch(ctx, command.Split(ctx.OnKey))
		}
	}
	return nil
}

func (m *Mux) shouldAskHelp(ctx *context.Context) bool {
	if !ctx.Config.Help || ctx.OnError != "" {
		return false
	}
	if m.prog != nil && len(m.prog.Stack) > 0 {
		return false
	}
	return true
}

func askHelp(ctx *context.Context, verb string) {
	fmt.Fprint(ctx.Out(), "Do you want some help? (Y/N) ")
	line, err := ctx.StdinReader().ReadString('\n')
	if err != nil {
		return
	}
	line = strings.TrimSpace(line)
	if line != "" && (line[0] == 'Y' || line[0] == 'y') {
		text, ok := help.Text(verb)
		if ok {
			fmt.Fprintln(ctx.Out(), text)
		}
	}
}
