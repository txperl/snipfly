//go:build windows

package platform

import (
	"os"
	"syscall"
)

func OsSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}
