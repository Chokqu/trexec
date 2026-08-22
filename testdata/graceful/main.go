package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	fmt.Printf("PID=%d started\n", os.Getpid())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	sig := <-sigCh
	fmt.Printf("PID=%d received %s, shutting down gracefully\n", os.Getpid(), sig)

	// Simulate graceful cleanup work
	time.Sleep(300 * time.Millisecond)

	fmt.Printf("PID=%d shutdown complete\n", os.Getpid())
	os.Exit(0)
}
