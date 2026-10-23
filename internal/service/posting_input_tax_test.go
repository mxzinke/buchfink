package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// -------------------------------------------------------------------------
// Der Vorsteuerausschluss an der Zeile
// -------------------------------------------------------------------------

// Null Promille abziehbar ist etwas anderes als „kein Vorsteuerschlüssel".
// Solange beides als null an der Zeile stand, ließ sich der Ausschluss des
// § 15 Abs. 1a UStG von der vollen Abziehbarkeit nicht unterscheiden.
func TestExcludedInputTaxCarriesItsOwnMarker(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.gifts(t)
	vendor := env.vendor(t, "Yachthafen GmbH", "DE", "")

	excluded, err := env.posting.PostIncomingReceipt(ctx, env.receipt(
		t, vendor.ID, "repraesentation", 100_000,
		domain.TaxRateStandard, domain.TaxTreatmentDomestic))
	if err != nil {
		t.Fatalf("Buchung: %v", err)
	}
	if got := shareOn(excluded, "6645"); got != domain.InputTaxExcluded {
		t.Errorf("Vorsteueranteil %d — erwartet den Marker %d für den Ausschluss nach "+
			"§ 15 Abs. 1a UStG", got, domain.InputTaxExcluded)
	}

	plain, err := env.posting.PostIncomingReceipt(ctx, env.receipt(
		t, vendor.ID, "fremdleistungen", 100_000,
		domain.TaxRateStandard, domain.TaxTreatmentDomestic))
	if err != nil {
		t.Fatalf("Buchung: %v", err)
	}
	if got := shareOn(plain, "5906"); got != 0 {
		t.Errorf("Vorsteueranteil %d — bei vollem Abzug bleibt das Feld leer, damit die "+
			"Hash-Kette bestehender Buchungen unverändert bleibt", got)
	}
}

// shareOn liefert den Vorsteueranteil der ersten Zeile auf einem Konto.
func shareOn(entry *domain.JournalEntry, account string) int {
	for _, l := range entry.Lines {
		if l.Account == account {
			return l.InputTaxShare
		}
	}
	return 0
}

// -------------------------------------------------------------------------
// Vorsteuerausschluss und eigene Steuerschuld
// -------------------------------------------------------------------------

// § 15 Abs. 1a UStG nimmt den Abzug, nicht die Steuerschuld.
//
// Beim innergemeinschaftlichen Erwerb und beim Reverse Charge entstehen beide
// aus derselben Bemessungsgrundlage. Die geschuldete Steuer wird aus der vollen
// und die Vorsteuer aus der abziehbaren Bemessungsgrundlage berechnet
// (siehe taxLinesForShare).
// Geprüft wird deshalb das Ergebnis: die Erwerb- bzw. § 13b-Steuer steht in
// voller Höhe, eine Vorsteuerzeile entsteht nicht, und die nicht abziehbare
// Steuer gehört zum Aufwand (§ 9b Abs. 1 EStG).
func TestExcludedInputTaxKeepsTheOwedTax(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.gifts(t)
	vendor := env.vendor(t, "Yacht Charter SARL", "FR", "FR12345678901")

	cases := []struct {
		treatment domain.TaxTreatment
		owed      string
		input     string
	}{
		{domain.TaxTreatmentIntraCommunityAcquisition,
			domain.AccountUmsatzsteuerIG19, domain.AccountVorsteuerIG19},
		{domain.TaxTreatmentReverseCharge,
			domain.AccountUmsatzsteuer13b19, domain.AccountVorsteuer13b19},
	}
	for _, c := range cases {
		req := env.receipt(t, vendor.ID, "repraesentation", 100_000,
			domain.TaxRateStandard, c.treatment)
		entry, err := env.posting.PostIncomingReceipt(ctx, req)
		if err != nil {
			t.Fatalf("%s: %v", c.treatment, err)
		}
		if got := creditOn(entry, c.owed); got != 19_000 {
			t.Errorf("%s: geschuldete Steuer %s € — erwartet 190,00 € auf die volle "+
				"Bemessungsgrundlage", c.treatment, got)
		}
		if got := debitOn(entry, c.input); got != 0 {
			t.Errorf("%s: Vorsteuer %s € — § 15 Abs. 1a UStG schließt den Abzug aus",
				c.treatment, got)
		}
		if got := debitOn(entry, "6645"); got != 119_000 {
			t.Errorf("%s: Aufwand %s € — erwartet 1.190,00 €, die nicht abziehbare "+
				"Steuer gehört dazu (§ 9b Abs. 1 EStG)", c.treatment, got)
		}
	}

	// Im Inland bleibt der Ausschluss, was er war: die Steuer gehört zum
	// Aufwand, eine Vorsteuerzeile entsteht nicht.
	entry, err := env.posting.PostIncomingReceipt(ctx, env.receipt(
		t, vendor.ID, "repraesentation", 100_000,
		domain.TaxRateStandard, domain.TaxTreatmentDomestic))
	if err != nil {
		t.Fatalf("Inlandsumsatz: %v", err)
	}
	if got := debitOn(entry, "6645"); got != 119_000 {
		t.Errorf("Aufwand %s € — erwartet 1.190,00 € brutto (§ 9b Abs. 1 EStG)", got)
	}
	if debitOn(entry, domain.AccountVorsteuer19) != 0 {
		t.Error("zu einer Aufwendung des § 4 Abs. 5 Satz 1 Nr. 4 EStG gehört keine Vorsteuer")
	}
}

