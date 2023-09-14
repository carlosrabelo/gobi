package expr

import "testing"

func TestBuiltinDBFNDX(t *testing.T) {
	t.Run("no table", func(t *testing.T) {
		got := mustEval(t, "DBF()", &testEnv{})
		if got.Str != "" {
			t.Fatalf("got %q", got.Str)
		}
		got = mustEval(t, "NDX(1)", &testEnv{})
		if got.Str != "" {
			t.Fatalf("got %q", got.Str)
		}
	})
	t.Run("paths", func(t *testing.T) {
		var ndx [7]string
		ndx[0] = "cust.ndx"
		env := &testEnv{area: Area{HasTable: true, DBF: "cust.dbf", NDX: ndx}}
		got := mustEval(t, "DBF", env)
		if got.Str != "cust.dbf" {
			t.Fatalf("got %q", got.Str)
		}
		got = mustEval(t, "NDX(1)", env)
		if got.Str != "cust.ndx" {
			t.Fatalf("got %q", got.Str)
		}
		got = mustEval(t, "NDX(2)", env)
		if got.Str != "" {
			t.Fatalf("got %q", got.Str)
		}
	})
}

func TestBuiltinFIELD(t *testing.T) {
	env := &testEnv{area: Area{HasTable: true, Fields: []string{"Name", "AGE"}}}
	got := mustEval(t, `FIELD(1)`, env)
	if got.Str != "NAME" {
		t.Fatalf("got %q, want NAME", got.Str)
	}
	got = mustEval(t, `FIELD(3)`, env)
	if got.Str != "" {
		t.Fatalf("got %q, want empty", got.Str)
	}
}

func TestBuiltinRECCOUNTRECSIZE(t *testing.T) {
	t.Run("no table", func(t *testing.T) {
		got := mustEval(t, "RECCOUNT()", &testEnv{})
		if got.Num != 0 {
			t.Fatalf("RECCOUNT got %v", got)
		}
		got = mustEval(t, "RECSIZE()", &testEnv{})
		if got.Num != 0 {
			t.Fatalf("RECSIZE got %v", got)
		}
	})
	t.Run("open", func(t *testing.T) {
		env := &testEnv{area: Area{HasTable: true, RecCount: 12, RecSize: 80}}
		got := mustEval(t, "RECCOUNT", env)
		if got.Num != 12 {
			t.Fatalf("got %v", got)
		}
		got = mustEval(t, "RECSIZE()", env)
		if got.Num != 80 {
			t.Fatalf("got %v", got)
		}
	})
}

func TestBuiltinCHR(t *testing.T) {
	got := mustEval(t, `CHR(65)`, &testEnv{})
	if got.Str != "A" {
		t.Fatalf("got %q, want A", got.Str)
	}
	tree, errs := Parse("CHR(256)")
	if len(errs) != 0 {
		t.Fatalf("parse: %v", errs)
	}
	_, err := Eval(tree, &testEnv{})
	if err == nil {
		t.Fatal("expected range error")
	}
}

func TestBuiltinSTR(t *testing.T) {
	got := mustEval(t, `STR(1.5, 10, 2)`, &testEnv{})
	if got.Str != "      1.50" {
		t.Fatalf("got %q", got.Str)
	}
	got = mustEval(t, `STR(1)`, &testEnv{})
	if got.Str != "         1" {
		t.Fatalf("got %q", got.Str)
	}
	got = mustEval(t, `STR(12345, 3, 0)`, &testEnv{})
	if got.Str != "***" {
		t.Fatalf("got %q, want ***", got.Str)
	}
}

func TestBuiltinVAL(t *testing.T) {
	got := mustEval(t, `VAL("")`, &testEnv{})
	if got.Num != 0 {
		t.Fatalf("got %v, want 0", got)
	}
	got = mustEval(t, `VAL("3.5")`, &testEnv{})
	if got.Num != 3.5 {
		t.Fatalf("got %v, want 3.5", got)
	}
	got = mustEval(t, `VAL("abc")`, &testEnv{})
	if got.Num != 0 {
		t.Fatalf("got %v, want 0", got)
	}
}

func TestBuiltinSUBSTR(t *testing.T) {
	got := mustEval(t, `SUBSTR("ABCD", 2, 2)`, &testEnv{})
	if got.Str != "BC" {
		t.Fatalf("got %q, want BC", got.Str)
	}
	got = mustEval(t, `SUBSTR("ABCD", 5, 1)`, &testEnv{})
	if got.Str != "" {
		t.Fatalf("got %q, want empty", got.Str)
	}
}

func TestBuiltinLEN(t *testing.T) {
	got := mustEval(t, `LEN("abcd")`, &testEnv{})
	if got.Num != 4 {
		t.Fatalf("got %v, want 4", got)
	}
}

func TestBuiltinLOWER(t *testing.T) {
	got := mustEval(t, `LOWER("AbC")`, &testEnv{})
	if got.Str != "abc" {
		t.Fatalf("got %q, want abc", got.Str)
	}
}

