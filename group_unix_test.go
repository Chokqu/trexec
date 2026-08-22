//go:build unix

package trexec_test

import (
	"context"
	"syscall"
	"testing"
	"time"

	"github.com/Chokqu/trexec"
)

func TestUnixProcessGroupIsolation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cmd := trexec.CommandContext(ctx, helperBin, []string{"-sleep=60s"},
		trexec.WithGracePeriod(1*time.Second),
	)

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	pid := cmd.PID()
	if pid <= 0 {
		t.Fatalf("invalid child PID: %d", pid)
	}

	// Verify the process group ID matches the child PID on Unix (setpgid(0, 0))
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		t.Fatalf("Getpgid(%d) failed: %v", pid, err)
	}
	if pgid != pid {
		t.Errorf("PGID = %d, want equal to child PID %d", pgid, pid)
	}

	cancel()
	result := cmd.Wait()
	if !result.Cancelled {
		t.Error("expected cancelled=true")
	}
}
