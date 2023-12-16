package script

import "testing"

func sampleProgram() *Program {
	prog, err := ParseSource("sample.prg", "* setup\nUSE people\nLIST\nRETURN\nSTORE 1 TO x\n")
	if err != nil {
		panic(err)
	}
	return prog
}

func TestControllerStartsAtFirstCommand(t *testing.T) {
	ctrl := NewController(sampleProgram())

	if ctrl.Index() != 0 {
		t.Fatalf("expected index 0, got %d", ctrl.Index())
	}
	line, ok := ctrl.Current()
	if !ok || line.Command.Verb != "USE" {
		t.Fatalf("unexpected current line: %#v", line)
	}
}

func TestControllerAdvanceStepsCommands(t *testing.T) {
	ctrl := NewController(sampleProgram())

	if !ctrl.Advance() {
		t.Fatal("expected more commands after first advance")
	}
	line, ok := ctrl.Current()
	if !ok || line.Command.Verb != "LIST" {
		t.Fatalf("unexpected line after advance: %#v", line)
	}

	ctrl.Advance()
	line, ok = ctrl.Current()
	if !ok || line.Command.Verb != "RETURN" {
		t.Fatalf("unexpected line after second advance: %#v", line)
	}
}

func TestControllerAtEnd(t *testing.T) {
	ctrl := NewController(sampleProgram())

	for ctrl.Advance() {
	}
	if !ctrl.AtEnd() {
		t.Fatal("expected controller at end")
	}
	if _, ok := ctrl.Current(); ok {
		t.Fatal("expected no current line at end")
	}
}

func TestControllerSetIndex(t *testing.T) {
	ctrl := NewController(sampleProgram())

	if err := ctrl.SetIndex(2); err != nil {
		t.Fatalf("SetIndex: %v", err)
	}
	line, ok := ctrl.Current()
	if !ok || line.Command.Verb != "RETURN" {
		t.Fatalf("unexpected line at index 2: %#v", line)
	}
	if err := ctrl.SetIndex(99); err == nil {
		t.Fatal("expected out of range error")
	}
}

func TestControllerJumpToLine(t *testing.T) {
	ctrl := NewController(sampleProgram())

	if err := ctrl.JumpToLine(3); err != nil {
		t.Fatalf("JumpToLine: %v", err)
	}
	line, ok := ctrl.Current()
	if !ok || line.Command.Verb != "LIST" {
		t.Fatalf("unexpected line after jump: %#v", line)
	}
}

func TestControllerPushPopFrame(t *testing.T) {
	outer := sampleProgram()
	inner, err := ParseSource("inner.prg", "STORE 2 TO y\n")
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	ctrl := NewController(outer)
	if err := ctrl.PushFrame(inner); err != nil {
		t.Fatalf("PushFrame: %v", err)
	}
	if ctrl.Depth() != 2 {
		t.Fatalf("expected depth 2, got %d", ctrl.Depth())
	}
	line, ok := ctrl.Current()
	if !ok || line.Command.Verb != "STORE" {
		t.Fatalf("unexpected inner command: %#v", line)
	}

	if !ctrl.PopFrame() {
		t.Fatal("expected caller frame to remain")
	}
	if ctrl.Depth() != 1 {
		t.Fatalf("expected depth 1, got %d", ctrl.Depth())
	}
	line, ok = ctrl.Current()
	if !ok || line.Command.Verb != "USE" {
		t.Fatalf("expected restored outer frame: %#v", line)
	}
}
