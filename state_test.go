package trexec

import (
	"testing"
)

func TestStateString(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{StateCreated, "created"},
		{StateStarting, "starting"},
		{StateRunning, "running"},
		{StateStopping, "stopping"},
		{StateKilling, "killing"},
		{StateDone, "done"},
		{State(99), "unknown(99)"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("State(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestCanTransition(t *testing.T) {
	valid := []struct {
		from, to State
	}{
		{StateCreated, StateStarting},
		{StateStarting, StateRunning},
		{StateStarting, StateDone},
		{StateRunning, StateStopping},
		{StateRunning, StateDone},
		{StateStopping, StateKilling},
		{StateStopping, StateDone},
		{StateKilling, StateDone},
	}
	for _, tt := range valid {
		if !canTransition(tt.from, tt.to) {
			t.Errorf("canTransition(%s, %s) = false, want true", tt.from, tt.to)
		}
	}

	invalid := []struct {
		from, to State
	}{
		{StateCreated, StateRunning},    // skip Starting
		{StateCreated, StateDone},       // skip everything
		{StateRunning, StateKilling},    // skip Stopping
		{StateDone, StateCreated},       // terminal
		{StateDone, StateRunning},       // terminal
		{StateKilling, StateStopping},   // backwards
		{StateStopping, StateRunning},   // backwards
		{StateStarting, StateStopping},  // skip Running
	}
	for _, tt := range invalid {
		if canTransition(tt.from, tt.to) {
			t.Errorf("canTransition(%s, %s) = true, want false", tt.from, tt.to)
		}
	}
}
