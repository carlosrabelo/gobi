package repl

import (
	"fmt"
	"sort"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

// HandlerFunc executes a parsed command in the given context.
type HandlerFunc func(ctx *context.Context, cmd Command) error

// CommandMux routes parsed commands to registered handlers,
// supporting dBase II abbreviation rules (minimum 4 chars).
type CommandMux struct {
	handlers map[string]HandlerFunc
}

// NewCommandMux creates a mux with known commands registered.
func NewCommandMux() *CommandMux {
	m := &CommandMux{handlers: make(map[string]HandlerFunc)}
	m.registerAll()
	return m
}

// Dispatch routes a parsed command to its handler.
func (m *CommandMux) Dispatch(ctx *context.Context, cmd Command) error {
	if h, ok := m.handlers[cmd.Verb]; ok {
		return h(ctx, cmd)
	}

	verb, ok := resolveVerb(cmd.Verb, m.handlers)
	if !ok {
		return fmt.Errorf("*** Unrecognized command verb")
	}
	return m.handlers[verb](ctx, cmd)
}

// Commands returns the sorted list of registered verbs.
func (m *CommandMux) Commands() []string {
	verbs := make([]string, 0, len(m.handlers))
	for v := range m.handlers {
		verbs = append(verbs, v)
	}
	sort.Strings(verbs)
	return verbs
}

func resolveVerb(input string, handlers map[string]HandlerFunc) (string, bool) {
	input = strings.ToUpper(input)
	var match string
	for verb := range handlers {
		if strings.HasPrefix(verb, input) {
			if match != "" {
				return "", false
			}
			match = verb
		}
	}
	if match == "" {
		return "", false
	}
	return match, true
}

func (m *CommandMux) register(verb string, handler HandlerFunc) {
	m.handlers[strings.ToUpper(verb)] = handler
}

func (m *CommandMux) registerAll() {
	m.register("QUIT", handleQuit)
	m.register("USE", handleUse)
	m.register("SELECT", handleSelect)
	m.register("CLOSE", handleClose)
	m.register("DISPLAY", handleDisplay)
	m.register("LIST", handleList)
	m.register("GOTO", handleGoto)
	m.register("GO", handleGo)
	m.register("SKIP", handleSkip)
	m.register("APPEND", handleAppend)
	m.register("REPLACE", handleReplace)
	m.register("DELETE", handleDelete)
	m.register("RECALL", handleRecall)
	m.register("PACK", handlePack)
	m.register("ZAP", handleZap)
	m.register("CREATE", handleCreate)
	m.register("EDIT", handleEdit)
	m.register("MODIFY", handleModify)
	m.register("COPY", handleCopy)
	m.register("UPDATE", handleUpdate)
	m.register("JOIN", handleJoin)
	m.register("TOTAL", handleTotal)
	m.register("LOCATE", handleLocate)
	m.register("CONTINUE", handleContinue)
	m.register("COUNT", handleCount)
	m.register("SUM", handleSum)
	m.register("AVERAGE", handleAverage)
	m.register("?", handleQuestion)
	m.register("??", handleQuestion)
	m.register("STORE", handleStore)
	m.register("SAVE", handleSave)
	m.register("RESTORE", handleRestore)
	m.register("RELEASE", handleRelease)
	m.register("DO", handleDo)
}

func stubHandler(verb string) HandlerFunc {
	return func(ctx *context.Context, cmd Command) error {
		return fmt.Errorf("*** %s: feature not yet implemented", verb)
	}
}

var commandMux *CommandMux
