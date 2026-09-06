package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/ebilanz"
	"github.com/buchfink/buchfink/internal/repository"
)

// Die Prüfungen der Welle 8: keine Buchung ohne Beleg, der Eigenbeleg als der
// zweite Weg dorthin, die überschreibbare Aufbewahrungsfrist, die
// Fehlerklassen der Klärungsliste und der Voranmeldungszeitraum aus der
// Vorjahressteuer.

// withSelfIssued rüstet den Belegdienst für den Eigenbeleg aus: Setzer,
// Unternehmensangaben und Nummernkreis.
func (e *testEnv) withSelfIssued(t *testing.T) *stubRenderer {
	t.Helper()
	renderer := &stubRenderer{}
	e.receipts.SetRenderer(renderer)
	e.receipts.SetSettingsSource(repository.NewSettingsRepository(e.db))
	e.receipts.SetNumberRepo(e.numberRepo)
	return renderer
}

// manualEntry ist ein Buchungssatz, wie ihn die Maske zusammenstellt: ein
// Aufwand gegen Bank, ohne Steuer, mit dem Steuerfall dazu.
func manualEntry() *domain.JournalEntry {
	return &domain.JournalEntry{
		BookingDate: "2026-03-01", DocumentDate: "2026-03-01",
		Description:  "Parkgebühr Kundentermin",
		TaxTreatment: domain.TaxTreatmentNotTaxable,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "6300", Amount: 500},
			{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 500},
		},
	}
}

// Der Eigenbeleg entsteht mit PDF und Kopfdaten (BEL-01 K1/K2, GOB-05 K1).
func TestSelfIssuedReceiptCarriesItsPDFAndHeader(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	renderer := env.withSelfIssued(t)

	receipt, err := env.receipts.CreateSelfIssued(ctx, SelfIssuedReceiptRequest{
		DocumentDate: "2026-03-01", GrossAmount: 500,
		Reason: "Parkgebühr am Automaten, Quittung nicht ausgedruckt",
	})
	if err != nil {
		t.Fatalf("Eigenbeleg: %v", err)
	}
	if renderer.documents != 1 {
		t.Fatalf("%d gesetzte Dokumente — der Eigenbeleg ist ein PDF", renderer.documents)
	}
	if receipt.Kind != domain.ReceiptKindSelfIssued {
		t.Errorf("Belegart %q — erwartet den Eigenbeleg", receipt.Kind)
	}
	if receipt.IssuerName != "Pfennig Ventures GmbH" {
		t.Errorf("Aussteller %q — der Eigenbeleg stammt vom eigenen Unternehmen", receipt.IssuerName)
	}
	if receipt.DocumentDate != "2026-03-01" || receipt.GrossAmount != 500 {
		t.Errorf("Kopfdaten unvollständig: %s / %s", receipt.DocumentDate, receipt.GrossAmount)
	}
	if err := receipt.ValidateHeader(); err != nil {
		t.Errorf("der Eigenbeleg muss ohne Nacharbeit buchbar sein: %v", err)
	}
	// Die Aufbewahrungsklasse ist die des Buchungsbelegs; ein Eigenbeleg ist
	// einer (§ 147 Abs. 1 Nr. 4 AO).
	if receipt.RetentionClass != domain.RetentionClassVouchers {
		t.Errorf("Aufbewahrungsklasse %q — erwartet die des Buchungsbelegs", receipt.RetentionClass)
	}
	// Der Satz, der den Eigenbeleg zum Eigenbeleg macht, steht im Dokument.
	if !strings.Contains(renderer.lastText, "kein Fremdbeleg") {
		t.Error("dem Dokument fehlt der Hinweis, dass kein Fremdbeleg vorliegt")
	}
	if !strings.Contains(renderer.lastText, "Parkgebühr am Automaten") {
		t.Error("dem Dokument fehlt der Grund")
	}
	// Die Originaldatei ist das erzeugte PDF, nicht abgeleitet: es gibt keine
	// Datei, aus der es entstanden wäre.
	original, ok := receipt.FileByRole(domain.ReceiptRoleOriginal)
	if !ok {
		t.Fatal("dem Eigenbeleg fehlt die Originaldatei")
	}
	if original.Derived {
		t.Error("das PDF des Eigenbelegs ist das Original und nicht abgeleitet")
	}
}

