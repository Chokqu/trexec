package cobraexec_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Chokqu/trexec"
	"github.com/Chokqu/trexec/cobraexec"
)

var helperBin string

func TestMain(m *testing.M) {
	tempDir, err := os.MkdirTemp("", "trexec-cobraexec-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	binName := "testhelper"
	if os.Getenv("GOOS") == "windows" || filepath.Ext(os.Args[0]) == ".exe" {
		binName += ".exe"
	}
	helperBin = filepath.Join(tempDir, binName)

	buildCmd := exec.Command("go", "build", "-o", helperBin, "../testdata/helper")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build testhelper: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

type mockCobraCommand struct {
	ctx    context.Context
	stdout *bytes.Buffer
	stderr *bytes.Buffer
	stdin  *bytes.Buffer
}

func newMockCobraCommand(ctx context.Context) *mockCobraCommand {
	return &mockCobraCommand{
		ctx:    ctx,
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		stdin:  &bytes.Buffer{},
	}
}

func (m *mockCobraCommand) Context() context.Context { return m.ctx }
func (m *mockCobraCommand) OutOrStdout() io.Writer   { return m.stdout }
func (m *mockCobraCommand) ErrOrStderr() io.Writer   { return m.stderr }
func (m *mockCobraCommand) InOrStdin() io.Reader     { return m.stdin }

func TestCobraExecRun(t *testing.T) {
	mockCmd := newMockCobraCommand(context.Background())

	res, err := cobraexec.Run(mockCmd, helperBin, []string{"-stdout=cobra_hello"})
	if err != nil || !res.Success() {
		t.Fatalf("cobraexec.Run failed: %v, res: %v", err, res)
	}

	got := strings.TrimSpace(mockCmd.stdout.String())
	if got != "cobra_hello" {
		t.Errorf("stdout = %q, want %q", got, "cobra_hello")
	}
}

func TestCobraExecOutput(t *testing.T) {
	mockCmd := newMockCobraCommand(context.Background())

	out, res, err := cobraexec.Output(mockCmd, helperBin, []string{"-stdout=cobra_output"})
	if err != nil || !res.Success() {
		t.Fatalf("cobraexec.Output failed: %v, res: %v", err, res)
	}

	got := strings.TrimSpace(string(out))
	if got != "cobra_output" {
		t.Errorf("output = %q, want %q", got, "cobra_output")
	}
}

func TestCobraExecCombinedOutput(t *testing.T) {
	mockCmd := newMockCobraCommand(context.Background())

	out, res, err := cobraexec.CombinedOutput(mockCmd, helperBin, []string{"-stdout=out1", "-stderr=err1"})
	if err != nil || !res.Success() {
		t.Fatalf("cobraexec.CombinedOutput failed: %v, res: %v", err, res)
	}

	got := string(out)
	if !strings.Contains(got, "out1") || !strings.Contains(got, "err1") {
		t.Errorf("combined output = %q, want both out1 and err1", got)
	}
}

func TestCobraExecWrapRunE(t *testing.T) {
	runE := cobraexec.WrapRunE(helperBin, []string{"-stdout=success_test"})
	mockCmd := newMockCobraCommand(context.Background())

	err := runE(mockCmd, []string{"cli-arg"})
	if err != nil {
		t.Fatalf("WrapRunE returned error: %v", err)
	}

	// Test graceful cancellation handling in WrapRunE
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel before or immediately

	runECancel := cobraexec.WrapRunE(helperBin, []string{"-graceful"}, trexec.WithGracePeriod(1*time.Second))
	mockCancelCmd := newMockCobraCommand(ctx)
	errCancel := runECancel(mockCancelCmd, nil)
	// Cancelled graceful exit should return nil error
	if errCancel != nil {
		t.Errorf("expected nil error on graceful exit, got: %v", errCancel)
	}
}
