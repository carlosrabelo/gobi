package repl

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func cmdCREATE(ctx *context.Context, cmd command.Command) error {
	if handled, err := createDesignerCmd(ctx, cmd); handled {
		return err
	}
	args := strings.TrimSpace(cmd.Args)
	from := strings.TrimSpace(cmd.Clauses["FROM"])
	if from != "" {
		name := args
		// CREATE name FROM other — strip trailing FROM clause text if present in Args.
		if i := strings.Index(strings.ToUpper(name), " FROM "); i >= 0 {
			name = strings.TrimSpace(name[:i])
		}
		if name == "" {
			return fail(ctx, "CREATE FROM requires a name")
		}
		return cmdCREATEFROM(ctx, name, from)
	}
	name := args
	if name == "" {
		return fail(ctx, "CREATE requires a file name")
	}
	pathName := name
	if filepath.Ext(pathName) == "" {
		pathName += ".dbf"
	}
	path, err := ctx.Resolve(pathName)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}

	fields, err := promptFields(ctx)
	if err != nil {
		return err
	}
	if len(fields) == 0 {
		return fail(ctx, "no fields defined")
	}
	if _, err := dbf.ValidateFields(fields); err != nil {
		return fail(ctx, "%s", err.Error())
	}

	if fileExists(path) && ctx.Config.Safety {
		_, err := confirmSafety(ctx, path+" already exists. Overwrite it? (Y/N)")
		if err != nil {
			return err
		}
	}

	now := time.Now()
	year := byte(now.Year() - 1900)
	month := byte(now.Month())
	day := byte(now.Day())

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".gobi-create-*")
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	tmpName := tmp.Name()
	tab, err := dbf.Create(tmp, fields, year, month, day)
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}
	if err := tab.Close(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fail(ctx, "%s", err.Error())
	}
	hasMemo := false
	for _, f := range fields {
		if f.Type == 'M' {
			hasMemo = true
			break
		}
	}
	if hasMemo {
		dbt := strings.TrimSuffix(path, filepath.Ext(path)) + ".dbt"
		mf, err := os.Create(dbt)
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		if _, err := dbf.CreateMemo(mf); err != nil {
			_ = mf.Close()
			return fail(ctx, "%s", err.Error())
		}
		_ = mf.Close()
	}
	return nil
}

func promptFields(ctx *context.Context) ([]dbf.Field, error) {
	var fields []dbf.Field
	for {
		name, err := readPromptLine(ctx, "Field name: ")
		if err != nil {
			return nil, fail(ctx, "%s", err.Error())
		}
		name = strings.TrimSpace(name)
		if name == "" {
			break
		}
		typeLine, err := readPromptLine(ctx, "Type: ")
		if err != nil {
			return nil, fail(ctx, "%s", err.Error())
		}
		typeLine = strings.TrimSpace(typeLine)
		if typeLine == "" {
			return nil, fail(ctx, "field type required")
		}
		typ := byte(strings.ToUpper(typeLine)[0])

		lenLine, err := readPromptLine(ctx, "Length: ")
		if err != nil {
			return nil, fail(ctx, "%s", err.Error())
		}
		length, err := strconv.Atoi(strings.TrimSpace(lenLine))
		if err != nil || length < 1 || length > 254 {
			return nil, fail(ctx, "invalid field length")
		}

		decLine, err := readPromptLine(ctx, "Decimals: ")
		if err != nil {
			return nil, fail(ctx, "%s", err.Error())
		}
		decimals, err := strconv.Atoi(strings.TrimSpace(decLine))
		if err != nil || decimals < 0 || decimals > 254 {
			return nil, fail(ctx, "invalid decimals")
		}

		f := dbf.Field{
			Name:     strings.ToUpper(name),
			Type:     typ,
			Length:   byte(length),
			Decimals: byte(decimals),
		}
		if _, err := dbf.ValidateFields([]dbf.Field{f}); err != nil {
			return nil, fail(ctx, "%s", err.Error())
		}
		fields = append(fields, f)
		if len(fields) > 128 {
			return nil, fail(ctx, "too many fields")
		}
	}
	return fields, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
