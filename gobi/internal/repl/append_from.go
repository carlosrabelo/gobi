package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

type appendImportFormat int

const (
	appendFormatDBF appendImportFormat = iota
	appendFormatSDF
	appendFormatDelimited
)

type delimitedImportOptions struct {
	delimiter rune
	quoteChar rune // 0 means fields are not quoted
}

func handleAppendFrom(ctx *context.Context, cmd Command) error {
	if cmd.FromClause == "" {
		return fmt.Errorf("*** APPEND requires FROM <filename>")
	}

	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	wseeker, ok := area.Table.Underlying().(io.ReadWriteSeeker)
	if !ok {
		return fmt.Errorf("*** Database file is not writable")
	}

	format, delimOpts, err := parseAppendFromOptions(cmd.Args)
	if err != nil {
		return err
	}

	forExp, whileExp, err := parseForWhileClauses(cmd)
	if err != nil {
		return err
	}

	var appended int
	var lastRec *dbf.Record
	var lastRecNo int

	switch format {
	case appendFormatDBF:
		appended, lastRecNo, lastRec, err = appendFromDBF(ctx, area, wseeker, cmd.FromClause, forExp, whileExp)
	case appendFormatSDF:
		appended, lastRecNo, lastRec, err = appendFromSDF(ctx, area, wseeker, cmd.FromClause, forExp, whileExp)
	case appendFormatDelimited:
		appended, lastRecNo, lastRec, err = appendFromDelimited(ctx, area, wseeker, cmd.FromClause, delimOpts, forExp, whileExp)
	default:
		return fmt.Errorf("*** Unsupported APPEND FROM format")
	}
	if err != nil {
		return err
	}

	if lastRec != nil {
		area.RecordNo = lastRecNo
		area.ActiveRecord = lastRec
	}

	talkPrint(ctx, "%05d RECORDS ADDED\r\n", appended)

	return nil
}

func appendFromDBF(ctx *context.Context, area *context.WorkArea, wseeker io.ReadWriteSeeker, filename string, forExp, whileExp expr.Expression) (int, int, *dbf.Record, error) {
	filePath := resolveDBFFilePath(ctx, filename)
	f, err := os.Open(filePath)
	if err != nil {
		return 0, -1, nil, fmt.Errorf("*** Could not open source file: %w", err)
	}
	defer f.Close()

	srcTbl, err := dbf.Open(f)
	if err != nil {
		return 0, -1, nil, fmt.Errorf("*** Error reading source database: %w", err)
	}

	rseeker, ok := srcTbl.Underlying().(io.ReadSeeker)
	if !ok {
		return 0, -1, nil, fmt.Errorf("*** Source database stream is not seekable")
	}

	dstTbl := area.Table
	recCount := int(srcTbl.Header.RecordCount)
	env := newReplEnvironment(ctx)
	appended := 0
	lastRecNo := -1
	var lastRec *dbf.Record

	for i := 0; i < recCount; i++ {
		rec, err := srcTbl.ReadRecordAt(rseeker, i)
		if err != nil {
			if err == io.EOF {
				break
			}
			return appended, lastRecNo, lastRec, fmt.Errorf("*** Error reading source record %d: %w", i, err)
		}

		if rec.Deleted {
			continue
		}

		values, err := mapSourceRecordToDest(srcTbl, rec, dstTbl)
		if err != nil {
			return appended, lastRecNo, lastRec, fmt.Errorf("*** Error mapping source record %d: %w", i+1, err)
		}

		pending, err := dbf.NewRecord(dstTbl, false, values)
		if err != nil {
			return appended, lastRecNo, lastRec, fmt.Errorf("*** Error building appended record %d: %w", i+1, err)
		}

		area.ActiveRecord = pending

		if whileExp != nil {
			res, err := expr.Eval(whileExp, env)
			if err != nil {
				return appended, lastRecNo, lastRec, fmt.Errorf("*** Evaluation error in WHILE clause: %w", err)
			}
			boolRes, ok := res.(*expr.BooleanObject)
			if !ok {
				return appended, lastRecNo, lastRec, fmt.Errorf("*** WHILE clause must evaluate to a logical value")
			}
			if !boolRes.Value {
				break
			}
		}

		if forExp != nil {
			res, err := expr.Eval(forExp, env)
			if err != nil {
				return appended, lastRecNo, lastRec, fmt.Errorf("*** Evaluation error in FOR clause: %w", err)
			}
			boolRes, ok := res.(*expr.BooleanObject)
			if !ok {
				return appended, lastRecNo, lastRec, fmt.Errorf("*** FOR clause must evaluate to a logical value")
			}
			if !boolRes.Value {
				continue
			}
		}

		recNo, err := dstTbl.AppendRecord(wseeker, pending)
		if err != nil {
			return appended, lastRecNo, lastRec, fmt.Errorf("*** Error appending record %d: %w", i+1, err)
		}

		appended++
		lastRecNo = recNo
		lastRec = pending
	}

	return appended, lastRecNo, lastRec, nil
}

