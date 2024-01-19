package repl

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
	"github.com/carlosrabelo/gobi/gobi/pkg/ndx"
)

type keyScanState struct {
	keyType      ndx.KeyType
	keyLength    uint16
	sawCharacter bool
	sawNumeric   bool
}

type indexScanResult struct {
	mappings   []ndx.LeafEntry
	expression string
	keyType    ndx.KeyType
	keyLength  uint16
}

func handleIndex(ctx *context.Context, cmd Command) error {
	expression, err := parseIndexOnExpression(cmd.Args)
	if err != nil {
		return err
	}
	if cmd.ToClause == "" {
		return fmt.Errorf("*** INDEX requires TO <filename>")
	}

	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	scan, err := scanIndexMappings(ctx, area, expression)
	if err != nil {
		return err
	}

	filePath := resolveNDXFilePath(ctx, cmd.ToClause)
	destroy, err := confirmDestroyExistingFile(ctx, ctx.StdinReader(), filePath)
	if err != nil {
		return err
	}
	if !destroy {
		return nil
	}

	closeIndexAtPath(area, filePath)

	header := ndx.NewHeaderForExpression(scan.expression, scan.keyType, scan.keyLength)
	idx, err := ndx.CreateIndexFile(filePath, header, scan.mappings)
	if err != nil {
		return fmt.Errorf("*** Error creating index: %w", err)
	}

	area.Indexes = append(area.Indexes, idx)

	talkPrint(ctx, "INDEX ON %s TO %s\r\n", scan.expression, filepath.Base(filePath))

	return nil
}

// syncOpenIndexesAfterAppend updates every open index after a record is appended.
func syncOpenIndexesAfterAppend(ctx *context.Context, area *context.WorkArea, recNo int) error {
	if area == nil || len(area.Indexes) == 0 {
		return nil
	}

	for _, idx := range area.Indexes {
		if idx == nil || idx.Manager() == nil {
			continue
		}
		expression := idx.Manager().Header().Expression
		key, err := evaluateIndexKeyForRecord(ctx, area, idx.Manager().Header(), expression, recNo)
		if err != nil {
			return err
		}

		if err := idx.Manager().InsertMapping(uint16(recNo+1), key); err != nil {
			scan, scanErr := scanIndexMappings(ctx, area, expression)
			if scanErr != nil {
				return scanErr
			}
			header := ndx.NewHeaderForExpression(scan.expression, scan.keyType, scan.keyLength)
			if rebuildErr := ndx.RebuildIndex(idx, header, scan.mappings); rebuildErr != nil {
				return fmt.Errorf("*** Error updating index %s: %w", filepath.Base(idx.Path), err)
			}
		}
	}
	return nil
}

// syncOpenIndexesAfterReplace updates index entries when a record's key values change.
func syncOpenIndexesAfterReplace(ctx *context.Context, area *context.WorkArea, recNo int, beforeRec *dbf.Record) error {
	if area == nil || len(area.Indexes) == 0 {
		return nil
	}

	for _, idx := range area.Indexes {
		if idx == nil || idx.Manager() == nil {
			continue
		}

		header := idx.Manager().Header()
		expression := header.Expression
		oldKey, err := evaluateIndexKeyFromRecord(ctx, area, header, expression, beforeRec, recNo)
		if err != nil {
			return err
		}
		newKey, err := evaluateIndexKeyForRecord(ctx, area, header, expression, recNo)
		if err != nil {
			return err
		}

		equal, err := indexKeysEqual(header, oldKey, newKey)
		if err != nil {
			return err
		}
		if equal {
			continue
		}

		needsRebuild := false
		if _, err := idx.Manager().DeleteMapping(oldKey); err != nil {
			if !errors.Is(err, ndx.ErrLeafKeyNotFound) {
				needsRebuild = true
			}
		}
		if !needsRebuild {
			if err := idx.Manager().InsertMapping(uint16(recNo+1), newKey); err != nil {
				needsRebuild = true
			}
		}
		if needsRebuild {
			scan, scanErr := scanIndexMappings(ctx, area, expression)
			if scanErr != nil {
				return scanErr
			}
			rebuildHeader := ndx.NewHeaderForExpression(scan.expression, scan.keyType, scan.keyLength)
			if err := ndx.RebuildIndex(idx, rebuildHeader, scan.mappings); err != nil {
				return fmt.Errorf("*** Error updating index %s: %w", filepath.Base(idx.Path), err)
			}
		}
	}
	return nil
}

func indexKeysEqual(header *ndx.Header, a, b ndx.Key) (bool, error) {
	cmp, err := ndx.CompareKeys(header, a, b)
	if err != nil {
		return false, err
	}
	return cmp == 0, nil
}

func evaluateIndexKeyForRecord(ctx *context.Context, area *context.WorkArea, header *ndx.Header, expression string, recNo int) (ndx.Key, error) {
	rseeker, ok := area.Table.Underlying().(io.ReadSeeker)
	if !ok {
		return nil, fmt.Errorf("*** Underlying database stream is not seekable")
	}

	rec, err := area.Table.ReadRecordAt(rseeker, recNo)
	if err != nil {
		return nil, fmt.Errorf("*** Error reading record %d: %w", recNo+1, err)
	}

	return evaluateIndexKeyFromRecord(ctx, area, header, expression, rec, recNo)
}

