package accounting

import (
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// supply baut eine innergemeinschaftliche Lieferung an einen Empfänger.
func supply(id uint, number, serviceTo string, contactID uint, amount domain.Cents) domain.JournalEntry {
	return zmEntry(id, number, serviceTo, contactID, "4125", amount)
}

// service baut eine sonstige Leistung mit Steuerschuld des Empfängers.
func serviceSupply(id uint, number, serviceTo string, contactID uint, amount domain.Cents) domain.JournalEntry {
	return zmEntry(id, number, serviceTo, contactID, "4337", amount)
}

func zmEntry(id uint, number, serviceTo string, contactID uint, account string, amount domain.Cents) domain.JournalEntry {
	cid := contactID
	return domain.JournalEntry{
		ID: id, EntryNumber: number, FiscalYear: 2026,
		BookingDate: serviceTo, DocumentDate: serviceTo,
		ServiceDateFrom: serviceTo, ServiceDateTo: serviceTo,
		Description: "Testbuchung " + number,
		Source:      domain.EntrySourceInvoice,
		ContactID:   &cid,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "10001", Amount: amount, ContactID: &cid},
			{Side: domain.SideCredit, Account: account, Amount: amount},
		},
	}
}

func recipients(m map[uint]ZMRecipient) func(uint) ZMRecipient {
	return func(id uint) ZMRecipient { return m[id] }
}

// Die Meldung fasst je USt-IdNr. und Meldeart zusammen: derselbe Abnehmer mit
// zwei Rechnungen ergibt eine Zeile, Lieferung und sonstige Leistung an denselben
// Abnehmer zwei.
func TestZMLinesGroupByVatIDAndKind(t *testing.T) {
	people := map[uint]ZMRecipient{
		1: {Name: "Client SARL", CountryCode: "FR", VatID: "FR12345678901", IsEU: true},
		2: {Name: "Cliente SRL", CountryCode: "IT", VatID: "IT00743110157", IsEU: true},
	}
	movements := ZMMovements(ZMSource{
		Entries: []domain.JournalEntry{
			supply(1, "2026-000001", "2026-01-15", 1, 500000),
			supply(2, "2026-000002", "2026-02-20", 1, 300000),
			serviceSupply(3, "2026-000003", "2026-03-01", 1, 100000),
			supply(4, "2026-000004", "2026-03-10", 2, 250000),
		},
		Recipient: recipients(people),
	})

	period, err := ParseVatPeriodKey("2026-Q1")
	if err != nil {
		t.Fatal(err)
	}
	lines, findings := ZMLines(period, movements, recipients(people))
	if len(findings) != 0 {
		t.Fatalf("erwartet keine Befunde, erhalten %v", findings)
	}
	if len(lines) != 3 {
		t.Fatalf("erwartet drei Meldezeilen, erhalten %d: %+v", len(lines), lines)
	}

	got := map[string]domain.Cents{}
	for _, l := range lines {
		got[l.VatID+string(l.Kind)] = l.Amount
	}
	if got["FR12345678901L"] != 800000 {
		t.Errorf("Lieferungen an FR12345678901 = %s, erwartet 8.000,00", got["FR12345678901L"])
	}
	if got["FR12345678901S"] != 100000 {
		t.Errorf("Leistungen an FR12345678901 = %s, erwartet 1.000,00", got["FR12345678901S"])
	}
	if got["IT00743110157L"] != 250000 {
		t.Errorf("Lieferungen an IT00743110157 = %s, erwartet 2.500,00", got["IT00743110157L"])
	}

	for _, l := range lines {
		if len(l.EntryIDs) == 0 {
			t.Errorf("Zeile %s trägt keine Buchungs-IDs — ohne sie gibt es keinen Drill-down", l.VatID)
		}
	}
}

