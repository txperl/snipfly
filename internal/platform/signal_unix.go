//go:build unix

package platform

import (
	"os"
	"syscall"
)

func OsSignals() []os.Signal {
	return []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP}
}
