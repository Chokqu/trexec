package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Chokqu/trexec"
	"github.com/Chokqu/trexec/telemetry"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("📊 trexec Real-Time Telemetry & Observability Demo")
	fmt.Println("==================================================")

	sink := telemetry.NewLoggerSink(os.Stdout, "📡 [METRICS]")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	opts := telemetry.HookOptions(sink, 200*time.Millisecond)
	opts = append(opts,
		trexec.WithStdout(os.Stdout),
		trexec.WithStderr(os.Stderr),
		trexec.WithGracePeriod(1*time.Second),
	)

	cmd := trexec.CommandContext(ctx, "go", []string{"version"}, opts...)

	fmt.Println("🚀 Starting workload with periodic telemetry poller...")
	res, err := cmd.Run()
	if err != nil {
		log.Fatalf("Command failed: %v", err)
	}

	fmt.Printf("🏁 Execution finished: %s\n", res)
	fmt.Printf("⏱️  Wall-clock Duration: %s, Exit Code: %d\n", res.Duration, res.ExitCode)
}
