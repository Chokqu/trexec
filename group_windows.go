//go:build windows

package trexec

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// windowsProcessGroup implements processGroup using Windows Job Objects
// with fallback support for restricted or nested CI environments.
//
// It assigns the child process to a Job Object configured with
// JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE so the operating system kernel
// guarantees that all child and descendant processes are terminated when
// the job handle is closed or terminated.
type windowsProcessGroup struct {
	job windows.Handle
	cmd *exec.Cmd
}

func newProcessGroup() (processGroup, error) {
	// Create an anonymous Job Object
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		// Fallback: continue without Job Object if restricted by environment
		return &windowsProcessGroup{job: 0}, nil
	}

	// Set kill-on-close limit: when the last job handle is closed,
	// all processes in the job are automatically terminated by the kernel.
	// Also set SILENT_BREAKAWAY_OK to prevent ERROR_ACCESS_DENIED in nested CI runners.
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE | windows.JOB_OBJECT_LIMIT_SILENT_BREAKAWAY_OK,
		},
	}

	_, err = windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	)
	if err != nil {
		// If limit setting fails, close handle and operate in fallback mode
		windows.CloseHandle(job)
		return &windowsProcessGroup{job: 0}, nil
	}

	return &windowsProcessGroup{job: job}, nil
}

func (g *windowsProcessGroup) setup(cmd *exec.Cmd) error {
	g.cmd = cmd

	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	// Start the process suspended so we can assign it to the Job Object
	// before it starts executing and potentially spawning grandchildren.
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_SUSPENDED

	// Create a new console process group for console control events.
	cmd.SysProcAttr.CreationFlags |= syscall.CREATE_NEW_PROCESS_GROUP

	return nil
}

func (g *windowsProcessGroup) activate(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return fmt.Errorf("trexec: process not created")
	}

	// If a Job Object is available, attempt to assign the process to it
	if g.job != 0 {
		processHandle, err := windows.OpenProcess(
			windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|windows.PROCESS_QUERY_INFORMATION,
			false,
			uint32(cmd.Process.Pid),
		)
		if err == nil {
			// Assign to Job Object
			_ = windows.AssignProcessToJobObject(g.job, processHandle)
			windows.CloseHandle(processHandle)
		}
	}

	// ALWAYS resume the suspended main thread so the process can execute
	err := resumeProcessThreads(cmd.Process.Pid)
	if err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("trexec: resume threads for PID %d: %w", cmd.Process.Pid, err)
	}

	return nil
}

func (g *windowsProcessGroup) signal(sig Signal) error {
	if g.cmd == nil || g.cmd.Process == nil {
		return os.ErrProcessDone
	}

	pgid := uint32(g.cmd.Process.Pid)

	switch sig {
	case SIGINT, SIGTERM, SIGHUP:
		// Send CTRL_BREAK_EVENT to the process group.
		// Note: On Windows, CTRL_C_EVENT is only delivered to processes sharing the same
		// console, whereas CTRL_BREAK_EVENT is universally delivered across process groups.
		_ = windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, pgid)
		return nil

	case SIGKILL:
		return g.terminate()

	default:
		return fmt.Errorf("trexec: unsupported signal %s on Windows", sig)
	}
}

func (g *windowsProcessGroup) terminate() error {
	var termErr error

	// 1. Terminate Job Object if available
	if g.job != 0 {
		termErr = windows.TerminateJobObject(g.job, 1)
	}

	// 2. Kill direct child process
	if g.cmd != nil && g.cmd.Process != nil {
		_ = g.cmd.Process.Kill()

		// 3. Fallback tree kill for non-Job Object environments
		if g.job == 0 {
			fallbackTreeKill(g.cmd.Process.Pid)
		}
	}

	return termErr
}

func (g *windowsProcessGroup) close() error {
	if g.job != 0 {
		err := windows.CloseHandle(g.job)
		g.job = 0
		return err
	}
	return nil
}

type jobObjectBasicProcessIdList struct {
	NumberOfAssignedProcesses uint32
	NumberOfProcessIdsInList  uint32
	ProcessIdList             [1]uintptr
}

