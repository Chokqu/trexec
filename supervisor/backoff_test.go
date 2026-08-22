package supervisor_test

import (
	"testing"
	"time"

	"github.com/Chokqu/trexec/supervisor"
)

func TestDefaultBackoff(t *testing.T) {
	b := supervisor.DefaultBackoff()
	if b.Min != 100*time.Millisecond {
		t.Errorf("expected Min = 100ms, got %v", b.Min)
	}
	if b.Max != 10*time.Second {
		t.Errorf("expected Max = 10s, got %v", b.Max)
	}
	if b.Factor != 2.0 {
		t.Errorf("expected Factor = 2.0, got %v", b.Factor)
	}
}

func TestBackoffExponentialGrowth(t *testing.T) {
	b := &supervisor.Backoff{
		Min:    100 * time.Millisecond,
		Max:    10 * time.Second,
		Factor: 2.0,
		Jitter: 0.0, // Disable jitter for deterministic tests
	}

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{attempt: 1, want: 100 * time.Millisecond},
		{attempt: 2, want: 200 * time.Millisecond},
		{attempt: 3, want: 400 * time.Millisecond},
		{attempt: 4, want: 800 * time.Millisecond},
		{attempt: 5, want: 1600 * time.Millisecond},
		{attempt: 10, want: 10 * time.Second}, // Capped at Max
	}

	for _, tt := range tests {
		got := b.Duration(tt.attempt)
		if got != tt.want {
			t.Errorf("attempt %d: got %v, want %v", tt.attempt, got, tt.want)
		}
	}
}

func TestBackoffJitterBounds(t *testing.T) {
	b := &supervisor.Backoff{
		Min:    100 * time.Millisecond,
		Max:    1 * time.Second,
		Factor: 2.0,
		Jitter: 0.2, // +/- 20%
	}

	for i := 1; i <= 20; i++ {
		dur := b.Duration(3) // base is 400ms, with +/- 20% jitter it should be between 320ms and 480ms
		if dur < 320*time.Millisecond || dur > 480*time.Millisecond {
			t.Errorf("duration out of jitter bounds: %v", dur)
		}
	}
}
