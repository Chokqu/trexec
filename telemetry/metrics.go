package telemetry

import (
	"fmt"
	"sync"
	"time"

	"github.com/Chokqu/trexec"
)

// Event represents a structured lifecycle or execution event emitted by a process tree.
type Event struct {
	// Timestamp is the moment the event occurred.
	Timestamp time.Time

	// State is the lifecycle state associated with this event.
	State trexec.State

	// PID is the direct process ID (or 0 if not yet started).
	PID int

	// ActiveProcesses is the count of running descendant processes at event time.
	ActiveProcesses int

	// ExitCode is the exit status (valid when State is StateDone).
	ExitCode int

	// Duration is the total execution time (valid when State is StateDone).
	Duration time.Duration

	// Error is the error associated with the event, if any.
	Error error

	// Message is an optional human-readable message describing the event.
	Message string
}

// String formats the event into a readable line.
func (e Event) String() string {
	if e.State == trexec.StateDone {
		return fmt.Sprintf("[%s] Event(State=%s, PID=%d, ExitCode=%d, Duration=%s, Error=%v)",
			e.Timestamp.Format(time.RFC3339), e.State, e.PID, e.ExitCode, e.Duration, e.Error)
	}
	return fmt.Sprintf("[%s] Event(State=%s, PID=%d, ActiveProcs=%d, Message=%q)",
		e.Timestamp.Format(time.RFC3339), e.State, e.PID, e.ActiveProcesses, e.Message)
}

// Sink defines the interface for consuming process tree telemetry events and metric snapshots.
type Sink interface {
	// EmitEvent records a lifecycle event.
	EmitEvent(event Event)

	// EmitMetrics records a periodic metric snapshot.
	EmitMetrics(metrics trexec.TreeMetrics)
}

// CallbackSink adapts user-provided functions into a Sink.
type CallbackSink struct {
	OnEvent   func(Event)
	OnMetrics func(trexec.TreeMetrics)
}

// EmitEvent calls OnEvent if non-nil.
func (s *CallbackSink) EmitEvent(event Event) {
	if s.OnEvent != nil {
		s.OnEvent(event)
	}
}

// EmitMetrics calls OnMetrics if non-nil.
func (s *CallbackSink) EmitMetrics(metrics trexec.TreeMetrics) {
	if s.OnMetrics != nil {
		s.OnMetrics(metrics)
	}
}

// MemorySink captures events and metrics in thread-safe slices for testing and aggregation.
type MemorySink struct {
	mu      sync.RWMutex
	events  []Event
	metrics []trexec.TreeMetrics
}

// NewMemorySink creates a new in-memory telemetry sink.
func NewMemorySink() *MemorySink {
	return &MemorySink{}
}

// EmitEvent appends an event to the memory sink.
func (m *MemorySink) EmitEvent(event Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

// EmitMetrics appends a metrics snapshot to the memory sink.
func (m *MemorySink) EmitMetrics(metrics trexec.TreeMetrics) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics = append(m.metrics, metrics)
}

// Events returns a copy of all captured events.
func (m *MemorySink) Events() []Event {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cp := make([]Event, len(m.events))
	copy(cp, m.events)
	return cp
}

// Metrics returns a copy of all captured metric snapshots.
func (m *MemorySink) Metrics() []trexec.TreeMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cp := make([]trexec.TreeMetrics, len(m.metrics))
	copy(cp, m.metrics)
	return cp
}

// Reset clears all recorded events and metrics.
func (m *MemorySink) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = nil
	m.metrics = nil
}
