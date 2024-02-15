package repl

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

const (
	browseTitleRow     = 0
	browseTitleCol     = 2
	browseHeaderRow    = 2
	browseFirstDataRow = 3
	browseRecNumWidth  = 6
	browseColGap       = 1
	browseLabelCol     = 1
)

type browseSession struct {
	ctx         *context.Context
	area        *context.WorkArea
	tbl         *dbf.Table
	rseeker     io.ReadSeeker
	in          io.Reader
	out         io.Writer
	tty         *os.File
	raw         bool
	colWidths   []int
	topRec      int
	leftCol     int
	curRec      int
	curCol      int
	visibleRows int
	statusRow   int
	editing     bool
	editValue   string
	editOrig    string
	editCursor  int
}

func handleBrowse(ctx *context.Context, cmd Command) error {
	area := ctx.GetActiveArea()
	if area == nil || area.Table == nil {
		return fmt.Errorf("*** No database file is in use")
	}

	rseeker, ok := area.Table.Underlying().(io.ReadSeeker)
	if !ok {
		return fmt.Errorf("*** Database stream is not seekable")
	}

	err := runBrowseMatrix(ctx, rseeker)
	if consoleErr := returnToConsole(ctx); consoleErr != nil && err == nil {
		err = consoleErr
	}
	return err
}

func runBrowseMatrix(ctx *context.Context, rseeker io.ReadSeeker) error {
	area := ctx.GetActiveArea()
	s := newBrowseSession(ctx, area, rseeker)
	s.ensureViewport()

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
				break
			}
			return fmt.Errorf("*** Error reading input: %w", err)
		}

		if s.handleKey(key) {
			break
		}
	}

	s.syncRecordPointer()
	return nil
}

func newBrowseSession(ctx *context.Context, area *context.WorkArea, rseeker io.ReadSeeker) *browseSession {
	s := &browseSession{
		ctx:       ctx,
		area:      area,
		tbl:       area.Table,
		rseeker:   rseeker,
		in:        ctx.Stdin,
		out:       ctx.Stdout,
		colWidths: browseColumnWidths(area.Table),
		curRec:    area.RecordNo,
		statusRow: ctx.Screen.CommandLineRow() - 1,
	}
	s.visibleRows = s.statusRow - browseFirstDataRow
	if s.visibleRows < 1 {
		s.visibleRows = 1
	}

	if f, ok := ctx.Stdin.(*os.File); ok && term.IsTerminal(f) {
		s.tty = f
		s.raw = true
	}
	return s
}

func browseColumnWidths(tbl *dbf.Table) []int {
	widths := make([]int, len(tbl.Fields))
	for i, fd := range tbl.Fields {
		width := len(fd.Name)
		if int(fd.Length) > width {
			width = int(fd.Length)
		}
		if width < 3 {
			width = 3
		}
		widths[i] = width
	}
	return widths
}

func (s *browseSession) ensureViewport() {
	recCount := int(s.tbl.Header.RecordCount)
	if recCount == 0 {
		s.topRec = 0
		s.curRec = 0
	} else {
		if s.curRec < 0 {
			s.curRec = 0
		}
		if s.curRec >= recCount {
			s.curRec = recCount - 1
		}
	}

	if s.curRec < s.topRec {
		s.topRec = s.curRec
	}
	if s.curRec >= s.topRec+s.visibleRows {
		s.topRec = s.curRec - s.visibleRows + 1
	}
	if s.topRec < 0 {
		s.topRec = 0
	}

	if s.curCol < s.leftCol {
		s.leftCol = s.curCol
	}
	visibleCols := s.visibleColumnCount()
	if visibleCols < 1 {
		visibleCols = 1
	}
	if s.curCol >= s.leftCol+visibleCols {
		s.leftCol = s.curCol - visibleCols + 1
	}
	if s.leftCol < 0 {
		s.leftCol = 0
	}
	if s.curCol >= len(s.colWidths) && len(s.colWidths) > 0 {
		s.curCol = len(s.colWidths) - 1
	}
}

func (s *browseSession) visibleColumnCount() int {
	available := s.ctx.Screen.Cols() - browseLabelCol - browseRecNumWidth - browseColGap
	if available <= 0 {
		return 0
	}

	count := 0
	used := 0
	for i := s.leftCol; i < len(s.colWidths); i++ {
		need := s.colWidths[i] + browseColGap
		if count > 0 && used+need > available {
			break
		}
		used += need
		count++
	}
	return count
}

