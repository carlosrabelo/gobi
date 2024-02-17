package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

func handleQuit(ctx *context.Context, cmd Command) error {
	return errQuit
}

func handleSelect(ctx *context.Context, cmd Command) error {
	arg := strings.ToUpper(strings.TrimSpace(cmd.Args))
	if arg == "" {
		return fmt.Errorf("*** SELECT requires PRIMARY or SECONDARY")
	}

	var msg string
	switch arg {
	case "PRIMARY":
		if err := ctx.SelectArea(context.Primary); err != nil {
			return err
		}
		msg = "Primary work area selected"
	case "SECONDARY":
		if err := ctx.SelectArea(context.Secondary); err != nil {
			return err
		}
		msg = "Secondary work area selected"
	default:
		return fmt.Errorf("*** Unrecognized SELECT option: %s", arg)
	}

	fmt.Fprintln(ctx.Stdout, msg)
	return nil
}

func closeWorkAreaDatabase(area *context.WorkArea, defaultAlias string) error {
	if area.Table == nil {
		return nil
	}
	if err := area.Table.Close(); err != nil {
		return fmt.Errorf("*** Error closing database: %w", err)
	}
	area.Table = nil
	area.RecordNo = 0
	area.ActiveRecord = nil
	area.Alias = defaultAlias
	clearLocateState(area)
	return nil
}

func handleClose(ctx *context.Context, cmd Command) error {
	arg := strings.ToUpper(strings.TrimSpace(cmd.Args))
	switch arg {
	case "", "DATABASES":
		for name, area := range ctx.WorkAreas {
			if err := closeWorkAreaDatabase(area, string(name)); err != nil {
				return err
			}
		}
		fmt.Fprintln(ctx.Stdout, "Database area closed")
	case "INDEX":
		area := ctx.GetActiveArea()
		closeOpenIndexes(area)
		fmt.Fprintln(ctx.Stdout, "Indexes closed")
	case "ALL":
		for name, area := range ctx.WorkAreas {
			if err := closeWorkAreaDatabase(area, string(name)); err != nil {
				return err
			}
			closeOpenIndexes(area)
		}
		fmt.Fprintln(ctx.Stdout, "All files closed")
	default:
		return fmt.Errorf("*** Unrecognized CLOSE option: %s", arg)
	}
	return nil
}

func handleDisplay(ctx *context.Context, cmd Command) error {
	arg := strings.ToUpper(strings.TrimSpace(cmd.Args))
	switch arg {
	case "MEMORY":
		return outputMemory(ctx)
	case "STRUCTURE":
		return displayStructure(ctx)
	default:
		if cmd.ToClause != "" {
			return fmt.Errorf("*** DISPLAY does not support TO clause")
		}
		return outputRecords(ctx, cmd, outputRecordsOpts{
			maxRecords:       displayPageSize(ctx),
			startFromCurrent: true,
			moveToEOFAfter:   false,
		})
	}
}

func displayPageSize(ctx *context.Context) int {
	rows := term.DefaultRows
	if ctx != nil && ctx.Screen != nil {
		rows = ctx.Screen.Rows()
	}
	size := rows - 4
	if size < 1 {
		size = 1
	}
	return size
}

type column struct {
	expr expr.Expression
	fd   *dbf.FieldDescriptor
}

type outputRecordsOpts struct {
	maxRecords       int
	maxScanned       int
	startFromCurrent bool
	moveToEOFAfter   bool
}

func handleList(ctx *context.Context, cmd Command) error {
	argUpper := strings.ToUpper(strings.TrimSpace(cmd.Args))
	if argUpper == "MEMORY" {
		return outputMemory(ctx)
	}

	if argUpper == "STRUCTURE" {
		return displayStructure(ctx)
	}

	return outputRecords(ctx, cmd, outputRecordsOpts{
		maxRecords:       0,
		startFromCurrent: false,
		moveToEOFAfter:   true,
	})
}

