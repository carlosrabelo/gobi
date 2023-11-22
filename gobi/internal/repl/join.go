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

const maxJoinRecords = 65534

type joinFieldSource int

const (
	joinFieldPrimary joinFieldSource = iota
	joinFieldSecondary
)

type joinFieldSpec struct {
	Descriptor dbf.FieldDescriptor
	Source     joinFieldSource
	Index      int
}

func handleJoin(ctx *context.Context, cmd Command) error {
	if cmd.ToClause == "" {
		return fmt.Errorf("*** JOIN requires TO <filename>")
	}
	if cmd.ForClause == "" {
		return fmt.Errorf("*** JOIN requires FOR <expression>")
	}

	primary := ctx.WorkAreas[context.Primary]
	secondary := ctx.WorkAreas[context.Secondary]
	if primary == nil || primary.Table == nil {
		return fmt.Errorf("*** No primary database file is in use")
	}
	if secondary == nil || secondary.Table == nil {
		return fmt.Errorf("*** No secondary database file is in use")
	}

	forExp, err := parseJoinForClause(cmd.ForClause)
	if err != nil {
		return err
	}

	fieldSpecs, err := parseJoinFields(primary.Table, secondary.Table, cmd.Args)
	if err != nil {
		return err
	}

	outFields := make([]dbf.FieldDescriptor, len(fieldSpecs))
	for i, spec := range fieldSpecs {
		outFields[i] = spec.Descriptor
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

	primarySeeker, ok := primary.Table.Underlying().(io.ReadSeeker)
	if !ok {
		return fmt.Errorf("*** Primary database stream is not seekable")
	}
	secondarySeeker, ok := secondary.Table.Underlying().(io.ReadSeeker)
	if !ok {
		return fmt.Errorf("*** Secondary database stream is not seekable")
	}

	env := newJoinEnvironment(ctx, primary, secondary)
	joined := 0

	for pIdx := 0; pIdx < int(primary.Table.Header.RecordCount); pIdx++ {
		pRec, err := primary.Table.ReadRecordAt(primarySeeker, pIdx)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("*** Error reading primary record %d: %w", pIdx+1, err)
		}
		if pRec.Deleted {
			continue
		}

		env.setRecords(pRec, pIdx)

		for sIdx := 0; sIdx < int(secondary.Table.Header.RecordCount); sIdx++ {
			sRec, err := secondary.Table.ReadRecordAt(secondarySeeker, sIdx)
			if err != nil {
				if err == io.EOF {
					break
				}
				return fmt.Errorf("*** Error reading secondary record %d: %w", sIdx+1, err)
			}
			if sRec.Deleted {
				continue
			}

			env.setSecondaryRecord(sRec, sIdx)

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

			if joined >= maxJoinRecords {
				return fmt.Errorf("*** JOIN ATTEMPTED TO GENERATE MORE THAN 65,534 RECORDS")
			}

			values, err := buildJoinRecordValues(fieldSpecs, primary.Table, pRec, secondary.Table, sRec)
			if err != nil {
				return fmt.Errorf("*** Error building joined record: %w", err)
			}

			newRec, err := dbf.NewRecord(dstTbl, false, values)
			if err != nil {
				return fmt.Errorf("*** Error building joined record: %w", err)
			}

			if _, err := dstTbl.AppendRecord(f, newRec); err != nil {
				return fmt.Errorf("*** Error writing joined record: %w", err)
			}

			joined++
		}
	}

	if err := dstTbl.Close(); err != nil {
		return fmt.Errorf("*** Error closing output database: %w", err)
	}

	talkPrint(ctx, "%05d RECORDS JOINED\r\n", joined)

	return nil
}

func parseJoinForClause(forClause string) (expr.Expression, error) {
	lexer := expr.NewLexer(forClause)
	parser := expr.NewParser(lexer)
	exp := parser.ParseExpression()
	if len(parser.Errors()) > 0 {
		return nil, fmt.Errorf("*** Syntax error in FOR clause: %s", strings.Join(parser.Errors(), "; "))
	}
	return exp, nil
}

func parseJoinFields(primary, secondary *dbf.Table, args string) ([]joinFieldSpec, error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return defaultJoinFieldSpecs(primary, secondary)
	}

	tokens := tokenize(args)
	start := 0
	if len(tokens) > 0 && strings.ToUpper(tokens[0]) == "FIELD" {
		start = 1
	}
	if start >= len(tokens) {
		return nil, fmt.Errorf("*** JOIN FIELD requires a field list")
	}

	parts := splitCommaOutsideParens(joinTokens(tokens[start:]))
	specs := make([]joinFieldSpec, 0, len(parts))
	for _, part := range parts {
		spec, err := resolveJoinFieldSpec(primary, secondary, part)
		if err != nil {
			return nil, err
		}
		specs = append(specs, spec)
	}

	if len(specs) == 0 {
		return nil, fmt.Errorf("*** JOIN FIELD requires a field list")
	}

	return specs, nil
}

