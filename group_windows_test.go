//go:build windows

package trexec_test

import (
	"context"
	"testing"
	"time"

	"github.com/Chokqu/trexec"
)

func TestWindowsJobObjectTreeKill(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Spawn recursive tree on Windows
	cmd := trexec.CommandContext(ctx, helperBin, []string{"-depth=2", "-sleep=60s"},
		trexec.WithGracePeriod(1*time.Second),
	)

	if err := cmd.Start(); err != nil {
		t.Fatalf("Windows cmd.Start failed: %v", err)
	}

	time.Sleep(300 * time.Millisecond)
	cancel()

	result := cmd.Wait()
	if !result.Cancelled {
		t.Errorf("expected cancelled=true on Windows, got %v", result.Cancelled)
	}
}

func TestWindowsGracefulCtrlBreak(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cmd := trexec.CommandContext(ctx, helperBin, []string{"-graceful"},
		trexec.WithGracePeriod(5*time.Second),
	)

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)
	cancel()

	result := cmd.Wait()
	if !result.Cancelled {
		t.Error("expected cancelled=true")
	}
}
