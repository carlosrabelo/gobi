package command

import "testing"

func TestSplitLISTFor(t *testing.T) {
	cmd := Split(`LIST NAME FOR AGE > 30`)
	if cmd.Verb != "LIST" {
		t.Fatalf("verb: got %q", cmd.Verb)
	}
	if cmd.Args != "NAME" {
		t.Fatalf("args: got %q", cmd.Args)
	}
	if cmd.Clauses["FOR"] != "AGE > 30" {
		t.Fatalf("FOR: got %q", cmd.Clauses["FOR"])
	}
}

func TestSplitQuotedFOR(t *testing.T) {
	cmd := Split(`LIST "FOR" NAME`)
	if cmd.Clauses["FOR"] != "" {
		t.Fatalf("quoted FOR must not start a clause, got %q", cmd.Clauses["FOR"])
	}
	if cmd.Args != `"FOR" NAME` {
		t.Fatalf("args: got %q", cmd.Args)
	}
}

func TestSplitREMARK(t *testing.T) {
	cmd := Split(`REMARK see FOR later`)
	if cmd.Verb != "REMARK" {
		t.Fatalf("verb: got %q", cmd.Verb)
	}
	if cmd.Args != "see FOR later" {
		t.Fatalf("args: got %q", cmd.Args)
	}
	if cmd.Clauses["FOR"] != "" {
		t.Fatalf("REMARK must not scan FOR, got %q", cmd.Clauses["FOR"])
	}
}

func TestSplitDoubleQuestion(t *testing.T) {
	cmd := Split(`?? NAME`)
	if cmd.Verb != "??" {
		t.Fatalf("verb: got %q", cmd.Verb)
	}
	if cmd.Args != "NAME" {
		t.Fatalf("args: got %q", cmd.Args)
	}
}

func TestSplitQuestionNoSpace(t *testing.T) {
	cmd := Split(`?NAME`)
	if cmd.Verb != "?" {
		t.Fatalf("verb: got %q", cmd.Verb)
	}
	if cmd.Args != "NAME" {
		t.Fatalf("args: got %q", cmd.Args)
	}
}

func TestSplitAmpComment(t *testing.T) {
	cmd := Split(`LIST NAME && trailing`)
	if cmd.Args != "NAME" {
		t.Fatalf("args: got %q", cmd.Args)
	}
}

func TestSplitFROM(t *testing.T) {
	cmd := Split(`COPY TO x FROM y`)
	if cmd.Clauses["TO"] != "x" {
		t.Fatalf("TO: got %q", cmd.Clauses["TO"])
	}
	if cmd.Clauses["FROM"] != "y" {
		t.Fatalf("FROM: got %q", cmd.Clauses["FROM"])
	}
}

func TestSplitFROMOnStaysInArgs(t *testing.T) {
	cmd := Split(`SET FROM ON`)
	if cmd.Clauses["FROM"] != "" {
		t.Fatalf("FROM ON must not take clause, got %q", cmd.Clauses["FROM"])
	}
	if !containsAll(cmd.Args, "FROM", "ON") {
		t.Fatalf("args: got %q", cmd.Args)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !containsWord(s, p) {
			return false
		}
	}
	return true
}

func containsWord(s, w string) bool {
	for _, t := range tokenize(s) {
		if t == w {
			return true
		}
	}
	return false
}
