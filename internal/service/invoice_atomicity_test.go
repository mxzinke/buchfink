package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Was geschieht, wenn das Dokument scheitert, nachdem Nummer und Buchung
// stehen.
//
// Der Fall ist vorgesehen (Status „Dokument fehlt", Knopf „Dokument erneut
// erzeugen"), und deshalb muss alles, was zur Rechnung gehört, vor ihm liegen.
// Was dahinter liegt, fehlt nach dem Fehlschlag dauerhaft — und ein
// Wiederholungsversuch stellte es nicht her, sondern buchte ein zweites Mal.

// Die Abschlagsrechnung behält ihren offenen Posten, auch wenn das Dokument
// scheitert.
//
// Entstünde er erst hinter der Dokumenterzeugung, gäbe es die Rechnung mit
// Nummer, Status „Dokument fehlt" und Art „Abschlag" — aber ohne offenen
// Posten: sie fehlte in der OP-Liste, ließe sich nicht vereinnahmen, zählte
// nicht gegen den Gesamtbetrag des Verbunds und nicht in die Verrechnung der
// Schlussrechnung. Kein Nachholen des Dokuments heilte das.
func TestFailedDocumentKeepsTheAdvanceOpenItem(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	wired := env.invoicesWired(t)
	group := env.group(t, wired, customer.ID, 1000000)

	_, err := env.invoicesWithBrokenRenderer(t).IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-03-01", Net: 400000,
	})
	if err == nil {
		t.Fatal("das Scheitern der Dokumenterzeugung muss gemeldet werden")
	}
	if !strings.Contains(err.Error(), "nachholen") {
		t.Errorf("die Meldung sagt nicht, dass sich das Dokument nachholen lässt: %v", err)
	}

	stored, err := env.invoiceRepoOf(t).FindByNumber(ctx, "RE-2026-0001")
	if err != nil {
		t.Fatalf("die Abschlagsrechnung muss mit ihrer Nummer gespeichert sein: %v", err)
	}
	if stored.Status != domain.InvoiceStatusPendingDocument {
		t.Errorf("Status = %q, erwartet %q", stored.Status, domain.InvoiceStatusPendingDocument)
	}

	open, err := wired.OpenAdvanceItems(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 1 || open[0].DocumentNumber != stored.InvoiceNumber {
		t.Fatalf("die Abschlagsrechnung fehlt in der OP-Liste: %+v", open)
	}

	// Und sie zählt im Verbund: gegen den Gesamtbetrag …
	groups, err := wired.GetInvoiceGroups(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || groups[0].ComputeProgress().BilledNet != 400000 {
		t.Fatalf("der Verbund kennt den Abschlag nicht: %+v", groups)
	}
	// … und als vereinnahmbarer Posten.
	advance, err := wired.SettleAdvance(ctx, SettleAdvanceRequest{
		AdvanceID: stored.ID, PaymentDate: "2026-04-15", PaymentAccount: domain.AccountBank,
	})
	if err != nil {
		t.Fatalf("Anzahlung vereinnahmen: %v", err)
	}
	if !advance.Settled() {
		t.Error("die vereinnahmte Anzahlung muss ihren Zahlungstag tragen")
	}
}

// Scheitert das Dokument der Schlussrechnung, ist der Verbund trotzdem
// geschlossen.
//
// Die Buchung steht dann: Gesamterlös, volle Umsatzsteuer und die Auflösung von
// 3272/3806. Bliebe der Verbund offen, käme der nächste Versuch
// „Schlussrechnung ausstellen" durch und buchte all das ein zweites Mal —
// doppelter Steuerausweis (§ 14c Abs. 1 UStG), und 3272 rutschte ins Soll. Der
// Weg aus dem Fehlschlag ist das Nachholen des Dokuments und nicht die zweite
// Schlussrechnung.
func TestFailedDocumentStillClosesTheInvoiceGroup(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	wired := env.invoicesWired(t)
	group := env.group(t, wired, customer.ID, 1000000)

	advanceInvoice, err := wired.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-03-01", Net: 400000,
	})
	if err != nil {
		t.Fatalf("Abschlagsrechnung ausstellen: %v", err)
	}
	if _, err := wired.SettleAdvance(ctx, SettleAdvanceRequest{
		AdvanceID: advanceInvoice.ID, PaymentDate: "2026-04-15", PaymentAccount: domain.AccountBank,
	}); err != nil {
		t.Fatalf("Anzahlung vereinnahmen: %v", err)
	}

	request := FinalInvoiceRequest{
		GroupID: group.ID, Date: "2026-11-30",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-11-30",
		Items: []domain.InvoiceItem{{
			Description: "Umbau Ladenlokal", QuantityMilli: 1000, UnitPrice: 1000000,
			TaxRate: domain.TaxRateStandard,
		}},
	}
	broken := env.invoicesWithBrokenRenderer(t)
	if _, err := broken.IssueFinalInvoice(ctx, request); err == nil {
		t.Fatal("das Scheitern der Dokumenterzeugung muss gemeldet werden")
	} else if !strings.Contains(err.Error(), "nachholen") {
		t.Errorf("die Meldung sagt nicht, dass sich das Dokument nachholen lässt: %v", err)
	}

	groups, err := wired.GetInvoiceGroups(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || !groups[0].Closed {
		t.Fatalf("der Verbund muss trotz des fehlenden Dokuments abgeschlossen sein: %+v", groups)
	}
	if len(groups[0].Advances) != 1 || !groups[0].Advances[0].SettledInFinal {
		t.Errorf("die Anzahlung muss als in der Schlussrechnung verrechnet vermerkt sein: %+v", groups[0].Advances)
	}

	// Der zweite Versuch wird abgewiesen — er wäre die zweite Schlussrechnung.
	if _, err := broken.IssueFinalInvoice(ctx, request); err == nil {
		t.Fatal("ein Verbund hat höchstens eine Schlussrechnung")
	}
	entries, err := env.journalRepo.FindAll(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	// Vereinnahmung und Schlussrechnung, nicht mehr.
	if len(entries) != 2 {
		t.Fatalf("es stehen %d Buchungen im Journal, erwartet 2", len(entries))
	}
	if got := env.accountBalance(t, accounting.AccountErhalteneAnzahlungen19); got != 0 {
		t.Errorf("Konto 3272 trägt %s, erwartet null", got)
	}
	if got := env.accountBalance(t, domain.AccountUmsatzsteuer19); got != -190000 {
		t.Errorf("Umsatzsteuer im Haben = %s, erwartet 1900,00 auf 10.000 netto", -got)
	}

	if testing.Short() {
		t.Skip("das Nachholen braucht den PDF-Renderer; die WASM-Kompilierung ist zu langsam für -short")
	}
	final, err := env.invoiceRepoOf(t).FindByNumber(ctx, "RE-2026-0002")
	if err != nil {
		t.Fatalf("die Schlussrechnung muss gespeichert sein: %v", err)
	}
	repaired, err := env.invoicesWiredWithDocuments(t).RegenerateDocument(ctx, final.ID)
	if err != nil {
		t.Fatalf("Dokument nachholen: %v", err)
	}
	if repaired.Status != domain.InvoiceStatusIssued || repaired.ReceiptID == nil {
		t.Fatalf("nach dem Nachholen: Status %q, Beleg %v", repaired.Status, repaired.ReceiptID)
	}
}

// Der Storno einer Schlussrechnung öffnet den Verbund wieder.
//
// Die Generalumkehr nimmt die Auflösung der Anzahlungen mit zurück: 3272 und
// 3806 stehen danach wieder im Haben. Gälten die Abschläge weiter als „in der
// Schlussrechnung verrechnet", wären sie nie wieder absetzbar, und eine neue
// Schlussrechnung scheiterte am geschlossenen Verbund — der Vorgang ließe sich
// nicht mehr zu Ende bringen.
func TestCancellingTheFinalInvoiceReopensTheGroup(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)
	group := env.group(t, svc, customer.ID, 1000000)

	advanceInvoice, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-03-01", Net: 400000,
	})
	if err != nil {
		t.Fatalf("Abschlagsrechnung ausstellen: %v", err)
	}
	if _, err := svc.SettleAdvance(ctx, SettleAdvanceRequest{
		AdvanceID: advanceInvoice.ID, PaymentDate: "2026-04-15", PaymentAccount: domain.AccountBank,
	}); err != nil {
		t.Fatalf("Anzahlung vereinnahmen: %v", err)
	}
	final, err := svc.IssueFinalInvoice(ctx, FinalInvoiceRequest{
		GroupID: group.ID, Date: "2026-11-30",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-11-30",
		Items: []domain.InvoiceItem{{
			Description: "Umbau Ladenlokal", QuantityMilli: 1000, UnitPrice: 1000000,
			TaxRate: domain.TaxRateStandard,
		}},
	})
	if err != nil {
		t.Fatalf("Schlussrechnung ausstellen: %v", err)
	}

	if _, err := svc.CancelWithDocument(ctx, final.ID, "Leistungsumfang falsch abgerechnet"); err != nil {
		t.Fatalf("Schlussrechnung stornieren: %v", err)
	}

	groups, err := svc.GetInvoiceGroups(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 {
		t.Fatalf("erwartet einen Verbund, erhalten %d", len(groups))
	}
	if groups[0].Closed || groups[0].FinalInvoiceID != nil {
		t.Errorf("der Verbund muss nach dem Storno wieder offen sein: %+v", groups[0])
	}
	if got := groups[0].DeductibleAdvances(); len(got) != 1 {
		t.Fatalf("die Anzahlung muss wieder absetzbar sein, erhalten %+v", got)
	}
	if groups[0].Advances[0].SettledInFinal {
		t.Error("die Anzahlung gilt weiter als verrechnet und wäre nie wieder absetzbar")
	}
	// Und die Buchhaltung stimmt dazu: die Anzahlung steht wieder auf 3272.
	if got := env.accountBalance(t, accounting.AccountErhalteneAnzahlungen19); got != -400000 {
		t.Errorf("Konto 3272 trägt %s im Haben, erwartet 4000,00", -got)
	}

	// Die Ersatz-Schlussrechnung geht wieder durch — und setzt die Anzahlung
	// erneut ab.
	replacement, err := svc.IssueFinalInvoice(ctx, FinalInvoiceRequest{
		GroupID: group.ID, Date: "2026-12-05",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-11-30",
		Items: []domain.InvoiceItem{{
			Description: "Umbau Ladenlokal", QuantityMilli: 1000, UnitPrice: 900000,
			TaxRate: domain.TaxRateStandard,
		}},
	})
	if err != nil {
		t.Fatalf("Ersatz-Schlussrechnung ausstellen: %v", err)
	}
	if replacement.PrepaidAmount != 476000 {
		t.Errorf("abgesetzte Anzahlungen = %s, erwartet 4760,00", replacement.PrepaidAmount)
	}
	if got := env.accountBalance(t, accounting.AccountErhalteneAnzahlungen19); got != 0 {
		t.Errorf("Konto 3272 trägt %s, erwartet null", got)
	}
}

