package script

import (
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/pkg/command"
)

// CaseBlock describes a compiled DO CASE ... ENDCASE region in command index space.
type CaseBlock struct {
	DoIndex        int   // DO CASE command index
	CaseIndexes    []int // CASE command indexes in source order
	OtherwiseIndex int   // -1 when OTHERWISE is absent
	EndIndex       int   // ENDCASE command index
}

// IsDoCase reports whether a command line opens a DO CASE block.
func IsDoCase(cmd command.Command) bool {
	return cmd.Verb == "DO" &&
		cmd.WhileClause == "" &&
		strings.EqualFold(strings.TrimSpace(cmd.Args), "CASE")
}

// buildCaseBlocks indexes DO CASE regions in executable command order.
func (p *Program) buildCaseBlocks() error {
	if p == nil {
		return nil
	}

	p.caseBlocks = make(map[int]CaseBlock)
	p.caseSkip = make(map[int]int)

	type pendingBlock struct {
		doIndex        int
		caseIndexes    []int
		otherwiseIndex int
		doLine         int
	}

	stack := make([]pendingBlock, 0)
	cmds := p.Commands()

	for i, line := range cmds {
		switch {
		case IsDoCase(line.Command):
			stack = append(stack, pendingBlock{doIndex: i, otherwiseIndex: -1, doLine: line.Number})
		case line.Command.Verb == "CASE":
			if len(stack) == 0 {
				return fmt.Errorf("script: CASE without matching DO CASE at line %d", line.Number)
			}
			top := &stack[len(stack)-1]
			if top.otherwiseIndex >= 0 {
				return fmt.Errorf("script: CASE after OTHERWISE at line %d", line.Number)
			}
			top.caseIndexes = append(top.caseIndexes, i)
		case line.Command.Verb == "OTHERWISE":
			if len(stack) == 0 {
				return fmt.Errorf("script: OTHERWISE without matching DO CASE at line %d", line.Number)
			}
			top := &stack[len(stack)-1]
			if top.otherwiseIndex >= 0 {
				return fmt.Errorf("script: multiple OTHERWISE for DO CASE at line %d", top.doLine)
			}
			top.otherwiseIndex = i
		case line.Command.Verb == "ENDCASE":
			if len(stack) == 0 {
				return fmt.Errorf("script: ENDCASE without matching DO CASE at line %d", line.Number)
			}
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			p.caseBlocks[top.doIndex] = CaseBlock{
				DoIndex:        top.doIndex,
				CaseIndexes:    top.caseIndexes,
				OtherwiseIndex: top.otherwiseIndex,
				EndIndex:       i,
			}
			for _, branch := range top.caseIndexes {
				p.caseSkip[branch] = i + 1
			}
			if top.otherwiseIndex >= 0 {
				p.caseSkip[top.otherwiseIndex] = i + 1
			}
		}
	}

	if len(stack) > 0 {
		return fmt.Errorf("script: unclosed DO CASE at line %d", stack[len(stack)-1].doLine)
	}

	return nil
}

// CaseBlockAt returns the compiled block for a DO CASE command index.
func (p *Program) CaseBlockAt(doIndex int) (CaseBlock, bool) {
	if p == nil {
		return CaseBlock{}, false
	}
	block, ok := p.caseBlocks[doIndex]
	return block, ok
}

// SkipAfterCaseBranch returns the command index following the matching
// ENDCASE for a CASE or OTHERWISE command reached after an executed branch.
func (p *Program) SkipAfterCaseBranch(branchIndex int) (int, bool) {
	if p == nil {
		return 0, false
	}
	target, ok := p.caseSkip[branchIndex]
	return target, ok
}
