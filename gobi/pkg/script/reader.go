package script

import (
	"fmt"
	"os"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

// Read opens path and parses the command file line by line.
func Read(path string) (*Program, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("read command file: %w", err)
	}
	return ParseSource(path, string(data))
}

// ParseSource parses in-memory PRG source into a Program.
func ParseSource(path, content string) (*Program, error) {
	prog := &Program{
		Path:  path,
		Lines: make([]Line, 0),
	}

	rawLines := splitSourceLines(content)
	for i := 0; i < len(rawLines); i++ {
		line := parseLine(i+1, rawLines[i])

		if line.Kind == LineCommand && line.Command.Verb == "TEXT" {
			body, consumed, ok := collectTextBlock(rawLines, i+1)
			if !ok {
				return nil, fmt.Errorf("script: unterminated TEXT block at line %d", i+1)
			}
			line.Text = body
			prog.Lines = append(prog.Lines, line)
			for j := i + 1; j <= i+consumed; j++ {
				prog.Lines = append(prog.Lines, Line{
					Number: j + 1,
					Source: strings.TrimRight(rawLines[j], "\r"),
					Kind:   LineText,
				})
			}
			i += consumed
			continue
		}

		prog.Lines = append(prog.Lines, line)
	}

	return prog, nil
}

// collectTextBlock gathers the literal lines following a TEXT command up to
// the matching ENDTEXT, preserving them verbatim (including blank lines and
// lines that would otherwise parse as comments or commands). The returned
// count covers the body lines plus the closing ENDTEXT line.
func collectTextBlock(rawLines []string, start int) (body []string, consumed int, ok bool) {
	for i := start; i < len(rawLines); i++ {
		line := strings.TrimRight(rawLines[i], "\r")
		if strings.ToUpper(strings.TrimSpace(line)) == "ENDTEXT" {
			return body, i - start + 1, true
		}
		body = append(body, line)
	}
	return nil, 0, false
}

func splitSourceLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	if content == "" {
		return nil
	}
	return strings.Split(content, "\n")
}

func parseLine(number int, raw string) Line {
	source := strings.TrimRight(raw, "\r")
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return Line{
			Number: number,
			Source: source,
			Kind:   LineEmpty,
		}
	}

	if strings.HasPrefix(trimmed, "*") {
		text := strings.TrimSpace(trimmed[1:])
		return Line{
			Number: number,
			Source: source,
			Kind:   LineRemark,
			Remark: text,
		}
	}

	// NOTE is the dBase II word form of the asterisk comment.
	upper := strings.ToUpper(trimmed)
	if upper == "NOTE" || strings.HasPrefix(upper, "NOTE ") || strings.HasPrefix(upper, "NOTE\t") {
		text := strings.TrimSpace(trimmed[len("NOTE"):])
		return Line{
			Number: number,
			Source: source,
			Kind:   LineRemark,
			Remark: text,
		}
	}

	return Line{
		Number:  number,
		Source:  source,
		Kind:    LineCommand,
		Command: command.Parse(trimmed),
	}
}
