package trexec

import "testing"

func TestSignalString(t *testing.T) {
	tests := []struct {
		sig  Signal
		want string
	}{
		{SIGTERM, "SIGTERM"},
		{SIGINT, "SIGINT"},
		{SIGHUP, "SIGHUP"},
		{SIGKILL, "SIGKILL"},
		{Signal(99), "SIGNAL(99)"},
	}

	for _, tt := range tests {
		if got := tt.sig.String(); got != tt.want {
			t.Errorf("Signal(%d).String() = %q, want %q", tt.sig, got, tt.want)
		}
	}
}

func TestWithGracefulSignalOption(t *testing.T) {
	opts := applyOptions([]Option{
		WithGracefulSignal(SIGINT),
	})
	if opts.gracefulSignal != SIGINT {
		t.Errorf("gracefulSignal = %s, want SIGINT", opts.gracefulSignal)
	}
}
