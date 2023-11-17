package repl

import (
	"strings"
	"testing"
)

func TestParseUpdateOptionsSecondary(t *testing.T) {
	opts, err := parseUpdateOptions(Command{
		Verb: "UPDATE",
		Args: "ON PARTNO ADD ONHAND REPLACE COST",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.keyField != "PARTNO" {
		t.Fatalf("keyField = %q", opts.keyField)
	}
	if len(opts.addFields) != 1 || opts.addFields[0] != "ONHAND" {
		t.Fatalf("addFields = %#v", opts.addFields)
	}
	if len(opts.replaceFields) != 1 || opts.replaceFields[0] != "COST" {
		t.Fatalf("replaceFields = %#v", opts.replaceFields)
	}
}

func TestParseUpdateOptionsFromFile(t *testing.T) {
	opts, err := parseUpdateOptions(Command{
		Verb:       "UPDATE",
		FromClause: "invupdat",
		Args:       "ON PARTNO ADD ONHAND REPLACE COST",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.fromFile != "invupdat" {
		t.Fatalf("fromFile = %q", opts.fromFile)
	}
}

func TestParseUpdateOptionsMissingOn(t *testing.T) {
	_, err := parseUpdateOptions(Command{Verb: "UPDATE", Args: "ADD ONHAND"})
	if err == nil || !strings.Contains(err.Error(), "ON") {
		t.Fatalf("expected ON error, got %v", err)
	}
}

func TestParseUpdateOptionsMissingFieldLists(t *testing.T) {
	_, err := parseUpdateOptions(Command{Verb: "UPDATE", Args: "ON PARTNO"})
	if err == nil || !strings.Contains(err.Error(), "ADD or REPLACE") {
		t.Fatalf("expected ADD or REPLACE error, got %v", err)
	}
}

func TestCompareUpdateValues(t *testing.T) {
	if compareUpdateValues("21828", "70296") >= 0 {
		t.Fatal("expected 21828 < 70296")
	}
	if compareUpdateValues(float64(10), float64(10)) != 0 {
		t.Fatal("expected numeric equality")
	}
}
