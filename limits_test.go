package trexec_test

import (
	"context"
	"testing"
	"time"

	"github.com/Chokqu/trexec"
)

func TestWithResourceLimits(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	limits := trexec.ResourceLimits{
		MaxMemoryBytes: 256 * 1024 * 1024, // 256 MB
		MaxProcesses:   10,
	}

	cmd := trexec.CommandContext(ctx, helperBin, []string{"-stdout=limits test"},
		trexec.WithResourceLimits(limits),
	)

	out, result, err := cmd.Output()
	if err != nil {
		t.Fatalf("command execution with resource limits failed: %v", err)
	}

	if !result.Success() {
		t.Errorf("expected success with valid resource limits, got: %s", result)
	}

	if string(out) != "limits test\n" {
		t.Errorf("output = %q, want %q", string(out), "limits test\n")
	}
}
