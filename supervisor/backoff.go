package supervisor

import (
	"math/rand"
	"time"
)

// Backoff calculates the exponential backoff duration for a given attempt.
type Backoff struct {
	// Min is the minimum initial backoff duration (default: 100ms).
	Min time.Duration

	// Max is the maximum backoff duration cap (default: 10s).
	Max time.Duration

	// Factor is the multiplication factor per attempt (default: 2.0).
	Factor float64

	// Jitter enables random duration variation between 0 and Jitter * duration (default: 0.1).
	Jitter float64
}

// DefaultBackoff returns a Backoff configuration suitable for production services.
func DefaultBackoff() *Backoff {
	return &Backoff{
		Min:    100 * time.Millisecond,
		Max:    10 * time.Second,
		Factor: 2.0,
		Jitter: 0.1,
	}
}

// Duration calculates the backoff duration for the specified attempt index (1-based).
func (b *Backoff) Duration(attempt int) time.Duration {
	if attempt <= 1 {
		return b.Min
	}

	min := b.Min
	if min <= 0 {
		min = 100 * time.Millisecond
	}

	max := b.Max
	if max <= 0 {
		max = 10 * time.Second
	}

	factor := b.Factor
	if factor <= 1.0 {
		factor = 2.0
	}

	// Calculate exponential growth: min * factor^(attempt - 1)
	dur := float64(min)
	for i := 1; i < attempt; i++ {
		dur *= factor
		if dur >= float64(max) {
			dur = float64(max)
			break
		}
	}

	// Apply jitter if configured
	if b.Jitter > 0 {
		jitterRange := dur * b.Jitter
		// rand.Float64() returns [0.0, 1.0)
		jitterOffset := (rand.Float64()*2 - 1) * jitterRange
		dur += jitterOffset
	}

	result := time.Duration(dur)
	if result < min {
		return min
	}
	if result > max {
		return max
	}
	return result
}