func TestBuiltinUPPER(t *testing.T) {
	got := mustEval(t, `UPPER("AbC")`, &testEnv{})
	if got.Str != "ABC" {
		t.Fatalf("got %q, want ABC", got.Str)
	}
}

func TestBuiltinTRIM(t *testing.T) {
	got := mustEval(t, `TRIM("abc  ")`, &testEnv{})
	if got.Str != "abc" {
		t.Fatalf("got %q, want abc", got.Str)
	}
}

func TestBuiltinFOUND(t *testing.T) {
	t.Run("no table false", func(t *testing.T) {
		got := mustEval(t, "FOUND()", &testEnv{})
		if got.Log {
			t.Fatalf("got %v, want .F.", got)
		}
	})
	t.Run("flag set", func(t *testing.T) {
		got := mustEval(t, "FOUND", &testEnv{area: Area{Found: true}})
		if !got.Log {
			t.Fatalf("got %v, want .T.", got)
		}
	})
}

func TestBuiltinDELETED(t *testing.T) {
	t.Run("no table", func(t *testing.T) {
		got := mustEval(t, "DELETED()", &testEnv{})
		if got.Log {
			t.Fatalf("got %v, want .F.", got)
		}
	})
	t.Run("marked", func(t *testing.T) {
		got := mustEval(t, "DELETED", &testEnv{area: Area{HasTable: true, RecNo: 1, RecCount: 2, Deleted: true}})
		if !got.Log {
			t.Fatalf("got %v, want .T.", got)
		}
	})
	t.Run("past last", func(t *testing.T) {
		got := mustEval(t, "DELETED()", &testEnv{area: Area{HasTable: true, RecNo: 3, RecCount: 2, Deleted: true}})
		if got.Log {
			t.Fatalf("got %v, want .F.", got)
		}
	})
}

func TestBuiltinRECNO(t *testing.T) {
	t.Run("no table is zero", func(t *testing.T) {
		got := mustEval(t, "RECNO()", &testEnv{})
		if got.Num != 0 {
			t.Fatalf("got %v, want 0", got)
		}
	})
	t.Run("on record", func(t *testing.T) {
		got := mustEval(t, "RECNO", &testEnv{area: Area{HasTable: true, RecNo: 2, RecCount: 5}})
		if got.Num != 2 {
			t.Fatalf("got %v, want 2", got)
		}
	})
	t.Run("past last", func(t *testing.T) {
		got := mustEval(t, "RECNO()", &testEnv{area: Area{HasTable: true, RecNo: 6, RecCount: 5}})
		if got.Num != 6 {
			t.Fatalf("got %v, want 6", got)
		}
	})
}

func TestBuiltinBOF(t *testing.T) {
	t.Run("no table is false", func(t *testing.T) {
		got := mustEval(t, "BOF()", &testEnv{})
		if got.Log {
			t.Fatalf("got %v, want .F.", got)
		}
	})
	t.Run("flag set", func(t *testing.T) {
		got := mustEval(t, "BOF", &testEnv{area: Area{HasTable: true, BOF: true, RecNo: 1, RecCount: 3}})
		if !got.Log {
			t.Fatalf("got %v, want .T.", got)
		}
	})
	t.Run("flag clear", func(t *testing.T) {
		got := mustEval(t, "BOF()", &testEnv{area: Area{HasTable: true, BOF: false, RecNo: 1, RecCount: 3}})
		if got.Log {
			t.Fatalf("got %v, want .F.", got)
		}
	})
}

func TestBuiltinEOF(t *testing.T) {
	t.Run("no table", func(t *testing.T) {
		got := mustEval(t, "EOF()", &testEnv{})
		if !got.Log {
			t.Fatalf("got %v, want .T.", got)
		}
	})
	t.Run("bare EOF same as call", func(t *testing.T) {
		got := mustEval(t, "EOF", &testEnv{})
		if !got.Log {
			t.Fatalf("got %v, want .T.", got)
		}
	})
	t.Run("on record", func(t *testing.T) {
		got := mustEval(t, "EOF()", &testEnv{area: Area{HasTable: true, RecNo: 1, RecCount: 3}})
		if got.Log {
			t.Fatalf("got %v, want .F.", got)
		}
	})
	t.Run("past last", func(t *testing.T) {
		got := mustEval(t, "EOF()", &testEnv{area: Area{HasTable: true, RecNo: 4, RecCount: 3}})
		if !got.Log {
			t.Fatalf("got %v, want .T.", got)
		}
	})
	t.Run("wrong arity", func(t *testing.T) {
		tree, errs := Parse("EOF(1)")
		if len(errs) != 0 {
			t.Fatalf("parse: %v", errs)
		}
		_, err := Eval(tree, &testEnv{})
		if err == nil {
			t.Fatal("expected arity error")
		}
	})
}
