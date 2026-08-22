package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Chokqu/trexec"
	"github.com/Chokqu/trexec/cobraexec"
)

// simpleCobraCommand represents a mock or minimal Cobra command execution structure.
type simpleCobraCommand struct {
	ctx    context.Context
	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader
}

func (s *simpleCobraCommand) Context() context.Context { return s.ctx }
func (s *simpleCobraCommand) OutOrStdout() io.Writer   { return s.stdout }
func (s *simpleCobraCommand) ErrOrStderr() io.Writer   { return s.stderr }
func (s *simpleCobraCommand) InOrStdin() io.Reader     { return s.stdin }

func main() {
	fmt.Println("==================================================")
	fmt.Println("🐍 trexec Cobra CLI Middleware Demo")
	fmt.Println("==================================================")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var buf bytes.Buffer
	cmd := &simpleCobraCommand{
		ctx:    ctx,
		stdout: &buf,
		stderr: os.Stderr,
		stdin:  os.Stdin,
	}

	// WrapRunE executes a command tree cleanly within Cobra's execution model
	runE := cobraexec.WrapRunE("go", []string{"env", "GOVERSION"},
		trexec.WithGracePeriod(1*time.Second),
	)

	fmt.Println("🚀 Executing command via cobraexec.WrapRunE...")
	if err := runE(cmd, []string{"--demo-arg"}); err != nil {
		fmt.Printf("Command returned error: %v\n", err)
	} else {
		fmt.Printf("✅ Success! Output captured via Cobra streams:\n%s", buf.String())
	}
}
