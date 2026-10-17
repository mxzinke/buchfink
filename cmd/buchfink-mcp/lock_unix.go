//go:build unix

package main

import (
	"os"
	"syscall"
)

// lockFile wartet, bis kein anderer Brücken-Prozess die Datei hält.
func lockFile(f *os.File) error { return syscall.Flock(int(f.Fd()), syscall.LOCK_EX) }

func unlockFile(f *os.File) error { return syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }
