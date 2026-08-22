package telemetry_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Chokqu/trexec"
	"github.com/Chokqu/trexec/telemetry"
)

var helperBin string

func TestMain(m *testing.M) {
	tempDir, err := os.MkdirTemp("", "trexec-telemetry-test-*")
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

	buildCmd := exec.Command("go", "build", "-o", helperBin, "../testdata/helper")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build testhelper: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestMemorySinkEventsAndMetrics(t *testing.T) {
	sink := telemetry.NewMemorySink()

	event := telemetry.Event{
		Timestamp:       time.Now(),
		State:           trexec.StateRunning,
		PID:             1234,
		ActiveProcesses: 3,
		Message:         "workload started",
	}
	sink.EmitEvent(event)

	metrics := trexec.TreeMetrics{
		Timestamp:        time.Now(),
		ActiveProcesses:  3,
		TotalMemoryBytes: 1024 * 1024 * 50,
		State:            trexec.StateRunning,
	}
	sink.EmitMetrics(metrics)

	events := sink.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].PID != 1234 || events[0].ActiveProcesses != 3 {
		t.Errorf("unexpected event data: %+v", events[0])
	}

	allMetrics := sink.Metrics()
	if len(allMetrics) != 1 {
		t.Fatalf("expected 1 metric snapshot, got %d", len(allMetrics))
	}
	if allMetrics[0].TotalMemoryBytes != 1024*1024*50 {
		t.Errorf("unexpected metrics data: %+v", allMetrics[0])
	}

	sink.Reset()
	if len(sink.Events()) != 0 || len(sink.Metrics()) != 0 {
		t.Errorf("sink not empty after reset")
	}
}

func TestLoggerSink(t *testing.T) {
	var buf bytes.Buffer
	logger := telemetry.NewLoggerSink(&buf, "[TEST-TREXEC]")

	logger.EmitEvent(telemetry.Event{
		Timestamp: time.Now(),
		State:     trexec.StateStarting,
		PID:       555,
		Message:   "starting workload",
	})

	logger.EmitMetrics(trexec.TreeMetrics{
		Timestamp:       time.Now(),
		ActiveProcesses: 2,
		State:           trexec.StateRunning,
	})

	output := buf.String()
	if !strings.Contains(output, "[TEST-TREXEC]") {
		t.Errorf("expected prefix in logger output, got: %s", output)
	}
	if !strings.Contains(output, "starting") || !strings.Contains(output, "Metrics") {
		t.Errorf("expected event and metrics log lines, got: %s", output)
	}
}

func TestWithMetricsPollIntervalIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var mu sync.Mutex
	var collected []trexec.TreeMetrics

	opts := []trexec.Option{
		trexec.WithMetricsPollInterval(20*time.Millisecond, func(m trexec.TreeMetrics) {
			mu.Lock()
			collected = append(collected, m)
			mu.Unlock()
		}),
	}

	cmd := trexec.CommandContext(ctx, helperBin, []string{"-sleep=100ms"}, opts...)
	res, err := cmd.Run()
	if err != nil || !res.Success() {
		t.Fatalf("command failed: %v, res: %v", err, res)
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	count := len(collected)
	mu.Unlock()

	if count < 1 {
		t.Errorf("expected at least 1 metric sample, got %d", count)
	}
}

func TestHookOptionsBridge(t *testing.T) {
	sink := telemetry.NewMemorySink()
	opts := telemetry.HookOptions(sink, 20*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := trexec.CommandContext(ctx, helperBin, []string{"-sleep=100ms"}, opts...)
	res, err := cmd.Run()
	if err != nil || !res.Success() {
		t.Fatalf("command failed: %v, res: %v", err, res)
	}

	time.Sleep(100 * time.Millisecond)

	if len(sink.Events()) == 0 {
		t.Errorf("expected captured events via HookOptions, got 0")
	}
	if len(sink.Metrics()) == 0 {
		t.Errorf("expected captured metrics via HookOptions, got 0")
	}
}