func outputRecords(ctx *context.Context, cmd Command, opts outputRecordsOpts) error {
	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	var err error
	var out io.Writer = ctx.Stdout
	if cmd.ToClause != "" {
		filePath := resolveOutputPath(ctx, cmd.ToClause)
		f, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("*** Could not create output file: %w", err)
		}
		defer f.Close()
		out = io.MultiWriter(ctx.Stdout, f)
	}

	var forExp, whileExp expr.Expression
	if forExp, whileExp, err = parseForWhileClauses(cmd); err != nil {
		return err
	}

	var cols []column
	if strings.TrimSpace(cmd.Args) == "" {
		for i := range area.Table.Fields {
			fd := &area.Table.Fields[i]
			ident := &expr.Identifier{
				Name: fd.Name,
			}
			cols = append(cols, column{expr: ident, fd: fd})
		}
	} else {
		parts := splitCommaOutsideParens(cmd.Args)
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			lexer := expr.NewLexer(part)
			parser := expr.NewParser(lexer)
			exp := parser.ParseExpression()
			if len(parser.Errors()) > 0 {
				return fmt.Errorf("*** Syntax error in field list: %s", strings.Join(parser.Errors(), "; "))
			}
			var matchedFd *dbf.FieldDescriptor
			if ident, ok := exp.(*expr.Identifier); ok {
				fd, idx := area.Table.FieldByName(ident.Name)
				if fd != nil {
					matchedFd = &area.Table.Fields[idx]
				}
			}
			cols = append(cols, column{expr: exp, fd: matchedFd})
		}
	}

	widths := make([]int, len(cols))
	var headerParts []string
	for i, col := range cols {
		name := col.expr.String()
		width := len(name)
		if col.fd != nil {
			if int(col.fd.Length) > width {
				width = int(col.fd.Length)
			}
		}
		if width < 5 {
			width = 5
		}
		widths[i] = width
		headerParts = append(headerParts, fmt.Sprintf("%-*s", width, strings.ToUpper(name)))
	}
	fmt.Fprintf(out, "Record#  %s\r\n", strings.Join(headerParts, "  "))

	rseeker, ok := area.Table.Underlying().(io.ReadSeeker)
	if !ok {
		return fmt.Errorf("*** Underlying database stream is not seekable")
	}

	recCount := int(area.Table.Header.RecordCount)

	seq, err := recordSequence(area)
	if err != nil {
		return err
	}

	startPos := 0
	if opts.startFromCurrent || whileExp != nil {
		startPos = positionInSequence(seq, area.RecordNo, recCount)
	}

	env := newReplEnvironment(ctx)
	displayed := 0
	scanned := 0
	lastDisplayed := -1
	var lastRecord *dbf.Record

	for _, i := range seq[startPos:] {
		if opts.maxRecords > 0 && displayed >= opts.maxRecords {
			break
		}
		if opts.maxScanned > 0 && scanned >= opts.maxScanned {
			break
		}
		scanned++

		rec, err := area.Table.ReadRecordAt(rseeker, i)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("*** Error reading record %d: %w", i, err)
		}

		area.RecordNo = i
		area.ActiveRecord = rec

		if whileExp != nil {
			res, err := expr.Eval(whileExp, env)
			if err != nil {
				return fmt.Errorf("*** Evaluation error in WHILE clause: %w", err)
			}
			boolRes, ok := res.(*expr.BooleanObject)
			if !ok {
				return fmt.Errorf("*** WHILE clause must evaluate to a logical value")
			}
			if !boolRes.Value {
				break
			}
		}

		if forExp != nil {
			res, err := expr.Eval(forExp, env)
			if err != nil {
				return fmt.Errorf("*** Evaluation error in FOR clause: %w", err)
			}
			boolRes, ok := res.(*expr.BooleanObject)
			if !ok {
				return fmt.Errorf("*** FOR clause must evaluate to a logical value")
			}
			if !boolRes.Value {
				continue
			}
		}

		fmt.Fprintf(out, "%5d  ", i+1)
		var valParts []string
		for j, col := range cols {
			valObj, err := expr.Eval(col.expr, env)
			if err != nil {
				valParts = append(valParts, fmt.Sprintf("%-*s", widths[j], ""))
				continue
			}
			valParts = append(valParts, formatColumnValue(col, valObj, widths[j]))
		}
		fmt.Fprintf(out, "%s\r\n", strings.Join(valParts, "  "))

		displayed++
		lastDisplayed = i
		lastRecord = rec
	}

	if opts.moveToEOFAfter && whileExp == nil {
		area.RecordNo = recCount
		area.ActiveRecord = nil
	} else if lastDisplayed >= 0 {
		area.RecordNo = lastDisplayed
		area.ActiveRecord = lastRecord
	}

	return nil
}

func formatColumnValue(col column, valObj expr.Object, width int) string {
	if col.fd != nil {
		switch col.fd.Type {
		case dbf.FieldTypeChar:
			return fmt.Sprintf("%-*s", width, valObj.String())
		case dbf.FieldTypeNumeric:
			if numObj, ok := valObj.(*expr.NumberObject); ok {
				if col.fd.DecimalCount > 0 {
					return fmt.Sprintf("%*.*f", width, col.fd.DecimalCount, numObj.Value)
				}
				return fmt.Sprintf("%*.0f", width, numObj.Value)
			}
			return fmt.Sprintf("%*s", width, valObj.String())
		case dbf.FieldTypeLogical:
			return fmt.Sprintf("%-*s", width, valObj.String())
		default:
			return fmt.Sprintf("%-*s", width, valObj.String())
		}
	}

	switch v := valObj.(type) {
	case *expr.NumberObject:
		return fmt.Sprintf("%*.2f", width, v.Value)
	default:
		return fmt.Sprintf("%-*s", width, valObj.String())
	}
}

