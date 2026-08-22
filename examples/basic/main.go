package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Chokqu/trexec"
)

func main() {
	fmt.Println("=== trexec Basic Example ===")

	// 1. Direct command execution with structured result
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fmt.Println("\n1. Running Go version check with full metadata capture...")
	result, err := trexec.RunWithResult(ctx, "go", []string{"version"},
		trexec.WithStdout(os.Stdout),
		trexec.WithStderr(os.Stderr),
		trexec.WithGracePeriod(2*time.Second),
	)
	if err != nil {
		log.Fatalf("Command start failed: %v", err)
	}

	fmt.Printf("Exit Code: %d, Cleaned PIDs: %d, Duration: %s, Success: %v\n",
		result.ExitCode, result.ProcessesCleaned, result.Duration.Round(time.Millisecond), result.Success())

	// 2. Output buffering convenience function
	fmt.Println("\n2. Capturing output directly with trexec.Output()...")
	out, outResult, err := trexec.Output(ctx, "go", []string{"env", "GOVERSION"})
	if err != nil {
		log.Fatalf("Output failed: %v", err)
	}
	fmt.Printf("Detected Go version: %s (duration: %s)\n",
		string(out), outResult.Duration.Round(time.Millisecond))
}
