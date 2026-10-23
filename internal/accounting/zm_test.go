package accounting

import (
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// serviceSupply baut eine sonstige Leistung mit Steuerschuld des Empfängers.
func serviceSupply(id uint, number, serviceTo string, contactID uint, amount domain.Cents) domain.JournalEntry {
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
			{Side: domain.SideCredit, Account: "4337", Amount: amount},
		},
	}
}

func recipients(m map[uint]ZMRecipient) func(uint) ZMRecipient {
	return func(id uint) ZMRecipient { return m[id] }
}

// Die Meldung fasst je USt-IdNr. zusammen: derselbe Empfänger mit zwei
// Rechnungen ergibt eine Zeile.
func TestZMLinesGroupByVatID(t *testing.T) {
	people := map[uint]ZMRecipient{
		1: {Name: "Client SARL", CountryCode: "FR", VatID: "FR12345678901", IsEU: true},
		2: {Name: "Cliente SRL", CountryCode: "IT", VatID: "IT00743110157", IsEU: true},
	}
	movements := ZMMovements(ZMSource{
		Entries: []domain.JournalEntry{
			serviceSupply(1, "2026-000001", "2026-01-15", 1, 500000),
			serviceSupply(2, "2026-000002", "2026-02-20", 1, 300000),
			serviceSupply(3, "2026-000003", "2026-03-10", 2, 250000),
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
	if len(lines) != 2 {
		t.Fatalf("erwartet zwei Meldezeilen, erhalten %d: %+v", len(lines), lines)
	}

	got := map[string]domain.Cents{}
	for _, l := range lines {
		if l.Kind != domain.ZMKindService {
			t.Errorf("Zeile %s hat die Meldeart %q, erwartet %q", l.VatID, l.Kind, domain.ZMKindService)
		}
		got[l.VatID] = l.Amount
		if len(l.EntryIDs) == 0 {
			t.Errorf("Zeile %s hat keine Buchungs-IDs — ohne sie gibt es keinen Drill-down", l.VatID)
		}
	}
	if got["FR12345678901"] != 800000 {
		t.Errorf("Leistungen an FR12345678901 = %s, erwartet 8.000,00", got["FR12345678901"])
	}
	if got["IT00743110157"] != 250000 {
		t.Errorf("Leistungen an IT00743110157 = %s, erwartet 2.500,00", got["IT00743110157"])
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
			serviceSupply(1, "2026-000001", "2026-01-15", 1, 500000),
			serviceSupply(2, "2026-000002", "2026-02-15", 2, 300000),
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

// Ohne USt-IdNr. ist der Umsatz nicht meldbar: § 18a Abs. 7 UStG verlangt sie.
func TestZMMissingVatIDBecomesAFinding(t *testing.T) {
	people := map[uint]ZMRecipient{
		1: {Name: "Client SARL", CountryCode: "FR", VatID: "", IsEU: true},
	}
	movements := ZMMovements(ZMSource{
		Entries:   []domain.JournalEntry{serviceSupply(1, "2026-000001", "2026-01-15", 1, 500000)},
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

// Für sonstige Leistungen ist das Kalendervierteljahr der Meldezeitraum
// (§ 18a Abs. 1 Satz 1 UStG) — unabhängig von ihrer Höhe.
func TestZMPeriodsAreQuarters(t *testing.T) {
	periods := ZMPeriodsOfYear(2026)
	if len(periods) != 4 {
		t.Fatalf("erwartet vier Quartalsmeldungen, erhalten %d", len(periods))
	}
	for i, p := range periods {
		if p.Type != domain.VatPeriodQuarter {
			t.Errorf("Zeitraum %s ist %s, erwartet Quartal", p.Key, p.Type)
		}
		if want := quarterPeriod(2026, i+1).Key; p.Key != want {
			t.Errorf("Zeitraum %d = %s, erwartet %s", i+1, p.Key, want)
		}
	}
}

// Die sonstige Leistung wird im Quartal ihres Leistungsdatums gemeldet
// (§ 18a Abs. 1 Satz 1 UStG).
func TestZMReportsServicesInTheirQuarter(t *testing.T) {
	movements := []ZMMovement{
		{EntryID: 1, Date: "2026-01-20", ContactID: 1, Amount: 100_000},
	}
	recipient := func(uint) ZMRecipient {
		return ZMRecipient{Name: "Client SARL", CountryCode: "FR", VatID: "FR12345678901", IsEU: true}
	}
	lines, _ := ZMLines(quarterPeriod(2026, 1), movements, recipient)
	if len(lines) != 1 || lines[0].Amount != 100_000 {
		t.Errorf("das Quartal meldet die sonstige Leistung nicht: %+v", lines)
	}
}
