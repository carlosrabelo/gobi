package repl

import (
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestFieldToStructureLine(t *testing.T) {
	line := fieldToStructureLine(dbf.FieldDescriptor{
		Name: "SALARY", Type: dbf.FieldTypeNumeric, Length: 8, DecimalCount: 2,
	})
	if line != "SALARY,N,8,2" {
		t.Fatalf("got %q, want SALARY,N,8,2", line)
	}

	line = fieldToStructureLine(dbf.FieldDescriptor{
		Name: "NAME", Type: dbf.FieldTypeChar, Length: 10,
	})
	if line != "NAME,C,10,0" {
		t.Fatalf("got %q, want NAME,C,10,0", line)
	}
}

func TestParseStructureLines(t *testing.T) {
	fields, err := parseStructureLines([]string{
		"NAME,C,10,0",
		"",
		"AGE,N,3,0",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fields) != 2 {
		t.Fatalf("field count = %d, want 2", len(fields))
	}
}

func TestParseStructureLinesDuplicate(t *testing.T) {
	_, err := parseStructureLines([]string{"NAME,C,10,0", "NAME,C,5,0"})
	if err == nil {
		t.Fatal("expected duplicate field error")
	}
}
