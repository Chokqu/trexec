//go:build unix

package trexec

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

// unixProcessGroup implements processGroup using POSIX process groups.
// It creates a new process group (via setpgid) and signals the entire
// group using kill with a negative PGID.
type unixProcessGroup struct {
	mu   sync.RWMutex
	pgid int
}

func newProcessGroup() (processGroup, error) {
	return &unixProcessGroup{}, nil
}

func (g *unixProcessGroup) setup(cmd *exec.Cmd) error {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	// Create a new process group with the child's PID as the PGID.
	// This means all descendants (that don't explicitly escape) will
	// share this group and can be signaled together.
	cmd.SysProcAttr.Setpgid = true
	cmd.SysProcAttr.Pgid = 0

	// Apply platform-specific safety nets (Pdeathsig on Linux).
	setupPdeathsig(cmd)

	return nil
}

func (g *unixProcessGroup) activate(cmd *exec.Cmd) error {
	// On Unix, setpgid happens during fork/exec — the PGID is the child's PID
	// when we used Pgid=0.
	g.mu.Lock()
	g.pgid = cmd.Process.Pid
	g.mu.Unlock()
	return nil
}

func (g *unixProcessGroup) signal(sig Signal) error {
	g.mu.RLock()
	pgid := g.pgid
	g.mu.RUnlock()

	if pgid == 0 {
		return os.ErrProcessDone
	}

	var s syscall.Signal
	switch sig {
	case SIGTERM:
		s = syscall.SIGTERM
	case SIGINT:
		s = syscall.SIGINT
	case SIGHUP:
		s = syscall.SIGHUP
	case SIGKILL:
		s = syscall.SIGKILL
	default:
		return fmt.Errorf("trexec: unsupported signal %s", sig)
	}

	// Negative PID sends the signal to the entire process group.
	err := syscall.Kill(-pgid, s)
	if err != nil {
		if errors.Is(err, syscall.ESRCH) {
			// No such process/group — already gone.
			return os.ErrProcessDone
		}
		return fmt.Errorf("trexec: kill(-%d, %s): %w", pgid, s, err)
	}
	return nil
}

func (g *unixProcessGroup) terminate() error {
	return g.signal(SIGKILL)
}

func (g *unixProcessGroup) close() error {
	// No OS resources to release on Unix — process groups are
	// purely a kernel-tracked attribute, not a handle.
	g.mu.Lock()
	g.pgid = 0
	g.mu.Unlock()
	return nil
}

func (g *unixProcessGroup) pids() ([]int, error) {
	g.mu.RLock()
	pgid := g.pgid
	g.mu.RUnlock()

	if pgid <= 0 {
		return nil, nil
	}
	pids := findGroupPIDs(pgid)
	if len(pids) == 0 {
		return []int{pgid}, nil
	}
	return pids, nil
}

func (g *unixProcessGroup) setLimits(limits *ResourceLimits) error {
	// On Unix, limits are managed via rlimit attributes or cgroups
	return nil
}

// findGroupPIDs inspects /proc (on Linux) to discover all live processes
// that belong to the specified process group ID.
func findGroupPIDs(targetPGID int) []int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return []int{targetPGID}
	}

	var pids []int
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}

		statBytes, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if err != nil {
			continue
		}
		// In /proc/[pid]/stat, format is: pid (comm) state ppid pgrp ...
		rparen := bytes.LastIndexByte(statBytes, ')')
		if rparen == -1 || len(statBytes) <= rparen+2 {
			continue
		}
		fields := strings.Fields(string(statBytes[rparen+2:]))
		if len(fields) >= 3 {
			// fields[0]=state, fields[1]=ppid, fields[2]=pgrp
			pgrp, err := strconv.Atoi(fields[2])
			if err == nil && pgrp == targetPGID {
				pids = append(pids, pid)
			}
		}
	}
	return pids
}
