package script

import (
	"os"
	"strings"
	"testing"
)

func TestParseSourceRemarksAndCommands(t *testing.T) {
	source := "REMARK Dept list program\r\n" +
		"* load data\r\n" +
		"\r\n" +
		"USE people\r\n" +
		"LIST FOR dept = 3\r\n" +
		"RETURN\r\n"

	prog, err := ParseSource("deptlist.prg", source)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	if len(prog.Lines) != 7 {
		t.Fatalf("expected 7 lines, got %d", len(prog.Lines))
	}

	if prog.Lines[0].Kind != LineCommand || prog.Lines[0].Command.Verb != "REMARK" ||
		prog.Lines[0].Command.Args != "Dept list program" {
		t.Fatalf("unexpected first line: %#v", prog.Lines[0])
	}
	if prog.Lines[1].Kind != LineRemark || prog.Lines[1].Remark != "load data" {
		t.Fatalf("unexpected second line: %#v", prog.Lines[1])
	}
	if prog.Lines[2].Kind != LineEmpty {
		t.Fatalf("expected empty line at index 2, got %#v", prog.Lines[2])
	}
	if prog.Lines[3].Kind != LineCommand || prog.Lines[3].Command.Verb != "USE" {
		t.Fatalf("unexpected USE line: %#v", prog.Lines[3])
	}
	if prog.Lines[4].Command.Verb != "LIST" || prog.Lines[4].Command.ForClause != "dept = 3" {
		t.Fatalf("unexpected LIST line: %#v", prog.Lines[4])
	}
	if prog.Lines[5].Command.Verb != "RETURN" {
		t.Fatalf("unexpected RETURN line: %#v", prog.Lines[5])
	}
	if prog.Lines[6].Kind != LineEmpty {
		t.Fatalf("expected trailing empty line, got %#v", prog.Lines[6])
	}
}

func TestParseSourceTextBlock(t *testing.T) {
	source := "TEXT\n" +
		"Hello, world\n" +
		"\n" +
		"* not a comment here\n" +
		"  indented line\n" +
		"ENDTEXT\n" +
		"RETURN\n"

	prog, err := ParseSource("text.prg", source)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	textLine := prog.Lines[0]
	if textLine.Kind != LineCommand || textLine.Command.Verb != "TEXT" {
		t.Fatalf("expected TEXT command line, got %#v", textLine)
	}
	want := []string{"Hello, world", "", "* not a comment here", "  indented line"}
	if len(textLine.Text) != len(want) {
		t.Fatalf("expected %d body lines, got %d", len(want), len(textLine.Text))
	}
	for i, line := range want {
		if textLine.Text[i] != line {
			t.Fatalf("body line %d = %q, want %q", i, textLine.Text[i], line)
		}
	}

	// Body lines and ENDTEXT must be marked literal and excluded from commands.
	for i := 1; i <= 5; i++ {
		if prog.Lines[i].Kind != LineText {
			t.Fatalf("expected LineText at index %d, got %#v", i, prog.Lines[i])
		}
	}
	cmds := prog.Commands()
	if len(cmds) != 2 || cmds[0].Command.Verb != "TEXT" || cmds[1].Command.Verb != "RETURN" {
		t.Fatalf("unexpected commands: %#v", cmds)
	}
	if cmds[1].Number != 7 {
		t.Fatalf("expected RETURN on line 7, got %d", cmds[1].Number)
	}
}

