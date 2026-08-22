# trexec

<p align="center">
  <img src="https://raw.githubusercontent.com/Chokqu/trexec/master/doc/logo.png" alt="trexec logo" width="120" onerror="this.style.display='none'"/>
  <br>
  <strong>Own the workload, not just the process.</strong><br>
  Cross-platform process tree lifecycle manager, supervisor, and dev engine for Go.<br>
  <em>Graceful shutdown, two-phase force-kill escalation, interactive I/O streaming, live-reloading, telemetry, and zero orphan processes on Linux, macOS, and Windows.</em>
</p>

<p align="center">
  <a href="https://github.com/Chokqu/trexec/actions/workflows/ci.yml"><img src="https://github.com/Chokqu/trexec/actions/workflows/ci.yml/badge.svg" alt="CI Status"></a>
  <a href="https://pkg.go.dev/github.com/Chokqu/trexec"><img src="https://pkg.go.dev/badge/github.com/Chokqu/trexec.svg" alt="Go Reference"></a>
  <a href="https://goreportcard.com/report/github.com/Chokqu/trexec"><img src="https://goreportcard.com/badge/github.com/Chokqu/trexec" alt="Go Report Card"></a>
  <a href="https://github.com/Chokqu/trexec/releases"><img src="https://img.shields.io/github/v/release/Chokqu/trexec?color=blue&label=version" alt="Latest Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-green.svg" alt="License: MIT"></a>
</p>

---

## 🎯 Why trexec?

Go's standard `os/exec` only manages a **single process PID**. In modern software, commands routinely spawn deep process trees:
- `npm run dev` &rarr; `sh` &rarr; `vite` &rarr; `esbuild` &rarr; worker threads
- `docker-compose up` &rarr; multiple container daemons
- `python main.py` &rarr; multiprocessing pools & workers

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

`trexec` solves this completely and natively across **Linux, macOS, and Windows** with **zero third-party runtime dependencies**.

---

## ⚡ Key Features

- **🌲 Complete Tree Ownership**: Eliminates orphan processes, zombie subprocesses, and lingering port locks.
- **⏱️ Two-Phase Graceful Shutdown**: Polite termination signal (`SIGTERM`, `SIGINT`, `Ctrl+Break`) &rarr; configurable grace period &rarr; kernel force-kill (`SIGKILL`, `TerminateJobObject`).
- **🛡️ Pipe Hang Guard**: Built-in I/O timeouts unblock stuck `io.Copy` goroutines if orphaned writers keep pipes open.
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

## 📦 Installation

```bash
go get github.com/Chokqu/trexec
```

Compatible with **Go 1.21, 1.22, 1.23, 1.24, 1.25, and 1.26+**.

---

## 🚀 Quick Start & Usage

### 1. One-Liner Command Execution with Structured Result

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/Chokqu/trexec"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    result, err := trexec.RunWithResult(ctx, "npm", []string{"run", "dev"},
        trexec.WithGracePeriod(5*time.Second), // SIGTERM -> wait 5s -> SIGKILL
        trexec.WithGracefulSignal(trexec.SIGINT),
        trexec.WithStdout(os.Stdout),
        trexec.WithStderr(os.Stderr),
    )
    if err != nil {
        log.Fatalf("Setup error: %v", err)
    }

    fmt.Printf("Exit Code: %d, Cancelled: %v, Graceful: %v, Cleaned PIDs: %d, Duration: %s\n",
        result.ExitCode, result.Cancelled, result.GracefullyTerminated, result.ProcessesCleaned, result.Duration)
}
```

---

### 2. Capturing Output (`Output` & `CombinedOutput`)

```go
// Direct standard output capture
out, result, err := trexec.Output(ctx, "git", []string{"status", "--porcelain"})
if err != nil {
    log.Fatalf("Git failed (exit %d): %v\nOutput: %s", result.ExitCode, err, string(out))
}
fmt.Printf("Git status:\n%s", out)

// Thread-safe combined stdout + stderr capture
comb, res, err := trexec.CombinedOutput(ctx, "go", []string{"test", "./..."})
```

---

### 3. Interactive Input Streaming (`StdinPipe`)

```go
cmd := trexec.CommandContext(ctx, "bc", nil, trexec.WithStdout(os.Stdout))

stdin, err := cmd.StdinPipe()
if err != nil {
    log.Fatal(err)
}

if err := cmd.Start(); err != nil {
    log.Fatal(err)
}

// Stream data dynamically into the process tree
fmt.Fprintln(stdin, "10 * 5")
fmt.Fprintln(stdin, "sqrt(144)")

// Close pipe to signal EOF
stdin.Close()

result := cmd.Wait()
```

---

### 4. Live-Reload DevServer (`trexec/watcher`)

```go
package main

import (
    "context"
    "log"
    "os"
    "time"

    "github.com/Chokqu/trexec/watcher"
)

