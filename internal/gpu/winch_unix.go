//go:build !windows

package gpu

import (
	"os"
	"os/signal"
	"syscall"
)

// notifyWindowChange delivers terminal resize notifications on sigCh.
func notifyWindowChange(sigCh chan<- os.Signal) {
	signal.Notify(sigCh, syscall.SIGWINCH)
}
