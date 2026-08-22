package trexec

import (
	"testing"
	"time"
)

func TestResultSuccess(t *testing.T) {
	tests := []struct {
		name   string
		result Result
		want   bool
	}{
		{
			name:   "clean exit",
			result: Result{ExitCode: 0},
			want:   true,
		},
		{
			name:   "non-zero exit",
			result: Result{ExitCode: 1},
			want:   false,
		},
		{
			name:   "cancelled graceful",
			result: Result{ExitCode: 0, Cancelled: true, GracefullyTerminated: true},
			want:   false,
		},
		{
			name:   "has error",
			result: Result{ExitCode: 0, Error: &ExitError{ExitCode: 0}},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.Success(); got != tt.want {
				t.Errorf("Success() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResultString(t *testing.T) {
	tests := []struct {
		name   string
		result Result
		want   string
	}{
		{
			name:   "success",
			result: Result{ExitCode: 0, Duration: 150 * time.Millisecond},
			want:   "exit=0 duration=150ms",
		},
		{
			name: "cancelled graceful",
			result: Result{
				ExitCode:             0,
				Cancelled:            true,
				GracefullyTerminated: true,
				Duration:             2 * time.Second,
			},
			want: "cancelled (graceful exit=0) duration=2s",
		},
		{
			name: "cancelled force-killed",
			result: Result{
				Cancelled:        true,
				ForceKilled:      true,
				ProcessesCleaned: 3,
				Duration:         7 * time.Second,
			},
			want: "cancelled (force-killed, 3 processes cleaned) duration=7s",
		},
		{
			name: "failure",
			result: Result{
				ExitCode: 1,
				Duration: 500 * time.Millisecond,
			},
			want: "exit=1 duration=500ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}
