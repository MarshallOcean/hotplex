//go:build windows

package sys

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"
)

var (
	taskkillPath     string
	taskkillPathOnce sync.Once
)

const (
	JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE = 0x2000
	JOB_OBJECT_LIMIT_BREAKAWAY_OK      = 0x800
)

type JOBOBJECT_BASIC_LIMIT_INFORMATION struct {
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

type JOBOBJECT_EXTENDED_LIMIT_INFORMATION struct {
	BasicLimitInformation JOBOBJECT_BASIC_LIMIT_INFORMATION
	IoInfo                struct {
		ReadOperationCount  uint64
		WriteOperationCount uint64
		ReadTransferCount   uint64
		WriteTransferCount  uint64
	}
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

var (
	kernel32DLL              *syscall.DLL
	createJobObjectW         *syscall.Proc
	setInformationJobObject  *syscall.Proc
	assignProcessToJobObject *syscall.Proc
	jobObjectInitOnce        sync.Once
	jobObjectInitErr         error
)

func initJobObjectAPI() error {
	jobObjectInitOnce.Do(func() {
		kernel32DLL = syscall.NewLazyDLL("kernel32.dll")

		createJobObjectW = kernel32DLL.NewProc("CreateJobObjectW")
		if createJobObjectW == nil {
			jobObjectInitErr = fmt.Errorf("failed to load CreateJobObjectW")
			return
		}

		setInformationJobObject = kernel32DLL.NewProc("SetInformationJobObject")
		if setInformationJobObject == nil {
			jobObjectInitErr = fmt.Errorf("failed to load SetInformationJobObject")
			return
		}

		assignProcessToJobObject = kernel32DLL.NewProc("AssignProcessToJobObject")
		if assignProcessToJobObject == nil {
			jobObjectInitErr = fmt.Errorf("failed to load AssignProcessToJobObject")
			return
		}
	})
	return jobObjectInitErr
}

func SetupCmdSysProcAttr(cmd *exec.Cmd) {
	// Job Object assignment deferred until after process creation
}

func CreateJobObject() (syscall.Handle, error) {
	if err := initJobObjectAPI(); err != nil {
		return 0, fmt.Errorf("job object API not available: %w", err)
	}

	handle, _, err := createJobObjectW.Call(0, 0)
	if handle == 0 {
		return 0, fmt.Errorf("CreateJobObject failed: %w", err)
	}

	info := JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE | JOB_OBJECT_LIMIT_BREAKAWAY_OK,
		},
	}

	ret, _, err := setInformationJobObject.Call(
		uintptr(handle),
		uintptr(9),
		uintptr(unsafe.Pointer(&info)),
		uintptr(unsafe.Sizeof(info)),
	)
	if ret == 0 {
		_ = syscall.CloseHandle(syscall.Handle(handle))
		return 0, fmt.Errorf("SetInformationJobObject failed: %w", err)
	}

	return syscall.Handle(handle), nil
}

func AssignProcessToJob(jobHandle syscall.Handle, process *os.Process) error {
	if err := initJobObjectAPI(); err != nil {
		return fmt.Errorf("job object API not available: %w", err)
	}

	ret, _, err := assignProcessToJobObject.Call(
		uintptr(jobHandle),
		uintptr(process.Pid),
	)
	if ret == 0 {
		return fmt.Errorf("AssignProcessToJobObject failed: %w", err)
	}
	return nil
}

func KillProcessGroup(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	taskkillPathOnce.Do(func() {
		var err error
		taskkillPath, err = exec.LookPath("taskkill")
		if err != nil {
			taskkillPath = os.Getenv("SystemRoot") + "\\system32\\taskkill.exe"
		}
	})

	killCmd := exec.Command(taskkillPath, "/F", "/T", "/PID", fmt.Sprintf("%d", cmd.Process.Pid))
	_ = killCmd.Run()
	_ = cmd.Process.Kill()
}

func IsProcessAlive(process *os.Process) bool {
	if process == nil {
		return false
	}
	return true
}
