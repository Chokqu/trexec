package trexec

import "fmt"

// State represents a point in the command lifecycle.
//
// The lifecycle follows a strict state machine:
//
//	Created → Starting → Running → Done          (normal exit)
//	Created → Starting → Running → Stopping → Done (graceful shutdown)
//	Created → Starting → Running → Stopping → Killing → Done (force kill)
type State int

const (
	// StateCreated means the Runner has been constructed but Start() has not been called.
	StateCreated State = iota

	// StateStarting means Start() has been called and the process group is being set up.
	StateStarting

	// StateRunning means the process is alive, I/O is active, and the context is being watched.
	StateRunning

	// StateStopping means a graceful termination signal has been sent and we are waiting
	// for the process to exit within the grace period.
	StateStopping

	// StateKilling means the grace period expired and a force-kill signal has been sent.
	StateKilling

	// StateDone means all processes have exited, I/O is cleaned up, and the Result is available.
	// This is a terminal state.
	StateDone
)

// String returns the human-readable name of the state.
func (s State) String() string {
	switch s {
	case StateCreated:
		return "created"
	case StateStarting:
		return "starting"
	case StateRunning:
		return "running"
	case StateStopping:
		return "stopping"
	case StateKilling:
		return "killing"
	case StateDone:
		return "done"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

// validTransitions defines the legal state transition graph.
// Each key maps to the set of states it can transition to.
var validTransitions = map[State][]State{
	StateCreated:  {StateStarting},
	StateStarting: {StateRunning, StateDone},  // Done if start fails
	StateRunning:  {StateStopping, StateDone}, // Done if natural exit
	StateStopping: {StateKilling, StateDone},  // Done if graceful exit
	StateKilling:  {StateDone},
	StateDone:     {}, // terminal — no transitions out
}

// canTransition reports whether transitioning from the current state to the
// target state is valid according to the state machine.
func canTransition(from, to State) bool {
	for _, s := range validTransitions[from] {
		if s == to {
			return true
		}
	}
	return false
}
