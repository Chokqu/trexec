package trexec

import (
	"context"
)

// CommandContext creates a new Runner bound to the given context.
//
// The command will be started in its own process group (Unix) or
// Job Object (Windows). When the context is cancelled, the runner
// initiates graceful shutdown:
//
//  1. Send graceful termination signal to entire process tree
//  2. Wait for grace period (default: 5 seconds)
//  3. Force-kill remaining processes
//  4. Clean up I/O pipes
//
// The args parameter accepts string arguments for the command.
// Use Option values via the opts parameter to configure behavior.
//
// Example:
//
//	cmd := trexec.CommandContext(ctx, "npm", "run", "dev",
//	    trexec.WithGracePeriod(5 * time.Second),
//	    trexec.WithStdout(os.Stdout),
//	)
func CommandContext(ctx context.Context, name string, args []string, opts ...Option) *Runner {
	o := applyOptions(opts)
	return newRunner(name, args, ctx.Done(), o)
}

// Run executes a command and blocks until completion or cancellation.
// Returns nil if the command exits with code 0.
// Returns an error for both setup failures and non-zero exits.
//
// For more control over the result, use RunWithResult.
//
// Example:
//
//	err := trexec.Run(ctx, "make", []string{"build"})
func Run(ctx context.Context, name string, args []string, opts ...Option) error {
	r := CommandContext(ctx, name, args, opts...)
	result, err := r.Run()
	if err != nil {
		return err
	}
	return result.Error
}

// RunWithResult executes a command and returns a structured Result.
//
// The error return is non-nil only for setup/start failures (binary not found,
// permission denied, etc.). For process-level outcomes (exit codes,
// cancellation, force-kill), check Result fields.
//
// Example:
//
//	result, err := trexec.RunWithResult(ctx, "npm", []string{"run", "dev"},
//	    trexec.WithGracePeriod(5 * time.Second),
//	    trexec.WithStdout(os.Stdout),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("exit=%d cancelled=%v\n", result.ExitCode, result.Cancelled)
func RunWithResult(ctx context.Context, name string, args []string, opts ...Option) (*Result, error) {
	r := CommandContext(ctx, name, args, opts...)
	return r.Run()
}

// Output executes a command and returns its standard output along with the Result.
//
// Example:
//
//	out, result, err := trexec.Output(ctx, "git", []string{"status", "--porcelain"})
func Output(ctx context.Context, name string, args []string, opts ...Option) ([]byte, *Result, error) {
	r := CommandContext(ctx, name, args, opts...)
	return r.Output()
}

// CombinedOutput executes a command and returns its combined standard output and
// standard error along with the Result.
//
// Example:
//
//	out, result, err := trexec.CombinedOutput(ctx, "go", []string{"test", "./..."})
func CombinedOutput(ctx context.Context, name string, args []string, opts ...Option) ([]byte, *Result, error) {
	r := CommandContext(ctx, name, args, opts...)
	return r.CombinedOutput()
}
