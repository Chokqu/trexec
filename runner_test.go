package trexec_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/Chokqu/trexec"
)

var helperBin string

func TestMain(m *testing.M) {
	tempDir, err := os.MkdirTemp("", "trexec-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	binName := "testhelper"
	if os.Getenv("GOOS") == "windows" || filepath.Ext(os.Args[0]) == ".exe" {
		binName += ".exe"
	}
	helperBin = filepath.Join(tempDir, binName)

	buildCmd := exec.Command("go", "build", "-o", helperBin, "./testdata/helper")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build testhelper: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestSimpleExecution(t *testing.T) {
	ctx := context.Background()
	var stdout bytes.Buffer

	result, err := trexec.RunWithResult(ctx, helperBin, []string{"-stdout=hello world"},
		trexec.WithStdout(&stdout),
	)
	if err != nil {
		t.Fatalf("RunWithResult failed: %v", err)
	}

	if !result.Success() {
		t.Errorf("expected success, got: %s", result)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if result.Cancelled {
		t.Error("should not be cancelled")
	}

	got := strings.TrimSpace(stdout.String())
	if got != "hello world" {
		t.Errorf("stdout = %q, want %q", got, "hello world")
	}
}

func TestNonZeroExit(t *testing.T) {
	ctx := context.Background()

	result, err := trexec.RunWithResult(ctx, helperBin, []string{"-exit=42"})
	if err != nil {
		t.Fatalf("RunWithResult failed: %v", err)
	}

	if result.Success() {
		t.Error("should not be success")
	}
	if result.ExitCode != 42 {
		t.Errorf("ExitCode = %d, want 42", result.ExitCode)
	}
	if result.Error == nil {
		t.Error("Error should not be nil")
	}
}

func TestStderrCapture(t *testing.T) {
	ctx := context.Background()
	var stderr bytes.Buffer

	result, err := trexec.RunWithResult(ctx, helperBin, []string{"-stderr=error_msg"},
		trexec.WithStderr(&stderr),
	)
	if err != nil {
		t.Fatalf("RunWithResult failed: %v", err)
	}

	if !result.Success() {
		t.Errorf("expected success, got: %s", result)
	}

	got := strings.TrimSpace(stderr.String())
	if got != "error_msg" {
		t.Errorf("stderr = %q, want %q", got, "error_msg")
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cmd := trexec.CommandContext(ctx, helperBin, []string{"-sleep=60s"},
		trexec.WithGracePeriod(1*time.Second),
	)

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Give the process a moment to start.
	time.Sleep(100 * time.Millisecond)

	// Cancel the context.
	cancel()

	result := cmd.Wait()

	if !result.Cancelled {
		t.Error("result should be cancelled")
	}
	if result.Duration < 50*time.Millisecond {
		t.Errorf("duration too short: %v", result.Duration)
	}
}

func TestGracefulShutdown(t *testing.T) {
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
		t.Error("result should be cancelled")
	}
	if !result.GracefullyTerminated {
		t.Error("result should be gracefully terminated")
	}
	if result.ForceKilled {
		t.Error("result should not be force-killed")
	}
}

func TestWithGracefulSignalINT(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Helper traps os.Interrupt / SIGINT
	cmd := trexec.CommandContext(ctx, helperBin, []string{"-graceful"},
		trexec.WithGracePeriod(5*time.Second),
		trexec.WithGracefulSignal(trexec.SIGINT),
	)

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)
	cancel()

	result := cmd.Wait()

	if !result.Cancelled {
		t.Error("result should be cancelled")
	}
	if !result.GracefullyTerminated {
		t.Error("result should be gracefully terminated via SIGINT")
	}
}

func TestForceKill(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Helper configured to ignore SIGTERM
	cmd := trexec.CommandContext(ctx, helperBin, []string{"-ignore-sig", "-sleep=60s"},
		trexec.WithGracePeriod(500*time.Millisecond),
	)

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	start := time.Now()
	cancel()
	result := cmd.Wait()
	elapsed := time.Since(start)

	if !result.Cancelled {
		t.Error("result should be cancelled")
	}
	if !result.ForceKilled {
		t.Error("result should be force-killed")
	}
	if result.GracefullyTerminated {
		t.Error("result should not be gracefully terminated")
	}

	// Should take approximately the grace period.
	if elapsed < 400*time.Millisecond || elapsed > 3*time.Second {
		t.Errorf("force kill took %v, expected ~500ms", elapsed)
	}
}

func TestZeroGracePeriod(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cmd := trexec.CommandContext(ctx, helperBin, []string{"-sleep=60s"},
		trexec.WithGracePeriod(0),
	)

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	start := time.Now()
	cancel()
	result := cmd.Wait()
	elapsed := time.Since(start)

	if !result.Cancelled {
		t.Error("result should be cancelled")
	}
	if !result.ForceKilled {
		t.Error("result should be force-killed (zero grace period)")
	}
	if elapsed > 2*time.Second {
		t.Errorf("zero grace period kill took %v, expected < 2s", elapsed)
	}
}

func TestProcessTreeKill(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Recursively spawns child processes to depth 2
	cmd := trexec.CommandContext(ctx, helperBin, []string{"-depth=2", "-sleep=60s"},
		trexec.WithGracePeriod(1*time.Second),
	)

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(300 * time.Millisecond)
	cancel()

	result := cmd.Wait()

	if !result.Cancelled {
		t.Error("result should be cancelled")
	}
}

func TestRunConvenienceFunction(t *testing.T) {
	err := trexec.Run(context.Background(), helperBin, []string{"-stdout=hello"})
	if err != nil {
		t.Errorf("Run() returned error: %v", err)
	}
}

func TestRunConvenienceFunctionError(t *testing.T) {
	err := trexec.Run(context.Background(), helperBin, []string{"-exit=1"})
	if err == nil {
		t.Error("Run() should return error for non-zero exit")
	}
}

func TestBinaryNotFound(t *testing.T) {
	_, err := trexec.RunWithResult(context.Background(),
		"nonexistent-binary-that-does-not-exist-12345",
		nil,
	)
	if err == nil {
		t.Error("expected error for nonexistent binary")
	}
}

func TestDoubleStart(t *testing.T) {
	ctx := context.Background()
	cmd := trexec.CommandContext(ctx, helperBin, []string{"-stdout=hello"})

	if err := cmd.Start(); err != nil {
		t.Fatalf("first Start() failed: %v", err)
	}

	err := cmd.Start()
	if err != trexec.ErrAlreadyStarted {
		t.Errorf("second Start() = %v, want ErrAlreadyStarted", err)
	}

	cmd.Wait() // cleanup
}

func TestWorkingDirectory(t *testing.T) {
	targetDir := os.TempDir()
	var stdout bytes.Buffer

	result, err := trexec.RunWithResult(context.Background(), helperBin, []string{"-pwd"},
		trexec.WithDir(targetDir),
		trexec.WithStdout(&stdout),
	)
	if err != nil {
		t.Fatalf("RunWithResult failed: %v", err)
	}
	if !result.Success() {
		t.Errorf("expected success, got: %s", result)
	}

	got := strings.TrimSpace(stdout.String())
	// Resolve symlinks if any (e.g. /var vs /private/var on macOS)
	targetResolved, _ := filepath.EvalSymlinks(targetDir)
	gotResolved, _ := filepath.EvalSymlinks(got)

	if !strings.EqualFold(gotResolved, targetResolved) {
		t.Errorf("pwd = %q (resolved %q), want %q (resolved %q)", got, gotResolved, targetDir, targetResolved)
	}
}

func TestWithSysProcAttr(t *testing.T) {
	ctx := context.Background()
	customAttr := &syscall.SysProcAttr{}

	result, err := trexec.RunWithResult(ctx, helperBin, []string{"-stdout=test_attr"},
		trexec.WithSysProcAttr(customAttr),
	)
	if err != nil {
		t.Fatalf("RunWithResult with custom SysProcAttr failed: %v", err)
	}
	if !result.Success() {
		t.Errorf("expected success with WithSysProcAttr, got: %s", result)
	}
}

func TestAlreadyCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before start

	result, err := trexec.RunWithResult(ctx, helperBin, []string{"-stdout=hello"})
	if err != nil {
		return
	}
	if result.ExitCode == 0 && !result.Cancelled {
		return
	}
	if result.Cancelled {
		return
	}
	t.Errorf("unexpected result: %s", result)
}

func TestStateChangeCallback(t *testing.T) {
	var mu sync.Mutex
	var states []trexec.State

	result, err := trexec.RunWithResult(context.Background(), helperBin, []string{"-stdout=hello"},
		trexec.WithOnStateChange(func(s trexec.State) {
			mu.Lock()
			states = append(states, s)
			mu.Unlock()
		}),
	)
	if err != nil {
		t.Fatalf("RunWithResult failed: %v", err)
	}
	if !result.Success() {
		t.Errorf("expected success, got: %s", result)
	}

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	count := len(states)
	mu.Unlock()

	if count < 2 {
		mu.Lock()
		t.Errorf("expected at least 2 state changes, got %d: %v", count, states)
		mu.Unlock()
	}
}

func TestResultDuration(t *testing.T) {
	start := time.Now()
	result, err := trexec.RunWithResult(context.Background(), helperBin, []string{"-sleep=100ms"})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("RunWithResult failed: %v", err)
	}

	if result.Duration < 50*time.Millisecond {
		t.Errorf("Duration = %v, expected >= 50ms", result.Duration)
	}
	if result.Duration > elapsed+500*time.Millisecond {
		t.Errorf("Duration %v much larger than elapsed %v", result.Duration, elapsed)
	}
}
