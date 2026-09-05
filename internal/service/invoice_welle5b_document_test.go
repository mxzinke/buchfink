package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Die Ausstellung mit Beleg: XRechnung, Kleinbetragsrechnung und
// Abschlagsrechnung enden alle drei in einem abgelegten Beleg — und was dort
// liegt, entscheidet über den Prüflauf und über den Vorsteuerabzug des
// Empfängers.

// withDocuments liefert den vollständig verdrahteten Dienst: Transaktions-
// klammer, Verbund, Lückenbericht und die Beleg-Pipeline.
func (e *testEnv) invoicesWiredWithDocuments(t *testing.T) *InvoiceService {
	t.Helper()
	svc := e.invoicesWired(t)
	svc.SetDocumentPipeline(e.receipts, sharedRenderer())
	return svc
}

// withSellerContact ergänzt den Ansprechpartner des Verkäufers. Die XRechnung
// verlangt ihn (BR-DE-5 bis BR-DE-7); ohne ihn scheitert die Ausstellung, und
// zwar richtigerweise.
func (e *testEnv) withSellerContact(t *testing.T) {
	t.Helper()
	settings := repository.NewSettingsRepository(e.db)
	cfg, err := settings.GetCompanySettings(context.Background())
	if err != nil {
		t.Fatalf("Unternehmensdaten lesen: %v", err)
	}
	cfg.ContactName = "Marlene Pfennig"
	cfg.ContactPhone = "+49 89 1234567"
	cfg.ContactEmail = "rechnung@pfennig.example"
	// BG-16: ohne Zahlungsanweisung weist die XRechnung die Rechnung zurück.
	cfg.BankName = "Stadtsparkasse München"
	cfg.IBAN = "DE02701500000000594937"
	cfg.BIC = "SSKMDEMM"
	if err := settings.UpdateCompanySettings(context.Background(), cfg); err != nil {
		t.Fatalf("Unternehmensdaten setzen: %v", err)
	}
}

// Die XRechnung: XML als Original und strukturierter Teil, PDF als Darstellung,
// und der Validierungsbericht am Beleg.
//
// Der Bericht ist kein Beiwerk. Wer Jahre später wissen will, gegen welches
// Regelwerk die eigene Rechnung geprüft worden ist, findet es nur dort — und
// ohne ihn ließe sich nicht belegen, dass überhaupt geprüft wurde.
func TestXRechnungIsFiledWithRolesAndValidationReport(t *testing.T) {
	if testing.Short() {
		t.Skip("die WASM-Kompilierung ist zu langsam für -short")
	}
	env := newTestEnv(t)
	ctx := context.Background()
	env.withSellerContact(t)

	authority := env.customer(t, "Landeshauptstadt München", "DE", "")
	authority.EInvoiceProfile = domain.EInvoiceProfileXRechnungCII
	authority.LeitwegID = "991-12345-67"
	if err := env.contacts.SaveContact(ctx, authority); err != nil {
		t.Fatalf("Kontakt speichern: %v", err)
	}

	inv := env.simpleInvoice(authority.ID, "2026-03-01", 100000)
	if err := env.invoicesWiredWithDocuments(t).Issue(ctx, inv); err != nil {
		t.Fatalf("XRechnung ausstellen: %v", err)
	}
	if inv.ReceiptID == nil {
		t.Fatal("die ausgestellte Rechnung muss auf ihren Beleg zeigen")
	}

	receipt, err := env.receipts.Get(ctx, *inv.ReceiptID)
	if err != nil {
		t.Fatalf("Beleg laden: %v", err)
	}
	if len(receipt.Files) != 3 {
		t.Fatalf("der Beleg trägt %d Dateien, erwartet Original, strukturierten Teil und Darstellung: %+v",
			len(receipt.Files), receipt.Files)
	}
	// Bei der XRechnung ist das XML die Rechnung; das PDF ist die von Buchfink
	// erzeugte Darstellung dazu.
	original, ok := receipt.FileByRole(domain.ReceiptRoleOriginal)
	if !ok || !strings.HasSuffix(original.FileName, ".xml") {
		t.Errorf("das Original muss das XML sein, ist aber %+v", original)
	}
	structured, ok := receipt.FileByRole(domain.ReceiptRoleStructured)
	if !ok || !strings.HasSuffix(structured.FileName, ".xml") {
		t.Errorf("der strukturierte Teil fehlt: %+v", structured)
	}
	rendering, ok := receipt.FileByRole(domain.ReceiptRoleRendering)
	if !ok || !strings.HasSuffix(rendering.FileName, ".pdf") {
		t.Errorf("die Darstellung als PDF fehlt: %+v", rendering)
	}
	if !rendering.Derived {
		t.Error("die Darstellung ist erzeugt und muss als solche gekennzeichnet sein (GoBD Rz. 125)")
	}

	if receipt.ValidatedAt == "" {
		t.Error("am Beleg fehlt der Zeitpunkt der Prüfung")
	}
	if !strings.Contains(receipt.ValidationRuleset, "xrechnung") {
		t.Errorf("das Regelwerk lautet %q, erwartet die deutsche Ausprägung", receipt.ValidationRuleset)
	}
	if receipt.ValidationErrors != 0 {
		t.Errorf("die eigene Rechnung hat %d Befunde: %s", receipt.ValidationErrors, receipt.ValidationFindings)
	}
	// Die Leitweg-ID steht im Datensatz: ohne sie ist die Rechnung im
	// Behördennetz nicht zustellbar (BT-10).
	if !strings.Contains(inv.ZUGFeRDXML, authority.LeitwegID) {
		t.Error("die Leitweg-ID fehlt im XML")
	}

	// Und der Prüflauf ist zufrieden: der Beleg ist gebucht und versiegelt.
	assertRule(t, runChecks(t, env.checks(t), "2026-03-31"), domain.CheckRuleReceiptUnbooked, 0, domain.CheckBlocking)
}