func (s *browseSession) columnStart(fieldIdx int) int {
	col := browseLabelCol + browseRecNumWidth + browseColGap
	for i := s.leftCol; i < fieldIdx; i++ {
		col += s.colWidths[i] + browseColGap
	}
	return col
}

func (s *browseSession) draw() error {
	s.ctx.Screen.Clear()
	s.drawTitle()
	s.drawHeader()
	if err := s.drawRows(); err != nil {
		return err
	}
	s.drawStatus()

	var highlights []term.Highlight
	if intensityEnabled(s.ctx) {
		recCount := int(s.tbl.Header.RecordCount)
		if recCount > 0 {
			row := browseFirstDataRow + (s.curRec - s.topRec)
			if row >= browseFirstDataRow && row < s.statusRow {
				highlights = []term.Highlight{{
					Row:    row,
					Col:    s.columnStart(s.curCol),
					Length: s.colWidths[s.curCol],
				}}
			}
		}
	}

	if err := presentScreen(s.ctx, s.out, highlights); err != nil {
		return err
	}

	row, col := s.cursorPosition()
	return term.MoveTo(s.out, row, col)
}

func (s *browseSession) drawTitle() {
	title := fmt.Sprintf("BROWSE - %s", s.area.Alias)
	s.ctx.Screen.WriteAt(browseTitleRow, browseTitleCol, title)
}

func (s *browseSession) drawHeader() {
	s.ctx.Screen.WriteAt(browseHeaderRow, browseLabelCol, padBrowseCell("Rec#", browseRecNumWidth))

	visibleCols := s.visibleColumnCount()
	for i := 0; i < visibleCols; i++ {
		fieldIdx := s.leftCol + i
		name := strings.ToUpper(s.tbl.Fields[fieldIdx].Name)
		text := padBrowseCell(name, s.colWidths[fieldIdx])
		s.ctx.Screen.WriteAt(browseHeaderRow, s.columnStart(fieldIdx), text)
	}
}

func (s *browseSession) drawRows() error {
	recCount := int(s.tbl.Header.RecordCount)
	visibleCols := s.visibleColumnCount()

	for row := 0; row < s.visibleRows; row++ {
		recIdx := s.topRec + row
		screenRow := browseFirstDataRow + row
		if recIdx >= recCount {
			break
		}

		rec, err := s.tbl.ReadRecordAt(s.rseeker, recIdx)
		if err != nil {
			return fmt.Errorf("*** Error reading record %d: %w", recIdx+1, err)
		}

		recLabel := fmt.Sprintf("%5d", recIdx+1)
		if rec.Deleted {
			recLabel = "*" + recLabel[1:]
		}
		s.ctx.Screen.WriteAt(screenRow, browseLabelCol, padBrowseCell(recLabel, browseRecNumWidth))

		cells, err := browseRecordCells(s.tbl, rec)
		if err != nil {
			return err
		}

		for i := 0; i < visibleCols; i++ {
			fieldIdx := s.leftCol + i
			cellText := cells[fieldIdx]
			if recIdx == s.curRec && fieldIdx == s.curCol && s.editing {
				cellText = s.editValue
			}
			text := padBrowseCell(cellText, s.colWidths[fieldIdx])
			s.ctx.Screen.WriteAt(screenRow, s.columnStart(fieldIdx), text)
		}
	}
	return nil
}

func (s *browseSession) drawStatus() {
	recCount := int(s.tbl.Header.RecordCount)
	cur := 0
	if recCount > 0 {
		cur = s.curRec + 1
	}
	fieldName := ""
	if len(s.tbl.Fields) > 0 && s.curCol >= 0 && s.curCol < len(s.tbl.Fields) {
		fieldName = s.tbl.Fields[s.curCol].Name
	}
	mode := ""
	if s.editing {
		mode = "  [Editing]"
	}
	status := fmt.Sprintf("Record %d/%d  Field: %s%s", cur, recCount, fieldName, mode)
	s.ctx.Screen.WriteAt(s.statusRow, browseLabelCol, status)
}

