package trexec

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func TestPipeManagerNormal(t *testing.T) {
	pm := newPipeManager()

	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}

	var buf bytes.Buffer
	pm.startCopy(&buf, pr)

	// Write data and close write end normally
	pw.Write([]byte("hello from pipe\n"))
	pw.Close()

	// Wait for copy to complete
	completed := pm.wait(1 * time.Second)
	if !completed {
		t.Error("pipeManager.wait timed out, expected normal completion")
	}

	if buf.String() != "hello from pipe\n" {
		t.Errorf("buffer = %q, want %q", buf.String(), "hello from pipe\n")
	}

	if errs := pm.copyErrors(); len(errs) != 0 {
		t.Errorf("unexpected copy errors: %v", errs)
	}
}

func TestPipeManagerForceCloseTimeout(t *testing.T) {
	pm := newPipeManager()

	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer pw.Close() // Keep writer open to simulate orphaned grandchild holding pipe

	var buf bytes.Buffer
	pm.startCopy(&buf, pr)

	// Write partial data, but DO NOT close pw
	pw.Write([]byte("partial data\n"))

	// wait with short timeout — should force-close pr and return false
	completed := pm.wait(100 * time.Millisecond)
	if completed {
		t.Error("expected timeout on unclosed pipe, got completed=true")
	}

	if buf.String() != "partial data\n" {
		t.Errorf("buffer = %q, want %q", buf.String(), "partial data\n")
	}
}
