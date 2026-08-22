package trexec

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

// Runner owns a command's entire lifecycle: process creation, I/O management,
// graceful shutdown, forced termination, and resource cleanup.
//
// A Runner is created via CommandContext and used via Start()/Wait() or Run().
// It must not be reused after Wait() returns.
type Runner struct {
	// Configuration (immutable after creation)
	name string
	args []string
	opts *options

	// OS resources (set during Start)
	cmd             *exec.Cmd
	group           processGroup
	pipes           *pipeManager
	stdinReadCloser io.Closer
	trackedPIDs     []int

	// Lifecycle
	state         State
	mu            sync.Mutex
	onStateChange func(State)
	startTime     time.Time

	// Concurrency guards
	started atomic.Bool
	waited  atomic.Bool

	// Channels for goroutine coordination
	procDone chan struct{} // closed when cmd.Process.Wait() returns
	procErr  error        // result of cmd.Process.Wait()

	// Context (from CommandContext)
	ctxDone <-chan struct{} // ctx.Done() channel
}

// newRunner creates a Runner with the given configuration.
// Does not start the process — call Start() or Run() for that.
func newRunner(name string, args []string, ctxDone <-chan struct{}, opts *options) *Runner {
	return &Runner{
		name:          name,
		args:          args,
		opts:          opts,
		state:         StateCreated,
		onStateChange: opts.onStateChange,
		ctxDone:       ctxDone,
		procDone:      make(chan struct{}),
	}
}

// transition attempts to move the state machine from `from` to `to`.
// Returns true if the transition succeeded, false if the current state
// is not `from` (meaning another goroutine already advanced it).
func (r *Runner) transition(from, to State) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.state != from {
		return false
	}
	if !canTransition(from, to) {
		return false
	}

	r.state = to

	if r.onStateChange != nil {
		fn := r.onStateChange
		go fn(to)
	}
	return true
}

// forceTransition sets the state unconditionally. Used only for terminal
// transitions to StateDone where we must reach Done regardless.
func (r *Runner) forceTransition(to State) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.state = to

	if r.onStateChange != nil {
		fn := r.onStateChange
		go fn(to)
	}
}

// currentState returns the current state (thread-safe).
func (r *Runner) currentState() State {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.state
}

// Start starts the command but does not wait for it to complete.
//
// It creates the process group, starts the process, activates the group
// (e.g., assigns Job Object on Windows), and launches I/O and lifecycle
// goroutines.
//
// After Start returns successfully, Wait() must be called to release resources.
// Calling Start() more than once returns ErrAlreadyStarted.
func (r *Runner) Start() error {
	if !r.started.CompareAndSwap(false, true) {
		return ErrAlreadyStarted
	}

	if !r.transition(StateCreated, StateStarting) {
		return fmt.Errorf("trexec: invalid state for Start(): %s", r.currentState())
	}

	// Create the platform-specific process group.
	group, err := newProcessGroup()
	if err != nil {
		r.forceTransition(StateDone)
		return fmt.Errorf("trexec: failed to create process group: %w", err)
	}
	r.group = group

	// Build exec.Cmd.
	r.cmd = exec.Command(r.name, r.args...)
	r.cmd.Dir = r.opts.dir
	r.cmd.Env = r.opts.env
	r.cmd.ExtraFiles = r.opts.extraFiles

	if r.opts.sysProcAttr != nil {
		// Clone user SysProcAttr to avoid mutating the caller's struct
		clone := *r.opts.sysProcAttr
		r.cmd.SysProcAttr = &clone
	}

	if r.opts.stdin != nil {
		r.cmd.Stdin = r.opts.stdin
		if r.stdinReadCloser != nil {
			defer r.stdinReadCloser.Close()
		}
	}

	// Configure the process group (sets SysProcAttr).
	if err := r.group.setup(r.cmd); err != nil {
		r.group.close()
		r.forceTransition(StateDone)
		return fmt.Errorf("trexec: process group setup failed: %w", err)
	}

	// Set up I/O pipes.
	r.pipes = newPipeManager()
	if r.opts.stdout != nil {
		pr, pw, err := os.Pipe()
		if err != nil {
			r.group.close()
			r.forceTransition(StateDone)
			return fmt.Errorf("trexec: stdout pipe: %w", err)
		}
		r.cmd.Stdout = pw
		r.pipes.startCopy(r.opts.stdout, pr)
		// pw will be closed after cmd.Start() (child has inherited it).
		defer pw.Close()
	}
	if r.opts.stderr != nil {
		pr, pw, err := os.Pipe()
		if err != nil {
			r.group.close()
			r.forceTransition(StateDone)
			return fmt.Errorf("trexec: stderr pipe: %w", err)
		}
		r.cmd.Stderr = pw
		r.pipes.startCopy(r.opts.stderr, pr)
		defer pw.Close()
	}

	// Start the process.
	r.startTime = time.Now()
	if err := r.cmd.Start(); err != nil {
		r.group.close()
		r.forceTransition(StateDone)
		return fmt.Errorf("trexec: start failed: %w", err)
	}

	// Activate the process group (Windows: assign Job Object + resume).
	if err := r.group.activate(r.cmd); err != nil {
		// Process started but couldn't be assigned — kill it.
		r.cmd.Process.Kill()
		r.cmd.Wait()
		r.group.close()
		r.forceTransition(StateDone)
		return fmt.Errorf("trexec: process group activation failed: %w", err)
	}

	// Apply kernel resource limits if configured
	if r.opts.resourceLimits != nil {
		_ = r.group.setLimits(r.opts.resourceLimits)
	}

	r.transition(StateStarting, StateRunning)

	// Launch process wait goroutine.
	go r.waitForProcess()

	if r.opts.onMetrics != nil {
		go r.pollMetrics()
	}

	return nil
}

