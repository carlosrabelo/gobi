package repl

import (
	"fmt"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/script"
)

func finishAdvance(ctrl *script.Controller) bool {
	if ctrl.Advance() {
		return false
	}
	return ctrl.Depth() <= 1
}

func executeScriptLine(ctx *context.Context, ctrl *script.Controller, line script.Line) (stop bool, err error) {
	prog := ctrl.Program()
	if prog == nil {
		return false, fmt.Errorf("*** script: no active program")
	}

	switch line.Command.Verb {
	case "RETURN":
		return true, nil
	}

	if err := scriptDispatch(ctx, line.Command); err != nil {
		return false, fmt.Errorf("*** Error in %s, line %d: %w", prog.Path, line.Number, err)
	}

	if finishAdvance(ctrl) {
		return true, nil
	}

	return false, nil
}