func appendFromSDF(ctx *context.Context, area *context.WorkArea, wseeker io.ReadWriteSeeker, filename string, forExp, whileExp expr.Expression) (int, int, *dbf.Record, error) {
	filePath := resolveTextImportPath(ctx, filename)
	f, err := os.Open(filePath)
	if err != nil {
		return 0, -1, nil, fmt.Errorf("*** Could not open source file: %w", err)
	}
	defer f.Close()

	return appendFromTextLines(ctx, area, wseeker, bufio.NewReader(f), forExp, whileExp, parseSDFLine)
}

func appendFromDelimited(ctx *context.Context, area *context.WorkArea, wseeker io.ReadWriteSeeker, filename string, opts delimitedImportOptions, forExp, whileExp expr.Expression) (int, int, *dbf.Record, error) {
	filePath := resolveTextImportPath(ctx, filename)
	f, err := os.Open(filePath)
	if err != nil {
		return 0, -1, nil, fmt.Errorf("*** Could not open source file: %w", err)
	}
	defer f.Close()

	parseLine := func(line string, tbl *dbf.Table) ([]interface{}, error) {
		return parseDelimitedLine(line, tbl, opts)
	}
	return appendFromTextLines(ctx, area, wseeker, bufio.NewReader(f), forExp, whileExp, parseLine)
}

type textLineParser func(line string, tbl *dbf.Table) ([]interface{}, error)

func appendFromTextLines(ctx *context.Context, area *context.WorkArea, wseeker io.ReadWriteSeeker, reader *bufio.Reader, forExp, whileExp expr.Expression, parseLine textLineParser) (int, int, *dbf.Record, error) {
	dstTbl := area.Table
	env := newReplEnvironment(ctx)
	appended := 0
	lastRecNo := -1
	var lastRec *dbf.Record
	lineNo := 0

	for {
		line, err := readImportTextLine(reader)
		if err != nil {
			if err == io.EOF {
				break
			}
			return appended, lastRecNo, lastRec, fmt.Errorf("*** Error reading source file: %w", err)
		}
		lineNo++

		if strings.TrimSpace(line) == "" {
			continue
		}

		values, err := parseLine(line, dstTbl)
		if err != nil {
			return appended, lastRecNo, lastRec, fmt.Errorf("*** Error parsing source line %d: %w", lineNo, err)
		}

		pending, err := dbf.NewRecord(dstTbl, false, values)
		if err != nil {
			return appended, lastRecNo, lastRec, fmt.Errorf("*** Error building appended record on line %d: %w", lineNo, err)
		}

		area.ActiveRecord = pending

		if whileExp != nil {
			res, err := expr.Eval(whileExp, env)
			if err != nil {
				return appended, lastRecNo, lastRec, fmt.Errorf("*** Evaluation error in WHILE clause: %w", err)
			}
			boolRes, ok := res.(*expr.BooleanObject)
			if !ok {
				return appended, lastRecNo, lastRec, fmt.Errorf("*** WHILE clause must evaluate to a logical value")
			}
			if !boolRes.Value {
				break
			}
		}

		if forExp != nil {
			res, err := expr.Eval(forExp, env)
			if err != nil {
				return appended, lastRecNo, lastRec, fmt.Errorf("*** Evaluation error in FOR clause: %w", err)
			}
			boolRes, ok := res.(*expr.BooleanObject)
			if !ok {
				return appended, lastRecNo, lastRec, fmt.Errorf("*** FOR clause must evaluate to a logical value")
			}
			if !boolRes.Value {
				continue
			}
		}

		recNo, err := dstTbl.AppendRecord(wseeker, pending)
		if err != nil {
			return appended, lastRecNo, lastRec, fmt.Errorf("*** Error appending record on line %d: %w", lineNo, err)
		}

		appended++
		lastRecNo = recNo
		lastRec = pending
	}

	return appended, lastRecNo, lastRec, nil
}

func readImportTextLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF && len(line) > 0 {
			return strings.TrimRight(line, "\r\n"), io.EOF
		}
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func mapSourceRecordToDest(srcTbl *dbf.Table, srcRec *dbf.Record, dstTbl *dbf.Table) ([]interface{}, error) {
	values := defaultFieldValues(dstTbl)
	for i, dfd := range dstTbl.Fields {
		_, srcIdx := srcTbl.FieldByName(dfd.Name)
		if srcIdx < 0 {
			continue
		}
		decoded, err := srcRec.DecodeField(srcTbl, srcIdx)
		if err != nil {
			return nil, err
		}
		values[i] = decoded
	}
	return values, nil
}