// pollMetrics periodically queries process metrics and invokes onMetrics.
func (r *Runner) pollMetrics() {
	interval := r.opts.metricsPollInterval
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-r.procDone:
			r.emitMetricsSnapshot(r.currentState())
			return
		case <-ticker.C:
			st := r.currentState()
			if st == StateDone {
				return
			}
			r.emitMetricsSnapshot(st)
		}
	}
}

// emitMetricsSnapshot gathers current metrics and invokes the onMetrics callback.
func (r *Runner) emitMetricsSnapshot(state State) {
	if r.opts.onMetrics == nil {
		return
	}

	var activeProcs int
	if r.group != nil {
		if pids, err := r.group.pids(); err == nil {
			activeProcs = len(pids)
		}
	}
	if activeProcs == 0 && state == StateRunning {
		activeProcs = 1
	}

	metrics := TreeMetrics{
		Timestamp:       time.Now(),
		ActiveProcesses: activeProcs,
		State:           state,
	}

	fn := r.opts.onMetrics
	fn(metrics)
}

// waitForProcess waits for the process to exit and records the result.
func (r *Runner) waitForProcess() {
	r.procErr = r.cmd.Wait()
	close(r.procDone)
}

// Wait blocks until the command completes (naturally or via cancellation)
// and returns the structured Result.
//
// Wait handles the complete shutdown lifecycle:
//   - If the context is cancelled: graceful signal → grace period → force kill
//   - Waits for I/O goroutines with timeout
//   - Releases process group resources
//
// Wait must be called exactly once after a successful Start().
func (r *Runner) Wait() *Result {
	if !r.waited.CompareAndSwap(false, true) {
		return &Result{
			ExitCode: -1,
			Error:    ErrAlreadyStarted,
		}
	}

	result := r.runLifecycle()

	// Cleanup: wait for I/O, close group.
	r.pipes.wait(r.opts.ioTimeout)
	r.group.close()
	r.forceTransition(StateDone)

	result.Duration = time.Since(r.startTime)
	return result
}

// runLifecycle manages the core lifecycle: wait for process exit or context
// cancellation, and handle the graceful → force kill escalation.
func (r *Runner) runLifecycle() *Result {
	select {
	case <-r.procDone:
		// Process exited on its own (no cancellation).
		return r.buildResult(false, false, false)

	case <-r.ctxDone:
		// Context was cancelled — begin shutdown sequence.
		return r.shutdownSequence()
	}
}

// shutdownSequence handles the graceful termination -> force kill escalation.
func (r *Runner) shutdownSequence() *Result {
	// Snapshot live PIDs before sending termination signals
	if r.group != nil {
		r.trackedPIDs, _ = r.group.pids()
	}

	// If grace period is zero, skip straight to force kill.
	if r.opts.gracePeriod <= 0 {
		r.transition(StateRunning, StateKilling)
		_ = r.group.terminate()

		// Hard deadline to prevent infinite hangs if terminate() fails or process is stuck in D state
		killDeadline := time.NewTimer(5 * time.Second)
		defer killDeadline.Stop()

		select {
		case <-r.procDone:
			return r.buildResult(true, false, true)
		case <-killDeadline.C:
			r.pipes.forceClose()
			return r.buildResult(true, false, true)
		}
	}

	// Phase 1: Graceful signal.
	r.transition(StateRunning, StateStopping)
	err := r.group.signal(r.opts.gracefulSignal)
	if err == os.ErrProcessDone {
		// Already exited — race between exit and cancel.
		<-r.procDone
		return r.buildResult(true, true, false)
	}

	// Wait for grace period or process exit.
	graceTimer := time.NewTimer(r.opts.gracePeriod)
	defer graceTimer.Stop()

	select {
	case <-r.procDone:
		// Process exited within grace period — graceful shutdown succeeded.
		return r.buildResult(true, true, false)

	case <-graceTimer.C:
		// Refresh PID snapshot before force-killing
		if r.group != nil {
			if livePIDs, err := r.group.pids(); err == nil && len(livePIDs) > 0 {
				r.trackedPIDs = livePIDs
			}
		}

		// Grace period expired — escalate to force kill.
		r.transition(StateStopping, StateKilling)
		_ = r.group.terminate()

		killDeadline := time.NewTimer(5 * time.Second)
		defer killDeadline.Stop()

		select {
		case <-r.procDone:
			return r.buildResult(true, false, true)
		case <-killDeadline.C:
			r.pipes.forceClose()
			return r.buildResult(true, false, true)
		}
	}
}

