//go:build !windows

package update

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// detach löst den Nachfolger von der Sitzung des Vorgängers, damit er dessen
// Ende überlebt.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

func waitForExit(pid int, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) && !exited(pid) {
		time.Sleep(100 * time.Millisecond)
	}
}

func exited(pid int) bool {
	// Signal 0 prüft nur, ob es den Prozess gibt.
	if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
		return true
	}
	// Einen beendeten Prozess, den sein Elternprozess noch nicht abgeholt hat,
	// führt Linux weiter; sein Zustand in /proc ist dann Z.
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return false
	}
	end := bytes.LastIndexByte(stat, ')')
	return end >= 0 && end+2 < len(stat) && stat[end+2] == 'Z'
}
