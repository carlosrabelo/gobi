package script

import "fmt"

// WhileBlock describes a compiled DO WHILE ... ENDDO region in command index space.
type WhileBlock struct {
	DoIndex    int
	StartIndex int // first command in the loop body
	EndIndex   int // ENDDO command index
}

// buildWhileBlocks indexes DO WHILE regions in executable command order.
func (p *Program) buildWhileBlocks() error {
	if p == nil {
		return nil
	}

	p.whileBlocks = make(map[int]WhileBlock)
	p.endDoBlocks = make(map[int]WhileBlock)

	type pendingBlock struct {
		doIndex int
		doLine  int
	}

	stack := make([]pendingBlock, 0)
	cmds := p.Commands()

	for i, line := range cmds {
		switch {
		case line.Command.Verb == "DO" && line.Command.WhileClause != "":
			stack = append(stack, pendingBlock{doIndex: i, doLine: line.Number})
		case line.Command.Verb == "ENDDO":
			if len(stack) == 0 {
				return fmt.Errorf("script: ENDDO without matching DO WHILE at line %d", line.Number)
			}
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			block := WhileBlock{
				DoIndex:    top.doIndex,
				StartIndex: top.doIndex + 1,
				EndIndex:   i,
			}
			p.whileBlocks[top.doIndex] = block
			p.endDoBlocks[i] = block
		}
	}

	if len(stack) > 0 {
		return fmt.Errorf("script: unclosed DO WHILE at line %d", stack[len(stack)-1].doLine)
	}

	return nil
}

// WhileBlockAt returns the compiled block for a DO WHILE command index.
func (p *Program) WhileBlockAt(doIndex int) (WhileBlock, bool) {
	if p == nil {
		return WhileBlock{}, false
	}
	block, ok := p.whileBlocks[doIndex]
	return block, ok
}

// WhileBlockForEnd returns the compiled block terminated by an ENDDO command index.
func (p *Program) WhileBlockForEnd(endIndex int) (WhileBlock, bool) {
	if p == nil {
		return WhileBlock{}, false
	}
	block, ok := p.endDoBlocks[endIndex]
	return block, ok
}

// WhileEnclosingAt returns the innermost DO WHILE block containing index.
// index must lie strictly between the DO WHILE header and its ENDDO.
func (p *Program) WhileEnclosingAt(index int) (WhileBlock, bool) {
	if p == nil || len(p.whileBlocks) == 0 {
		return WhileBlock{}, false
	}

	var best WhileBlock
	found := false
	for _, block := range p.whileBlocks {
		if index <= block.DoIndex || index >= block.EndIndex {
			continue
		}
		if !found || block.EndIndex < best.EndIndex {
			best = block
			found = true
		}
	}
	return best, found
}
