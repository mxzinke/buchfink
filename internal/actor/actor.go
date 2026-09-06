// Package actor benennt, wer gehandelt hat.
//
// GoBD Rz. 34 und der Grundsatz der Nachvollziehbarkeit verlangen, dass sich zu
// jeder Aufzeichnung und jeder Änderung feststellen lässt, wer sie veranlasst
// hat. Buchfink läuft am Einzelplatz und kennt keine Benutzerverwaltung — es
// gibt keine Anmeldung, an der eine Kennung hinge. Was es gibt, ist das Konto
// des Betriebssystems und der Name des Rechners, und das ist die
// Bearbeiterkennung: sie unterscheidet die Buchhalterin von ihrem Steuerberater,
// der dieselbe Datei auf seinem Rechner öffnet.
//
// Der Einzelplatzbetrieb wird in der Verfahrensdokumentation beschrieben; die
// Kennung ist dort die Zuordnung, die eine Benutzerverwaltung sonst leistete.
package actor

import (
	"os"
	"os/user"
	"strings"
	"sync"
)

// Unknown steht, wo weder Benutzer noch Rechner zu ermitteln sind.
//
// Ein leeres Feld wäre schlechter: es sähe aus wie „nicht erfasst" und ließe
// offen, ob die Kennung fehlt oder nie erhoben wurde.
const Unknown = "unbekannt"

var (
	once   sync.Once
	cached string
)

// Actor liefert die Bearbeiterkennung `<Benutzer>@<Rechner>`.
//
// Das Ergebnis wird einmal ermittelt und danach behalten: es steht an jeder
// Buchung und an jedem Protokolleintrag, und weder Benutzer noch Rechnername
// ändern sich, während das Programm läuft.
func Actor() string {
	once.Do(func() { cached = compute() })
	return cached
}

func compute() string {
	name := ""
	if u, err := user.Current(); err == nil {
		name = strings.TrimSpace(u.Username)
	}
	if name == "" {
		name = strings.TrimSpace(os.Getenv("USER"))
	}
	if name == "" {
		name = strings.TrimSpace(os.Getenv("USERNAME"))
	}

	host, err := os.Hostname()
	if err != nil {
		host = ""
	}
	host = strings.TrimSpace(host)

	return format(name, host)
}

// format setzt die Kennung aus Benutzer und Rechner zusammen.
//
// Die Form bleibt in jedem Fall `<Benutzer>@<Rechner>`, auch wenn eine der
// beiden Angaben fehlt: die Verfahrensdokumentation und die Feldbeschreibung
// des Exports beschreiben die Kennung als „Benutzerkonto und Rechnername", und
// ein Wert, der einmal `anna@buero-pc` und einmal nur `anna` lautet, ließe sich
// später weder gruppieren noch auseinanderhalten — ein Bearbeiter sähe aus wie
// zwei. Steht statt einer Angabe „unbekannt", ist am Wert ablesbar, welche der
// beiden fehlte.
func format(name, host string) string {
	if name == "" && host == "" {
		return Unknown
	}
	if name == "" {
		name = Unknown
	}
	if host == "" {
		host = Unknown
	}
	return name + "@" + host
}
