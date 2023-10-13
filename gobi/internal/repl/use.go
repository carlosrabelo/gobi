package repl

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
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

	filename := strings.TrimSpace(cmd.Args)
	if filename == "" {
		return nil
	}
	if fields := strings.Fields(filename); len(fields) > 1 {
		return fmt.Errorf("*** Unexpected argument: %s", fields[1])
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

	return nil
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
