# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.0.0] - 2026-08-23

### Added
- **Core Process Tree Engine (`trexec`)**:
  - Full cross-platform process tree lifecycle management across Linux, macOS, and Windows.
  - Zero third-party runtime dependencies in core (`x/sys` on Windows).
  - 6-state formal lifecycle engine (`Created`, `Starting`, `Running`, `Stopping`, `Killing`, `Done`).
  - Two-phase graceful shutdown: configurable grace period with force-kill escalation.
  - Deadlock-free pipe manager (`pipeManager`) with configurable I/O timeouts preventing orphaned grandchild hangs.
  - Output buffering helpers: `Output()` and `CombinedOutput()` with mutex-synchronized buffers.
  - Interactive stdin streaming: `StdinPipe()` returning `io.WriteCloser` with clean EOF propagation.
  - Process tree introspection: `Result.DescendantPIDs []int` and `Result.ProcessesCleaned int` using Win32 `QueryInformationJobObject` and Linux `/proc`.
  - Unified cross-platform signals: `trexec.WithGracefulSignal(trexec.SIGINT)` compiling anywhere without OS build tags.
  - Resource sandboxing: `WithResourceLimits` supporting memory (`MaxMemoryBytes`) and process count limits (`MaxProcesses`).
  - Periodic real-time metric polling: `WithMetricsPollInterval` emitting `TreeMetrics`.

- **Multi-Worker Process Supervisor (`trexec/supervisor`)**:
  - Embedded in-binary worker pool supervision.
  - Configurable restart policies: `RestartAlways`, `RestartOnFailure`, `RestartNever`.
  - Exponential backoff with random jitter (`supervisor.Backoff`).
  - Real-time status reporting: `Supervisor.Status()` and graceful pool drain `Supervisor.Stop()`.

- **Live-Reload DevTools Engine (`trexec/watcher`)**:
  - Pure Go debounced filesystem watcher (`watcher.Watcher`) with modtime tracking and zero external dependencies.
  - Extension filtering (`.go`, `.html`, `.json`) and directory ignore patterns (`.git`, `node_modules`, `vendor`).
  - Process tree hot-reloader (`watcher.Reloader`) with guaranteed previous process tree termination and socket/port release before respawning.

- **Real-Time Telemetry & Observability (`trexec/telemetry`)**:
  - Structured lifecycle events: `telemetry.Event`.
  - Extensible sinks: `MemorySink`, `LoggerSink`, `CallbackSink`.
  - `HookOptions` bridging metric pollers directly into OpenTelemetry and Prometheus collectors.

- **Cobra CLI Middleware (`trexec/cobraexec`)**:
  - `cobraexec.Run`, `cobraexec.Output`, `cobraexec.CombinedOutput`, and `cobraexec.WrapRunE` integrating seamlessly with `github.com/spf13/cobra` commands without creating a hard dependency.

- **Nanosecond Benchmark Suite (`benchmarks_test.go`)**:
  - Baseline comparison against `os/exec.Command`.
  - Tree kill latency measurement (~699 µs to kill multi-level trees).
  - Heap allocation footprint tracking (`b.ReportAllocs()`).

- **Chaos & Fuzz Testing (`fuzz_test.go`)**:
  - Random context cancellation timing and command fuzzing under extreme scheduler interleaving.