func parseForWhileClauses(cmd Command) (expr.Expression, expr.Expression, error) {
	var forExp expr.Expression
	if cmd.ForClause != "" {
		lexer := expr.NewLexer(cmd.ForClause)
		parser := expr.NewParser(lexer)
		forExp = parser.ParseExpression()
		if len(parser.Errors()) > 0 {
			return nil, nil, fmt.Errorf("*** Syntax error in FOR clause: %s", strings.Join(parser.Errors(), "; "))
		}
	}

	var whileExp expr.Expression
	if cmd.WhileClause != "" {
		lexer := expr.NewLexer(cmd.WhileClause)
		parser := expr.NewParser(lexer)
		whileExp = parser.ParseExpression()
		if len(parser.Errors()) > 0 {
			return nil, nil, fmt.Errorf("*** Syntax error in WHILE clause: %s", strings.Join(parser.Errors(), "; "))
		}
	}

	return forExp, whileExp, nil
}

func recordSequence(area *context.WorkArea) ([]int, error) {
	recCount := int(area.Table.Header.RecordCount)
	seq := make([]int, recCount)
	for i := range seq {
		seq[i] = i
	}
	return seq, nil
}

func positionInSequence(seq []int, recNo, recCount int) int {
	if recNo >= recCount {
		return len(seq)
	}
	for pos, recIdx := range seq {
		if recIdx == recNo {
			return pos
		}
	}
	return len(seq)
}

func splitCommaOutsideParens(s string) []string {
	var parts []string
	var cur strings.Builder
	parens := 0
	inSingle := false
	inDouble := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
			cur.WriteByte(ch)
		case ch == '"' && !inSingle:
			inDouble = !inDouble
			cur.WriteByte(ch)
		case ch == '(' && !inSingle && !inDouble:
			parens++
			cur.WriteByte(ch)
		case ch == ')' && !inSingle && !inDouble:
			parens--
			cur.WriteByte(ch)
		case ch == ',' && !inSingle && !inDouble && parens == 0:
			parts = append(parts, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteByte(ch)
		}
	}
	if cur.Len() > 0 {
		parts = append(parts, strings.TrimSpace(cur.String()))
	}
	return parts
}

func displayStructure(ctx *context.Context) error {
	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	tbl := area.Table
	fmt.Fprintf(ctx.Stdout, "STRUCTURE FOR FILE:  %s.DBF\r\n", area.Alias)
	fmt.Fprintf(ctx.Stdout, "NUMBER OF RECORDS:   %05d\r\n", tbl.Header.RecordCount)
	fmt.Fprintf(ctx.Stdout, "DATE OF LAST UPDATE: 00/00/00\r\n")
	fmt.Fprintf(ctx.Stdout, "FLD       NAME       TYPE WIDTH DEC\r\n")

	totalWidth := 1
	for i, fd := range tbl.Fields {
		decStr := ""
		if fd.Type == dbf.FieldTypeNumeric {
			decStr = fmt.Sprintf("%03d", fd.DecimalCount)
		}
		fmt.Fprintf(ctx.Stdout, "%03d       %-10s  %c   %03d   %3s\r\n",
			i+1, fd.Name, fd.Type, fd.Length, decStr)
		totalWidth += int(fd.Length)
	}
	fmt.Fprintf(ctx.Stdout, "** TOTAL **                %05d\r\n", totalWidth)
	return nil
}

func handleGo(ctx *context.Context, cmd Command) error {
	arg := strings.ToUpper(strings.TrimSpace(cmd.Args))
	parts := strings.Fields(arg)
	if len(parts) == 2 && parts[0] == "TO" {
		return gotoRecordFromArgs(ctx, parts[1])
	}
	switch arg {
	case "TOP":
		return goTop(ctx)
	case "BOTTOM":
		return goBottom(ctx)
	default:
		return fmt.Errorf("*** Unrecognized GO option: %s", strings.TrimSpace(cmd.Args))
	}
}

func handleGoto(ctx *context.Context, cmd Command) error {
	return gotoRecordFromArgs(ctx, strings.TrimSpace(cmd.Args))
}

func gotoRecordFromArgs(ctx *context.Context, arg string) error {
	if arg == "" {
		return fmt.Errorf("*** GOTO requires a record number")
	}
	userRecNo, err := strconv.Atoi(arg)
	if err != nil {
		return fmt.Errorf("*** Invalid record number: %s", arg)
	}
	return goToRecord(ctx, userRecNo)
}

