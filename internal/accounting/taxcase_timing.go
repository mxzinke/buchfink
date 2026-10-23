package accounting

import (
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// Timing bestimmt den Voranmeldungszeitraum nach dem gespeicherten Schlüssel
// der Zeile. Auch unbekannte Präfixe und Skontokorrekturen behalten ihre
// Entstehungsregel. receivedAt ist der Eingang des Belegs, der für den
// Vorsteuerabzug neben Leistung und Rechnung maßgeblich ist.
func (c TaxCase) Timing(entry *domain.JournalEntry, line domain.JournalLine, receivedAt string) string {
	if entry == nil {
		return ""
	}
	// Die Rücknahme einer Ausgangsrechnung wirkt im Zeitraum der Berichtigung
	// und nicht im Ursprungszeitraum.
	//
	// § 17 Abs. 1 Satz 8 UStG: die Berichtigung ist für den Besteuerungszeitraum
	// vorzunehmen, in dem die Änderung eingetreten ist. Für die zu Unrecht oder
	// zu hoch ausgewiesene Steuer sagt Abschn. 14c.1 Abs. 5 UStAE dasselbe: der
	// Mehrbetrag bleibt bis zur Berichtigung der Rechnung geschuldet, und die
	// Berichtigung wirkt im Zeitraum, in dem sie gegenüber dem Empfänger
	// erklärt wird. Beides führt auf den Tag des Stornodokuments, den die
	// Generalumkehr als Buchungsdatum hat.
	if entry.Kind == domain.EntryKindReversal && entry.Source == domain.EntrySourceInvoice {
		return firstNonEmpty(entry.BookingDate, entry.DocumentDate)
	}
	// Die vereinnahmte Anzahlung folgt dem Geld und nicht der Leistung: nach
	// § 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG entsteht die Steuer mit Ablauf des
	// Voranmeldungszeitraums, in dem das Teilentgelt vereinnahmt worden ist —
	// auch bei Sollversteuerung, und lange bevor die Leistung erbracht ist. Ohne
	// diesen Zweig entschiede der Leistungszeitraum, und der steht bei einer
	// Anzahlung noch in der Zukunft.
	if entry.Source == domain.EntrySourceAdvance {
		return firstNonEmpty(entry.BookingDate, entry.DocumentDate)
	}
	switch timingOf(line.TaxKey) {
	case timingOutput:
		// § 13 Abs. 1 Nr. 1 Buchst. a UStG: die Steuer entsteht mit Ablauf des
		// Voranmeldungszeitraums, in dem die Leistung ausgeführt worden ist.
		return firstNonEmpty(entry.ServiceDateTo, entry.ServiceDateFrom, entry.DocumentDate, entry.BookingDate)

	case timingReverseChargeEU:
		// § 13b Abs. 1 UStG: die Steuer für die sonstige Leistung eines
		// EU-Unternehmers entsteht mit Ablauf des Voranmeldungszeitraums der
		// Leistung, unabhängig vom Rechnungsdatum. Die Vorsteuer daraus ist im
		// selben Zeitraum abziehbar (§ 15 Abs. 1 Satz 1 Nr. 4 UStG).
		return firstNonEmpty(entry.ServiceDateTo, entry.ServiceDateFrom, entry.DocumentDate, entry.BookingDate)

	case timingDocument:
		// Erwerbsteuer (§ 13 Abs. 1 Nr. 6 UStG) und § 13b-Steuer nach Abs. 2
		// entstehen mit Ausstellung der Rechnung, spätestens mit Ablauf des
		// Folgemonats der Leistung. Die Vorsteuer daraus folgt demselben Datum
		// (§ 15 Abs. 1 Satz 1 Nr. 3 und 4 UStG): fielen die beiden Beine
		// auseinander, ergäbe ein Beleg, der zusammen null ergibt, in einem
		// Monat eine Zahllast und im nächsten eine Erstattung.
		return documentOrFollowingMonth(entry)

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
	timingReverseChargeEU
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
	case strings.HasPrefix(taxKey, TaxKeyReverseChargeEUPrefix):
		return timingReverseChargeEU
	case strings.HasPrefix(taxKey, "IG"), strings.HasPrefix(taxKey, TaxKeyReverseChargePrefix):
		return timingDocument
	}
	return timingRevenue
}

// documentOrFollowingMonth ist das Rechnungsdatum, höchstens aber der letzte
// Tag des Monats nach dem Ende der Leistung. Ohne Rechnungsdatum entscheidet
// der Folgemonat, ohne Leistungsdatum das Rechnungsdatum.
func documentOrFollowingMonth(entry *domain.JournalEntry) string {
	serviceEnd := firstNonEmpty(entry.ServiceDateTo, entry.ServiceDateFrom)
	document := entry.DocumentDate
	latestDate := endOfFollowingMonth(serviceEnd)
	switch {
	case document == "" && latestDate == "":
		return entry.BookingDate
	case document == "":
		return latestDate
	case latestDate == "" || document < latestDate:
		return document
	}
	return latestDate
}

// endOfFollowingMonth liefert den letzten Tag des Monats nach dem Datum, ""
// bei einem ungültigen Datum.
func endOfFollowingMonth(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return ""
	}
	first := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	return first.AddDate(0, 2, -1).Format("2006-01-02")
}

// prepaymentDate bestimmt, ob und wann eine Zahlung vor der Leistung wirkt.
// Die Vorsteuer aus einer Rechnung setzt deren Besitz voraus; für § 13b und
// die Umsatzsteuer entscheidet der Zahlungstag (§ 13 Abs. 1, § 13b Abs. 4,
// § 15 Abs. 1 UStG).
func (v TaxCaseVAT) prepaymentDate(documentDate, receivedAt string) func(string) string {
	switch v.Prepayment {
	case TaxPrepaymentOnPayment:
		return func(payment string) string { return payment }
	case TaxPrepaymentWithInvoice:
		return func(payment string) string { return latest(payment, documentDate, receivedAt) }
	default:
		return nil
	}
}
