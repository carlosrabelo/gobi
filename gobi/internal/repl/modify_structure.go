package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

const modifyPanelSize = 16

type modifyStructureSession struct {
	ctx       *context.Context
	area      *context.WorkArea
	tbl       *dbf.Table
	wseeker   io.ReadWriteSeeker
	in        io.Reader
	out       io.Writer
	frame     *term.FrameWriter
	tty       *os.File
	raw       bool
	lines     []string
	panel     int
	rowIdx    int
	cursorPos int
	dirty     bool
}

func runModifyStructureForm(ctx *context.Context) error {
	area := ctx.GetActiveArea()
	wseeker, ok := area.Table.Underlying().(io.ReadWriteSeeker)
	if !ok {
		return fmt.Errorf("*** Database file is not writable")
	}

	lines := make([]string, len(area.Table.Fields))
	for i, fd := range area.Table.Fields {
		lines[i] = fieldToStructureLine(fd)
	}

	s := &modifyStructureSession{
		ctx:     ctx,
		area:    area,
		tbl:     area.Table,
		wseeker: wseeker,
		in:      ctx.Stdin,
		out:     ctx.Stdout,
		frame:   term.NewFrameWriter(ctx.Stdout),
		lines:   lines,
	}

	if f, ok := ctx.Stdin.(*os.File); ok && term.IsTerminal(f) {
		s.tty = f
		s.raw = true
	}

	var kbd *term.Keyboard
	if s.raw {
		rawMode, err := term.EnterRawMode(s.tty)
		if err != nil {
			s.raw = false
		} else {
			defer rawMode.Close()
			kbd = term.NewKeyboard(s.tty)
		}
	}

	for {
		s.draw()
		key, err := readReplKey(kbd, s.raw, s.in)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("*** Error reading input: %w", err)
		}

		quit, err := s.handleKey(key)
		if err != nil {
			return err
		}
		if quit {
			return nil
		}
	}
}

func fieldToStructureLine(fd dbf.FieldDescriptor) string {
	if fd.Type == dbf.FieldTypeNumeric {
		return fmt.Sprintf("%s,%c,%d,%d", fd.Name, fd.Type, fd.Length, fd.DecimalCount)
	}
	return fmt.Sprintf("%s,%c,%d,0", fd.Name, fd.Type, fd.Length)
}

func (s *modifyStructureSession) globalIdx() int {
	return s.panel*modifyPanelSize + s.rowIdx
}

func (s *modifyStructureSession) currentLine() string {
	idx := s.globalIdx()
	if idx >= len(s.lines) {
		return ""
	}
	return s.lines[idx]
}

func (s *modifyStructureSession) setCurrentLine(val string) {
	idx := s.globalIdx()
	for len(s.lines) <= idx {
		s.lines = append(s.lines, "")
	}
	s.lines[idx] = val
	s.dirty = true
}

func (s *modifyStructureSession) draw() {
	s.frame.Begin()
	fmt.Fprintf(s.frame.Back(), "MODIFY STRUCTURE - %s\r\n\r\n", s.area.Alias)
	fmt.Fprintln(s.frame.Back(), "FIELD NAME,TYPE,WIDTH,DECIMAL PLACES")
	fmt.Fprintln(s.frame.Back())

	for i := 0; i < modifyPanelSize; i++ {
		prefix := "  "
		if i == s.rowIdx {
			prefix = "> "
		}
		global := s.panel*modifyPanelSize + i + 1
		line := ""
		idx := s.panel*modifyPanelSize + i
		if idx < len(s.lines) {
			line = s.lines[idx]
		}
		fmt.Fprintf(s.frame.Back(), "%s%02d  %s\r\n", prefix, global, line)
	}

	if s.panel > 0 || len(s.lines) > modifyPanelSize {
		fmt.Fprintf(s.frame.Back(), "\r\nPanel %d\r\n", s.panel+1)
	}
	s.frame.Present()
}

