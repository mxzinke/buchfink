package wailsbridge

// emptyList ersetzt eine nicht belegte Liste durch eine leere.
//
// Die Bridge geht als JSON an die Oberfläche, und ein nicht belegter Go-Slice
// wird dort zu `null`. Die Masken lesen die Antworten ohne Umweg — `rows.map`,
// `paths.length` —, und `null.map` wirft im Render einen TypeError, der ohne
// ErrorBoundary den ganzen Baum mitnimmt. Betroffen ist jeweils der Randfall,
// den niemand von Hand ausprobiert: der abgebrochene Dateidialog, das Jahr ohne
// einen einzigen Treffer, der Kontakt ohne Abfrage.
//
// Die Signatur nimmt das Ergebnispaar eines Aufrufs, damit ein durchreichendes
// `return svc.X(…)` in der Bridge eine Klammer braucht und keinen
// Zwischenschritt. Auch im Fehlerfall kommt eine leere Liste zurück: der Fehler
// wird geworfen, aber die Antwort bleibt eine Liste.
func emptyList[T any](items []T, err error) ([]T, error) {
	if items == nil {
		return make([]T, 0), err
	}
	return items, err
}
