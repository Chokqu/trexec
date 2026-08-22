package trexec

// ResourceLimits defines kernel-enforced resource constraints for a process tree.
//
// These limits apply to the entire collection of descendant processes as a single unit:
//   - Windows: Configured on the Win32 Job Object via JobObjectExtendedLimitInformation.
//   - Linux/Unix: Configured via POSIX rlimits or platform-specific controls.
type ResourceLimits struct {
	// MaxMemoryBytes limits the total committed memory (RAM) allocated across
	// all processes in the tree. If exceeded, the OS kernel denies further allocations.
	MaxMemoryBytes int64

	// MaxProcesses limits the total number of simultaneously active processes
	// allowed in the process tree. If exceeded, process spawning fails.
	MaxProcesses int
}

// WithResourceLimits applies hard resource boundaries to the command and its descendants.
func WithResourceLimits(limits ResourceLimits) Option {
	return func(o *options) {
		o.resourceLimits = &limits
	}
}
