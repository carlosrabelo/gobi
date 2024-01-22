package repl

import (
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
	"github.com/carlosrabelo/gobi/gobi/pkg/ndx"
)

func handleFind(ctx *context.Context, cmd Command) error {
	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	idx, err := activeIndex(area)
	if err != nil {
		return err
	}

	header := idx.Manager().Header()
	if header.KeyType != ndx.KeyTypeCharacter {
		return fmt.Errorf("*** FIND requires a character index")
	}

	prefix, err := parseFindString(cmd.Args)
	if err != nil {
		return err
	}

	result, found, err := idx.Manager().SearchPrefix(ndx.Key(prefix))
	if err != nil {
		return fmt.Errorf("*** Index search error: %w", err)
	}

	return finishIndexSearch(ctx, area, found, result.RecordNumber)
}

func handleSeek(ctx *context.Context, cmd Command) error {
	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	idx, err := activeIndex(area)
	if err != nil {
		return err
	}

	expression := strings.TrimSpace(cmd.Args)
	if expression == "" {
		return fmt.Errorf("*** SEEK requires an expression")
	}

	keyExp, err := parseReplExpression(expression)
	if err != nil {
		return err
	}

	env := newReplEnvironment(ctx)
	value, err := expr.Eval(keyExp, env)
	if err != nil {
		return fmt.Errorf("*** Evaluation error in SEEK expression: %w", err)
	}

	key, err := indexKeyFromValue(idx.Manager().Header(), value)
	if err != nil {
		return err
	}

	result, found, err := idx.Manager().SearchExact(key)
	if err != nil {
		return fmt.Errorf("*** Index search error: %w", err)
	}

	return finishIndexSearch(ctx, area, found, result.RecordNumber)
}

func activeIndex(area *context.WorkArea) (*ndx.Index, error) {
	if area == nil || len(area.Indexes) == 0 {
		return nil, fmt.Errorf("*** No index files are in use")
	}
	idx := area.Indexes[0]
	if idx == nil || idx.Manager() == nil {
		return nil, fmt.Errorf("*** No index files are in use")
	}
	return idx, nil
}

func indexKeyFromValue(h *ndx.Header, value expr.Object) (ndx.Key, error) {
	text, _, err := indexValueText(value)
	if err != nil {
		return nil, err
	}
	return ndx.KeyFromText(h, text)
}

func parseFindString(args string) (string, error) {
	trimmed := strings.TrimSpace(args)
	if trimmed == "" {
		return "", fmt.Errorf("*** FIND requires a search string")
	}
	if len(trimmed) >= 2 {
		if trimmed[0] == '\'' && trimmed[len(trimmed)-1] == '\'' {
			return trimmed[1 : len(trimmed)-1], nil
		}
		if trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"' {
			return trimmed[1 : len(trimmed)-1], nil
		}
	}
	return trimmed, nil
}

func finishIndexSearch(ctx *context.Context, area *context.WorkArea, found bool, recordNumber uint16) error {
	clearLocateState(area)
	if !found {
		area.Found = false
		area.RecordNo = int(area.Table.Header.RecordCount)
		area.ActiveRecord = nil
		return nil
	}

	area.Found = true
	if err := goToRecord(ctx, int(recordNumber)); err != nil {
		return err
	}

	talkPrint(ctx, "RECORD: %05d\r\n", recordNumber)
	return nil
}
