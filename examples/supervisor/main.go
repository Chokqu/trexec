package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/Chokqu/trexec"
)

// SupervisedWorker demonstrates an embedded supervisor loop that automatically
// monitors and restarts worker processes upon failure with exponential backoff.
func SupervisedWorker(ctx context.Context, name string, command string, args []string) {
	backoff := 500 * time.Millisecond
	maxBackoff := 5 * time.Second
	attempt := 1

	for {
		select {
		case <-ctx.Done():
			log.Printf("[%s] Supervisor shutting down worker...", name)
			return
		default:
		}

		log.Printf("[%s] Starting worker process (attempt #%d)...", name, attempt)

		// Each run gets its own bounded lifecycle with guaranteed tree cleanup
		result, err := trexec.RunWithResult(ctx, command, args,
			trexec.WithGracePeriod(2*time.Second),
			trexec.WithGracefulSignal(trexec.SIGTERM),
			trexec.WithStdout(os.Stdout),
			trexec.WithStderr(os.Stderr),
		)

		if err != nil {
			log.Printf("[%s] Start error: %v", name, err)
		} else {
			log.Printf("[%s] Worker stopped: %s (cleaned %d processes)", name, result, result.ProcessesCleaned)
		}

		if ctx.Err() != nil {
			log.Printf("[%s] Context cancelled, exiting supervisor loop.", name)
			return
		}

		log.Printf("[%s] Worker exited unexpectedly, backing off for %s before restart...", name, backoff)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		}

		// Exponential backoff
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
		attempt++
	}
}

func main() {
	fmt.Println("=== trexec Production Worker Supervisor ===")
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Supervise a background task
	go SupervisedWorker(ctx, "worker-1", "go", []string{"version"})

	// Run supervisor for a demo duration
	time.Sleep(3 * time.Second)
	fmt.Println("\nDemo duration finished, stopping supervisor...")
	cancel()
	time.Sleep(500 * time.Millisecond)
	fmt.Println("All workers safely drained.")
}
