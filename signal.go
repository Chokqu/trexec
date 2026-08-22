package trexec

import "fmt"

// Signal represents a cross-platform process termination signal.
//
// Instead of requiring callers to manage platform-specific syscall signals
// (like syscall.SIGTERM on Unix vs CTRL_BREAK_EVENT on Windows), trexec exposes
// unified signal identifiers that each platform backend automatically translates
// to the appropriate kernel primitive.
type Signal int

const (
	// SIGTERM represents a polite request to terminate gracefully.
	//   - Unix:    Broadcasts syscall.SIGTERM to the entire process group (-PGID).
	//   - Windows: Sends CTRL_BREAK_EVENT to the process group.
	SIGTERM Signal = iota

	// SIGINT represents a keyboard interrupt (like Ctrl+C).
	//   - Unix:    Broadcasts syscall.SIGINT to the entire process group (-PGID).
	//   - Windows: Sends CTRL_C_EVENT to the process group.
	SIGINT

	// SIGHUP represents a terminal hangup or configuration reload.
	//   - Unix:    Broadcasts syscall.SIGHUP to the entire process group (-PGID).
	//   - Windows: Sends CTRL_BREAK_EVENT to the process group.
	SIGHUP

	// SIGKILL represents an unconditional kernel-level force-kill.
	//   - Unix:    Broadcasts syscall.SIGKILL to the entire process group (-PGID).
	//   - Windows: Calls TerminateJobObject on the Job Object / TerminateProcess.
	SIGKILL
)

// String returns the canonical name of the signal.
func (s Signal) String() string {
	switch s {
	case SIGTERM:
		return "SIGTERM"
	case SIGINT:
		return "SIGINT"
	case SIGHUP:
		return "SIGHUP"
	case SIGKILL:
		return "SIGKILL"
	default:
		return fmt.Sprintf("SIGNAL(%d)", int(s))
	}
}
