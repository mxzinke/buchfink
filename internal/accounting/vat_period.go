package accounting

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Periodenzuordnung der Umsatzsteuer steht an dieser einen Stelle.
//
// Sie ist nicht das Buchungsdatum. Das Buchungsdatum sagt, wann erfasst wurde;
// die Voranmeldung fragt, wann die Steuer entstanden ist — und darauf gibt das
// UStG je nach Tatbestand eine andere Antwort. Solange die Auswertung nach dem
// Buchungsdatum aggregierte, war jede nachträglich erfasste Dezemberrechnung im
// Januar gemeldet. Deshalb: eine Funktion, eine Tabelle von Fällen, ein Test.

// VatPeriodFor liefert das Datum, nach dem eine Journalzeile ihrem
// Voranmeldungszeitraum zugeordnet wird.
//
// Der dritte Parameter ist der Belegeingang (`Receipt.ReceivedAt`). Er steht
// nicht an der Buchung, sondern am Beleg, und ohne ihn ließe sich der
// Vorsteuerfall nicht vollständig entscheiden: der Abzug setzt neben der
// Leistung den *Besitz* der Rechnung voraus (§ 15 Abs. 1 Satz 1 Nr. 1 Satz 2
// UStG). Ist kein Beleg erfasst, bleibt der Parameter leer und die Regel fällt
// auf Leistung und Rechnungsdatum zurück.
//
// Die Generalumkehr braucht keinen eigenen Zweig: sie übernimmt Belegdatum und
// Leistungszeitraum der Ursprungsbuchung, also ergeben dieselben Regeln
// denselben Zeitraum. Ist der bereits übermittelt, wird daraus ein Nachtrag —
// das entscheidet die Voranmeldung, nicht diese Funktion.
func VatPeriodFor(entry *domain.JournalEntry, line domain.JournalLine, receivedAt string) string {
	if entry == nil {
		return ""
	}
	switch timingOf(line.TaxKey) {
	case timingOutput:
		// § 13 Abs. 1 Nr. 1 Buchst. a UStG: die Steuer entsteht mit Ablauf des
		// Voranmeldungszeitraums, in dem die Leistung ausgeführt worden ist.
		return firstNonEmpty(entry.ServiceDateTo, entry.ServiceDateFrom, entry.DocumentDate, entry.BookingDate)

	case timingDocument:
		// Erwerbsteuer (§ 13 Abs. 1 Nr. 6 UStG) und § 13b-Steuer als Empfänger
		// (§ 13b Abs. 1 und 2 UStG) entstehen mit Ausstellung der Rechnung,
		// spätestens im Folgemonat der Leistung. Buchfink vereinfacht auf das
		// Belegdatum — die Rechnung liegt vor, sonst wäre nicht gebucht worden.
		//
		// Die Vorsteuer aus beiden folgt demselben Datum und nicht der Regel für
		// Rechnungsvorsteuer: § 15 Abs. 1 Satz 1 Nr. 3 und 4 UStG lässt sie in
		// dem Zeitraum abziehen, in dem die Steuer entstanden ist. Fielen die
		// beiden Beine auseinander, ergäbe ein Reverse-Charge-Beleg in einem
		// Monat eine Zahllast und im nächsten eine Erstattung, obwohl er
		// zusammen null ergibt.
		return firstNonEmpty(entry.DocumentDate, entry.ServiceDateTo, entry.BookingDate)

	case timingInput:
		// § 15 Abs. 1 Satz 1 Nr. 1 UStG: Leistung ausgeführt *und* Rechnung
		// vorhanden. Maßgeblich ist damit das spätere der beiden Daten — und
		// wenn der Beleg noch später eingegangen ist, dessen Eingang: vorher
		// konnte niemand den Abzug geltend machen.
		return latest(
			firstNonEmpty(entry.ServiceDateTo, entry.ServiceDateFrom),
			entry.DocumentDate,
			receivedAt,
		)
	}

	// Zeilen ohne Steuerschlüssel sind Erlös-, Aufwands- und Bestandszeilen. Für
	// die Voranmeldung zählen davon die Erlöse, und für sie gilt dieselbe Regel
	// wie für die Umsatzsteuer darauf: sonst stünden Umsatz und Steuer in
	// verschiedenen Monaten.
	return firstNonEmpty(entry.ServiceDateTo, entry.ServiceDateFrom, entry.DocumentDate, entry.BookingDate)
}

