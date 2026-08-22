package watcher_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Chokqu/trexec/watcher"
)

var helperBin string

func TestMain(m *testing.M) {
	tempDir, err := os.MkdirTemp("", "trexec-watcher-test-*")
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

func TestWatcherDetectChangesAndDebounce(t *testing.T) {
	watchDir, err := os.MkdirTemp("", "watcher-test-dir-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(watchDir)

	testFile := filepath.Join(watchDir, "app.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := watcher.Config{
		Paths:        []string{watchDir},
		Extensions:   []string{".go"},
		IgnoredNames: []string{".git"},
		Debounce:     100 * time.Millisecond,
		PollInterval: 50 * time.Millisecond,
	}

	w := watcher.New(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ch, err := w.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}

	// Give watcher a moment to initialize baseline
	time.Sleep(100 * time.Millisecond)

	// Perform 3 rapid writes within the debounce window
	for i := 0; i < 3; i++ {
		_ = os.WriteFile(testFile, []byte(fmt.Sprintf("package main // write %d", i)), 0644)
		time.Sleep(20 * time.Millisecond)
	}

	select {
	case changed, ok := <-ch:
		if !ok {
			t.Fatal("channel closed prematurely")
		}
		if len(changed) == 0 {
			t.Errorf("expected changed files, got 0")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for debounced change event")
	}
}

func TestWatcherExtensionAndIgnoreFiltering(t *testing.T) {
	watchDir, err := os.MkdirTemp("", "watcher-filter-dir-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(watchDir)

	gitDir := filepath.Join(watchDir, ".git")
	_ = os.MkdirAll(gitDir, 0755)

	cfg := watcher.Config{
		Paths:        []string{watchDir},
		Extensions:   []string{".go"},
		IgnoredNames: []string{".git"},
		Debounce:     50 * time.Millisecond,
		PollInterval: 50 * time.Millisecond,
	}

	w := watcher.New(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	ch, err := w.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)

	// 1. Write ignored file extension (.txt)
	_ = os.WriteFile(filepath.Join(watchDir, "readme.txt"), []byte("text"), 0644)
	// 2. Write file inside ignored directory (.git/HEAD)
	_ = os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main"), 0644)

	select {
	case changed := <-ch:
		t.Errorf("unexpected change event for ignored/non-matching files: %v", changed)
	case <-time.After(300 * time.Millisecond):
		// Success: no event received for ignored files
	}
}

func TestReloaderRestartOnModification(t *testing.T) {
	watchDir, err := os.MkdirTemp("", "reloader-test-dir-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(watchDir)

	sourceFile := filepath.Join(watchDir, "main.go")
	_ = os.WriteFile(sourceFile, []byte("package main"), 0644)

	var mu sync.Mutex
	restarts := 0

	reloader := watcher.NewReloader(watcher.ReloaderConfig{
		Watcher: watcher.Config{
			Paths:        []string{watchDir},
			Extensions:   []string{".go"},
			Debounce:     50 * time.Millisecond,
			PollInterval: 50 * time.Millisecond,
		},
		Command:     helperBin,
		Args:        []string{"-sleep=60s"},
		GracePeriod: 200 * time.Millisecond,
		OnRestart: func(attempt int, changedFiles []string) {
			mu.Lock()
			restarts++
			mu.Unlock()
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- reloader.Run(ctx)
	}()

	// Wait for process to spawn
	time.Sleep(200 * time.Millisecond)

	// Trigger file modification to initiate reload
	_ = os.WriteFile(sourceFile, []byte("package main // modified"), 0644)

	// Wait for reload trigger
	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	rCount := restarts
	mu.Unlock()

	if rCount < 1 {
		t.Errorf("expected at least 1 reload, got %d", rCount)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("reloader exited with error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("reloader timed out shutting down")
	}
}
