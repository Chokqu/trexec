package telemetry

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/Chokqu/trexec"
)

// LoggerSink writes telemetry events and metrics formatted as plain text to an io.Writer.
type LoggerSink struct {
	mu     sync.Mutex
	writer io.Writer
	prefix string
}

// NewLoggerSink constructs a logger sink writing to the provided destination with an optional prefix.
func NewLoggerSink(w io.Writer, prefix string) *LoggerSink {
	return &LoggerSink{
		writer: w,
		prefix: prefix,
	}
}

// EmitEvent writes the formatted event to the destination writer.
func (l *LoggerSink) EmitEvent(event Event) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.writer == nil {
		return
	}
	pfx := l.prefix
	if pfx != "" {
		pfx += " "
	}
	fmt.Fprintf(l.writer, "%s%s\n", pfx, event.String())
}

// EmitMetrics writes the formatted metric snapshot to the destination writer.
func (l *LoggerSink) EmitMetrics(metrics trexec.TreeMetrics) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.writer == nil {
		return
	}
	pfx := l.prefix
	if pfx != "" {
		pfx += " "
	}
	fmt.Fprintf(l.writer, "%s[%s] Metrics(State=%s, ActiveProcs=%d, MemoryBytes=%d, CPUTime=%s)\n",
		pfx, metrics.Timestamp.Format(time.RFC3339), metrics.State, metrics.ActiveProcesses, metrics.TotalMemoryBytes, metrics.TotalCPUTime)
}

// HookOptions provides standard Options for seamless integration with trexec runners.
func HookOptions(sink Sink, metricsInterval time.Duration) []trexec.Option {
	if sink == nil {
		return nil
	}

	opts := []trexec.Option{
		trexec.WithOnStateChange(func(state trexec.State) {
			sink.EmitEvent(Event{
				Timestamp: time.Now(),
				State:     state,
			})
		}),
	}

	if metricsInterval > 0 {
		opts = append(opts, trexec.WithMetricsPollInterval(metricsInterval, func(m trexec.TreeMetrics) {
			sink.EmitMetrics(m)
		}))
	}

	return opts
}