// vatTiming ordnet einen Steuerschlüssel seiner Entstehungsregel zu.
type vatTiming int

const (
	timingRevenue vatTiming = iota
	timingOutput
	timingDocument
	timingInput
)

func timingOf(taxKey string) vatTiming {
	switch {
	case taxKey == "":
		return timingRevenue
	case strings.HasPrefix(taxKey, "UST"):
		// UST19, UST7 und UST14C: geschuldete Umsatzsteuer des Leistenden.
		return timingOutput
	case strings.HasPrefix(taxKey, "VST"):
		return timingInput
	case strings.HasPrefix(taxKey, "IG"), strings.HasPrefix(taxKey, "RC"):
		return timingDocument
	}
	return timingRevenue
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func latest(values ...string) string {
	out := ""
	for _, v := range values {
		if v > out {
			out = v
		}
	}
	return out
}

// -------------------------------------------------------------
// Zeiträume
// -------------------------------------------------------------

// VatPeriod ist ein Voranmeldungszeitraum mit Schlüssel, Grenzen und Fälligkeit.
type VatPeriod struct {
	Key  string               `json:"key"`
	Type domain.VatPeriodType `json:"type"`
	// Label ist die Bezeichnung für die Oberfläche, z. B. "März 2026".
	Label string `json:"label"`
	From  string `json:"from"`
	To    string `json:"to"`
	Year  int    `json:"year"`
}

var monthNames = [...]string{
	"Januar", "Februar", "März", "April", "Mai", "Juni",
	"Juli", "August", "September", "Oktober", "November", "Dezember",
}

// VatPeriodsOfYear liefert die Zeiträume eines Kalenderjahres für einen
// Zeitraumtyp, in zeitlicher Reihenfolge.
func VatPeriodsOfYear(year int, periodType domain.VatPeriodType) []VatPeriod {
	switch periodType {
	case domain.VatPeriodMonth:
		out := make([]VatPeriod, 0, 12)
		for m := 1; m <= 12; m++ {
			out = append(out, monthPeriod(year, m))
		}
		return out
	case domain.VatPeriodQuarter:
		out := make([]VatPeriod, 0, 4)
		for q := 1; q <= 4; q++ {
			out = append(out, quarterPeriod(year, q))
		}
		return out
	case domain.VatPeriodYear:
		return []VatPeriod{yearPeriod(year)}
	}
	return nil
}

func monthPeriod(year, month int) VatPeriod {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, -1)
	return VatPeriod{
		Key:   fmt.Sprintf("%d-%02d", year, month),
		Type:  domain.VatPeriodMonth,
		Label: fmt.Sprintf("%s %d", monthNames[month-1], year),
		From:  start.Format("2006-01-02"),
		To:    end.Format("2006-01-02"),
		Year:  year,
	}
}

func quarterPeriod(year, quarter int) VatPeriod {
	start := time.Date(year, time.Month(quarter*3-2), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 3, -1)
	return VatPeriod{
		Key:   fmt.Sprintf("%d-Q%d", year, quarter),
		Type:  domain.VatPeriodQuarter,
		Label: fmt.Sprintf("%d. Quartal %d", quarter, year),
		From:  start.Format("2006-01-02"),
		To:    end.Format("2006-01-02"),
		Year:  year,
	}
}

func yearPeriod(year int) VatPeriod {
	return VatPeriod{
		Key:   fmt.Sprintf("%d", year),
		Type:  domain.VatPeriodYear,
		Label: fmt.Sprintf("Jahr %d", year),
		From:  fmt.Sprintf("%d-01-01", year),
		To:    fmt.Sprintf("%d-12-31", year),
		Year:  year,
	}
}

