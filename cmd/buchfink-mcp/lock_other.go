//go:build !unix && !windows

package main

import "os"

// Ohne Dateisperre der Plattform hilft nur, eine Brücke je Benutzer zu starten.
func lockFile(*os.File) error   { return nil }
func unlockFile(*os.File) error { return nil }
