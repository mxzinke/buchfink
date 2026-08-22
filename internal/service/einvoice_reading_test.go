package service

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/invoice"
)

// Das Übersetzen des WASM-Moduls kostet gut zehn Sekunden. Ein Renderer für das
// ganze Testpaket macht aus einer Minute Wartezeit eine.
var (
	rendererOnce sync.Once
	renderer     *invoice.Renderer
)

func sharedRenderer() *invoice.Renderer {
	rendererOnce.Do(func() { renderer = invoice.NewRenderer() })
	return renderer
}

func (e *testEnv) einvoices() *EInvoiceService {
	return NewEInvoiceService(e.receipts, e.contactRepo, e.fiscalYear)
}

// hybridReceipt legt einen Beleg mit einem echten hybriden PDF ab — erzeugt über
// denselben Weg, den eine Ausgangsrechnung nimmt. Das ist die realistischste
// Eingangsrechnung, die sich ohne Fremddatei herstellen lässt.
func (e *testEnv) hybridReceipt(t *testing.T, supplier *domain.CompanySettings, treatment domain.TaxTreatment, rate domain.TaxRate, net domain.Cents) *domain.Receipt {
	t.Helper()
	ctx := context.Background()

	inv := &domain.Invoice{
		FiscalYear: 2026, ContactID: 1, InvoiceNumber: "RE-LIEF-4711",
		Date: "2026-05-04", ServiceDateFrom: "2026-04-01", ServiceDateTo: "2026-04-30",
		DueDate: "2026-05-18", ContactName: "Buchfink-Mandant",
		TaxTreatment: treatment, Currency: "EUR",
		Items: []domain.InvoiceItem{{
			Position: 1, Description: "Wartung Serverschrank", QuantityMilli: 1000,
			Unit: "Stk", UnitPrice: net, TaxRate: rate,
		}},
	}
	inv.Recalculate()

	buyer := &domain.Contact{
		Name: "Buchfink-Mandant", Address: "Hauptstraße 1, 80331 München",
		CountryCode: "DE", VatID: "DE111111111",
	}
	xml, err := invoice.GenerateZUGFeRDXML(inv, supplier, buyer)
	if err != nil {
		t.Fatalf("Lieferanten-XML: %v", err)
	}

	pdf, err := sharedRenderer().RenderInvoicePDF(ctx, inv, supplier, buyer, xml)
	if err != nil {
		t.Fatalf("Lieferanten-PDF: %v", err)
	}

	receipt, err := e.receipts.File(ctx, FileReceiptRequest{
		Direction:   domain.DirectionIncoming,
		ReceivedVia: domain.ReceivedViaEmail,
		Files: []NewFile{
			{Role: domain.ReceiptRoleOriginal, FileName: "eingangsrechnung.pdf", Content: pdf},
		},
	})
	if err != nil {
		t.Fatalf("Beleg ablegen: %v", err)
	}
	return receipt
}

func supplierSettings(name, vatID string) *domain.CompanySettings {
	return &domain.CompanySettings{
		CompanyName: name, Street: "Lieferantenweg 3", ZipCity: "20095 Hamburg",
		VatID: vatID, TaxNumber: "22/333/44444",
	}
}

// Ablegen und Auslesen sind zwei Schritte: der Beleg kommt in der empfangenen
// Form herein, der strukturierte Teil wird daraus gezogen und als abgeleitet
// gekennzeichnet.
func TestExtractingTheStructuredPartFromAHybridPDF(t *testing.T) {
	if testing.Short() {
		t.Skip("die WASM-Kompilierung ist zu langsam für -short")
	}
	env := newTestEnv(t)
	ctx := context.Background()

	receipt := env.hybridReceipt(t, supplierSettings("Netzwerk GmbH", "DE555666777"),
		domain.TaxTreatmentDomestic, domain.TaxRateStandard, 100000)

	if _, ok := receipt.FileByRole(domain.ReceiptRoleStructured); ok {
		t.Fatal("vor dem Auslesen darf es keinen strukturierten Teil geben")
	}

	updated, err := env.einvoices().ExtractStructuredPart(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("strukturierten Teil holen: %v", err)
	}

	structured, ok := updated.FileByRole(domain.ReceiptRoleStructured)
	if !ok {
		t.Fatal("der strukturierte Teil fehlt")
	}
	if !structured.Derived {
		t.Error("aus dem PDF gezogen heißt abgeleitet")
	}
	if structured.FileName != "factur-x.xml" {
		t.Errorf("Dateiname = %q, erwartet factur-x.xml", structured.FileName)
	}

	// Der Beleg-Hash hat sich geändert, das Original ist unangetastet geblieben.
	if updated.ReceiptHash == receipt.ReceiptHash {
		t.Error("eine zusätzliche Datei muss den Beleg-Hash ändern")
	}
	original, _ := updated.FileByRole(domain.ReceiptRoleOriginal)
	if original.Derived {
		t.Error("die empfangene Form bleibt die empfangene Form")
	}
}

