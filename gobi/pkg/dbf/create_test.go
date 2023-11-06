package dbf

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestCreateEmptyTable(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "NAME", Type: FieldTypeChar, Length: 10},
		{Name: "AGE", Type: FieldTypeNumeric, Length: 3, DecimalCount: 0},
	}

	pf := &packFile{}
	tbl, err := Create(pf, fields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tbl.Header.RecordCount != 0 {
		t.Fatalf("record count = %d, want 0", tbl.Header.RecordCount)
	}
	if tbl.Header.RecordLen != 14 {
		t.Fatalf("record length = %d, want 14", tbl.Header.RecordLen)
	}

	wantSize := tbl.HeaderSize() + 1
	if len(pf.data) != wantSize {
		t.Fatalf("file size = %d, want %d", len(pf.data), wantSize)
	}

	reopened, err := Open(bytes.NewReader(pf.data))
	if err != nil {
		t.Fatalf("reopen created file: %v", err)
	}
	if len(reopened.Fields) != 2 {
		t.Fatalf("field count = %d, want 2", len(reopened.Fields))
	}
	if reopened.Fields[0].Name != "NAME" {
		t.Errorf("fields[0].Name = %q, want NAME", reopened.Fields[0].Name)
	}
}

func TestCreateNoFields(t *testing.T) {
	pf := &packFile{data: make([]byte, 64)}
	_, err := Create(pf, nil)
	if err == nil || !strings.Contains(err.Error(), "at least one field") {
		t.Fatalf("expected at least one field error, got %v", err)
	}
}

func TestCreateTooManyFields(t *testing.T) {
	fields := make([]FieldDescriptor, maxFieldCount+1)
	for i := range fields {
		fields[i] = FieldDescriptor{Name: fmt.Sprintf("FLD%02d", i), Type: FieldTypeChar, Length: 1}
	}
	pf := &packFile{data: make([]byte, 1024)}
	_, err := Create(pf, fields)
	if err == nil || !strings.Contains(err.Error(), "too many fields") {
		t.Fatalf("expected too many fields error, got %v", err)
	}
}

func TestValidateFieldName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"NAME", true},
		{"DEPT:NUM", true},
		{"A1", true},
		{"", false},
		{"1NAME", false},
		{"BAD NAME", false},
		{"VERYLONGNAME", false},
	}
	for _, tt := range tests {
		err := ValidateFieldName(tt.name)
		if tt.valid && err != nil {
			t.Errorf("ValidateFieldName(%q) = %v, want nil", tt.name, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("ValidateFieldName(%q) = nil, want error", tt.name)
		}
	}
}

func TestCreateLogicalField(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "ACTIVE", Type: FieldTypeLogical, Length: 1},
	}
	pf := &packFile{data: make([]byte, 128)}
	tbl, err := Create(pf, fields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tbl.Header.RecordLen != 2 {
		t.Fatalf("record length = %d, want 2", tbl.Header.RecordLen)
	}
}

func TestCreateInvalidLogicalWidth(t *testing.T) {
	fields := []FieldDescriptor{
		{Name: "ACTIVE", Type: FieldTypeLogical, Length: 2},
	}
	pf := &packFile{data: make([]byte, 128)}
	_, err := Create(pf, fields)
	if err == nil || !strings.Contains(err.Error(), "logical") {
		t.Fatalf("expected logical width error, got %v", err)
	}
}
