//go:build windows

package mcpserver

import (
	"errors"
	"syscall"
)

// wsaeaddrinuse ist WSAEADDRINUSE; syscall.EADDRINUSE trifft unter Windows nicht.
const wsaeaddrinuse = syscall.Errno(10048)

// addrInUse meldet einen belegten Port.
func addrInUse(err error) bool {
	return errors.Is(err, wsaeaddrinuse) || errors.Is(err, syscall.EADDRINUSE)
}
