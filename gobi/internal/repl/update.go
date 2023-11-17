package repl

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

type updateOptions struct {
	keyField      string
	addFields     []string
	replaceFields []string
	fromFile      string
}

func handleUpdate(ctx *context.Context, cmd Command) error {
	opts, err := parseUpdateOptions(cmd)
	if err != nil {
		return err
	}

	primaryArea := ctx.WorkAreas[context.Primary]
	if primaryArea == nil || primaryArea.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	wseeker, ok := primaryArea.Table.Underlying().(io.ReadWriteSeeker)
	if !ok {
		return fmt.Errorf("*** Database file is not writable")
	}

	srcTbl, srcClose, err := openUpdateSource(ctx, opts)
	if err != nil {
		return err
	}
	if srcClose != nil {
		defer srcClose()
	}

	dstTbl := primaryArea.Table
	if err := validateUpdateFields(dstTbl, srcTbl, opts); err != nil {
		return err
	}

	srcSeeker, ok := srcTbl.Underlying().(io.ReadSeeker)
	if !ok {
		return fmt.Errorf("*** Source database stream is not seekable")
	}

	updated, lastRecNo, lastRec, err := mergeUpdateRecords(primaryArea, dstTbl, wseeker, srcTbl, srcSeeker, opts)
	if err != nil {
		return err
	}

	if lastRec != nil {
		primaryArea.RecordNo = lastRecNo
		primaryArea.ActiveRecord = lastRec
	}

	talkPrint(ctx, "%05d RECORDS UPDATED\r\n", updated)

	return nil
}

func openUpdateSource(ctx *context.Context, opts updateOptions) (*dbf.Table, func(), error) {
	if opts.fromFile != "" {
		filePath := resolveDBFFilePath(ctx, opts.fromFile)
		f, err := os.Open(filePath)
		if err != nil {
			return nil, nil, fmt.Errorf("*** Could not open source file: %w", err)
		}
		tbl, err := dbf.Open(f)
		if err != nil {
			f.Close()
			return nil, nil, fmt.Errorf("*** Error reading source database: %w", err)
		}
		return tbl, func() { tbl.Close() }, nil
	}

	secondary := ctx.WorkAreas[context.Secondary]
	if secondary == nil || secondary.Table == nil {
		return nil, nil, fmt.Errorf("*** No secondary database file is in use")
	}

	return secondary.Table, nil, nil
}

func parseUpdateOptions(cmd Command) (updateOptions, error) {
	opts := updateOptions{
		fromFile: strings.TrimSpace(cmd.FromClause),
	}

	args := strings.TrimSpace(cmd.Args)
	if args == "" && opts.fromFile == "" {
		return opts, fmt.Errorf("*** UPDATE requires ON <keyfield>")
	}

	tokens := tokenize(args)
	i := 0
	for i < len(tokens) {
		upper := strings.ToUpper(tokens[i])
		switch upper {
		case "ON":
			if opts.keyField != "" {
				return opts, fmt.Errorf("*** UPDATE allows only one ON phrase")
			}
			if i+1 >= len(tokens) {
				return opts, fmt.Errorf("*** UPDATE ON requires a key field")
			}
			opts.keyField = strings.ToUpper(strings.TrimSpace(tokens[i+1]))
			i += 2
		case "ADD":
			fields, next, err := parseUpdateFieldList(tokens, i+1)
			if err != nil {
				return opts, err
			}
			opts.addFields = append(opts.addFields, fields...)
			i = next
		case "REPLACE":
			fields, next, err := parseUpdateFieldList(tokens, i+1)
			if err != nil {
				return opts, err
			}
			opts.replaceFields = append(opts.replaceFields, fields...)
			i = next
		default:
			return opts, fmt.Errorf("*** Unexpected UPDATE option: %s", tokens[i])
		}
	}

	if opts.keyField == "" {
		return opts, fmt.Errorf("*** UPDATE requires ON <keyfield>")
	}
	if len(opts.addFields) == 0 && len(opts.replaceFields) == 0 {
		return opts, fmt.Errorf("*** UPDATE requires ADD or REPLACE field list")
	}

	return opts, nil
}

