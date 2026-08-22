# trexec

<p align="center">
  <img src="https://raw.githubusercontent.com/Chokqu/trexec/master/doc/logo.png" alt="trexec logo" width="120" onerror="this.style.display='none'"/>
  <br>
  <strong>Own the workload, not just the process.</strong><br>
  Cross-platform process tree lifecycle manager, supervisor, and live-reload engine for Go.<br>
  <em>Graceful shutdown, two-phase force-kill escalation, interactive I/O streaming, telemetry, and zero orphan processes on Linux, macOS, and Windows.</em>
</p>

<p align="center">
  <a href="https://github.com/Chokqu/trexec/actions/workflows/ci.yml"><img src="https://github.com/Chokqu/trexec/actions/workflows/ci.yml/badge.svg" alt="CI Status"></a>
  <a href="https://pkg.go.dev/github.com/Chokqu/trexec"><img src="https://pkg.go.dev/badge/github.com/Chokqu/trexec.svg" alt="Go Reference"></a>
  <a href="https://goreportcard.com/report/github.com/Chokqu/trexec"><img src="https://goreportcard.com/badge/github.com/Chokqu/trexec" alt="Go Report Card"></a>
  <a href="https://github.com/Chokqu/trexec/releases"><img src="https://img.shields.io/github/v/release/Chokqu/trexec?color=blue&label=version" alt="Latest Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-green.svg" alt="License: MIT"></a>
</p>

---

## 📑 Table of Contents

