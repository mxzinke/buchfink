package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Periodenzuordnung ist der Kern der Voranmeldung: sie entscheidet, in
// welchem Monat ein Umsatz gemeldet wird. Vorher war es das Buchungsdatum, und
// damit war jede nachträglich erfasste Dezemberrechnung im Januar gemeldet.
func TestVatPeriodForFollowsTheTaxEvent(t *testing.T) {
	entry := func(booking, document, serviceTo string) *domain.JournalEntry {
		return &domain.JournalEntry{
			BookingDate:     booking,
			DocumentDate:    document,
			ServiceDateFrom: serviceTo,
			ServiceDateTo:   serviceTo,
		}
	}

	cases := []struct {
		name       string
		entry      *domain.JournalEntry
		taxKey     string
		receivedAt string
		want       string
		reason     string
	}{
		{
			name:   "Umsatzsteuer nach dem Leistungsende",
			entry:  entry("2026-04-20", "2026-04-02", "2026-03-31"),
			taxKey: "UST19",
			want:   "2026-03-31",
			reason: "§ 13 Abs. 1 Nr. 1 Buchst. a UStG stellt auf die Ausführung der Leistung ab",
		},
		{
			name:   "Erlöszeile folgt der Steuer darauf",
			entry:  entry("2026-04-20", "2026-04-02", "2026-03-31"),
			taxKey: "",
			want:   "2026-03-31",
			reason: "Umsatz und Steuer darauf gehören in denselben Zeitraum",
		},
		{
			name:   "Erwerbsteuer nach dem Belegdatum",
			entry:  entry("2026-05-02", "2026-04-10", "2026-03-20"),
			taxKey: "IG19_UST",
			want:   "2026-04-10",
			reason: "§ 13 Abs. 1 Nr. 6 UStG stellt auf die Ausstellung der Rechnung ab",
		},
		{
			name:   "Vorsteuer aus dem Erwerb folgt demselben Datum",
			entry:  entry("2026-05-02", "2026-04-10", "2026-03-20"),
			taxKey: "IG19_VST",
			want:   "2026-04-10",
			reason: "§ 15 Abs. 1 Satz 1 Nr. 3 UStG lässt den Abzug im Zeitraum der Entstehung zu",
		},
		{
			name:   "§ 13b-Steuer nach dem Belegdatum",
			entry:  entry("2026-05-02", "2026-04-10", "2026-03-20"),
			taxKey: "RC19_UST",
			want:   "2026-04-10",
			reason: "§ 13b Abs. 1 UStG stellt auf die Rechnungsausstellung ab",
		},
		{
			name:   "Vorsteuer aus § 13b folgt demselben Datum",
			entry:  entry("2026-05-02", "2026-04-10", "2026-03-20"),
			taxKey: "RC19_VST",
			want:   "2026-04-10",
			reason: "sonst ergäbe ein § 13b-Beleg in einem Monat Zahllast und im nächsten Erstattung",
		},
		{
			name:   "Rechnungsvorsteuer: Rechnung später als die Leistung",
			entry:  entry("2026-05-02", "2026-04-03", "2026-03-31"),
			taxKey: "VST19",
			want:   "2026-04-03",
			reason: "§ 15 Abs. 1 Satz 1 Nr. 1 UStG verlangt Leistung und Rechnung",
		},
		{
			name:   "Rechnungsvorsteuer: Leistung später als die Rechnung",
			entry:  entry("2026-05-02", "2026-03-01", "2026-04-30"),
			taxKey: "VST19",
			want:   "2026-04-30",
			reason: "die Vorauszahlungsrechnung allein berechtigt nicht zum Abzug",
		},
		{
			name:       "Rechnungsvorsteuer: Beleg erst später eingegangen",
			entry:      entry("2026-06-01", "2026-04-03", "2026-03-31"),
			taxKey:     "VST19",
			receivedAt: "2026-05-12",
			want:       "2026-05-12",
			reason:     "vor dem Besitz der Rechnung konnte niemand den Abzug geltend machen",
		},
		{
			name:       "Ein früherer Belegeingang verschiebt nichts",
			entry:      entry("2026-06-01", "2026-04-03", "2026-03-31"),
			taxKey:     "VST19",
			receivedAt: "2026-04-01",
			want:       "2026-04-03",
			reason:     "maßgeblich ist das späteste der drei Daten",
		},
		{
			name:   "§ 14c-Steuer folgt dem Ausgangsumsatz",
			entry:  entry("2026-04-20", "2026-04-02", "2026-03-31"),
			taxKey: TaxKeyUnlawful,
			want:   "2026-03-31",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := VatPeriodFor(tc.entry, domain.JournalLine{TaxKey: tc.taxKey}, tc.receivedAt)
			if got != tc.want {
				t.Errorf("Zeitraumdatum = %s, erwartet %s (%s)", got, tc.want, tc.reason)
			}
		})
	}
}

