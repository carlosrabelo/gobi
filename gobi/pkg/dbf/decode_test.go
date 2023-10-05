package dbf

import (
	"bytes"
	"testing"
)

func TestDecodeFieldChar(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 10},
	}
	records := [][]byte{
		{0x20, 'H', 'E', 'L', 'L', 'O', ' ', ' ', ' ', ' ', ' '},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	val, err := rec.DecodeField(tbl, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s, ok := val.(string)
	if !ok {
		t.Fatalf("expected string, got %T", val)
	}
	if s != "HELLO" {
		t.Errorf("value = %q, want %q", s, "HELLO")
	}
}

func TestDecodeFieldCharAllSpaces(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, ' ', ' ', ' ', ' ', ' '},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	val, err := rec.DecodeField(tbl, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := val.(string)
	if s != "" {
		t.Errorf("value = %q, want empty string", s)
	}
}

func TestDecodeFieldCharNoPadding(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'A', 'B', 'C', 'D', 'E'},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	val, err := rec.DecodeField(tbl, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := val.(string)
	if s != "ABCDE" {
		t.Errorf("value = %q, want %q", s, "ABCDE")
	}
}

func TestDecodeFieldCharPreservesLeadingSpaces(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 8},
	}
	records := [][]byte{
		{0x20, ' ', ' ', 'H', 'I', ' ', ' ', ' ', ' '},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	val, err := rec.DecodeField(tbl, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := val.(string)
	if s != "  HI" {
		t.Errorf("value = %q, want %q", s, "  HI")
	}
}

func TestDecodeFieldUnsupportedType(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'H', 'E', 'L', 'L', 'O'},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	tbl.Fields[0].Type = 'X'
	_, err := rec.DecodeField(tbl, 0)
	if err == nil {
		t.Fatal("expected error for unsupported field type")
	}
}

func TestDecodeFieldIndexOutOfRange(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
	}
	records := [][]byte{
		{0x20, 'H', 'E', 'L', 'L', 'O'},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	_, err := rec.DecodeField(tbl, 5)
	if err == nil {
		t.Fatal("expected error for out-of-range index")
	}
}

func TestDecodeFieldMultipleCharFields(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "FIRST", Type: FieldTypeChar, Length: 5},
		{Name: "LAST", Type: FieldTypeChar, Length: 8},
	}
	records := [][]byte{
		{0x20, 'J', 'O', 'H', 'N', ' ', 'D', 'O', 'E', ' ', ' ', ' ', ' ', ' '},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	first, _ := rec.DecodeField(tbl, 0)
	if first.(string) != "JOHN" {
		t.Errorf("first = %q, want %q", first, "JOHN")
	}

	last, _ := rec.DecodeField(tbl, 1)
	if last.(string) != "DOE" {
		t.Errorf("last = %q, want %q", last, "DOE")
	}
}

func openTableWithRecord(t *testing.T, fields []FieldDescriptor, records [][]byte) (*Table, *Record) {
	t.Helper()
	data := buildDBFWithRecords(fields, records)
	tbl, err := Open(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("open table: %v", err)
	}
	rec, err := tbl.ReadRecord(bytes.NewReader(records[0]))
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	return tbl, rec
}

func TestDecodeFieldNumericInteger(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "AGE", Type: FieldTypeNumeric, Length: 5},
	}
	records := [][]byte{
		{0x20, ' ', ' ', '2', '5', '3'},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	val, err := rec.DecodeField(tbl, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	n, ok := val.(float64)
	if !ok {
		t.Fatalf("expected float64, got %T", val)
	}
	if n != 253 {
		t.Errorf("value = %v, want 253", n)
	}
}

func TestDecodeFieldNumericDecimal(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "PRICE", Type: FieldTypeNumeric, Length: 8, DecimalCount: 2},
	}
	records := [][]byte{
		{0x20, ' ', '1', '2', '3', '4', '.', '5', '6'},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	val, err := rec.DecodeField(tbl, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	n := val.(float64)
	if n != 1234.56 {
		t.Errorf("value = %v, want 1234.56", n)
	}
}

func TestDecodeFieldNumericLeftPadded(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "AGE", Type: FieldTypeNumeric, Length: 10},
	}
	records := [][]byte{
		{0x20, ' ', ' ', ' ', ' ', ' ', ' ', ' ', '4', '2', '0'},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	val, err := rec.DecodeField(tbl, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	n := val.(float64)
	if n != 420 {
		t.Errorf("value = %v, want 420", n)
	}
}

func TestDecodeFieldNumericAllSpaces(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "VAL", Type: FieldTypeNumeric, Length: 5},
	}
	records := [][]byte{
		{0x20, ' ', ' ', ' ', ' ', ' '},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	val, err := rec.DecodeField(tbl, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	n := val.(float64)
	if n != 0 {
		t.Errorf("value = %v, want 0", n)
	}
}

func TestDecodeFieldNumericZero(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "VAL", Type: FieldTypeNumeric, Length: 3},
	}
	records := [][]byte{
		{0x20, ' ', '0', '0'},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	val, err := rec.DecodeField(tbl, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.(float64) != 0 {
		t.Errorf("value = %v, want 0", val)
	}
}

func TestDecodeFieldNumericNegative(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "VAL", Type: FieldTypeNumeric, Length: 6},
	}
	records := [][]byte{
		{0x20, ' ', '-', '1', '2', '.', '5'},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	val, err := rec.DecodeField(tbl, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.(float64) != -12.5 {
		t.Errorf("value = %v, want -12.5", val)
	}
}

func TestDecodeFieldNumericInvalid(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "VAL", Type: FieldTypeNumeric, Length: 5},
	}
	records := [][]byte{
		{0x20, ' ', 'A', 'B', 'C', 'D'},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	_, err := rec.DecodeField(tbl, 0)
	if err == nil {
		t.Fatal("expected error for invalid numeric")
	}
}

