package service

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/einvoice"
)

// -------------------------------------------------------------------------
// Ungeprüfte E-Rechnung
// -------------------------------------------------------------------------

// Ein Rechnungsdatensatz, den nie jemand geprüft hat, ist kein geprüfter. Ihn
// durchzulassen, weil kein Fehler gemeldet wurde, hieße Schweigen als Zustimmung
// zu lesen — es wurde nur nie gefragt.
func TestUncheckedEInvoiceIsAFinding(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Agentur GmbH", "DE", "")
	receipt, err := env.receipts.File(ctx, FileReceiptRequest{
		// Die Kopfdaten gehören seit Welle 6 zu jedem buchbaren Beleg
		// (BEL-02): ohne Belegdatum, Aussteller und Betrag weist
		// ValidateBookable die Buchung zurück.
		DocumentDate: "2026-03-01", IssuerName: "Lieferant GmbH", GrossAmount: 11900,
		Direction: domain.DirectionIncoming, FiscalYear: env.fiscalYear,
		Files: []NewFile{
			{Role: domain.ReceiptRoleOriginal, Path: env.writeTempFile(t, "rechnung.pdf", minimalPDF)},
			{Role: domain.ReceiptRoleStructured, FileName: "rechnung.xml",
				Content: []byte("<Invoice/>"), Derived: true},
		},
	})
	if err != nil {
		t.Fatalf("Beleg ablegen: %v", err)
	}

	req := env.receipt(t, vendor.ID, "fremdleistungen", 100_000,
		domain.TaxRateStandard, domain.TaxTreatmentDomestic)
	req.ReceiptID = receipt.ID

	preview, err := env.posting.PreviewIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	found := false
	for _, f := range preview.InputTaxFindings {
		if f.Code == findingEInvoiceUnchecked {
			found = true
		}
	}
	if !found {
		t.Fatalf("kein Befund zum ungeprüften Rechnungsdatensatz: %+v", preview.InputTaxFindings)
	}
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err == nil {
		t.Error("ohne Prüfergebnis und ohne Grund wird nicht mit Vorsteuer gebucht")
	}

	req.OverrideReason = "Datensatz vom Lieferanten nachgereicht, Prüfung folgt"
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err != nil {
		t.Errorf("mit festgehaltenem Grund muss die Buchung durchgehen: %v", err)
	}
}

// Die Nachbesserungen der Welle 8: die verlängerte Aufbewahrungsfrist übersteht
// die Kopfdaten, die Klärungsliste prüft die Stammdaten, und die Nachträge der
// ZM zählen eine sonstige Leistung nicht doppelt.

// rulesOf sammelt die Regelkennungen aller Befunde einer Klasse.
func rulesOf(out *domain.ReceiptFindings, class domain.ValidationFindingClass) map[string]domain.ValidationFinding {
	found := map[string]domain.ValidationFinding{}
	for _, g := range out.Groups {
		if g.Class != class {
			continue
		}
		for _, f := range g.Findings {
			found[f.Rule] = f
		}
	}
	return found
}

// Die Klasse „Inhaltsfehler" prüft die Pflichtangaben gegen die Stammdaten
// (RECH-07 K2): Anschrift und Steuernummer des Ausstellers, eigene Anschrift.
func TestContentFindingsCheckMasterData(t *testing.T) {
	receipt := &domain.Receipt{
		ID: 11, ReceiptNumber: "ER-2026-0011", Direction: domain.DirectionIncoming,
		Kind: domain.ReceiptKindInvoice, DocumentDate: "2026-03-01",
		// Über 250 €: eine Kleinbetragsrechnung (§ 33 UStDV) verlangt weniger.
		IssuerName: "Büromarkt GmbH", GrossAmount: 119000, TaxAmount: 19000,
	}
	out := ClassifyReceiptFindings(receipt, ReceiptCheckContext{
		Contact: &domain.Contact{Name: "Büromarkt GmbH"},
		Company: &domain.CompanySettings{CompanyName: "Pfennig Ventures GmbH"},
	})
	found := rulesOf(out, domain.ValidationClassContent)
	for _, rule := range []string{
		findingIssuerAddress, findingIssuerTaxNumber, "content_recipient_address",
	} {
		f, ok := found[rule]
		if !ok {
			t.Errorf("der Befund %s fehlt", rule)
			continue
		}
		if f.Norm == "" || f.InputTaxEffect == "" {
			t.Errorf("dem Befund %s fehlt die Norm oder die Folge für den Vorsteuerabzug", rule)
		}
		if !f.Blocking {
			t.Errorf("der Befund %s ist eine fehlende Pflichtangabe und hält die Vorsteuer an", rule)
		}
	}

	// Vollständige Stammdaten: keiner der drei Befunde.
	complete := ClassifyReceiptFindings(receipt, ReceiptCheckContext{
		Contact: &domain.Contact{
			Name: "Büromarkt GmbH", Street: "Lieferantenweg 3",
			PostalCode: "20095", City: "Hamburg", TaxID: "12/345/67890",
		},
		Company: &domain.CompanySettings{
			CompanyName: "Pfennig Ventures GmbH", Street: "Hauptstraße 1", PostalCode: "80331", City: "München",
		},
	})
	for rule := range rulesOf(complete, domain.ValidationClassContent) {
		if rule == findingIssuerAddress || rule == findingIssuerTaxNumber ||
			rule == "content_recipient_address" || rule == "content_issuer_unknown" {
			t.Errorf("bei vollständigen Stammdaten gibt es den Befund %s nicht", rule)
		}
	}
}

