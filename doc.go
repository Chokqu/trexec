// Package trexec provides process tree lifecycle management for Go.
//
// trexec is a higher-level alternative to os/exec that treats the entire
// process tree spawned by a command as a single ownable, cancellable,
// cleanable unit of work. It works consistently across Linux, macOS,
// and Windows.
//
// # The Problem
//
// Go's os/exec manages a single process (PID). But real commands spawn
// process trees — npm spawns node, which spawns esbuild, etc. When you
// cancel the command, os/exec kills the direct child. The rest of the
// tree keeps running, leaking ports, files, and resources.
//
// # What trexec Does
//
// trexec uses OS-level primitives to group and manage all descendant
// processes as a unit:
//
//   - Unix: process groups via setpgid + kill(-pgid, signal)
//   - Linux: Pdeathsig as a safety net for parent crashes
//   - Windows: Job Objects with JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
//
// # Graceful Shutdown
//
// When a context is cancelled, trexec follows a two-phase shutdown:
//
//  1. Send a graceful termination signal (SIGTERM / Ctrl+Break) to the
//     entire process tree
//  2. Wait for a configurable grace period
//  3. If processes remain, force-kill the entire tree (SIGKILL / TerminateJobObject)
//  4. Clean up I/O pipes and return a structured Result
//
// # Quick Start
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	// Run with structured result
//	result, err := trexec.RunWithResult(ctx, "npm", []string{"run", "dev"},
//	    trexec.WithGracePeriod(5*time.Second),
//	    trexec.WithStdout(os.Stdout),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("exit=%d cancelled=%v cleaned=%d\n",
//	    result.ExitCode, result.Cancelled, result.ProcessesCleaned)
//
//	// Capture output directly
//	out, res, err := trexec.Output(ctx, "git", []string{"status", "--porcelain"})
//	if err != nil {
//	    log.Fatalf("git failed: %v", err)
//	}
//	fmt.Printf("Status: %s\n", out)
package trexec
