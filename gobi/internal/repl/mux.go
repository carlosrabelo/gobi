package repl

import (
	"errors"
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

// ErrQuit is returned by QUIT and ends the prompt loop without failure.
var ErrQuit = errors.New("quit")

// Handler runs one registered verb.
type Handler func(ctx *context.Context, cmd command.Command) error

// Mux maps upper-case verbs to handlers.
type Mux struct {
	handlers map[string]Handler
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
	m.Register("CLOSE", cmdCLOSE)
	m.Register("SELECT", cmdSELECT)
	m.Register("GOTO", cmdGOTO)
	m.Register("GO", cmdGO)
	m.Register("SKIP", cmdSKIP)
	m.Register("DISPLAY", cmdDISPLAY)
	m.Register("LIST", cmdLIST)
	m.Register("?", cmdQuestion)
	m.Register("??", cmdQuestionQuestion)
	m.Register("CREATE", cmdCREATE)
	m.Register("APPEND", cmdAPPEND)
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
	m.Register("LOCATE", cmdLOCATE)
	m.Register("CONTINUE", cmdCONTINUE)
	m.Register("COUNT", cmdCOUNT)
	m.Register("SUM", cmdSUM)
	m.Register("AVERAGE", cmdAVERAGE)
	return m
}

// Register binds verb to h. A second register replaces the handler.
func (m *Mux) Register(verb string, h Handler) {
	m.handlers[strings.ToUpper(verb)] = h
}

// Dispatch runs the handler for cmd.Verb.
func (m *Mux) Dispatch(ctx *context.Context, cmd command.Command) error {
	if cmd.Verb == "" {
		return nil
	}
	h, ok := m.handlers[cmd.Verb]
	if !ok {
		msg := "*** Unrecognized command: " + cmd.Verb
		fmt.Fprintln(ctx.Stderr, msg)
		return errors.New(msg)
	}
	return h(ctx, cmd)
}
