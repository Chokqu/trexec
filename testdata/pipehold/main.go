package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "holder" {
		// Grandchild holds the inherited stdout pipe open
		for i := 0; i < 30; i++ {
			fmt.Fprintln(os.Stdout, "pipe holder active")
			time.Sleep(1 * time.Second)
		}
		return
	}

	// Direct child spawns holder, inherits stdout, and exits
	cmd := exec.Command(os.Args[0], "holder")
	cmd.Stdout = os.Stdout
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start holder: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("DIRECT_CHILD PID=%d exiting while holder PID=%d keeps stdout open\n",
		os.Getpid(), cmd.Process.Pid)
	os.Exit(0)
}
