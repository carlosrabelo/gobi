package context

import (
	"strings"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

func TestNewContext(t *testing.T) {
	ctx := New()
	if ctx.ActiveArea != Primary {
		t.Errorf("expected active area to be %s, got %s", Primary, ctx.ActiveArea)
	}
	if ctx.Config == nil {
		t.Fatal("expected Config to be initialized")
	}
	if ctx.Variables == nil {
		t.Error("expected Variables registry to be initialized")
	}
	if len(ctx.WorkAreas) != 2 {
		t.Errorf("expected 2 work areas, got %d", len(ctx.WorkAreas))
	}
	if ctx.Screen == nil {
		t.Fatal("expected Screen buffer to be initialized")
	}
	if ctx.Screen.Cols() != term.DefaultCols || ctx.Screen.Rows() != term.DefaultRows {
		t.Fatalf("unexpected screen size: %dx%d", ctx.Screen.Cols(), ctx.Screen.Rows())
	}
}

func TestSelectArea(t *testing.T) {
	ctx := New()
	err := ctx.SelectArea(Secondary)
	if err != nil {
		t.Fatalf("unexpected error selecting area: %v", err)
	}
	if ctx.ActiveArea != Secondary {
		t.Errorf("expected active area to be %s, got %s", Secondary, ctx.ActiveArea)
	}

	err = ctx.SelectArea(AreaName("INVALID"))
	if err == nil {
		t.Error("expected error when selecting invalid area, got nil")
	}
}

func TestStdinReaderIsSharedAcrossCalls(t *testing.T) {
	ctx := New()
	ctx.Stdin = strings.NewReader("first\nsecond\n")

	r1 := ctx.StdinReader()
	line, err := r1.ReadString('\n')
	if err != nil || line != "first\n" {
		t.Fatalf("expected first line, got %q (%v)", line, err)
	}

	r2 := ctx.StdinReader()
	if r1 != r2 {
		t.Fatal("expected the same shared reader instance")
	}
	line, err = r2.ReadString('\n')
	if err != nil || line != "second\n" {
		t.Fatalf("expected second line from shared buffer, got %q (%v)", line, err)
	}
}

func TestStdinReaderRebuiltWhenStdinChanges(t *testing.T) {
	ctx := New()
	ctx.Stdin = strings.NewReader("old\n")
	first := ctx.StdinReader()

	ctx.Stdin = strings.NewReader("new\n")
	second := ctx.StdinReader()

	if first == second {
		t.Fatal("expected a new reader after Stdin reassignment")
	}
	line, err := second.ReadString('\n')
	if err != nil || line != "new\n" {
		t.Fatalf("expected line from new source, got %q (%v)", line, err)
	}
}
