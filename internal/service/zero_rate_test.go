package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// § 12 Abs. 3 UStG ist keine Steuerbefreiung: der Umsatz ist steuerpflichtig zum
// Satz null. Der SKR04 hat dafür ein eigenes Erlöskonto, das nicht mit den
// Konten für steuerfreie Umsätze zusammenfällt.
func TestZeroRatedSupplyBooksToItsOwnRevenueAccount(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Anlagenbetreiber", "DE", "")

	inv := &domain.Invoice{
		ContactID: customer.ID, Date: "2026-05-04",
		ServiceDateFrom: "2026-05-04", ServiceDateTo: "2026-05-04",
		TaxTreatment: domain.TaxTreatmentZeroRated,
		Items: []domain.InvoiceItem{{
			Description: "Solarmodule inkl. Installation", QuantityMilli: 1000,
			UnitPrice: 1200000, TaxRate: domain.TaxRateNone,
		}},
	}
	if err := env.invoices(t).Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung: %v", err)
	}

	entry, err := env.journalRepo.FindByID(ctx, *inv.JournalEntryID)
	if err != nil {
		t.Fatalf("Buchung laden: %v", err)
	}

	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, customer.LedgerAccount, 1200000},
		{domain.SideCredit, "4290", 1200000}, // Erlöse 0 % USt
	})
	if entry.TaxTreatment != domain.TaxTreatmentZeroRated {
		t.Errorf("der Steuerfall muss an der Buchung stehen, ist aber %q", entry.TaxTreatment)
	}
	for _, l := range entry.Lines {
		if l.Account == "4150" || l.Account == "4120" || l.Account == "4125" {
			t.Errorf("ein nullbesteuerter Umsatz gehört nicht auf ein Konto für steuerfreie Umsätze (%s)", l.Account)
		}
	}
}

// In der Auswertung steht der nullbesteuerte Umsatz bei den steuerpflichtigen,
// nicht bei den steuerfreien — sonst behauptete die Zahl das Gegenteil des
// Gesetzes.
func TestZeroRatedRevenueIsReportedSeparatelyFromExempt(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Anlagenbetreiber", "DE", "")
	invoices := env.invoices(t)

	zeroRated := &domain.Invoice{
		ContactID: customer.ID, Date: "2026-05-04",
		ServiceDateFrom: "2026-05-04", ServiceDateTo: "2026-05-04",
		TaxTreatment: domain.TaxTreatmentZeroRated,
		Items: []domain.InvoiceItem{{
			Description: "Solarmodule", QuantityMilli: 1000, UnitPrice: 1200000, TaxRate: domain.TaxRateNone,
		}},
	}
	if err := invoices.Issue(ctx, zeroRated); err != nil {
		t.Fatalf("Rechnung Nullsteuersatz: %v", err)
	}

	exempt := &domain.Invoice{
		ContactID: customer.ID, Date: "2026-05-05",
		ServiceDateFrom: "2026-05-05", ServiceDateTo: "2026-05-05",
		TaxTreatment: domain.TaxTreatmentExempt,
		Items: []domain.InvoiceItem{{
			Description: "Steuerfreie Leistung", QuantityMilli: 1000, UnitPrice: 300000, TaxRate: domain.TaxRateNone,
		}},
	}
	if err := invoices.Issue(ctx, exempt); err != nil {
		t.Fatalf("Rechnung steuerfrei: %v", err)
	}

	summary, err := env.vat().Summary(ctx, "", "")
	if err != nil {
		t.Fatalf("USt-Auswertung: %v", err)
	}
	if summary.ZeroRatedRevenue != 1200000 {
		t.Errorf("Umsatz zum Nullsteuersatz = %s, erwartet 12.000,00", summary.ZeroRatedRevenue)
	}
	if summary.ExemptRevenue != 300000 {
		t.Errorf("steuerfreier Umsatz = %s, erwartet 3.000,00", summary.ExemptRevenue)
	}
	if summary.OutputTax != 0 {
		t.Errorf("beide Fälle erzeugen keine Umsatzsteuer, gemeldet sind aber %s", summary.OutputTax)
	}
}

// Ein steuerpflichtiger Inlandsumsatz ohne Steuersatz ist ein Modellfehler: es
// wäre nicht mehr erkennbar, ob steuerfrei, nicht steuerbar oder nullbesteuert
// gemeint war.
func TestDomesticWithoutRateIsRejected(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Lieferant", "DE", "")

	req := env.receipt(t, vendor.ID, "buerobedarf", 5000, domain.TaxRateNone, domain.TaxTreatmentDomestic)
	_, err := env.posting.PostIncomingReceipt(ctx, req)
	if err == nil {
		t.Fatal("ein steuerpflichtiger Inlandsumsatz ohne Steuersatz muss abgelehnt werden")
	}
	for _, want := range []string{"19", "steuerfrei", "Nullsteuersatz"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("die Meldung soll %q nennen, lautet aber: %v", want, err)
		}
	}

	// Mit benanntem Steuerfall geht es.
	req = env.receipt(t, vendor.ID, "buerobedarf", 5000, domain.TaxRateNone, domain.TaxTreatmentNotTaxable)
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err != nil {
		t.Fatalf("mit benanntem Steuerfall muss gebucht werden können: %v", err)
	}
}

// Die Gruppen, die keine Vorsteuer tragen, schlagen den Steuerfall vor, statt
// ihn hinter einem Satz von null zu verstecken.
func TestGroupsWithoutInputTaxProposeATreatment(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Vermieter", "DE", "")

	group, err := env.postingGroup("miete")
	if err != nil {
		t.Fatalf("Gruppe laden: %v", err)
	}
	if group.Treatment() == domain.TaxTreatmentDomestic {
		t.Fatal("die Gruppe Miete darf nicht den steuerpflichtigen Inlandsfall vorschlagen — sie trägt keine Vorsteuer")
	}

	// Und mit dem vorgeschlagenen Steuerfall lässt sie sich buchen.
	req := env.receipt(t, vendor.ID, "miete", 150000, group.DefaultRate, group.Treatment())
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err != nil {
		t.Fatalf("Miete mit dem vorgeschlagenen Steuerfall buchen: %v", err)
	}
}