// Das Länderkennzeichen der Meldung kommt aus der USt-IdNr. und nicht aus dem
// Land des Kontakts.
//
// Bei Griechenland gehen beide auseinander: das Land heißt „GR", das Präfix der
// USt-IdNr. „EL" (Nordirland „XI" gegen „GB"). Nähme die Zeile das Kontaktland,
// stünde in der Datei „GR;EL123456789" statt „EL;123456789" — das BZSt-Portal
// nimmt das nicht an.
func TestZMCountryCodeComesFromTheVatID(t *testing.T) {
	people := map[uint]ZMRecipient{
		1: {Name: "Hellenic AE", CountryCode: "GR", VatID: "EL123456789", IsEU: true},
		2: {Name: "Ulster Ltd", CountryCode: "GB", VatID: "XI987654321", IsEU: true},
	}
	movements := ZMMovements(ZMSource{
		Entries: []domain.JournalEntry{
			supply(1, "2026-000001", "2026-01-15", 1, 500000),
			supply(2, "2026-000002", "2026-02-15", 2, 300000),
		},
		Recipient: recipients(people),
	})
	period, _ := ParseVatPeriodKey("2026-Q1")
	lines, findings := ZMLines(period, movements, recipients(people))
	if len(findings) != 0 {
		t.Fatalf("erwartet keine Befunde, erhalten %v", findings)
	}
	if len(lines) != 2 {
		t.Fatalf("erwartet zwei Meldezeilen, erhalten %d: %+v", len(lines), lines)
	}
	got := map[string]string{}
	for _, l := range lines {
		got[l.VatID] = l.CountryCode
	}
	if got["EL123456789"] != "EL" {
		t.Errorf("Länderkennzeichen zu EL123456789 = %q, erwartet \"EL\" statt des Kontaktlands \"GR\"", got["EL123456789"])
	}
	if got["XI987654321"] != "XI" {
		t.Errorf("Länderkennzeichen zu XI987654321 = %q, erwartet \"XI\" statt des Kontaktlands \"GB\"", got["XI987654321"])
	}
}

// Ohne USt-IdNr. ist der Umsatz nicht meldbar: § 18a Abs. 7 UStG verlangt sie,
// und die Steuerbefreiung der Lieferung hängt ebenfalls an ihr.
func TestZMMissingVatIDBecomesAFinding(t *testing.T) {
	people := map[uint]ZMRecipient{
		1: {Name: "Client SARL", CountryCode: "FR", VatID: "", IsEU: true},
	}
	movements := ZMMovements(ZMSource{
		Entries:   []domain.JournalEntry{supply(1, "2026-000001", "2026-01-15", 1, 500000)},
		Recipient: recipients(people),
	})
	period, _ := ParseVatPeriodKey("2026-Q1")
	lines, findings := ZMLines(period, movements, recipients(people))

	if len(lines) != 0 {
		t.Errorf("ohne USt-IdNr. entsteht keine Meldezeile, erhalten %d", len(lines))
	}
	if len(findings) != 1 || !strings.Contains(findings[0], "Client SARL") {
		t.Fatalf("erwartet einen Befund zu Client SARL, erhalten %v", findings)
	}
	if !strings.Contains(findings[0], "USt-IdNr") {
		t.Errorf("der Befund muss die fehlende USt-IdNr. benennen: %s", findings[0])
	}
}

// Eine Leistung an einen Empfänger im Drittland ist zwar nicht steuerbar, aber
// nicht meldepflichtig — die Zusammenfassende Meldung kennt nur das übrige
// Gemeinschaftsgebiet.
func TestZMIgnoresThirdCountryServices(t *testing.T) {
	people := map[uint]ZMRecipient{
		1: {Name: "US Corp", CountryCode: "US", VatID: "", IsEU: false},
	}
	movements := ZMMovements(ZMSource{
		Entries:   []domain.JournalEntry{serviceSupply(1, "2026-000001", "2026-01-15", 1, 500000)},
		Recipient: recipients(people),
	})
	if len(movements) != 0 {
		t.Errorf("erwartet keine meldepflichtigen Umsätze, erhalten %d", len(movements))
	}
}

