package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Chokqu/trexec/watcher"
)

func main() {
	// Create a demo watched directory and test file
	demoDir, err := os.MkdirTemp("", "trexec-watch-demo-*")
	if err != nil {
		log.Fatalf("failed to create demo dir: %v", err)
	}
	defer os.RemoveAll(demoDir)

	demoFile := filepath.Join(demoDir, "server.go")
	_ = os.WriteFile(demoFile, []byte("// initial file"), 0644)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println("==================================================")
	fmt.Println("🚀 trexec Live-Reload DevServer Demo")
	fmt.Printf("📂 Watching directory: %s\n", demoDir)
	fmt.Println("💡 Edit files in the directory to trigger hot-reloading")
	fmt.Println("🛑 Press Ctrl+C to terminate cleanly")
	fmt.Println("==================================================")

	reloader := watcher.NewReloader(watcher.ReloaderConfig{
		Watcher: watcher.Config{
			Paths:        []string{demoDir},
			Extensions:   []string{".go", ".html", ".json"},
			IgnoredNames: []string{".git", "vendor"},
			Debounce:     150 * time.Millisecond,
			PollInterval: 100 * time.Millisecond,
		},
		Command:     "go",
		Args:        []string{"version"},
		GracePeriod: 2 * time.Second,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		OnRestart: func(attempt int, changedFiles []string) {
			log.Printf("🔄 [Reload #%d] Files changed: %v -> Drained old workload, starting new tree...", attempt, changedFiles)
		},
	})

	// Simulate a file modification after 500ms
	go func() {
		time.Sleep(500 * time.Millisecond)
		log.Println("📝 Simulating code change by modifying server.go...")
		_ = os.WriteFile(demoFile, []byte("// updated code content"), 0644)
	}()

	// Auto-exit after 2 seconds for automated demo run
	go func() {
		time.Sleep(2 * time.Second)
		log.Println("✅ Demo completed successfully, stopping reloader...")
		stop()
	}()

	if err := reloader.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("Reloader failed: %v", err)
	}

	fmt.Println("👋 Reloader shutdown complete. All child process trees drained.")
}