// ParseVatPeriodKey liest einen Zeitraumschlüssel: "2026-03", "2026-Q1", "2026".
func ParseVatPeriodKey(key string) (VatPeriod, error) {
	key = strings.TrimSpace(key)
	switch {
	case len(key) == 4:
		year, err := strconv.Atoi(key)
		if err != nil {
			return VatPeriod{}, fmt.Errorf("unbekannter Zeitraum %q", key)
		}
		return yearPeriod(year), nil

	case len(key) == 7 && key[4] == '-' && key[5] == 'Q':
		year, err := strconv.Atoi(key[:4])
		quarter, err2 := strconv.Atoi(key[6:])
		if err != nil || err2 != nil || quarter < 1 || quarter > 4 {
			return VatPeriod{}, fmt.Errorf("unbekannter Zeitraum %q", key)
		}
		return quarterPeriod(year, quarter), nil

	case len(key) == 7 && key[4] == '-':
		year, err := strconv.Atoi(key[:4])
		month, err2 := strconv.Atoi(key[5:])
		if err != nil || err2 != nil || month < 1 || month > 12 {
			return VatPeriod{}, fmt.Errorf("unbekannter Zeitraum %q", key)
		}
		return monthPeriod(year, month), nil
	}
	return VatPeriod{}, fmt.Errorf("unbekannter Zeitraum %q (erwartet 2026-03, 2026-Q1 oder 2026)", key)
}

// VatPeriodOf liefert den Zeitraum, in den ein Datum bei gegebenem Zeitraumtyp
// fällt.
func VatPeriodOf(date string, periodType domain.VatPeriodType) (VatPeriod, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return VatPeriod{}, fmt.Errorf("%q ist kein Datum (erwartet JJJJ-MM-TT)", date)
	}
	switch periodType {
	case domain.VatPeriodMonth:
		return monthPeriod(t.Year(), int(t.Month())), nil
	case domain.VatPeriodQuarter:
		return quarterPeriod(t.Year(), (int(t.Month())-1)/3+1), nil
	case domain.VatPeriodYear:
		return yearPeriod(t.Year()), nil
	}
	return VatPeriod{}, fmt.Errorf("unbekannter Voranmeldungszeitraum %q", periodType)
}

// VatDueDate ist die Fälligkeit einer Voranmeldung: der 10. Tag nach Ablauf des
// Zeitraums (§ 18 Abs. 1 Satz 1 UStG), mit Dauerfristverlängerung einen Monat
// später (§ 46 UStDV).
//
// Fällt der Tag auf einen Samstag oder Sonntag, verschiebt sich die Frist auf
// den nächsten Werktag (§ 108 Abs. 3 AO i. V. m. § 193 BGB). Feiertage bleiben
// bewusst außen vor: sie sind Landesrecht, und ein falsch geratener Feiertag
// wäre schlechter als eine Frist, die einen Tag zu früh mahnt.
func VatDueDate(p VatPeriod, permanentExtension bool) string {
	end, err := time.Parse("2006-01-02", p.To)
	if err != nil {
		return ""
	}
	due := time.Date(end.Year(), end.Month(), 10, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
	if permanentExtension {
		due = due.AddDate(0, 1, 0)
	}
	return NextWorkday(due).Format("2006-01-02")
}

// ZMDueDate ist die Fälligkeit der Zusammenfassenden Meldung: der 25. Tag nach
// Ablauf des Meldezeitraums (§ 18a Abs. 1 Satz 1 UStG). Die
// Dauerfristverlängerung gilt für sie nicht (§ 18a Abs. 1 Satz 5 UStG).
func ZMDueDate(p VatPeriod) string {
	end, err := time.Parse("2006-01-02", p.To)
	if err != nil {
		return ""
	}
	due := time.Date(end.Year(), end.Month(), 25, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
	return NextWorkday(due).Format("2006-01-02")
}

// NextWorkday schiebt einen Samstag oder Sonntag auf den folgenden Montag.
func NextWorkday(t time.Time) time.Time {
	switch t.Weekday() {
	case time.Saturday:
		return t.AddDate(0, 0, 2)
	case time.Sunday:
		return t.AddDate(0, 0, 1)
	}
	return t
}
