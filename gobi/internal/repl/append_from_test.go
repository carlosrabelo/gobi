package repl

import (
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestParseAppendFromOptionsDBF(t *testing.T) {
	format, _, err := parseAppendFromOptions("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if format != appendFormatDBF {
		t.Fatalf("format = %v, want DBF", format)
	}
}

func TestParseAppendFromOptionsSDF(t *testing.T) {
	format, _, err := parseAppendFromOptions("SDF")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if format != appendFormatSDF {
		t.Fatalf("format = %v, want SDF", format)
	}
}

func TestParseAppendFromOptionsDelimited(t *testing.T) {
	format, opts, err := parseAppendFromOptions("DELIMITED")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if format != appendFormatDelimited {
		t.Fatalf("format = %v, want DELIMITED", format)
	}
	if opts.quoteChar != '\'' {
		t.Fatalf("quoteChar = %q, want single quote", opts.quoteChar)
	}
}

func TestParseAppendFromOptionsDelimitedWithComma(t *testing.T) {
	format, opts, err := parseAppendFromOptions("DELIMITED WITH ,")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if format != appendFormatDelimited {
		t.Fatalf("format = %v, want DELIMITED", format)
	}
	if opts.delimiter != ',' || opts.quoteChar != 0 {
		t.Fatalf("unexpected opts: %+v", opts)
	}
}

func TestParseSDFLine(t *testing.T) {
	tbl := &dbf.Table{
		Fields: []dbf.FieldDescriptor{
			{Name: "NAME", Type: dbf.FieldTypeChar, Length: 10},
			{Name: "AGE", Type: dbf.FieldTypeNumeric, Length: 3, DecimalCount: 0},
		},
	}

	values, err := parseSDFLine("Alice     25", tbl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if values[0] != "Alice     " {
		t.Fatalf("name = %q", values[0])
	}
	if values[1] != "25" {
		t.Fatalf("age = %q", values[1])
	}
}

func TestParseDelimitedLineDefault(t *testing.T) {
	tbl := &dbf.Table{
		Fields: []dbf.FieldDescriptor{
			{Name: "NAME", Type: dbf.FieldTypeChar, Length: 10},
			{Name: "AGE", Type: dbf.FieldTypeNumeric, Length: 3, DecimalCount: 0},
		},
	}

	values, err := parseDelimitedLine("'Alice', 25", tbl, delimitedImportOptions{delimiter: ',', quoteChar: '\''})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if values[0] != "Alice" {
		t.Fatalf("name = %q", values[0])
	}
	if values[1] != "25" {
		t.Fatalf("age = %q", values[1])
	}
}

func TestParseDelimitedLineCSV(t *testing.T) {
	tbl := &dbf.Table{
		Fields: []dbf.FieldDescriptor{
			{Name: "NAME", Type: dbf.FieldTypeChar, Length: 10},
			{Name: "AGE", Type: dbf.FieldTypeNumeric, Length: 3, DecimalCount: 0},
		},
	}

	values, err := parseDelimitedLine("Alice, 25", tbl, delimitedImportOptions{delimiter: ',', quoteChar: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if values[0] != "Alice" {
		t.Fatalf("name = %q", values[0])
	}
	if values[1] != "25" {
		t.Fatalf("age = %q", values[1])
	}
}

func TestMapSourceRecordToDestMatchingFields(t *testing.T) {
	srcTbl := &dbf.Table{
		Fields: []dbf.FieldDescriptor{
			{Name: "NAME", Type: dbf.FieldTypeChar, Length: 10},
			{Name: "AGE", Type: dbf.FieldTypeNumeric, Length: 3, DecimalCount: 0},
		},
		Offset: []int{0, 10, 13},
		Header: &dbf.Header{RecordLen: 14},
	}
	dstTbl := &dbf.Table{
		Fields: []dbf.FieldDescriptor{
			{Name: "NAME", Type: dbf.FieldTypeChar, Length: 10},
			{Name: "CITY", Type: dbf.FieldTypeChar, Length: 8},
			{Name: "AGE", Type: dbf.FieldTypeNumeric, Length: 3, DecimalCount: 0},
		},
		Offset: []int{0, 10, 18, 21},
		Header: &dbf.Header{RecordLen: 22},
	}

	srcRec := &dbf.Record{
		Data: append([]byte{0x20}, append([]byte("Alice     "), []byte(" 25")...)...),
	}

	values, err := mapSourceRecordToDest(srcTbl, srcRec, dstTbl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if values[0] != "Alice" {
		t.Fatalf("name = %q", values[0])
	}
	if values[1] != "" {
		t.Fatalf("city = %q, want empty", values[1])
	}
	if values[2] != float64(25) {
		t.Fatalf("age = %v", values[2])
	}
}

func TestSplitDelimitedFieldsQuoted(t *testing.T) {
	parts, err := splitDelimitedFields("'Alice', 25", delimitedImportOptions{delimiter: ',', quoteChar: '\''}, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parts) != 2 || parts[0] != "Alice" || strings.TrimSpace(parts[1]) != "25" {
		t.Fatalf("unexpected parts: %#v", parts)
	}
}
