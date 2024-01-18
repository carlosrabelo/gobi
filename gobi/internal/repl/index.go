package repl

import (
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
