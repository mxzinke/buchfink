package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/invoice"
	"github.com/buchfink/buchfink/internal/repository"
)

// invoicesWired liefert den Rechnungsdienst mit der Verdrahtung der Welle 5b:
// Transaktionsklammer, Lückenbericht, Rechnungsverbund und Geschäftsjahre.
//
// Ohne Beleg-Pipeline — die meisten dieser Prüfungen betreffen Nummernvergabe
// und Buchung, und das Übersetzen des WASM-Moduls kostet Sekunden.
func (e *testEnv) invoicesWired(t *testing.T) *InvoiceService {
	t.Helper()
	svc := e.invoices(t)
	svc.SetRegistry(InvoiceRegistry{
		Tx:          repository.NewTxRunner(e.db),
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
// Das ist der Kern der Welle: § 14 Abs. 4 Nr. 4 UStG verlangt eine einmalige,
// fortlaufende Nummer, GoBD Rz. 42 verlangt sie lückenlos. Vorher lief die
// Nummernvergabe in einer eigenen Transaktion vor der Buchung — und jede
// gescheiterte Buchung ließ eine verbrauchte Nummer ohne Rechnung zurück.
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

// Scheitert das Dokument, bleiben Nummer und Buchung stehen — sichtbar als
// „Dokument fehlt". Die Nummer ist nicht verloren, das Dokument lässt sich
// nachholen.
func TestFailedDocumentLeavesInvoicePendingAndRecoverable(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")

	inv := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	err := env.invoicesWithBrokenRenderer(t).Issue(ctx, inv)
	if err == nil {
		t.Fatal("das Scheitern der Dokumenterzeugung muss gemeldet werden")
	}
	if !strings.Contains(err.Error(), "nachholen") {
		t.Errorf("die Meldung sagt nicht, dass sich das Dokument nachholen lässt: %v", err)
	}

	stored, err := env.invoiceRepoOf(t).FindByNumber(ctx, "RE-2026-0001")
	if err != nil {
		t.Fatalf("die Rechnung muss mit ihrer Nummer gespeichert sein: %v", err)
	}
	if stored.Status != domain.InvoiceStatusPendingDocument {
		t.Errorf("Status = %q, erwartet %q", stored.Status, domain.InvoiceStatusPendingDocument)
	}
	if stored.JournalEntryID == nil {
		t.Error("die Rechnung muss gebucht sein — die Buchung war nicht das Problem")
	}
	if stored.ReceiptID != nil {
		t.Error("ohne erzeugtes Dokument darf kein Beleg entstanden sein")
	}

	// Die verbrauchte Nummer trägt eine Rechnung: keine Lücke.
	report, err := env.invoicesWired(t).NumberGaps(ctx, 2026)
	if err != nil {
		t.Fatalf("Lückenbericht: %v", err)
	}
	if len(report.Gaps) != 0 {
		t.Errorf("der Bericht meldet %d Lücken, erwartet keine", len(report.Gaps))
	}

	if testing.Short() {
		t.Skip("das Nachholen braucht den PDF-Renderer; die WASM-Kompilierung ist zu langsam für -short")
	}
	svc := env.invoicesWired(t)
	svc.SetDocumentPipeline(env.receipts, sharedRenderer())
	repaired, err := svc.RegenerateDocument(ctx, stored.ID)
	if err != nil {
		t.Fatalf("Dokument nachholen: %v", err)
	}
	if repaired.Status != domain.InvoiceStatusIssued || repaired.ReceiptID == nil {
		t.Fatalf("nach dem Nachholen: Status %q, Beleg %v", repaired.Status, repaired.ReceiptID)
	}
	receipt, err := env.receipts.Get(ctx, *repaired.ReceiptID)
	if err != nil {
		t.Fatalf("Beleg laden: %v", err)
	}
	if receipt.ReceiptNumber != repaired.InvoiceNumber {
		t.Errorf("der Beleg trägt %q, die Rechnung %q", receipt.ReceiptNumber, repaired.InvoiceNumber)
	}
	// Der Validierungsbericht der eigenen Rechnung gehört an den Beleg: sonst
	// lässt sich Jahre später nicht mehr belegen, gegen welches Regelwerk
	// geprüft wurde.
	if receipt.ValidatedAt == "" || receipt.ValidationRuleset == "" {
		t.Errorf("am Beleg fehlt der Validierungsbericht: %+v", receipt)
	}

	// Ein zweites Dokument zur selben Nummer wäre ein zweiter Beleg zu einem
	// Vorgang.
	if _, err := svc.RegenerateDocument(ctx, stored.ID); err == nil {
		t.Error("ein zweites Dokument zur selben Rechnung darf nicht entstehen")
	}
}

// Der Lückenbericht vergleicht den Zählerstand mit den vergebenen Nummern und
// nennt zu jeder Lücke den dokumentierten Grund.
func TestInvoiceNumberGapReport(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)

	for i := 0; i < 3; i++ {
		inv := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
		if err := svc.Issue(ctx, inv); err != nil {
			t.Fatalf("Rechnung %d ausstellen: %v", i+1, err)
		}
	}
	report, err := svc.NumberGaps(ctx, 2026)
	if err != nil {
		t.Fatalf("Lückenbericht: %v", err)
	}
	if report.Issued != 3 || len(report.Gaps) != 0 {
		t.Fatalf("drei ausgestellte Rechnungen, keine Lücke — erhalten %+v", report)
	}

	// Ein Eingriff von außen: die zweite Rechnung verschwindet aus dem Bestand.
	// Genau das soll der Bericht sichtbar machen, denn der Zähler weiß noch,
	// dass die Nummer vergeben war.
	if err := env.db.Where("invoice_number = ?", "RE-2026-0002").
		Delete(&domain.Invoice{}).Error; err != nil {
		t.Fatalf("Rechnung entfernen: %v", err)
	}
	report, err = svc.NumberGaps(ctx, 2026)
	if err != nil {
		t.Fatalf("Lückenbericht: %v", err)
	}
	if len(report.Gaps) != 1 || report.Gaps[0].Number != "RE-2026-0002" {
		t.Fatalf("erwartet die Lücke RE-2026-0002, erhalten %+v", report.Gaps)
	}
	if report.Gaps[0].Reason != domain.NumberGapUnknown {
		t.Errorf("eine unbegründete Lücke muss als solche erscheinen, ist aber %q", report.Gaps[0].Reason)
	}

	if err := svc.RecordNumberGapReason(ctx, 2026, 2, domain.NumberGapTest, "Probelauf bei der Einrichtung"); err != nil {
		t.Fatalf("Grund dokumentieren: %v", err)
	}
	report, err = svc.NumberGaps(ctx, 2026)
	if err != nil {
		t.Fatalf("Lückenbericht: %v", err)
	}
	if len(report.Gaps) != 1 || report.Gaps[0].Reason != domain.NumberGapTest {
		t.Fatalf("der dokumentierte Grund fehlt: %+v", report.Gaps)
	}
	if report.Gaps[0].Detail != "Probelauf bei der Einrichtung" || report.Gaps[0].RecordedAt == "" {
		t.Errorf("Begründung und Zeitpunkt gehören an die Lücke: %+v", report.Gaps[0])
	}

	// Der Grund steht auch im Protokoll und nicht nur in der Tabelle. Er wird
	// nachträglich erfasst; wer wann welche Begründung eingetragen hat, ist
	// dann selbst eine Angabe (GoBD Rz. 58) — und es ist die Angabe, nach der
	// die Betriebsprüfung fragt.
	entries, err := repository.NewAuditRepository(env.db).FindAll(ctx, 200)
	if err != nil {
		t.Fatal(err)
	}
	logged := false
	for _, e := range entries {
		if e.EntityType == "NUMBER_GAP" && strings.Contains(e.Details, "RE-2026-0002") &&
			strings.Contains(e.Details, "Probelauf bei der Einrichtung") {
			logged = true
		}
	}
	if !logged {
		t.Errorf("die Begründung der Lücke RE-2026-0002 fehlt im Protokoll: %+v", entries)
	}
}

// Das Nummernformat ist einstellbar; die Rechnung folgt der Einstellung.
func TestConfiguredInvoiceNumberFormatIsUsed(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	settings := repository.NewSettingsRepository(env.db)
	cfg, err := settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cfg.InvoiceNumberFormat = "AR-{JAHR}-{NR:5}"
	if err := settings.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatal(err)
	}

	customer := env.customer(t, "Kunde GmbH", "DE", "")
	inv := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := env.invoicesWired(t).Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}
	if inv.InvoiceNumber != "AR-2026-00001" {
		t.Errorf("Rechnungsnummer = %q, erwartet AR-2026-00001", inv.InvoiceNumber)
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

	// Bei einer innergemeinschaftlichen Lieferung ist sie nicht wählbar
	// (§ 33 Satz 2 UStDV).
	eu := env.customer(t, "Kunde AT", "AT", "ATU12345678")
	ig := env.simpleInvoice(eu.ID, "2026-03-01", 10000)
	ig.TaxTreatment = domain.TaxTreatmentIntraCommunitySupply
	ig.Items[0].TaxRate = domain.TaxRateNone
	ig.SmallAmount = true
	err = svc.Issue(ctx, ig)
	if err == nil || !strings.Contains(err.Error(), "§ 33 Satz 2 UStDV") {
		t.Errorf("bei ig. Lieferung ist die Kleinbetragsrechnung nicht zulässig, erhalten: %v", err)
	}

	// Die Ausfuhr nimmt § 33 Satz 2 UStDV nicht aus — die Vorschrift zählt nur
	// Fernverkauf, ig. Lieferung und § 13b auf. Buchfink bietet die
	// Kleinbetragsrechnung trotzdem nur für den steuerpflichtigen Inlandsumsatz
	// an; die Meldung muss das als eigene Entscheidung sagen und sich nicht auf
	// eine Norm berufen, die sie nicht deckt.
	thirdCountry := env.customer(t, "Kunde CH", "CH", "")
	export := env.simpleInvoice(thirdCountry.ID, "2026-03-01", 10000)
	export.TaxTreatment = domain.TaxTreatmentExport
	export.Items[0].TaxRate = domain.TaxRateNone
	export.SmallAmount = true
	err = svc.Issue(ctx, export)
	if err == nil {
		t.Error("die Kleinbetragsrechnung ist nur für den Inlandsumsatz vorgesehen")
	} else if strings.Contains(err.Error(), "§ 33 Satz 2 UStDV") {
		t.Errorf("die Ausfuhr steht nicht in § 33 Satz 2 UStDV; die Meldung darf sie nicht darauf "+
			"stützen: %v", err)
	} else if !strings.Contains(err.Error(), "Inlandsumsatz") {
		t.Errorf("die Meldung muss den Grund nennen, erhalten: %v", err)
	}

	// Innerhalb der Grenze und im Inland geht sie durch — auch ohne Empfänger.
	ok := smallAmountSale("2026-03-01", 10000)
	if err := svc.Issue(ctx, ok); err != nil {
		t.Fatalf("die Kleinbetragsrechnung ohne Empfänger muss ausstellbar sein: %v", err)
	}
	if ok.JournalEntryID == nil {
		t.Fatal("auch der Barverkauf wird gebucht")
	}
}

// smallAmountSale ist die Kleinbetragsrechnung ohne Empfänger: der Barverkauf
// des § 33 UStDV.
func smallAmountSale(date string, net domain.Cents) *domain.Invoice {
	return &domain.Invoice{
		Date: date, ServiceDateFrom: date, ServiceDateTo: date,
		TaxTreatment: domain.TaxTreatmentDomestic, SmallAmount: true,
		Items: []domain.InvoiceItem{{
			Description: "Verkauf über den Ladentisch", QuantityMilli: 1000, Unit: "C62",
			UnitPrice: net, TaxRate: domain.TaxRateStandard,
		}},
	}
}

// Die Kleinbetragsrechnung ohne Empfänger wird gegen das Zahlungsmittel
// gebucht.
//
// § 33 UStDV lässt den Leistungsempfänger weg. Ohne ihn gibt es kein
// Personenkonto — und es gibt auch nichts, was darauf stünde: der Barverkauf
// ist bezahlt, wenn die Rechnung entsteht.
func TestSmallAmountInvoiceWithoutRecipientIsBookedAsCashSale(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.invoicesWired(t)

	inv := smallAmountSale("2026-03-01", 10000)
	if err := svc.Issue(ctx, inv); err != nil {
		t.Fatalf("Kleinbetragsrechnung ausstellen: %v", err)
	}
	if inv.JournalEntryID == nil {
		t.Fatal("die Kleinbetragsrechnung muss gebucht sein")
	}

	entry, err := env.journalRepo.FindByID(ctx, *inv.JournalEntryID)
	if err != nil {
		t.Fatal(err)
	}
	if got := lineAmount(t, entry, domain.AccountKasse, domain.SideDebit); got != 11900 {
		t.Errorf("Kasse im Soll = %s, erwartet 119,00", got)
	}
	if got := lineAmount(t, entry, domain.AccountUmsatzsteuer19, domain.SideCredit); got != 1900 {
		t.Errorf("Umsatzsteuer im Haben = %s, erwartet 19,00", got)
	}
	if entry.ContactID != nil {
		t.Error("ohne Empfänger darf die Buchung keinem Personenkonto zugeordnet sein")
	}

	// Das Zahlungsmittel ist wählbar: wer die Karte nimmt, bucht nicht in die
	// Kasse.
	onBank := smallAmountSale("2026-03-02", 10000)
	onBank.PaymentAccount = domain.AccountBank
	if err := svc.Issue(ctx, onBank); err != nil {
		t.Fatalf("Kleinbetragsrechnung über die Bank: %v", err)
	}
	bankEntry, err := env.journalRepo.FindByID(ctx, *onBank.JournalEntryID)
	if err != nil {
		t.Fatal(err)
	}
	if got := lineAmount(t, bankEntry, domain.AccountBank, domain.SideDebit); got != 11900 {
		t.Errorf("Bank im Soll = %s, erwartet 119,00", got)
	}

	// Und die Vorschau kennt denselben Fall: sonst hätte der Barverkauf als
	// einziger Rechnungsfall keine.
	preview, err := svc.Preview(ctx, smallAmountSale("2026-03-03", 10000))
	if err != nil {
		t.Fatalf("Vorschau ohne Empfänger: %v", err)
	}
	if !preview.Balanced || preview.Gross != 11900 {
		t.Errorf("Vorschau = %+v, erwartet 119,00 ausgeglichen", preview)
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

// Scheitert die Generalumkehr, entsteht kein Stornodokument und keine Nummer
// wird verbraucht.
//
// Das ist die Klammer der Entscheidung 1, angewandt auf den Storno: Nummer,
// Dokument und Buchung entstehen zusammen oder gar nicht. Vorher lag die
// Generalumkehr hinter der Nummernvergabe — ein nummeriertes Stornodokument
// ohne Buchung blieb zurück, und der nächste Versuch verbrauchte eine zweite
// Nummer.
func TestCancelIsAtomicWhenTheReversalFails(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)

	original := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := svc.Issue(ctx, original); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}
	// Die Buchung wird außerhalb des Rechnungswegs storniert. Der Storno über
	// den Rechnungsweg muss daran scheitern — eine Buchung lässt sich nur einmal
	// durch Generalumkehr zurücknehmen.
	if _, err := env.journal.Reverse(ctx, *original.JournalEntryID, "Korrektur im Journal"); err != nil {
		t.Fatalf("Generalumkehr im Journal: %v", err)
	}

	_, err := svc.CancelWithDocument(ctx, original.ID, "Leistung nicht erbracht")
	if err == nil {
		t.Fatal("der Storno muss scheitern, wenn die Generalumkehr nicht möglich ist")
	}

	// Keine zweite Nummer verbraucht …
	next, err := env.numberRepo.Peek(ctx, domain.NumberRangeInvoice, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if next != 2 {
		t.Errorf("der Zähler steht auf %d, erwartet 2 — der gescheiterte Storno hat eine Nummer verbraucht", next)
	}
	// … kein Stornodokument entstanden …
	numbers, err := env.invoiceRepoOf(t).FindNumbers(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(numbers) != 1 {
		t.Errorf("erwartet nur die Ursprungsrechnung, erhalten %v", numbers)
	}
	// … und die Ursprungsrechnung steht unverändert da.
	reloaded, err := env.invoiceRepoOf(t).FindByID(ctx, original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Status == domain.InvoiceStatusCancelled || reloaded.CancelledByInvoiceID != nil {
		t.Errorf("die Ursprungsrechnung gilt als storniert, obwohl der Storno gescheitert ist: %q", reloaded.Status)
	}
	// Und der Lückenbericht meldet keine Lücke: die Transaktion ist
	// zurückgerollt.
	report, err := svc.NumberGaps(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Gaps) != 0 {
		t.Errorf("der Bericht meldet %d Lücken, erwartet keine: %+v", len(report.Gaps), report.Gaps)
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

// 2027 hängt die Übergangsregel am Vorjahresumsatz: über 800.000 € ist sie
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

// Der Storno: eigenes Dokument mit eigener Nummer, negierte Beträge, Bezug auf
// die Ursprungsrechnung, Generalumkehr im Journal — und die Ursprungsrechnung
// bleibt unverändert archiviert.
func TestCancelWithDocument(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)

	original := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := svc.Issue(ctx, original); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}

	if _, err := svc.CancelWithDocument(ctx, original.ID, ""); err == nil {
		t.Error("ein Storno ohne Begründung darf nicht durchgehen (GoBD Rz. 58)")
	}

	storno, err := svc.CancelWithDocument(ctx, original.ID, "Leistung nicht erbracht")
	if err != nil {
		t.Fatalf("stornieren: %v", err)
	}

	if storno.InvoiceNumber == original.InvoiceNumber {
		t.Error("das Stornodokument braucht eine eigene Nummer")
	}
	if storno.Kind != domain.InvoiceKindCancellation {
		t.Errorf("Dokumentart = %q, erwartet %q", storno.Kind, domain.InvoiceKindCancellation)
	}
	if storno.GrossAmount != -original.GrossAmount {
		t.Errorf("Storno über %s, erwartet %s", storno.GrossAmount, -original.GrossAmount)
	}
	if storno.CorrectsInvoiceNumber != original.InvoiceNumber || storno.CorrectsInvoiceDate != original.Date {
		t.Errorf("der Bezug auf die stornierte Rechnung fehlt: %q vom %q",
			storno.CorrectsInvoiceNumber, storno.CorrectsInvoiceDate)
	}

	// Die Kette ist in beide Richtungen lesbar.
	reloaded, err := env.invoiceRepoOf(t).FindByID(ctx, original.ID)
	if err != nil {
		t.Fatalf("Ursprungsrechnung laden: %v", err)
	}
	if reloaded.Status != domain.InvoiceStatusCancelled {
		t.Errorf("die Ursprungsrechnung steht auf %q, erwartet %q", reloaded.Status, domain.InvoiceStatusCancelled)
	}
	if reloaded.CancelledByInvoiceID == nil || *reloaded.CancelledByInvoiceID != storno.ID {
		t.Error("die Ursprungsrechnung muss auf ihr Stornodokument zeigen")
	}
	// Unverändert archiviert: Beträge und Positionen bleiben, wie sie waren.
	if reloaded.GrossAmount != original.GrossAmount || len(reloaded.Items) != 1 {
		t.Error("die Ursprungsrechnung darf nicht verändert werden")
	}

	// Und im Journal steht die Generalumkehr.
	reversal, err := env.journalRepo.FindReversalOf(ctx, *original.JournalEntryID)
	if err != nil || reversal == nil {
		t.Fatalf("die Generalumkehr fehlt: %v", err)
	}
	for _, l := range reversal.Lines {
		if l.Amount > 0 {
			t.Errorf("eine Generalumkehr trägt negative Beträge, hier steht %s auf %s", l.Amount, l.Account)
		}
	}
	if storno.JournalEntryID == nil || *storno.JournalEntryID != reversal.ID {
		t.Error("das Stornodokument muss auf die Generalumkehr zeigen")
	}

	// Ein zweites Storno gibt es nicht.
	if _, err := svc.CancelWithDocument(ctx, original.ID, "nochmal"); err == nil {
		t.Error("eine bereits stornierte Rechnung darf nicht erneut storniert werden")
	}
}

// Ein Storno des Stornos gibt es nicht — weder über den Storno- noch über den
// Berichtigungsweg.
//
// Der Journalweg fängt den Fall nur zufällig ab: eine Generalumkehr lässt sich
// nicht erneut umkehren. Bei der Abschlagsrechnung greift das nicht, denn sie
// bucht beim Ausstellen nichts — ihr Stornodokument hat keine Buchung, die sich
// sperren ließe, und ohne die Prüfung der Dokumentart entstünde ein zweites
// Dokument mit den wieder positiven Beträgen der Ursprungsrechnung.
func TestCancellationCannotBeCancelledOrCorrected(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)

	original := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := svc.Issue(ctx, original); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}
	storno, err := svc.CancelWithDocument(ctx, original.ID, "Leistung nicht erbracht")
	if err != nil {
		t.Fatalf("stornieren: %v", err)
	}

	// Das Stornodokument steht auf „ausgestellt" wie jede Rechnung; die Sperre
	// darf deshalb nicht am Status hängen, sondern an der Dokumentart.
	if !storno.Status.IsIssued() {
		t.Fatalf("das Stornodokument steht auf %q, erwartet ausgestellt", storno.Status)
	}

	_, err = svc.CancelWithDocument(ctx, storno.ID, "doch nicht")
	if err == nil || !strings.Contains(err.Error(), "Stornorechnung") {
		t.Errorf("das Storno einer Stornorechnung muss mit einem Hinweis auf die Dokumentart "+
			"abgewiesen werden, erhalten: %v", err)
	}
	if err != nil && !strings.Contains(err.Error(), original.InvoiceNumber) {
		t.Errorf("die Abweisung soll die berichtigbare Rechnung nennen, erhalten: %v", err)
	}

	replacement := env.simpleInvoice(customer.ID, "2026-03-20", 90000)
	_, err = svc.CorrectInvoice(ctx, storno.ID, "doch nicht", replacement)
	if err == nil || !strings.Contains(err.Error(), "Stornorechnung") {
		t.Errorf("die Berichtigung einer Stornorechnung muss abgewiesen werden, erhalten: %v", err)
	}

	// Und der abgewiesene Versuch hat nichts angefasst.
	reloaded, err := env.invoiceRepoOf(t).FindByID(ctx, storno.ID)
	if err != nil {
		t.Fatalf("Stornodokument laden: %v", err)
	}
	if reloaded.Status != storno.Status || reloaded.CancelledByInvoiceID != nil {
		t.Errorf("das Stornodokument steht nach dem abgewiesenen Versuch auf %q (storniert durch %v)",
			reloaded.Status, reloaded.CancelledByInvoiceID)
	}
	if replacement.InvoiceNumber != "" {
		t.Errorf("die abgewiesene Berichtigung hat die Nummer %s verbraucht", replacement.InvoiceNumber)
	}

	// Der Abschlag: sein Storno trägt keine Buchung, und der Journalweg sperrt
	// hier nichts.
	group := env.group(t, svc, customer.ID, 1000000)
	advance, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-03-01", Net: 400000,
	})
	if err != nil {
		t.Fatalf("Abschlagsrechnung ausstellen: %v", err)
	}
	advanceStorno, err := svc.CancelWithDocument(ctx, advance.ID, "Auftrag geändert")
	if err != nil {
		t.Fatalf("Abschlag stornieren: %v", err)
	}
	if advanceStorno.JournalEntryID != nil {
		t.Fatal("das Storno eines nicht vereinnahmten Abschlags bucht nichts; der Test prüft sonst den Journalweg")
	}
	if _, err := svc.CancelWithDocument(ctx, advanceStorno.ID, "doch nicht"); err == nil {
		t.Error("auch das Storno eines Abschlagsstornos muss abgewiesen werden")
	}
}

