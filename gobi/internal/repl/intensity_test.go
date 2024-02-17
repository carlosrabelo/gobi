package repl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/dbf"
	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

func TestApplySetIntensityOff(t *testing.T) {
	ctx := testCtx()
	ctx.Stdout = &bytes.Buffer{}

	applySetIntensity(ctx, []string{"INTENSITY", "OFF"})
	if ctx.Config.Intensity {
		t.Fatal("expected Intensity to be false")
	}
	if !strings.Contains(ctx.Stdout.(*bytes.Buffer).String(), "Intensity: OFF") {
		t.Fatal("expected SET confirmation")
	}
}

func TestApplySetIntensityOn(t *testing.T) {
	ctx := testCtx()
	ctx.Config.Intensity = false
	ctx.Stdout = &bytes.Buffer{}

	applySetIntensity(ctx, []string{"INTENSITY", "ON"})
	if !ctx.Config.Intensity {
		t.Fatal("expected Intensity to be true")
	}
}

func TestPresentScreenUsesReverseVideoWhenIntensityOn(t *testing.T) {
	ctx := testCtx()
	ctx.Config.Intensity = true
	var out bytes.Buffer

	ctx.Screen.Clear()
	ctx.Screen.WriteAt(2, 5, "HELLO")

	err := presentScreen(ctx, &out, []term.Highlight{{Row: 2, Col: 5, Length: 5}})
	if err != nil {
		t.Fatalf("presentScreen: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "\033[7m") {
		t.Fatalf("expected reverse-video sequence, got %q", got)
	}
}

func TestPresentScreenSkipsReverseVideoWhenIntensityOff(t *testing.T) {
	ctx := testCtx()
	ctx.Config.Intensity = false
	var out bytes.Buffer

	ctx.Screen.Clear()
	ctx.Screen.WriteAt(2, 5, "HELLO")

	err := presentScreen(ctx, &out, []term.Highlight{{Row: 2, Col: 5, Length: 5}})
	if err != nil {
		t.Fatalf("presentScreen: %v", err)
	}
	got := out.String()
	if strings.Contains(got, "\033[7m") {
		t.Fatalf("expected no reverse-video with INTENSITY OFF, got %q", got)
	}
}

func TestReadSessionHighlightsActiveFieldWithIntensity(t *testing.T) {
	ctx := testCtx()
	ctx.Config.Intensity = true
	ctx.Stdout = &bytes.Buffer{}
	ctx.Screen.Clear()
	ctx.Screen.RegisterGet(4, 10, "NAME", "XXXXX")
	ctx.Screen.WriteAt(4, 10, "     ")

	s := &readSession{
		ctx: ctx,
		fields: []readFieldState{
			{def: term.GetField{Row: 4, Col: 10, Name: "NAME", Picture: "XXXXX"}, value: "Ann", spec: buildPictureSpec("XXXXX"), maxWidth: 5},
		},
		out: ctx.Stdout,
	}

	if err := s.draw(); err != nil {
		t.Fatalf("draw: %v", err)
	}
	if !strings.Contains(ctx.Stdout.(*bytes.Buffer).String(), "\033[7m") {
		t.Fatal("expected reverse-video on active GET field")
	}
}

func TestEditSessionHighlightsActiveFieldWithIntensity(t *testing.T) {
	ctx := testCtx()
	ctx.Config.Intensity = true
	ctx.Stdout = &bytes.Buffer{}
	area := ctx.GetActiveArea()
	area.Alias = "PEOPLE"

	s := &editSession{
		ctx:      ctx,
		area:     area,
		out:      ctx.Stdout,
		frame:    term.NewFrameWriter(ctx.Stdout),
		fieldIdx: 0,
		values:   []string{"Alice"},
		tbl:      dbfTableWithNameField(t),
	}

	s.draw()
	if !strings.Contains(ctx.Stdout.(*bytes.Buffer).String(), "\033[7m") {
		t.Fatal("expected reverse-video on active EDIT field")
	}
}

func dbfTableWithNameField(t *testing.T) *dbf.Table {
	t.Helper()
	return &dbf.Table{
		Fields: []dbf.FieldDescriptor{
			{Name: "NAME", Type: dbf.FieldTypeChar, Length: 10},
		},
	}
}