func (s *browseSession) cursorPosition() (int, int) {
	recCount := int(s.tbl.Header.RecordCount)
	if recCount == 0 {
		return browseHeaderRow, s.columnStart(maxInt(0, s.curCol))
	}

	row := browseFirstDataRow + (s.curRec - s.topRec)
	if row < browseFirstDataRow {
		row = browseFirstDataRow
	}
	if row >= s.statusRow {
		row = s.statusRow - 1
	}

	col := s.columnStart(s.curCol)
	if s.editing {
		col += s.editCursor
		maxCol := s.columnStart(s.curCol) + s.colWidths[s.curCol]
		if col >= maxCol {
			col = maxCol - 1
		}
	}
	return row, col
}

func (s *browseSession) handleKey(key byte) bool {
	switch key {
	case editKeyCtrlQ, editKeyEsc:
		return true
	case replKeyUp:
		s.moveUp()
	case replKeyDown:
		s.moveDown()
	case replKeyLeft:
		s.moveLeft()
	case replKeyRight:
		s.moveRight()
	}
	return false
}

func (s *browseSession) handleEditKey(key byte) bool {
	switch key {
	case editKeyEsc:
		s.cancelEdit()
	case editKeyEnter:
		if err := s.commitEdit(); err != nil {
			reportValidationError(s.ctx, err)
		}
	case editKeyCtrlY:
		s.editValue = ""
		s.editCursor = 0
	case editKeyDel, editKeyCtrlH:
		if s.editCursor > 0 {
			s.editValue = s.editValue[:s.editCursor-1] + s.editValue[s.editCursor:]
			s.editCursor--
		}
	case replKeyUp:
		if err := s.commitEdit(); err != nil {
			reportValidationError(s.ctx, err)
		} else {
			s.moveUp()
		}
	case replKeyDown:
		if err := s.commitEdit(); err != nil {
			reportValidationError(s.ctx, err)
		} else {
			s.moveDown()
		}
	case replKeyLeft:
		if err := s.commitEdit(); err != nil {
			reportValidationError(s.ctx, err)
		} else {
			s.moveLeft()
		}
	case replKeyRight:
		if err := s.commitEdit(); err != nil {
			reportValidationError(s.ctx, err)
		} else {
			s.moveRight()
		}
	default:
		if key >= 32 && key < 127 {
			if err := s.insertEditChar(byte(key)); err != nil {
				fmt.Fprintln(s.ctx.Stderr, err.Error())
			}
		}
	}
	return false
}

func (s *browseSession) writable() (io.ReadWriteSeeker, bool) {
	wseeker, ok := s.area.Table.Underlying().(io.ReadWriteSeeker)
	return wseeker, ok
}

func (s *browseSession) beginEdit() error {
	if int(s.tbl.Header.RecordCount) == 0 {
		return nil
	}
	if _, ok := s.writable(); !ok {
		return fmt.Errorf("*** Database file is not writable")
	}

	cells, err := s.currentRecordCells()
	if err != nil {
		return err
	}

	s.editing = true
	s.editValue = cells[s.curCol]
	s.editOrig = s.editValue
	s.editCursor = len(s.editValue)
	return nil
}

func (s *browseSession) cancelEdit() {
	s.editing = false
	s.editValue = ""
	s.editOrig = ""
	s.editCursor = 0
}

func (s *browseSession) commitEdit() error {
	if !s.editing {
		return nil
	}

	wseeker, ok := s.writable()
	if !ok {
		s.cancelEdit()
		return fmt.Errorf("*** Database file is not writable")
	}

	rec, err := s.tbl.ReadRecordAt(s.rseeker, s.curRec)
	if err != nil {
		s.cancelEdit()
		return fmt.Errorf("*** Error reading record %d: %w", s.curRec+1, err)
	}

	fd := s.tbl.Fields[s.curCol]
	parsed, err := parseFieldInput(fd, s.editValue)
	if err != nil {
		return err
	}

	updated, err := replaceRecordField(s.tbl, rec, s.curCol, parsed)
	if err != nil {
		return fmt.Errorf("*** Error updating record %d: %w", s.curRec+1, err)
	}

	if err := s.tbl.WriteRecordAt(wseeker, s.curRec, updated); err != nil {
		return fmt.Errorf("*** Error writing record %d: %w", s.curRec+1, err)
	}

	s.area.RecordNo = s.curRec
	s.area.ActiveRecord = updated
	s.cancelEdit()
	return nil
}

