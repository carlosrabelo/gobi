package repl

import (
	"fmt"
	"os"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/script"
)

var scriptDispatch func(*context.Context, Command) error

func init() {
	commandMux = NewCommandMux()
	scriptDispatch = commandMux.Dispatch
}

// RunScript loads and executes a command file using the default command mux.
func RunScript(ctx *context.Context, filename string) error {
	prog, err := loadScript(ctx, filename)
	if err != nil {
		return err
	}
	return RunProgram(ctx, prog)
}

// RunProgram executes prog using a script instruction pointer controller.
func RunProgram(ctx *context.Context, prog *script.Program) error {
	if scriptDispatch == nil {
		return fmt.Errorf("*** script dispatcher not initialized")
	}
	if ctx.Script != nil {
		return fmt.Errorf("*** script already running")
	}

	ctrl := script.NewController(prog)
	ctx.Script = ctrl
	pushScriptFrame(ctx, prog.Path)

	defer func() {
		ctx.Script = nil
		ctx.ExecutionStack = ctx.ExecutionStack[:0]
	}()

	return runScriptLoop(ctx)
}

func loadScript(ctx *context.Context, filename string) (*script.Program, error) {
	path := resolveDataPath(ctx, filename, ".prg")

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("*** Command file not found")
		}
		return nil, fmt.Errorf("*** %s", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("*** not a command file")
	}

	prog, err := script.Read(path)
	if err != nil {
		return nil, fmt.Errorf("*** %s", err.Error())
	}
	return prog, nil
}

func pushScriptFrame(ctx *context.Context, path string) {
	ctx.ExecutionStack = append(ctx.ExecutionStack, path)
}

func popScriptFrame(ctx *context.Context) {
	if len(ctx.ExecutionStack) > 0 {
		ctx.ExecutionStack = ctx.ExecutionStack[:len(ctx.ExecutionStack)-1]
	}
}

func popCallerFrame(ctx *context.Context, ctrl *script.Controller) bool {
	if ctrl.Depth() <= 1 {
		return false
	}
	popScriptFrame(ctx)
	ctrl.PopFrame()
	ctrl.Advance()
	return true
}

func clearScriptExecution(ctx *context.Context) {
	ctx.Script = nil
	ctx.ExecutionStack = ctx.ExecutionStack[:0]
}

func runScriptLoop(ctx *context.Context) error {
	ctrl := ctx.Script
	if ctrl == nil {
		return fmt.Errorf("*** script dispatcher not initialized")
	}

	for {
		line, ok := ctrl.Current()
		if !ok {
			if ctrl.Depth() <= 1 {
				return nil
			}
			popCallerFrame(ctx, ctrl)
			continue
		}

		stop, err := executeScriptLine(ctx, ctrl, line)
		if err != nil {
			return err
		}
		if stop {
			return nil
		}
	}
}
