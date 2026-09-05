package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Kein `null` in den Listen des Rechnungswesens.
//
// Die Rechnungsseite liest die Listen ohne Umweg — `items.map`,
// `precedingRefs.length`, `advances.map`, `gaps.length`. Ein nicht belegter
// Go-Slice wird in JSON zu `null`, und `null.length` nimmt im Render den Baum
// mit. Betroffen wäre der Regelfall: die Rechnung ohne Bezug auf eine
// vorausgegangene, der Verbund ohne Abschlag, der Nummernkreis ohne Lücke.
func TestInvoiceOutputsHaveNoNilLists(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)

	inv := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := svc.Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}
	assertNoNilSlices(t, "ausgestellte Rechnung", inv)
	assertNoNullLists(t, "ausgestellte Rechnung", inv, "items", "precedingRefs")

	invoices, err := svc.GetInvoices(ctx, 2026)
	if err != nil {
		t.Fatalf("Ausgangsrechnungen: %v", err)
	}
	assertNoNilSlices(t, "Ausgangsrechnungen", invoices)
	if len(invoices) == 0 {
		t.Fatal("erwartet die ausgestellte Rechnung")
	}
	assertNoNullLists(t, "gelesene Rechnung", invoices[0], "items", "precedingRefs")

	// Der Verbund und seine Abschläge.
	group := env.group(t, svc, customer.ID, 1000000)
	assertNoNilSlices(t, "angelegter Rechnungsverbund", group)
	assertNoNullLists(t, "angelegter Rechnungsverbund", group, "advances")

	advance, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-03-02", Net: 400000,
	})
	if err != nil {
		t.Fatalf("Abschlagsrechnung ausstellen: %v", err)
	}
	assertNoNilSlices(t, "Abschlagsrechnung", advance)
	assertNoNullLists(t, "Abschlagsrechnung", advance, "items", "precedingRefs")

	groups, err := svc.GetInvoiceGroups(ctx, 2026)
	if err != nil {
		t.Fatalf("Rechnungsverbünde: %v", err)
	}
	assertNoNilSlices(t, "Rechnungsverbünde", groups)

	open, err := svc.OpenAdvanceItems(ctx)
	if err != nil {
		t.Fatalf("offene Abschläge: %v", err)
	}
	assertNoNilSlices(t, "offene Abschläge", open)

	report, err := svc.NumberGaps(ctx, 2026)
	if err != nil {
		t.Fatalf("Lückenbericht: %v", err)
	}
	assertNoNilSlices(t, "Lückenbericht", report)
	assertNoNullLists(t, "Lückenbericht", report, "gaps")

	// Storno und Berichtigung: beide Dokumente gehen als JSON zurück an die
	// Oberfläche, und beide tragen Positionen und Bezüge.
	storno, err := svc.CancelWithDocument(ctx, inv.ID, "Leistung nicht erbracht")
	if err != nil {
		t.Fatalf("stornieren: %v", err)
	}
	assertNoNilSlices(t, "Stornorechnung", storno)
	assertNoNullLists(t, "Stornorechnung", storno, "items", "precedingRefs")

	// Und die Auswahllisten der Masken.
	assertNoNilSlices(t, "Mengeneinheiten", domain.UnitCodes())
	assertNoNilSlices(t, "E-Rechnungsformate", domain.EInvoiceProfiles())
	assertNoNilSlices(t, "Versandwege", domain.InvoiceSentViaOptions())
	assertNoNilSlices(t, "Lückengründe", domain.NumberGapReasons())
}
