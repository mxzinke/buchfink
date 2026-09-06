// Package changelog hält die Versionshistorie des Programms.
//
// Sie ist eingebettet und nicht daneben abgelegt: eine Historie, die als Datei
// neben dem Programm liegt, fehlt in dem Moment, in dem sie gebraucht wird —
// bei der Prüfung einer Sicherung auf einem anderen Rechner. UNV-06 verlangt,
// dass sich zu jeder Programmfassung feststellen lässt, was sie geändert hat;
// dafür muss die Historie mit dem Programm reisen.
package changelog

import (
	_ "embed"
	"strings"
)

//go:embed CHANGELOG.md
var markdown string

// Markdown liefert die Änderungshistorie als Markdown. So geht sie ins
// Prüferpaket: als die Datei, die sie ist.
func Markdown() string { return markdown }

// Section rendert die Historie als Abschnitt zum Einbetten — in die
// Verfahrensdokumentation, die sie unter einer eigenen Überschrift führt.
//
// Nicht die Datei selbst: die trägt ihren eigenen Titel und einen Absatz über
// ihr Format, und beides gehört nicht in ein Dokument, das ein Prüfer liest.
// Die Fassungen stehen als fette Zeile und nicht als Überschrift, damit sie
// die Gliederung des einbettenden Dokuments nicht durchbrechen.
func Section() string {
	var out strings.Builder
	for i, entry := range Entries() {
		if i > 0 {
			out.WriteString("\n")
		}
		out.WriteString("**")
		out.WriteString(entry.Version)
		if entry.Date != "" {
			out.WriteString(" — ")
			out.WriteString(entry.Date)
		}
		out.WriteString("**\n\n")
		if entry.Summary != "" {
			out.WriteString(entry.Summary)
			out.WriteString("\n\n")
		}
		for _, change := range entry.Changes {
			out.WriteString("- ")
			out.WriteString(change)
			out.WriteString("\n")
		}
	}
	return out.String()
}

// Entry ist eine Fassung des Programms.
type Entry struct {
	// Version ist die Bezeichnung der Fassung, „v0.1".
	Version string `json:"version"`
	// Date ist der Tag der Fassung.
	Date string `json:"date"`
	// Summary sind die Sätze unter der Überschrift: wofür diese Fassung steht.
	// Kann leer sein.
	Summary string `json:"summary"`
	// Changes sind die einzelnen Änderungen.
	Changes []string `json:"changes"`
}

// Entries liefert die Fassungen in der Reihenfolge der Datei, die neueste
// zuerst.
//
// Gelesen wird hier und nicht in der Oberfläche: die Datei liegt in diesem
// Paket, und das Format einer Datei kennt das Paket, das sie hält. Eine
// Oberfläche, die Markdown zerlegt, zerlegt es beim nächsten Umbruch falsch.
//
// Erwartet wird je Fassung eine Überschrift „## <Fassung> — <Datum>", darunter
// ein Absatz und darunter Strichpunkte. Was vor der ersten Überschrift steht,
// beschreibt die Datei selbst und ist keine Fassung.
//
// Der Rückfall ist Auslassen und nicht Raten: eine Überschrift ohne Trennstrich
// gilt ganz als Fassungsbezeichnung, und Text außerhalb einer Fassung
// verschwindet. Das Markdown bleibt daneben vollständig verfügbar (Markdown),
// damit nichts endgültig verloren geht, was hier nicht durchkommt.
func Entries() []Entry {
	entries := make([]Entry, 0, 8)
	// summary sammelt die Absatzzeilen der laufenden Fassung, bis der erste
	// Strichpunkt kommt. Danach gehört Fließtext nicht mehr zur
	// Zusammenfassung, sondern zur Fortsetzung des letzten Punktes.
	var summary []string

	for _, raw := range strings.Split(markdown, "\n") {
		line := strings.TrimSpace(raw)

		if headline, ok := strings.CutPrefix(line, "## "); ok {
			flushSummary(entries, &summary)
			version, date := splitHeadline(headline)
			entries = append(entries, Entry{Version: version, Date: date, Changes: []string{}})
			summary = nil
			continue
		}
		// Alles vor der ersten Fassung beschreibt die Datei.
		if len(entries) == 0 {
			continue
		}
		current := &entries[len(entries)-1]

		if point, ok := strings.CutPrefix(line, "- "); ok {
			current.Changes = append(current.Changes, point)
			continue
		}
		if line == "" {
			continue
		}
		// Eine eingerückte Fortsetzung gehört zum Punkt darüber: die Datei
		// bricht lange Sätze um, und ein Umbruch ist keine neue Änderung.
		if len(current.Changes) > 0 {
			last := len(current.Changes) - 1
			current.Changes[last] += " " + line
			continue
		}
		summary = append(summary, line)
	}
	flushSummary(entries, &summary)
	return entries
}

// flushSummary hängt die gesammelten Absatzzeilen an die laufende Fassung.
func flushSummary(entries []Entry, summary *[]string) {
	if len(entries) == 0 || len(*summary) == 0 {
		return
	}
	entries[len(entries)-1].Summary = strings.Join(*summary, " ")
	*summary = nil
}

// splitHeadline trennt „v0.1 — 2026-09-06" in Fassung und Datum. Fehlt der
// Trennstrich, ist die ganze Überschrift die Fassungsbezeichnung.
func splitHeadline(headline string) (version, date string) {
	for _, dash := range []string{" — ", " – ", " - "} {
		if before, after, ok := strings.Cut(headline, dash); ok {
			return strings.TrimSpace(before), strings.TrimSpace(after)
		}
	}
	return strings.TrimSpace(headline), ""
}
