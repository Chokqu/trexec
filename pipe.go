package trexec

import (
	"io"
	"os"
	"sync"
	"time"
)

// pipeManager handles the lifecycle of I/O copy goroutines that transfer
// data between the child process's pipes and the user-provided writers.
//
// It solves the "orphaned pipe writer" problem: if a grandchild process
// inherits the pipe's write end and the direct child exits, the io.Copy
// goroutine will hang because EOF never arrives. pipeManager handles this
// by force-closing the read end after a timeout.
type pipeManager struct {
	mu sync.Mutex

	// pipes holds the parent-side of each pipe (the read ends for stdout/stderr).
	// These are closed to unblock stuck io.Copy goroutines.
	pipes []*os.File

	// wg tracks all running io.Copy goroutines.
	wg sync.WaitGroup

	// errors collects any I/O errors (non-EOF) from copy goroutines.
	errors []error
}

func newPipeManager() *pipeManager {
	return &pipeManager{}
}

// startCopy launches a goroutine that copies from src to dst.
// The src pipe is tracked for forced closure on timeout.
func (pm *pipeManager) startCopy(dst io.Writer, src *os.File) {
	pm.mu.Lock()
	pm.pipes = append(pm.pipes, src)
	pm.mu.Unlock()

	pm.wg.Add(1)
	go func() {
		defer pm.wg.Done()
		_, err := io.Copy(dst, src)
		if err != nil {
			pm.mu.Lock()
			pm.errors = append(pm.errors, err)
			pm.mu.Unlock()
		}
	}()
}

// wait waits for all I/O goroutines to finish, with a timeout.
// If the timeout expires, it force-closes the pipes to unblock stuck goroutines.
// Returns true if all goroutines finished before the timeout.
func (pm *pipeManager) wait(timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		pm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		// Timeout — force-close pipes to unblock io.Copy goroutines.
		pm.forceClose()

		// Wait for goroutines to notice the closed pipes and exit.
		<-done
		return false
	}
}

// forceClose closes all tracked pipes, which will cause any blocked
// io.Copy calls to return with an error, unblocking the goroutines.
func (pm *pipeManager) forceClose() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	for _, p := range pm.pipes {
		p.Close()
	}
}

// copyErrors returns any non-EOF errors from the I/O goroutines.
func (pm *pipeManager) copyErrors() []error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	// Return a copy to avoid races.
	errs := make([]error, len(pm.errors))
	copy(errs, pm.errors)
	return errs
}

// syncWriter wraps an io.Writer with a mutex to synchronize concurrent writes.
// Used by CombinedOutput to safely multiplex stdout and stderr into a single buffer.
type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func newSyncWriter(w io.Writer) *syncWriter {
	return &syncWriter{w: w}
}

func (s *syncWriter) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}
