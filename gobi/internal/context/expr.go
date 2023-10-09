package context

import (
	"strings"

	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

// ExprArea builds the active-area facts for cursor built-ins.
func (c *Context) ExprArea() expr.Area {
	a := c.Active()
	if !a.HasTable() {
		return expr.Area{}
	}
	ea := expr.Area{
		HasTable: true,
		RecNo:    a.Cursor,
		RecCount: a.RecCount(),
		RecSize:  int(a.Table.Header.RecordLen),
		Deleted:  a.Landed && a.Record.Deleted,
		Found:    a.Found,
		BOF:      a.BOF,
		DBF:      a.Path,
		NDX:      a.NDX,
	}
	for _, f := range a.Table.Header.Fields {
		ea.Fields = append(ea.Fields, f.Name)
	}
	return ea
}

// FieldValue returns the named field from area (alias empty = active). ok is false if missing.
func (c *Context) FieldValue(alias, name string) (expr.Value, bool) {
	a := c.AreaByAlias(alias)
	if a == nil || !a.HasTable() || !a.Landed {
		return expr.Value{}, false
	}
	want := strings.ToUpper(name)
	for i, f := range a.Table.Header.Fields {
		if f.Name == want {
			return dbfToExpr(f, a.Record.Values[i]), true
		}
	}
	return expr.Value{}, false
}

// AreaByAlias returns the work area with that alias, or the active area when alias is empty.
func (c *Context) AreaByAlias(alias string) *Area {
	if alias == "" {
		return c.Active()
	}
	want := strings.ToUpper(alias)
	for i := 1; i <= AreaCount; i++ {
		a := c.Areas[i]
		if a.HasTable() && a.Alias == want {
			return a
		}
	}
	return nil
}

func dbfToExpr(f dbf.Field, v dbf.Value) expr.Value {
	switch f.Type {
	case 'N':
		return expr.NumberVal(v.Num)
	case 'L':
		return expr.LogicalVal(v.Log)
	case 'D', 'M':
		if v.Empty {
			return expr.StringVal("")
		}
		return expr.StringVal(v.Str)
	default:
		return expr.StringVal(v.Str)
	}
}
