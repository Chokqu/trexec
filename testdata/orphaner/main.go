package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "child" {
		fmt.Printf("ORPHAN_CHILD PID=%d\n", os.Getpid())
		time.Sleep(60 * time.Second)
		return
	}

	cmd := exec.Command(os.Args[0], "child")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start child: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("PARENT PID=%d started child PID=%d, now exiting immediately\n",
		os.Getpid(), cmd.Process.Pid)
	// Exit without waiting, leaving child as orphan
	os.Exit(0)
}
