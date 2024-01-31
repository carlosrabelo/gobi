package repl

import (
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func cmdREPLACE(ctx *context.Context, cmd command.Command) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	scope := parseScope(cmd)
	fieldName, exprSrc, err := parseReplaceArgs(scope.Rest)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	fi := fieldIndex(a, fieldName)
	if fi < 0 {
		return fail(ctx, "unknown field: %s", fieldName)
	}
	tree, errs := expr.Parse(exprSrc)
	if len(errs) > 0 || tree == nil {
		msg := "invalid expression"
		if len(errs) > 0 {
			msg = errs[0]
		}
		return fail(ctx, "%s", msg)
	}
	forExpr, err := parseOptionalLogic(cmd.Clauses["FOR"])
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	whileExpr, err := parseOptionalLogic(cmd.Clauses["WHILE"])
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}

	env := newEnv(ctx)
	f := a.Table.Header.Fields[fi]

	if scope.Kind == "" && forExpr == nil && whileExpr == nil {
		if !a.Landed {
			return fail(ctx, "no current record")
		}
		return replaceOne(ctx, env, tree, fi, f)
	}

	count := a.RecCount()
	start, end := scopeRange(count, a.Cursor, scope)
	if scope.Kind == "" {
		start = 1
		end = count
		if whileExpr != nil {
			start = a.Cursor
			if start < 1 {
				start = 1
			}
		}
	}
	if end < start {
		return nil
	}
	last := 0
	changed := 0
	for n := start; n <= end; n++ {
		a.Cursor = n
		a.BOF = false
		if err := a.LoadCurrent(); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		if ctx.Config.Deleted && a.Record.Deleted {
			continue
		}
		if whileExpr != nil {
			ok, err := evalLogic(whileExpr, env)
			if err != nil {
				return fail(ctx, "%s", err.Error())
			}
			if !ok {
				break
			}
		}
		if forExpr != nil {
			ok, err := evalLogic(forExpr, env)
			if err != nil {
				return fail(ctx, "%s", err.Error())
			}
			if !ok {
				continue
			}
		}
		if err := replaceOne(ctx, env, tree, fi, f); err != nil {
			return err
		}
		last = n
		changed++
	}
	if last > 0 {
		a.Cursor = last
		a.BOF = false
		_ = a.LoadCurrent()
	}
	if ctx.Config.Talk && (scope.Kind != "" || forExpr != nil || whileExpr != nil) {
		fmt.Fprintf(ctx.Out(), "%d records replaced\n", changed)
	}
	return nil
}

func replaceOne(ctx *context.Context, env expr.Env, tree expr.Expression, fi int, f dbf.Field) error {
	a := ctx.Active()
	v, err := expr.Eval(tree, env)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if f.Type == 'M' {
		if a.Table.Memo == nil {
			return fail(ctx, "no memo file")
		}
		text := v.String()
		block, err := a.Table.Memo.AllocWrite(text)
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		rec := a.Record
		rec.Values[fi] = dbf.Value{Str: dbf.FormatBlockDigits(block)}
		if err := a.Table.Write(a.Cursor, rec); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		a.Record = rec
		return syncAllIndexes(ctx)
	}
	dv, err := exprToDBF(f, v)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	rec := a.Record
	rec.Values[fi] = dv
	if err := a.Table.Write(a.Cursor, rec); err != nil {
		return fail(ctx, "%s", err.Error())
	}
	a.Record = rec
	return syncAllIndexes(ctx)
}

func parseReplaceArgs(args string) (field, exprSrc string, err error) {
	args = strings.TrimSpace(args)
	up := strings.ToUpper(args)
	i := strings.Index(up, " WITH ")
	if i < 0 {
		return "", "", errString("REPLACE requires WITH")
	}
	field = strings.TrimSpace(args[:i])
	exprSrc = strings.TrimSpace(args[i+6:])
	if field == "" || exprSrc == "" {
		return "", "", errString("REPLACE requires field and expression")
	}
	return field, exprSrc, nil
}

func fieldIndex(a *context.Area, name string) int {
	want := strings.ToUpper(name)
	for i, f := range a.Table.Header.Fields {
		if f.Name == want {
			return i
		}
	}
	return -1
}

func exprToDBF(f dbf.Field, v expr.Value) (dbf.Value, error) {
	switch f.Type {
	case 'C':
		s := v.String()
		if len(s) > int(f.Length) {
			s = s[:f.Length]
		}
		return dbf.Value{Str: s}, nil
	case 'N':
		n := v.Num
		if v.Kind == expr.KindString {
			parsed, err := parseFieldInput(f, v.Str)
			if err != nil {
				return dbf.Value{}, err
			}
			return parsed, nil
		}
		if v.Kind == expr.KindLogical {
			if v.Log {
				n = 1
			} else {
				n = 0
			}
		}
		return dbf.Value{Num: n}, nil
	case 'L':
		if v.Kind == expr.KindLogical {
			return dbf.Value{Log: v.Log}, nil
		}
		s := strings.TrimSpace(v.String())
		if s == "" {
			return dbf.Value{Log: false}, nil
		}
		c := s[0]
		return dbf.Value{Log: c == 'Y' || c == 'y' || c == 'T' || c == 't'}, nil
	case 'D', 'M':
		s := v.String()
		if s == "" {
			return dbf.Value{Empty: true}, nil
		}
		return dbf.Value{Str: s}, nil
	default:
		return dbf.Value{Str: v.String()}, nil
	}
}