// Der Eigenbeleg ohne Grund entsteht nicht: der Grund ist sein ganzer Inhalt.
func TestSelfIssuedReceiptNeedsItsReason(t *testing.T) {
	env := newTestEnv(t)
	env.withSelfIssued(t)

	_, err := env.receipts.CreateSelfIssued(context.Background(), SelfIssuedReceiptRequest{
		DocumentDate: "2026-03-01", GrossAmount: 500,
	})
	if err == nil {
		t.Fatal("ein Eigenbeleg ohne Grund belegt nichts")
	}
	if !strings.Contains(err.Error(), "Grund") {
		t.Errorf("die Meldung nennt nicht, was fehlt: %v", err)
	}
}

// Die Handbuchung ohne Beleg wird abgewiesen (§ 146 Abs. 1 AO, GoBD Rz. 61).
func TestManualEntryWithoutReceiptIsRefused(t *testing.T) {
	env := newTestEnv(t)

	_, err := env.posting.PostManualEntry(context.Background(), ManualEntryRequest{
		Entry: *manualEntry(),
	})
	if err == nil {
		t.Fatal("eine Buchung ohne Beleg darf nicht entstehen")
	}
	if !strings.Contains(err.Error(), "Beleg") || !strings.Contains(err.Error(), "146") {
		t.Errorf("die Meldung nennt weder den Beleg noch die Vorschrift: %v", err)
	}
	entries, err := env.journalRepo.FindAll(context.Background(), 2026)
	if err != nil {
		t.Fatalf("Journal lesen: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("%d Buchungen im Journal — die abgewiesene darf nichts hinterlassen", len(entries))
	}
}

// Mit einem Eigenbeleg geht dieselbe Buchung durch, und beide entstehen
// zusammen: die Buchung verweist auf den Beleg, der Beleg ist versiegelt.
func TestManualEntryCreatesItsSelfIssuedReceipt(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.withSelfIssued(t)

	entry, err := env.posting.PostManualEntry(ctx, ManualEntryRequest{
		Entry: *manualEntry(),
		SelfIssued: &SelfIssuedReceiptRequest{
			Reason: "Parkgebühr am Automaten, Quittung nicht ausgedruckt",
		},
	})
	if err != nil {
		t.Fatalf("Handbuchung mit Eigenbeleg: %v", err)
	}
	if entry.ReceiptID == nil {
		t.Fatal("die Buchung verweist auf keinen Beleg")
	}
	receipt, err := env.receipts.Get(ctx, *entry.ReceiptID)
	if err != nil {
		t.Fatalf("Beleg lesen: %v", err)
	}
	if receipt.Status != domain.ReceiptStatusSealed {
		t.Errorf("Belegstatus %q — der gebuchte Beleg ist versiegelt", receipt.Status)
	}
	if receipt.GrossAmount != 500 {
		t.Errorf("Betrag des Eigenbelegs %s € — erwartet den Bruttobetrag der Buchung", receipt.GrossAmount)
	}
	if entry.ReceiptHash != receipt.ReceiptHash {
		t.Error("die Buchung trägt einen anderen Beleg-Hash als der Beleg")
	}
	if entry.DocumentNumber != receipt.ReceiptNumber {
		t.Errorf("Belegfeld %q — erwartet die Belegnummer %q",
			entry.DocumentNumber, receipt.ReceiptNumber)
	}
}

// Scheitert die Buchung, bleibt kein Eigenbeleg im Speicher zurück: er wäre
// genau der ungebuchte Beleg, den der Prüflauf meldet.
func TestFailedManualEntryLeavesNoSelfIssuedReceipt(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.withSelfIssued(t)

	broken := manualEntry()
	broken.Lines[1].Account = "9999999"
	_, err := env.posting.PostManualEntry(ctx, ManualEntryRequest{
		Entry:      *broken,
		SelfIssued: &SelfIssuedReceiptRequest{Reason: "Parkgebühr"},
	})
	if err == nil {
		t.Fatal("eine Buchung auf ein unbekanntes Konto darf nicht entstehen")
	}
	receipts, err := env.receipts.List(ctx, "")
	if err != nil {
		t.Fatalf("Belege lesen: %v", err)
	}
	if len(receipts) != 0 {
		t.Errorf("%d Belege abgelegt — die gescheiterte Buchung darf keinen hinterlassen",
			len(receipts))
	}
}

// Die Handbuchung trägt den Steuerfall, und eine Zeile auf einem Steuerkonto
// trägt Schlüssel und Bemessungsgrundlage.
func TestManualEntryRequiresItsTaxDetails(t *testing.T) {
	withoutTreatment := manualEntry()
	withoutTreatment.TaxTreatment = ""
	if err := ValidateManualTaxLines(withoutTreatment); err == nil {
		t.Error("eine Handbuchung ohne Steuerfall läuft an der Voranmeldung vorbei")
	} else if !strings.Contains(err.Error(), "Steuerfall") {
		t.Errorf("die Meldung nennt nicht, was fehlt: %v", err)
	}

	// Ein steuerpflichtiger Inlandsumsatz ohne Steuerzeile ist keiner.
	withoutTaxLine := manualEntry()
	withoutTaxLine.TaxTreatment = domain.TaxTreatmentDomestic
	if err := ValidateManualTaxLines(withoutTaxLine); err == nil {
		t.Error("ein steuerpflichtiger Inlandsumsatz ohne Steuerzeile darf nicht durchgehen")
	}

	// Die Steuerzeile ohne Bemessungsgrundlage meldet die Steuer ohne den
	// Umsatz darunter.
	withoutBase := manualEntry()
	withoutBase.TaxTreatment = domain.TaxTreatmentDomestic
	withoutBase.Lines = []domain.JournalLine{
		{Side: domain.SideDebit, Account: "6300", Amount: 10000},
		{Side: domain.SideDebit, Account: domain.AccountVorsteuer19, Amount: 1900, TaxKey: "VST19"},
		{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 11900},
	}
	err := ValidateManualTaxLines(withoutBase)
	if err == nil {
		t.Fatal("eine Steuerzeile ohne Bemessungsgrundlage darf nicht durchgehen")
	}
	if !strings.Contains(err.Error(), "Bemessungsgrundlage") {
		t.Errorf("die Meldung nennt nicht, was fehlt: %v", err)
	}

	// Vollständig geht sie durch.
	complete := manualEntry()
	complete.TaxTreatment = domain.TaxTreatmentDomestic
	complete.Lines = []domain.JournalLine{
		{Side: domain.SideDebit, Account: "6300", Amount: 10000},
		{Side: domain.SideDebit, Account: domain.AccountVorsteuer19, Amount: 1900,
			TaxKey: "VST19", TaxBase: 10000},
		{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 11900},
	}
	if err := ValidateManualTaxLines(complete); err != nil {
		t.Errorf("die vollständige Buchung muss durchgehen: %v", err)
	}
}

// Die Aufbewahrungsfrist lässt sich nur verlängern, und die Verlängerung steht
// mit Vorher/Nachher im Änderungsprotokoll (ARC-01 K2).
func TestRetentionOverrideOnlyExtendsAndIsLogged(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.withSelfIssued(t)

	receipt, err := env.receipts.CreateSelfIssued(ctx, SelfIssuedReceiptRequest{
		DocumentDate: "2026-03-01", GrossAmount: 500, Reason: "Parkgebühr",
	})
	if err != nil {
		t.Fatalf("Eigenbeleg: %v", err)
	}
	if receipt.RetentionUntil != "2034-12-31" {
		t.Fatalf("Frist bis %s — acht Jahre ab dem Schluss von 2026", receipt.RetentionUntil)
	}

	// Nach unten: abgewiesen. Die gesetzliche Frist ist die Untergrenze.
	if _, err := env.receipts.OverrideRetention(
		ctx, receipt.ID, domain.RetentionClassLetters, "kürzer reicht"); err == nil {
		t.Error("eine kürzere Frist wäre ein Verstoß gegen § 257 HGB und § 147 AO")
	}

	// Ohne Grund: abgewiesen.
	if _, err := env.receipts.OverrideRetention(
		ctx, receipt.ID, domain.RetentionClassBooks, "  "); err == nil {
		t.Error("zur Verlängerung gehört ihr Grund")
	}

	updated, err := env.receipts.OverrideRetention(
		ctx, receipt.ID, domain.RetentionClassBooks,
		"Beleg gehört zum Rechtsstreit 4 O 112/26")
	if err != nil {
		t.Fatalf("Verlängerung: %v", err)
	}
	if updated.RetentionUntil != "2036-12-31" {
		t.Errorf("Frist bis %s — erwartet zehn Jahre ab dem Schluss von 2026",
			updated.RetentionUntil)
	}
	if updated.RetentionOverrideReason == "" || updated.RetentionOverrideAt == "" {
		t.Error("die Verlängerung steht ohne Grund und ohne Zeitpunkt am Beleg")
	}

	entries, err := repository.NewAuditRepository(env.db).FindAll(ctx, 0)
	if err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	found := false
	for _, e := range entries {
		if strings.Contains(e.Details, "Aufbewahrungsfrist") &&
			strings.Contains(e.Details, "2034-12-31") &&
			strings.Contains(e.Details, "2036-12-31") {
			found = true
		}
	}
	if !found {
		t.Error("das Protokoll nennt nicht beide Fristen — vorher und nachher")
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
	// Der Beleg trägt kein Belegdatum: eine Pflichtangabe des § 14 Abs. 4 UStG.
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

// Der Voranmeldungszeitraum folgt der Steuer des Vorjahres (§ 18 Abs. 2 UStG).
func TestVatPeriodProposalFollowsThePriorYearTax(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.vatReturns(t)
	repo := repository.NewVatReturnRepository(env.db)

	// Vier Quartale des Vorjahres mit zusammen 12.000 € Steuer.
	for _, key := range []string{"2025-Q1", "2025-Q2", "2025-Q3", "2025-Q4"} {
		rec := &domain.VatReturn{
			FiscalYear: 2025, PeriodKey: key, PeriodType: domain.VatPeriodQuarter,
			Status: domain.VatReturnSubmitted, Payable: 300_000,
		}
		if err := repo.Create(ctx, rec); err != nil {
			t.Fatalf("Voranmeldung %s: %v", key, err)
		}
	}

	proposal, err := svc.SuggestPeriodType(ctx, 2026)
	if err != nil {
		t.Fatalf("Vorschlag: %v", err)
	}
	if proposal.PriorYearTax != 1_200_000 {
		t.Errorf("Vorjahressteuer %s € — erwartet 12.000,00 €", proposal.PriorYearTax)
	}
	if proposal.Proposed != domain.VatPeriodMonth {
		t.Errorf("Vorschlag %q — über 9.000 € ist monatlich anzumelden", proposal.Proposed)
	}
	if !proposal.Changes {
		t.Error("der Mandant steht auf Quartal; der Vorschlag weicht ab und muss das sagen")
	}
	if !proposal.Complete {
		t.Errorf("alle vier Quartale liegen vor, es fehlen angeblich %d", proposal.MissingPeriods)
	}
	if !strings.Contains(proposal.Note, "§ 18 Abs. 2") &&
		!strings.Contains(proposal.Reference, "§ 18 Abs. 2") {
		t.Error("der Vorschlag nennt seine Vorschrift nicht")
	}
}

// Unter der Grenze bleibt es beim Vierteljahr — die Befreiung spricht das
// Finanzamt aus und nicht das Programm.
func TestVatPeriodProposalStaysQuarterlyBelowTheThreshold(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.vatReturns(t)
	repo := repository.NewVatReturnRepository(env.db)

	for _, key := range []string{"2025-Q1", "2025-Q2", "2025-Q3", "2025-Q4"} {
		rec := &domain.VatReturn{
			FiscalYear: 2025, PeriodKey: key, PeriodType: domain.VatPeriodQuarter,
			Status: domain.VatReturnSubmitted, Payable: 25_000,
		}
		if err := repo.Create(ctx, rec); err != nil {
			t.Fatalf("Voranmeldung %s: %v", key, err)
		}
	}

	proposal, err := svc.SuggestPeriodType(ctx, 2026)
	if err != nil {
		t.Fatalf("Vorschlag: %v", err)
	}
	if proposal.Proposed != domain.VatPeriodQuarter {
		t.Errorf("Vorschlag %q — unter 2.000 € bleibt es beim Vierteljahr, bis das "+
			"Finanzamt befreit", proposal.Proposed)
	}
	if !strings.Contains(proposal.Note, "Finanzamt") {
		t.Error("der Hinweis muss sagen, dass die Befreiung das Finanzamt ausspricht")
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

// Der Prüflauf meldet einen Monat, der zwei Monate nach seinem Ende nicht
// festgeschrieben ist (UNV-02 K2).
func TestPeriodNotCommittedIsReportedAfterTwoMonths(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Ende März: der Januar ist überfällig, aber die Zweimonatsgrenze ist noch
	// nicht überschritten.
	run, err := env.checksOn(t, "2026-03-31").Preview(ctx, CheckRequest{CutoffDate: "2026-03-31"})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
	if len(findingsFor(run, domain.CheckRulePeriodNotCommitted)) != 0 {
		t.Error("am 31. März sind seit dem Ende des Januars erst zwei Monate vergangen")
	}

	run, err = env.checksOn(t, "2026-04-01").Preview(ctx, CheckRequest{CutoffDate: "2026-04-01"})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
	found := findingsFor(run, domain.CheckRulePeriodNotCommitted)
	if len(found) == 0 {
		t.Fatal("am 1. April ist der Januar zwei Monate nach seinem Ende nicht festgeschrieben")
	}
	if found[0].ObjectID != "2026-01" {
		t.Errorf("gemeldeter Zeitraum %q — erwartet den Januar", found[0].ObjectID)
	}
	if found[0].Severity != domain.CheckWarning {
		t.Errorf("Gewicht %q — erwartet einen Hinweis", found[0].Severity)
	}

	// Nach der Festschreibung ist die Meldung weg.
	env.commitUntil(t, "2026-01-31")
	run, err = env.checksOn(t, "2026-04-01").Preview(ctx, CheckRequest{CutoffDate: "2026-04-01"})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
	for _, f := range findingsFor(run, domain.CheckRulePeriodNotCommitted) {
		if f.ObjectID == "2026-01" {
			t.Error("der festgeschriebene Januar darf nicht mehr gemeldet werden")
		}
	}
}

// Die Frist zur Festschreibung folgt der Dauerfristverlängerung: wer einen
// Monat später anmelden darf, schreibt einen Monat später fest.
func TestCommitDeadlineFollowsThePermanentExtension(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	before, err := env.deadlines(t).Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	january, ok := deadlineByKey(before, "festschreibung.2026-01")
	if !ok {
		t.Fatal("die Festschreibung des Januars fehlt")
	}
	if january.DueDate != "2026-02-28" {
		t.Fatalf("ohne Dauerfrist fällig am %s, erwartet 2026-02-28", january.DueDate)
	}

	env.setSettings(t, func(c *domain.CompanySettings) { c.PermanentExtension = true })
	after, err := env.deadlines(t).Deadlines(ctx, 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	january, _ = deadlineByKey(after, "festschreibung.2026-01")
	if january.DueDate != "2026-03-31" {
		t.Errorf("mit Dauerfristverlängerung fällig am %s, erwartet 2026-03-31", january.DueDate)
	}
}

// Das eigene Konto trägt seine Gliederungsposition — und damit erscheint es in
// Bilanz und GuV und in der E-Bilanz (BEL-06 K2).
func TestCustomAccountCarriesItsPositionIntoTheStatement(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := NewAccountService(
		repository.NewAccountRepository(env.db), repository.NewAuditRepository(env.db))

	const position = "guv.guv_8.sonstige_betriebliche_aufwendungen"
	account, err := svc.CreateCustom(ctx, CustomAccountRequest{
		Number: "6011", Name: "Fremdleistungen Fotografie", HGBPosition: position,
	})
	if err != nil {
		t.Fatalf("eigenes Konto: %v", err)
	}
	if !account.IsCustom || !account.IsActive {
		t.Error("das angelegte Konto ist ein eigenes und ist bebuchbar")
	}
	if account.PositionID != position || account.HGBCode != "GuV.8" {
		t.Errorf("Position %q / %q — erwartet die gewählte", account.PositionID, account.HGBCode)
	}
	if account.Type != domain.AccountTypeExpense {
		t.Errorf("Kontoart %q — die Position sagt Aufwand", account.Type)
	}

	// Eine belegte Nummer wird nicht überschrieben: das zerstörte die
	// Zuordnung des Kontenrahmens.
	if _, err := svc.CreateCustom(ctx, CustomAccountRequest{
		Number: "6300", Name: "Doppelt", HGBPosition: position,
	}); err == nil {
		t.Error("eine belegte Kontonummer darf nicht ein zweites Mal vergeben werden")
	}
	// Ohne Position gibt es kein Konto: es fehlte im Abschluss.
	if _, err := svc.CreateCustom(ctx, CustomAccountRequest{
		Number: "6012", Name: "Ohne Position",
	}); err == nil {
		t.Error("ein Konto ohne Gliederungsposition erschiene in keinem Abschluss")
	}
	// Die Klasse 9 bleibt den Vortrags- und statistischen Konten vorbehalten.
	if _, err := svc.CreateCustom(ctx, CustomAccountRequest{
		Number: "9999", Name: "Statistisch", HGBPosition: position,
	}); err == nil {
		t.Error("die Klasse 9 ist den Vortrags- und statistischen Konten vorbehalten")
	}
	// Die Klasse 0 ist die des Anlagevermögens: mit einer Position des
	// Anlagevermögens ist ein eigenes Anlagenkonto anlegbar, mit einer
	// Aufwandsposition nicht.
	const assetPosition = "bilanz.aktiva_a_ii.technische_anlagen_und_maschinen"
	asset, err := svc.CreateCustom(ctx, CustomAccountRequest{
		Number: "0499", Name: "Prüfstand", HGBPosition: assetPosition,
	})
	if err != nil {
		t.Fatalf("eigenes Anlagenkonto: %v", err)
	}
	if asset.PositionID != assetPosition {
		t.Errorf("Position %q — erwartet die des Anlagevermögens", asset.PositionID)
	}
	if _, err := svc.CreateCustom(ctx, CustomAccountRequest{
		Number: "0498", Name: "Aufwand in der Anlagenklasse", HGBPosition: position,
	}); err == nil {
		t.Error("eine Aufwandsposition trägt kein Konto der Klasse 0")
	}

	// Gebucht wird darauf wie auf jedes andere Konto, und die Bilanz trägt es.
	account.DebitSum = 250_000
	bank, err := repository.NewAccountRepository(env.db).FindByNumber(ctx, domain.AccountBank)
	if err != nil {
		t.Fatalf("Bankkonto: %v", err)
	}
	bank.CreditSum = 250_000
	stmt, err := accounting.BuildStatement(
		[]domain.Account{*account, *bank}, nil, domain.DepthFull)
	if err != nil {
		t.Fatalf("Bilanz und GuV: %v", err)
	}
	found := false
	for _, line := range stmt.Income {
		for _, a := range line.Accounts {
			if a.Number == "6011" {
				found = true
			}
		}
	}
	if !found {
		t.Error("das eigene Konto steht in keiner Position der GuV")
	}

	// Und die E-Bilanz bildet es ab: der Kontennachweis nennt es.
	xbrl, report, err := ebilanz.GenerateEBilanzXBRL(ebilanz.InstanceInput{
		Settings:   companySettingsFor(t, env),
		Statement:  stmt,
		Accounts:   []domain.Account{*account, *bank},
		FiscalYear: 2026, StartDate: "2026-01-01", EndDate: "2026-12-31",
		PriorStartDate: "2025-01-01", PriorEndDate: "2025-12-31",
	})
	if err != nil {
		t.Fatalf("E-Bilanz: %v", err)
	}
	if !strings.Contains(xbrl, "6011") {
		t.Error("die E-Bilanz kennt das eigene Konto nicht")
	}
	for _, row := range report.Blocking {
		if row.Account == "6011" {
			t.Errorf("das eigene Konto gilt als nicht zuordenbar: %+v", row)
		}
	}

	// Gesperrt wird mit Grund, und ein Konto des SKR04 überhaupt nicht.
	if _, err := svc.SetBlocked(ctx, "6011", true, ""); err == nil {
		t.Error("zum Sperren gehört eine Begründung")
	}
	blocked, err := svc.SetBlocked(ctx, "6011", true, "Leistung wird nicht mehr eingekauft")
	if err != nil {
		t.Fatalf("sperren: %v", err)
	}
	if blocked.IsActive {
		t.Error("das gesperrte Konto ist nicht mehr bebuchbar")
	}
	if _, err := svc.SetBlocked(ctx, "6300", true, "Grund"); err == nil {
		t.Error("ein Konto des SKR04 wird nicht gesperrt")
	}
}

// companySettingsFor liest die Unternehmensdaten der Testumgebung.
func companySettingsFor(t *testing.T, env *testEnv) *domain.CompanySettings {
	t.Helper()
	cfg, err := repository.NewSettingsRepository(env.db).
		GetCompanySettings(context.Background())
	if err != nil {
		t.Fatalf("Unternehmensdaten: %v", err)
	}
	return cfg
}

// Der Befund über den nicht festgeschriebenen Monat wird zur Aufgabe „Monat
// festschreiben" (UNV-02 K2).
func TestNotCommittedPeriodBecomesATask(t *testing.T) {
	env := newTestEnv(t)
	svc := NewTaskService(repository.NewSettingsRepository(env.db), 2026)
	svc.SetCheckSource(stubCheckSource{run: &domain.CheckRun{Findings: []domain.CheckFinding{{
		Rule: domain.CheckRulePeriodNotCommitted, Severity: domain.CheckWarning,
		ObjectID: "2026-01", ObjectName: "Januar 2026",
		Message: "Januar 2026 ist seit dem 2026-03-31 nicht festgeschrieben",
	}}}})

	list, err := svc.Tasks(context.Background(), TaskOptions{Today: "2026-04-01"})
	if err != nil {
		t.Fatalf("Aufgabenliste: %v", err)
	}
	found := false
	for _, group := range [][]domain.Task{list.Overdue, list.Open, list.Upcoming} {
		for _, task := range group {
			if task.Title == "Monat festschreiben" {
				found = true
				if task.Reference == "" {
					t.Error("der Aufgabe fehlt ihre Fundstelle")
				}
			}
		}
	}
	if !found {
		t.Errorf("die Aufgabe „Monat festschreiben\" fehlt: %+v", list)
	}
}
