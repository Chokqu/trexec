package trexec

import (
	"context"
	"errors"
	"fmt"
)

// Sentinel errors for lifecycle violations.
var (
	// ErrAlreadyStarted is returned when Start() is called more than once on a Runner.
	ErrAlreadyStarted = errors.New("trexec: command already started")

	// ErrNotStarted is returned when Wait() is called before a successful Start().
	ErrNotStarted = errors.New("trexec: command not started")
)

// ExitError represents a process that exited with a non-zero code or was killed.
// It implements the error interface and supports errors.Is / errors.As.
type ExitError struct {
	// ExitCode is the process exit code. -1 if killed before normal exit.
	ExitCode int

	// Signal is the signal name that killed the process (e.g. "killed", "terminated").
	// Empty if the process exited on its own.
	Signal string

	// Stderr contains captured stderr output if stderr was not redirected
	// to a user-provided writer. May be nil or truncated.
	Stderr []byte

	// Cancelled indicates this exit was a consequence of context cancellation.
	Cancelled bool
}

// Error returns a human-readable description of the exit failure.
func (e *ExitError) Error() string {
	if e.Cancelled {
		if e.Signal != "" {
			return fmt.Sprintf("trexec: process killed by %s (cancelled)", e.Signal)
		}
		return fmt.Sprintf("trexec: process exited with code %d (cancelled)", e.ExitCode)
	}
	if e.Signal != "" {
		return fmt.Sprintf("trexec: process killed by %s", e.Signal)
	}
	return fmt.Sprintf("trexec: process exited with code %d", e.ExitCode)
}

// Unwrap allows errors.Is(err, context.Canceled) to work when the exit
// was caused by cancellation.
func (e *ExitError) Unwrap() error {
	if e.Cancelled {
		return context.Canceled
	}
	return nil
}
