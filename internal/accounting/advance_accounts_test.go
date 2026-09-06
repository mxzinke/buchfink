package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Das Konto der erhaltenen Anzahlung folgt dem Steuersatz, weil die Anzahlung
// mit der Vereinnahmung versteuert ist und die Schlussrechnung genau diesen
// Betrag wieder auflösen muss. Ein Sammelkonto ließe offen, welcher Teil des
// Saldos welche Steuer trägt.
func TestAdvanceAccountFor(t *testing.T) {
	cases := map[domain.TaxRate]string{
		domain.TaxRateStandard: "3272",
		domain.TaxRateReduced:  "3260",
		domain.TaxRateNone:     "3250",
	}
	for rate, want := range cases {
		got, err := AdvanceAccountFor(rate)
		if err != nil {
			t.Fatalf("AdvanceAccountFor(%s): %v", rate.Label(), err)
		}
		if got != want {
			t.Errorf("AdvanceAccountFor(%s) = %s, erwartet %s", rate.Label(), got, want)
		}
	}
	if _, err := AdvanceAccountFor(domain.TaxRate(1600)); err == nil {
		t.Error("für einen Satz ohne Konto darf kein Konto geraten werden")
	}
}

// Die Forderungsausbuchung braucht das Konto zum Steuersatz der Rechnung: der
// Aufwand und seine Steuerkorrektur nach § 17 Abs. 2 Nr. 1 UStG gehören
// zusammen.
func TestWriteOffAccountFor(t *testing.T) {
	cases := map[domain.TaxRate]string{
		domain.TaxRateStandard: "6936",
		domain.TaxRateReduced:  "6931",
		domain.TaxRateNone:     "6930",
	}
	for rate, want := range cases {
		got, err := WriteOffAccountFor(rate)
		if err != nil {
			t.Fatalf("WriteOffAccountFor(%s): %v", rate.Label(), err)
		}
		if got != want {
			t.Errorf("WriteOffAccountFor(%s) = %s, erwartet %s", rate.Label(), got, want)
		}
	}
}

// Die verwendeten Konten müssen im Kontenplan stehen und bebuchbar sein — sonst
// scheitert die Buchung erst beim Anwender.
func TestAdvanceAndWriteOffAccountsExist(t *testing.T) {
	chart := NewChart(DefaultSKR04Accounts())
	for _, account := range []string{
		AccountErhalteneAnzahlungen19, AccountErhalteneAnzahlungen7, AccountErhalteneAnzahlungen,
		AccountGeleisteteAnzahlungenVorraete, AccountGeleisteteAnzahlungenImmateriell,
		AccountGeleisteteAnzahlungenAnlagen,
		AccountForderungsverluste, AccountForderungsverluste7, AccountForderungsverluste19,
	} {
		if err := chart.EnsurePostable(account); err != nil {
			t.Errorf("Konto %s ist nicht bebuchbar: %v", account, err)
		}
	}
}

// § 27 Abs. 38 UStG kennt drei Stufen. Auf der Ausstellerseite ist der
// Vorjahresumsatz bekannt, und damit lässt sich die mittlere Stufe entscheiden
// statt weiterzureichen.
func TestEInvoiceIssueTransition(t *testing.T) {
	const overLimit = domain.Cents(80_000_100)
	const underLimit = domain.Cents(50_000_000)

	cases := []struct {
		date    string
		revenue domain.Cents
		want    EInvoiceTransition
		why     string
	}{
		{"2026-12-31", overLimit, EInvoiceTransitionAllowed,
			"bis Ende 2026 gilt die Übergangsregel ohne Bedingung"},
		{"2027-06-01", underLimit, EInvoiceTransitionAllowed,
			"2027 bis 800.000 € Vorjahresumsatz weiterhin zulässig"},
		{"2027-06-01", overLimit, EInvoiceTransitionExpired,
			"2027 über 800.000 € Vorjahresumsatz nicht mehr zulässig"},
		{"2027-06-01", 0, EInvoiceTransitionAllowed,
			"ein nicht erfasster Vorjahresumsatz darf keine Rechnung blockieren, die das Gesetz erlaubt"},
		{"2028-01-01", 0, EInvoiceTransitionExpired,
			"ab 2028 gilt keine Übergangsregel mehr"},
	}
	for _, c := range cases {
		if got := EInvoiceIssueTransitionFor(c.date, c.revenue); got != c.want {
			t.Errorf("EInvoiceIssueTransitionFor(%s, %s) = %q, erwartet %q — %s",
				c.date, c.revenue, got, c.want, c.why)
		}
	}

	// Genau auf der Grenze ist sie nicht überschritten: § 27 Abs. 38 Nr. 2 UStG
	// sagt „nicht mehr als 800.000 Euro".
	if got := EInvoiceIssueTransitionFor("2027-06-01", EInvoiceTransitionRevenueLimit); got != EInvoiceTransitionAllowed {
		t.Errorf("genau 800.000 € liegen noch innerhalb der Grenze, erhalten %q", got)
	}
}

// Die Kleinbetragsgrenze ist datiert: 150 € bis 2016, 250 € ab 2017
// (§ 33 UStDV). Ein fester Wert im Code verböte die Nacharbeit eines alten
// Jahres.
func TestSmallAmountLimitIsDated(t *testing.T) {
	old, err := TaxParametersFor("2016-12-31")
	if err != nil {
		t.Fatal(err)
	}
	if old.SmallAmountInvoiceLimit != 15000 {
		t.Errorf("2016: Grenze = %s, erwartet 150,00", old.SmallAmountInvoiceLimit)
	}
	now, err := TaxParametersFor("2026-03-01")
	if err != nil {
		t.Fatal(err)
	}
	if now.SmallAmountInvoiceLimit != 25000 {
		t.Errorf("2026: Grenze = %s, erwartet 250,00", now.SmallAmountInvoiceLimit)
	}
}

// Die vereinnahmte Anzahlung folgt dem Geld und nicht der Leistung
// (§ 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG) — auch bei Sollversteuerung und
// lange bevor die Leistung erbracht ist.
func TestVatPeriodOfAdvanceFollowsThePayment(t *testing.T) {
	entry := &domain.JournalEntry{
		Source:      domain.EntrySourceAdvance,
		BookingDate: "2026-04-15", DocumentDate: "2026-04-15",
		// Der Leistungszeitraum steht auf der Zukunft, wie es bei einer
		// Anzahlung der Fall ist.
		ServiceDateFrom: "2026-11-01", ServiceDateTo: "2026-11-30",
	}
	line := domain.JournalLine{Side: domain.SideCredit, Account: "3806", TaxKey: "UST19", Amount: 76000, TaxBase: 400000}

	if got := VatPeriodFor(entry, line, ""); got != "2026-04-15" {
		t.Errorf("Zeitraum der Anzahlung = %q, erwartet das Zahlungsdatum 2026-04-15", got)
	}

	// Die gewöhnliche Ausgangsrechnung bleibt beim Leistungszeitraum.
	ordinary := &domain.JournalEntry{
		Source:      domain.EntrySourceInvoice,
		BookingDate: "2026-04-15", DocumentDate: "2026-04-15",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-03-31",
	}
	if got := VatPeriodFor(ordinary, line, ""); got != "2026-03-31" {
		t.Errorf("Zeitraum der Rechnung = %q, erwartet den Leistungszeitraum 2026-03-31", got)
	}
}
