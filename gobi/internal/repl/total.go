package repl

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

type totalOptions struct {
	keyField  string
	sumFields []string
}

type totalFieldRole int

const (
	totalRoleKey totalFieldRole = iota
	totalRoleSum
	totalRoleCopy
)

type totalFieldMapping struct {
	Descriptor dbf.FieldDescriptor
	SrcIdx     int
	Role       totalFieldRole
}

type totalGroupState struct {
	keyValue interface{}
	active   bool
	sums     map[int]float64
	first    map[int]interface{}
}

func handleTotal(ctx *context.Context, cmd Command) error {
	if cmd.ToClause == "" {
		return fmt.Errorf("*** TOTAL requires TO <filename>")
	}

	opts, err := parseTotalOptions(cmd.Args)
	if err != nil {
		return err
	}

	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	forExp, _, err := parseForWhileClauses(cmd)
	if err != nil {
		return err
	}

	srcTbl := area.Table
	rseeker, ok := srcTbl.Underlying().(io.ReadSeeker)
	if !ok {
		return fmt.Errorf("*** Underlying database stream is not seekable")
	}

	mappings, sumSrcIdxs, err := buildTotalFieldMappings(srcTbl, ctx, cmd.ToClause, opts)
	if err != nil {
		return err
	}

	outFields := make([]dbf.FieldDescriptor, len(mappings))
	for i, m := range mappings {
		outFields[i] = m.Descriptor
	}

	filePath := resolveDBFFilePath(ctx, cmd.ToClause)
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("*** Could not create output file: %w", err)
	}
	defer f.Close()

	dstTbl, err := dbf.Create(f, outFields)
	if err != nil {
		return fmt.Errorf("*** Error creating output database: %w", err)
	}

	_, keySrcIdx := srcTbl.FieldByName(opts.keyField)
	if keySrcIdx < 0 {
		return fmt.Errorf("*** Unknown key field: %s", opts.keyField)
	}

	env := newReplEnvironment(ctx)
	group := &totalGroupState{
		sums:  make(map[int]float64),
		first: make(map[int]interface{}),
	}
	written := 0

	flush := func() error {
		if !group.active {
			return nil
		}
		values, err := totalGroupValues(mappings, group)
		if err != nil {
			return err
		}
		rec, err := dbf.NewRecord(dstTbl, false, values)
		if err != nil {
			return fmt.Errorf("*** Error building total record: %w", err)
		}
		if _, err := dstTbl.AppendRecord(f, rec); err != nil {
			return fmt.Errorf("*** Error writing total record: %w", err)
		}
		written++
		return nil
	}

	recCount := int(srcTbl.Header.RecordCount)
	for i := 0; i < recCount; i++ {
		rec, err := srcTbl.ReadRecordAt(rseeker, i)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("*** Error reading record %d: %w", i+1, err)
		}
		if rec.Deleted {
			continue
		}

		area.RecordNo = i
		area.ActiveRecord = rec

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

		keyVal, err := rec.DecodeField(srcTbl, keySrcIdx)
		if err != nil {
			return fmt.Errorf("*** Error decoding key field in record %d: %w", i+1, err)
		}

		if group.active {
			cmp := compareUpdateValues(group.keyValue, keyVal)
			if cmp != 0 {
				if err := flush(); err != nil {
					return err
				}
				group = &totalGroupState{
					sums:  make(map[int]float64),
					first: make(map[int]interface{}),
				}
			}
		}

		if !group.active {
			group.active = true
			group.keyValue = keyVal
		}

		for srcIdx := range sumSrcIdxs {
			val, err := rec.DecodeField(srcTbl, srcIdx)
			if err != nil {
				return fmt.Errorf("*** Error decoding numeric field in record %d: %w", i+1, err)
			}
			num, err := valueToFloat64(val)
			if err != nil {
				return fmt.Errorf("*** Error totaling field %s in record %d: %w", srcTbl.Fields[srcIdx].Name, i+1, err)
			}
			group.sums[srcIdx] += num
		}

		for _, mapping := range mappings {
			if mapping.Role != totalRoleCopy || mapping.SrcIdx < 0 {
				continue
			}
			if _, seen := group.first[mapping.SrcIdx]; seen {
				continue
			}
			val, err := rec.DecodeField(srcTbl, mapping.SrcIdx)
			if err != nil {
				return fmt.Errorf("*** Error decoding field in record %d: %w", i+1, err)
			}
			group.first[mapping.SrcIdx] = val
		}
	}

	if err := flush(); err != nil {
		return err
	}

	if err := dstTbl.Close(); err != nil {
		return fmt.Errorf("*** Error closing output database: %w", err)
	}

	talkPrint(ctx, "%05d RECORDS COPIED\r\n", written)

	return nil
}