// Die Berichtigung: Storno plus neue Rechnung mit Bezug auf die berichtigte.
func TestCorrectInvoiceIssuesReplacementWithReference(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)

	original := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := svc.Issue(ctx, original); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}

	replacement := env.simpleInvoice(customer.ID, "2026-03-15", 120000)
	corrected, err := svc.CorrectInvoice(ctx, original.ID, "Falscher Stundensatz", replacement)
	if err != nil {
		t.Fatalf("berichtigen: %v", err)
	}
	if corrected.Kind != domain.InvoiceKindCorrection {
		t.Errorf("die berichtigte Rechnung ist %q, erwartet %q", corrected.Kind, domain.InvoiceKindCorrection)
	}
	if corrected.CorrectsInvoiceNumber != original.InvoiceNumber {
		t.Errorf("der Bezug fehlt: %q", corrected.CorrectsInvoiceNumber)
	}
	if corrected.JournalEntryID == nil {
		t.Error("die berichtigte Rechnung muss gebucht sein")
	}
	if corrected.NetAmount != 120000 {
		t.Errorf("die berichtigte Rechnung lautet über %s, erwartet 1200,00", corrected.NetAmount)
	}

	// Drei Dokumente, drei Nummern: Ursprung, Storno, Berichtigung.
	numbers, err := env.invoiceRepoOf(t).FindNumbers(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(numbers) != 3 {
		t.Errorf("erwartet drei Dokumente im Nummernkreis, erhalten %d: %v", len(numbers), numbers)
	}
}

