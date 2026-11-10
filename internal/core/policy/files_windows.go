//go:build windows

package policy

import (
	"os"
	"syscall"
)

// openNoFollow öffnet eine Datei. Unter Windows folgt das Öffnen einem
// Symlink; die Prüfung danach (SameFile, erneutes Auflösen) fängt ihn ab.
func openNoFollow(path string) (*os.File, error) {
	return os.Open(path)
}

// linkCount ist die Zahl der harten Verweise auf eine geöffnete Datei.
func linkCount(f *os.File, _ os.FileInfo) (uint64, error) {
	var d syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(syscall.Handle(f.Fd()), &d); err != nil {
		return 0, err
	}
	return uint64(d.NumberOfLinks), nil
}

// openedPath ist unter Windows nicht bekannt.
func openedPath(*os.File) (string, bool) { return "", false }
