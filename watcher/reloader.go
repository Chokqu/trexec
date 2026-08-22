package watcher

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/Chokqu/trexec"
)

// ReloaderConfig configures the live-reloading process supervisor.
type ReloaderConfig struct {
	// Watcher specifies the filesystem watching configuration.
	Watcher Config

	// Command is the binary name or command to execute.
	Command string

	// Args are the command line arguments.
	Args []string

	// Dir specifies the working directory.
	Dir string

	// Env specifies the process environment variables.
	Env []string

	// Stdout is the destination for standard output.
	Stdout io.Writer

	// Stderr is the destination for standard error.
	Stderr io.Writer

	// GracePeriod is the duration to wait before force-killing on reload. Defaults to 3s.
	GracePeriod time.Duration

	// GracefulSignal is the initial signal sent on reload. Defaults to trexec.SIGINT.
	GracefulSignal trexec.Signal

	// OnRestart is invoked right before the newly reloaded workload is spawned.
	OnRestart func(attempt int, changedFiles []string)
}

// Reloader supervises a long-running process tree, restarting it whenever filesystem changes occur.
type Reloader struct {
	cfg     ReloaderConfig
	watcher *Watcher
	mu      sync.Mutex
	running bool
}

// NewReloader constructs a new live Reloader instance.
func NewReloader(cfg ReloaderConfig) *Reloader {
	if cfg.GracePeriod <= 0 {
		cfg.GracePeriod = 3 * time.Second
	}
	return &Reloader{
		cfg:     cfg,
		watcher: New(cfg.Watcher),
	}
}

// Run starts the initial process workload and monitors for changes, restarting
// the workload upon detecting file modifications until ctx is cancelled.
func (r *Reloader) Run(ctx context.Context) error {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return errors.New("reloader: already running")
	}
	r.running = true
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		r.running = false
		r.mu.Unlock()
	}()

	changeCh, err := r.watcher.Start(ctx)
	if err != nil {
		return err
	}

	attempt := 1

	for {
		// Create a cancellable context dedicated to the current process tree
		workloadCtx, workloadCancel := context.WithCancel(ctx)

		var opts []trexec.Option
		if r.cfg.GracePeriod > 0 {
			opts = append(opts, trexec.WithGracePeriod(r.cfg.GracePeriod))
		}
		opts = append(opts, trexec.WithGracefulSignal(r.cfg.GracefulSignal))
		if r.cfg.Stdout != nil {
			opts = append(opts, trexec.WithStdout(r.cfg.Stdout))
		}
		if r.cfg.Stderr != nil {
			opts = append(opts, trexec.WithStderr(r.cfg.Stderr))
		}
		if r.cfg.Dir != "" {
			opts = append(opts, trexec.WithDir(r.cfg.Dir))
		}
		if len(r.cfg.Env) > 0 {
			opts = append(opts, trexec.WithEnv(r.cfg.Env))
		}

		runner := trexec.CommandContext(workloadCtx, r.cfg.Command, r.cfg.Args, opts...)
		if err := runner.Start(); err != nil {
			workloadCancel()
			if ctx.Err() != nil {
				return nil
			}
			// Wait for change to retry
			select {
			case <-ctx.Done():
				return nil
			case changedFiles, ok := <-changeCh:
				if !ok {
					return nil
				}
				if r.cfg.OnRestart != nil {
					r.cfg.OnRestart(attempt, changedFiles)
				}
				attempt++
				continue
			}
		}

		// Wait in goroutine for natural process exit
		procFinished := make(chan struct{})
		go func() {
			_ = runner.Wait()
			close(procFinished)
		}()

		// Wait for either:
		// 1. Filesystem change event -> trigger restart
		// 2. Parent context cancelled -> terminate and exit
		// 3. Process natural exit -> wait for next change to restart
		select {
		case <-ctx.Done():
			workloadCancel()
			<-procFinished
			return nil

		case changedFiles, ok := <-changeCh:
			if !ok {
				workloadCancel()
				<-procFinished
				return nil
			}

			// Cancel running workload and wait for complete process tree death
			workloadCancel()
			<-procFinished

			if r.cfg.OnRestart != nil {
				r.cfg.OnRestart(attempt, changedFiles)
			}
			attempt++

		case <-procFinished:
			// Process exited on its own; wait for next file modification to restart
			workloadCancel()
			select {
			case <-ctx.Done():
				return nil
			case changedFiles, ok := <-changeCh:
				if !ok {
					return nil
				}
				if r.cfg.OnRestart != nil {
					r.cfg.OnRestart(attempt, changedFiles)
				}
				attempt++
			}
		}
	}
}
