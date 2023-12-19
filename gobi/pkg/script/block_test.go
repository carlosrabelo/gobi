package script

import "testing"

func TestBuildIfBlocksSimple(t *testing.T) {
	prog, err := ParseSource("x.prg", "IF .T.\nSTORE 1 TO a\nELSE\nSTORE 2 TO b\nENDIF\n")
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	block, ok := prog.IfBlockAt(0)
	if !ok {
		t.Fatal("expected IF block at index 0")
	}
	if block.ElseIndex != 2 || block.EndIndex != 4 {
		t.Fatalf("unexpected block: %#v", block)
	}

	skip, ok := prog.SkipAfterElse(2)
	if !ok || skip != 5 {
		t.Fatalf("expected skip target 5, got %d ok=%v", skip, ok)
	}
}

func TestBuildIfBlocksWithoutElse(t *testing.T) {
	prog, err := ParseSource("x.prg", "IF .F.\nSTORE 1 TO a\nENDIF\n")
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	block, ok := prog.IfBlockAt(0)
	if !ok || block.ElseIndex != -1 || block.EndIndex != 2 {
		t.Fatalf("unexpected block: %#v ok=%v", block, ok)
	}
}

func TestBuildIfBlocksNested(t *testing.T) {
	source := "IF .T.\nIF .F.\nSTORE 1 TO inner\nENDIF\nSTORE 2 TO outer\nENDIF\n"
	prog, err := ParseSource("x.prg", source)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	outer, ok := prog.IfBlockAt(0)
	if !ok || outer.EndIndex != 5 {
		t.Fatalf("unexpected outer block: %#v", outer)
	}

	inner, ok := prog.IfBlockAt(1)
	if !ok || inner.EndIndex != 3 {
		t.Fatalf("unexpected inner block: %#v", inner)
	}
}

func TestBuildIfBlocksUnmatchedEndif(t *testing.T) {
	_, err := ParseSource("x.prg", "STORE 1 TO a\nENDIF\n")
	if err == nil {
		t.Fatal("expected unmatched ENDIF error")
	}
}

func TestBuildIfBlocksUnclosedIf(t *testing.T) {
	_, err := ParseSource("x.prg", "IF .T.\nSTORE 1 TO a\n")
	if err == nil {
		t.Fatal("expected unclosed IF error")
	}
}

func TestBuildIfBlocksElseWithoutIf(t *testing.T) {
	_, err := ParseSource("x.prg", "ELSE\nSTORE 1 TO a\nENDIF\n")
	if err == nil {
		t.Fatal("expected ELSE without IF error")
	}
}
