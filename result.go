package trexec

import (
	"fmt"
	"time"
)

// Result contains the complete outcome of a command execution.
//
// Unlike os/exec which returns a single error, Result preserves all
// information about what happened: exit code, whether cancellation was
// involved, whether the process shut down gracefully or was force-killed,
// and timing information.
type Result struct {
	// ExitCode is the process exit code.
	// 0 means success. -1 means the process was killed before it could exit normally.
	ExitCode int

	// Cancelled is true if the command was stopped due to context cancellation.
	Cancelled bool

	// GracefullyTerminated is true if the process exited on its own after
	// receiving a graceful termination signal (SIGTERM on Unix, Ctrl+Break on Windows).
	// This can only be true when Cancelled is also true.
	GracefullyTerminated bool

	// ForceKilled is true if the process had to be forcefully terminated
	// after the grace period expired. This can only be true when Cancelled is also true.
	ForceKilled bool

	// Duration is the total wall-clock time from Start() to cleanup completion.
	Duration time.Duration

	// ProcessesCleaned is the count of descendant processes that were terminated
	// during cleanup.
	ProcessesCleaned int

	// DescendantPIDs contains the process IDs of the descendant processes
	// tracked by the process group / Job Object during execution.
	DescendantPIDs []int

	// Error contains the underlying error, if any.
	// For non-zero exits: *ExitError. For cancellation-related kills: *ExitError
	// with Cancelled=true. nil for successful exits.
	Error error
}

// Success returns true if the command completed with exit code 0
// and was not cancelled. This is the "everything went perfectly" check.
func (r *Result) Success() bool {
	return r.ExitCode == 0 && !r.Cancelled && r.Error == nil
}

// String returns a human-readable summary of the result.
func (r *Result) String() string {
	switch {
	case r.Success():
		return fmt.Sprintf("exit=0 duration=%s", r.Duration.Round(time.Millisecond))
	case r.Cancelled && r.GracefullyTerminated:
		return fmt.Sprintf("cancelled (graceful exit=%d) duration=%s",
			r.ExitCode, r.Duration.Round(time.Millisecond))
	case r.Cancelled && r.ForceKilled:
		return fmt.Sprintf("cancelled (force-killed, %d processes cleaned) duration=%s",
			r.ProcessesCleaned, r.Duration.Round(time.Millisecond))
	case r.Cancelled:
		return fmt.Sprintf("cancelled exit=%d duration=%s",
			r.ExitCode, r.Duration.Round(time.Millisecond))
	default:
		return fmt.Sprintf("exit=%d duration=%s",
			r.ExitCode, r.Duration.Round(time.Millisecond))
	}
}