// Die Kleinbetragsrechnung ohne Empfänger geht als PDF hinaus.
//
// EN 16931 verlangt den Namen des Erwerbers (BR-07); § 33 UStDV erlässt ihn und
// nimmt die Kleinbetragsrechnung zugleich von der E-Rechnungspflicht aus. Einen
// Namen zu erfinden, um die Norm zu bedienen, wäre eine maschinenlesbare
// Falschangabe.
func TestSmallAmountInvoiceWithoutRecipientIsPdfOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("die WASM-Kompilierung ist zu langsam für -short")
	}
	env := newTestEnv(t)
	ctx := context.Background()

	inv := smallAmountSale("2026-03-01", 10000)
	if err := env.invoicesWiredWithDocuments(t).Issue(ctx, inv); err != nil {
		t.Fatalf("Kleinbetragsrechnung ausstellen: %v", err)
	}
	if inv.EInvoiceProfile != domain.EInvoiceProfilePDFOnly {
		t.Errorf("Format = %q, erwartet die reine PDF-Rechnung", inv.EInvoiceProfile)
	}
	if inv.ZUGFeRDXML != "" {
		t.Error("zur Kleinbetragsrechnung ohne Empfänger darf kein strukturierter Datensatz entstehen")
	}

	receipt, err := env.receipts.Get(ctx, *inv.ReceiptID)
	if err != nil {
		t.Fatalf("Beleg laden: %v", err)
	}
	if len(receipt.Files) != 1 {
		t.Fatalf("der Beleg trägt %d Dateien, erwartet allein das PDF: %+v", len(receipt.Files), receipt.Files)
	}
	if _, ok := receipt.FileByRole(domain.ReceiptRoleStructured); ok {
		t.Error("ohne Empfänger gibt es keinen strukturierten Teil")
	}

	// Auch nach dem Ende der Übergangsfrist bleibt sie zulässig: die Ausnahme
	// des § 33 UStDV hängt nicht am Datum.
	late := smallAmountSale("2028-03-01", 10000)
	if err := env.invoicesWiredWithDocuments(t).Issue(ctx, late); err != nil {
		t.Errorf("die Kleinbetragsrechnung bleibt von der E-Rechnungspflicht ausgenommen: %v", err)
	}
}

