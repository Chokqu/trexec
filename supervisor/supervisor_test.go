package supervisor_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Chokqu/trexec/supervisor"
)

var helperBin string

func TestMain(m *testing.M) {
	tempDir, err := os.MkdirTemp("", "trexec-sup-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	helperBin = filepath.Join(tempDir, "testhelper")
	if runtime.GOOS == "windows" {
		helperBin += ".exe"
	}

	buildCmd := exec.Command("go", "build", "-o", helperBin, "../testdata/helper")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build testhelper: %v\noutput: %s\n", err, out)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestSupervisorRestartNever(t *testing.T) {
	sup := supervisor.New()
	err := sup.Add(supervisor.Spec{
		Name:          "oneshot",
		Command:       helperBin,
		Args:          []string{"-exit=0"},
		RestartPolicy: supervisor.RestartNever,
	})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sup.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	results := sup.Wait()
	res, ok := results["oneshot"]
	if !ok || res == nil {
		t.Fatal("missing result for oneshot worker")
	}

	if !res.Success() {
		t.Errorf("expected success, got: %s", res)
	}

	status := sup.Status()["oneshot"]
	if status.Restarts != 0 {
		t.Errorf("expected 0 restarts, got %d", status.Restarts)
	}
}

func TestSupervisorRestartOnFailure(t *testing.T) {
	sup := supervisor.New()
	err := sup.Add(supervisor.Spec{
		Name:          "failing-worker",
		Command:       helperBin,
		Args:          []string{"-exit=42"},
		RestartPolicy: supervisor.RestartOnFailure,
		MaxRestarts:   2,
		Backoff: &supervisor.Backoff{
			Min:    10 * time.Millisecond,
			Max:    50 * time.Millisecond,
			Factor: 2.0,
			Jitter: 0.0,
		},
	})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sup.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	results := sup.Wait()
	res, ok := results["failing-worker"]
	if !ok || res == nil {
		t.Fatal("missing result for failing-worker")
	}

	if res.ExitCode != 42 {
		t.Errorf("ExitCode = %d, want 42", res.ExitCode)
	}

	status := sup.Status()["failing-worker"]
	if status.Restarts != 2 {
		t.Errorf("Restarts = %d, want 2", status.Restarts)
	}
}

func TestSupervisorStop(t *testing.T) {
	sup := supervisor.New()
	err := sup.Add(supervisor.Spec{
		Name:          "long-running",
		Command:       helperBin,
		Args:          []string{"-sleep=60s"},
		RestartPolicy: supervisor.RestartAlways,
		GracePeriod:   500 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	ctx := context.Background()
	if err := sup.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Give worker time to start
	time.Sleep(150 * time.Millisecond)

	status := sup.Status()["long-running"]
	if !status.Running {
		t.Error("worker should be running")
	}

	// Stop supervisor gracefully
	if err := sup.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	statusAfter := sup.Status()["long-running"]
	if statusAfter.Running {
		t.Error("worker should no longer be running after Stop()")
	}
}
