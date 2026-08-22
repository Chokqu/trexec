//go:build !linux && unix

package trexec

import "os/exec"

// setupPdeathsig is a no-op on non-Linux Unix systems (macOS, FreeBSD, etc.).
// These platforms lack the prctl(PR_SET_PDEATHSIG) mechanism.
// Process group management is the sole cleanup mechanism on these platforms.
func setupPdeathsig(_ *exec.Cmd) {
	// Not available on this platform.
}
