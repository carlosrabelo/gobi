package repl

import (
	"strconv"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

const maxRecords = 4294967295

func cmdAPPEND(ctx *context.Context, cmd command.Command) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	fields := a.Table.Header.Fields
	for {
		if uint64(a.RecCount()) >= maxRecords {
			return fail(ctx, "record count limit exceeded")
		}
		rec := dbf.Record{Values: make([]dbf.Value, len(fields))}
		stopped := false
		for i, f := range fields {
			line, err := readPromptLine(ctx, f.Name+": ")
			if err != nil {
				return fail(ctx, "%s", err.Error())
			}
			if i == 0 && strings.TrimSpace(line) == "" {
				stopped = true
				break
			}
			v, err := parseFieldInput(f, line)
			if err != nil {
				return fail(ctx, "%s", err.Error())
			}
			rec.Values[i] = v
		}
		if stopped {
			break
		}
		if err := a.Table.Append(rec); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		a.Cursor = a.RecCount()
		a.BOF = false
		if err := a.LoadCurrent(); err != nil {
			return fail(ctx, "%s", err.Error())
		}
	}
	return nil
}

func parseFieldInput(f dbf.Field, line string) (dbf.Value, error) {
	s := line
	switch f.Type {
	case 'C':
		return dbf.Value{Str: s}, nil
	case 'N':
		s = strings.TrimSpace(s)
		if s == "" {
			return dbf.Value{Num: 0}, nil
		}
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return dbf.Value{}, err
		}
		return dbf.Value{Num: n}, nil
	case 'L':
		s = strings.TrimSpace(s)
		if s == "" {
			return dbf.Value{Log: false}, nil
		}
		c := s[0]
		switch c {
		case 'Y', 'y', 'T', 't':
			return dbf.Value{Log: true}, nil
		default:
			return dbf.Value{Log: false}, nil
		}
	case 'D':
		s = strings.TrimSpace(s)
		if s == "" {
			return dbf.Value{Empty: true}, nil
		}
		return dbf.Value{Str: s}, nil
	case 'M':
		s = strings.TrimSpace(s)
		if s == "" {
			return dbf.Value{Empty: true}, nil
		}
		return dbf.Value{Str: s}, nil
	default:
		return dbf.Value{Str: s}, nil
	}
}
