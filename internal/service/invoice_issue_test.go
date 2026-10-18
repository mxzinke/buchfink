package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/invoice"
	"github.com/buchfink/buchfink/internal/repository"
)

// invoicesWired liefert den Rechnungsdienst mit
// Transaktionsklammer, Lückenbericht, Rechnungsverbund und Geschäftsjahre.
//
// Ohne Beleg-Pipeline — die meisten dieser Prüfungen betreffen Nummernvergabe
// und Buchung, und das Übersetzen des WASM-Moduls kostet Sekunden.
func (e *testEnv) invoicesWired(t *testing.T) *InvoiceService {
	t.Helper()
	svc := e.invoices(t)
	svc.SetRegistry(InvoiceRegistry{
		Gaps:        repository.NewNumberGapRepository(e.db),
		Groups:      repository.NewInvoiceGroupRepository(e.db),
		FiscalYears: repository.NewFiscalYearRepository(e.db),
		Bank:        repository.NewBankRepository(e.db),
	})
	return svc
}

// invoicesWithBrokenRenderer ist derselbe Dienst mit einem Renderer, der nichts
// erzeugen kann.
//
// Das ist kein Behelf, sondern der Prüfgegenstand: ein geschlossener Renderer
// lässt die Dokumenterzeugung scheitern, nachdem Nummer und Buchung stehen —
// genau der Fall, für den es den Zustand „Dokument fehlt" gibt.
func (e *testEnv) invoicesWithBrokenRenderer(t *testing.T) *InvoiceService {
	t.Helper()
	svc := e.invoicesWired(t)
	broken := invoice.NewRenderer()
	if err := broken.Close(context.Background()); err != nil {
		t.Fatalf("Renderer schließen: %v", err)
	}
	svc.SetDocumentPipeline(e.receipts, broken)
	return svc
}

func (e *testEnv) simpleInvoice(customerID uint, date string, net domain.Cents) *domain.Invoice {
	return &domain.Invoice{
		ContactID: customerID, Date: date,
		ServiceDateFrom: date, ServiceDateTo: date,
		TaxTreatment: domain.TaxTreatmentDomestic,
		Items: []domain.InvoiceItem{{
			Description: "Beratung", QuantityMilli: 1000, Unit: "HUR",
			UnitPrice: net, TaxRate: domain.TaxRateStandard,
		}},
	}
}

// Scheitert die Buchung, ist die Nummer nicht verbraucht und die Rechnung nicht
// gespeichert.
//
// Nummernvergabe und Buchung gehören in dieselbe Transaktion, damit eine
// gescheiterte Buchung keine Nummer verbraucht (§ 14 Abs. 4 Nr. 4 UStG,
// GoBD Rz. 42).
func TestFailedPostingConsumesNoInvoiceNumber(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")

	// Istversteuerung lässt die Buchung scheitern — nach der Nummernvergabe.
	settings := repository.NewSettingsRepository(env.db)
	cfg, err := settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Einstellungen: %v", err)
	}
	cfg.TaxationType = "IST"
	if err := settings.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatalf("Einstellungen setzen: %v", err)
	}

	svc := env.invoicesWired(t)
	inv := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := svc.Issue(ctx, inv); err == nil {
		t.Fatal("die Rechnung wurde trotz Istversteuerung ausgestellt")
	}

	next, err := env.numberRepo.Peek(ctx, domain.NumberRangeInvoice, 2026)
	if err != nil {
		t.Fatalf("Nummernkreis lesen: %v", err)
	}
	if next != 1 {
		t.Errorf("der Zähler steht auf %d, erwartet 1 — die gescheiterte Buchung hat eine Nummer verbraucht", next)
	}

	invoices, err := env.invoiceRepoOf(t).FindAll(ctx, 2026)
	if err != nil {
		t.Fatalf("Rechnungen lesen: %v", err)
	}
	if len(invoices) != 0 {
		t.Errorf("es steht %d Rechnung(en) in der Datenbank, erwartet keine", len(invoices))
	}

	// Und der Lückenbericht meldet keine Lücke: es gibt keine.
	report, err := svc.NumberGaps(ctx, 2026)
	if err != nil {
		t.Fatalf("Lückenbericht: %v", err)
	}
	if len(report.Gaps) != 0 {
		t.Errorf("der Bericht meldet %d Lücken, erwartet keine: %+v", len(report.Gaps), report.Gaps)
	}
}

// invoiceRepoOf liefert ein Rechnungsrepository auf derselben Datenbank.
func (e *testEnv) invoiceRepoOf(t *testing.T) domain.InvoiceRepository {
	t.Helper()
	return repository.NewInvoiceRepository(e.db)
}

