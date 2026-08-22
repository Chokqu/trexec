package trexec

import (
	"context"
	"errors"
	"testing"
)

func TestExitErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		err  ExitError
		want string
	}{
		{
			name: "normal exit code",
			err:  ExitError{ExitCode: 1},
			want: "trexec: process exited with code 1",
		},
		{
			name: "killed by signal",
			err:  ExitError{ExitCode: -1, Signal: "killed"},
			want: "trexec: process killed by killed",
		},
		{
			name: "cancelled with exit code",
			err:  ExitError{ExitCode: 1, Cancelled: true},
			want: "trexec: process exited with code 1 (cancelled)",
		},
		{
			name: "cancelled and killed",
			err:  ExitError{ExitCode: -1, Signal: "killed", Cancelled: true},
			want: "trexec: process killed by killed (cancelled)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExitErrorUnwrap(t *testing.T) {
	t.Run("cancelled unwraps to context.Canceled", func(t *testing.T) {
		err := &ExitError{ExitCode: -1, Cancelled: true}
		if !errors.Is(err, context.Canceled) {
			t.Error("errors.Is(err, context.Canceled) = false, want true")
		}
	})

	t.Run("not cancelled unwraps to nil", func(t *testing.T) {
		err := &ExitError{ExitCode: 1, Cancelled: false}
		if errors.Is(err, context.Canceled) {
			t.Error("errors.Is(err, context.Canceled) = true, want false")
		}
	})
}

func TestExitErrorAs(t *testing.T) {
	var origErr error = &ExitError{ExitCode: 42, Signal: "terminated"}

	var exitErr *ExitError
	if !errors.As(origErr, &exitErr) {
		t.Fatal("errors.As failed")
	}
	if exitErr.ExitCode != 42 {
		t.Errorf("ExitCode = %d, want 42", exitErr.ExitCode)
	}
	if exitErr.Signal != "terminated" {
		t.Errorf("Signal = %q, want %q", exitErr.Signal, "terminated")
	}
}

func TestSentinelErrors(t *testing.T) {
	if ErrAlreadyStarted.Error() != "trexec: command already started" {
		t.Errorf("unexpected ErrAlreadyStarted message: %q", ErrAlreadyStarted.Error())
	}
	if ErrNotStarted.Error() != "trexec: command not started" {
		t.Errorf("unexpected ErrNotStarted message: %q", ErrNotStarted.Error())
	}
}
