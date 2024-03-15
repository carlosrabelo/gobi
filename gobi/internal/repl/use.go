package repl

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/ndx"
)

func handleUse(ctx *context.Context, cmd Command) error {
	area := ctx.GetActiveArea()

	if area.Table != nil {
		if err := area.Table.Close(); err != nil {
			return fmt.Errorf("*** Error closing current table: %w", err)
		}
		area.Table = nil
		area.RecordNo = 0
		area.ActiveRecord = nil
		clearLocateState(area)
	}
	closeOpenIndexes(area)

	filename, indexNames, err := parseUseArgs(cmd.Args)
	if err != nil {
		return err
	}
	if filename == "" {
		return nil
	}

	filePath := resolveDBFFilePath(ctx, filename)

	f, err := os.OpenFile(filePath, os.O_RDWR, 0)
	if err != nil {
		f, err = os.Open(filePath)
		if err != nil {
			return fmt.Errorf("*** Could not open file: %w", err)
		}
	}

	tbl, err := dbf.Open(f)
	if err != nil {
		f.Close()
		return fmt.Errorf("*** Error reading DBF header: %w", err)
	}

	area.Table = tbl
	area.Alias = strings.ToUpper(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
	area.RecordNo = 0
	area.ActiveRecord = nil
	clearLocateState(area)

	if tbl.Header.RecordCount > 0 {
		if rseeker, ok := tbl.Underlying().(io.ReadSeeker); ok {
			rec, err := tbl.ReadRecordAt(rseeker, 0)
			if err == nil {
				area.ActiveRecord = rec
			}
		}
	}

	if err := bindUseIndexes(ctx, area, indexNames); err != nil {
		closeOpenIndexes(area)
		_ = tbl.Close()
		area.Table = nil
		area.RecordNo = 0
		area.ActiveRecord = nil
		clearLocateState(area)
		return err
	}

	return nil
}

func parseUseArgs(args string) (filename string, indexNames []string, err error) {
	fields := strings.Fields(args)
	if len(fields) == 0 {
		return "", nil, nil
	}

	filename = fields[0]
	if len(fields) == 1 {
		return filename, nil, nil
	}

	if !strings.EqualFold(fields[1], "INDEX") {
		return "", nil, fmt.Errorf("*** Unexpected argument: %s", fields[1])
	}

	indexNames = splitIndexNames(strings.Join(fields[2:], " "))
	if len(indexNames) == 0 {
		return "", nil, fmt.Errorf("*** INDEX requires at least one index file name")
	}
	return filename, indexNames, nil
}

func bindUseIndexes(ctx *context.Context, area *context.WorkArea, indexNames []string) error {
	for _, name := range indexNames {
		filePath := resolveNDXFilePath(ctx, name)
		idx, err := ndx.OpenIndex(filePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("*** Index file not found: %s", name)
			}
			return fmt.Errorf("*** Error opening index %s: %w", name, err)
		}
		area.Indexes = append(area.Indexes, idx)
	}
	return nil
}

func splitIndexNames(list string) []string {
	var names []string
	for _, part := range strings.Split(list, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			names = append(names, part)
		}
	}
	return names
}

func closeOpenIndexes(area *context.WorkArea) {
	for _, idx := range area.Indexes {
		if idx != nil {
			_ = idx.Close()
		}
	}
	area.Indexes = nil
}

func clearLocateState(area *context.WorkArea) {
	if area == nil {
		return
	}
	area.Found = false
	area.LocateActive = false
	area.LocateFor = ""
	area.LocateWhile = ""
	area.LocateEnd = 0
}
