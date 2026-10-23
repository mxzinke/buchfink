//go:build !windows

package mcpserver

import (
	"errors"
	"syscall"
)

// addrInUse meldet einen belegten Port.
func addrInUse(err error) bool { return errors.Is(err, syscall.EADDRINUSE) }