// Die Berichtigung einer Schlussrechnung setzt die Anzahlungen wieder ab.
//
// Liefe die Ersatzrechnung über den gewöhnlichen Weg, wiese sie den vollen
// Erlös und die volle Steuer aus, ohne BT-113 und ohne Auflösung von
// 3272/3806: doppelter Steuerausweis (§ 14c Abs. 1 UStG) und eine Forderung
// über einen längst gezahlten Betrag.
func TestCorrectingTheFinalInvoiceDeductsTheAdvancesAgain(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)
	group := env.group(t, svc, customer.ID, 1000000)

	advanceInvoice, err := svc.IssueAdvanceInvoice(ctx, AdvanceInvoiceRequest{
		GroupID: group.ID, Date: "2026-03-01", Net: 400000,
	})
	if err != nil {
		t.Fatalf("Abschlagsrechnung ausstellen: %v", err)
	}
	if _, err := svc.SettleAdvance(ctx, SettleAdvanceRequest{
		AdvanceID: advanceInvoice.ID, PaymentDate: "2026-04-15", PaymentAccount: domain.AccountBank,
	}); err != nil {
		t.Fatalf("Anzahlung vereinnahmen: %v", err)
	}
	final, err := svc.IssueFinalInvoice(ctx, FinalInvoiceRequest{
		GroupID: group.ID, Date: "2026-11-30",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-11-30",
		Items: []domain.InvoiceItem{{
			Description: "Umbau Ladenlokal", QuantityMilli: 1000, UnitPrice: 1000000,
			TaxRate: domain.TaxRateStandard,
		}},
	})
	if err != nil {
		t.Fatalf("Schlussrechnung ausstellen: %v", err)
	}

	corrected, err := svc.CorrectInvoice(ctx, final.ID, "Position doppelt abgerechnet", &domain.Invoice{
		ContactID: customer.ID, Date: "2026-12-05",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-11-30",
		TaxTreatment: domain.TaxTreatmentDomestic,
		Items: []domain.InvoiceItem{{
			Description: "Umbau Ladenlokal", QuantityMilli: 1000, UnitPrice: 900000,
			TaxRate: domain.TaxRateStandard,
		}},
	})
	if err != nil {
		t.Fatalf("Schlussrechnung berichtigen: %v", err)
	}

	// Der Bezug auf die berichtigte Rechnung (BG-3) …
	if corrected.CorrectsInvoiceNumber != final.InvoiceNumber {
		t.Errorf("die berichtigte Rechnung verweist auf %q, erwartet %q",
			corrected.CorrectsInvoiceNumber, final.InvoiceNumber)
	}
	// … und die abgesetzten Anzahlungen (BT-113, BG-3).
	if corrected.PrepaidAmount != 476000 {
		t.Errorf("abgesetzte Anzahlungen = %s, erwartet 4760,00", corrected.PrepaidAmount)
	}
	if len(corrected.PrecedingRefs) != 1 || corrected.PrecedingRefs[0].Number != advanceInvoice.InvoiceNumber {
		t.Errorf("die abgesetzte Abschlagsrechnung fehlt im Bezug: %+v", corrected.PrecedingRefs)
	}
	if corrected.OpenAmount() != 595000 {
		t.Errorf("Zahlbetrag = %s, erwartet 5950,00", corrected.OpenAmount())
	}

	// Die Probe auf das Ganze: die Anzahlung ist genau einmal aufgelöst, die
	// Umsatzsteuer steht auf 9.000 € netto, und die Forderung entspricht der
	// berichtigten Schlussrechnung abzüglich der Anzahlung.
	if got := env.accountBalance(t, accounting.AccountErhalteneAnzahlungen19); got != 0 {
		t.Errorf("Konto 3272 trägt %s, erwartet null", got)
	}
	if got := env.accountBalance(t, domain.AccountUmsatzsteuer19); got != -171000 {
		t.Errorf("Umsatzsteuer im Haben = %s, erwartet 1710,00 auf 9.000 netto", -got)
	}
	if got := env.accountBalance(t, customer.LedgerAccount); got != 595000 {
		t.Errorf("Forderung = %s, erwartet 5950,00", got)
	}

	// Der Verbund ist wieder geschlossen — mit der berichtigten Rechnung.
	groups, err := svc.GetInvoiceGroups(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || !groups[0].Closed {
		t.Fatalf("der Verbund muss nach der Berichtigung abgeschlossen sein: %+v", groups)
	}
	if groups[0].FinalInvoiceID == nil || *groups[0].FinalInvoiceID != corrected.ID {
		t.Errorf("der Verbund zeigt auf %v, erwartet die berichtigte Rechnung %d",
			groups[0].FinalInvoiceID, corrected.ID)
	}
}