// Der Versandvermerk hält fest, wann und wie die Rechnung hinausgegangen ist.
func TestMarkInvoiceSent(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)

	inv := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := svc.Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}

	if _, err := svc.MarkSent(ctx, inv.ID, "2026-03-02", "Brieftaube", ""); err == nil {
		t.Error("ein unbekannter Versandweg muss abgewiesen werden")
	}

	sent, err := svc.MarkSent(ctx, inv.ID, "2026-03-02", domain.InvoiceSentViaEmail, "an buchhaltung@kunde.example")
	if err != nil {
		t.Fatalf("Versand vermerken: %v", err)
	}
	if sent.SentAt != "2026-03-02" || sent.SentVia != domain.InvoiceSentViaEmail {
		t.Errorf("Versandvermerk = %q / %q", sent.SentAt, sent.SentVia)
	}
	stored, err := env.invoiceRepoOf(t).FindByID(ctx, inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.SentNote != "an buchhaltung@kunde.example" {
		t.Errorf("die Notiz zum Versand fehlt: %q", stored.SentNote)
	}
}

// Kein `null` in den Listen, die an die Oberfläche gehen: die Ansicht liest
// `gaps.length` und `advances.length` ohne Umweg, und `null.length` nimmt im
// Render den ganzen Baum mit.
func TestWelle5bListsMarshalEmptyNotNull(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)

	report, err := svc.NumberGaps(ctx, 2026)
	if err != nil {
		t.Fatalf("Lückenbericht: %v", err)
	}
	assertNoNullLists(t, "Lückenbericht", report, "gaps")

	group := env.group(t, svc, customer.ID, 1000000)
	assertNoNullLists(t, "Rechnungsverbund", group, "advances")

	inv := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := svc.Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}
	stored, err := env.invoiceRepoOf(t).FindByID(ctx, inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertNoNullLists(t, "Rechnung", stored, "items", "precedingRefs")
}
