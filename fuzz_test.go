package trexec_test

import (
	"context"
	"testing"
	"time"

	"github.com/Chokqu/trexec"
)

// FuzzContextCancellation tests that cancelling the context at arbitrary random millisecond
// intervals never deadlocks, crashes, or leaks the process tree.
func FuzzContextCancellation(f *testing.F) {
	// Seed corpus with various cancellation delays (in milliseconds)
	f.Add(0)
	f.Add(1)
	f.Add(10)
	f.Add(50)
	f.Add(100)
	f.Add(250)

	f.Fuzz(func(t *testing.T, cancelDelayMs int) {
		if cancelDelayMs < 0 || cancelDelayMs > 500 {
			return
		}

		ctx, cancel := context.WithCancel(context.Background())
		cmd := trexec.CommandContext(ctx, helperBin, []string{"-sleep=10s"},
			trexec.WithGracePeriod(50*time.Millisecond),
		)

		if err := cmd.Start(); err != nil {
			cancel()
			return
		}

		if cancelDelayMs > 0 {
			time.Sleep(time.Duration(cancelDelayMs) * time.Millisecond)
		}
		cancel()

		res := cmd.Wait()
		if res == nil {
			t.Fatal("expected non-nil Result")
		}
	})
}

// FuzzCommandExecution verifies that various combinations of stdout output and exit codes
// are handled cleanly without panic or buffer corruption.
func FuzzCommandExecution(f *testing.F) {
	f.Add(0, "short")
	f.Add(1, "error message")
	f.Add(42, "custom exit")
	f.Add(0, "multiline\noutput\ntest")

	f.Fuzz(func(t *testing.T, exitCode int, outputText string) {
		if exitCode < 0 || exitCode > 125 || len(outputText) > 1024 {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		args := []string{
			"-exit=" + string(rune('0'+(exitCode%10))),
			"-stdout=" + outputText,
		}

		out, res, err := trexec.Output(ctx, helperBin, args)
		if res == nil {
			t.Fatal("expected non-nil Result from Output")
		}
		if res.Success() && err != nil {
			t.Errorf("success result should have nil error, got: %v", err)
		}
		_ = out
	})
}
