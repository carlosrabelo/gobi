package script

import "testing"

func TestBuildWhileBlocksSimple(t *testing.T) {
	source := "STORE 0 TO n\nDO WHILE n < 3\nSTORE n + 1 TO n\nENDDO\n"
	prog, err := ParseSource("x.prg", source)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	block, ok := prog.WhileBlockAt(1)
	if !ok {
		t.Fatal("expected DO WHILE block at index 1")
	}
	if block.DoIndex != 1 || block.StartIndex != 2 || block.EndIndex != 3 {
		t.Fatalf("unexpected block: %#v", block)
	}

	byEnd, ok := prog.WhileBlockForEnd(3)
	if !ok || byEnd.DoIndex != 1 {
		t.Fatalf("unexpected ENDDO lookup: %#v ok=%v", byEnd, ok)
	}
}

func TestBuildWhileBlocksNested(t *testing.T) {
	source := "DO WHILE .T.\nDO WHILE .F.\nSTORE 1 TO inner\nENDDO\nENDDO\n"
	prog, err := ParseSource("x.prg", source)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	outer, ok := prog.WhileBlockAt(0)
	if !ok || outer.EndIndex != 4 {
		t.Fatalf("unexpected outer block: %#v", outer)
	}

	inner, ok := prog.WhileBlockAt(1)
	if !ok || inner.EndIndex != 3 {
		t.Fatalf("unexpected inner block: %#v", inner)
	}
}

func TestBuildWhileBlocksUnmatchedEnddo(t *testing.T) {
	_, err := ParseSource("x.prg", "STORE 1 TO n\nENDDO\n")
	if err == nil {
		t.Fatal("expected unmatched ENDDO error")
	}
}

func TestBuildWhileBlocksUnclosedDoWhile(t *testing.T) {
	_, err := ParseSource("x.prg", "DO WHILE .T.\nSTORE 1 TO n\n")
	if err == nil {
		t.Fatal("expected unclosed DO WHILE error")
	}
}

func TestWhileEnclosingAtNested(t *testing.T) {
	source := "DO WHILE .T.\nDO WHILE .F.\nSTORE 1 TO inner\nLOOP\nENDDO\nEXIT\nENDDO\n"
	prog, err := ParseSource("x.prg", source)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	innerLoop, ok := prog.WhileEnclosingAt(3)
	if !ok || innerLoop.DoIndex != 1 {
		t.Fatalf("expected inner LOOP target at index 1, got %#v ok=%v", innerLoop, ok)
	}

	outerExit, ok := prog.WhileEnclosingAt(5)
	if !ok || outerExit.DoIndex != 0 {
		t.Fatalf("expected outer EXIT context at index 0, got %#v ok=%v", outerExit, ok)
	}

	if _, ok := prog.WhileEnclosingAt(0); ok {
		t.Fatal("expected DO WHILE header not to be enclosed")
	}
	if _, ok := prog.WhileEnclosingAt(6); ok {
		t.Fatal("expected ENDDO line not to be enclosed")
	}
}

func TestBuildWhileBlocksIgnoresDoFile(t *testing.T) {
	source := "DO helper\nENDDO\n"
	_, err := ParseSource("x.prg", source)
	if err == nil {
		t.Fatal("expected ENDDO without DO WHILE error for DO file + ENDDO")
	}
}
