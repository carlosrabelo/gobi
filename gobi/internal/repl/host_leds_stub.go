//go:build !linux && !windows

package repl

// hostCapsOn / hostNumOn report keyboard LED state when the host can probe it.
// Unsupported hosts leave them off (segment stays blank until Ins/Del apply).
func hostCapsOn() bool { return false }
func hostNumOn() bool  { return false }