func goToRecord(ctx *context.Context, userRecNo int) error {
	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}
	if userRecNo < 1 {
		return fmt.Errorf("*** Record number out of range")
	}

	recCount := int(area.Table.Header.RecordCount)
	recIdx := userRecNo - 1

	if recIdx >= recCount {
		area.RecordNo = recIdx
		area.ActiveRecord = nil
		return nil
	}

	rseeker, ok := area.Table.Underlying().(io.ReadSeeker)
	if !ok {
		return fmt.Errorf("*** Underlying database stream is not seekable")
	}

	rec, err := area.Table.ReadRecordAt(rseeker, recIdx)
	if err != nil {
		return fmt.Errorf("*** Error reading record %d: %w", userRecNo, err)
	}

	area.RecordNo = recIdx
	area.ActiveRecord = rec
	return nil
}

func goTop(ctx *context.Context) error {
	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}
	return goToRecord(ctx, 1)
}

func goBottom(ctx *context.Context) error {
	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}
	recCount := int(area.Table.Header.RecordCount)
	if recCount == 0 {
		return goToRecord(ctx, 1)
	}
	return goToRecord(ctx, recCount)
}

func handleSkip(ctx *context.Context, cmd Command) error {
	arg := strings.TrimSpace(cmd.Args)
	delta := 1
	if arg != "" {
		n, err := strconv.Atoi(arg)
		if err != nil {
			return fmt.Errorf("*** Invalid SKIP value: %s", arg)
		}
		delta = n
	}
	return skipRecords(ctx, delta)
}

func skipRecords(ctx *context.Context, delta int) error {
	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}
	return goToRecord(ctx, area.RecordNo+1+delta)
}

func handleAppend(ctx *context.Context, cmd Command) error {
	if strings.TrimSpace(cmd.FromClause) != "" {
		return handleAppendFrom(ctx, cmd)
	}
	arg := strings.ToUpper(strings.TrimSpace(cmd.Args))
	if arg != "" {
		return fmt.Errorf("*** APPEND requires FROM <filename>")
	}

	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	if appendScreenAvailable(ctx) {
		err := runAppendScreen(ctx)
		if consoleErr := returnToConsole(ctx); consoleErr != nil && err == nil {
			err = consoleErr
		}
		return err
	}

	wseeker, ok := area.Table.Underlying().(io.ReadWriteSeeker)
	if !ok {
		return fmt.Errorf("*** Database file is not writable")
	}

	reader := ctx.StdinReader()
	tbl := area.Table

	for {
		values := make([]interface{}, len(tbl.Fields))
		cancelled := false

		for i, fd := range tbl.Fields {
			line, err := readAppendLine(ctx, reader, fmt.Sprintf("%s ? ", fd.Name))
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return fmt.Errorf("*** Error reading input: %w", err)
			}

			if i == 0 && strings.TrimSpace(line) == "" {
				cancelled = true
				break
			}

			val, err := parseFieldInput(fd, line)
			if err != nil {
				return err
			}
			values[i] = val
		}

		if cancelled {
			break
		}

		rec, err := dbf.NewRecord(tbl, false, values)
		if err != nil {
			return fmt.Errorf("*** Error building record: %w", err)
		}

		recNo, err := tbl.AppendRecord(wseeker, rec)
		if err != nil {
			return fmt.Errorf("*** Error appending record: %w", err)
		}

		area.RecordNo = recNo
		area.ActiveRecord = rec

		if err := syncOpenIndexesAfterAppend(ctx, area, recNo); err != nil {
			return err
		}

		talkPrint(ctx, "New record added\r\n")
	}

	return nil
}

func readAppendLine(ctx *context.Context, reader *bufio.Reader, promptText string) (string, error) {
	fmt.Fprint(ctx.Stdout, promptText)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func parseFieldInput(fd dbf.FieldDescriptor, line string) (interface{}, error) {
	switch fd.Type {
	case dbf.FieldTypeChar:
		return line, nil
	case dbf.FieldTypeNumeric:
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			return "", nil
		}
		if _, err := strconv.ParseFloat(trimmed, 64); err != nil {
			return nil, fmt.Errorf("*** Invalid numeric value for %s: %s", fd.Name, line)
		}
		return trimmed, nil
	case dbf.FieldTypeLogical:
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			return false, nil
		}
		switch strings.ToUpper(trimmed)[0] {
		case 'T', 'Y':
			return true, nil
		case 'F', 'N':
			return false, nil
		default:
			return nil, fmt.Errorf("*** Invalid logical value for %s: %s", fd.Name, line)
		}
	default:
		return nil, fmt.Errorf("*** Unsupported field type for %s", fd.Name)
	}
}

