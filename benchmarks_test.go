package trexec_test

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/Chokqu/trexec"
	"github.com/Chokqu/trexec/supervisor"
)

// BenchmarkStandardExec measures baseline os/exec performance and allocations.
func BenchmarkStandardExec(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(helperBin, "-exit=0")
		if err := cmd.Run(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTrexecRun measures trexec.Run execution latency and heap allocations.
func BenchmarkTrexecRun(b *testing.B) {
	b.ReportAllocs()
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		err := trexec.Run(ctx, helperBin, []string{"-exit=0"})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTrexecRunWithResult measures trexec.RunWithResult execution latency.
func BenchmarkTrexecRunWithResult(b *testing.B) {
	b.ReportAllocs()
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		res, err := trexec.RunWithResult(ctx, helperBin, []string{"-exit=0"})
		if err != nil || !res.Success() {
			b.Fatalf("RunWithResult failed: %v, res: %v", err, res)
		}
	}
}

// BenchmarkTrexecSpawnLatency measures the initialization and Start() latency.
func BenchmarkTrexecSpawnLatency(b *testing.B) {
	b.ReportAllocs()
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		cmd := trexec.CommandContext(ctx, helperBin, []string{"-exit=0"})
		if err := cmd.Start(); err != nil {
			b.Fatal(err)
		}
		_ = cmd.Wait()
	}
}

// BenchmarkTrexecOutputBuffering measures Output() throughput and memory allocations.
func BenchmarkTrexecOutputBuffering(b *testing.B) {
	b.ReportAllocs()
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		out, res, err := trexec.Output(ctx, helperBin, []string{"-stdout=benchmarking_trexec_output_stream"})
		if err != nil || !res.Success() || len(out) == 0 {
			b.Fatalf("Output failed: %v, res: %v, out: %q", err, res, string(out))
		}
	}
}

// BenchmarkTrexecCombinedOutput measures CombinedOutput() thread-safe multiplexing.
func BenchmarkTrexecCombinedOutput(b *testing.B) {
	b.ReportAllocs()
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		out, res, err := trexec.CombinedOutput(ctx, helperBin, []string{"-stdout=out_line", "-stderr=err_line"})
		if err != nil || !res.Success() || len(out) == 0 {
			b.Fatalf("CombinedOutput failed: %v, res: %v, out: %q", err, res, string(out))
		}
	}
}

// BenchmarkTrexecTreeKill measures process tree cancellation and termination latency.
func BenchmarkTrexecTreeKill(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cmd := trexec.CommandContext(ctx, helperBin, []string{"-depth=2", "-sleep=60s"},
			trexec.WithGracePeriod(0), // Immediate force kill for pure kill latency measurement
		)
		if err := cmd.Start(); err != nil {
			b.Fatal(err)
		}
		// Cancel immediately to trigger tree kill
		cancel()
		res := cmd.Wait()
		if !res.Cancelled {
			b.Fatalf("expected tree kill cancellation, got: %s", res)
		}
	}
}

// BenchmarkTrexecSupervisorRestart measures supervisor worker restart performance.
func BenchmarkTrexecSupervisorRestart(b *testing.B) {
	b.ReportAllocs()
	sup := supervisor.New()
	err := sup.Add(supervisor.Spec{
		Name:          "bench-worker",
		Command:       helperBin,
		Args:          []string{"-exit=1"},
		RestartPolicy: supervisor.RestartOnFailure,
		MaxRestarts:   b.N,
		Backoff:       supervisor.NewBackoff(1*time.Millisecond, 1*time.Millisecond, 1.0, 0),
	})
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	if err := sup.Start(ctx); err != nil {
		b.Fatal(err)
	}
	_ = sup.Wait()
}
