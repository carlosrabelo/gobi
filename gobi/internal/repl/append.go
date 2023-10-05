package repl

import (
	"bufio"
	"os"
	"path/filepath"
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
	if from := strings.TrimSpace(cmd.Clauses["FROM"]); from != "" {
		return appendFrom(ctx, from, cmd.Args)
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

func appendFrom(ctx *context.Context, fromName, args string) error {
	mode := strings.ToUpper(strings.TrimSpace(args))
	pathName := fromName
	if mode == "" || mode == "SDF" || strings.HasPrefix(mode, "DELIMITED") {
		if mode == "" && filepath.Ext(pathName) == "" {
			pathName += ".dbf"
		}
	}
	path, err := ctx.Resolve(pathName)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	switch {
	case mode == "":
		return appendFromDBF(ctx, path)
	case mode == "SDF":
		return appendFromSDF(ctx, path)
	case mode == "DELIMITED" || strings.HasPrefix(mode, "DELIMITED"):
		return appendFromDelimited(ctx, path)
	default:
		return fail(ctx, "invalid APPEND FROM option")
	}
}

func appendFromDBF(ctx *context.Context, path string) error {
	a := ctx.Active()
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	defer f.Close()
	src, err := dbf.Open(f)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	tgtFields := a.Table.Header.Fields
	srcFields := src.Header.Fields
	srcIdx := map[string]int{}
	for i, f := range srcFields {
		srcIdx[f.Name] = i
	}
	last := 0
	for n := 1; n <= int(src.Header.RecordCount); n++ {
		srec, err := src.Read(n)
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		if srec.Deleted {
			continue
		}
		if uint64(a.RecCount()) >= maxRecords {
			return fail(ctx, "record count limit exceeded")
		}
		rec := dbf.Record{Values: make([]dbf.Value, len(tgtFields))}
		for i, tf := range tgtFields {
			if si, ok := srcIdx[tf.Name]; ok {
				rec.Values[i] = srec.Values[si]
			} else {
				rec.Values[i] = blankValue(tf)
			}
		}
		if err := a.Table.Append(rec); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		last = a.RecCount()
	}
	if last > 0 {
		a.Cursor = last
		a.BOF = false
		return a.LoadCurrent()
	}
	return nil
}

func blankValue(f dbf.Field) dbf.Value {
	switch f.Type {
	case 'N':
		return dbf.Value{Num: 0}
	case 'L':
		return dbf.Value{Log: false}
	case 'D', 'M':
		return dbf.Value{Empty: true}
	default:
		return dbf.Value{Str: ""}
	}
}

func appendFromSDF(ctx *context.Context, path string) error {
	a := ctx.Active()
	fields := a.Table.Header.Fields
	width := 0
	for _, f := range fields {
		width += int(f.Length)
	}
	f, err := os.Open(path)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	last := 0
	for sc.Scan() {
		line := sc.Text()
		if len(line) < width {
			line += strings.Repeat(" ", width-len(line))
		}
		raw := append([]byte{' '}, []byte(line[:width])...)
		srec, err := dbf.DecodeRecord(fields, raw)
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		if err := a.Table.Append(srec); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		last = a.RecCount()
	}
	if err := sc.Err(); err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if last > 0 {
		a.Cursor = last
		a.BOF = false
		return a.LoadCurrent()
	}
	return nil
}

func appendFromDelimited(ctx *context.Context, path string) error {
	a := ctx.Active()
	fields := a.Table.Header.Fields
	f, err := os.Open(path)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	last := 0
	for sc.Scan() {
		parts := splitCSVLine(sc.Text())
		rec := dbf.Record{Values: make([]dbf.Value, len(fields))}
		for i, f := range fields {
			s := ""
			if i < len(parts) {
				s = parts[i]
			}
			v, err := parseFieldInput(f, s)
			if err != nil {
				return fail(ctx, "%s", err.Error())
			}
			rec.Values[i] = v
		}
		if err := a.Table.Append(rec); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		last = a.RecCount()
	}
	if err := sc.Err(); err != nil {
		return fail(ctx, "%s", err.Error())
	}
	if last > 0 {
		a.Cursor = last
		a.BOF = false
		return a.LoadCurrent()
	}
	return nil
}

func splitCSVLine(line string) []string {
	var parts []string
	var b strings.Builder
	inQ := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case c == '"':
			inQ = !inQ
		case c == ',' && !inQ:
			parts = append(parts, b.String())
			b.Reset()
		default:
			b.WriteByte(c)
		}
	}
	parts = append(parts, b.String())
	return parts
}
