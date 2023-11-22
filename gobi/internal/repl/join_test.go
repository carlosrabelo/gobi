package repl

import (
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
)

func TestParseJoinFieldsDefault(t *testing.T) {
	primary := &dbf.Table{
		Fields: []dbf.FieldDescriptor{
			{Name: "NAME", Type: dbf.FieldTypeChar, Length: 10},
			{Name: "KEY", Type: dbf.FieldTypeChar, Length: 3},
		},
	}
	secondary := &dbf.Table{
		Fields: []dbf.FieldDescriptor{
			{Name: "KEY", Type: dbf.FieldTypeChar, Length: 3},
			{Name: "TITLE", Type: dbf.FieldTypeChar, Length: 12},
		},
	}

	specs, err := parseJoinFields(primary, secondary, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specs) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(specs))
	}
	if specs[2].Descriptor.Name != "TITLE" {
		t.Fatalf("expected TITLE as third field, got %s", specs[2].Descriptor.Name)
	}
}

func TestParseJoinFieldsExplicit(t *testing.T) {
	primary := &dbf.Table{
		Fields: []dbf.FieldDescriptor{
			{Name: "NAME", Type: dbf.FieldTypeChar, Length: 10},
			{Name: "KEY", Type: dbf.FieldTypeChar, Length: 3},
		},
	}
	secondary := &dbf.Table{
		Fields: []dbf.FieldDescriptor{
			{Name: "KEY", Type: dbf.FieldTypeChar, Length: 3},
			{Name: "TITLE", Type: dbf.FieldTypeChar, Length: 12},
		},
	}

	specs, err := parseJoinFields(primary, secondary, "FIELD NAME, S.TITLE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specs) != 2 || specs[0].Descriptor.Name != "NAME" || specs[1].Descriptor.Name != "TITLE" {
		t.Fatalf("unexpected specs: %+v", specs)
	}
	if specs[1].Source != joinFieldSecondary {
		t.Fatalf("expected secondary TITLE field")
	}
}

func TestParseJoinFieldsUnknown(t *testing.T) {
	primary := &dbf.Table{
		Fields: []dbf.FieldDescriptor{{Name: "NAME", Type: dbf.FieldTypeChar, Length: 10}},
	}
	secondary := &dbf.Table{
		Fields: []dbf.FieldDescriptor{{Name: "TITLE", Type: dbf.FieldTypeChar, Length: 12}},
	}

	_, err := parseJoinFields(primary, secondary, "FIELD NOPE")
	if err == nil || !strings.Contains(err.Error(), "Unknown field") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
}

func TestJoinEnvironmentPrefixedFields(t *testing.T) {
	primaryTbl := &dbf.Table{
		Fields: []dbf.FieldDescriptor{{Name: "KEY", Type: dbf.FieldTypeChar, Length: 3}},
		Offset: []int{0, 3, 4},
		Header: &dbf.Header{RecordLen: 4},
	}
	secondaryTbl := &dbf.Table{
		Fields: []dbf.FieldDescriptor{{Name: "KEY", Type: dbf.FieldTypeChar, Length: 3}},
		Offset: []int{0, 3, 4},
		Header: &dbf.Header{RecordLen: 4},
	}

	ctx := testCtx()
	primary := ctx.WorkAreas[context.Primary]
	secondary := ctx.WorkAreas[context.Secondary]
	primary.Table = primaryTbl
	secondary.Table = secondaryTbl

	env := newJoinEnvironment(ctx, primary, secondary)
	env.setRecords(&dbf.Record{Data: []byte{0x20, '0', '0', '1'}}, 0)
	env.setSecondaryRecord(&dbf.Record{Data: []byte{0x20, '0', '0', '2'}}, 0)

	pObj, ok := env.GetField("P.KEY")
	if !ok || pObj.String() != "001" {
		t.Fatalf("primary key = %#v ok=%v", pObj, ok)
	}
	sObj, ok := env.GetField("S.KEY")
	if !ok || sObj.String() != "002" {
		t.Fatalf("secondary key = %#v ok=%v", sObj, ok)
	}
}