func (s *modifyStructureSession) handleKey(key byte) (quit bool, err error) {
	switch key {
	case editKeyCtrlQ:
		return true, nil

	case editKeyCtrlW:
		return true, s.applyStructure()

	case editKeyCtrlG, 24:
		if (s.panel+1)*modifyPanelSize < len(s.lines)+modifyPanelSize {
			s.panel++
			s.rowIdx = 0
			s.cursorPos = 0
		}
		return false, nil

	case editKeyCtrlR:
		if s.panel > 0 {
			s.panel--
			s.rowIdx = 0
			s.cursorPos = 0
		}
		return false, nil

	case editKeyCtrlN:
		idx := s.globalIdx()
		s.lines = append(s.lines[:idx], append([]string{""}, s.lines[idx:]...)...)
		s.dirty = true
		s.cursorPos = 0
		return false, nil

	case editKeyCtrlT:
		idx := s.globalIdx()
		if idx < len(s.lines) {
			s.lines = append(s.lines[:idx], s.lines[idx+1:]...)
			s.dirty = true
			if s.rowIdx >= modifyPanelSize-1 {
				s.rowIdx = modifyPanelSize - 1
			}
			s.cursorPos = len(s.currentLine())
		}
		return false, nil

	case editKeyCtrlY:
		s.setCurrentLine("")
		s.cursorPos = 0
		return false, nil

	case editKeyTab:
		if s.rowIdx == modifyPanelSize-1 {
			if (s.panel+1)*modifyPanelSize < len(s.lines)+modifyPanelSize {
				s.panel++
				s.rowIdx = 0
			}
		} else {
			s.rowIdx++
		}
		s.cursorPos = len(s.currentLine())
		return false, nil

	case editKeyCtrlK:
		if s.rowIdx == 0 {
			if s.panel > 0 {
				s.panel--
				s.rowIdx = modifyPanelSize - 1
			}
		} else {
			s.rowIdx--
		}
		s.cursorPos = len(s.currentLine())
		return false, nil

	case editKeyDel, editKeyCtrlH:
		val := s.currentLine()
		if s.cursorPos > 0 {
			val = val[:s.cursorPos-1] + val[s.cursorPos:]
			s.cursorPos--
			s.setCurrentLine(val)
		} else if s.rowIdx > 0 || s.panel > 0 {
			if s.rowIdx == 0 {
				s.panel--
				s.rowIdx = modifyPanelSize - 1
			} else {
				s.rowIdx--
			}
			s.cursorPos = len(s.currentLine())
		}
		return false, nil

	default:
		if key >= 32 && key < 127 {
			val := s.currentLine()
			if s.cursorPos > len(val) {
				s.cursorPos = len(val)
			}
			val = val[:s.cursorPos] + string(key) + val[s.cursorPos:]
			s.setCurrentLine(val)
			s.cursorPos++
		}
		return false, nil
	}
}

func (s *modifyStructureSession) applyStructure() error {
	fields, err := parseStructureLines(s.lines)
	if err != nil {
		return err
	}

	if _, err := dbf.RewriteStructure(s.wseeker, fields); err != nil {
		return fmt.Errorf("*** Error modifying structure: %w", err)
	}

	if _, err := s.wseeker.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("*** Error seeking database: %w", err)
	}

	newTbl, err := dbf.Open(s.wseeker)
	if err != nil {
		return fmt.Errorf("*** Error reopening database: %w", err)
	}

	s.area.Table = newTbl
	s.area.RecordNo = 0
	s.area.ActiveRecord = nil
	return nil
}

func parseStructureLines(lines []string) ([]dbf.FieldDescriptor, error) {
	var fields []dbf.FieldDescriptor
	seenNames := make(map[string]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fd, err := parseCreateFieldDefinition(line)
		if err != nil {
			return nil, err
		}
		if seenNames[fd.Name] {
			return nil, fmt.Errorf("*** Duplicate field name: %s", fd.Name)
		}
		seenNames[fd.Name] = true
		fields = append(fields, fd)
	}

	if len(fields) == 0 {
		return nil, fmt.Errorf("*** At least one field is required")
	}
	if len(fields) > 32 {
		return nil, fmt.Errorf("*** Too many fields (max 32)")
	}

	return fields, nil
}

func confirmDataLoss(ctx *context.Context, reader *bufio.Reader) (bool, error) {
	line, err := readAppendLine(ctx, reader, "ALL DATA WILL BE LOST. CONTINUE? (Y/N) ")
	if err != nil {
		if err == io.EOF {
			return false, nil
		}
		return false, fmt.Errorf("*** Error reading input: %w", err)
	}

	switch strings.ToUpper(strings.TrimSpace(line)) {
	case "Y":
		return true, nil
	case "N", "":
		return false, nil
	default:
		return false, fmt.Errorf("*** Invalid response")
	}
}
