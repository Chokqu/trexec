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

func main() {
	fmt.Println("=== trexec Dev Server Lifecycle Manager ===")
	fmt.Println("Starting background service with process group supervision...")

	// Listen for Ctrl+C from user
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Launch a simulated dev server process
	// Works identically across Linux, macOS, and Windows without bash/sh
	cmd := trexec.CommandContext(ctx, "go", []string{"version"},
		trexec.WithGracePeriod(3*time.Second),
		trexec.WithGracefulSignal(trexec.SIGINT),
		trexec.WithStdout(os.Stdout),
		trexec.WithStderr(os.Stderr),
		trexec.WithOnStateChange(func(s trexec.State) {
			log.Printf("[trexec:state] -> %s", s)
		}),
	)

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start service: %v", err)
	}

	log.Printf("Service started (Root PID: %d). Press Ctrl+C to test graceful shutdown...", cmd.PID())

	result := cmd.Wait()

	fmt.Println("\n--- Service Termination Summary ---")
	fmt.Printf("Outcome:                 %s\n", result)
	fmt.Printf("Cancelled by user:       %v\n", result.Cancelled)
	fmt.Printf("Gracefully Terminated:   %v\n", result.GracefullyTerminated)
	fmt.Printf("Force Killed:            %v\n", result.ForceKilled)
	fmt.Printf("Processes Cleaned:       %d\n", result.ProcessesCleaned)
	fmt.Printf("Total Uptime:            %s\n", result.Duration.Round(time.Millisecond))
}
