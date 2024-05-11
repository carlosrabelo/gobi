package repl

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

const (
	editKeyCtrlG = 7
	editKeyCtrlH = 8
	editKeyCtrlK = 11
	editKeyCtrlN = 14
	editKeyCtrlQ = 17
	editKeyCtrlR = 18
	editKeyCtrlT = 20
	editKeyCtrlU = 21
	editKeyCtrlW = 23
	editKeyCtrlY = 25
	editKeyTab   = 9
	editKeyEnter = 13
	editKeyEsc   = 27
	editKeyDel   = 127
)

type editSession struct {
	ctx       *context.Context
	area      *context.WorkArea
	tbl       *dbf.Table
	wseeker   io.ReadWriteSeeker
	in        io.Reader
	out       io.Writer
	frame     *term.FrameWriter
	tty       *os.File
	raw       bool
	recIdx    int
	fieldIdx  int
	cursorPos int
	values    []string
	deleted   bool
	dirty     bool
}

func runEditForm(ctx *context.Context, recIdx int) error {
	area := ctx.GetActiveArea()
	wseeker, ok := area.Table.Underlying().(io.ReadWriteSeeker)
	if !ok {
		return fmt.Errorf("*** Database file is not writable")
	}

	s := &editSession{
		ctx:     ctx,
		area:    area,
		tbl:     area.Table,
		wseeker: wseeker,
		in:      ctx.Stdin,
		out:     ctx.Stdout,
		frame:   term.NewFrameWriter(ctx.Stdout),
		recIdx:  recIdx,
	}

	if f, ok := ctx.Stdin.(*os.File); ok && term.IsTerminal(f) {
		s.tty = f
		s.raw = true
	}

	if err := s.loadRecord(); err != nil {
		return err
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

func (s *editSession) loadRecord() error {
	rec, err := s.tbl.ReadRecordAt(s.wseeker, s.recIdx)
	if err != nil {
		return fmt.Errorf("*** Error reading record %d: %w", s.recIdx+1, err)
	}

	values, err := recordToEditValues(s.tbl, rec)
	if err != nil {
		return fmt.Errorf("*** Error decoding record %d: %w", s.recIdx+1, err)
	}

	s.values = values
	s.deleted = rec.Deleted
	s.dirty = false
	s.fieldIdx = 0
	s.cursorPos = 0
	s.area.RecordNo = s.recIdx
	s.area.ActiveRecord = rec
	return nil
}

func recordToEditValues(tbl *dbf.Table, rec *dbf.Record) ([]string, error) {
	values := make([]string, len(tbl.Fields))
	for i, fd := range tbl.Fields {
		decoded, err := rec.DecodeField(tbl, i)
		if err != nil {
			return nil, err
		}
		values[i] = fieldValueToEditString(fd, decoded)
	}
	return values, nil
}

func fieldValueToEditString(fd dbf.FieldDescriptor, val interface{}) string {
	switch fd.Type {
	case dbf.FieldTypeChar:
		return val.(string)
	case dbf.FieldTypeNumeric:
		f := val.(float64)
		if fd.DecimalCount > 0 {
			return strconv.FormatFloat(f, 'f', int(fd.DecimalCount), 64)
		}
		return strconv.FormatFloat(f, 'f', 0, 64)
	case dbf.FieldTypeLogical:
		if val.(bool) {
			return "T"
		}
		return "F"
	default:
		return fmt.Sprintf("%v", val)
	}
}

func (s *editSession) draw() {
	s.frame.Begin()
	fmt.Fprintf(s.frame.Back(), "EDIT - %s\r\n\r\n", s.area.Alias)

	intensity := intensityEnabled(s.ctx)
	for i, fd := range s.tbl.Fields {
		prefix := "  "
		if i == s.fieldIdx {
			prefix = "> "
		}
		if intensity && i == s.fieldIdx {
			_ = term.SetReverse(s.frame.Back())
		}
		fmt.Fprintf(s.frame.Back(), "%s%-12s %s\r\n", prefix, fd.Name+":", s.values[i])
		if intensity && i == s.fieldIdx {
			_ = term.Reset(s.frame.Back())
		}
	}

	fmt.Fprintf(s.frame.Back(), "\r\nRECORD # %05d", s.recIdx+1)
	if s.deleted {
		fmt.Fprint(s.frame.Back(), "  *")
	}
	fmt.Fprint(s.frame.Back(), "\r\n")
	s.frame.Present()
}

func (s *editSession) handleKey(key byte) (quit bool, err error) {
	recCount := int(s.tbl.Header.RecordCount)

	switch key {
	case editKeyCtrlQ:
		return true, nil

	case editKeyCtrlW:
		if err := s.saveCurrent(); err != nil {
			reportValidationError(s.ctx, err)
			return false, nil
		}
		return true, nil

	case editKeyCtrlG, 24: // Ctrl+X
		if err := s.saveCurrent(); err != nil {
			reportValidationError(s.ctx, err)
			return false, nil
		}
		if s.recIdx+1 < recCount {
			s.recIdx++
			return false, s.loadRecord()
		}
		return true, nil

	case editKeyCtrlR:
		if err := s.saveCurrent(); err != nil {
			reportValidationError(s.ctx, err)
			return false, nil
		}
		if s.recIdx > 0 {
			s.recIdx--
			return false, s.loadRecord()
		}
		return true, nil

	case editKeyCtrlU:
		s.deleted = !s.deleted
		s.dirty = true
		return false, nil

	case editKeyCtrlY:
		s.values[s.fieldIdx] = ""
		s.cursorPos = 0
		s.dirty = true
		return false, nil

	case editKeyTab:
		s.fieldIdx = (s.fieldIdx + 1) % len(s.tbl.Fields)
		s.cursorPos = len(s.values[s.fieldIdx])
		return false, nil

	case editKeyCtrlK:
		if s.fieldIdx == 0 {
			s.fieldIdx = len(s.tbl.Fields) - 1
		} else {
			s.fieldIdx--
		}
		s.cursorPos = len(s.values[s.fieldIdx])
		return false, nil

	case editKeyEnter:
		if s.fieldIdx == len(s.tbl.Fields)-1 {
			if err := s.saveCurrent(); err != nil {
				reportValidationError(s.ctx, err)
				return false, nil
			}
			if s.recIdx+1 < recCount {
				s.recIdx++
				return false, s.loadRecord()
			}
			return true, nil
		}
		s.fieldIdx++
		s.cursorPos = 0
		return false, nil

	case editKeyDel, editKeyCtrlH:
		if s.cursorPos > 0 {
			val := s.values[s.fieldIdx]
			s.values[s.fieldIdx] = val[:s.cursorPos-1] + val[s.cursorPos:]
			s.cursorPos--
			s.dirty = true
		} else if s.fieldIdx > 0 {
			s.fieldIdx--
			s.cursorPos = len(s.values[s.fieldIdx])
		}
		return false, nil

	default:
		if key >= 32 && key < 127 {
			if err := s.insertChar(byte(key)); err != nil {
				return false, err
			}
		}
		return false, nil
	}
}

func (s *editSession) insertChar(ch byte) error {
	fd := s.tbl.Fields[s.fieldIdx]
	maxLen := int(fd.Length)
	if fd.Type == dbf.FieldTypeLogical {
		maxLen = 1
	}

	val := s.values[s.fieldIdx]
	if len(val) < s.cursorPos {
		s.cursorPos = len(val)
	}

	if fd.Type == dbf.FieldTypeLogical {
		s.values[s.fieldIdx] = strings.ToUpper(string(ch))[:1]
		s.cursorPos = 1
		s.dirty = true
		return nil
	}

	if len(val) < maxLen {
		val = val[:s.cursorPos] + string(ch) + val[s.cursorPos:]
	} else if s.cursorPos < maxLen {
		val = val[:s.cursorPos] + string(ch) + val[s.cursorPos+1:]
	} else {
		return nil
	}

	if len(val) > maxLen {
		val = val[:maxLen]
	}

	s.values[s.fieldIdx] = val
	if s.cursorPos < maxLen {
		s.cursorPos++
	}
	s.dirty = true
	return nil
}

func (s *editSession) saveCurrent() error {
	parsed := make([]interface{}, len(s.tbl.Fields))
	for i, fd := range s.tbl.Fields {
		val, err := parseFieldInput(fd, s.values[i])
		if err != nil {
			return err
		}
		parsed[i] = val
	}

	rec, err := dbf.NewRecord(s.tbl, s.deleted, parsed)
	if err != nil {
		return fmt.Errorf("*** Error building record: %w", err)
	}

	if err := s.tbl.WriteRecordAt(s.wseeker, s.recIdx, rec); err != nil {
		return fmt.Errorf("*** Error writing record %d: %w", s.recIdx+1, err)
	}

	s.area.RecordNo = s.recIdx
	s.area.ActiveRecord = rec
	s.dirty = false
	return nil
}
