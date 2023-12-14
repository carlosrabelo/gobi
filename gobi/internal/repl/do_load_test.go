package repl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/script"
)

func TestDispatchDoLoadsParsedProgram(t *testing.T) {
	tempDir := t.TempDir()
	content := "* setup\r\nUSE people\r\nLIST\r\nRETURN\r\n"
	if err := os.WriteFile(filepath.Join(tempDir, "deptlist.prg"), []byte(content), 0644); err != nil {
		t.Fatalf("write prg: %v", err)
	}

	prog, err := script.Load(tempDir, "deptlist")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	cmds := prog.Commands()
	if len(cmds) != 3 {
		t.Fatalf("expected 3 executable lines, got %d", len(cmds))
	}
	if cmds[0].Command.Verb != "USE" || cmds[1].Command.Verb != "LIST" || cmds[2].Command.Verb != "RETURN" {
		t.Fatalf("unexpected commands: %#v", cmds)
	}
}
