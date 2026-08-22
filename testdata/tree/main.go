package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"
)

func main() {
	depth := 2
	if len(os.Args) > 1 {
		if d, err := strconv.Atoi(os.Args[1]); err == nil && d >= 0 {
			depth = d
		}
	}

	fmt.Printf("PID=%d PPID=%d depth=%d\n", os.Getpid(), os.Getppid(), depth)

	if depth > 0 {
		cmd := exec.Command(os.Args[0], strconv.Itoa(depth-1))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to start child: %v\n", err)
			os.Exit(1)
		}
		defer cmd.Wait()
	}

	time.Sleep(60 * time.Second)
}