func handleReplace(ctx *context.Context, cmd Command) error {
	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	scope, rest, err := parseScopeClause(cmd.Args)
	if err != nil {
		return err
	}

	fieldName, valueExprStr, err := parseReplaceArgs(rest)
	if err != nil {
		return err
	}

	fd, fieldIdx := area.Table.FieldByName(fieldName)
	if fd == nil {
		return fmt.Errorf("*** Unknown field: %s", fieldName)
	}

	lexer := expr.NewLexer(valueExprStr)
	parser := expr.NewParser(lexer)
	valueExp := parser.ParseExpression()
	if len(parser.Errors()) > 0 {
		return fmt.Errorf("*** Syntax error in REPLACE expression: %s", strings.Join(parser.Errors(), "; "))
	}

	forExp, whileExp, err := parseForWhileClauses(cmd)
	if err != nil {
		return err
	}

	wseeker, ok := area.Table.Underlying().(io.ReadWriteSeeker)
	if !ok {
		return fmt.Errorf("*** Database file is not writable")
	}

	tbl := area.Table
	recCount := int(tbl.Header.RecordCount)
	startRec, limit := scanRange(area, scope, forExp != nil, whileExp != nil, true)
	if !scope.explicit && forExp == nil && whileExp == nil && startRec >= recCount {
		return fmt.Errorf("*** Record number out of range")
	}

	env := newReplEnvironment(ctx)
	replaced := 0
	scanned := 0
	lastReplaced := -1
	var lastRecord *dbf.Record

	for i := startRec; i < recCount; i++ {
		if limit > 0 && scanned >= limit {
			break
		}
		scanned++

		rec, err := tbl.ReadRecordAt(wseeker, i)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("*** Error reading record %d: %w", i, err)
		}

		area.RecordNo = i
		area.ActiveRecord = rec

		if whileExp != nil {
			res, err := expr.Eval(whileExp, env)
			if err != nil {
				return fmt.Errorf("*** Evaluation error in WHILE clause: %w", err)
			}
			boolRes, ok := res.(*expr.BooleanObject)
			if !ok {
				return fmt.Errorf("*** WHILE clause must evaluate to a logical value")
			}
			if !boolRes.Value {
				break
			}
		}

		if forExp != nil {
			res, err := expr.Eval(forExp, env)
			if err != nil {
				return fmt.Errorf("*** Evaluation error in FOR clause: %w", err)
			}
			boolRes, ok := res.(*expr.BooleanObject)
			if !ok {
				return fmt.Errorf("*** FOR clause must evaluate to a logical value")
			}
			if !boolRes.Value {
				continue
			}
		}

		valObj, err := expr.Eval(valueExp, env)
		if err != nil {
			return fmt.Errorf("*** Evaluation error in REPLACE expression: %w", err)
		}

		val, err := exprObjectToValue(valObj)
		if err != nil {
			return err
		}

		updated, err := replaceRecordField(tbl, rec, fieldIdx, val)
		if err != nil {
			return fmt.Errorf("*** Error updating record %d: %w", i+1, err)
		}

		if err := tbl.WriteRecordAt(wseeker, i, updated); err != nil {
			return fmt.Errorf("*** Error writing record %d: %w", i+1, err)
		}

		if err := syncOpenIndexesAfterReplace(ctx, area, i, rec); err != nil {
			return err
		}

		area.ActiveRecord = updated
		replaced++
		lastReplaced = i
		lastRecord = updated
	}

	if lastReplaced >= 0 {
		area.RecordNo = lastReplaced
		area.ActiveRecord = lastRecord
	}

	if replaced > 0 {
		talkPrint(ctx, "%d record(s) replaced\r\n", replaced)
	}

	return nil
}

func parseReplaceArgs(args string) (string, string, error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return "", "", fmt.Errorf("*** REPLACE requires <field> WITH <expr>")
	}

	tokens := tokenize(args)
	withIdx := -1
	for i, tok := range tokens {
		if strings.ToUpper(tok) == "WITH" {
			withIdx = i
			break
		}
	}
	if withIdx < 1 {
		return "", "", fmt.Errorf("*** REPLACE requires <field> WITH <expr>")
	}

	fieldName := strings.ToUpper(tokens[0])
	valueExpr := joinTokens(tokens[withIdx+1:])
	if valueExpr == "" {
		return "", "", fmt.Errorf("*** REPLACE requires <field> WITH <expr>")
	}

	return fieldName, valueExpr, nil
}

func exprObjectToValue(obj expr.Object) (interface{}, error) {
	switch v := obj.(type) {
	case *expr.StringObject:
		return v.Value, nil
	case *expr.NumberObject:
		return v.Value, nil
	case *expr.BooleanObject:
		return v.Value, nil
	default:
		return nil, fmt.Errorf("*** REPLACE expression must evaluate to a string, number, or logical value")
	}
}

