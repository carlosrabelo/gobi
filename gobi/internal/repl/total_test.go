package repl

import (
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestParseTotalOptions(t *testing.T) {
	opts, err := parseTotalOptions("ON DEPTNUM FIELD SALARY, BONUS")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.keyField != "DEPTNUM" {
		t.Fatalf("keyField = %q", opts.keyField)
	}
	if len(opts.sumFields) != 2 {
		t.Fatalf("sumFields = %#v", opts.sumFields)
	}
}

func TestParseTotalOptionsMissingOn(t *testing.T) {
	_, err := parseTotalOptions("FIELD SALARY")
	if err == nil || !strings.Contains(err.Error(), "ON") {
		t.Fatalf("expected ON error, got %v", err)
	}
}

func TestBuildTotalOutputFieldsSubset(t *testing.T) {
	srcTbl := &dbf.Table{
		Fields: []dbf.FieldDescriptor{
			{Name: "DEPTNUM", Type: dbf.FieldTypeChar, Length: 3},
			{Name: "SALARY", Type: dbf.FieldTypeNumeric, Length: 8, DecimalCount: 2},
			{Name: "NAME", Type: dbf.FieldTypeChar, Length: 10},
		},
	}

	fields, err := buildTotalOutputFields(srcTbl, "DEPTNUM", []string{"SALARY"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fields) != 2 || fields[0].Name != "DEPTNUM" || fields[1].Name != "SALARY" {
		t.Fatalf("unexpected fields: %+v", fields)
	}
}

func TestBuildTotalOutputFieldsNonNumeric(t *testing.T) {
	srcTbl := &dbf.Table{
		Fields: []dbf.FieldDescriptor{
			{Name: "DEPTNUM", Type: dbf.FieldTypeChar, Length: 3},
			{Name: "NAME", Type: dbf.FieldTypeChar, Length: 10},
		},
	}

	_, err := buildTotalOutputFields(srcTbl, "DEPTNUM", []string{"NAME"})
	if err == nil || !strings.Contains(err.Error(), "numeric") {
		t.Fatalf("expected numeric field error, got %v", err)
	}
}

func TestTotalGroupValues(t *testing.T) {
	mappings := []totalFieldMapping{
		{Descriptor: dbf.FieldDescriptor{Name: "DEPTNUM", Type: dbf.FieldTypeChar, Length: 3}, SrcIdx: 0, Role: totalRoleKey},
		{Descriptor: dbf.FieldDescriptor{Name: "SALARY", Type: dbf.FieldTypeNumeric, Length: 8, DecimalCount: 2}, SrcIdx: 1, Role: totalRoleSum},
	}
	group := &totalGroupState{
		active:   true,
		keyValue: "16",
		sums:     map[int]float64{1: 38625.0},
		first:    map[int]interface{}{},
	}

	values, err := totalGroupValues(mappings, group)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if values[0] != "16" {
		t.Fatalf("key = %v", values[0])
	}
	if values[1].(float64) != 38625.0 {
		t.Fatalf("salary = %v", values[1])
	}
}