func parseTotalOptions(args string) (totalOptions, error) {
	opts := totalOptions{}
	args = strings.TrimSpace(args)
	if args == "" {
		return opts, fmt.Errorf("*** TOTAL requires ON <keyfield>")
	}

	tokens := tokenize(args)
	i := 0
	for i < len(tokens) {
		switch strings.ToUpper(tokens[i]) {
		case "ON":
			if opts.keyField != "" {
				return opts, fmt.Errorf("*** TOTAL allows only one ON phrase")
			}
			if i+1 >= len(tokens) {
				return opts, fmt.Errorf("*** TOTAL ON requires a key field")
			}
			opts.keyField = strings.ToUpper(strings.TrimSpace(tokens[i+1]))
			i += 2
		case "FIELD":
			fields, next, err := parseTotalFieldList(tokens, i+1)
			if err != nil {
				return opts, err
			}
			opts.sumFields = append(opts.sumFields, fields...)
			i = next
		default:
			return opts, fmt.Errorf("*** Unexpected TOTAL option: %s", tokens[i])
		}
	}

	if opts.keyField == "" {
		return opts, fmt.Errorf("*** TOTAL requires ON <keyfield>")
	}

	return opts, nil
}

func parseTotalFieldList(tokens []string, start int) ([]string, int, error) {
	if start >= len(tokens) {
		return nil, start, fmt.Errorf("*** TOTAL FIELD requires a field list")
	}

	end := start
	for end < len(tokens) {
		switch strings.ToUpper(tokens[end]) {
		case "ON", "FIELD", "FOR":
			break
		default:
			end++
			continue
		}
		break
	}

	if end == start {
		return nil, start, fmt.Errorf("*** TOTAL FIELD requires a field list")
	}

	parts := splitCommaOutsideParens(joinTokens(tokens[start:end]))
	fields := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.ToUpper(strings.TrimSpace(part))
		if name == "" {
			continue
		}
		fields = append(fields, name)
	}
	if len(fields) == 0 {
		return nil, start, fmt.Errorf("*** TOTAL FIELD requires a field list")
	}

	return fields, end, nil
}