// buildResult constructs a Result from the current process state.
func (r *Runner) buildResult(cancelled, graceful, forceKilled bool) *Result {
	pids := r.trackedPIDs
	if len(pids) == 0 && r.group != nil {
		pids, _ = r.group.pids()
	}

	result := &Result{
		Cancelled:            cancelled,
		GracefullyTerminated: graceful,
		ForceKilled:          forceKilled,
		DescendantPIDs:       pids,
		ProcessesCleaned:     len(pids),
	}

	if r.procErr != nil {
		// Extract exit code from the os/exec error.
		if exitErr, ok := r.procErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}

		sig := ""
		if forceKilled {
			sig = "killed"
		}
		result.Error = &ExitError{
			ExitCode:  result.ExitCode,
			Signal:    sig,
			Cancelled: cancelled,
		}
	}
	// If procErr is nil, ExitCode stays 0 and Error stays nil — success.

	return result
}

// Run calls Start and then Wait. This is the most common usage pattern.
//
// The error return is non-nil only for setup/start failures (binary not found,
// permission denied, etc.). For process-level outcomes, check Result fields.
func (r *Runner) Run() (*Result, error) {
	if err := r.Start(); err != nil {
		return nil, err
	}
	return r.Wait(), nil
}

// PID returns the PID of the direct child process, or 0 if not yet started.
func (r *Runner) PID() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cmd != nil && r.cmd.Process != nil {
		return r.cmd.Process.Pid
	}
	return 0
}

// Output runs the command, waits for it to complete or cancel, and returns
// its standard output along with the structured Result.
//
// If stdout was already configured via WithStdout, Output returns an error.
// If the command fails with a non-zero exit code or is cancelled, Output returns
// the captured stdout bytes alongside the Result and any ExitError.
func (r *Runner) Output() ([]byte, *Result, error) {
	if r.opts.stdout != nil {
		return nil, nil, errors.New("trexec: Stdout already set")
	}
	var buf bytes.Buffer
	r.opts.stdout = &buf

	result, err := r.Run()
	if err != nil {
		return nil, nil, err
	}
	if result.Error != nil {
		return buf.Bytes(), result, result.Error
	}
	return buf.Bytes(), result, nil
}

// CombinedOutput runs the command, waits for it to complete or cancel, and
// returns its combined standard output and standard error along with the structured Result.
//
// If stdout or stderr was already configured, CombinedOutput returns an error.
// Output streams are synchronized so concurrent writes will not corrupt the buffer.
func (r *Runner) CombinedOutput() ([]byte, *Result, error) {
	if r.opts.stdout != nil || r.opts.stderr != nil {
		return nil, nil, errors.New("trexec: Stdout or Stderr already set")
	}
	var buf bytes.Buffer
	sw := newSyncWriter(&buf)
	r.opts.stdout = sw
	r.opts.stderr = sw

	result, err := r.Run()
	if err != nil {
		return nil, nil, err
	}
	if result.Error != nil {
		return buf.Bytes(), result, result.Error
	}
	return buf.Bytes(), result, nil
}

// StdinPipe returns a pipe that will be connected to the command's standard input
// when the command starts.
//
// The returned io.WriteCloser must be closed by the caller to send EOF to the child process.
// If the caller does not close it, it will be closed automatically when Wait() finishes.
// StdinPipe must be called before Start().
func (r *Runner) StdinPipe() (io.WriteCloser, error) {
	if r.started.Load() {
		return nil, ErrAlreadyStarted
	}
	if r.opts.stdin != nil {
		return nil, errors.New("trexec: Stdin already set")
	}

	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("trexec: StdinPipe: %w", err)
	}

	r.opts.stdin = pr
	r.stdinReadCloser = pr
	return pw, nil
}
