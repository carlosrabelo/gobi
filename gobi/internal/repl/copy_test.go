package repl

import (
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestParseCopyFieldsAll(t *testing.T) {
	tbl := &dbf.Table{
		Fields: []dbf.FieldDescriptor{
			{Name: "NAME", Type: dbf.FieldTypeChar, Length: 10},
			{Name: "AGE", Type: dbf.FieldTypeNumeric, Length: 3, DecimalCount: 0},
		},
	}

	fields, idxs, err := parseCopyFields(tbl, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fields) != 2 || len(idxs) != 2 {
		t.Fatalf("expected 2 fields, got %d/%d", len(fields), len(idxs))
	}
}

func TestParseCopyFieldsSubset(t *testing.T) {
	tbl := &dbf.Table{
		Fields: []dbf.FieldDescriptor{
			{Name: "NAME", Type: dbf.FieldTypeChar, Length: 10},
			{Name: "AGE", Type: dbf.FieldTypeNumeric, Length: 3, DecimalCount: 0},
		},
	}

	fields, idxs, err := parseCopyFields(tbl, "FIELD NAME, AGE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fields) != 2 {
		t.Fatalf("field count = %d, want 2", len(fields))
	}
	if fields[0].Name != "NAME" || idxs[1] != 1 {
		t.Fatalf("unexpected fields: %+v %+v", fields, idxs)
	}
}

func TestParseCopyFieldsUnknown(t *testing.T) {
	tbl := &dbf.Table{
		Fields: []dbf.FieldDescriptor{
			{Name: "NAME", Type: dbf.FieldTypeChar, Length: 10},
		},
	}

	_, _, err := parseCopyFields(tbl, "FIELD NOPE")
	if err == nil || !strings.Contains(err.Error(), "Unknown field") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
}

func TestParseCopyFieldsEmptyFieldList(t *testing.T) {
	tbl := &dbf.Table{
		Fields: []dbf.FieldDescriptor{
			{Name: "NAME", Type: dbf.FieldTypeChar, Length: 10},
		},
	}

	_, _, err := parseCopyFields(tbl, "FIELD")
	if err == nil || !strings.Contains(err.Error(), "field list") {
		t.Fatalf("expected field list error, got %v", err)
	}
}