func TestDecodeFieldMixedTypes(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
		{Name: "AGE", Type: FieldTypeNumeric, Length: 3},
	}
	records := [][]byte{
		{0x20, 'J', 'O', 'H', 'N', ' ', ' ', '3', '5'},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	name, _ := rec.DecodeField(tbl, 0)
	if name.(string) != "JOHN" {
		t.Errorf("name = %q, want %q", name, "JOHN")
	}

	age, _ := rec.DecodeField(tbl, 1)
	if age.(float64) != 35 {
		t.Errorf("age = %v, want 35", age)
	}
}

func TestDecodeFieldLogicalTrue(t *testing.T) {
	for _, c := range []byte{'T', 't', 'Y', 'y'} {
		t.Run(string(c), func(t *testing.T) {
			fields := []FieldDescriptor{
				{Name: "FLAG", Type: FieldTypeLogical, Length: 1},
			}
			records := [][]byte{
				{0x20, c},
			}
			tbl, rec := openTableWithRecord(t, fields, records)

			val, err := rec.DecodeField(tbl, 0)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if val.(bool) != true {
				t.Errorf("value for %c = %v, want true", c, val)
			}
		})
	}
}

func TestDecodeFieldLogicalFalse(t *testing.T) {
	for _, c := range []byte{'F', 'f', 'N', 'n'} {
		t.Run(string(c), func(t *testing.T) {
			fields := []FieldDescriptor{
				{Name: "FLAG", Type: FieldTypeLogical, Length: 1},
			}
			records := [][]byte{
				{0x20, c},
			}
			tbl, rec := openTableWithRecord(t, fields, records)

			val, err := rec.DecodeField(tbl, 0)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if val.(bool) != false {
				t.Errorf("value for %c = %v, want false", c, val)
			}
		})
	}
}

func TestDecodeFieldLogicalUninitialized(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "FLAG", Type: FieldTypeLogical, Length: 1},
	}
	records := [][]byte{
		{0x20, '?'},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	val, err := rec.DecodeField(tbl, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.(bool) != false {
		t.Errorf("value = %v, want false for '?'", val)
	}
}

func TestDecodeFieldAllThreeTypes(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 5},
		{Name: "AGE", Type: FieldTypeNumeric, Length: 3},
		{Name: "ACTIVE", Type: FieldTypeLogical, Length: 1},
	}
	records := [][]byte{
		{0x20, 'J', 'O', 'H', 'N', ' ', ' ', '2', '5', 'T'},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	name, _ := rec.DecodeField(tbl, 0)
	if name.(string) != "JOHN" {
		t.Errorf("name = %q, want %q", name, "JOHN")
	}

	age, _ := rec.DecodeField(tbl, 1)
	if age.(float64) != 25 {
		t.Errorf("age = %v, want 25", age)
	}

	active, _ := rec.DecodeField(tbl, 2)
	if active.(bool) != true {
		t.Errorf("active = %v, want true", active)
	}
}

func TestDecodeFieldLogicalEmpty(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "FLAG", Type: FieldTypeLogical, Length: 0},
	}
	records := [][]byte{
		{0x20},
	}
	tbl, rec := openTableWithRecord(t, fields, records)

	val, err := rec.DecodeField(tbl, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.(bool) != false {
		t.Errorf("value = %v, want false for empty slice", val)
	}
}
