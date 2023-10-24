package repl

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
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
	if arg == "STRUCTURE" {
		return displayStructure(ctx)
	}
	return fmt.Errorf("*** DISPLAY: feature not yet implemented")
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
