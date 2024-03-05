package script

import (
	"strings"
	"testing"
)

func TestBuildCaseBlocks(t *testing.T) {
	source := "DO CASE\n" +
		"CASE x = 1\n" +
		"STORE 1 TO y\n" +
		"CASE x = 2\n" +
		"STORE 2 TO y\n" +
		"OTHERWISE\n" +
		"STORE 0 TO y\n" +
		"ENDCASE\n"

	prog, err := ParseSource("case.prg", source)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	block, ok := prog.CaseBlockAt(0)
	if !ok {
		t.Fatal("expected CASE block at index 0")
	}
	if block.DoIndex != 0 || block.EndIndex != 7 {
		t.Fatalf("unexpected block bounds: %#v", block)
	}
	if len(block.CaseIndexes) != 2 || block.CaseIndexes[0] != 1 || block.CaseIndexes[1] != 3 {
		t.Fatalf("unexpected CASE indexes: %#v", block.CaseIndexes)
	}
	if block.OtherwiseIndex != 5 {
		t.Fatalf("unexpected OTHERWISE index: %d", block.OtherwiseIndex)
	}

	// Fall-through skips from every branch land after ENDCASE.
	for _, branch := range []int{1, 3, 5} {
		skip, ok := prog.SkipAfterCaseBranch(branch)
		if !ok || skip != 8 {
			t.Fatalf("skip for branch %d = %d ok=%v, want 8", branch, skip, ok)
		}
	}
}

func TestBuildCaseBlocksWithoutOtherwise(t *testing.T) {
	prog, err := ParseSource("case.prg", "DO CASE\nCASE x = 1\nSTORE 1 TO y\nENDCASE\n")
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	block, ok := prog.CaseBlockAt(0)
	if !ok || block.OtherwiseIndex != -1 {
		t.Fatalf("unexpected block: %#v ok=%v", block, ok)
	}
}

func TestBuildCaseBlocksNested(t *testing.T) {
	source := "DO CASE\n" +
		"CASE x = 1\n" +
		"DO CASE\n" +
		"CASE y = 1\n" +
		"STORE 1 TO z\n" +
		"ENDCASE\n" +
		"ENDCASE\n"

	prog, err := ParseSource("nested.prg", source)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	outer, ok := prog.CaseBlockAt(0)
	if !ok || outer.EndIndex != 6 || len(outer.CaseIndexes) != 1 {
		t.Fatalf("unexpected outer block: %#v ok=%v", outer, ok)
	}
	inner, ok := prog.CaseBlockAt(2)
	if !ok || inner.EndIndex != 5 || len(inner.CaseIndexes) != 1 || inner.CaseIndexes[0] != 3 {
		t.Fatalf("unexpected inner block: %#v ok=%v", inner, ok)
	}
}

func TestBuildCaseBlocksErrors(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   string
	}{
		{"case outside", "CASE x = 1\n", "CASE without matching DO CASE"},
		{"otherwise outside", "OTHERWISE\n", "OTHERWISE without matching DO CASE"},
		{"endcase outside", "ENDCASE\n", "ENDCASE without matching DO CASE"},
		{"unclosed", "DO CASE\nCASE x = 1\n", "unclosed DO CASE"},
		{"case after otherwise", "DO CASE\nOTHERWISE\nCASE x = 1\nENDCASE\n", "CASE after OTHERWISE"},
		{"double otherwise", "DO CASE\nOTHERWISE\nOTHERWISE\nENDCASE\n", "multiple OTHERWISE"},
	}

	for _, tc := range cases {
		_, err := ParseSource("bad.prg", tc.source)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: expected %q error, got %v", tc.name, tc.want, err)
		}
	}
}

func TestIsDoCase(t *testing.T) {
	prog, err := ParseSource("x.prg", "do case\nENDCASE\nDO payroll\n")
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	cmds := prog.Commands()
	if !IsDoCase(cmds[0].Command) {
		t.Fatalf("expected lowercase do case to match: %#v", cmds[0].Command)
	}
	if IsDoCase(cmds[2].Command) {
		t.Fatalf("DO with filename must not match: %#v", cmds[2].Command)
	}
}