func (s *browseSession) insertEditChar(ch byte) error {
	if !s.editing {
		return nil
	}

	fd := s.tbl.Fields[s.curCol]
	maxLen := int(fd.Length)
	if fd.Type == dbf.FieldTypeLogical {
		maxLen = 1
	}

	val := s.editValue
	if len(val) < s.editCursor {
		s.editCursor = len(val)
	}

	if fd.Type == dbf.FieldTypeLogical {
		s.editValue = strings.ToUpper(string(ch))[:1]
		s.editCursor = 1
		return nil
	}

	if len(val) < maxLen {
		val = val[:s.editCursor] + string(ch) + val[s.editCursor:]
	} else if s.editCursor < maxLen {
		val = val[:s.editCursor] + string(ch) + val[s.editCursor+1:]
	} else {
		return nil
	}

	if len(val) > maxLen {
		val = val[:maxLen]
	}

	s.editValue = val
	if s.editCursor < maxLen {
		s.editCursor++
	}
	return nil
}

func (s *browseSession) toggleRecordDelete() error {
	if int(s.tbl.Header.RecordCount) == 0 {
		return nil
	}

	wseeker, ok := s.writable()
	if !ok {
		return fmt.Errorf("*** Database file is not writable")
	}

	rec, err := s.tbl.ReadRecordAt(s.rseeker, s.curRec)
	if err != nil {
		return fmt.Errorf("*** Error reading record %d: %w", s.curRec+1, err)
	}

	var marked *dbf.Record
	if rec.Deleted {
		marked, err = markRecordRecalled(s.tbl, rec)
	} else {
		marked, err = markRecordDeleted(s.tbl, rec)
	}
	if err != nil {
		return fmt.Errorf("*** Error updating record %d: %w", s.curRec+1, err)
	}

	if err := s.tbl.WriteRecordAt(wseeker, s.curRec, marked); err != nil {
		return fmt.Errorf("*** Error writing record %d: %w", s.curRec+1, err)
	}

	s.area.RecordNo = s.curRec
	s.area.ActiveRecord = marked
	return nil
}

func (s *browseSession) currentRecordCells() ([]string, error) {
	rec, err := s.tbl.ReadRecordAt(s.rseeker, s.curRec)
	if err != nil {
		return nil, fmt.Errorf("*** Error reading record %d: %w", s.curRec+1, err)
	}
	return browseRecordCells(s.tbl, rec)
}

func (s *browseSession) moveUp() {
	s.cancelEdit()
	if s.curRec > 0 {
		s.curRec--
		s.ensureViewport()
	}
}

func (s *browseSession) moveDown() {
	s.cancelEdit()
	recCount := int(s.tbl.Header.RecordCount)
	if recCount == 0 {
		return
	}
	if s.curRec < recCount-1 {
		s.curRec++
		s.ensureViewport()
	}
}

func (s *browseSession) moveLeft() {
	s.cancelEdit()
	if s.curCol > 0 {
		s.curCol--
		s.ensureViewport()
	}
}

func (s *browseSession) moveRight() {
	s.cancelEdit()
	if len(s.colWidths) == 0 {
		return
	}
	if s.curCol < len(s.colWidths)-1 {
		s.curCol++
		s.ensureViewport()
	}
}

func (s *browseSession) syncRecordPointer() {
	recCount := int(s.tbl.Header.RecordCount)
	if recCount == 0 {
		s.area.RecordNo = 0
		s.area.ActiveRecord = nil
		return
	}

	rec, err := s.tbl.ReadRecordAt(s.rseeker, s.curRec)
	if err != nil {
		return
	}
	s.area.RecordNo = s.curRec
	s.area.ActiveRecord = rec
}

func browseRecordCells(tbl *dbf.Table, rec *dbf.Record) ([]string, error) {
	cells := make([]string, len(tbl.Fields))
	for i, fd := range tbl.Fields {
		val, err := rec.DecodeField(tbl, i)
		if err != nil {
			return nil, err
		}
		cells[i] = fieldValueToEditString(fd, val)
	}
	return cells, nil
}

func padBrowseCell(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if len(text) > width {
		text = text[:width]
	}
	return fmt.Sprintf("%-*s", width, text)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
