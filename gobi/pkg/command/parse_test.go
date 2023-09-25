package command

import "testing"

func TestParseEmpty(t *testing.T) {
	cmd := Parse("")
	if cmd.Verb != "" {
		t.Fatalf("expected empty verb, got %q", cmd.Verb)
	}
}

func TestParseBlank(t *testing.T) {
	cmd := Parse("   ")
	if cmd.Verb != "" {
		t.Fatalf("expected empty verb, got %q", cmd.Verb)
	}
}

func TestParseVerbOnly(t *testing.T) {
	cmd := Parse("QUIT")
	if cmd.Verb != "QUIT" {
		t.Fatalf("expected QUIT, got %q", cmd.Verb)
	}
	if cmd.RawVerb != "QUIT" {
		t.Fatalf("expected raw QUIT, got %q", cmd.RawVerb)
	}
	if cmd.Args != "" {
		t.Fatalf("expected empty args, got %q", cmd.Args)
	}
}

func TestParseVerbLowercase(t *testing.T) {
	cmd := Parse("quit")
	if cmd.Verb != "QUIT" {
		t.Fatalf("expected QUIT, got %q", cmd.Verb)
	}
	if cmd.RawVerb != "quit" {
		t.Fatalf("expected raw quit, got %q", cmd.RawVerb)
	}
}

func TestParseSimpleArgs(t *testing.T) {
	cmd := Parse("USE customers")
	if cmd.Verb != "USE" {
		t.Fatalf("expected USE, got %q", cmd.Verb)
	}
	if cmd.Args != "customers" {
		t.Fatalf("expected 'customers', got %q", cmd.Args)
	}
}

func TestParseMultipleArgs(t *testing.T) {
	cmd := Parse("LIST name, address, phone")
	if cmd.Verb != "LIST" {
		t.Fatalf("expected LIST, got %q", cmd.Verb)
	}
	if cmd.Args != "name, address, phone" {
		t.Fatalf("expected 'name, address, phone', got %q", cmd.Args)
	}
}

func TestParseQuotedString(t *testing.T) {
	cmd := Parse(`REPLACE name WITH "John Doe"`)
	if cmd.Verb != "REPLACE" {
		t.Fatalf("expected REPLACE, got %q", cmd.Verb)
	}
	if cmd.Args != `name WITH "John Doe"` {
		t.Fatalf("expected 'name WITH \"John Doe\"', got %q", cmd.Args)
	}
}

func TestParseNumber(t *testing.T) {
	cmd := Parse("GOTO 5")
	if cmd.Verb != "GOTO" {
		t.Fatalf("expected GOTO, got %q", cmd.Verb)
	}
	if cmd.Args != "5" {
		t.Fatalf("expected '5', got %q", cmd.Args)
	}
}

func TestParseQuestion(t *testing.T) {
	cmd1 := Parse("? 1 + 1")
	if cmd1.Verb != "?" {
		t.Fatalf("expected ?, got %q", cmd1.Verb)
	}
	if cmd1.Args != "1 + 1" {
		t.Fatalf("expected '1 + 1', got %q", cmd1.Args)
	}

	cmd2 := Parse("?1+1")
	if cmd2.Verb != "?" {
		t.Fatalf("expected ?, got %q", cmd2.Verb)
	}
	if cmd2.Args != "1+1" {
		t.Fatalf("expected '1+1', got %q", cmd2.Args)
	}
}

func TestParseDoubleQuestion(t *testing.T) {
	cmd1 := Parse(`?? "Hello"`)
	if cmd1.Verb != "??" {
		t.Fatalf("expected ??, got %q", cmd1.Verb)
	}
	if cmd1.Args != `"Hello"` {
		t.Fatalf("expected '\"Hello\"', got %q", cmd1.Args)
	}

	cmd2 := Parse(`??"Hello"`)
	if cmd2.Verb != "??" {
		t.Fatalf("expected ??, got %q", cmd2.Verb)
	}
	if cmd2.Args != `"Hello"` {
		t.Fatalf("expected '\"Hello\"', got %q", cmd2.Args)
	}
}

func TestParseForClause(t *testing.T) {
	cmd := Parse("LIST FOR name = \"Smith\"")
	if cmd.Verb != "LIST" {
		t.Fatalf("expected LIST, got %q", cmd.Verb)
	}
	if cmd.ForClause != "name = \"Smith\"" {
		t.Fatalf("expected 'name = \"Smith\"', got %q", cmd.ForClause)
	}
	if cmd.Args != "" {
		t.Fatalf("expected empty args, got %q", cmd.Args)
	}
}

func TestParseForClauseLowercase(t *testing.T) {
	cmd := Parse("list for name = 'Smith'")
	if cmd.Verb != "LIST" {
		t.Fatalf("expected LIST, got %q", cmd.Verb)
	}
	if cmd.ForClause != "name = 'Smith'" {
		t.Fatalf("expected 'name = \\'Smith\\'', got %q", cmd.ForClause)
	}
}

func TestParseArgsAndFor(t *testing.T) {
	cmd := Parse("LIST name, address FOR age > 21")
	if cmd.Verb != "LIST" {
		t.Fatalf("expected LIST, got %q", cmd.Verb)
	}
	if cmd.Args != "name, address" {
		t.Fatalf("expected 'name, address', got %q", cmd.Args)
	}
	if cmd.ForClause != "age > 21" {
		t.Fatalf("expected 'age > 21', got %q", cmd.ForClause)
	}
}

func TestParseQuotedStringInFor(t *testing.T) {
	cmd := Parse(`LIST FOR name = "John Doe"`)
	if cmd.Verb != "LIST" {
		t.Fatalf("expected LIST, got %q", cmd.Verb)
	}
	if cmd.ForClause != `name = "John Doe"` {
		t.Fatalf("expected 'name = \"John Doe\"', got %q", cmd.ForClause)
	}
}

func TestParseSingleQuotes(t *testing.T) {
	cmd := Parse("LIST FOR name = 'John Doe'")
	if cmd.Verb != "LIST" {
		t.Fatalf("expected LIST, got %q", cmd.Verb)
	}
	if cmd.ForClause != "name = 'John Doe'" {
		t.Fatalf("expected \"name = 'John Doe'\", got %q", cmd.ForClause)
	}
}

func TestParseExtraWhitespace(t *testing.T) {
	cmd := Parse("  LIST    name ,  address   FOR   age   >   21  ")
	if cmd.Verb != "LIST" {
		t.Fatalf("expected LIST, got %q", cmd.Verb)
	}
	if cmd.RawVerb != "LIST" {
		t.Fatalf("expected raw LIST, got %q", cmd.RawVerb)
	}
	if cmd.Args != "name , address" {
		t.Fatalf("expected 'name , address', got %q", cmd.Args)
	}
	if cmd.ForClause != "age > 21" {
		t.Fatalf("expected 'age > 21', got %q", cmd.ForClause)
	}
}

