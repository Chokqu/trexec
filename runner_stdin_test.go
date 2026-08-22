package trexec_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Chokqu/trexec"
)

func TestStdinPipeInteractive(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var stdout bytes.Buffer
	cmd := trexec.CommandContext(ctx, helperBin, []string{"-stdin"},
		trexec.WithStdout(&stdout),
	)

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe failed: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start failed: %v", err)
	}

	// Stream lines into the process dynamically
	for i := 1; i <= 3; i++ {
		_, err := fmt.Fprintf(stdinPipe, "line %d\n", i)
		if err != nil {
			t.Fatalf("writing to stdinPipe failed: %v", err)
		}
	}

	// Close pipe to signal EOF to child
	if err := stdinPipe.Close(); err != nil {
		t.Fatalf("closing stdinPipe failed: %v", err)
	}

	result := cmd.Wait()
	if !result.Success() {
		t.Errorf("expected success, got: %s", result)
	}

	got := stdout.String()
	for i := 1; i <= 3; i++ {
		expectedLine := fmt.Sprintf("ECHO:line %d", i)
		if !strings.Contains(got, expectedLine) {
			t.Errorf("missing %q in output: %q", expectedLine, got)
		}
	}
}

func TestStdinPipeAfterStartError(t *testing.T) {
	ctx := context.Background()
	cmd := trexec.CommandContext(ctx, helperBin, []string{"-sleep=100ms"})

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	_, err := cmd.StdinPipe()
	if err != trexec.ErrAlreadyStarted {
		t.Errorf("expected ErrAlreadyStarted, got: %v", err)
	}

	cmd.Wait()
}

func TestStdinPipeAlreadySetError(t *testing.T) {
	ctx := context.Background()
	reader := strings.NewReader("already set input")
	cmd := trexec.CommandContext(ctx, helperBin, []string{"-stdin"},
		trexec.WithStdin(reader),
	)

	_, err := cmd.StdinPipe()
	if err == nil {
		t.Error("expected error when WithStdin is already configured")
	}
}
