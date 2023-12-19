package script

import "fmt"

// IfBlock describes a compiled IF ... [ELSE] ... ENDIF region in command index space.
type IfBlock struct {
	IfIndex   int
	ElseIndex int // -1 when ELSE is absent
	EndIndex  int // ENDIF command index
}

// buildIfBlocks indexes IF regions in executable command order.
func (p *Program) buildIfBlocks() error {
	if p == nil {
		return nil
	}

	p.ifBlocks = make(map[int]IfBlock)
	p.elseSkip = make(map[int]int)

	type pendingBlock struct {
		ifIndex   int
		elseIndex int
		ifLine    int
	}

	stack := make([]pendingBlock, 0)
	cmds := p.Commands()

	for i, line := range cmds {
		switch line.Command.Verb {
		case "IF":
			stack = append(stack, pendingBlock{ifIndex: i, elseIndex: -1, ifLine: line.Number})
		case "ELSE":
			if len(stack) == 0 {
				return fmt.Errorf("script: ELSE without matching IF at line %d", line.Number)
			}
			top := &stack[len(stack)-1]
			if top.elseIndex >= 0 {
				return fmt.Errorf("script: multiple ELSE for IF at line %d", top.ifLine)
			}
			top.elseIndex = i
		case "ENDIF":
			if len(stack) == 0 {
				return fmt.Errorf("script: ENDIF without matching IF at line %d", line.Number)
			}
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			block := IfBlock{
				IfIndex:   top.ifIndex,
				ElseIndex: top.elseIndex,
				EndIndex:  i,
			}
			p.ifBlocks[top.ifIndex] = block
			if top.elseIndex >= 0 {
				p.elseSkip[top.elseIndex] = i + 1
			}
		}
	}

	if len(stack) > 0 {
		return fmt.Errorf("script: unclosed IF at line %d", stack[len(stack)-1].ifLine)
	}

	return nil
}

// IfBlockAt returns the compiled block for an IF command index.
func (p *Program) IfBlockAt(ifIndex int) (IfBlock, bool) {
	if p == nil {
		return IfBlock{}, false
	}
	block, ok := p.ifBlocks[ifIndex]
	return block, ok
}

// SkipAfterElse returns the command index following the matching ENDIF.
func (p *Program) SkipAfterElse(elseIndex int) (int, bool) {
	if p == nil {
		return 0, false
	}
	target, ok := p.elseSkip[elseIndex]
	return target, ok
}