// Der Buchungsvorschlag: Beträge und Steuersatz kommen aus dem XML, der
// Lieferant über die USt-IdNr., das Konto bleibt offen.
func TestProposalFillsWhatTheDocumentKnows(t *testing.T) {
	if testing.Short() {
		t.Skip("die WASM-Kompilierung ist zu langsam für -short")
	}
	env := newTestEnv(t)
	ctx := context.Background()

	vendor := env.vendor(t, "Netzwerk GmbH", "DE", "DE555666777")
	receipt := env.hybridReceipt(t, supplierSettings("Netzwerk GmbH", "DE555666777"),
		domain.TaxTreatmentDomestic, domain.TaxRateStandard, 100000)

	svc := env.einvoices()
	if _, err := svc.ExtractStructuredPart(ctx, receipt.ID); err != nil {
		t.Fatalf("strukturierten Teil holen: %v", err)
	}
	proposal, err := svc.Propose(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("Vorschlag: %v", err)
	}

	if !proposal.MatchedContact || proposal.Request.ContactID != vendor.ID {
		t.Errorf("der Lieferant muss über die USt-IdNr. gefunden werden, ContactID = %d", proposal.Request.ContactID)
	}
	if proposal.InvoiceNumber != "RE-LIEF-4711" {
		t.Errorf("Rechnungsnummer = %q", proposal.InvoiceNumber)
	}
	if proposal.Request.DocumentDate != "2026-05-04" {
		t.Errorf("Belegdatum = %q, erwartet 2026-05-04", proposal.Request.DocumentDate)
	}
	if proposal.Request.ServiceDateTo != "2026-04-30" {
		t.Errorf("Leistungsdatum = %q, erwartet 2026-04-30", proposal.Request.ServiceDateTo)
	}
	if proposal.Request.TaxTreatment != domain.TaxTreatmentDomestic {
		t.Errorf("Steuerfall = %q", proposal.Request.TaxTreatment)
	}
	if proposal.GrossAmount != 119000 {
		t.Errorf("Bruttobetrag = %s, erwartet 1.190,00", proposal.GrossAmount)
	}

	if len(proposal.Request.Positions) != 1 {
		t.Fatalf("erwartet eine Position, erhalten %d", len(proposal.Request.Positions))
	}
	pos := proposal.Request.Positions[0]
	if pos.Net != 100000 || pos.TaxRate != domain.TaxRateStandard {
		t.Errorf("Position = %s netto, %s", pos.Net, pos.TaxRate.Label())
	}
	// Welches Aufwandskonto zutrifft, sagt keine Rechnung.
	if pos.PostingGroup != "" || pos.Account != "" {
		t.Error("der Vorschlag darf kein Konto raten")
	}
	if len(proposal.Notes) == 0 {
		t.Error("die offene Buchungsgruppe soll als Hinweis stehen")
	}

	// Und der Vorschlag lässt sich mit ergänzter Gruppe buchen.
	req := proposal.Request
	req.Positions[0].PostingGroup = "fremdleistungen"
	entry, err := env.posting.PostIncomingReceipt(ctx, req)
	if err != nil {
		t.Fatalf("den Vorschlag buchen: %v", err)
	}
	assertLines(t, entry, []bookedLine{
		{domain.SideDebit, "5906", 100000},
		{domain.SideDebit, "1406", 19000},
		{domain.SideCredit, vendor.LedgerAccount, 119000},
	})
}

