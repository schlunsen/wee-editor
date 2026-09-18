//go:build windows

package gpu

import "os"

// notifyWindowChange is a no-op on Windows, which has no SIGWINCH; the remote
// PTY simply keeps the size negotiated when the session started.
func notifyWindowChange(chan<- os.Signal) {}