func defaultJoinFieldSpecs(primary, secondary *dbf.Table) ([]joinFieldSpec, error) {
	specs := make([]joinFieldSpec, 0, len(primary.Fields)+len(secondary.Fields))
	seen := make(map[string]bool)

	for i, fd := range primary.Fields {
		specs = append(specs, joinFieldSpec{
			Descriptor: fd,
			Source:     joinFieldPrimary,
			Index:      i,
		})
		seen[fd.Name] = true
		if len(specs) >= 32 {
			return specs, nil
		}
	}

	for i, fd := range secondary.Fields {
		if seen[fd.Name] {
			continue
		}
		specs = append(specs, joinFieldSpec{
			Descriptor: fd,
			Source:     joinFieldSecondary,
			Index:      i,
		})
		if len(specs) >= 32 {
			break
		}
	}

	return specs, nil
}

func resolveJoinFieldSpec(primary, secondary *dbf.Table, token string) (joinFieldSpec, error) {
	source := joinFieldPrimary
	name := strings.ToUpper(strings.TrimSpace(token))

	switch {
	case strings.HasPrefix(name, "P."):
		source = joinFieldPrimary
		name = strings.TrimSpace(name[2:])
	case strings.HasPrefix(name, "S."):
		source = joinFieldSecondary
		name = strings.TrimSpace(name[2:])
	}

	if name == "" {
		return joinFieldSpec{}, fmt.Errorf("*** JOIN FIELD requires a field name")
	}

	tbl := primary
	if source == joinFieldSecondary {
		tbl = secondary
	}

	fd, idx := tbl.FieldByName(name)
	if fd == nil {
		return joinFieldSpec{}, fmt.Errorf("*** Unknown field: %s", name)
	}

	return joinFieldSpec{
		Descriptor: *fd,
		Source:     source,
		Index:      idx,
	}, nil
}

func buildJoinRecordValues(specs []joinFieldSpec, primaryTbl *dbf.Table, primaryRec *dbf.Record, secondaryTbl *dbf.Table, secondaryRec *dbf.Record) ([]interface{}, error) {
	values := make([]interface{}, len(specs))
	for i, spec := range specs {
		switch spec.Source {
		case joinFieldPrimary:
			val, err := primaryRec.DecodeField(primaryTbl, spec.Index)
			if err != nil {
				return nil, err
			}
			values[i] = val
		case joinFieldSecondary:
			val, err := secondaryRec.DecodeField(secondaryTbl, spec.Index)
			if err != nil {
				return nil, err
			}
			values[i] = val
		default:
			return nil, fmt.Errorf("*** Invalid join field source")
		}
	}
	return values, nil
}

type joinEnvironment struct {
	ctx          *context.Context
	primary      *context.WorkArea
	secondary    *context.WorkArea
	primaryRec   *dbf.Record
	secondaryRec *dbf.Record
}

func newJoinEnvironment(ctx *context.Context, primary, secondary *context.WorkArea) *joinEnvironment {
	return &joinEnvironment{
		ctx:       ctx,
		primary:   primary,
		secondary: secondary,
	}
}

func (env *joinEnvironment) setRecords(primaryRec *dbf.Record, primaryIdx int) {
	env.primaryRec = primaryRec
	env.primary.RecordNo = primaryIdx
	env.primary.ActiveRecord = primaryRec
}

func (env *joinEnvironment) setSecondaryRecord(secondaryRec *dbf.Record, secondaryIdx int) {
	env.secondaryRec = secondaryRec
	env.secondary.RecordNo = secondaryIdx
	env.secondary.ActiveRecord = secondaryRec
}

func (env *joinEnvironment) GetVariable(name string) (expr.Object, bool) {
	repl := newReplEnvironment(env.ctx)
	return repl.GetVariable(name)
}

func (env *joinEnvironment) GetField(name string) (expr.Object, bool) {
	name = strings.ToUpper(name)
	switch {
	case strings.HasPrefix(name, "P."):
		return env.fieldFromArea(env.primary, env.primaryRec, strings.TrimSpace(name[2:]))
	case strings.HasPrefix(name, "S."):
		return env.fieldFromArea(env.secondary, env.secondaryRec, strings.TrimSpace(name[2:]))
	}

	if obj, ok := env.fieldFromArea(env.primary, env.primaryRec, name); ok {
		return obj, true
	}
	return env.fieldFromArea(env.secondary, env.secondaryRec, name)
}

func (env *joinEnvironment) fieldFromArea(area *context.WorkArea, rec *dbf.Record, name string) (expr.Object, bool) {
	if area == nil || area.Table == nil || rec == nil {
		return nil, false
	}
	fd, idx := area.Table.FieldByName(name)
	if fd == nil {
		return nil, false
	}
	val, err := rec.DecodeField(area.Table, idx)
	if err != nil {
		return nil, false
	}
	return objectFromValue(val), true
}

func (env *joinEnvironment) CallFunction(name string, args []expr.Object) (expr.Object, error) {
	repl := newReplEnvironment(env.ctx)
	return repl.CallFunction(name, args)
}

func objectFromValue(val interface{}) expr.Object {
	switch v := val.(type) {
	case string:
		return &expr.StringObject{Value: v}
	case float64:
		return &expr.NumberObject{Value: v}
	case bool:
		return &expr.BooleanObject{Value: v}
	default:
		return &expr.StringObject{Value: fmt.Sprintf("%v", v)}
	}
}