// Der Kategoriecode wird gedreht: eine innergemeinschaftliche Lieferung des
// Lieferanten ist bei uns ein Erwerb — mit Erwerbsteuer und Vorsteuer.
func TestProposalInvertsTheCategoryCodeForTheRecipient(t *testing.T) {
	if testing.Short() {
		t.Skip("die WASM-Kompilierung ist zu langsam für -short")
	}
	env := newTestEnv(t)
	ctx := context.Background()

	vendor := env.vendor(t, "Groothandel B.V.", "NL", "NL123456789B01")
	receipt := env.hybridReceipt(t, supplierSettings("Groothandel B.V.", "NL123456789B01"),
		domain.TaxTreatmentIntraCommunitySupply, domain.TaxRateNone, 200000)

	svc := env.einvoices()
	if _, err := svc.ExtractStructuredPart(ctx, receipt.ID); err != nil {
		t.Fatalf("strukturierten Teil holen: %v", err)
	}
	proposal, err := svc.Propose(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("Vorschlag: %v", err)
	}

	if proposal.Request.TaxTreatment != domain.TaxTreatmentIntraCommunityAcquisition {
		t.Fatalf("Steuerfall = %q, erwartet innergemeinschaftlichen Erwerb — der Code steht aus Sicht des Ausstellers im Dokument",
			proposal.Request.TaxTreatment)
	}

	// Gebucht entstehen beide Steuerzeilen, und die USt-Auswertung sieht den
	// Vorgang auf beiden Seiten.
	req := proposal.Request
	req.Positions[0].PostingGroup = "wareneingang"
	req.Positions[0].TaxRate = domain.TaxRateStandard
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err != nil {
		t.Fatalf("den Vorschlag buchen: %v", err)
	}

	summary, err := env.vat().Summary(ctx, "", "")
	if err != nil {
		t.Fatalf("USt-Auswertung: %v", err)
	}
	if summary.IntraCommunityAcquisitionTax != 38000 {
		t.Errorf("Erwerbsteuer = %s, erwartet 380,00", summary.IntraCommunityAcquisitionTax)
	}
	if summary.InputTax != 38000 {
		t.Errorf("Vorsteuer = %s, erwartet 380,00", summary.InputTax)
	}
	if summary.Payable != 0 {
		t.Errorf("Zahllast = %s, erwartet 0,00 — der Erwerb ist zahlungsneutral, aber meldepflichtig", summary.Payable)
	}
	_ = vendor
}

// Ein unbekannter Lieferant wird benannt, nicht stillschweigend weggelassen.
func TestProposalNamesAnUnknownSupplier(t *testing.T) {
	if testing.Short() {
		t.Skip("die WASM-Kompilierung ist zu langsam für -short")
	}
	env := newTestEnv(t)
	ctx := context.Background()

	receipt := env.hybridReceipt(t, supplierSettings("Nie Gesehen GmbH", "DE999888777"),
		domain.TaxTreatmentDomestic, domain.TaxRateStandard, 50000)

	svc := env.einvoices()
	if _, err := svc.ExtractStructuredPart(ctx, receipt.ID); err != nil {
		t.Fatalf("strukturierten Teil holen: %v", err)
	}
	proposal, err := svc.Propose(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("Vorschlag: %v", err)
	}

	if proposal.MatchedContact {
		t.Error("ein unbekannter Lieferant darf nicht als gefunden gelten")
	}
	if proposal.Request.ContactID != 0 {
		t.Error("ohne Kontakt darf keine ContactID gesetzt sein")
	}
	var mentioned bool
	for _, note := range proposal.Notes {
		if strings.Contains(note, "Nie Gesehen GmbH") {
			mentioned = true
		}
	}
	if !mentioned {
		t.Errorf("der unbekannte Lieferant soll benannt werden, Hinweise: %v", proposal.Notes)
	}
}

// Ohne strukturierten Teil kein Vorschlag — der Vorsteuerabzug hängt daran.
func TestProposalRefusesWithoutTheStructuredPart(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	receipt := env.fileIncoming(t, "papierrechnung.pdf")
	_, err := env.einvoices().Propose(ctx, receipt.ID)
	if err == nil {
		t.Fatal("ohne strukturierten Teil darf kein Vorschlag entstehen")
	}
	if !strings.Contains(err.Error(), "Vorsteuerabzug") {
		t.Errorf("die Meldung soll den Grund nennen, lautet aber: %v", err)
	}
}

// Ein Scan enthält keinen strukturierten Teil, und das wird gesagt statt geraten.
func TestExtractingFromAScanIsRefused(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	receipt, err := env.receipts.File(ctx, FileReceiptRequest{
		Direction: domain.DirectionIncoming,
		Files: []NewFile{{
			Role: domain.ReceiptRoleOriginal, FileName: "scan.png",
			// Ein minimales PNG.
			Content: []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89"),
		}},
	})
	if err != nil {
		t.Fatalf("Scan ablegen: %v", err)
	}

	if _, err := env.einvoices().ExtractStructuredPart(ctx, receipt.ID); err == nil {
		t.Error("aus einem Scan lässt sich kein Rechnungsdatensatz holen")
	}
}

