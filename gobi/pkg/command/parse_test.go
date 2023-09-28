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

func TestParseRemarkKeepsRawText(t *testing.T) {
	cmd := Parse("remark Send results TO printer FOR review")
	if cmd.Verb != "REMARK" {
		t.Fatalf("expected REMARK, got %q", cmd.Verb)
	}
	if cmd.Args != "Send results TO printer FOR review" {
		t.Fatalf("REMARK text mangled: %q", cmd.Args)
	}
	if cmd.ToClause != "" || cmd.ForClause != "" || cmd.WhileClause != "" || cmd.FromClause != "" {
		t.Fatalf("REMARK must not extract clauses: %#v", cmd)
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

func TestParseWhileClause(t *testing.T) {
	cmd := Parse("LIST WHILE active = .T.")
	if cmd.Verb != "LIST" {
		t.Fatalf("expected LIST, got %q", cmd.Verb)
	}
	if cmd.Args != "" {
		t.Fatalf("expected empty args, got %q", cmd.Args)
	}
	if cmd.WhileClause != "active = .T." {
		t.Fatalf("expected 'active = .T.', got %q", cmd.WhileClause)
	}
}

func TestParseToClause(t *testing.T) {
	cmd := Parse("SAVE TO backup.mem")
	if cmd.Verb != "SAVE" {
		t.Fatalf("expected SAVE, got %q", cmd.Verb)
	}
	if cmd.ToClause != "backup.mem" {
		t.Fatalf("expected 'backup.mem', got %q", cmd.ToClause)
	}
}

func TestParseRestoreFromClause(t *testing.T) {
	cmd := Parse("RESTORE FROM memfile")
	if cmd.Verb != "RESTORE" {
		t.Fatalf("expected RESTORE, got %q", cmd.Verb)
	}
	if cmd.FromClause != "memfile" {
		t.Fatalf("expected 'memfile', got %q", cmd.FromClause)
	}
}

func TestParseReleaseAll(t *testing.T) {
	cmd := Parse("RELEASE ALL")
	if cmd.Verb != "RELEASE" {
		t.Fatalf("expected RELEASE, got %q", cmd.Verb)
	}
	if cmd.Args != "ALL" {
		t.Fatalf("expected ALL, got %q", cmd.Args)
	}
}

func TestParseReleaseVariable(t *testing.T) {
	cmd := Parse("RELEASE another, third")
	if cmd.Verb != "RELEASE" {
		t.Fatalf("expected RELEASE, got %q", cmd.Verb)
	}
	if cmd.Args != "another, third" {
		t.Fatalf("expected 'another, third', got %q", cmd.Args)
	}
}

func TestParseDoFilename(t *testing.T) {
	cmd := Parse("DO deptlist")
	if cmd.Verb != "DO" {
		t.Fatalf("expected DO, got %q", cmd.Verb)
	}
	if cmd.Args != "deptlist" {
		t.Fatalf("expected deptlist, got %q", cmd.Args)
	}
}

func TestParseForAndWhile(t *testing.T) {
	cmd := Parse("LIST name FOR age > 21 WHILE active")
	if cmd.Verb != "LIST" {
		t.Fatalf("expected LIST, got %q", cmd.Verb)
	}
	if cmd.Args != "name" {
		t.Fatalf("expected 'name', got %q", cmd.Args)
	}
	if cmd.ForClause != "age > 21" {
		t.Fatalf("expected 'age > 21', got %q", cmd.ForClause)
	}
	if cmd.WhileClause != "active" {
		t.Fatalf("expected 'active', got %q", cmd.WhileClause)
	}
}

func TestParseForAndTo(t *testing.T) {
	cmd := Parse("COPY TO backup FOR id > 0")
	if cmd.Verb != "COPY" {
		t.Fatalf("expected COPY, got %q", cmd.Verb)
	}
	if cmd.ToClause != "backup" {
		t.Fatalf("expected 'backup', got %q", cmd.ToClause)
	}
	if cmd.ForClause != "id > 0" {
		t.Fatalf("expected 'id > 0', got %q", cmd.ForClause)
	}
}

func TestParseAllThreeClauses(t *testing.T) {
	cmd := Parse("COPY TO backup FOR id > 0 WHILE active")
	if cmd.Verb != "COPY" {
		t.Fatalf("expected COPY, got %q", cmd.Verb)
	}
	if cmd.ToClause != "backup" {
		t.Fatalf("expected 'backup', got %q", cmd.ToClause)
	}
	if cmd.ForClause != "id > 0" {
		t.Fatalf("expected 'id > 0', got %q", cmd.ForClause)
	}
	if cmd.WhileClause != "active" {
		t.Fatalf("expected 'active', got %q", cmd.WhileClause)
	}
}

func TestParseForBeforeTo(t *testing.T) {
	cmd := Parse("LIST FOR id > 0 TO file.txt")
	if cmd.Verb != "LIST" {
		t.Fatalf("expected LIST, got %q", cmd.Verb)
	}
	if cmd.ForClause != "id > 0" {
		t.Fatalf("expected 'id > 0', got %q", cmd.ForClause)
	}
	if cmd.ToClause != "file.txt" {
		t.Fatalf("expected 'file.txt', got %q", cmd.ToClause)
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

func TestParseWithInArgs(t *testing.T) {
	cmd := Parse("STORE 42 TO m_var")
	if cmd.Verb != "STORE" {
		t.Fatalf("expected STORE, got %q", cmd.Verb)
	}
	if cmd.ToClause != "m_var" {
		t.Fatalf("expected 'm_var', got %q", cmd.ToClause)
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

func TestParseFromClause(t *testing.T) {
	cmd := Parse("APPEND FROM backup.dbf")
	if cmd.Verb != "APPEND" {
		t.Fatalf("expected APPEND, got %q", cmd.Verb)
	}
	if cmd.FromClause != "backup.dbf" {
		t.Fatalf("expected 'backup.dbf', got %q", cmd.FromClause)
	}
}

func TestParseFromWithSDFAndFor(t *testing.T) {
	cmd := Parse("APPEND FROM data.txt SDF FOR .T.")
	if cmd.FromClause != "data.txt" {
		t.Fatalf("expected 'data.txt', got %q", cmd.FromClause)
	}
	if cmd.Args != "SDF" {
		t.Fatalf("expected 'SDF', got %q", cmd.Args)
	}
	if cmd.ForClause != ".T." {
		t.Fatalf("expected '.T.', got %q", cmd.ForClause)
	}
}

func TestParseUpdateFromOn(t *testing.T) {
	cmd := Parse("UPDATE FROM ON PARTNO ADD ONHAND REPLACE COST")
	if cmd.FromClause != "" {
		t.Fatalf("expected empty FROM filename for secondary update, got %q", cmd.FromClause)
	}
	if cmd.Args != "ON PARTNO ADD ONHAND REPLACE COST" {
		t.Fatalf("unexpected args: %q", cmd.Args)
	}
}

func TestParseUpdateOnFromFile(t *testing.T) {
	cmd := Parse("UPDATE ON PARTNO FROM invupdat ADD ONHAND REPLACE COST")
	if cmd.FromClause != "invupdat" {
		t.Fatalf("expected invupdat, got %q", cmd.FromClause)
	}
	if cmd.Args != "ON PARTNO ADD ONHAND REPLACE COST" {
		t.Fatalf("unexpected args: %q", cmd.Args)
	}
}

func TestParseJoinToForField(t *testing.T) {
	cmd := Parse("JOIN TO joinout FOR P.KEY = S.KEY FIELD NAME, S.TITLE")
	if cmd.Verb != "JOIN" {
		t.Fatalf("expected JOIN, got %q", cmd.Verb)
	}
	if cmd.ToClause != "joinout" {
		t.Fatalf("expected joinout, got %q", cmd.ToClause)
	}
	if cmd.ForClause != "P.KEY = S.KEY" {
		t.Fatalf("expected join FOR clause, got %q", cmd.ForClause)
	}
	if cmd.Args != "FIELD NAME, S.TITLE" {
		t.Fatalf("expected FIELD args, got %q", cmd.Args)
	}
}

func TestParseLocateFor(t *testing.T) {
	cmd := Parse("LOCATE ALL FOR AGE >= 35 WHILE RECNO() <= 3")
	if cmd.Verb != "LOCATE" {
		t.Fatalf("expected LOCATE, got %q", cmd.Verb)
	}
	if cmd.Args != "ALL" {
		t.Fatalf("expected ALL scope in args, got %q", cmd.Args)
	}
	if cmd.ForClause != "AGE >= 35" {
		t.Fatalf("expected FOR clause, got %q", cmd.ForClause)
	}
	if cmd.WhileClause != "RECNO() <= 3" {
		t.Fatalf("expected WHILE clause, got %q", cmd.WhileClause)
	}
}

func TestParseCountForTo(t *testing.T) {
	cmd := Parse("COUNT ALL FOR AGE >= 35 TO adults")
	if cmd.Verb != "COUNT" {
		t.Fatalf("expected COUNT, got %q", cmd.Verb)
	}
	if cmd.Args != "ALL" {
		t.Fatalf("expected ALL scope in args, got %q", cmd.Args)
	}
	if cmd.ForClause != "AGE >= 35" {
		t.Fatalf("expected FOR clause, got %q", cmd.ForClause)
	}
	if cmd.ToClause != "adults" {
		t.Fatalf("expected TO clause, got %q", cmd.ToClause)
	}
}

func TestParseSumForTo(t *testing.T) {
	cmd := Parse("SUM AGE, AGE * 2 FOR AGE >= 35 TO total, double")
	if cmd.Verb != "SUM" {
		t.Fatalf("expected SUM, got %q", cmd.Verb)
	}
	if cmd.Args != "AGE, AGE * 2" {
		t.Fatalf("expected SUM expressions in args, got %q", cmd.Args)
	}
	if cmd.ForClause != "AGE >= 35" {
		t.Fatalf("expected FOR clause, got %q", cmd.ForClause)
	}
	if cmd.ToClause != "total, double" {
		t.Fatalf("expected TO clause, got %q", cmd.ToClause)
	}
}

func TestParseAverageForTo(t *testing.T) {
	cmd := Parse("AVERAGE AGE FOR AGE >= 35 TO avgage")
	if cmd.Verb != "AVERAGE" {
		t.Fatalf("expected AVERAGE, got %q", cmd.Verb)
	}
	if cmd.Args != "AGE" {
		t.Fatalf("expected AGE in args, got %q", cmd.Args)
	}
	if cmd.ForClause != "AGE >= 35" {
		t.Fatalf("expected FOR clause, got %q", cmd.ForClause)
	}
	if cmd.ToClause != "avgage" {
		t.Fatalf("expected TO clause, got %q", cmd.ToClause)
	}
}