// Ohne Kontakt sagt die Liste, dass die Stammdaten nicht geprüft wurden — eine
// Aussage und keine Lücke.
func TestContentFindingsReportAMissingContact(t *testing.T) {
	out := ClassifyReceiptFindings(&domain.Receipt{
		ID: 12, ReceiptNumber: "ER-2026-0012", Direction: domain.DirectionIncoming,
		Kind: domain.ReceiptKindInvoice, DocumentDate: "2026-03-01",
		IssuerName: "Unbekannt GmbH", GrossAmount: 11900, TaxAmount: 1900,
	}, ReceiptCheckContext{})
	if _, ok := rulesOf(out, domain.ValidationClassContent)["content_issuer_unknown"]; !ok {
		t.Error("ein fehlender Kontakt ist ein eigener Befund")
	}
}

// Leistungsdatum und Entgeltaufschlüsselung prüft die Liste am strukturierten
// Datensatz (§ 14 Abs. 4 Nr. 6, 7 und 8 UStG).
func TestContentFindingsCheckServiceDateAndBreakdown(t *testing.T) {
	receipt := &domain.Receipt{
		ID: 13, ReceiptNumber: "ER-2026-0013", Direction: domain.DirectionIncoming,
		Kind: domain.ReceiptKindInvoice, DocumentDate: "2026-03-01",
		// Über 250 €: eine Kleinbetragsrechnung (§ 33 UStDV) verlangt weniger.
		IssuerName: "Büromarkt GmbH", GrossAmount: 119000, TaxAmount: 19000,
	}
	broken := &einvoice.Invoice{}
	broken.Totals.TaxBasisTotal = einvoice.NewAmount("100.00")
	broken.Totals.TaxTotal = einvoice.NewAmount("19.00")
	broken.Totals.GrandTotal = einvoice.NewAmount("120.00")

	found := rulesOf(ClassifyReceiptFindings(receipt, ReceiptCheckContext{Invoice: broken}),
		domain.ValidationClassContent)
	for _, rule := range []string{"content_service_date", "content_tax_breakdown", "content_amount_breakdown"} {
		if _, ok := found[rule]; !ok {
			t.Errorf("der Befund %s fehlt", rule)
		}
	}

	// Ein Datensatz mit Lieferdatum, Aufschlüsselung und stimmigen Summen hat
	// keinen dieser Befunde.
	good := &einvoice.Invoice{
		Delivery:     &einvoice.Delivery{Date: einvoice.NewDate("2026-02-28")},
		VATBreakdown: []einvoice.VATBreakdown{{}},
	}
	good.Totals.TaxBasisTotal = einvoice.NewAmount("100.00")
	good.Totals.TaxTotal = einvoice.NewAmount("19.00")
	good.Totals.GrandTotal = einvoice.NewAmount("119.00")
	found = rulesOf(ClassifyReceiptFindings(receipt, ReceiptCheckContext{Invoice: good}),
		domain.ValidationClassContent)
	for _, rule := range []string{"content_service_date", "content_tax_breakdown", "content_amount_breakdown"} {
		if _, ok := found[rule]; ok {
			t.Errorf("der Befund %s gehört nicht zu einem vollständigen Datensatz", rule)
		}
	}
}

