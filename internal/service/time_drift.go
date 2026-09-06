package service

import (
	"fmt"
	"time"
)

// TimeDriftTolerance ist die Abweichung, bis zu der Systemzeit und beglaubigte
// Zeit als dieselbe gelten.
//
// Fünf Minuten sind großzügig für eine Uhr, die per NTP läuft, und eng genug,
// um eine von Hand verstellte zu bemerken. Der Weg zum Zeitstempeldienst und
// zurück braucht Sekunden, nicht Minuten; alles darüber ist die Uhr und nicht
// das Netz.
const TimeDriftTolerance = 5 * time.Minute

// TimeDriftNote vergleicht die Systemzeit mit der beglaubigten Zeit des
// Zeitstempeldienstes und beschreibt eine nennenswerte Abweichung.
//
// Leer heißt: die beiden Zeiten stimmen im Rahmen der Toleranz überein.
//
// Der Vergleich gehört zur Festschreibung, weil dort ohnehin eine beglaubigte
// Zeit eintrifft — es ist die einzige Zeitangabe im ganzen Programm, die nicht
// von der Uhr dieses Rechners stammt. Geht die Uhr falsch, tragen sämtliche
// Buchungen und Protokolleinträge einen Zeitpunkt, den es nicht gab, und die
// zeitgerechte Erfassung (§ 146 Abs. 1 AO) ließe sich nicht mehr belegen.
func TimeDriftNote(system, trusted time.Time, tsaName string) string {
	drift := system.UTC().Sub(trusted.UTC())
	if drift < 0 {
		drift = -drift
	}
	if drift <= TimeDriftTolerance {
		return ""
	}

	direction := "geht vor"
	if system.UTC().Before(trusted.UTC()) {
		direction = "geht nach"
	}
	name := tsaName
	if name == "" {
		name = "dem Zeitstempeldienst"
	}
	return fmt.Sprintf(
		"Die Uhr dieses Rechners %s um %s gegenüber %s. Prüfe die Systemzeit — "+
			"Buchungs- und Protokollzeitpunkte hängen an ihr.",
		direction, formatDrift(drift), name)
}

// formatDrift schreibt die Abweichung in der größten sinnvollen Einheit.
func formatDrift(d time.Duration) string {
	switch {
	case d >= 48*time.Hour:
		return fmt.Sprintf("%d Tage", int(d.Hours()/24))
	case d >= 2*time.Hour:
		return fmt.Sprintf("%d Stunden", int(d.Hours()))
	default:
		return fmt.Sprintf("%d Minuten", int(d.Minutes()))
	}
}