// invoiceGroupRepoOf liefert das Repository der Rechnungsverbünde auf derselben
// Datenbank.
func (e *testEnv) invoiceGroupRepoOf(t *testing.T) domain.InvoiceGroupRepository {
	t.Helper()
	return repository.NewInvoiceGroupRepository(e.db)
}

// Scheitert das Dokument, gibt es weder Rechnung noch Buchung, und die Nummer
// ist wieder frei: Nummer, Dokument und Buchung sind eine Transaktion.
func TestFailedDocumentLeavesNoInvoiceAndFreesTheNumber(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")

	inv := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := env.invoicesWithBrokenRenderer(t).Issue(ctx, inv); err == nil {
		t.Fatal("das Scheitern der Dokumenterzeugung muss gemeldet werden")
	}
	if inv.InvoiceNumber != "" || inv.ID != 0 || inv.JournalEntryID != nil || inv.ReceiptID != nil {
		t.Errorf("die Rechnung trägt nach dem Rollback noch Schlüssel: %+v", inv)
	}
	if _, err := env.invoiceRepoOf(t).FindByNumber(ctx, "RE-2026-0001"); err == nil {
		t.Error("ohne Dokument darf die Rechnung nicht stehen bleiben")
	}
	if entries, err := env.journalRepo.FindAll(ctx, 2026); err != nil || len(entries) != 0 {
		t.Errorf("%d Buchungen nach dem Fehlschlag (%v), erwartet keine", len(entries), err)
	}
	report, err := env.invoicesWired(t).NumberGaps(ctx, 2026)
	if err != nil {
		t.Fatalf("Lückenbericht: %v", err)
	}
	if len(report.Gaps) != 0 {
		t.Errorf("der Bericht meldet %d Lücken, erwartet keine", len(report.Gaps))
	}

	if testing.Short() {
		t.Skip("der zweite Versuch braucht den PDF-Renderer; die WASM-Kompilierung ist zu langsam für -short")
	}
	svc := env.invoicesWiredWithDocuments(t)
	if err := svc.Issue(ctx, inv); err != nil {
		t.Fatalf("zweiter Versuch: %v", err)
	}
	if inv.InvoiceNumber != "RE-2026-0001" || inv.Status != domain.InvoiceStatusIssued {
		t.Errorf("Nummer %s, Status %q — erwartet RE-2026-0001, ausgestellt", inv.InvoiceNumber, inv.Status)
	}
	if inv.ReceiptID == nil || inv.JournalEntryID == nil {
		t.Fatalf("die Rechnung muss Beleg und Buchung haben: %+v", inv)
	}
	receipt, err := env.receipts.Get(ctx, *inv.ReceiptID)
	if err != nil {
		t.Fatalf("Beleg laden: %v", err)
	}
	if receipt.Status != domain.ReceiptStatusSealed || receipt.JournalEntryID == nil || *receipt.JournalEntryID != *inv.JournalEntryID {
		t.Errorf("der Beleg muss auf die Buchung versiegelt sein: %+v", receipt)
	}
	// Der Validierungsbericht der eigenen Rechnung gehört an den Beleg.
	if receipt.ValidatedAt == "" || receipt.ValidationRuleset == "" {
		t.Errorf("am Beleg fehlt der Validierungsbericht: %+v", receipt)
	}
	entry, err := env.journalRepo.FindByID(ctx, *inv.JournalEntryID)
	if err != nil {
		t.Fatal(err)
	}
	if entry.ReceiptID == nil || *entry.ReceiptID != *inv.ReceiptID {
		t.Errorf("die Buchung muss auf den Beleg der Rechnung verweisen: %v", entry.ReceiptID)
	}
	if entry.ReceiptHash == "" || entry.ReceiptHash != receipt.ReceiptHash {
		t.Errorf("die Buchung muss den Beleg-Hash tragen: %q, Beleg %q", entry.ReceiptHash, receipt.ReceiptHash)
	}
	if _, err := svc.RegenerateDocument(ctx, inv.ID); err == nil {
		t.Error("ein zweites Dokument zur selben Rechnung darf nicht entstehen")
	}
}