// -------------------------------------------------------------------------
// Vorsteuerschlüssel: die Rundung je Steuersatzgruppe
// -------------------------------------------------------------------------

// Zwei Positionen zum selben Steuersatz mit geteiltem Vorsteuerabzug ergeben
// zusammen den Bruttobetrag der Rechnung.
//
// Die nicht abziehbare Vorsteuer wird je Steuersatzgruppe einmal gerundet, wie
// die Steuerzeile selbst. Wurde sie je Position gerundet, kamen 79,31 € statt
// 79,33 € heraus, und die Gegenzeile glich die zwei Cent still aus: die
// Verbindlichkeit stimmte dann mit der Rechnung des Lieferanten nicht überein.
func TestSplitInputTaxRoundsOncePerRateGroup(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Autohaus GmbH", "DE", "")

	req := env.receipt(t, vendor.ID, "fahrzeugkosten", 3_333,
		domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.Positions[0].InputTaxShare = 500
	req.Positions[0].InputTaxShareReason = "Kfz zur Hälfte betrieblich genutzt, Fahrtenbuch 2026"
	req.Positions = append(req.Positions, req.Positions[0])

	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Buchung: %v", err)
	}

	var expense, inputTax, settlement domain.Cents
	for _, l := range entry.Lines {
		switch {
		case l.TaxKey == "VST19":
			inputTax += l.Amount
		case l.Side == domain.SideDebit:
			expense += l.Amount
		default:
			settlement += l.Amount
		}
	}

	// 66,66 € netto, 19 % einmal gerundet: 12,67 € — brutto 79,33 €.
	if settlement != 7_933 {
		t.Errorf("Gegenzeile %s € — erwartet 79,33 € (66,66 € netto zuzüglich 12,67 € Steuer)",
			settlement)
	}
	// Abziehbar ist die Steuer auf 33,34 €: 6,33 €.
	if inputTax != 633 {
		t.Errorf("Vorsteuer %s € — erwartet 6,33 € (19 %% von 33,34 €)", inputTax)
	}
	// Der Aufwand trägt den Rest: 66,66 € zuzüglich 6,34 € nicht abziehbarer
	// Vorsteuer.
	if expense != 7_300 {
		t.Errorf("Aufwand %s € — erwartet 73,00 €", expense)
	}
	if !entry.IsBalanced() {
		t.Error("die Buchung ist nicht ausgeglichen")
	}
}

// Der Vorsteuerschlüssel wirkt beim § 13b-Umsatz nur auf die Vorsteuerzeile;
// die geschuldete Steuer bleibt voll (UST-07 K2).
func TestInputTaxShareOnReverseChargeKeepsTheOwedTax(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Software SARL", "FR", "FR12345678901")

	req := env.receipt(t, vendor.ID, "software", 100_000,
		domain.TaxRateStandard, domain.TaxTreatmentReverseCharge)
	req.Positions[0].InputTaxShare = 600
	req.Positions[0].InputTaxShareReason = "Zu 60 % betrieblich genutzt, Aufteilung nach Nutzung"

	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("§ 13b mit Vorsteuerschlüssel: %v", err)
	}
	if got := creditOn(entry, domain.AccountUmsatzsteuer13b19); got != 19_000 {
		t.Errorf("geschuldete Steuer %s € — § 15 Abs. 4 UStG teilt den Abzug, nicht die "+
			"Steuerschuld (Kz 46/47)", got)
	}
	if got := debitOn(entry, domain.AccountVorsteuer13b19); got != 11_400 {
		t.Errorf("Vorsteuer %s € — erwartet 60 %% von 190,00 € (Kz 67)", got)
	}
	// Der nicht abziehbare Teil gehört zum Aufwand (§ 9b Abs. 1 EStG) und
	// steht deshalb auf dem Aufwandskonto der Gruppe — nicht auf einem eigenen
	// Konto für nicht abziehbare Vorsteuer.
	if got := debitOn(entry, "6837"); got != 0 {
		t.Errorf("%s € auf 6837 — die nicht abziehbare Steuer gehört zum Aufwand der "+
			"Leistung und nicht auf ein eigenes Konto", got)
	}
	var expense domain.Cents
	for _, l := range entry.Lines {
		if l.Side == domain.SideDebit && !strings.HasPrefix(l.Account, "1") &&
			!strings.HasPrefix(l.Account, "3") {
			expense += l.Amount
		}
	}
	if expense != 107_600 {
		t.Errorf("Aufwand %s € — erwartet 1.076,00 €: netto zuzüglich der nicht "+
			"abziehbaren Steuer von 76,00 €", expense)
	}
	// Und die Buchung geht auf: die Verbindlichkeit ist der Nettobetrag, weil
	// der Lieferant keine Steuer berechnet.
	if !entry.IsBalanced() {
		t.Error("die Buchung ist nicht ausgeglichen")
	}
}
