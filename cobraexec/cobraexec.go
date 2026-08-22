package cobraexec

import (
	"context"
	"io"

	"github.com/Chokqu/trexec"
)

// CobraLikeCommand abstracts any CLI command exposing standard Cobra execution attributes.
// This interface allows seamless integration with github.com/spf13/cobra without forcing
// it as a hard external dependency in caller go.mod files.
type CobraLikeCommand interface {
	// Context returns the active context bound to the CLI command (e.g. from signal traps).
	Context() context.Context

	// OutOrStdout returns the writer configured for command stdout.
	OutOrStdout() io.Writer

	// ErrOrStderr returns the writer configured for command stderr.
	ErrOrStderr() io.Writer

	// InOrStdin returns the reader configured for command stdin.
	InOrStdin() io.Reader
}

// Run executes a subprocess workload bound to a Cobra command's context and I/O streams.
func Run(cmd CobraLikeCommand, name string, args []string, opts ...trexec.Option) (*trexec.Result, error) {
	ctx := context.Background()
	var defaultOpts []trexec.Option

	if cmd != nil {
		if c := cmd.Context(); c != nil {
			ctx = c
		}
		if out := cmd.OutOrStdout(); out != nil {
			defaultOpts = append(defaultOpts, trexec.WithStdout(out))
		}
		if errOut := cmd.ErrOrStderr(); errOut != nil {
			defaultOpts = append(defaultOpts, trexec.WithStderr(errOut))
		}
		if in := cmd.InOrStdin(); in != nil {
			defaultOpts = append(defaultOpts, trexec.WithStdin(in))
		}
	}

	mergedOpts := append(defaultOpts, opts...)
	return trexec.RunWithResult(ctx, name, args, mergedOpts...)
}

// Output captures standard output from a subprocess tree while binding to the Cobra command's context.
func Output(cmd CobraLikeCommand, name string, args []string, opts ...trexec.Option) ([]byte, *trexec.Result, error) {
	ctx := context.Background()
	if cmd != nil {
		if c := cmd.Context(); c != nil {
			ctx = c
		}
	}
	return trexec.Output(ctx, name, args, opts...)
}

// CombinedOutput captures combined stdout and stderr from a subprocess tree while binding to the Cobra command's context.
func CombinedOutput(cmd CobraLikeCommand, name string, args []string, opts ...trexec.Option) ([]byte, *trexec.Result, error) {
	ctx := context.Background()
	if cmd != nil {
		if c := cmd.Context(); c != nil {
			ctx = c
		}
	}
	return trexec.CombinedOutput(ctx, name, args, opts...)
}

// WrapRunE returns an idiomatic Cobra RunE closure executing a subprocess tree.
//
// Usage:
//
//	var devCmd = &cobra.Command{
//	    Use:   "dev",
//	    RunE:  cobraexec.WrapRunE("npm", []string{"run", "dev"}, trexec.WithGracePeriod(5*time.Second)),
//	}
func WrapRunE(name string, args []string, opts ...trexec.Option) func(cmd CobraLikeCommand, cliArgs []string) error {
	return func(cmd CobraLikeCommand, cliArgs []string) error {
		res, err := Run(cmd, name, args, opts...)
		if err != nil {
			return err
		}
		if res.Cancelled && res.GracefullyTerminated {
			return nil
		}
		return res.Error
	}
}
