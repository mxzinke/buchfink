package domain

import (
	"fmt"
	"strings"
)

// cloudFolderMarkers sind Pfadbestandteile bekannter Synchronisationsordner.
//
// Erkannt wird am Namen und nicht am Dateisystem: die Anbieter legen ihre
// Ordner unter diesen Namen an, und ein Programm, das den
// Synchronisationsdienst selbst befragte, bräuchte dafür Rechte, die es sonst
// nicht braucht.
var cloudFolderMarkers = []struct {
	needle string
	name   string
}{
	{"onedrive", "OneDrive"},
	{"dropbox", "Dropbox"},
	{"google drive", "Google Drive"},
	{"googledrive", "Google Drive"},
	{"gdrive", "Google Drive"},
	{"icloud", "iCloud"},
	{"nextcloud", "Nextcloud"},
	{"owncloud", "ownCloud"},
	{"magentacloud", "MagentaCLOUD"},
}

// CloudFolderWarning meldet, wenn der Datenordner in einem
// Synchronisationsordner liegt. Leer heißt: unauffällig.
//
// EU-Speicherung setzt nach § 146 Abs. 2a AO vollständigen Datenzugriff voraus.
// Für Drittstaaten verlangt Abs. 2b eine Bewilligung. Den tatsächlichen
// Speicherort eines Synchronisationsdienstes kann die Pfadprüfung nicht ermitteln.
func CloudFolderWarning(path string) string {
	lowered := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	for _, marker := range cloudFolderMarkers {
		if !strings.Contains(lowered, marker.needle) {
			continue
		}
		return fmt.Sprintf(
			"Der Datenordner liegt in einem %s-Ordner. Elektronische Bücher dürfen in anderen "+
				"EU-Mitgliedstaaten gespeichert werden, wenn der gesetzliche Datenzugriff vollständig möglich bleibt "+
				"(§ 146 Abs. 2a AO). Für die Speicherung in Drittstaaten ist eine Bewilligung des Finanzamts "+
				"nach § 146 Abs. 2b AO nötig. Wo ein Synchronisationsdienst die Daten tatsächlich "+
				"speichert, lässt sich von hier aus nicht feststellen. Ein Synchronisationsordner ist "+
				"außerdem kein Sicherungsziel: er spiegelt auch das Löschen. Lege die Daten in einen "+
				"gewöhnlichen Ordner und sichere sie getrennt.", marker.name)
	}
	return ""
}