func TestParseSourceUnterminatedTextBlock(t *testing.T) {
	_, err := ParseSource("bad.prg", "STORE 1 TO x\nTEXT\nno end\n")
	if err == nil {
		t.Fatal("expected error for unterminated TEXT block")
	}
	if !strings.Contains(err.Error(), "unterminated TEXT block at line 2") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseSourceEmptyTextBlock(t *testing.T) {
	prog, err := ParseSource("empty.prg", "TEXT\nENDTEXT\n")
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	if prog.Lines[0].Command.Verb != "TEXT" || len(prog.Lines[0].Text) != 0 {
		t.Fatalf("expected empty TEXT body, got %#v", prog.Lines[0])
	}
}

func TestParseSourceLineNumbers(t *testing.T) {
	prog, err := ParseSource("x.prg", "QUIT\nRETURN\n")
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	if prog.Lines[0].Number != 1 || prog.Lines[1].Number != 2 {
		t.Fatalf("unexpected line numbers: %#v", prog.Lines)
	}
}

func TestProgramCommandsFiltersExecutableLines(t *testing.T) {
	prog, err := ParseSource("x.prg", "* setup\n\nUSE people\nRETURN\n")
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	cmds := prog.Commands()
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}
	if cmds[0].Command.Verb != "USE" || cmds[1].Command.Verb != "RETURN" {
		t.Fatalf("unexpected commands: %#v", cmds)
	}
}

func TestReadProgramFromDisk(t *testing.T) {
	tempDir := t.TempDir()
	path := tempDir + "/sample.prg"
	content := "STORE 1 TO one\nSTORE 'abc' TO name\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	prog, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(prog.Commands()) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(prog.Commands()))
	}
	if prog.Commands()[1].Command.ToClause != "name" {
		t.Fatalf("unexpected second command: %#v", prog.Commands()[1])
	}
}

func TestParseSourceRemarkOnly(t *testing.T) {
	prog, err := ParseSource("x.prg", "REMARK\n")
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	if prog.Lines[0].Kind != LineCommand || prog.Lines[0].Command.Verb != "REMARK" ||
		prog.Lines[0].Command.Args != "" {
		t.Fatalf("unexpected REMARK line: %#v", prog.Lines[0])
	}
}

func TestParseSourceNoteIsSilentRemark(t *testing.T) {
	prog, err := ParseSource("x.prg", "NOTE setup data\nnote lowercase too\nNOTE\nNOTEBOOK\nUSE people\n")
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	if prog.Lines[0].Kind != LineRemark || prog.Lines[0].Remark != "setup data" {
		t.Fatalf("unexpected NOTE line: %#v", prog.Lines[0])
	}
	if prog.Lines[1].Kind != LineRemark || prog.Lines[1].Remark != "lowercase too" {
		t.Fatalf("unexpected lowercase note line: %#v", prog.Lines[1])
	}
	if prog.Lines[2].Kind != LineRemark || prog.Lines[2].Remark != "" {
		t.Fatalf("unexpected bare NOTE line: %#v", prog.Lines[2])
	}
	// A word merely starting with NOTE is a command, not a comment.
	if prog.Lines[3].Kind != LineCommand || prog.Lines[3].Command.Verb != "NOTEBOOK" {
		t.Fatalf("unexpected NOTEBOOK line: %#v", prog.Lines[3])
	}

	cmds := prog.Commands()
	if len(cmds) != 2 || cmds[0].Command.Verb != "NOTEBOOK" || cmds[1].Command.Verb != "USE" {
		t.Fatalf("unexpected commands: %#v", cmds)
	}
}

func TestParseSourceRemarkKeepsClauseKeywords(t *testing.T) {
	prog, err := ParseSource("x.prg", "REMARK send output TO the printer FOR review\n")
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	line := prog.Lines[0]
	if line.Kind != LineCommand || line.Command.Verb != "REMARK" {
		t.Fatalf("unexpected REMARK line: %#v", line)
	}
	if line.Command.Args != "send output TO the printer FOR review" {
		t.Fatalf("REMARK text mangled: %q", line.Command.Args)
	}
	if line.Command.ToClause != "" || line.Command.ForClause != "" {
		t.Fatalf("REMARK must not extract clauses: %#v", line.Command)
	}
}

func TestParseSourcePreservesSourceText(t *testing.T) {
	prog, err := ParseSource("x.prg", "  LIST name  \n")
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	if prog.Lines[0].Source != "  LIST name  " {
		t.Fatalf("expected preserved source, got %q", prog.Lines[0].Source)
	}
}
