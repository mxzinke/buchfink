//go:build windows

package update

import (
	"os/exec"
	"syscall"
	"time"
)

const (
	// processSynchronize erlaubt, auf das Ende eines Prozesses zu warten.
	processSynchronize = 0x00100000
	detachedProcess    = 0x00000008
)

// detach löst den Nachfolger von der Konsole und Prozessgruppe des
// Vorgängers.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: detachedProcess | syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

func waitForExit(pid int, timeout time.Duration) {
	handle, err := syscall.OpenProcess(processSynchronize, false, uint32(pid))
	if err != nil {
		return
	}
	defer syscall.CloseHandle(handle)
	_, _ = syscall.WaitForSingleObject(handle, uint32(timeout/time.Millisecond))
}
