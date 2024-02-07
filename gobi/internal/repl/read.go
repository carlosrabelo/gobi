package repl

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

var errReadAbort = errors.New("read aborted")

type readFieldState struct {
	def      term.GetField
	value    string
	spec     pictureSpec
	maxWidth int
	slotIdx  int
}

type readSession struct {
	ctx      *context.Context
	fields   []readFieldState
	fieldIdx int
	in       io.Reader
	out      io.Writer
	tty      *os.File
	raw      bool
}

func handleRead(ctx *context.Context, cmd Command) error {
	fields := ctx.Screen.GetFields()
	if len(fields) == 0 {
		return fmt.Errorf("*** READ requires @ GET fields")
	}
	err := runReadForm(ctx)
	if consoleErr := returnToConsole(ctx); consoleErr != nil && err == nil {
		err = consoleErr
	}
	if err == errReadAbort {
		return nil
	}
	return err
}

func runReadForm(ctx *context.Context) error {
	states := make([]readFieldState, 0, len(ctx.Screen.GetFields()))
	for _, field := range ctx.Screen.GetFields() {
		spec := buildPictureSpec(field.Picture)
		value := strings.TrimRight(resolveGetDisplayValue(ctx, field.Name), " ")
		states = append(states, readFieldState{
			def:      field,
			value:    value,
			spec:     spec,
			maxWidth: readFieldWidth(ctx.Screen, field, spec),
		})
	}

	s := &readSession{
		ctx:    ctx,
		fields: states,
		in:     ctx.Stdin,
		out:    ctx.Stdout,
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
		if err := s.draw(); err != nil {
			return err
		}

		key, err := readReplKey(kbd, s.raw, s.in)
		if err != nil {
			if err == io.EOF {
				return s.commitAll()
			}
			return fmt.Errorf("*** Error reading input: %w", err)
		}

		done, err := s.handleKey(key)
		if err == errReadAbort {
			return errReadAbort
		}
		if err != nil {
			reportValidationError(s.ctx, err)
			continue
		}
		if done {
			return s.commitAll()
		}
	}
}

func (s *readSession) current() *readFieldState {
	return &s.fields[s.fieldIdx]
}

func (s *readSession) fieldDisplay(field readFieldState) string {
	if field.spec.width > 0 {
		return overlayPictureValue(field.spec, field.value)
	}
	if field.maxWidth > 0 && len(field.value) > field.maxWidth {
		return field.value[:field.maxWidth]
	}
	return field.value
}

func (s *readSession) draw() error {
	for i := range s.fields {
		display := s.fieldDisplay(s.fields[i])
		s.ctx.Screen.WriteAt(s.fields[i].def.Row, s.fields[i].def.Col, display)
	}

	var highlights []term.Highlight
	if intensityEnabled(s.ctx) {
		field := s.current()
		length := readFieldHighlightWidth(*field)
		highlights = []term.Highlight{{
			Row:    field.def.Row,
			Col:    field.def.Col,
			Length: length,
		}}
	}

	if err := presentScreen(s.ctx, s.out, highlights); err != nil {
		return err
	}

	field := s.current()
	row := field.def.Row
	col := field.def.Col
	if field.spec.width > 0 {
		col = cursorColumnForPicture(field.spec, field.def.Col, field.slotIdx)
	} else {
		if field.slotIdx < 0 {
			field.slotIdx = len(field.value)
		}
		col += field.slotIdx
	}
	return term.MoveTo(s.out, row, col)
}

func readFieldHighlightWidth(field readFieldState) int {
	if field.spec.width > 0 {
		return field.spec.width
	}
	if field.maxWidth > 0 {
		return field.maxWidth
	}
	if len(field.value) == 0 {
		return 1
	}
	return len(field.value)
}