func replaceRecordField(tbl *dbf.Table, rec *dbf.Record, fieldIdx int, val interface{}) (*dbf.Record, error) {
	values := make([]interface{}, len(tbl.Fields))
	for i := range tbl.Fields {
		if i == fieldIdx {
			values[i] = val
			continue
		}
		decoded, err := rec.DecodeField(tbl, i)
		if err != nil {
			return nil, err
		}
		values[i] = decoded
	}
	return dbf.NewRecord(tbl, rec.Deleted, values)
}

func handleDelete(ctx *context.Context, cmd Command) error {
	scope, rest, err := parseScopeClause(cmd.Args)
	if err != nil {
		return err
	}
	if rest != "" {
		return fmt.Errorf("*** Unexpected argument: %s", rest)
	}

	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	forExp, whileExp, err := parseForWhileClauses(cmd)
	if err != nil {
		return err
	}

	wseeker, ok := area.Table.Underlying().(io.ReadWriteSeeker)
	if !ok {
		return fmt.Errorf("*** Database file is not writable")
	}

	tbl := area.Table
	recCount := int(tbl.Header.RecordCount)
	startRec, limit := scanRange(area, scope, forExp != nil, whileExp != nil, true)
	if !scope.explicit && forExp == nil && whileExp == nil && startRec >= recCount {
		return fmt.Errorf("*** Record number out of range")
	}

	env := newReplEnvironment(ctx)
	deleted := 0
	scanned := 0
	lastDeleted := -1
	var lastRecord *dbf.Record

	for i := startRec; i < recCount; i++ {
		if limit > 0 && scanned >= limit {
			break
		}
		scanned++

		rec, err := tbl.ReadRecordAt(wseeker, i)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("*** Error reading record %d: %w", i, err)
		}

		area.RecordNo = i
		area.ActiveRecord = rec

		if whileExp != nil {
			res, err := expr.Eval(whileExp, env)
			if err != nil {
				return fmt.Errorf("*** Evaluation error in WHILE clause: %w", err)
			}
			boolRes, ok := res.(*expr.BooleanObject)
			if !ok {
				return fmt.Errorf("*** WHILE clause must evaluate to a logical value")
			}
			if !boolRes.Value {
				break
			}
		}

		if forExp != nil {
			res, err := expr.Eval(forExp, env)
			if err != nil {
				return fmt.Errorf("*** Evaluation error in FOR clause: %w", err)
			}
			boolRes, ok := res.(*expr.BooleanObject)
			if !ok {
				return fmt.Errorf("*** FOR clause must evaluate to a logical value")
			}
			if !boolRes.Value {
				continue
			}
		}

		if rec.Deleted {
			lastDeleted = i
			lastRecord = rec
			continue
		}

		marked, err := markRecordDeleted(tbl, rec)
		if err != nil {
			return fmt.Errorf("*** Error deleting record %d: %w", i+1, err)
		}

		if err := tbl.WriteRecordAt(wseeker, i, marked); err != nil {
			return fmt.Errorf("*** Error writing record %d: %w", i+1, err)
		}

		area.ActiveRecord = marked
		deleted++
		lastDeleted = i
		lastRecord = marked
	}

	if lastDeleted >= 0 {
		area.RecordNo = lastDeleted
		area.ActiveRecord = lastRecord
	}

	if deleted > 0 {
		talkPrint(ctx, "%d record(s) deleted\r\n", deleted)
	}

	return nil
}

func markRecordDeleted(tbl *dbf.Table, rec *dbf.Record) (*dbf.Record, error) {
	values := make([]interface{}, len(tbl.Fields))
	for i := range tbl.Fields {
		decoded, err := rec.DecodeField(tbl, i)
		if err != nil {
			return nil, err
		}
		values[i] = decoded
	}
	return dbf.NewRecord(tbl, true, values)
}