// Die Generalumkehr wirkt in dem Zeitraum, dem die Ursprungsbuchung zugeordnet
// war — sie übernimmt Beleg- und Leistungsdatum und trägt nur ihr eigenes
// Buchungsdatum. Ohne diese Eigenschaft stünde die Korrektur im Monat ihrer
// Erfassung und der ursprüngliche Umsatz bliebe im alten Monat stehen.
func TestVatPeriodForReversalKeepsTheOriginalPeriod(t *testing.T) {
	original := &domain.JournalEntry{
		BookingDate: "2026-03-15", DocumentDate: "2026-03-10",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-03-05",
	}
	reversal := &domain.JournalEntry{
		Kind:        domain.EntryKindReversal,
		BookingDate: "2026-07-20", // Datum der Korrektur
		// Generalumkehr übernimmt die Daten der Ursprungsbuchung.
		DocumentDate:    original.DocumentDate,
		ServiceDateFrom: original.ServiceDateFrom, ServiceDateTo: original.ServiceDateTo,
	}

	line := domain.JournalLine{TaxKey: "UST19"}
	if got := VatPeriodFor(reversal, line, ""); got != VatPeriodFor(original, line, "") {
		t.Errorf("die Generalumkehr wirkt zum %s, die Ursprungsbuchung zum %s — beide gehören in denselben Zeitraum",
			got, VatPeriodFor(original, line, ""))
	}
}

func TestVatPeriodKeys(t *testing.T) {
	cases := []struct {
		key      string
		wantFrom string
		wantTo   string
		wantType domain.VatPeriodType
	}{
		{"2026-03", "2026-03-01", "2026-03-31", domain.VatPeriodMonth},
		{"2026-02", "2026-02-01", "2026-02-28", domain.VatPeriodMonth},
		{"2026-Q1", "2026-01-01", "2026-03-31", domain.VatPeriodQuarter},
		{"2026-Q4", "2026-10-01", "2026-12-31", domain.VatPeriodQuarter},
		{"2026", "2026-01-01", "2026-12-31", domain.VatPeriodYear},
	}
	for _, tc := range cases {
		p, err := ParseVatPeriodKey(tc.key)
		if err != nil {
			t.Fatalf("%s: %v", tc.key, err)
		}
		if p.From != tc.wantFrom || p.To != tc.wantTo || p.Type != tc.wantType {
			t.Errorf("%s = %s bis %s (%s), erwartet %s bis %s (%s)",
				tc.key, p.From, p.To, p.Type, tc.wantFrom, tc.wantTo, tc.wantType)
		}
	}
	if _, err := ParseVatPeriodKey("Quartal 1"); err == nil {
		t.Error("ein unbekannter Zeitraumschlüssel muss abgewiesen werden")
	}
}

// Die Voranmeldung ist am 10. Tag nach Ablauf des Zeitraums fällig; die
// Dauerfristverlängerung schiebt sie um einen Monat. Fällt der Tag auf ein
// Wochenende, gilt der nächste Werktag.
func TestVatDueDate(t *testing.T) {
	cases := []struct {
		key       string
		extension bool
		want      string
		reason    string
	}{
		{"2026-01", false, "2026-02-10", "10. Februar 2026 ist ein Dienstag"},
		{"2026-01", true, "2026-03-10", "mit Dauerfrist einen Monat später"},
		{"2026-04", false, "2026-05-11", "der 10. Mai 2026 ist ein Sonntag"},
		{"2026-Q1", false, "2026-04-10", "Quartal endet am 31. März"},
		{"2026-Q4", true, "2027-02-10", "Jahreswechsel mit Dauerfrist"},
	}
	for _, tc := range cases {
		p, err := ParseVatPeriodKey(tc.key)
		if err != nil {
			t.Fatalf("%s: %v", tc.key, err)
		}
		if got := VatDueDate(p, tc.extension); got != tc.want {
			t.Errorf("%s (Dauerfrist %v) fällig am %s, erwartet %s (%s)", tc.key, tc.extension, got, tc.want, tc.reason)
		}
	}
}

// Die Zusammenfassende Meldung ist am 25. Tag nach Ablauf des Meldezeitraums
// fällig — und die Dauerfristverlängerung gilt für sie nicht.
func TestZMDueDate(t *testing.T) {
	p, err := ParseVatPeriodKey("2026-Q1")
	if err != nil {
		t.Fatal(err)
	}
	if got := ZMDueDate(p); got != "2026-04-27" {
		// Der 25. April 2026 ist ein Samstag.
		t.Errorf("ZM für Q1 2026 fällig am %s, erwartet 2026-04-27 (der 25. ist ein Samstag)", got)
	}
}
