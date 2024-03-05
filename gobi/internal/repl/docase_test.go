package repl

import (
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/script"
)

func runCaseProgram(t *testing.T, source string) map[string]interface{} {
	t.Helper()
	prog, err := script.ParseSource("case.prg", source)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	ctx := testCtx()
	ctx.Config.Talk = false
	if err := RunProgram(ctx, prog); err != nil {
		t.Fatalf("RunProgram: %v", err)
	}

	vars := make(map[string]interface{})
	for _, name := range []string{"Y", "TRACE", "AFTER"} {
		if val, ok := ctx.Variables.Get(name); ok {
			vars[name] = val
		}
	}
	return vars
}

func TestDoCaseRunsFirstTrueBranch(t *testing.T) {
	vars := runCaseProgram(t, "STORE 2 TO x\n"+
		"DO CASE\n"+
		"CASE x = 1\n"+
		"STORE 'one' TO y\n"+
		"CASE x = 2\n"+
		"STORE 'two' TO y\n"+
		"OTHERWISE\n"+
		"STORE 'other' TO y\n"+
		"ENDCASE\n"+
		"STORE 1 TO after\n")

	if vars["Y"] != "two" {
		t.Fatalf("Y = %#v, want \"two\"", vars["Y"])
	}
	if _, ok := vars["AFTER"]; !ok {
		t.Fatal("expected execution to continue after ENDCASE")
	}
}

func TestDoCaseOnlyFirstMatchingBranchRuns(t *testing.T) {
	prog, err := script.ParseSource("case.prg", "DO CASE\n"+
		"CASE .T.\n"+
		"STORE 1 TO ranA\n"+
		"CASE .T.\n"+
		"STORE 1 TO ranB\n"+
		"OTHERWISE\n"+
		"STORE 1 TO ranC\n"+
		"ENDCASE\n")
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	ctx := testCtx()
	ctx.Config.Talk = false
	if err := RunProgram(ctx, prog); err != nil {
		t.Fatalf("RunProgram: %v", err)
	}

	if _, ok := ctx.Variables.Get("RANA"); !ok {
		t.Fatal("expected first CASE branch to run")
	}
	if _, ok := ctx.Variables.Get("RANB"); ok {
		t.Fatal("second true CASE must not run")
	}
	if _, ok := ctx.Variables.Get("RANC"); ok {
		t.Fatal("OTHERWISE must not run when a CASE matched")
	}
}

func TestDoCaseFallsBackToOtherwise(t *testing.T) {
	vars := runCaseProgram(t, "STORE 9 TO x\n"+
		"DO CASE\n"+
		"CASE x = 1\n"+
		"STORE 'one' TO y\n"+
		"OTHERWISE\n"+
		"STORE 'other' TO y\n"+
		"ENDCASE\n")

	if vars["Y"] != "other" {
		t.Fatalf("Y = %#v, want \"other\"", vars["Y"])
	}
}

func TestDoCaseNoMatchWithoutOtherwiseSkipsAll(t *testing.T) {
	vars := runCaseProgram(t, "STORE 9 TO x\n"+
		"DO CASE\n"+
		"CASE x = 1\n"+
		"STORE 'one' TO y\n"+
		"ENDCASE\n"+
		"STORE 1 TO after\n")

	if _, ok := vars["Y"]; ok {
		t.Fatalf("expected no branch to run, got Y = %#v", vars["Y"])
	}
	if _, ok := vars["AFTER"]; !ok {
		t.Fatal("expected execution to continue after ENDCASE")
	}
}

func TestDoCaseIgnoresStatementsBeforeFirstCase(t *testing.T) {
	vars := runCaseProgram(t, "DO CASE\n"+
		"STORE 'leak' TO trace\n"+
		"CASE .T.\n"+
		"STORE 'ok' TO y\n"+
		"ENDCASE\n")

	if _, ok := vars["TRACE"]; ok {
		t.Fatal("statements between DO CASE and first CASE must not run")
	}
	if vars["Y"] != "ok" {
		t.Fatalf("Y = %#v, want \"ok\"", vars["Y"])
	}
}

func TestDoCaseNestedSelectsInnerBranch(t *testing.T) {
	vars := runCaseProgram(t, "STORE 1 TO x\n"+
		"STORE 2 TO z\n"+
		"DO CASE\n"+
		"CASE x = 1\n"+
		"DO CASE\n"+
		"CASE z = 1\n"+
		"STORE 'inner-one' TO y\n"+
		"CASE z = 2\n"+
		"STORE 'inner-two' TO y\n"+
		"ENDCASE\n"+
		"OTHERWISE\n"+
		"STORE 'outer' TO y\n"+
		"ENDCASE\n")

	if vars["Y"] != "inner-two" {
		t.Fatalf("Y = %#v, want \"inner-two\"", vars["Y"])
	}
}

func TestDoCaseInvalidExpressionFails(t *testing.T) {
	prog, err := script.ParseSource("bad.prg", "DO CASE\nCASE 1 + 1\nSTORE 1 TO y\nENDCASE\n")
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	ctx := testCtx()
	err = RunProgram(ctx, prog)
	if err == nil || !strings.Contains(err.Error(), "CASE expression must evaluate") {
		t.Fatalf("expected logical expression error, got %v", err)
	}
}

func TestDispatchStrayCaseVerbsFail(t *testing.T) {
	ctx := testCtx()
	for verb, want := range map[string]string{
		"CASE":      "CASE without matching DO CASE",
		"OTHERWISE": "OTHERWISE without matching DO CASE",
		"ENDCASE":   "ENDCASE without matching DO CASE",
	} {
		err := commandMux.Dispatch(ctx, Command{Verb: verb})
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("%s: expected %q, got %v", verb, want, err)
		}
	}
}
