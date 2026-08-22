package trexec_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Chokqu/trexec"
)

func TestOutputSuccess(t *testing.T) {
	ctx := context.Background()
	cmd := trexec.CommandContext(ctx, helperBin, []string{"-stdout=captured output"})

	out, result, err := cmd.Output()
	if err != nil {
		t.Fatalf("Output failed: %v", err)
	}

	if !result.Success() {
		t.Errorf("expected success, got: %s", result)
	}

	got := strings.TrimSpace(string(out))
	if got != "captured output" {
		t.Errorf("Output = %q, want %q", got, "captured output")
	}
}

func TestOutputNonZeroExit(t *testing.T) {
	ctx := context.Background()
	cmd := trexec.CommandContext(ctx, helperBin, []string{"-stdout=error details", "-exit=42"})

	out, result, err := cmd.Output()
	if err == nil {
		t.Error("expected error for exit 42")
	}

	if result.ExitCode != 42 {
		t.Errorf("ExitCode = %d, want 42", result.ExitCode)
	}

	got := strings.TrimSpace(string(out))
	if got != "error details" {
		t.Errorf("Output = %q, want %q", got, "error details")
	}
}

func TestOutputAlreadySetError(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	cmd := trexec.CommandContext(ctx, helperBin, []string{"-stdout=hello"},
		trexec.WithStdout(&buf),
	)

	_, _, err := cmd.Output()
	if err == nil {
		t.Error("expected error when WithStdout is already set")
	}
}

func TestCombinedOutput(t *testing.T) {
	ctx := context.Background()
	cmd := trexec.CommandContext(ctx, helperBin, []string{
		"-stdout=msg on stdout",
		"-stderr=msg on stderr",
	})

	out, result, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CombinedOutput failed: %v", err)
	}

	if !result.Success() {
		t.Errorf("expected success, got: %s", result)
	}

	outputStr := string(out)
	if !strings.Contains(outputStr, "msg on stdout") {
		t.Errorf("missing stdout in combined output: %q", outputStr)
	}
	if !strings.Contains(outputStr, "msg on stderr") {
		t.Errorf("missing stderr in combined output: %q", outputStr)
	}
}

func TestCombinedOutputAlreadySetError(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	cmd := trexec.CommandContext(ctx, helperBin, []string{"-stdout=hello"},
		trexec.WithStderr(&buf),
	)

	_, _, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("expected error when WithStderr is already set")
	}
}

func TestOutputPackageHelpers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, result, err := trexec.Output(ctx, helperBin, []string{"-stdout=pkg helper out"})
	if err != nil {
		t.Fatalf("trexec.Output helper failed: %v", err)
	}
	if !result.Success() || strings.TrimSpace(string(out)) != "pkg helper out" {
		t.Errorf("unexpected output helper result: %q, %s", string(out), result)
	}

	combOut, combResult, err := trexec.CombinedOutput(ctx, helperBin, []string{
		"-stdout=combined stdout",
		"-stderr=combined stderr",
	})
	if err != nil {
		t.Fatalf("trexec.CombinedOutput helper failed: %v", err)
	}
	if !combResult.Success() {
		t.Errorf("unexpected combined helper result: %s", combResult)
	}
	if !strings.Contains(string(combOut), "combined stdout") || !strings.Contains(string(combOut), "combined stderr") {
		t.Errorf("combined helper missing stream data: %q", string(combOut))
	}
}