// Der Belegdienst findet den Aussteller in den Stammdaten und prüft gegen ihn.
func TestFindingsResolveTheIssuerFromTheContacts(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	contact := &domain.Contact{
		Type: domain.ContactTypeVendor, Name: "Büromarkt GmbH", CountryCode: "DE",
		Street: "Lieferantenweg 3", PostalCode: "20095", City: "Hamburg",
	}
	if err := env.contacts.SaveContact(ctx, contact); err != nil {
		t.Fatalf("Lieferant: %v", err)
	}
	receipt := env.fileIncoming(t, "rechnung.pdf")
	if _, err := env.receipts.SaveHeader(ctx, receipt.ID, domain.ReceiptHeader{
		Kind: domain.ReceiptKindInvoice, DocumentDate: "2026-03-01",
		IssuerName: "büromarkt gmbh", GrossAmount: 119000, TaxAmount: 19000, Currency: "EUR",
	}); err != nil {
		t.Fatalf("Kopfdaten: %v", err)
	}

	out, err := env.receipts.Findings(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("Klärungsliste: %v", err)
	}
	found := rulesOf(out, domain.ValidationClassContent)
	if _, ok := found["content_issuer_unknown"]; ok {
		t.Error("der Aussteller steht in den Stammdaten und ist zu finden")
	}
	// Der Lieferant hat weder Steuernummer noch USt-IdNr.: das ist der Befund,
	// den die Stammdatenprüfung liefern soll.
	if _, ok := found[findingIssuerTaxNumber]; !ok {
		t.Error("ohne Steuernummer und USt-IdNr. des Ausstellers fehlt eine Pflichtangabe")
	}
	// Die eigenen Unternehmensdaten des Testaufbaus sind vollständig.
	if _, ok := found["content_recipient_address"]; ok {
		t.Error("die eigene Anschrift ist vollständig gesetzt")
	}
}

// Die Klärungsliste trennt Formatfehler, Geschäftsregelfehler und
// Inhaltsfehler und nennt je Befund die Folge für den Vorsteuerabzug
// (RECH-02 K5, RECH-07 K2).
func TestReceiptFindingsAreGroupedByClass(t *testing.T) {
	receipt := &domain.Receipt{
		ID: 7, ReceiptNumber: "ER-2026-0007", Direction: domain.DirectionIncoming,
		Kind: domain.ReceiptKindInvoice, ValidatedAt: "2026-03-01T10:00:00Z",
		GrossAmount: 11900, TaxAmount: 1900,
		ValidationFindings: `[
			{"rule":"BR-DE-15","severity":"fatal","message":"Die Leitweg-ID fehlt."},
			{"rule":"BR-CO-15","severity":"fatal","message":"Der Bruttobetrag stimmt nicht."},
			{"rule":"syntax","severity":"fatal","message":"Unerwartetes Element."}
		]`,
	}
	out := ClassifyReceiptFindings(receipt, ReceiptCheckContext{})

	byClass := map[domain.ValidationFindingClass][]domain.ValidationFinding{}
	for _, g := range out.Groups {
		byClass[g.Class] = g.Findings
	}
	if len(byClass[domain.ValidationClassBusinessRule]) != 2 {
		t.Errorf("%d Geschäftsregelfehler — BR-DE-15 und BR-CO-15 gehören dazu",
			len(byClass[domain.ValidationClassBusinessRule]))
	}
	if len(byClass[domain.ValidationClassFormat]) == 0 {
		t.Error("ein Befund ohne BR-Kennung ist ein Formatfehler")
	}
	// Der Beleg hat kein Belegdatum: eine Pflichtangabe des § 14 Abs. 4 UStG.
	content := byClass[domain.ValidationClassContent]
	if len(content) == 0 {
		t.Fatal("das fehlende Rechnungsdatum ist ein Inhaltsfehler")
	}
	for _, f := range content {
		if f.Norm == "" || f.InputTaxEffect == "" {
			t.Errorf("dem Befund %s fehlt die Norm oder die Folge für den Vorsteuerabzug", f.Rule)
		}
	}
	if out.Total != len(byClass[domain.ValidationClassBusinessRule])+
		len(byClass[domain.ValidationClassFormat])+len(content) {
		t.Errorf("die Gesamtzahl %d passt nicht zu den Gruppen", out.Total)
	}
	if out.Blocking == 0 {
		t.Error("ein fataler Befund hält die Buchung mit Vorsteuer an")
	}
}

// Ein ungeprüfter Beleg ist kein fehlerfreier: die Liste sagt, dass niemand
// nachgesehen hat.
func TestReceiptFindingsSayWhenNothingWasChecked(t *testing.T) {
	out := ClassifyReceiptFindings(&domain.Receipt{
		ID: 1, ReceiptNumber: "ER-2026-0001", Direction: domain.DirectionIncoming,
		Kind: domain.ReceiptKindInvoice, DocumentDate: "2026-03-01",
		IssuerName: "Büromarkt GmbH", GrossAmount: 11900, TaxAmount: 1900,
	}, ReceiptCheckContext{})
	if out.Checked {
		t.Error("ohne Prüfzeitpunkt ist der Beleg nicht geprüft")
	}
	if out.Groups == nil {
		t.Error("die Gruppen sind eine leere Liste und nicht nil")
	}
}