func (g *windowsProcessGroup) pids() ([]int, error) {
	if g.job == 0 {
		if g.cmd != nil && g.cmd.Process != nil {
			return []int{g.cmd.Process.Pid}, nil
		}
		return nil, nil
	}

	const maxProcs = 512
	bufSize := uint32(unsafe.Sizeof(jobObjectBasicProcessIdList{}) +
		(maxProcs-1)*unsafe.Sizeof(uintptr(0)))
	buf := make([]byte, bufSize)

	var returnLen uint32
	err := windows.QueryInformationJobObject(
		g.job,
		windows.JobObjectBasicProcessIdList,
		uintptr(unsafe.Pointer(&buf[0])),
		bufSize,
		&returnLen,
	)
	if err != nil {
		if g.cmd != nil && g.cmd.Process != nil {
			return []int{g.cmd.Process.Pid}, nil
		}
		return nil, err
	}

	info := (*jobObjectBasicProcessIdList)(unsafe.Pointer(&buf[0]))
	count := int(info.NumberOfProcessIdsInList)
	if count == 0 {
		if g.cmd != nil && g.cmd.Process != nil {
			return []int{g.cmd.Process.Pid}, nil
		}
		return nil, nil
	}

	pids := make([]int, 0, count)
	pidSlice := unsafe.Slice(&info.ProcessIdList[0], count)
	for _, pid := range pidSlice {
		pids = append(pids, int(pid))
	}
	return pids, nil
}

type jobObjectExtendedLimitInformation struct {
	BasicLimitInformation struct {
		PerProcessUserTimeLimit int64
		PerJobUserTimeLimit     int64
		LimitFlags              uint32
		MinimumWorkingSetSize   uintptr
		MaximumWorkingSetSize   uintptr
		ActiveProcessLimit      uint32
		Affinity                uintptr
		PriorityClass           uint32
		SchedulingClass         uint32
	}
	IoInfo struct {
		ReadOperationCount  uint64
		WriteOperationCount uint64
		OtherOperationCount uint64
		ReadTransferCount   uint64
		WriteTransferCount  uint64
		OtherTransferCount  uint64
	}
	ProcessMemoryLimit     uintptr
	JobMemoryLimit         uintptr
	PeakProcessMemoryLimit uintptr
	PeakJobMemoryLimit     uintptr
}

func (g *windowsProcessGroup) setLimits(limits *ResourceLimits) error {
	if limits == nil || g.job == 0 {
		return nil
	}

	var info jobObjectExtendedLimitInformation
	if limits.MaxMemoryBytes > 0 {
		const JOB_OBJECT_LIMIT_JOB_MEMORY = 0x00000200
		info.BasicLimitInformation.LimitFlags |= JOB_OBJECT_LIMIT_JOB_MEMORY
		info.JobMemoryLimit = uintptr(limits.MaxMemoryBytes)
	}

	if limits.MaxProcesses > 0 {
		const JOB_OBJECT_LIMIT_ACTIVE_PROCESS = 0x00000008
		info.BasicLimitInformation.LimitFlags |= JOB_OBJECT_LIMIT_ACTIVE_PROCESS
		info.BasicLimitInformation.ActiveProcessLimit = uint32(limits.MaxProcesses)
	}

	if info.BasicLimitInformation.LimitFlags == 0 {
		return nil
	}

	const JobObjectExtendedLimitInformation = 9
	_, err := windows.SetInformationJobObject(
		g.job,
		JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	)
	return err
}

// resumeProcessThreads finds and resumes all threads belonging to the given PID.
// It retries with backoff in case Windows thread table registration is delayed.
func resumeProcessThreads(pid int) error {
	var lastErr error

	for attempt := 0; attempt < 10; attempt++ {
		snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Millisecond)
			continue
		}

		var te windows.ThreadEntry32
		te.Size = uint32(unsafe.Sizeof(te))

		resumedAny := false
		err = windows.Thread32First(snapshot, &te)
		for err == nil {
			if te.OwnerProcessID == uint32(pid) {
				thread, openErr := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, te.ThreadID)
				if openErr == nil {
					// Unwind suspend count to 0
					for {
						count, resumeErr := windows.ResumeThread(thread)
						if count == 0xFFFFFFFF || count <= 1 || resumeErr != nil {
							break
						}
					}
					windows.CloseHandle(thread)
					resumedAny = true
				}
			}
			err = windows.Thread32Next(snapshot, &te)
		}
		windows.CloseHandle(snapshot)

		if resumedAny {
			return nil
		}

		time.Sleep(2 * time.Millisecond)
	}

	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("no thread found for PID %d after retry", pid)
}

// fallbackTreeKill attempts to terminate a process tree using taskkill on Windows
// when Job Objects are disabled or restricted.
func fallbackTreeKill(pid int) {
	if pid <= 0 {
		return
	}
	killCmd := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
	_ = killCmd.Run()
}