func buildTotalFieldMappings(srcTbl *dbf.Table, ctx *context.Context, toClause string, opts totalOptions) ([]totalFieldMapping, map[int]bool, error) {
	filePath := resolveDBFFilePath(ctx, toClause)
	var outFields []dbf.FieldDescriptor

	if _, err := os.Stat(filePath); err == nil {
		f, err := os.Open(filePath)
		if err != nil {
			return nil, nil, fmt.Errorf("*** Could not open output file: %w", err)
		}
		existing, err := dbf.Open(f)
		f.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("*** Error reading output database structure: %w", err)
		}
		outFields = existing.Fields
	} else if len(opts.sumFields) == 0 {
		outFields = append(outFields, srcTbl.Fields...)
	} else {
		built, err := buildTotalOutputFields(srcTbl, opts.keyField, opts.sumFields)
		if err != nil {
			return nil, nil, err
		}
		outFields = built
	}

	sumSrcIdxs, err := resolveTotalSumFields(srcTbl, opts)
	if err != nil {
		return nil, nil, err
	}

	mappings := make([]totalFieldMapping, 0, len(outFields))
	for _, outFD := range outFields {
		srcIdx := -1
		if idx := fieldIndexByName(srcTbl, outFD.Name); idx >= 0 {
			srcIdx = idx
		}

		role := totalRoleCopy
		if strings.EqualFold(outFD.Name, opts.keyField) {
			role = totalRoleKey
		} else if sumSrcIdxs[srcIdx] {
			role = totalRoleSum
		}

		if role == totalRoleKey && srcIdx < 0 {
			return nil, nil, fmt.Errorf("*** Unknown key field in output structure: %s", outFD.Name)
		}
		if role == totalRoleSum && srcIdx < 0 {
			return nil, nil, fmt.Errorf("*** Unknown total field in source database: %s", outFD.Name)
		}

		mappings = append(mappings, totalFieldMapping{
			Descriptor: outFD,
			SrcIdx:     srcIdx,
			Role:       role,
		})
	}

	return mappings, sumSrcIdxs, nil
}

func buildTotalOutputFields(srcTbl *dbf.Table, keyField string, sumFieldNames []string) ([]dbf.FieldDescriptor, error) {
	keyFD, keyIdx := srcTbl.FieldByName(keyField)
	if keyFD == nil {
		return nil, fmt.Errorf("*** Unknown key field: %s", keyField)
	}

	out := []dbf.FieldDescriptor{*keyFD}
	seen := map[string]bool{keyField: true}

	for _, name := range sumFieldNames {
		if seen[name] {
			continue
		}
		fd, _ := srcTbl.FieldByName(name)
		if fd == nil {
			return nil, fmt.Errorf("*** Unknown TOTAL field: %s", name)
		}
		if fd.Type != dbf.FieldTypeNumeric {
			return nil, fmt.Errorf("*** TOTAL FIELD must be numeric: %s", name)
		}
		out = append(out, *fd)
		seen[name] = true
	}

	_ = keyIdx
	return out, nil
}

func resolveTotalSumFields(srcTbl *dbf.Table, opts totalOptions) (map[int]bool, error) {
	sumIdxs := make(map[int]bool)

	if len(opts.sumFields) == 0 {
		for i, fd := range srcTbl.Fields {
			if fd.Type == dbf.FieldTypeNumeric {
				sumIdxs[i] = true
			}
		}
		return sumIdxs, nil
	}

	for _, name := range opts.sumFields {
		fd, idx := srcTbl.FieldByName(name)
		if fd == nil {
			return nil, fmt.Errorf("*** Unknown TOTAL field: %s", name)
		}
		if fd.Type != dbf.FieldTypeNumeric {
			return nil, fmt.Errorf("*** TOTAL FIELD must be numeric: %s", name)
		}
		sumIdxs[idx] = true
	}

	return sumIdxs, nil
}

func fieldIndexByName(tbl *dbf.Table, name string) int {
	_, idx := tbl.FieldByName(strings.ToUpper(name))
	return idx
}

func totalGroupValues(mappings []totalFieldMapping, group *totalGroupState) ([]interface{}, error) {
	values := make([]interface{}, len(mappings))
	for i, mapping := range mappings {
		switch mapping.Role {
		case totalRoleKey:
			values[i] = group.keyValue
		case totalRoleSum:
			values[i] = group.sums[mapping.SrcIdx]
		case totalRoleCopy:
			if mapping.SrcIdx < 0 {
				values[i] = ""
				continue
			}
			val, ok := group.first[mapping.SrcIdx]
			if !ok {
				switch mapping.Descriptor.Type {
				case dbf.FieldTypeNumeric:
					values[i] = float64(0)
				case dbf.FieldTypeLogical:
					values[i] = false
				default:
					values[i] = ""
				}
				continue
			}
			values[i] = val
		default:
			return nil, fmt.Errorf("*** Invalid total field mapping")
		}
	}
	return values, nil
}
