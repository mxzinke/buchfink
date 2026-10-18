//go:build !windows

package policy

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
)

// openNoFollow öffnet eine Datei und folgt dabei keinem Symlink im letzten
// Glied des Pfads.
func openNoFollow(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
}

// linkCount ist die Zahl der harten Verweise auf eine geöffnete Datei.
func linkCount(_ *os.File, info os.FileInfo) (uint64, error) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("unbekannte Dateiangaben")
	}
	return uint64(st.Nlink), nil
}

// openedPath ist der Pfad, unter dem das System die geöffnete Datei führt;
// nur unter Linux bekannt.
func openedPath(f *os.File) (string, bool) {
	if runtime.GOOS != "linux" {
		return "", false
	}
	p, err := os.Readlink(fmt.Sprintf("/proc/self/fd/%d", f.Fd()))
	if err != nil {
		return "", false
	}
	return p, true
}