// Ohne vollständige Anschrift des Empfängers entsteht keine Rechnung: § 14
// Abs. 4 Nr. 1 UStG macht sie zur Pflichtangabe, und ihr Fehlen kostet den
// Empfänger den Vorsteuerabzug.
func TestIssueRequiresCompleteBuyerAddress(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	incomplete := &domain.Contact{
		Type: domain.ContactTypeCustomer, Name: "Kunde ohne Anschrift", CountryCode: "DE",
	}
	if err := env.contacts.SaveContact(ctx, incomplete); err != nil {
		t.Fatalf("Kunde anlegen: %v", err)
	}

	inv := env.simpleInvoice(incomplete.ID, "2026-03-01", 100000)
	err := env.invoicesWired(t).Issue(ctx, inv)
	if err == nil {
		t.Fatal("eine Rechnung ohne Empfängeranschrift darf nicht entstehen")
	}
	if !strings.Contains(err.Error(), "Straße") {
		t.Errorf("die Meldung nennt das fehlende Feld nicht: %v", err)
	}
	// Und keine Nummer verbraucht: die Prüfung läuft vor der Vergabe.
	next, _ := env.numberRepo.Peek(ctx, domain.NumberRangeInvoice, 2026)
	if next != 1 {
		t.Errorf("die abgewiesene Rechnung hat eine Nummer verbraucht (Zähler steht auf %d)", next)
	}
}

// Die Kleinbetragsrechnung ist an zwei Bedingungen gebunden: den Betrag und den
// Steuerfall (§ 33 UStDV).
func TestSmallAmountInvoiceLimits(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)

	// 300 € brutto liegen über der Grenze von 250 €.
	over := env.simpleInvoice(customer.ID, "2026-03-01", 30000)
	over.SmallAmount = true
	err := svc.Issue(ctx, over)
	if err == nil || !strings.Contains(err.Error(), "§ 33 UStDV") {
		t.Errorf("über der Grenze muss die Kleinbetragsrechnung abgewiesen werden, erhalten: %v", err)
	}

	// Bei der Steuerschuldnerschaft des Leistungsempfängers ist sie nicht
	// wählbar (§ 33 Satz 2 UStDV).
	eu := env.customer(t, "Kunde AT", "AT", "ATU12345678")
	rc := env.simpleInvoice(eu.ID, "2026-03-01", 10000)
	rc.TaxTreatment = domain.TaxTreatmentReverseChargeSupply
	rc.Items[0].TaxRate = domain.TaxRateNone
	rc.SmallAmount = true
	err = svc.Issue(ctx, rc)
	if err == nil || !strings.Contains(err.Error(), "§ 33 Satz 2 UStDV") {
		t.Errorf("bei § 13b ist die Kleinbetragsrechnung nicht zulässig, erhalten: %v", err)
	}

	// Den steuerfreien Umsatz nimmt § 33 Satz 2 UStDV nicht aus — die Vorschrift
	// zählt nur Fernverkauf, ig. Lieferung und § 13b auf. Buchfink bietet die
	// Kleinbetragsrechnung trotzdem nur für den steuerpflichtigen Inlandsumsatz
	// an; die Meldung muss das als eigene Entscheidung sagen und sich nicht auf
	// eine Norm berufen, die sie nicht deckt.
	exempt := env.simpleInvoice(customer.ID, "2026-03-01", 10000)
	exempt.TaxTreatment = domain.TaxTreatmentExempt
	exempt.Items[0].TaxRate = domain.TaxRateNone
	exempt.SmallAmount = true
	err = svc.Issue(ctx, exempt)
	if err == nil {
		t.Error("die Kleinbetragsrechnung ist nur für den Inlandsumsatz vorgesehen")
	} else if strings.Contains(err.Error(), "§ 33 Satz 2 UStDV") {
		t.Errorf("der steuerfreie Umsatz steht nicht in § 33 Satz 2 UStDV; die Meldung darf sich nicht "+
			"darauf stützen: %v", err)
	} else if !strings.Contains(err.Error(), "Inlandsumsatz") {
		t.Errorf("die Meldung muss den Grund nennen, erhalten: %v", err)
	}

	// Ohne Empfänger geht auch sie nicht: jede Rechnung läuft gegen ein
	// Personenkonto, einen Barverkauf führt Buchfink nicht.
	anonymous := env.simpleInvoice(0, "2026-03-01", 10000)
	anonymous.SmallAmount = true
	if err := svc.Issue(ctx, anonymous); err == nil || !strings.Contains(err.Error(), "Rechnungsempfänger") {
		t.Errorf("eine Kleinbetragsrechnung ohne Empfänger muss abgewiesen werden, erhalten: %v", err)
	}
	if _, err := svc.Preview(ctx, env.simpleInvoice(0, "2026-03-01", 10000)); err == nil {
		t.Error("auch die Vorschau verlangt einen Empfänger")
	}

	// Innerhalb der Grenze und im Inland geht sie durch.
	ok := env.simpleInvoice(customer.ID, "2026-03-01", 10000)
	ok.SmallAmount = true
	if err := svc.Issue(ctx, ok); err != nil {
		t.Fatalf("die Kleinbetragsrechnung muss ausstellbar sein: %v", err)
	}
	if ok.JournalEntryID == nil {
		t.Fatal("die Kleinbetragsrechnung wird gebucht")
	}
	entry, err := env.journalRepo.FindByID(ctx, *ok.JournalEntryID)
	if err != nil {
		t.Fatal(err)
	}
	if got := lineAmount(t, entry, customer.LedgerAccount, domain.SideDebit); got != 11900 {
		t.Errorf("Forderung im Soll = %s, erwartet 119,00 auf dem Personenkonto", got)
	}
}

