// Package syncdir erkennt, ob ein Ordner von einem Dienst synchronisiert
// wird. Die Datenbank eines Mandanten verträgt das nicht, solange Buchfink
// läuft: der Dienst kopiert eine halb geschriebene Datei oder legt eine
// Konfliktkopie daneben. Die Einrichtung warnt deshalb, blockiert aber nicht.
package syncdir

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Probe ist die Umgebung, in der erkannt wird.
type Probe struct {
	// OS ist runtime.GOOS des Rechners.
	OS string
	// Home ist das Benutzerverzeichnis.
	Home string
	// Getenv liest eine Umgebungsvariable.
	Getenv func(string) string
	// Exists meldet, ob es eine Datei oder einen Ordner gibt; Glob findet
	// Dateien nach Muster. Beide prüfen die Marker der Dienste in den
	// Elternordnern.
	Exists func(string) bool
	Glob   func(string) []string
}

// Detect prüft einen Ordner auf diesem Rechner.
func Detect(path string) string {
	home, _ := os.UserHomeDir()
	return Probe{
		OS: runtime.GOOS, Home: home, Getenv: os.Getenv,
		Exists: func(p string) bool { _, err := os.Lstat(p); return err == nil },
		Glob:   func(p string) []string { m, _ := filepath.Glob(p); return m },
	}.Detect(path)
}

// Detect nennt den Dienst, der den Ordner synchronisiert, oder "".
func (p Probe) Detect(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	target := p.norm(path)
	home := p.norm(p.Home)
	under := func(base string) bool {
		base = strings.TrimRight(base, "/")
		return base != "" && (target == base || strings.HasPrefix(target, base+"/"))
	}
	underPrefix := func(parent, prefix string) bool {
		parent = strings.TrimRight(parent, "/") + "/"
		if !strings.HasPrefix(target, parent) {
			return false
		}
		first := strings.SplitN(strings.TrimPrefix(target, parent), "/", 2)[0]
		return strings.HasPrefix(first, prefix)
	}
	cloud := home + "/" + p.fold("Library/CloudStorage")
	switch {
	case home == "":
	case under(home + "/" + p.fold("Library/Mobile Documents")), underPrefix(cloud, p.fold("iCloud")),
		under(home + "/" + p.fold("iCloudDrive")):
		return "iCloud Drive"
	case underPrefix(cloud, p.fold("OneDrive")):
		return "OneDrive"
	case underPrefix(cloud, p.fold("GoogleDrive")), under(home + "/" + p.fold("Google Drive")):
		return "Google Drive"
	case underPrefix(cloud, p.fold("Dropbox")), underPrefix(home, p.fold("Dropbox")):
		return "Dropbox"
	}
	switch {
	case p.underEnv(target, "OneDrive"), p.underEnv(target, "OneDriveCommercial"), p.underEnv(target, "OneDriveConsumer"):
		return "OneDrive"
	case p.googleDriveLetter(target):
		return "Google Drive"
	}
	return p.marker(path)
}

// marker sucht die Kennzeichen der Dienste, die einen beliebigen Ordner
// synchronisieren, im Ordner selbst und allen Elternordnern.
func (p Probe) marker(path string) string {
	if p.Exists == nil {
		return ""
	}
	dir := filepath.Clean(path)
	for {
		switch {
		case p.Exists(filepath.Join(dir, ".stfolder")):
			return "Syncthing"
		case p.Exists(filepath.Join(dir, ".owncloudsync.log")), p.glob(filepath.Join(dir, ".sync_*.db")),
			p.glob(filepath.Join(dir, "._sync_*.db")):
			return "Nextcloud/ownCloud"
		case p.Exists(filepath.Join(dir, ".dropbox")):
			return "Dropbox"
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func (p Probe) glob(pattern string) bool {
	return p.Glob != nil && len(p.Glob(pattern)) > 0
}

// underEnv meldet, ob der Pfad unter dem Ordner einer Umgebungsvariable
// liegt (OneDrive unter Windows).
func (p Probe) underEnv(target, name string) bool {
	if p.Getenv == nil {
		return false
	}
	value := strings.TrimSpace(p.Getenv(name))
	if value == "" {
		return false
	}
	base := strings.TrimRight(p.norm(value), "/")
	return base != "" && (target == base || strings.HasPrefix(target, base+"/"))
}

// googleDriveLetter erkennt das Laufwerk von Google Drive für Windows: „G:\Meine
// Ablage", „G:\My Drive" oder die geteilten Ablagen.
func (p Probe) googleDriveLetter(target string) bool {
	if p.OS != "windows" || len(target) < 3 || target[1] != ':' || target[2] != '/' {
		return false
	}
	first := strings.SplitN(target[3:], "/", 2)[0]
	for _, name := range []string{"Meine Ablage", "My Drive", "Geteilte Ablagen", "Shared drives"} {
		if first == p.fold(name) {
			return true
		}
	}
	return false
}

// norm bringt einen Pfad in die Form, in der verglichen wird: Schrägstriche,
// ohne Endstrich, unter Windows und macOS ohne Groß- und Kleinschreibung.
func (p Probe) norm(path string) string {
	path = strings.ReplaceAll(strings.TrimSpace(path), `\`, "/")
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
	}
	return p.fold(path)
}

func (p Probe) fold(s string) string {
	if p.OS == "windows" || p.OS == "darwin" {
		return strings.ToLower(s)
	}
	return s
}

// Warning ist der Hinweis zu einem synchronisierten Datenordner; leer ohne
// Dienst.
func Warning(service string) string {
	if service == "" {
		return ""
	}
	return "Der Ordner liegt in " + service + ". Die Datenbank verträgt keine Synchronisation, während Buchfink " +
		"läuft; besser ist ein lokaler Ordner mit einer Sicherung."
}