func (s *readSession) handleKey(key byte) (done bool, err error) {
	switch key {
	case editKeyEsc:
		return true, errReadAbort

	case editKeyTab:
		if err := s.validateCurrent(); err != nil {
			return false, err
		}
		if s.fieldIdx == len(s.fields)-1 {
			return true, nil
		}
		s.fieldIdx++
		s.fields[s.fieldIdx].slotIdx = 0
		return false, nil

	case editKeyCtrlK:
		if s.fieldIdx == 0 {
			s.fieldIdx = len(s.fields) - 1
		} else {
			s.fieldIdx--
		}
		s.current().slotIdx = 0
		return false, nil

	case editKeyEnter:
		if err := s.validateCurrent(); err != nil {
			return false, err
		}
		if s.fieldIdx == len(s.fields)-1 {
			return true, nil
		}
		s.fieldIdx++
		s.current().slotIdx = 0
		return false, nil

	case editKeyDel, editKeyCtrlH:
		field := s.current()
		if field.spec.width > 0 {
			field.value, field.slotIdx = pictureDeleteChar(field.spec, field.value, field.slotIdx)
		} else {
			if field.slotIdx > len(field.value) {
				field.slotIdx = len(field.value)
			}
			if field.slotIdx > 0 {
				field.value = field.value[:field.slotIdx-1] + field.value[field.slotIdx:]
				field.slotIdx--
			}
		}
		return false, nil

	default:
		if key < 32 || key > 126 {
			return false, nil
		}
		return false, s.insertChar(byte(key))
	}
}

func (s *readSession) insertChar(ch byte) error {
	field := s.current()
	if field.spec.width > 0 {
		value, slotIdx, err := pictureInsertChar(field.spec, field.value, field.slotIdx, ch)
		if err != nil {
			return fmt.Errorf("*** %s: %v", field.def.Name, err)
		}
		field.value = value
		field.slotIdx = slotIdx
		return nil
	}

	if field.slotIdx > len(field.value) {
		field.slotIdx = len(field.value)
	}
	if len(field.value) >= field.maxWidth {
		return nil
	}
	field.value = field.value[:field.slotIdx] + string(ch) + field.value[field.slotIdx:]
	field.slotIdx++
	return nil
}

func (s *readSession) validateCurrent() error {
	field := s.current()
	if err := validateReadValue(field.value, field.def.Picture); err != nil {
		return fmt.Errorf("*** %s: %v", field.def.Name, err)
	}
	return nil
}

func (s *readSession) commitAll() error {
	for _, field := range s.fields {
		if err := validateReadValue(field.value, field.def.Picture); err != nil {
			return fmt.Errorf("*** %s: %v", field.def.Name, err)
		}
		if err := commitReadValue(s.ctx, field.def.Name, field.value); err != nil {
			return err
		}
	}
	return nil
}

func commitReadValue(ctx *context.Context, name, value string) error {
	value = strings.TrimRight(value, " ")

	if _, ok := ctx.Variables.Get(name); ok {
		return commitReadMemvar(ctx, name, value)
	}

	if committed, err := commitReadTableField(ctx, name, value); err != nil {
		return err
	} else if committed {
		return nil
	}

	return ctx.Variables.Set(name, value)
}

func commitReadMemvar(ctx *context.Context, name, value string) error {
	existing, _ := ctx.Variables.Get(name)
	switch existing.(type) {
	case float64:
		if value == "" {
			return ctx.Variables.Set(name, float64(0))
		}
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("*** Invalid numeric value for %s: %s", name, value)
		}
		return ctx.Variables.Set(name, num)
	case int:
		if value == "" {
			return ctx.Variables.Set(name, 0)
		}
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("*** Invalid numeric value for %s: %s", name, value)
		}
		return ctx.Variables.Set(name, int(num))
	case bool:
		if value == "" {
			return ctx.Variables.Set(name, false)
		}
		switch strings.ToUpper(value)[0] {
		case 'T', 'Y':
			return ctx.Variables.Set(name, true)
		case 'F', 'N':
			return ctx.Variables.Set(name, false)
		default:
			return fmt.Errorf("*** Invalid logical value for %s: %s", name, value)
		}
	default:
		return ctx.Variables.Set(name, value)
	}
}

func commitReadTableField(ctx *context.Context, name, value string) (bool, error) {
	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil || area.ActiveRecord == nil {
		return false, nil
	}

	fd, idx := area.Table.FieldByName(name)
	if fd == nil {
		return false, nil
	}

	parsed, err := parseFieldInput(*fd, value)
	if err != nil {
		return false, err
	}

	rec, err := replaceRecordField(area.Table, area.ActiveRecord, idx, parsed)
	if err != nil {
		return false, err
	}
	area.ActiveRecord = rec
	return true, nil
}