// Das Prüfergebnis bleibt am Beleg: mit Zeitpunkt, Regelwerk und Version. Ein
// Urteil ohne die Regeln, die es erzeugt haben, ist später nicht nachvollziehbar.
func TestValidationResultIsKeptWithTheReceipt(t *testing.T) {
	if testing.Short() {
		t.Skip("die WASM-Kompilierung ist zu langsam für -short")
	}
	env := newTestEnv(t)
	ctx := context.Background()

	receipt := env.hybridReceipt(t, supplierSettings("Netzwerk GmbH", "DE555666777"),
		domain.TaxTreatmentDomestic, domain.TaxRateStandard, 100000)

	updated, err := env.einvoices().ExtractStructuredPart(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("strukturierten Teil holen: %v", err)
	}

	if updated.DetectedFormat != string(invoice.FormatCII) {
		t.Errorf("erkanntes Format = %q, erwartet %q", updated.DetectedFormat, invoice.FormatCII)
	}
	if updated.DetectedProfile != "urn:cen.eu:en16931:2017" {
		t.Errorf("erkanntes Profil = %q", updated.DetectedProfile)
	}
	if updated.ValidationRuleset != invoice.EN16931RulesetID {
		t.Errorf("Regelwerk = %q", updated.ValidationRuleset)
	}
	if updated.ValidationVersion != invoice.EN16931RulesetVersion {
		t.Errorf("Regelwerksversion = %q", updated.ValidationVersion)
	}
	if updated.ValidatedAt == "" {
		t.Error("der Prüfzeitpunkt fehlt")
	}
	if updated.ValidationCoverage != string(invoice.CoveragePartial) {
		t.Errorf("Prüfumfang = %q — es darf keinen Wert für Vollständigkeit geben", updated.ValidationCoverage)
	}
	if updated.ValidationErrors != 0 {
		t.Errorf("eine von Buchfink erzeugte Rechnung soll fehlerfrei sein, gemeldet sind %d", updated.ValidationErrors)
	}

	// Und die Prüfung lässt sich wiederholen — das Regelwerk ist versioniert.
	result, err := env.einvoices().Validate(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("erneut prüfen: %v", err)
	}
	if !result.Valid() {
		t.Errorf("die erneute Prüfung meldet %d Fehler", result.ErrorCount())
	}
}

// Eine Prüfung darf auch nach dem Buchen noch laufen: sie fasst keine Datei an,
// der Beleg-Hash bleibt unberührt und die Kette damit heil.
func TestValidationCanRunOnASealedReceipt(t *testing.T) {
	if testing.Short() {
		t.Skip("die WASM-Kompilierung ist zu langsam für -short")
	}
	env := newTestEnv(t)
	ctx := context.Background()

	vendor := env.vendor(t, "Netzwerk GmbH", "DE", "DE555666777")
	receipt := env.hybridReceipt(t, supplierSettings("Netzwerk GmbH", "DE555666777"),
		domain.TaxTreatmentDomestic, domain.TaxRateStandard, 100000)

	svc := env.einvoices()
	if _, err := svc.ExtractStructuredPart(ctx, receipt.ID); err != nil {
		t.Fatalf("strukturierten Teil holen: %v", err)
	}
	proposal, err := svc.Propose(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("Vorschlag: %v", err)
	}
	req := proposal.Request
	req.ContactID = vendor.ID
	req.Positions[0].PostingGroup = "fremdleistungen"
	if _, err := env.posting.PostIncomingReceipt(ctx, req); err != nil {
		t.Fatalf("buchen: %v", err)
	}

	before, err := env.receipts.Get(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("Beleg laden: %v", err)
	}

	if _, err := svc.Validate(ctx, receipt.ID); err != nil {
		t.Fatalf("Prüfung nach dem Buchen: %v", err)
	}

	after, err := env.receipts.Get(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("Beleg laden: %v", err)
	}
	if after.ReceiptHash != before.ReceiptHash {
		t.Error("eine Prüfung darf den Beleg-Hash nicht verändern")
	}

	result, err := env.journal.VerifyIntegrity(ctx)
	if err != nil {
		t.Fatalf("Integritätsprüfung: %v", err)
	}
	if !result.IsValid {
		t.Errorf("die Kette muss nach einer Prüfung heil sein: %s", result.Message)
	}
}