// Der Beleg der Abschlagsrechnung wird mit der Vereinnahmung versiegelt — und
// sperrt bis dahin die Festschreibung nicht.
//
// Beides gehört zusammen: vor der Zahlung gibt es nichts zu buchen (§ 13 Abs. 1
// Nr. 1 Buchst. a Satz 4 UStG), und der Prüflauf darf daraus keinen Verstoß
// machen. Mit der Zahlung entsteht die Buchung, und dann muss der Beleg auf ihr
// versiegelt sein — sonst meldete der Prüflauf einen abgelegten, ungebuchten
// Beleg und sperrte die Festschreibung des Zeitraums.
func TestAdvanceReceiptsAreSealedWithTheSettlement(t *testing.T) {
	if testing.Short() {
		t.Skip("die WASM-Kompilierung ist zu langsam für -short")
	}
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWiredWithDocuments(t)
	group := env.group(t, svc, customer.ID, 1000000)

	advance, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-03-01", Net: 400000,
	})
	if err != nil {
		t.Fatalf("Abschlagsrechnung ausstellen: %v", err)
	}
	if advance.ReceiptID == nil {
		t.Fatal("auch die Abschlagsrechnung wird als Beleg abgelegt")
	}

	// Vor der Vereinnahmung: ein Hinweis, keine Sperre.
	run := runChecks(t, env.checks(t), "2026-03-31")
	pending := findingsFor(run, domain.CheckRuleReceiptUnbooked)
	if len(pending) != 1 {
		t.Fatalf("erwartet einen Hinweis auf die ungebuchte Abschlagsrechnung, erhalten %+v", pending)
	}
	if pending[0].Severity != domain.CheckWarning {
		t.Errorf("Schwere = %q, erwartet einen Hinweis — die Buchung entsteht erst mit der Vereinnahmung",
			pending[0].Severity)
	}

	if _, err := svc.SettleAdvance(ctx, SettleAdvanceRequest{
		AdvanceID: advance.ID, PaymentDate: "2026-03-20", PaymentAccount: domain.AccountBank,
	}); err != nil {
		t.Fatalf("Anzahlung vereinnahmen: %v", err)
	}

	receipt, err := env.receipts.Get(ctx, *advance.ReceiptID)
	if err != nil {
		t.Fatalf("Beleg laden: %v", err)
	}
	if receipt.Status != domain.ReceiptStatusSealed {
		t.Errorf("der Beleg steht auf %q, erwartet versiegelt", receipt.Status)
	}
	if receipt.JournalEntryID == nil {
		t.Fatal("der Beleg muss auf die Vereinnahmungsbuchung zeigen")
	}
	assertRule(t, runChecks(t, env.checks(t), "2026-03-31"), domain.CheckRuleReceiptUnbooked, 0, domain.CheckBlocking)
}

// Auch das Stornodokument wird versiegelt: es trägt die Generalumkehr.
func TestCancellationDocumentIsSealedOnItsReversal(t *testing.T) {
	if testing.Short() {
		t.Skip("die WASM-Kompilierung ist zu langsam für -short")
	}
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWiredWithDocuments(t)

	original := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := svc.Issue(ctx, original); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}
	storno, err := svc.CancelWithDocument(ctx, original.ID, "Leistung nicht erbracht")
	if err != nil {
		t.Fatalf("stornieren: %v", err)
	}
	if storno.ReceiptID == nil || storno.JournalEntryID == nil {
		t.Fatal("das Stornodokument braucht Beleg und Buchung")
	}

	receipt, err := env.receipts.Get(ctx, *storno.ReceiptID)
	if err != nil {
		t.Fatalf("Beleg laden: %v", err)
	}
	if receipt.Status != domain.ReceiptStatusSealed {
		t.Errorf("der Stornobeleg steht auf %q, erwartet versiegelt", receipt.Status)
	}
	if receipt.JournalEntryID == nil || *receipt.JournalEntryID != *storno.JournalEntryID {
		t.Error("der Stornobeleg muss auf die Generalumkehr zeigen")
	}
	// Der Prüflauf meldet weder zum Storno noch zur Ursprungsrechnung einen
	// ungebuchten Beleg — sonst wäre der Zeitraum nicht festschreibbar.
	assertRule(t, runChecks(t, env.checks(t), "2026-12-31"), domain.CheckRuleReceiptUnbooked, 0, domain.CheckBlocking)
}