// Eine vorbelegte Rechnungsnummer wird nicht übernommen: sie kommt aus dem
// Nummernkreis (§ 14 Abs. 4 Nr. 4 UStG).
func TestIssueRejectsAPresetInvoiceNumber(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")

	inv := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	inv.InvoiceNumber = "RE-2026-9999"
	err := env.invoicesWired(t).Issue(ctx, inv)
	if err == nil || !strings.Contains(err.Error(), "Nummernkreis") {
		t.Fatalf("eine vorbelegte Nummer muss abgewiesen werden, erhalten: %v", err)
	}
	if next, _ := env.numberRepo.Peek(ctx, domain.NumberRangeInvoice, 2026); next != 1 {
		t.Errorf("der Zähler steht auf %d, erwartet 1", next)
	}
}

// Eine XRechnung ohne Leitweg-ID lässt sich nicht zustellen (BR-DE-15) und wird
// deshalb gar nicht erst ausgestellt.
func TestXRechnungNeedsRouteID(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Behörde", "DE", "")
	customer.EInvoiceProfile = domain.EInvoiceProfileXRechnungCII
	if err := env.contacts.SaveContact(ctx, customer); err != nil {
		t.Fatalf("Kontakt speichern: %v", err)
	}

	inv := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	err := env.invoicesWired(t).Issue(ctx, inv)
	if err == nil || !strings.Contains(err.Error(), "Leitweg-ID") {
		t.Errorf("ohne Leitweg-ID darf keine XRechnung entstehen, erhalten: %v", err)
	}
}

// Ab 2028 ist die sonstige Rechnung ohne strukturierten Datensatz nicht mehr
// zulässig (§ 27 Abs. 38 UStG).
func TestPdfOnlyIsRejectedAfterTheTransition(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	customer.EInvoiceProfile = domain.EInvoiceProfilePDFOnly
	if err := env.contacts.SaveContact(ctx, customer); err != nil {
		t.Fatalf("Kontakt speichern: %v", err)
	}
	svc := env.invoicesWired(t)

	// 2026 ist sie noch erlaubt.
	early := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := svc.Issue(ctx, early); err != nil {
		t.Fatalf("2026 muss die sonstige Rechnung noch durchgehen: %v", err)
	}

	late := env.simpleInvoice(customer.ID, "2028-03-01", 100000)
	err := svc.Issue(ctx, late)
	if err == nil || !strings.Contains(err.Error(), "§ 27 Abs. 38 UStG") {
		t.Errorf("ab 2028 muss die sonstige Rechnung abgewiesen werden, erhalten: %v", err)
	}
}

// 2027 richtet sich die Übergangsregel nach dem Vorjahresumsatz: über 800.000 € ist sie
// verbraucht (§ 27 Abs. 38 Nr. 2 UStG).
func TestPdfOnlyIn2027DependsOnPriorYearRevenue(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	customer.EInvoiceProfile = domain.EInvoiceProfilePDFOnly
	if err := env.contacts.SaveContact(ctx, customer); err != nil {
		t.Fatalf("Kontakt speichern: %v", err)
	}

	years := repository.NewFiscalYearRepository(env.db)
	fy := domain.NewFiscalYear(2027, "2027-01-01", "2027-12-31")
	fy.PriorYearRevenue = 90_000_000 // 900.000 €
	if err := years.Save(ctx, fy); err != nil {
		t.Fatalf("Geschäftsjahr speichern: %v", err)
	}

	svc := env.invoicesWired(t)
	inv := env.simpleInvoice(customer.ID, "2027-03-01", 100000)
	err := svc.Issue(ctx, inv)
	if err == nil || !strings.Contains(err.Error(), "§ 27 Abs. 38 UStG") {
		t.Errorf("über 800.000 € Vorjahresumsatz ist die sonstige Rechnung 2027 nicht mehr zulässig, erhalten: %v", err)
	}

	fy.PriorYearRevenue = 50_000_000 // 500.000 €
	if err := years.Save(ctx, fy); err != nil {
		t.Fatalf("Geschäftsjahr speichern: %v", err)
	}
	inv2 := env.simpleInvoice(customer.ID, "2027-03-01", 100000)
	if err := svc.Issue(ctx, inv2); err != nil {
		t.Errorf("bis 800.000 € Vorjahresumsatz bleibt sie 2027 zulässig: %v", err)
	}
}
