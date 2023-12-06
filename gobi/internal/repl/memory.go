package repl

import (
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
)

const minMemvarNameWidth = 10

func outputMemory(ctx *context.Context) error {
	return writeMemoryTable(ctx, ctx.Stdout)
}

func writeMemoryTable(ctx *context.Context, out io.Writer) error {
	names := ctx.Variables.Names()
	entries := make([]memoryEntry, 0, len(names))

	nameWidth := minMemvarNameWidth
	totalBytes := 0

	for _, name := range names {
		val, _ := ctx.Variables.Get(name)
		entry := memoryEntry{
			Name:      name,
			Type:      memvarTypeChar(val),
			Value:     formatMemvarValue(val),
			ByteCount: memvarByteSize(val),
		}
		entries = append(entries, entry)
		if len(name) > nameWidth {
			nameWidth = len(name)
		}
		totalBytes += entry.ByteCount
	}

	for _, entry := range entries {
		fmt.Fprintf(out, "%-*s  (%c)  %s\r\n", nameWidth, entry.Name, entry.Type, entry.Value)
	}

	fmt.Fprintf(out, "** TOTAL ** %02d VARIABLES USED %05d BYTES USED\r\n", len(entries), totalBytes)
	return nil
}

type memoryEntry struct {
	Name      string
	Type      byte
	Value     string
	ByteCount int
}

func memvarTypeChar(val interface{}) byte {
	switch val.(type) {
	case float64, int:
		return 'N'
	case bool:
		return 'L'
	default:
		return 'C'
	}
}

func formatMemvarValue(val interface{}) string {
	switch v := val.(type) {
	case float64:
		if math.Mod(v, 1) == 0 {
			return fmt.Sprintf("%.0f", v)
		}
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.10f", v), "0"), ".")
	case int:
		return fmt.Sprintf("%d", v)
	case bool:
		if v {
			return ".T."
		}
		return ".F."
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

func memvarByteSize(val interface{}) int {
	switch v := val.(type) {
	case string:
		return len(v)
	case float64:
		return 8
	case int:
		return 8
	case bool:
		return 1
	default:
		return len(formatMemvarValue(v))
	}
}