func handleRecall(ctx *context.Context, cmd Command) error {
	scope, rest, err := parseScopeClause(cmd.Args)
	if err != nil {
		return err
	}
	if rest != "" {
		return fmt.Errorf("*** Unexpected argument: %s", rest)
	}

	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	forExp, whileExp, err := parseForWhileClauses(cmd)
	if err != nil {
		return err
	}

	wseeker, ok := area.Table.Underlying().(io.ReadWriteSeeker)
	if !ok {
		return fmt.Errorf("*** Database file is not writable")
	}

	tbl := area.Table
	recCount := int(tbl.Header.RecordCount)
	startRec, limit := scanRange(area, scope, forExp != nil, whileExp != nil, true)
	if !scope.explicit && forExp == nil && whileExp == nil && startRec >= recCount {
		return fmt.Errorf("*** Record number out of range")
	}

	env := newReplEnvironment(ctx)
	recalled := 0
	scanned := 0
	lastRecalled := -1
	var lastRecord *dbf.Record

	for i := startRec; i < recCount; i++ {
		if limit > 0 && scanned >= limit {
			break
		}
		scanned++

		rec, err := tbl.ReadRecordAt(wseeker, i)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("*** Error reading record %d: %w", i, err)
		}

		area.RecordNo = i
		area.ActiveRecord = rec

		if whileExp != nil {
			res, err := expr.Eval(whileExp, env)
			if err != nil {
				return fmt.Errorf("*** Evaluation error in WHILE clause: %w", err)
			}
			boolRes, ok := res.(*expr.BooleanObject)
			if !ok {
				return fmt.Errorf("*** WHILE clause must evaluate to a logical value")
			}
			if !boolRes.Value {
				break
			}
		}

		if forExp != nil {
			res, err := expr.Eval(forExp, env)
			if err != nil {
				return fmt.Errorf("*** Evaluation error in FOR clause: %w", err)
			}
			boolRes, ok := res.(*expr.BooleanObject)
			if !ok {
				return fmt.Errorf("*** FOR clause must evaluate to a logical value")
			}
			if !boolRes.Value {
				continue
			}
		}

		if !rec.Deleted {
			lastRecalled = i
			lastRecord = rec
			continue
		}

		restored, err := markRecordRecalled(tbl, rec)
		if err != nil {
			return fmt.Errorf("*** Error recalling record %d: %w", i+1, err)
		}

		if err := tbl.WriteRecordAt(wseeker, i, restored); err != nil {
			return fmt.Errorf("*** Error writing record %d: %w", i+1, err)
		}

		area.ActiveRecord = restored
		recalled++
		lastRecalled = i
		lastRecord = restored
	}

	if lastRecalled >= 0 {
		area.RecordNo = lastRecalled
		area.ActiveRecord = lastRecord
	}

	if recalled > 0 {
		talkPrint(ctx, "%d record(s) recalled\r\n", recalled)
	}

	return nil
}

func markRecordRecalled(tbl *dbf.Table, rec *dbf.Record) (*dbf.Record, error) {
	values := make([]interface{}, len(tbl.Fields))
	for i := range tbl.Fields {
		decoded, err := rec.DecodeField(tbl, i)
		if err != nil {
			return nil, err
		}
		values[i] = decoded
	}
	return dbf.NewRecord(tbl, false, values)
}

func handlePack(ctx *context.Context, cmd Command) error {
	if strings.TrimSpace(cmd.Args) != "" {
		return fmt.Errorf("*** Unexpected argument: %s", strings.TrimSpace(cmd.Args))
	}

	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	wseeker, ok := area.Table.Underlying().(io.ReadWriteSeeker)
	if !ok {
		return fmt.Errorf("*** Database file is not writable")
	}

	removed, err := area.Table.Pack(wseeker)
	if err != nil {
		return fmt.Errorf("*** Error packing database: %w", err)
	}

	area.RecordNo = 0
	if area.Table.Header.RecordCount > 0 {
		rec, err := area.Table.ReadRecordAt(wseeker, 0)
		if err != nil {
			return fmt.Errorf("*** Error reading first record after pack: %w", err)
		}
		area.ActiveRecord = rec
	} else {
		area.ActiveRecord = nil
	}

	if removed > 0 {
		talkPrint(ctx, "%d record(s) packed\r\n", removed)
	}

	return nil
}

func handleZap(ctx *context.Context, cmd Command) error {
	if strings.TrimSpace(cmd.Args) != "" {
		return fmt.Errorf("*** Unexpected argument: %s", strings.TrimSpace(cmd.Args))
	}

	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	wseeker, ok := area.Table.Underlying().(io.ReadWriteSeeker)
	if !ok {
		return fmt.Errorf("*** Database file is not writable")
	}

	removed, err := area.Table.Zap(wseeker)
	if err != nil {
		return fmt.Errorf("*** Error zapping database: %w", err)
	}

	area.RecordNo = 0
	area.ActiveRecord = nil

	if removed > 0 {
		talkPrint(ctx, "%d record(s) zapped\r\n", removed)
	}

	return nil
}
func handleCreate(ctx *context.Context, cmd Command) error {
	reader := ctx.StdinReader()
	filename := strings.TrimSpace(cmd.Args)

	if filename == "" {
		line, err := readAppendLine(ctx, reader, "ENTER FILENAME: ")
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("*** Error reading input: %w", err)
		}
		filename = strings.TrimSpace(line)
		if filename == "" {
			return fmt.Errorf("*** CREATE requires a filename")
		}
	}

	filePath := resolveDBFFilePath(ctx, filename)

	destroy, err := confirmDestroyExistingFile(ctx, reader, filePath)
	if err != nil {
		return err
	}
	if !destroy {
		return nil
	}

	fmt.Fprintln(ctx.Stdout, "ENTER RECORD STRUCTURE AS FOLLOWS:")
	fmt.Fprintln(ctx.Stdout, "FIELD NAME,TYPE,WIDTH,DECIMAL PLACES")

	fields, err := readCreateFieldDefinitions(ctx, reader)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("*** Could not create file: %w", err)
	}

	tbl, err := dbf.Create(f, fields)
	if err != nil {
		f.Close()
		return fmt.Errorf("*** Error creating database: %w", err)
	}
	tbl.Close()

	return handleUse(ctx, Command{Verb: "USE", Args: filename})
}

