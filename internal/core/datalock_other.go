//go:build !unix && !windows

package core

import "os"

// Ohne Dateisperre der Plattform schützt nur die Schreibsperre im Prozess.
func lockFile(*os.File) error   { return nil }
func unlockFile(*os.File) error { return nil }
