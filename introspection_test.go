package trexec_test

import (
	"context"
	"testing"
	"time"

	"github.com/Chokqu/trexec"
)

func TestDescendantPIDsTracking(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Recursively spawns child processes to depth 2 (root -> child -> grandchild)
	cmd := trexec.CommandContext(ctx, helperBin, []string{"-depth=2", "-sleep=60s"},
		trexec.WithGracePeriod(1*time.Second),
	)

	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start failed: %v", err)
	}

	// Give the tree time to spawn grandchildren
	time.Sleep(300 * time.Millisecond)
	cancel()

	result := cmd.Wait()

	if !result.Cancelled {
		t.Error("expected result.Cancelled to be true")
	}

	// Verify that descendant PIDs were tracked
	if len(result.DescendantPIDs) == 0 {
		t.Error("expected at least 1 tracked PID in DescendantPIDs")
	}
	if result.ProcessesCleaned == 0 {
		t.Error("expected ProcessesCleaned to be > 0")
	}
}
