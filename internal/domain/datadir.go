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
// Der Hinweis hält nichts an, und das ist Absicht: § 146 Abs. 2 AO verlangt die
// Führung der Bücher im Inland, § 146 Abs. 2a AO lässt die Verlagerung ins
// Ausland nur mit Bewilligung des Finanzamts zu — und wo ein
// Synchronisationsdienst die Daten tatsächlich ablegt, weiß Buchfink nicht.
// Was es weiß, ist, dass die Frage sich stellt. Sie zu stellen ist besser, als
// sie stillschweigend mit „wird schon passen" zu beantworten.
func CloudFolderWarning(path string) string {
	lowered := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	for _, marker := range cloudFolderMarkers {
		if !strings.Contains(lowered, marker.needle) {
			continue
		}
		return fmt.Sprintf(
			"Der Datenordner liegt in einem %s-Ordner. Buchführung ist grundsätzlich im Inland zu "+
				"führen; die Verlagerung elektronischer Bücher ins Ausland bedarf der Bewilligung des "+
				"Finanzamts (§ 146 Abs. 2, 2a AO). Wo ein Synchronisationsdienst die Daten tatsächlich "+
				"speichert, lässt sich von hier aus nicht feststellen. Ein Synchronisationsordner ist "+
				"außerdem kein Sicherungsziel: er spiegelt auch das Löschen. Lege die Daten in einen "+
				"gewöhnlichen Ordner und sichere sie getrennt.", marker.name)
	}
	return ""
}