// Der Regelfall ist das Quartal. Übersteigen die ig. Lieferungen 50.000 Euro,
// wird ab dem Quartal der Überschreitung monatlich gemeldet — und das bleibt
// vier Quartale lang so (§ 18a Abs. 1 Sätze 1 und 2 UStG).
func TestZMSwitchesToMonthlyAboveTheThreshold(t *testing.T) {
	people := map[uint]ZMRecipient{
		1: {Name: "Client SARL", CountryCode: "FR", VatID: "FR12345678901", IsEU: true},
	}

	below := ZMMovements(ZMSource{
		Entries:   []domain.JournalEntry{supply(1, "2026-000001", "2026-02-15", 1, 4_000_000)},
		Recipient: recipients(people),
	})
	periods := ZMPeriodsOfYear(2026, below)
	if len(periods) != 4 {
		t.Fatalf("unter der Grenze erwartet vier Quartalsmeldungen, erhalten %d", len(periods))
	}
	if periods[0].Key != "2026-Q1" {
		t.Errorf("erster Zeitraum = %s, erwartet 2026-Q1", periods[0].Key)
	}

	// 50.000,01 Euro im ersten Quartal: ab dem Quartal der Überschreitung
	// monatlich, und die vier folgenden Quartale ebenfalls.
	above := ZMMovements(ZMSource{
		Entries:   []domain.JournalEntry{supply(1, "2026-000001", "2026-02-15", 1, 5_000_001)},
		Recipient: recipients(people),
	})
	periods = ZMPeriodsOfYear(2026, above)
	if len(periods) != 12 {
		t.Fatalf("über der Grenze erwartet zwölf Monatsmeldungen, erhalten %d", len(periods))
	}
	for _, p := range periods {
		if p.Type != domain.VatPeriodMonth {
			t.Fatalf("Zeitraum %s ist %s, erwartet Monat", p.Key, p.Type)
		}
	}
}

// Die Rückschau reicht vier Quartale zurück: eine Überschreitung im Vorjahr
// wirkt in das laufende Jahr hinein und endet dann.
func TestZMThresholdLooksBackFourQuarters(t *testing.T) {
	people := map[uint]ZMRecipient{
		1: {Name: "Client SARL", CountryCode: "FR", VatID: "FR12345678901", IsEU: true},
	}
	movements := ZMMovements(ZMSource{
		// Überschreitung im vierten Quartal 2025.
		Entries:   []domain.JournalEntry{supply(1, "2025-000009", "2025-11-15", 1, 6_000_000)},
		Recipient: recipients(people),
	})

	periods := ZMPeriodsOfYear(2026, movements)
	monthly := 0
	for _, p := range periods {
		if p.Type == domain.VatPeriodMonth {
			monthly++
		}
	}
	// Q1 bis Q4 2026 liegen alle innerhalb von vier Quartalen nach Q4 2025.
	if monthly != 12 {
		t.Errorf("nach einer Überschreitung im Q4 2025 erwartet zwölf Monatsmeldungen 2026, erhalten %d", monthly)
	}

	// Ein Jahr weiter ist die Rückschau abgelaufen.
	periods = ZMPeriodsOfYear(2027, movements)
	for _, p := range periods {
		if p.Type != domain.VatPeriodQuarter {
			t.Errorf("2027 erwartet Quartalsmeldungen, %s ist %s", p.Key, p.Type)
		}
	}
}

// Für die Grenze zählen nur Lieferungen und Dreiecksgeschäfte, nicht die
// sonstigen Leistungen — § 18a Abs. 1 Satz 2 UStG nennt sie nicht.
func TestZMThresholdIgnoresServices(t *testing.T) {
	people := map[uint]ZMRecipient{
		1: {Name: "Client SARL", CountryCode: "FR", VatID: "FR12345678901", IsEU: true},
	}
	movements := ZMMovements(ZMSource{
		Entries:   []domain.JournalEntry{serviceSupply(1, "2026-000001", "2026-02-15", 1, 9_000_000)},
		Recipient: recipients(people),
	})
	for _, p := range ZMPeriodsOfYear(2026, movements) {
		if p.Type != domain.VatPeriodQuarter {
			t.Fatalf("sonstige Leistungen lösen die Monatsmeldung nicht aus, %s ist %s", p.Key, p.Type)
		}
	}
}