func confirmDestroyExistingFile(ctx *context.Context, reader *bufio.Reader, filePath string) (bool, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return true, nil
	}

	line, err := readAppendLine(ctx, reader, "DESTROY EXISTING FILE? (Y/N) ")
	if err != nil {
		if err == io.EOF {
			return false, nil
		}
		return false, fmt.Errorf("*** Error reading input: %w", err)
	}

	switch strings.ToUpper(strings.TrimSpace(line)) {
	case "Y":
		return true, nil
	case "N", "":
		return false, nil
	default:
		return false, fmt.Errorf("*** Invalid response to DESTROY EXISTING FILE?")
	}
}

func readCreateFieldDefinitions(ctx *context.Context, reader *bufio.Reader) ([]dbf.FieldDescriptor, error) {
	var fields []dbf.FieldDescriptor
	seenNames := make(map[string]bool)

	for i := 1; i <= 32; i++ {
		line, err := readAppendLine(ctx, reader, fmt.Sprintf("%03d ", i))
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("*** Error reading input: %w", err)
		}
		if strings.TrimSpace(line) == "" {
			break
		}

		fd, err := parseCreateFieldDefinition(line)
		if err != nil {
			return nil, err
		}
		if seenNames[fd.Name] {
			return nil, fmt.Errorf("*** Duplicate field name: %s", fd.Name)
		}
		seenNames[fd.Name] = true
		fields = append(fields, fd)
	}

	if len(fields) == 0 {
		return nil, fmt.Errorf("*** At least one field is required")
	}
	return fields, nil
}

func parseCreateFieldDefinition(line string) (dbf.FieldDescriptor, error) {
	parts := strings.Split(line, ",")
	if len(parts) < 3 {
		return dbf.FieldDescriptor{}, fmt.Errorf("*** Invalid field definition")
	}

	name := strings.ToUpper(strings.TrimSpace(parts[0]))
	if err := dbf.ValidateFieldName(name); err != nil {
		return dbf.FieldDescriptor{}, fmt.Errorf("*** BAD NAME FIELD")
	}

	typeStr := strings.ToUpper(strings.TrimSpace(parts[1]))
	if len(typeStr) != 1 {
		return dbf.FieldDescriptor{}, fmt.Errorf("*** Invalid field type")
	}

	width, err := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil || width <= 0 || width > 254 {
		return dbf.FieldDescriptor{}, fmt.Errorf("*** Invalid field width")
	}

	dec := 0
	if len(parts) >= 4 {
		decPart := strings.TrimSpace(parts[3])
		if decPart != "" {
			dec, err = strconv.Atoi(decPart)
			if err != nil || dec < 0 {
				return dbf.FieldDescriptor{}, fmt.Errorf("*** Invalid decimal places")
			}
		}
	}

	var fd dbf.FieldDescriptor
	fd.Name = name

	switch typeStr[0] {
	case 'C':
		fd.Type = dbf.FieldTypeChar
		fd.Length = byte(width)
	case 'N':
		fd.Type = dbf.FieldTypeNumeric
		fd.Length = byte(width)
		fd.DecimalCount = byte(dec)
	case 'L':
		fd.Type = dbf.FieldTypeLogical
		fd.Length = 1
	default:
		return dbf.FieldDescriptor{}, fmt.Errorf("*** Invalid field type")
	}

	return fd, nil
}

func handleQuestion(ctx *context.Context, cmd Command) error {
	arg := strings.TrimSpace(cmd.Args)
	if arg == "" {
		if cmd.Verb == "?" {
			fmt.Fprintln(ctx.Stdout)
		}
		return nil
	}

	lexer := expr.NewLexer(arg)
	parser := expr.NewParser(lexer)
	exp := parser.ParseExpression()
	if len(parser.Errors()) > 0 {
		return fmt.Errorf("*** Syntax error: %s", strings.Join(parser.Errors(), "; "))
	}

	env := newReplEnvironment(ctx)
	obj, err := expr.Eval(exp, env)
	if err != nil {
		return fmt.Errorf("*** Evaluation error: %w", err)
	}

	if cmd.Verb == "?" {
		fmt.Fprintln(ctx.Stdout, obj.String())
	} else {
		fmt.Fprint(ctx.Stdout, obj.String())
	}

	return nil
}
