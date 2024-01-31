package hostterm

import "io"

// EnterAltScreen switches to the terminal's alternate screen buffer when
// Caps.Clear is available (ANSI). Saves the host scrollback for LeaveAltScreen.
func EnterAltScreen(out io.Writer, caps Caps) error {
	if !caps.Clear {
		return nil
	}
	_, err := io.WriteString(out, "\x1b[?1049h")
	return err
}

// LeaveAltScreen restores the host screen saved by EnterAltScreen.
func LeaveAltScreen(out io.Writer, caps Caps) error {
	if !caps.Clear {
		return nil
	}
	_, err := io.WriteString(out, "\x1b[?1049l")
	return err
}