// Ein Regelverstoß verhindert die Ausstellung — vor der Nummernvergabe.
//
// Entscheidung 4 verlangt es so: „ein Verstoß verhindert die Ausstellung".
// Liefe die Prüfung erst beim Erzeugen des Dokuments, hinterließe der Verstoß
// eine gebuchte Rechnung mit verbrauchter Nummer im Zustand „Dokument fehlt" —
// und wäre er nicht durch Stammdaten heilbar, bliebe nur das Storno mit einer
// zweiten Nummer.
func TestRulesetViolationPreventsIssuing(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Leitweg-ID vorhanden, Ansprechpartner und Bankverbindung des Verkäufers
	// nicht: BR-DE-2 ff. und BG-16 schlagen zu.
	authority := env.customer(t, "Landeshauptstadt München", "DE", "")
	authority.EInvoiceProfile = domain.EInvoiceProfileXRechnungCII
	authority.LeitwegID = "991-12345-67"
	if err := env.contacts.SaveContact(ctx, authority); err != nil {
		t.Fatalf("Kontakt speichern: %v", err)
	}

	svc := env.invoicesWired(t)
	inv := env.simpleInvoice(authority.ID, "2026-03-01", 100000)
	err := svc.Issue(ctx, inv)
	if err == nil {
		t.Fatal("die XRechnung ohne Ansprechpartner darf nicht ausgestellt werden")
	}
	if !strings.Contains(err.Error(), "XRechnung") {
		t.Errorf("die Meldung nennt das Regelwerk nicht: %v", err)
	}

	// Keine Nummer verbraucht, keine Rechnung, keine Buchung.
	next, err := env.numberRepo.Peek(ctx, domain.NumberRangeInvoice, 2026)
	if err != nil {
		t.Fatalf("Nummernkreis lesen: %v", err)
	}
	if next != 1 {
		t.Errorf("der Zähler steht auf %d, erwartet 1", next)
	}
	invoices, err := env.invoiceRepoOf(t).FindAll(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(invoices) != 0 {
		t.Errorf("es steht %d Rechnung(en) in der Datenbank, erwartet keine", len(invoices))
	}
	entries, err := env.journalRepo.FindAll(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("es stehen %d Buchungen im Journal, erwartet keine", len(entries))
	}

	// Mit den fehlenden Stammdaten geht dieselbe Rechnung hinaus.
	env.withSellerContact(t)
	if err := env.invoicesWired(t).Issue(ctx, env.simpleInvoice(authority.ID, "2026-03-01", 100000)); err != nil {
		t.Fatalf("nach dem Ergänzen der Stammdaten muss die XRechnung ausstellbar sein: %v", err)
	}
}

// Ein untaugliches Nummernformat wird abgewiesen und nicht stillschweigend
// ersetzt.
//
// `RE-{JAHR}` ohne Zähler wäre ein Nummernkreis, in dem jede Rechnung dieselbe
// Nummer trüge (§ 14 Abs. 4 Nr. 4 UStG). Es durch die Voreinstellung zu
// ersetzen ließe den Anwender glauben, sein Format sei gespeichert.
func TestInvalidInvoiceNumberFormatIsRejected(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	settings := repository.NewSettingsRepository(env.db)

	cfg, err := settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Einstellungen lesen: %v", err)
	}
	cfg.InvoiceNumberFormat = "RE-{JAHR}"
	if err := settings.UpdateCompanySettings(ctx, cfg); err == nil {
		t.Fatal("ein Format ohne Zähler muss abgewiesen werden")
	} else if !strings.Contains(err.Error(), "{NR}") {
		t.Errorf("die Meldung nennt den fehlenden Platzhalter nicht: %v", err)
	}

	stored, err := settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stored.InvoiceNumberFormat == "RE-{JAHR}" {
		t.Error("das abgewiesene Format wurde trotzdem gespeichert")
	}

	// Ein leeres Feld heißt „nicht festgelegt" und bekommt die Voreinstellung.
	cfg.InvoiceNumberFormat = ""
	if err := settings.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatalf("ohne Angabe muss die Voreinstellung greifen: %v", err)
	}
	stored, err = settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stored.InvoiceNumberFormat != domain.DefaultInvoiceNumberFormat {
		t.Errorf("Nummernformat = %q, erwartet %q", stored.InvoiceNumberFormat, domain.DefaultInvoiceNumberFormat)
	}
}
