package trexec

import "os/exec"

// processGroup manages a collection of related processes using
// OS-level primitives. Each platform provides its own implementation:
//   - Unix:    process groups via setpgid + kill(-pgid, signal)
//   - Windows: Job Objects with KILL_ON_JOB_CLOSE
type processGroup interface {
	// setup configures the exec.Cmd to use this process group.
	// Called before cmd.Start(). Must set SysProcAttr fields.
	setup(cmd *exec.Cmd) error

	// activate performs any post-Start() setup that requires the running process.
	// On Windows: assigns the suspended process to the Job Object and resumes it.
	// On Unix: records the PGID (setpgid happens during fork, so this is a no-op
	// beyond recording the PID).
	activate(cmd *exec.Cmd) error

	// signal sends a termination signal to all processes in the group.
	// Returns os.ErrProcessDone (or equivalent) if the group is already gone.
	signal(sig Signal) error

	// terminate forcefully kills all processes in the group.
	// This is the last resort — SIGKILL on Unix, TerminateJobObject on Windows.
	terminate() error

	// close releases any OS resources held by the group (e.g., Job Object handles).
	// Must be called after all processes have exited.
	close() error

	// pids returns the list of process IDs currently associated with this group.
	pids() ([]int, error)

	// setLimits applies resource boundaries to the group.
	setLimits(limits *ResourceLimits) error
}
