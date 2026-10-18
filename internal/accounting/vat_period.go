package accounting

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/calendar"
	"github.com/buchfink/buchfink/internal/domain"
)

// VatPeriodFor liefert das Datum der Steuerentstehung für eine Journalzeile.
// receivedAt ist der Belegeingang; Zahlungen vor der Leistung berücksichtigt
// die Auswertung anhand der Zahlungszuordnungen (vat_prepayment.go).
func VatPeriodFor(entry *domain.JournalEntry, line domain.JournalLine, receivedAt string) string {
	if entry == nil {
		return ""
	}
	return TaxCaseFor(entry.TaxTreatment).Timing(entry, line, receivedAt)
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

// VatStatutoryDate ist das gesetzliche Ende der Voranmeldungsfrist: der 10. Tag
// nach Ablauf des Zeitraums (§ 18 Abs. 1 Satz 1 UStG), mit
// Dauerfristverlängerung einen Monat später (§ 46 UStDV).
func VatStatutoryDate(p VatPeriod, permanentExtension bool) (time.Time, bool) {
	end, err := time.Parse("2006-01-02", p.To)
	if err != nil {
		return time.Time{}, false
	}
	due := time.Date(end.Year(), end.Month(), 10, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
	if permanentExtension {
		due = due.AddDate(0, 1, 0)
	}
	return due, true
}

// VatDueDate ist die Fälligkeit einer Voranmeldung mit den bundesweiten
// Feiertagen; VatDueDateIn kennt auch die des Landes.
func VatDueDate(p VatPeriod, permanentExtension bool) string {
	return VatDueDateIn(p, permanentExtension, "")
}

// VatDueDateIn ist die Fälligkeit einer Voranmeldung im Land state: das
// gesetzliche Ende, verschoben nach § 108 Abs. 3 AO.
func VatDueDateIn(p VatPeriod, permanentExtension bool, state string) string {
	due, ok := VatStatutoryDate(p, permanentExtension)
	if !ok {
		return ""
	}
	return calendar.Format(calendar.NextWorkday(due, state))
}

// ZMStatutoryDate ist das gesetzliche Ende der Meldefrist der Zusammenfassenden
// Meldung: der 25. Tag nach Ablauf des Meldezeitraums (§ 18a Abs. 2 Satz 1
// UStG). Die Dauerfristverlängerung gilt für sie nicht (§ 46 UStDV).
func ZMStatutoryDate(p VatPeriod) (time.Time, bool) {
	end, err := time.Parse("2006-01-02", p.To)
	if err != nil {
		return time.Time{}, false
	}
	return time.Date(end.Year(), end.Month(), 25, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0), true
}

// ZMDueDate ist die Fälligkeit der Zusammenfassenden Meldung mit den
// bundesweiten Feiertagen.
func ZMDueDate(p VatPeriod) string { return ZMDueDateIn(p, "") }

// ZMDueDateIn ist die Fälligkeit der Zusammenfassenden Meldung im Land state.
func ZMDueDateIn(p VatPeriod, state string) string {
	due, ok := ZMStatutoryDate(p)
	if !ok {
		return ""
	}
	return calendar.Format(calendar.NextWorkday(due, state))
}

// NextWorkday schiebt einen Tag nach § 108 Abs. 3 AO auf den nächsten Werktag;
// Feiertage sind die bundesweiten.
func NextWorkday(t time.Time) time.Time { return calendar.NextWorkday(t, "") }
