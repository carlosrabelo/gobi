package expr

import "testing"

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