func evaluateIndexKeyFromRecord(ctx *context.Context, area *context.WorkArea, header *ndx.Header, expression string, rec *dbf.Record, recNo int) (ndx.Key, error) {
	keyExp, err := parseReplExpression(expression)
	if err != nil {
		return nil, err
	}

	savedRecordNo := area.RecordNo
	savedRecord := area.ActiveRecord
	defer func() {
		area.RecordNo = savedRecordNo
		area.ActiveRecord = savedRecord
	}()

	area.RecordNo = recNo
	area.ActiveRecord = rec

	value, err := expr.Eval(keyExp, newReplEnvironment(ctx))
	if err != nil {
		return nil, fmt.Errorf("*** Evaluation error in index expression: %w", err)
	}

	return indexKeyFromValue(header, value)
}

func scanIndexMappings(ctx *context.Context, area *context.WorkArea, expression string) (*indexScanResult, error) {
	keyExp, err := parseReplExpression(expression)
	if err != nil {
		return nil, err
	}

	rseeker, ok := area.Table.Underlying().(io.ReadSeeker)
	if !ok {
		return nil, fmt.Errorf("*** Underlying database stream is not seekable")
	}

	savedRecordNo := area.RecordNo
	savedRecord := area.ActiveRecord
	defer func() {
		area.RecordNo = savedRecordNo
		area.ActiveRecord = savedRecord
	}()

	env := newReplEnvironment(ctx)
	state := keyScanState{keyType: ndx.KeyTypeCharacter}
	var mappings []ndx.LeafEntry

	recCount := int(area.Table.Header.RecordCount)
	for i := 0; i < recCount; i++ {
		rec, err := area.Table.ReadRecordAt(rseeker, i)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("*** Error reading record %d: %w", i+1, err)
		}

		area.RecordNo = i
		area.ActiveRecord = rec

		value, err := expr.Eval(keyExp, env)
		if err != nil {
			return nil, fmt.Errorf("*** Evaluation error in index expression: %w", err)
		}

		text, numeric, err := indexValueText(value)
		if err != nil {
			return nil, err
		}
		if err := updateKeyScanState(&state, text, numeric); err != nil {
			return nil, err
		}

		mappings = append(mappings, ndx.LeafEntry{
			RecordNumber: uint16(i + 1),
			Key:          ndx.Key(text),
		})
	}

	if state.keyLength == 0 {
		state.keyLength = inferKeyLengthFromExpression(area.Table, expression)
	}

	return &indexScanResult{
		mappings:   mappings,
		expression: expression,
		keyType:    state.keyType,
		keyLength:  state.keyLength,
	}, nil
}

func parseReplExpression(source string) (expr.Expression, error) {
	lexer := expr.NewLexer(source)
	parser := expr.NewParser(lexer)
	exp := parser.ParseExpression()
	if len(parser.Errors()) > 0 {
		return nil, fmt.Errorf("*** Syntax error: %s", strings.Join(parser.Errors(), "; "))
	}
	return exp, nil
}

func parseIndexOnExpression(args string) (string, error) {
	trimmed := strings.TrimSpace(args)
	if trimmed == "" {
		return "", fmt.Errorf("*** INDEX requires ON <expression>")
	}
	upper := strings.ToUpper(trimmed)
	if !strings.HasPrefix(upper, "ON ") {
		return "", fmt.Errorf("*** INDEX requires ON <expression>")
	}
	expression := strings.TrimSpace(trimmed[2:])
	if expression == "" {
		return "", fmt.Errorf("*** INDEX requires ON <expression>")
	}
	return expression, nil
}

func indexValueText(value expr.Object) (string, bool, error) {
	switch obj := value.(type) {
	case *expr.StringObject:
		return obj.Value, false, nil
	case *expr.NumberObject:
		if obj.Value == float64(int64(obj.Value)) {
			return fmt.Sprintf("%d", int64(obj.Value)), true, nil
		}
		return fmt.Sprintf("%g", obj.Value), true, nil
	case *expr.BooleanObject:
		if obj.Value {
			return ".T.", false, nil
		}
		return ".F.", false, nil
	default:
		return "", false, fmt.Errorf("*** Index expression must evaluate to character or numeric values")
	}
}

func indexKeyFromValue(h *ndx.Header, value expr.Object) (ndx.Key, error) {
	text, _, err := indexValueText(value)
	if err != nil {
		return nil, err
	}
	return ndx.KeyFromText(h, text)
}

func updateKeyScanState(state *keyScanState, text string, numeric bool) error {
	if numeric {
		if state.sawCharacter {
			return fmt.Errorf("*** Index expression must not mix character and numeric values")
		}
		state.sawNumeric = true
		state.keyType = ndx.KeyTypeNumeric
	} else {
		if state.sawNumeric {
			return fmt.Errorf("*** Index expression must not mix character and numeric values")
		}
		state.sawCharacter = true
		state.keyType = ndx.KeyTypeCharacter
	}
	return ndx.UpdateKeyMetadata(&state.keyType, &state.keyLength, text, numeric)
}

func inferKeyLengthFromExpression(tbl *dbf.Table, expression string) uint16 {
	if tbl == nil {
		return 10
	}
	name := strings.ToUpper(strings.TrimSpace(expression))
	if strings.ContainsAny(name, "+-*/()") {
		return 10
	}
	fd, _ := tbl.FieldByName(name)
	if fd != nil {
		return uint16(fd.Length)
	}
	return 10
}

func closeIndexAtPath(area *context.WorkArea, filePath string) {
	if area == nil {
		return
	}
	remaining := area.Indexes[:0]
	for _, idx := range area.Indexes {
		if idx != nil && idx.Path == filePath {
			_ = idx.Close()
			continue
		}
		remaining = append(remaining, idx)
	}
	area.Indexes = remaining
}