func defaultFieldValues(tbl *dbf.Table) []interface{} {
	values := make([]interface{}, len(tbl.Fields))
	for i, fd := range tbl.Fields {
		switch fd.Type {
		case dbf.FieldTypeChar, dbf.FieldTypeNumeric:
			values[i] = ""
		case dbf.FieldTypeLogical:
			values[i] = false
		default:
			values[i] = ""
		}
	}
	return values
}

func tableDataWidth(tbl *dbf.Table) int {
	width := 0
	for _, fd := range tbl.Fields {
		width += int(fd.Length)
	}
	return width
}

func parseSDFLine(line string, tbl *dbf.Table) ([]interface{}, error) {
	width := tableDataWidth(tbl)
	if len(line) < width {
		line += strings.Repeat(" ", width-len(line))
	} else if len(line) > width {
		line = line[:width]
	}

	values := defaultFieldValues(tbl)
	offset := 0
	for i, fd := range tbl.Fields {
		part := line[offset : offset+int(fd.Length)]
		offset += int(fd.Length)
		val, err := parseFieldInput(fd, part)
		if err != nil {
			return nil, err
		}
		values[i] = val
	}
	return values, nil
}

func parseDelimitedLine(line string, tbl *dbf.Table, opts delimitedImportOptions) ([]interface{}, error) {
	parts, err := splitDelimitedFields(line, opts, len(tbl.Fields))
	if err != nil {
		return nil, err
	}

	values := defaultFieldValues(tbl)
	for i, fd := range tbl.Fields {
		if i >= len(parts) {
			break
		}
		part := parts[i]
		if opts.quoteChar == 0 && fd.Type == dbf.FieldTypeChar {
			part = strings.TrimRight(part, " ")
		}
		if opts.quoteChar == 0 && fd.Type == dbf.FieldTypeNumeric {
			part = strings.TrimLeft(part, " ")
		}
		val, err := parseFieldInput(fd, part)
		if err != nil {
			return nil, err
		}
		values[i] = val
	}
	return values, nil
}

func splitDelimitedFields(line string, opts delimitedImportOptions, maxFields int) ([]string, error) {
	if opts.quoteChar == 0 {
		return splitUnquotedDelimited(line, opts.delimiter, maxFields), nil
	}

	var parts []string
	var cur strings.Builder
	inQuote := false
	i := 0
	for i < len(line) {
		ch := rune(line[i])
		if ch == opts.quoteChar {
			if inQuote {
				inQuote = false
				i++
				continue
			}
			if cur.Len() == 0 {
				inQuote = true
				i++
				continue
			}
		}
		if !inQuote && ch == opts.delimiter {
			parts = append(parts, cur.String())
			cur.Reset()
			if len(parts) >= maxFields {
				break
			}
			i++
			continue
		}
		cur.WriteRune(ch)
		i++
	}
	parts = append(parts, cur.String())
	return parts, nil
}

func splitUnquotedDelimited(line string, delimiter rune, maxFields int) []string {
	var parts []string
	start := 0
	for i, ch := range line {
		if ch == delimiter {
			parts = append(parts, line[start:i])
			start = i + 1
			if len(parts) >= maxFields-1 {
				break
			}
		}
	}
	parts = append(parts, line[start:])
	return parts
}

func parseAppendFromOptions(args string) (appendImportFormat, delimitedImportOptions, error) {
	opts := delimitedImportOptions{
		delimiter: ',',
		quoteChar: '\'',
	}

	args = strings.TrimSpace(args)
	if args == "" {
		return appendFormatDBF, opts, nil
	}

	tokens := tokenize(args)
	if len(tokens) == 0 {
		return appendFormatDBF, opts, nil
	}

	switch strings.ToUpper(tokens[0]) {
	case "SDF":
		return appendFormatSDF, opts, nil
	case "DELIMITED":
		if len(tokens) == 1 {
			return appendFormatDelimited, opts, nil
		}
		if strings.ToUpper(tokens[1]) != "WITH" {
			return appendFormatDBF, opts, fmt.Errorf("*** Unknown APPEND FROM option: %s", tokens[0])
		}
		if len(tokens) < 3 {
			return appendFormatDBF, opts, fmt.Errorf("*** DELIMITED WITH requires a delimiter character")
		}
		delim := unquoteDelimiterToken(tokens[2])
		if delim == "" {
			return appendFormatDBF, opts, fmt.Errorf("*** DELIMITED WITH requires a delimiter character")
		}
		opts.delimiter = []rune(delim)[0]
		if opts.delimiter == ',' {
			opts.quoteChar = 0
		}
		return appendFormatDelimited, opts, nil
	default:
		return appendFormatDBF, opts, fmt.Errorf("*** Unknown APPEND FROM option: %s", tokens[0])
	}
}

func unquoteDelimiterToken(token string) string {
	token = strings.TrimSpace(token)
	if len(token) >= 2 {
		if (token[0] == '\'' && token[len(token)-1] == '\'') ||
			(token[0] == '"' && token[len(token)-1] == '"') {
			return token[1 : len(token)-1]
		}
	}
	return token
}