1. [Why trexec? (The Problem Space)](#-why-trexec-the-problem-space)
2. [Key Value Proposition & Architecture](#-key-value-proposition--architecture)
3. [Installation & Compatibility](#-installation--compatibility)
4. [Complete API Reference Manual](#-complete-api-reference-manual)
   - 4.1 [Core Package (`trexec`)](#41-core-package-trexec)
     - [Constructors & Entrypoints](#constructors--entrypoints)
     - [Runner Methods](#runner-methods)
     - [Functional Options (`Option`)](#functional-options-option)
     - [Data Types & Structs](#data-types--structs)
   - 4.2 [Supervisor Package (`trexec/supervisor`)](#42-supervisor-package-trexecsupervisor)
   - 4.3 [Watcher & Live-Reload Package (`trexec/watcher`)](#43-watcher--live-reload-package-trexecwatcher)
   - 4.4 [Telemetry & Observability Package (`trexec/telemetry`)](#44-telemetry--observability-package-trexectelemetry)
   - 4.5 [Cobra CLI Middleware (`trexec/cobraexec`)](#45-cobra-cli-middleware-trexeccobraexec)
5. [Operating System Kernel Primitives](#-operating-system-kernel-primitives)
6. [Performance Benchmarks](#-performance-benchmarks)
7. [Production Recipes & Patterns](#-production-recipes--patterns)
8. [License](#-license)

---

## 🎯 Why trexec? (The Problem Space)

Go's standard `os/exec` only manages a **single root process PID**. In modern software, commands routinely spawn deep process trees:
- `npm run dev` &rarr; `sh` &rarr; `vite` &rarr; `esbuild` &rarr; worker threads
- `docker-compose up` &rarr; multiple container daemons
- `python main.py` &rarr; multiprocessing pools & background workers

### The `os/exec` Problem
When a `context.Context` is cancelled with `os/exec.CommandContext`:
1. `os/exec` sends `SIGKILL` only to the root process (`npm`).
2. The child and grandchild processes (`vite`, `esbuild`, workers) are **orphaned**.
3. Ports remain bound, file locks remain held, and the next run fails with `EADDRINUSE: address already in use`.
4. If grandchildren hold open `stdout`/`stderr` file descriptors, `cmd.Wait()` **hangs indefinitely** waiting for EOF.

```
os/exec.CommandContext (Standard Library)
your-app ──► npm (KILLED)
              └── vite (ORPHANED ⚠️)
                  └── esbuild (ORPHANED ⚠️ — Port 3000 stays locked!)

trexec.CommandContext (Cross-Platform Solution)
your-app ──► [ Process Group / Windows Job Object ]
              ├── npm       (Gracefully stopped -> Cleaned ✅)
              ├── vite      (Gracefully stopped -> Cleaned ✅)
              └── esbuild   (Gracefully stopped -> Cleaned ✅ — Port released!)
```

`trexec` solves this natively across **Linux, macOS, and Windows** with **zero third-party runtime dependencies**.

---

## ⚡ Key Value Proposition & Architecture

- **🌲 Complete Tree Ownership**: Treats the entire descendant process tree as an ownable, cleanable workload.
- **⏱️ Two-Phase Graceful Shutdown**: Polite termination signal (`SIGTERM`, `SIGINT`, `Ctrl+Break`) &rarr; configurable grace period &rarr; kernel force-kill (`SIGKILL`, `TerminateJobObject`).
- **🛡️ Pipe Deadlock Timeout Guard**: Built-in I/O timeouts unblock stuck `io.Copy` goroutines if orphaned writers keep pipes open.
- **🔀 Synchronized Stream Buffering**: `Output()` and `CombinedOutput()` return captured byte streams alongside rich outcome metadata.
- **💬 Interactive Stdin Streaming**: `StdinPipe()` provides dynamic `io.WriteCloser` streaming with clean EOF signaling.
- **🔍 Process Tree Introspection**: Returns `Result.DescendantPIDs []int` and `Result.ProcessesCleaned int` using Windows Job Object query APIs and Linux `/proc`.
- **🔄 Pure Go Live-Reload Engine**: `trexec/watcher` debounces filesystem events, drains old workloads completely, and guarantees socket release on restart.
- **👮 Embedded Multi-Worker Supervisor**: `trexec/supervisor` provides in-process worker pool management, restart policies (`RestartAlways`, `RestartOnFailure`), and exponential backoff with jitter.
- **📊 Real-Time Telemetry & Metrics**: `trexec/telemetry` & `WithMetricsPollInterval` stream live process counts, memory usage, CPU time, and state transitions to OpenTelemetry/Prometheus.
- **🐍 Cobra CLI Integration**: `trexec/cobraexec` wraps `spf13/cobra` commands with zero-leak process tree lifecycle supervision.
- **🌐 Unified Cross-Platform Signals**: Write `trexec.WithGracefulSignal(trexec.SIGINT)` once; compiles and executes identically on Linux, macOS, and Windows with zero OS-specific imports.
- **🚦 6-State Formal Lifecycle**: Track execution state (`Created` &rarr; `Starting` &rarr; `Running` &rarr; `Stopping` &rarr; `Killing` &rarr; `Done`) via `WithOnStateChange`.
- **📦 Zero Third-Party Runtime Dependencies**: Pure Go standard library on Unix; official `golang.org/x/sys` on Windows.

---

## 📦 Installation & Compatibility

```bash
go get github.com/Chokqu/trexec
```

Compatible with **Go 1.21, 1.22, 1.23, 1.24, 1.25, and 1.26+** on:
- **Linux** (x86_64, ARM64, ARM)
- **macOS** (Apple Silicon ARM64, Intel x86_64)
- **Windows** (x86_64, ARM64)

---

## 📖 Complete API Reference Manual

---

### 4.1 Core Package (`trexec`)

#### Constructors & Entrypoints

##### `CommandContext`
```go
func CommandContext(ctx context.Context, name string, args ...any) *Runner
```
Creates a new `Runner` bound to the provided context.
- **Parameters**:
  - `ctx`: Parent context governing command lifecycle.
  - `name`: Binary name or executable path.
  - `args`: Variadic slice accepting command-line arguments (strings or `[]string`) and `Option` functional configurations in any order.
- **Returns**: `*Runner` ready for `Start()`, `Wait()`, `Run()`, `Output()`, `CombinedOutput()`, or `StdinPipe()`.

##### `Run`
```go
func Run(ctx context.Context, name string, args ...any) error
```
Convenience helper that initializes a `Runner`, executes the command, blocks until natural completion or cancellation, and returns an `error` if setup fails or exit code is non-zero.

##### `RunWithResult`
```go
func RunWithResult(ctx context.Context, name string, args ...any) (*Result, error)
```
Executes a command and returns the full structured `*Result`.
- **Returns**: `(*Result, error)`. The returned `error` is non-nil only for startup/fork failures (binary not found, permission denied). Command crashes and cancellations are stored inside `Result.ExitCode` and `Result.Error`.

##### `Output`
```go
func Output(ctx context.Context, name string, args []string, opts ...Option) ([]byte, *Result, error)
```
Direct package-level helper that runs the command and captures its standard output.

##### `CombinedOutput`
```go
func CombinedOutput(ctx context.Context, name string, args []string, opts ...Option) ([]byte, *Result, error)
```
Direct package-level helper that runs the command and captures thread-safe interleaved standard output and standard error.

---

#### Runner Methods

##### `(*Runner).Start() error`
Starts the command process tree. Creates process group / Job Object, launches I/O goroutines, starts context monitor, and returns immediately without blocking.

##### `(*Runner).Wait() *Result`
Blocks until command completion or cancellation cleanup finishes. Executes two-phase graceful shutdown if context was cancelled, cleans up pipes with `WithIOTimeout`, closes OS handles, and returns `*Result`.

##### `(*Runner).Run() (*Result, error)`
Calls `Start()` followed by `Wait()`.

##### `(*Runner).Output() ([]byte, *Result, error)`
Runs the command, captures `stdout` into an internal buffer, and returns captured bytes along with `*Result`. Returns error if `WithStdout` was already configured.

##### `(*Runner).CombinedOutput() ([]byte, *Result, error)`
Runs the command, captures `stdout` and `stderr` through a mutex-synchronized writer, and returns combined bytes along with `*Result`. Returns error if `WithStdout` or `WithStderr` was already configured.

##### `(*Runner).StdinPipe() (io.WriteCloser, error)`
Returns a pipe write end connected to child stdin. Must be called before `Start()`. Closing the writer sends `EOF` to child. Automatically closed on `Wait()` if left open.

##### `(*Runner).PID() int`
Returns the process ID of the direct child process, or 0 if not running.

---

#### Functional Options (`Option`)

| Option | Signature | Default | Description |
|---|---|---|---|
| `WithGracePeriod` | `func(d time.Duration) Option` | `5s` | Duration to wait after graceful signal before force-killing. Set to `0` for immediate force-kill. |
| `WithGracefulSignal` | `func(sig Signal) Option` | `SIGTERM` | Polite signal sent on cancellation (`SIGINT`, `SIGTERM`, `SIGHUP`). |
| `WithIOTimeout` | `func(d time.Duration) Option` | `2s` | Timeout to wait for I/O goroutines after child exits before force-closing pipes. |
| `WithStdout` | `func(w io.Writer) Option` | `nil` (discard) | Destination writer for command standard output. |
| `WithStderr` | `func(w io.Writer) Option` | `nil` (discard) | Destination writer for command standard error. |
| `WithStdin` | `func(r io.Reader) Option` | `nil` | Source reader for command standard input. |
| `WithDir` | `func(dir string) Option` | `""` (inherit) | Working directory for the process tree. |
| `WithEnv` | `func(env []string) Option` | `nil` (inherit) | Environment variables formatted as `["KEY=VALUE"]`. |
| `WithExtraFiles` | `func(files []*os.File) Option`| `nil` | Additional open file descriptors passed to child (fd 3, 4, ...). |
| `WithSysProcAttr` | `func(attr *syscall.SysProcAttr) Option` | `nil` | Merges user platform attributes with required process group attributes. |
| `WithOnStateChange` | `func(fn func(State)) Option` | `nil` | Asynchronous callback invoked on every lifecycle state transition. |
| `WithResourceLimits` | `func(limits ResourceLimits) Option` | `nil` | Kernel-enforced memory and process count boundaries. |
| `WithMetricsPollInterval` | `func(interval time.Duration, cb func(TreeMetrics)) Option` | `nil` | Periodic telemetry poller streaming real-time RAM, CPU, PID metrics. |

---

#### Data Types & Structs

##### `Result`
```go
type Result struct {
    ExitCode             int           // Process exit code (-1 if killed)
    Cancelled            bool          // True if stopped due to context cancellation
    GracefullyTerminated bool          // True if stopped within grace period
    ForceKilled          bool          // True if killed after grace period expired
    Duration             time.Duration // Total wall-clock execution duration
    ProcessesCleaned     int           // Count of descendant processes terminated
    DescendantPIDs       []int         // Exact slice of tracked descendant PIDs
    Error                error         // Underlying ExitError or execution error
}

func (r *Result) Success() bool
func (r *Result) String() string
```

##### `TreeMetrics`
```go
type TreeMetrics struct {
    Timestamp        time.Time     // Snapshot timestamp
    ActiveProcesses  int           // Count of live descendant processes
    TotalMemoryBytes int64         // Cumulative committed RAM (if available)
    TotalCPUTime     time.Duration // Cumulative CPU execution time (if available)
    State            State         // Current lifecycle state
}
```

##### `ResourceLimits`
```go
type ResourceLimits struct {
    MaxMemoryBytes int64 // Maximum committed memory across process tree
    MaxProcesses   int   // Maximum simultaneously active processes allowed
}
```

##### `Signal`
```go
type Signal int

const (
    SIGTERM Signal = iota // Unix: kill(-pgid, SIGTERM), Windows: CTRL_BREAK_EVENT
    SIGINT                // Unix: kill(-pgid, SIGINT), Windows: CTRL_C_EVENT
    SIGHUP                // Unix: kill(-pgid, SIGHUP), Windows: CTRL_BREAK_EVENT
    SIGKILL               // Unix: kill(-pgid, SIGKILL), Windows: TerminateJobObject
)
```

##### `State`
```go
type State int

const (
    StateCreated  State = iota // Created, not started
    StateStarting              // Configuring group, calling fork/exec
    StateRunning               // Process alive, I/O active, context monitored
    StateStopping              // Graceful termination signal active
    StateKilling               // Force kill in progress
    StateDone                  // Cleaned up, handles closed, result ready
)
```

##### `ExitError`
```go
type ExitError struct {
    ExitCode  int
    Signal    string
    Stderr    []byte
    Cancelled bool
}

func (e *ExitError) Error() string
func (e *ExitError) Unwrap() error // Returns context.Canceled if Cancelled == true
```

---

### 4.2 Supervisor Package (`trexec/supervisor`)

The `supervisor` package provides in-process multi-worker supervision with exponential backoff and jitter.

```go
import "github.com/Chokqu/trexec/supervisor"
```

#### Types & API

##### `Supervisor`
```go
type Supervisor struct {}

func New() *Supervisor
func (s *Supervisor) Add(spec Spec) error
func (s *Supervisor) Start(ctx context.Context) error
func (s *Supervisor) Wait() map[string]*trexec.Result
func (s *Supervisor) Stop() error
func (s *Supervisor) Status() map[string]WorkerStatus
```

##### `Spec`
```go
type Spec struct {
    Name          string
    Command       string
    Args          []string
    RestartPolicy RestartPolicy
    MaxRestarts   int
    Backoff       *Backoff
    GracePeriod   time.Duration
    GracefulSignal trexec.Signal
    Stdout        io.Writer
    Stderr        io.Writer
    Dir           string
    Env           []string
}
```

##### `RestartPolicy`
```go
type RestartPolicy int

const (
    RestartNever RestartPolicy = iota
    RestartAlways
    RestartOnFailure
)
```

##### `Backoff`
```go
type Backoff struct {
    Min    time.Duration // Default: 100ms
    Max    time.Duration // Default: 10s
    Factor float64       // Default: 2.0
    Jitter float64       // Default: 0.1 (±10%)
}

func DefaultBackoff() *Backoff
func NewBackoff(min, max time.Duration, factor, jitter float64) *Backoff
func (b *Backoff) Duration(attempt int) time.Duration
```

---

### 4.3 Watcher & Live-Reload Package (`trexec/watcher`)

The `watcher` package provides a pure Go debounced filesystem change detector and hot-reloading devserver.

```go
import "github.com/Chokqu/trexec/watcher"
```

#### Types & API

##### `Watcher`
```go
type Watcher struct {}

func New(cfg Config) *Watcher
func (w *Watcher) Start(ctx context.Context) (<-chan []string, error)
```

##### `Config`
```go
type Config struct {
    Paths        []string      // Root directories/files to watch
    Extensions   []string      // Extensions to match (e.g. [".go", ".html"])
    IgnoredNames []string      // Directories/files to ignore ([".git", "vendor"])
    Debounce     time.Duration // Coalesce window (default: 150ms)
    PollInterval time.Duration // Filesystem scan period (default: 200ms)
}

func DefaultConfig(paths ...string) Config
```

##### `Reloader`
```go
type Reloader struct {}

func NewReloader(cfg ReloaderConfig) *Reloader
func (r *Reloader) Run(ctx context.Context) error
```

##### `ReloaderConfig`
```go
type ReloaderConfig struct {
    Watcher        Config
    Command        string
    Args           []string
    Dir            string
    Env            []string
    Stdout         io.Writer
    Stderr         io.Writer
    GracePeriod    time.Duration
    GracefulSignal trexec.Signal
    OnRestart      func(attempt int, changedFiles []string)
}
```

---

### 4.4 Telemetry & Observability Package (`trexec/telemetry`)

The `telemetry` package routes structured lifecycle events and periodic metrics to observability stacks.

```go
import "github.com/Chokqu/trexec/telemetry"
```

#### Types & API

##### `Event`
```go
type Event struct {
    Timestamp       time.Time
    State           trexec.State
    PID             int
    ActiveProcesses int
    ExitCode        int
    Duration        time.Duration
    Error           error
    Message         string
}
```

##### `Sink` Interface
```go
type Sink interface {
    EmitEvent(event Event)
    EmitMetrics(metrics trexec.TreeMetrics)
}
```

##### Built-in Sinks
- `MemorySink`: Thread-safe slice buffer for inspection and tests (`NewMemorySink()`, `Events()`, `Metrics()`, `Reset()`).
- `LoggerSink`: Formats events and metrics to text output (`NewLoggerSink(w io.Writer, prefix string)`).
- `CallbackSink`: Direct closure adapter (`OnEvent`, `OnMetrics`).
- `HookOptions`: Convenience bridge helper:
  ```go
  func HookOptions(sink Sink, metricsInterval time.Duration) []trexec.Option
  ```

---

### 4.5 Cobra CLI Middleware (`trexec/cobraexec`)

The `cobraexec` package provides middleware for `github.com/spf13/cobra` without forcing an external dependency.

```go
import "github.com/Chokqu/trexec/cobraexec"
```

#### Types & API

##### `CobraLikeCommand` Interface
```go
type CobraLikeCommand interface {
    Context() context.Context
    OutOrStdout() io.Writer
    ErrOrStderr() io.Writer
    InOrStdin() io.Reader
}
```

##### Middleware Functions
```go
func Run(cmd CobraLikeCommand, name string, args []string, opts ...trexec.Option) (*trexec.Result, error)
func Output(cmd CobraLikeCommand, name string, args []string, opts ...trexec.Option) ([]byte, *trexec.Result, error)
func CombinedOutput(cmd CobraLikeCommand, name string, args []string, opts ...trexec.Option) ([]byte, *trexec.Result, error)
func WrapRunE(name string, args []string, opts ...trexec.Option) func(cmd CobraLikeCommand, cliArgs []string) error
```

---

## 🔬 Operating System Kernel Primitives

### 🐧 Linux & Unix Backend
1. **Process Groups (`setpgid`)**: Child process becomes the leader of its own process group (`setpgid(0, 0)`). All descendants inherit `PGID = child.PID`.
2. **Negative PID Broadcasting**: Signals are delivered to `-pgid` (`syscall.Kill(-pgid, sig)`), delivering signals simultaneously to all child and grandchild processes.
3. **Linux `PR_SET_PDEATHSIG`**: Configures kernel death signals so child trees terminate automatically if the parent process crashes.
4. **`/proc` Introspection**: Scans `/proc/*/stat` (field 5 `pgrp == targetPGID`) to enumerate live descendant PIDs.

### 🪟 Windows Backend
1. **Win32 Job Objects**: Anonymous Job Object configured with `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE | JOB_OBJECT_LIMIT_SILENT_BREAKAWAY_OK`.
2. **Suspended Process Assignment**: Spawns the root process with `CREATE_SUSPENDED`, assigns handle via `AssignProcessToJobObject`, and unwinds thread suspension counts using `CreateToolhelp32Snapshot(TH32CS_SNAPTHREAD, 0)`.
3. **Graceful Console Events**: Sends `CTRL_BREAK_EVENT` or `CTRL_C_EVENT` via `GenerateConsoleCtrlEvent`.
4. **Fallback Tree Termination**: If running in restricted sandboxes where Job Object creation is blocked, transparently degrades to recursive process tree termination (`taskkill /F /T /PID`).

---

## ⚡ Performance Benchmarks

Measured on Apple Silicon (M-series / ARM64, 8 threads, Go 1.26):

| Benchmark Target | Latency | Memory Allocations | Description |
|---|---|---|---|
| `BenchmarkStandardExec` (Baseline) | `2.82 ms/op` | `10,368 B/op (30 allocs)` | Raw `os/exec.Command().Run()` |
| `BenchmarkTrexecRun` | `3.17 ms/op` | `11,584 B/op (47 allocs)` | `trexec.Run()` with tree ownership |
| `BenchmarkTrexecSpawnLatency` | `2.93 ms/op` | `11,558 B/op (47 allocs)` | Group creation, fork/exec, initialization |
| `BenchmarkTrexecTreeKill` | **`699 µs/op`** | `12,160 B/op (55 allocs)` | Immediate force-kill of multi-level tree |
| `BenchmarkTrexecOutputBuffering` | `3.09 ms/op` | `13,512 B/op (52 allocs)` | Capturing stdout with rich Result struct |
| `BenchmarkTrexecCombinedOutput` | `2.91 ms/op` | `77,839 B/op (56 allocs)` | Synchronized multi-stream capture |
| `BenchmarkTrexecSupervisorRestart` | `4.38 ms/op` | `12,453 B/op (57 allocs)` | Worker termination & supervised respawn |

---

## 💡 Production Recipes & Patterns

### 1. Dev Server with Graceful Shutdown
```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
defer stop()

result, err := trexec.RunWithResult(ctx, "npm", []string{"run", "dev"},
    trexec.WithGracePeriod(5*time.Second),
    trexec.WithGracefulSignal(trexec.SIGINT),
    trexec.WithStdout(os.Stdout),
    trexec.WithStderr(os.Stderr),
)
if err != nil {
    log.Fatal(err)
}
if result.Cancelled && result.GracefullyTerminated {
    log.Println("Dev server cleanly shut down.")
}
```

### 2. Live-Reload File Watcher
```go
reloader := watcher.NewReloader(watcher.ReloaderConfig{
    Watcher:     watcher.DefaultConfig("./src"),
    Command:     "go",
    Args:        []string{"run", "./src"},
    GracePeriod: 2 * time.Second,
    Stdout:      os.Stdout,
    Stderr:      os.Stderr,
})
_ = reloader.Run(ctx)
```

### 3. Cobra CLI Middleware
```go
var serveCmd = &cobra.Command{
    Use:   "serve",
    Short: "Start backend service",
    RunE:  cobraexec.WrapRunE("python", []string{"app.py"}, trexec.WithGracePeriod(3*time.Second)),
}
```

---

## 📄 License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.
