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

// NewCommandMux creates a mux with all known dBase II commands registered.
func NewCommandMux() *CommandMux {
	m := &CommandMux{handlers: make(map[string]HandlerFunc)}
	m.registerAll()
	return m
}

// Dispatch routes a parsed command to its handler.
// Returns an error if the verb is unknown or ambiguous.
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

// resolveVerb finds the unique full verb that has input as a prefix.
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
	m.register("?", handleQuestion)
	m.register("@", handleAt)
	m.register("ACCEPT", handleAccept)
	m.register("APPEND", handleAppend)
	m.register("AVERAGE", handleAverage)
	m.register("??", handleQuestion)
	m.register("BROWSE", handleBrowse)
	m.register("CANCEL", handleCancel)
	m.register("CASE", handleCaseInteractive)
	m.register("CHANGE", handleChange)
	m.register("CLEAR", handleClear)
	m.register("CLOSE", handleClose)
	m.register("CONTINUE", handleContinue)
	m.register("COPY", handleCopy)
	m.register("COUNT", handleCount)
	m.register("CREATE", handleCreate)
	m.register("DELETE", handleDelete)
	m.register("DIR", handleDir)
	m.register("DISPLAY", handleDisplay)
	m.register("DO", handleDo)
	m.register("EDIT", handleEdit)
	m.register("ENDCASE", handleEndCaseInteractive)
	m.register("DIR", handleDir)
	m.register("ERASE", handleErase)
	m.register("FIND", handleFind)
	m.register("GO", handleGo)
	m.register("GOTO", handleGoto)
	m.register("HELP", handleHelp)
	m.register("INDEX", handleIndex)
	m.register("INPUT", handleInput)
	m.register("INSERT", handleInsert)
	m.register("JOIN", handleJoin)
	m.register("LIST", handleList)
	m.register("MODIFY", handleModify)
	m.register("LOCATE", handleLocate)
	m.register("LOOP", stubHandler("LOOP"))
	m.register("NOTE", handleNote)
	m.register("OTHERWISE", handleOtherwiseInteractive)
	m.register("PACK", handlePack)
	m.register("QUIT", handleQuit)
	m.register("READ", handleRead)
	m.register("RECALL", handleRecall)
	m.register("REINDEX", handleReindex)
	m.register("RELEASE", handleRelease)
	m.register("REMARK", handleRemark)
	m.register("RENAME", handleRename)
	m.register("REPLACE", handleReplace)
	m.register("RESTORE", handleRestore)
	m.register("RETURN", stubHandler("RETURN"))
	m.register("SAVE", handleSave)
	m.register("SEEK", handleSeek)
	m.register("SELECT", handleSelect)
	m.register("SET", handleSet)
	m.register("SKIP", handleSkip)
	m.register("SORT", handleSort)
	m.register("STORE", handleStore)
	m.register("SUM", handleSum)
	m.register("TEXT", handleTextInteractive)
	m.register("ENDTEXT", handleEndTextInteractive)
	m.register("TOTAL", handleTotal)
	m.register("UPDATE", handleUpdate)
	m.register("USE", handleUse)
	m.register("WAIT", handleWait)
	m.register("ZAP", handleZap)
}

// stubHandler returns a handler that prints "not implemented" for the given verb.
func stubHandler(verb string) HandlerFunc {
	return func(ctx *context.Context, cmd Command) error {
		return fmt.Errorf("*** %s: feature not yet implemented", verb)
	}
}
