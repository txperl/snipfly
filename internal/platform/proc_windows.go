//go:build windows

package platform

import (
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

func SetProcAttr(cmd *exec.Cmd, usePTY bool) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

func StartPTY(_ *exec.Cmd) (*os.File, error) {
	return nil, ErrPTYUnsupported
}

func IsEIO(_ error) bool {
	return false
}

// StopProcess attempts graceful shutdown via CTRL_BREAK_EVENT, then
// escalates to forceful tree kill after a 5-second timeout.
func StopProcess(cmd *exec.Cmd, done <-chan struct{}) {
	pid := cmd.Process.Pid

	// Try graceful shutdown via CTRL_BREAK_EVENT
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	generateConsoleCtrlEvent := kernel32.NewProc("GenerateConsoleCtrlEvent")
	generateConsoleCtrlEvent.Call(syscall.CTRL_BREAK_EVENT, uintptr(pid))

	// Wait up to 5 seconds for graceful exit
	select {
	case <-done:
		return
	case <-time.After(5 * time.Second):
	}

	// Force kill the process tree
	pidStr := strconv.Itoa(pid)
	kill := exec.Command("taskkill", "/T", "/F", "/PID", pidStr)
	if err := kill.Run(); err != nil {
		_ = cmd.Process.Kill()
	}
}
