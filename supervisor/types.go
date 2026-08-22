package supervisor

import (
	"fmt"
	"io"
	"time"

	"github.com/Chokqu/trexec"
)

// RestartPolicy defines how the supervisor handles process termination.
type RestartPolicy int

const (
	// RestartNever never restarts the process regardless of its exit code.
	RestartNever RestartPolicy = iota

	// RestartAlways unconditionally restarts the process whenever it exits.
	RestartAlways

	// RestartOnFailure restarts the process only if it exits with a non-zero code or error.
	RestartOnFailure
)

// String returns the string representation of the restart policy.
func (p RestartPolicy) String() string {
	switch p {
	case RestartNever:
		return "RestartNever"
	case RestartAlways:
		return "RestartAlways"
	case RestartOnFailure:
		return "RestartOnFailure"
	default:
		return fmt.Sprintf("RestartPolicy(%d)", int(p))
	}
}

// Spec defines the configuration for a supervised process.
type Spec struct {
	// Name is the unique identifier for this supervised process.
	Name string

	// Command is the executable binary name or path.
	Command string

	// Args are the command-line arguments.
	Args []string

	// RestartPolicy defines under what exit conditions the process is restarted.
	RestartPolicy RestartPolicy

	// MaxRestarts limits the number of restart attempts. <= 0 means unlimited.
	MaxRestarts int

	// Backoff configures exponential delay between restarts. Default: DefaultBackoff().
	Backoff *Backoff

	// GracePeriod is the duration to wait before force-killing on shutdown. Default: 5s.
	GracePeriod time.Duration

	// GracefulSignal is the initial signal sent on cancellation. Default: trexec.SIGTERM.
	GracefulSignal trexec.Signal

	// Stdout is the destination for standard output.
	Stdout io.Writer

	// Stderr is the destination for standard error.
	Stderr io.Writer

	// Dir specifies the working directory for the command.
	Dir string

	// Env specifies the environment variables for the command.
	Env []string
}

// WorkerStatus contains the real-time status of a supervised worker.
type WorkerStatus struct {
	Name         string
	Running      bool
	Restarts     int
	LastExitCode int
	LastPID      int
	LastDuration time.Duration
	LastResult   *trexec.Result
}
