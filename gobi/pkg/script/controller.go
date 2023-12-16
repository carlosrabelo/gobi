package script

import "fmt"

// Controller tracks instruction pointer state while executing a command file.
type Controller struct {
	frames []*frame
}

type frame struct {
	program *Program
	ip      int
}

// NewController returns a controller positioned at the first executable line.
func NewController(prog *Program) *Controller {
	if prog == nil {
		return &Controller{}
	}
	return &Controller{
		frames: []*frame{{program: prog, ip: 0}},
	}
}

// Depth returns the number of active script frames.
func (c *Controller) Depth() int {
	if c == nil {
		return 0
	}
	return len(c.frames)
}

// Program returns the active program, or nil when no frame is active.
func (c *Controller) Program() *Program {
	f := c.activeFrame()
	if f == nil {
		return nil
	}
	return f.program
}

// Index returns the active instruction pointer, or -1 when idle.
func (c *Controller) Index() int {
	f := c.activeFrame()
	if f == nil {
		return -1
	}
	return f.ip
}

// CommandCount returns executable commands in the active program.
func (c *Controller) CommandCount() int {
	f := c.activeFrame()
	if f == nil || f.program == nil {
		return 0
	}
	return len(f.program.Commands())
}

// Current returns the command line at the instruction pointer.
func (c *Controller) Current() (Line, bool) {
	f := c.activeFrame()
	if f == nil || f.program == nil {
		return Line{}, false
	}
	cmds := f.program.Commands()
	if f.ip < 0 || f.ip >= len(cmds) {
		return Line{}, false
	}
	return cmds[f.ip], true
}

// AtEnd reports whether the instruction pointer is past the last command.
func (c *Controller) AtEnd() bool {
	f := c.activeFrame()
	if f == nil || f.program == nil {
		return true
	}
	return f.ip >= len(f.program.Commands())
}

// Advance moves to the next executable line and reports whether one remains.
func (c *Controller) Advance() bool {
	f := c.activeFrame()
	if f == nil {
		return false
	}
	f.ip++
	return f.ip < len(f.program.Commands())
}

// SetIndex positions the instruction pointer at index within executable commands.
func (c *Controller) SetIndex(index int) error {
	f := c.activeFrame()
	if f == nil || f.program == nil {
		return fmt.Errorf("script: no active program")
	}
	count := len(f.program.Commands())
	if index < 0 || index > count {
		return fmt.Errorf("script: instruction index out of range")
	}
	f.ip = index
	return nil
}

// JumpToLine positions the instruction pointer at the first command on lineNumber.
func (c *Controller) JumpToLine(lineNumber int) error {
	f := c.activeFrame()
	if f == nil || f.program == nil {
		return fmt.Errorf("script: no active program")
	}
	cmds := f.program.Commands()
	for i, line := range cmds {
		if line.Number >= lineNumber {
			f.ip = i
			return nil
		}
	}
	return fmt.Errorf("script: line %d not found", lineNumber)
}

// PushFrame starts executing prog on a nested call frame.
func (c *Controller) PushFrame(prog *Program) error {
	if c == nil {
		return fmt.Errorf("script: nil controller")
	}
	if prog == nil {
		return fmt.Errorf("script: nil program")
	}
	c.frames = append(c.frames, &frame{program: prog, ip: 0})
	return nil
}

// PopFrame ends the active frame and restores the caller frame when present.
func (c *Controller) PopFrame() bool {
	if c == nil || len(c.frames) == 0 {
		return false
	}
	c.frames = c.frames[:len(c.frames)-1]
	return len(c.frames) > 0
}

func (c *Controller) activeFrame() *frame {
	if c == nil || len(c.frames) == 0 {
		return nil
	}
	return c.frames[len(c.frames)-1]
}
