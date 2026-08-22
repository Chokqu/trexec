//go:build linux

package trexec

import (
	"os/exec"
	"syscall"
)

// setupPdeathsig configures the child process to receive SIGKILL if the
// parent thread dies. This is a Linux-only safety net using prctl(PR_SET_PDEATHSIG).
//
// This handles the case where the Go parent is killed with SIGKILL and
// cannot run any cleanup code — the kernel will automatically signal the child.
//
// Note: Pdeathsig is tied to the parent *thread*, not the parent *process*.
// In Go, the runtime may schedule goroutines on different threads, but the
// thread that calls fork/exec is the one whose death triggers Pdeathsig.
func setupPdeathsig(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Pdeathsig = syscall.SIGKILL
}