func parseUpdateFieldList(tokens []string, start int) ([]string, int, error) {
	if start >= len(tokens) {
		return nil, start, fmt.Errorf("*** UPDATE field list is missing")
	}

	end := start
	for end < len(tokens) {
		switch strings.ToUpper(tokens[end]) {
		case "ON", "ADD", "REPLACE":
			break
		default:
			end++
			continue
		}
		break
	}

	if end == start {
		return nil, start, fmt.Errorf("*** UPDATE field list is missing")
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
		return nil, start, fmt.Errorf("*** UPDATE field list is missing")
	}

	return fields, end, nil
}

func validateUpdateFields(dstTbl, srcTbl *dbf.Table, opts updateOptions) error {
	dstKey, dstKeyIdx := dstTbl.FieldByName(opts.keyField)
	if dstKey == nil {
		return fmt.Errorf("*** Unknown key field in primary database: %s", opts.keyField)
	}
	srcKey, srcKeyIdx := srcTbl.FieldByName(opts.keyField)
	if srcKey == nil {
		return fmt.Errorf("*** Unknown key field in source database: %s", opts.keyField)
	}
	if dstKey.Length != srcKey.Length {
		return fmt.Errorf("*** KEYS ARE NOT THE SAME LENGTH")
	}
	_ = dstKeyIdx
	_ = srcKeyIdx

	for _, name := range opts.addFields {
		dstFD, _ := dstTbl.FieldByName(name)
		srcFD, _ := srcTbl.FieldByName(name)
		if dstFD == nil {
			return fmt.Errorf("*** Unknown ADD field in primary database: %s", name)
		}
		if srcFD == nil {
			return fmt.Errorf("*** Unknown ADD field in source database: %s", name)
		}
		if dstFD.Type != dbf.FieldTypeNumeric || srcFD.Type != dbf.FieldTypeNumeric {
			return fmt.Errorf("*** UPDATE ADD field must be numeric: %s", name)
		}
	}

	for _, name := range opts.replaceFields {
		if _, idx := dstTbl.FieldByName(name); idx < 0 {
			return fmt.Errorf("*** Unknown REPLACE field in primary database: %s", name)
		}
		if _, idx := srcTbl.FieldByName(name); idx < 0 {
			return fmt.Errorf("*** Unknown REPLACE field in source database: %s", name)
		}
	}

	return nil
}

func mergeUpdateRecords(primaryArea *context.WorkArea, dstTbl *dbf.Table, dstSeeker io.ReadWriteSeeker, srcTbl *dbf.Table, srcSeeker io.ReadSeeker, opts updateOptions) (int, int, *dbf.Record, error) {
	_, dstKeyIdx := dstTbl.FieldByName(opts.keyField)
	_, srcKeyIdx := srcTbl.FieldByName(opts.keyField)

	dstCount := int(dstTbl.Header.RecordCount)
	srcCount := int(srcTbl.Header.RecordCount)
	dstIdx := 0
	srcIdx := 0
	updated := 0
	lastRecNo := -1
	var lastRec *dbf.Record

	for dstIdx < dstCount && srcIdx < srcCount {
		dstRec, err := dstTbl.ReadRecordAt(dstSeeker, dstIdx)
		if err != nil {
			return updated, lastRecNo, lastRec, fmt.Errorf("*** Error reading primary record %d: %w", dstIdx+1, err)
		}
		srcRec, err := srcTbl.ReadRecordAt(srcSeeker, srcIdx)
		if err != nil {
			return updated, lastRecNo, lastRec, fmt.Errorf("*** Error reading source record %d: %w", srcIdx+1, err)
		}

		if dstRec.Deleted {
			dstIdx++
			continue
		}
		if srcRec.Deleted {
			srcIdx++
			continue
		}

		cmp, err := compareRecordKeys(dstTbl, dstRec, dstKeyIdx, srcTbl, srcRec, srcKeyIdx)
		if err != nil {
			return updated, lastRecNo, lastRec, err
		}

		switch {
		case cmp < 0:
			dstIdx++
		case cmp > 0:
			srcIdx++
		default:
			updatedRec, err := applyUpdateRecord(dstTbl, dstRec, srcTbl, srcRec, opts)
			if err != nil {
				return updated, lastRecNo, lastRec, fmt.Errorf("*** Error updating record %d: %w", dstIdx+1, err)
			}
			if err := dstTbl.WriteRecordAt(dstSeeker, dstIdx, updatedRec); err != nil {
				return updated, lastRecNo, lastRec, fmt.Errorf("*** Error writing record %d: %w", dstIdx+1, err)
			}

			updated++
			lastRecNo = dstIdx
			lastRec = updatedRec
			primaryArea.RecordNo = dstIdx
			primaryArea.ActiveRecord = updatedRec
			dstIdx++
			srcIdx++
		}
	}

	return updated, lastRecNo, lastRec, nil
}

