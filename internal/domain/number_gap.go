package domain

import "context"

// Der Lückenbericht des Rechnungsnummernkreises.
//
// § 14 Abs. 4 Nr. 4 UStG verlangt eine einmalige, fortlaufende Nummer, und die
// Betriebsprüfung fragt zuerst nach den fehlenden. Seit Nummer, Rechnung und
// Buchung in einer Transaktion entstehen, kann eine Nummer nicht mehr durch
// eine gescheiterte Buchung verloren gehen — der Rollback gibt sie zurück.
// Übrig bleiben zwei Fälle: ein Absturz zwischen Zählerstand und Commit, und
// Daten aus der Zeit davor. Beide sind erklärungsbedürftig, und die Erklärung
// gehört dokumentiert statt mündlich vorgetragen.

// NumberGapReason is why a number of the series carries no document.
type NumberGapReason string

const (
	// NumberGapAborted: die Vergabe ist abgebrochen, nachdem der Zähler
	// weitergezählt hatte.
	NumberGapAborted NumberGapReason = "aborted"
	// NumberGapTest: die Nummer wurde bei einem Probelauf verbraucht.
	NumberGapTest NumberGapReason = "test"
	// NumberGapCancelled: die Rechnung wurde storniert und aus dem Bestand
	// entfernt — ein Fall, den Buchfink selbst nicht erzeugt, den es aber in
	// übernommenen Beständen gibt.
	NumberGapCancelled NumberGapReason = "cancelled"
	// NumberGapUnknown: die Lücke ist da und niemand hat sie begründet.
	NumberGapUnknown NumberGapReason = "unknown"
)

// Label ist der Klartext für Bericht und Oberfläche.
func (r NumberGapReason) Label() string {
	switch r {
	case NumberGapAborted:
		return "Abbruch bei der Vergabe"
	case NumberGapTest:
		return "Probelauf"
	case NumberGapCancelled:
		return "Storniert und entfernt"
	default:
		return "Nicht dokumentiert"
	}
}

// NumberGapReasonOption is one reason to choose from when documenting a gap.
//
// Die Beschriftung kommt aus `Label()` und damit aus derselben Quelle wie die
// des Berichts: Der Grund, den der Anwender auswählt, und der Grund, den die
// Zeile danach zeigt, sind dasselbe Wort.
type NumberGapReasonOption struct {
	Reason NumberGapReason `json:"reason"`
	Label  string          `json:"label"`
}

// NumberGapReasons lists the reasons a gap can be documented with.
func NumberGapReasons() []NumberGapReasonOption {
	reasons := []NumberGapReason{
		NumberGapAborted, NumberGapTest, NumberGapCancelled, NumberGapUnknown,
	}
	out := make([]NumberGapReasonOption, 0, len(reasons))
	for _, reason := range reasons {
		out = append(out, NumberGapReasonOption{Reason: reason, Label: reason.Label()})
	}
	return out
}

// NumberGap is a recorded reason for a missing number of a series.
type NumberGap struct {
	ID         uint            `gorm:"primaryKey" json:"id"`
	Key        NumberRangeKey  `gorm:"size:30;not null;index" json:"key"`
	FiscalYear int             `gorm:"index;not null" json:"fiscalYear"`
	Sequence   int64           `gorm:"not null;index" json:"sequence"`
	Number     string          `gorm:"size:50;not null" json:"number"`
	Reason     NumberGapReason `gorm:"size:20;not null" json:"reason"`
	// Detail is the free-text explanation, e.g. the error the failed attempt
	// produced. A reason code alone answers "why" only in the coarsest way.
	Detail string `gorm:"size:500;serializer:encrypted" json:"detail,omitempty"`
	// RecordedAt ist der Zeitpunkt in ISO-8601, an dem die Lücke vermerkt wurde.
	RecordedAt string `gorm:"size:25;not null" json:"recordedAt"`
}

// NumberGapEntry is one line of the report: the missing number with whatever is
// known about it.
type NumberGapEntry struct {
	Sequence   int64           `json:"sequence"`
	Number     string          `json:"number"`
	Reason     NumberGapReason `json:"reason"`
	Label      string          `json:"label"`
	Detail     string          `json:"detail,omitempty"`
	RecordedAt string          `json:"recordedAt,omitempty"`
}

// NumberGapReport is the answer to "which invoice numbers are missing".
type NumberGapReport struct {
	FiscalYear int `json:"fiscalYear"`
	// Issued is how many numbers the counter handed out, Used how many of them
	// carry a document.
	Issued int64            `json:"issued"`
	Used   int              `json:"used"`
	Gaps   []NumberGapEntry `json:"gaps"`
}

// NumberGapRepository persists the recorded reasons.
type NumberGapRepository interface {
	Record(ctx context.Context, gap *NumberGap) error
	FindByYear(ctx context.Context, key NumberRangeKey, fiscalYear int) ([]NumberGap, error)
}

// BuildNumberGapReport compares the counter against the numbers actually in
// use.
//
// The counter is the authority for how many numbers were handed out: it is the
// one value that a deleted row cannot change. Everything between 1 and the last
// handed-out number that no document carries is a gap, and a recorded reason is
// attached where one exists.
func BuildNumberGapReport(fiscalYear int, nextSequence int64, numbers []string, recorded []NumberGap, format string) NumberGapReport {
	used := map[int64]bool{}
	for _, n := range numbers {
		if seq, ok := ParseInvoiceSequence(n, fiscalYear, format); ok {
			used[seq] = true
		}
	}
	reasons := map[int64]NumberGap{}
	for _, g := range recorded {
		reasons[g.Sequence] = g
	}

	report := NumberGapReport{
		FiscalYear: fiscalYear,
		Issued:     nextSequence - 1,
		Used:       len(numbers),
		Gaps:       make([]NumberGapEntry, 0),
	}
	for seq := int64(1); seq < nextSequence; seq++ {
		if used[seq] {
			continue
		}
		entry := NumberGapEntry{
			Sequence: seq,
			Number:   FormatInvoiceNumberWith(format, fiscalYear, seq),
			Reason:   NumberGapUnknown,
		}
		if g, ok := reasons[seq]; ok {
			entry.Reason = g.Reason
			entry.Detail = g.Detail
			entry.RecordedAt = g.RecordedAt
			if g.Number != "" {
				entry.Number = g.Number
			}
		}
		entry.Label = entry.Reason.Label()
		report.Gaps = append(report.Gaps, entry)
	}
	return report
}
