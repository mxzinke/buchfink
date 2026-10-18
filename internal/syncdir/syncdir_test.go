package syncdir

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectByLocation(t *testing.T) {
	env := map[string]string{
		"OneDrive":           `C:\Users\anna\OneDrive`,
		"OneDriveCommercial": `C:\Users\anna\OneDrive - Pfennig GmbH`,
		"OneDriveConsumer":   `D:\Privat\OneDrive`,
	}
	none := func(string) string { return "" }
	mac := Probe{OS: "darwin", Home: "/Users/anna", Getenv: none}
	win := Probe{OS: "windows", Home: `C:\Users\anna`, Getenv: func(k string) string { return env[k] }}
	linux := Probe{OS: "linux", Home: "/home/anna", Getenv: none}
	cases := []struct {
		name  string
		probe Probe
		path  string
		want  string
	}{
		{"iCloud Mobile Documents", mac, "/Users/anna/Library/Mobile Documents/com~apple~CloudDocs/Buchfink", "iCloud Drive"},
		{"iCloud CloudStorage", mac, "/Users/anna/Library/CloudStorage/iCloudDrive-x/Buchhaltung", "iCloud Drive"},
		{"iCloud ohne Rücksicht auf Groß- und Kleinschreibung", mac, "/users/ANNA/library/mobile documents/Buchfink", "iCloud Drive"},
		{"iCloud unter Windows", win, `C:\Users\anna\iCloudDrive\Buchfink`, "iCloud Drive"},
		{"OneDrive macOS", mac, "/Users/anna/Library/CloudStorage/OneDrive-Persönlich/Buchfink", "OneDrive"},
		{"OneDrive Windows", win, `C:\Users\anna\OneDrive\Dokumente\Buchfink`, "OneDrive"},
		{"OneDrive geschäftlich", win, `c:\users\anna\onedrive - pfennig gmbh\Buchfink`, "OneDrive"},
		{"OneDrive privat", win, `D:\Privat\OneDrive\Buchfink`, "OneDrive"},
		{"OneDrive-Ordner selbst", win, `C:\Users\anna\OneDrive\`, "OneDrive"},
		{"Google Drive macOS", mac, "/Users/anna/Library/CloudStorage/GoogleDrive-anna@example.com/Meine Ablage/Buchfink", "Google Drive"},
		{"Google Drive alter Ort", mac, "/Users/anna/Google Drive/Buchfink", "Google Drive"},
		{"Google Drive als Laufwerk", win, `G:\Meine Ablage\Buchfink`, "Google Drive"},
		{"Google Drive englisch", win, `G:\My Drive\Buchfink`, "Google Drive"},
		{"Dropbox im Heimordner", linux, "/home/anna/Dropbox/Buchfink", "Dropbox"},
		{"Dropbox Business", mac, "/Users/anna/Dropbox (Pfennig)/Buchfink", "Dropbox"},
		{"Dropbox CloudStorage", mac, "/Users/anna/Library/CloudStorage/Dropbox/Buchfink", "Dropbox"},
		{"lokal macOS", mac, "/Users/anna/Documents/Buchfink", ""},
		{"lokal Windows", win, `C:\Users\anna\Documents\Buchfink`, ""},
		{"lokal Linux", linux, "/home/anna/Buchfink", ""},
		{"OneDrive nur im Namen", win, `C:\Users\anna\OneDriveBackup`, ""},
		{"Google-Drive-Laufwerk, anderer Ordner", win, `G:\Buchfink`, ""},
		{"Linux unterscheidet Groß- und Kleinschreibung", linux, "/home/anna/dropbox/x", ""},
		{"anderer Benutzer", mac, "/Users/bernd/Library/Mobile Documents/x", ""},
		{"leer", linux, "", ""},
	}
	for _, tc := range cases {
		if got := tc.probe.Detect(tc.path); got != tc.want {
			t.Errorf("%s: %q, erwartet %q", tc.name, got, tc.want)
		}
	}
}

func TestDetectByMarker(t *testing.T) {
	base := t.TempDir()
	cases := []struct {
		name   string
		marker string
		dir    bool
		want   string
	}{
		{"Syncthing", ".stfolder", true, "Syncthing"},
		{"Nextcloud", ".sync_3f2a9c.db", false, "Nextcloud/ownCloud"},
		{"Nextcloud unter Windows", "._sync_3f2a9c.db", false, "Nextcloud/ownCloud"},
		{"ownCloud", ".owncloudsync.log", false, "Nextcloud/ownCloud"},
		{"Dropbox an anderer Stelle", ".dropbox", false, "Dropbox"},
		{"ohne Marker", "", false, ""},
	}
	probe := Probe{OS: "linux", Home: "/nirgends", Getenv: func(string) string { return "" },
		Exists: func(p string) bool { _, err := os.Lstat(p); return err == nil },
		Glob:   func(p string) []string { m, _ := filepath.Glob(p); return m }}
	for _, tc := range cases {
		root := filepath.Join(base, tc.name)
		target := filepath.Join(root, "Firma", "Buchfink")
		if err := os.MkdirAll(target, 0o755); err != nil {
			t.Fatal(err)
		}
		if tc.marker != "" {
			marker := filepath.Join(root, tc.marker)
			var err error
			if tc.dir {
				err = os.Mkdir(marker, 0o755)
			} else {
				err = os.WriteFile(marker, []byte("x"), 0o600)
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		if got := probe.Detect(target); got != tc.want {
			t.Errorf("%s: %q, erwartet %q", tc.name, got, tc.want)
		}
		// Ein Ordner, den es noch nicht gibt, erbt den Marker darüber.
		if got := probe.Detect(filepath.Join(target, "neu")); got != tc.want {
			t.Errorf("%s, neuer Unterordner: %q, erwartet %q", tc.name, got, tc.want)
		}
	}
}

func TestWarningNamesTheService(t *testing.T) {
	if Warning("") != "" {
		t.Error("ohne Dienst keine Warnung")
	}
	if w := Warning("Dropbox"); !strings.Contains(w, "Dropbox") || !strings.Contains(w, "Sicherung") {
		t.Errorf("Warnung: %q", w)
	}
}
