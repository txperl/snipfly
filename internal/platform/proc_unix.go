//go:build unix

package platform

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/creack/pty"
)

func SetProcAttr(cmd *exec.Cmd, usePTY bool) {
	if usePTY {
		// pty.Start() sets Setsid+Setctty which conflicts with Setpgid.
		// Setsid already creates a new process group, so Setpgid is unnecessary.
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	} else {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}
}

func StartPTY(cmd *exec.Cmd) (*os.File, error) {
	return pty.Start(cmd)
}

func IsEIO(err error) bool {
	return errors.Is(err, syscall.EIO)
}

// StopProcess sends SIGTERM to the process group, then SIGKILL after 5 seconds.
func StopProcess(cmd *exec.Cmd, done <-chan struct{}) {
	pid := cmd.Process.Pid
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		// Fallback: signal the process directly
		_ = cmd.Process.Signal(syscall.SIGTERM)
	} else {
		_ = syscall.Kill(-pgid, syscall.SIGTERM)
	}

	// Wait up to 5 seconds for graceful exit, then SIGKILL
	select {
	case <-done:
		return
	case <-time.After(5 * time.Second):
	}

	if err != nil {
		_ = cmd.Process.Kill()
	} else {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	}
}
