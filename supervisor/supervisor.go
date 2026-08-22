package supervisor

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/Chokqu/trexec"
)

// Supervisor manages a pool of supervised worker processes.
type Supervisor struct {
	mu      sync.RWMutex
	workers map[string]*worker
	started bool
	stopped bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	results map[string]*trexec.Result
}

// New creates a new Supervisor instance.
func New() *Supervisor {
	return &Supervisor{
		workers: make(map[string]*worker),
		results: make(map[string]*trexec.Result),
	}
}

// Add registers a new process specification to be supervised.
// Must be called before Start().
func (s *Supervisor) Add(spec Spec) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return errors.New("supervisor: cannot Add spec after Start()")
	}
	if spec.Name == "" {
		return errors.New("supervisor: spec name cannot be empty")
	}
	if _, exists := s.workers[spec.Name]; exists {
		return fmt.Errorf("supervisor: worker with name %q already exists", spec.Name)
	}

	s.workers[spec.Name] = newWorker(spec)
	return nil
}

// Start begins supervising all registered workers in background goroutines.
func (s *Supervisor) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return errors.New("supervisor: already started")
	}
	s.started = true

	supCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	workers := make([]*worker, 0, len(s.workers))
	for _, w := range s.workers {
		workers = append(workers, w)
	}
	s.mu.Unlock()

	for _, w := range workers {
		s.wg.Add(1)
		go func(wrk *worker) {
			defer s.wg.Done()
			res := wrk.run(supCtx)
			s.mu.Lock()
			s.results[wrk.spec.Name] = res
			s.mu.Unlock()
		}(w)
	}

	return nil
}

// Wait blocks until all workers stop or the context is cancelled, and returns
// the final Result for each registered worker.
func (s *Supervisor) Wait() map[string]*trexec.Result {
	s.wg.Wait()
	s.mu.RLock()
	defer s.mu.RUnlock()

	resCopy := make(map[string]*trexec.Result, len(s.results))
	for k, v := range s.results {
		resCopy[k] = v
	}
	return resCopy
}

// Stop gracefully cancels all supervised workers and waits for them to drain.
func (s *Supervisor) Stop() error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return errors.New("supervisor: not started")
	}
	if s.stopped {
		s.mu.Unlock()
		return nil
	}
	s.stopped = true
	cancel := s.cancel
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	s.wg.Wait()
	return nil
}

// Status returns a snapshot of the current status of all supervised workers.
func (s *Supervisor) Status() map[string]WorkerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	statusMap := make(map[string]WorkerStatus, len(s.workers))
	for name, w := range s.workers {
		statusMap[name] = w.status()
	}
	return statusMap
}
