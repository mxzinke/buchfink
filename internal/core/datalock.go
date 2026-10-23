package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// lockFileName ist die Sperrdatei im Datenordner. Sie liegt neben der
// Datenbank und gehört nicht in die Sicherung.
const lockFileName = "buchfink.lock"

// ErrTenantInUse: der Datenordner ist schon in einem anderen Prozess offen.
var ErrTenantInUse = errors.New(
	"der Mandant ist schon geöffnet, in einem anderen Buchfink-Fenster oder Programm. " +
		"Ein Mandant hat genau einen Schreiber; bitte das andere Fenster schließen")

// errLocked meldet die plattformeigene Sperre als belegt.
var errLocked = errors.New("gesperrt")

// dataLock hält einen Datenordner für diesen Prozess. Schreibsperre,
// Kontenplan und Prüfermodus im Speicher setzen einen Schreiber je Mandant
// voraus; ein zweiter Prozess auf demselben Ordner wird beim Öffnen
// abgewiesen.
type dataLock struct{ file *os.File }

func lockDataDir(dataDir string) (*dataLock, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("der Datenordner %s ließ sich nicht anlegen: %w", dataDir, err)
	}
	path := filepath.Join(dataDir, lockFileName)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("die Sperrdatei %s ließ sich nicht öffnen: %w", path, err)
	}
	if err := lockFile(file); err != nil {
		_ = file.Close()
		if errors.Is(err, errLocked) {
			return nil, ErrTenantInUse
		}
		return nil, fmt.Errorf("die Sperrdatei %s ließ sich nicht sperren: %w", path, err)
	}
	return &dataLock{file: file}, nil
}

func (l *dataLock) release() error {
	if l == nil {
		return nil
	}
	return errors.Join(unlockFile(l.file), l.file.Close())
}