func compareRecordKeys(dstTbl *dbf.Table, dstRec *dbf.Record, dstKeyIdx int, srcTbl *dbf.Table, srcRec *dbf.Record, srcKeyIdx int) (int, error) {
	dstVal, err := dstRec.DecodeField(dstTbl, dstKeyIdx)
	if err != nil {
		return 0, err
	}
	srcVal, err := srcRec.DecodeField(srcTbl, srcKeyIdx)
	if err != nil {
		return 0, err
	}
	return compareUpdateValues(dstVal, srcVal), nil
}

func compareUpdateValues(a, b interface{}) int {
	as := fmt.Sprintf("%v", a)
	bs := fmt.Sprintf("%v", b)

	if af, okA := a.(float64); okA {
		if bf, okB := b.(float64); okB {
			switch {
			case af < bf:
				return -1
			case af > bf:
				return 1
			default:
				return 0
			}
		}
	}

	as = strings.ToUpper(strings.TrimSpace(as))
	bs = strings.ToUpper(strings.TrimSpace(bs))
	switch strings.Compare(as, bs) {
	case -1:
		return -1
	case 1:
		return 1
	default:
		return 0
	}
}

func applyUpdateRecord(dstTbl *dbf.Table, dstRec *dbf.Record, srcTbl *dbf.Table, srcRec *dbf.Record, opts updateOptions) (*dbf.Record, error) {
	rec := dstRec
	var err error

	for _, name := range opts.addFields {
		_, dstIdx := dstTbl.FieldByName(name)
		_, srcIdx := srcTbl.FieldByName(name)
		rec, err = addRecordField(dstTbl, srcTbl, rec, srcRec, dstIdx, srcIdx)
		if err != nil {
			return nil, err
		}
	}

	for _, name := range opts.replaceFields {
		_, dstIdx := dstTbl.FieldByName(name)
		_, srcIdx := srcTbl.FieldByName(name)
		val, err := srcRec.DecodeField(srcTbl, srcIdx)
		if err != nil {
			return nil, err
		}
		rec, err = replaceRecordField(dstTbl, rec, dstIdx, val)
		if err != nil {
			return nil, err
		}
	}

	return rec, nil
}

func addRecordField(dstTbl, srcTbl *dbf.Table, dstRec, srcRec *dbf.Record, dstIdx, srcIdx int) (*dbf.Record, error) {
	dstVal, err := dstRec.DecodeField(dstTbl, dstIdx)
	if err != nil {
		return nil, err
	}
	srcVal, err := srcRec.DecodeField(srcTbl, srcIdx)
	if err != nil {
		return nil, err
	}

	dstNum, err := valueToFloat64(dstVal)
	if err != nil {
		return nil, err
	}
	srcNum, err := valueToFloat64(srcVal)
	if err != nil {
		return nil, err
	}

	return replaceRecordField(dstTbl, dstRec, dstIdx, dstNum+srcNum)
}

func valueToFloat64(val interface{}) (float64, error) {
	switch v := val.(type) {
	case float64:
		return v, nil
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, nil
		}
		var f float64
		_, err := fmt.Sscanf(trimmed, "%f", &f)
		if err != nil {
			return 0, fmt.Errorf("invalid numeric value %q", v)
		}
		return f, nil
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("invalid numeric type %T", val)
	}
}
