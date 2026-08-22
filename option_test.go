package trexec

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func TestOptionDefaults(t *testing.T) {
	o := defaultOptions()

	if o.gracePeriod != 5*time.Second {
		t.Errorf("default gracePeriod = %v, want 5s", o.gracePeriod)
	}
	if o.ioTimeout != 2*time.Second {
		t.Errorf("default ioTimeout = %v, want 2s", o.ioTimeout)
	}
	if o.stdout != nil {
		t.Error("default stdout should be nil")
	}
	if o.stderr != nil {
		t.Error("default stderr should be nil")
	}
	if o.stdin != nil {
		t.Error("default stdin should be nil")
	}
	if o.dir != "" {
		t.Errorf("default dir = %q, want empty", o.dir)
	}
	if o.env != nil {
		t.Error("default env should be nil")
	}
}

func TestOptionOverrides(t *testing.T) {
	opts := applyOptions([]Option{
		WithGracePeriod(10 * time.Second),
		WithIOTimeout(5 * time.Second),
		WithStdout(os.Stdout),
		WithStderr(os.Stderr),
		WithDir("/tmp"),
		WithEnv([]string{"FOO=bar"}),
	})

	if opts.gracePeriod != 10*time.Second {
		t.Errorf("gracePeriod = %v, want 10s", opts.gracePeriod)
	}
	if opts.ioTimeout != 5*time.Second {
		t.Errorf("ioTimeout = %v, want 5s", opts.ioTimeout)
	}
	if opts.stdout != os.Stdout {
		t.Error("stdout not set correctly")
	}
	if opts.stderr != os.Stderr {
		t.Error("stderr not set correctly")
	}
	if opts.dir != "/tmp" {
		t.Errorf("dir = %q, want /tmp", opts.dir)
	}
	if len(opts.env) != 1 || opts.env[0] != "FOO=bar" {
		t.Errorf("env = %v, want [FOO=bar]", opts.env)
	}
}

func TestWithZeroGracePeriod(t *testing.T) {
	opts := applyOptions([]Option{
		WithGracePeriod(0),
	})
	if opts.gracePeriod != 0 {
		t.Errorf("gracePeriod = %v, want 0", opts.gracePeriod)
	}
}

func TestWithSysProcAttrOption(t *testing.T) {
	custom := &syscall.SysProcAttr{}
	opts := applyOptions([]Option{
		WithSysProcAttr(custom),
	})
	if opts.sysProcAttr != custom {
		t.Error("sysProcAttr was not set in options")
	}
}
