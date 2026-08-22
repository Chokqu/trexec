package trexec

import (
	"io"
	"os"
	"syscall"
	"time"
)

// Option configures a Runner. Use the With* functions to create Options.
type Option func(*options)

// options holds the resolved configuration for a Runner.
type options struct {
	gracePeriod    time.Duration
	ioTimeout      time.Duration
	gracefulSignal Signal
	resourceLimits *ResourceLimits
	stdout         io.Writer
	stderr         io.Writer
	stdin          io.Reader
	dir            string
	env            []string
	extraFiles     []*os.File
	sysProcAttr         *syscall.SysProcAttr
	onStateChange       func(State)
	metricsPollInterval time.Duration
	onMetrics           func(TreeMetrics)
}

// defaultOptions returns the default configuration.
func defaultOptions() *options {
	return &options{
		gracePeriod:    5 * time.Second,
		ioTimeout:      2 * time.Second,
		gracefulSignal: SIGTERM,
	}
}

// applyOptions applies all Option functions to the default options.
func applyOptions(opts []Option) *options {
	o := defaultOptions()
	for _, fn := range opts {
		fn(o)
	}
	return o
}

// WithGracePeriod sets how long to wait after sending a graceful termination
// signal before force-killing. Default: 5 seconds.
//
// Set to 0 to skip graceful shutdown and immediately force-kill on cancellation.
func WithGracePeriod(d time.Duration) Option {
	return func(o *options) {
		o.gracePeriod = d
	}
}

// WithGracefulSignal configures the initial polite signal sent on cancellation.
// Default: SIGTERM.
//
// On Unix, this broadcasts the specified signal (e.g. SIGINT, SIGHUP, SIGTERM)
// to the entire process group. On Windows, SIGINT maps to CTRL_C_EVENT and
// SIGTERM/SIGHUP maps to CTRL_BREAK_EVENT.
func WithGracefulSignal(sig Signal) Option {
	return func(o *options) {
		o.gracefulSignal = sig
	}
}

// WithIOTimeout sets how long to wait for I/O goroutines to finish after
// the process exits. This handles the case where orphaned subprocesses
// hold pipes open. Default: 2 seconds.
func WithIOTimeout(d time.Duration) Option {
	return func(o *options) {
		o.ioTimeout = d
	}
}

// WithStdout sets the writer for the command's standard output.
// If not set, stdout is discarded.
func WithStdout(w io.Writer) Option {
	return func(o *options) {
		o.stdout = w
	}
}

// WithStderr sets the writer for the command's standard error.
// If not set, stderr is discarded.
func WithStderr(w io.Writer) Option {
	return func(o *options) {
		o.stderr = w
	}
}

// WithStdin sets the reader for the command's standard input.
func WithStdin(r io.Reader) Option {
	return func(o *options) {
		o.stdin = r
	}
}

// WithDir sets the working directory of the command.
// If empty, the current process's working directory is used.
func WithDir(dir string) Option {
	return func(o *options) {
		o.dir = dir
	}
}

// WithEnv sets the environment variables for the command.
// Each entry should be in "KEY=VALUE" format.
// If nil, the current process's environment is inherited.
func WithEnv(env []string) Option {
	return func(o *options) {
		o.env = env
	}
}

// WithExtraFiles passes additional open files to the child process.
// They will be available as file descriptors 3, 4, 5, etc.
func WithExtraFiles(files []*os.File) Option {
	return func(o *options) {
		o.extraFiles = files
	}
}

// WithSysProcAttr sets additional platform-specific process attributes.
// trexec will merge these with its own required fields (e.g., Setpgid on Unix).
// Conflicting fields will cause Start() to return an error.
func WithSysProcAttr(attr *syscall.SysProcAttr) Option {
	return func(o *options) {
		o.sysProcAttr = attr
	}
}

// WithOnStateChange registers a callback invoked on lifecycle state transitions.
// The callback is called from a dedicated goroutine and must not block.
// Useful for logging and monitoring.
func WithOnStateChange(fn func(State)) Option {
	return func(o *options) {
		o.onStateChange = fn
	}
}

// TreeMetrics contains a real-time snapshot of the process tree's resource consumption and state.
type TreeMetrics struct {
	// Timestamp is the moment the metrics snapshot was taken.
	Timestamp time.Time

	// ActiveProcesses is the count of live descendant processes in the tree.
	ActiveProcesses int

	// TotalMemoryBytes is the cumulative resident memory across the process tree (if available).
	TotalMemoryBytes int64

	// TotalCPUTime is the aggregate CPU execution duration (if available).
	TotalCPUTime time.Duration

	// State is the current lifecycle state of the command.
	State State
}

// WithMetricsPollInterval configures periodic metric emission for the process tree.
// If interval is <= 0, a default interval of 500ms is used.
// The callback is invoked from a dedicated goroutine and must not block.
func WithMetricsPollInterval(interval time.Duration, callback func(TreeMetrics)) Option {
	return func(o *options) {
		if interval <= 0 {
			interval = 500 * time.Millisecond
		}
		o.metricsPollInterval = interval
		o.onMetrics = callback
	}
}