func main() {
    reloader := watcher.NewReloader(watcher.ReloaderConfig{
        Watcher: watcher.DefaultConfig("./src", "./cmd"),
        Command: "go",
        Args:    []string{"run", "./cmd/server"},
        Stdout:  os.Stdout,
        Stderr:  os.Stderr,
        GracePeriod: 2 * time.Second,
        OnRestart: func(attempt int, changedFiles []string) {
            log.Printf("[reloader] Files modified (%v) -> Restarting server (attempt #%d)...", changedFiles, attempt)
        },
    })

    if err := reloader.Run(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

---

### 5. Multi-Worker Supervisor with Exponential Backoff (`trexec/supervisor`)

```go
sup := supervisor.New()

// Register workers with custom restart policies
_ = sup.Add(supervisor.Spec{
    Name:          "api-server",
    Command:       "./bin/api",
    RestartPolicy: supervisor.RestartAlways,
    GracePeriod:   5 * time.Second,
})

_ = sup.Add(supervisor.Spec{
    Name:          "queue-consumer",
    Command:       "./bin/worker",
    RestartPolicy: supervisor.RestartOnFailure,
    MaxRestarts:   10,
    Backoff:       supervisor.DefaultBackoff(),
})

// Start supervising pool
_ = sup.Start(ctx)

// Wait for pool shutdown or interrupt
results := sup.Wait()
```

---

### 6. Cobra CLI Integration (`trexec/cobraexec`)

```go
package cmd

import (
    "time"

    "github.com/Chokqu/trexec"
    "github.com/Chokqu/trexec/cobraexec"
    "github.com/spf13/cobra"
)

var devCmd = &cobra.Command{
    Use:   "dev",
    Short: "Run frontend development server with automatic tree supervision",
    RunE:  cobraexec.WrapRunE("npm", []string{"run", "dev"}, trexec.WithGracePeriod(5*time.Second)),
}
```

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
| `BenchmarkTrexecSupervisorRestart` | `4.38 ms/op` | `12,453 B/op (57 allocs)` | Worker termination & supervised respawn |

---

## 📊 Feature Comparison Matrix

| Feature | `os/exec` (Stdlib) | `jesseduffield/kill` | `tree-kill` (Node.js) | `trexec` (This Library) |
|---|:---:|:---:|:---:|:---:|
| **Kills entire descendant tree** | ❌ (Direct PID only) | ⚠️ (Unix only) | ⚠️ (via `taskkill.exe`) | **✅ Full native tree cleanup** |
| **Two-phase graceful escalation** | ❌ | ❌ | ❌ | **✅ Built-in (`WithGracePeriod`)** |
| **Windows Job Object kernel integration** | ❌ | ❌ | ❌ | **✅ Native (`KILL_ON_JOB_CLOSE`)** |
| **Pipe deadlock timeout protection** | ⚠️ (`WaitDelay`) | ❌ | ❌ | **✅ Automatic (`WithIOTimeout`)** |
| **Rich outcome metadata (`Result`)** | ❌ (Single error) | ❌ | ❌ | **✅ Structured `Result` struct** |
| **Cross-platform signal abstraction** | ❌ (Requires `syscall`) | ❌ | ❌ | **✅ Unified `trexec.Signal`** |
| **Interactive `StdinPipe` streaming** | ⚠️ (Leaky handles) | ❌ | ❌ | **✅ Safe bounded streaming** |
| **Descendant PID introspection** | ❌ | ❌ | ❌ | **✅ `Result.DescendantPIDs`** |
| **Live-Reload File Watcher Engine** | ❌ | ❌ | ❌ | **✅ Built-in `trexec/watcher`** |
| **Embedded Multi-Worker Supervisor** | ❌ | ❌ | ❌ | **✅ Built-in `trexec/supervisor`** |
| **Zero third-party runtime dependencies** | ✅ | ✅ | ❌ | **✅ Stdlib + official `x/sys`** |

---

## 🔬 How It Works Under the Hood

### 🐧 Linux & Unix Backend
1. **Process Groups (`setpgid`)**: During `fork/exec`, the child process is assigned as the leader of a new process group (`setpgid(0, 0)`).
2. **Negative PID Broadcasting**: Signals are sent to `-pgid` (`syscall.Kill(-pgid, sig)`), delivering signals to all child and grandchild processes simultaneously.
3. **Linux `PR_SET_PDEATHSIG`**: Configures kernel death signals so child trees terminate automatically if the parent process crashes.
4. **`/proc` Introspection**: Scans `/proc/*/stat` to enumerate live member PIDs.

### 🪟 Windows Backend
1. **Win32 Job Objects**: Creates an anonymous Job Object with `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE | JOB_OBJECT_LIMIT_SILENT_BREAKAWAY_OK`.
2. **Suspended Process Assignment**: Spawns the root process with `CREATE_SUSPENDED`, binds it to the Job Object via `AssignProcessToJobObject`, and unwinds thread suspension counts using `Toolhelp32Snapshot`.
3. **Graceful Console Events**: Sends `CTRL_BREAK_EVENT` or `CTRL_C_EVENT` to console process groups for clean application shutdown.
4. **Fallback Tree Termination**: If running in restricted sandboxes or nested CI runners where Job Object creation is blocked, transparently degrades to recursive process tree termination (`taskkill /F /T /PID`).

---

## 🤝 Contributing

Contributions, issues, and feature requests are welcome! Feel free to open an issue or pull request on GitHub.

---

## 📄 License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.
