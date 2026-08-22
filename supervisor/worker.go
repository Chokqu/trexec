package supervisor

import (
	"context"
	"sync"
	"time"

	"github.com/Chokqu/trexec"
)

type worker struct {
	spec Spec
	mu   sync.RWMutex

	running      bool
	restarts     int
	lastExitCode int
	lastDuration time.Duration
	lastResult   *trexec.Result

	currentRunner *trexec.Runner
}

func newWorker(spec Spec) *worker {
	if spec.Backoff == nil {
		spec.Backoff = DefaultBackoff()
	}
	if spec.GracePeriod == 0 {
		spec.GracePeriod = 5 * time.Second
	}
	return &worker{spec: spec}
}

func (w *worker) run(ctx context.Context) *trexec.Result {
	attempt := 1

	for {
		select {
		case <-ctx.Done():
			w.mu.Lock()
			w.running = false
			res := w.lastResult
			w.mu.Unlock()
			return res
		default:
		}

		var opts []trexec.Option
		if w.spec.GracePeriod > 0 {
			opts = append(opts, trexec.WithGracePeriod(w.spec.GracePeriod))
		}
		opts = append(opts, trexec.WithGracefulSignal(w.spec.GracefulSignal))
		if w.spec.Stdout != nil {
			opts = append(opts, trexec.WithStdout(w.spec.Stdout))
		}
		if w.spec.Stderr != nil {
			opts = append(opts, trexec.WithStderr(w.spec.Stderr))
		}
		if w.spec.Dir != "" {
			opts = append(opts, trexec.WithDir(w.spec.Dir))
		}
		if len(w.spec.Env) > 0 {
			opts = append(opts, trexec.WithEnv(w.spec.Env))
		}

		runner := trexec.CommandContext(ctx, w.spec.Command, w.spec.Args, opts...)

		w.mu.Lock()
		w.currentRunner = runner
		w.running = true
		w.mu.Unlock()

		result, err := runner.Run()

		w.mu.Lock()
		w.running = false
		if result != nil {
			w.lastExitCode = result.ExitCode
			w.lastDuration = result.Duration
			w.lastResult = result
		} else if err != nil {
			w.lastExitCode = -1
		}
		w.mu.Unlock()

		// If context was cancelled, shutdown loop
		if ctx.Err() != nil {
			return result
		}

		// Determine if restart is required
		shouldRestart := false
		switch w.spec.RestartPolicy {
		case RestartAlways:
			shouldRestart = true
		case RestartOnFailure:
			if result != nil && !result.Success() {
				shouldRestart = true
			} else if err != nil {
				shouldRestart = true
			}
		case RestartNever:
			shouldRestart = false
		}

		if !shouldRestart {
			return result
		}

		if w.spec.MaxRestarts > 0 && w.restarts >= w.spec.MaxRestarts {
			return result
		}

		w.mu.Lock()
		w.restarts++
		w.mu.Unlock()

		backoffDur := w.spec.Backoff.Duration(attempt)
		attempt++

		select {
		case <-time.After(backoffDur):
		case <-ctx.Done():
			return result
		}
	}
}

func (w *worker) status() WorkerStatus {
	w.mu.RLock()
	defer w.mu.RUnlock()

	pid := 0
	if w.currentRunner != nil {
		pid = w.currentRunner.PID()
	}

	return WorkerStatus{
		Name:         w.spec.Name,
		Running:      w.running,
		Restarts:     w.restarts,
		LastExitCode: w.lastExitCode,
		LastPID:      pid,
		LastDuration: w.lastDuration,
		LastResult:   w.lastResult,
	}
}
