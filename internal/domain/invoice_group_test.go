package domain

import (
	"testing"
)

// Die Invarianten des Rechnungsverbunds. Sie sind die Konstruktion, nach der
// sich § 14c richtet: eine Schlussrechnung ohne Verrechnung weist die Steuer
// zweimal aus.
func TestInvoiceGroupInvariants(t *testing.T) {
	group := &InvoiceGroup{
		Title: "Umbau", TotalNet: 1000000, TaxRate: TaxRateStandard,
		Advances: []AdvanceItem{
			{NetAmount: 400000, TaxAmount: 76000, GrossAmount: 476000, SettledAt: "2026-04-15"},
			{NetAmount: 300000, TaxAmount: 57000, GrossAmount: 357000},
		},
	}

	p := group.ComputeProgress()
	if p.BilledNet != 700000 {
		t.Errorf("abgerechnet = %s, erwartet 7000,00", p.BilledNet)
	}
	if p.ReceivedNet != 400000 || p.ReceivedTax != 76000 {
		t.Errorf("vereinnahmt = %s / %s, erwartet 4000,00 / 760,00", p.ReceivedNet, p.ReceivedTax)
	}
	if p.OpenNet != 300000 {
		t.Errorf("noch nicht abgerechnet = %s, erwartet 3000,00", p.OpenNet)
	}

	// Nur der berechnete *und* vereinnahmte Abschlag ist abzusetzen
	// (§ 14 Abs. 5 Satz 2 UStG).
	if got := group.DeductibleAdvances(); len(got) != 1 || got[0].NetAmount != 400000 {
		t.Errorf("abzusetzen sind %d Abschläge, erwartet den einen vereinnahmten", len(got))
	}

	// Die Summe der Abschläge darf den Gesamtbetrag nicht überschreiten.
	if err := group.EnsureAdvanceFits(200000); err != nil {
		t.Errorf("ein passender Abschlag wurde abgewiesen: %v", err)
	}
	if err := group.EnsureAdvanceFits(400000); err == nil {
		t.Error("ein Abschlag über den Gesamtbetrag hinaus muss abgewiesen werden")
	}

	// Ein stornierter Abschlag fällt aus der Verrechnung heraus: mit ihm
	// entfällt die Rechnung im Sinne des § 14 Abs. 5 Satz 2 UStG.
	group.Advances[0].Cancelled = true
	if len(group.DeductibleAdvances()) != 0 {
		t.Error("ein stornierter Abschlag darf nicht mehr abgesetzt werden")
	}
	if group.ComputeProgress().BilledNet != 300000 {
		t.Errorf("der stornierte Abschlag zählt nicht mehr mit: %s", group.ComputeProgress().BilledNet)
	}

	// Nach der Schlussrechnung nimmt der Verbund keine Abschläge mehr auf.
	group.Closed = true
	if err := group.EnsureAdvanceFits(1000); err == nil {
		t.Error("ein abgeschlossener Verbund darf keinen weiteren Abschlag aufnehmen")
	}
}
