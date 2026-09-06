package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/einvoice"
	"github.com/buchfink/buchfink/internal/repository"
)

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

// Eine verlängerte Aufbewahrungsfrist bleibt verlängert, auch wenn danach die
// Kopfdaten erfasst werden (ARC-01 K2, Entscheidung 6).
func TestSaveHeaderKeepsAnExtendedRetention(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	receipt := env.fileIncoming(t, "rechnung.pdf")
	if _, err := env.receipts.SaveHeader(ctx, receipt.ID, domain.ReceiptHeader{
		Kind: domain.ReceiptKindInvoice, DocumentDate: "2026-03-01",
		IssuerName: "Büromarkt GmbH", GrossAmount: 11900, TaxAmount: 1900, Currency: "EUR",
	}); err != nil {
		t.Fatalf("Kopfdaten: %v", err)
	}
	extended, err := env.receipts.OverrideRetention(ctx, receipt.ID,
		domain.RetentionClassBooks, "Beleg gehört zu einem laufenden Einspruchsverfahren")
	if err != nil {
		t.Fatalf("Verlängerung: %v", err)
	}
	if extended.RetentionUntil != "2036-12-31" {
		t.Fatalf("Frist %s — zehn Jahre auf 2026 enden am 31.12.2036", extended.RetentionUntil)
	}

	// Dieselben Kopfdaten noch einmal, mit einem anderen Belegdatum: die
	// Neuberechnung aus Belegart und Belegdatum ergäbe wieder die kürzere
	// Belegfrist, und die Verlängerung wäre stillschweigend zurückgenommen.
	after, err := env.receipts.SaveHeader(ctx, receipt.ID, domain.ReceiptHeader{
		Kind: domain.ReceiptKindInvoice, DocumentDate: "2026-04-01",
		IssuerName: "Büromarkt GmbH", GrossAmount: 11900, TaxAmount: 1900, Currency: "EUR",
	})
	if err != nil {
		t.Fatalf("Kopfdaten nach der Verlängerung: %v", err)
	}
	if after.RetentionUntil != "2036-12-31" {
		t.Errorf("Frist %s — die Verlängerung auf 2036-12-31 darf nicht zurückfallen", after.RetentionUntil)
	}
	if after.RetentionClass != domain.RetentionClassBooks {
		t.Errorf("Klasse %s — die überschriebene Klasse bleibt", after.RetentionClass)
	}
	if after.RetentionOverrideReason == "" {
		t.Error("der Grund der Verlängerung bleibt am Beleg")
	}
}

// Die Klasse „Inhaltsfehler" prüft die Pflichtangaben gegen die Stammdaten
// (RECH-07 K2): Anschrift und Steuernummer des Ausstellers, eigene Anschrift.
func TestContentFindingsCheckMasterData(t *testing.T) {
	receipt := &domain.Receipt{
		ID: 11, ReceiptNumber: "ER-2026-0011", Direction: domain.DirectionIncoming,
		Kind: domain.ReceiptKindInvoice, DocumentDate: "2026-03-01",
		IssuerName: "Büromarkt GmbH", GrossAmount: 11900, TaxAmount: 1900,
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
			CompanyName: "Pfennig Ventures GmbH", Street: "Hauptstraße 1", ZipCity: "80331 München",
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
		IssuerName: "Büromarkt GmbH", GrossAmount: 11900, TaxAmount: 1900,
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
		IssuerName: "büromarkt gmbh", GrossAmount: 11900, TaxAmount: 1900, Currency: "EUR",
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

// Ein Nachtrag zur ZM zählt eine sonstige Leistung nicht doppelt (UST-04 K3).
//
// Die Januarmeldung ist übermittelt und enthält die sonstige Leistung nicht,
// weil sie nach § 18a Abs. 1 Satz 3 UStG in die Märzmeldung gehört. Wer sie
// deshalb als Nachtrag zum Januar führte, meldete sie zweimal.
func TestZMLateEntriesFollowTheServicePeriod(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")

	// Lieferungen über der Grenze des § 18a Abs. 1 Satz 2 UStG: das Jahr wird
	// monatlich gemeldet.
	env.euInvoice(t, customer.ID, "2026-01-15", 6_000_000, domain.TaxTreatmentIntraCommunitySupply)
	// Die sonstige Leistung aus dem Januar. Sie gehört in die Märzmeldung.
	env.euInvoice(t, customer.ID, "2026-01-20", 500_000, domain.TaxTreatmentReverseChargeSupply)

	// Die Januarmeldung ist übermittelt. Sie wird unmittelbar abgelegt, weil
	// der Weg über ConfirmSubmitted die Festschreibung des Zeitraums verlangt
	// und hier nur der übermittelte Stand gebraucht wird.
	zmRepo := repository.NewZMReturnRepository(env.db)
	january := &domain.ZMReturn{
		FiscalYear: 2026, PeriodType: domain.VatPeriodMonth, PeriodKey: "2026-01",
		PeriodFrom: "2026-01-01", PeriodTo: "2026-01-31",
		Status: domain.VatReturnSubmitted, SubmittedAt: "2026-02-20",
		TransferTicket: "TT-2026-01",
	}
	if err := zmRepo.Create(ctx, january); err != nil {
		t.Fatalf("Januarmeldung ablegen: %v", err)
	}

	march, err := env.zmReturns(t).Draft(ctx, "2026-03")
	if err != nil {
		t.Fatalf("Märzmeldung: %v", err)
	}
	services := domain.Cents(0)
	for _, line := range march.Lines {
		if line.Kind == domain.ZMKindService {
			services += line.Amount
		}
	}
	if services != 500_000 {
		t.Fatalf("%s sonstige Leistungen — die Januarleistung gehört in die Märzmeldung", services)
	}
	for _, late := range march.LateEntries {
		if late.Kind == domain.ZMKindService {
			t.Errorf("die sonstige Leistung steht als Nachtrag zu %s und zugleich in den Zeilen — "+
				"wer dem Nachtrag folgt, meldet sie zweimal", late.PeriodKey)
		}
	}
	// Die Lieferung aus dem Januar ist in der übermittelten Meldung nicht
	// enthalten: sie muss als Nachtrag erscheinen, sonst prüfte der Test eine
	// Erkennung, die gar nicht läuft.
	supplies := 0
	for _, late := range march.LateEntries {
		if late.Kind == domain.ZMKindSupply {
			supplies++
		}
	}
	if supplies == 0 {
		t.Error("die nicht gemeldete Januarlieferung gehört als Nachtrag in die Liste")
	}
}

// „Monat festschreiben" steht ab dem 10. des Folgemonats als offene Aufgabe und
// ab dem Tag nach der Frist als überfällige (UNV-02 K2, Entscheidung 5).
func TestCommitTaskIsOpenFromTheTenthOfTheNextMonth(t *testing.T) {
	env := newTestEnv(t)

	// Die Frist der Januarfestschreibung mit Dauerfristverlängerung: Ende März.
	// Sie liegt damit weiter als der Vorlauf von dreißig Tagen entfernt, und
	// die Aufgabe muss trotzdem erscheinen.
	deadline := domain.Deadline{
		Key: DeadlineKeyCommit + ".2026-01", Title: "Januar 2026 festschreiben",
		DueDate: "2026-03-31", Reference: "GoBD Rz. 107",
		Description: "Nach der Festschreibung nimmt der Monat keine Buchung mehr auf.",
	}

	groupOf := func(today string) (domain.TaskGroup, bool) {
		t.Helper()
		svc := NewTaskService(repository.NewSettingsRepository(env.db), 2026)
		svc.SetDeadlineSource(stubDeadlineSource{deadlines: []domain.Deadline{deadline}})
		list, err := svc.Tasks(context.Background(), TaskOptions{Today: today})
		if err != nil {
			t.Fatalf("Aufgabenliste zum %s: %v", today, err)
		}
		for _, task := range list.Overdue {
			if task.Title == deadline.Title {
				return domain.TaskGroupOverdue, true
			}
		}
		for _, task := range list.Open {
			if task.Title == deadline.Title {
				return domain.TaskGroupOpen, true
			}
		}
		for _, task := range list.Upcoming {
			if task.Title == deadline.Title {
				return domain.TaskGroupUpcoming, true
			}
		}
		return "", false
	}

	// Am 9. Februar ist die Voranmeldung des Januars noch nicht abzugeben: die
	// Festschreibung ist noch nicht dran und keine offene Aufgabe.
	if group, ok := groupOf("2026-02-09"); ok && group == domain.TaskGroupOpen {
		t.Error("vor dem 10. des Folgemonats ist die Festschreibung noch nicht offen")
	}
	if group, ok := groupOf("2026-02-10"); !ok || group != domain.TaskGroupOpen {
		t.Errorf("am 10. des Folgemonats gehört „Januar 2026 festschreiben\" in die offenen "+
			"Aufgaben, erhalten %q (gefunden: %v)", group, ok)
	}
	if group, ok := groupOf("2026-04-01"); !ok || group != domain.TaskGroupOverdue {
		t.Errorf("nach der Frist ist die Festschreibung überfällig, erhalten %q (gefunden: %v)",
			group, ok)
	}
}

// Die Herausgabe der gefilterten Journalmenge steht als Zugriff im Protokoll
// (QUE-02 K2).
func TestFilteredJournalCSVIsLogged(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	auditRepo := repository.NewAuditRepository(env.db)
	env.accounting.SetAuditRepo(auditRepo)

	entry := simpleEntry("6815", domain.AccountKasse, 10000)
	if _, err := env.journal.Post(ctx, entry); err != nil {
		t.Fatalf("Buchung: %v", err)
	}

	csv, err := env.accounting.FilterEntriesCSV(ctx, accounting.JournalFilter{Account: "6815"})
	if err != nil {
		t.Fatalf("gefiltertes Journal als CSV: %v", err)
	}
	if !strings.Contains(csv, "6815") {
		t.Fatal("die Ausgabe enthält die gefilterte Zeile nicht")
	}

	entries, err := auditRepo.FindAll(ctx, 100)
	if err != nil {
		t.Fatalf("Protokoll: %v", err)
	}
	found := false
	for _, e := range entries {
		if e.Action == domain.AuditActionExport && e.EntityType == "JOURNAL_FILTER" {
			found = true
			if !strings.Contains(e.Details, "6815") {
				t.Errorf("der Protokolleintrag nennt den Filter nicht: %s", e.Details)
			}
		}
	}
	if !found {
		t.Error("die Herausgabe der Journalzeilen steht nicht im Protokoll")
	}
}

// Die Steuerangaben der Handbuchung prüft der Buchungsweg selbst und nicht nur
// der Vorbau PostManualEntry (Welle 8, Entscheidung 1).
func TestPostRejectsAManualEntryWithoutATaxCase(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	entry := simpleEntry("6815", domain.AccountBank, 10000)
	entry.TaxTreatment = ""
	if _, err := env.journal.Post(ctx, entry); err == nil {
		t.Fatal("eine Handbuchung ohne Steuerfall wird abgewiesen")
	} else if !strings.Contains(err.Error(), "Steuerfall") {
		t.Errorf("die Meldung nennt den Grund nicht: %v", err)
	}

	// Mit Steuerfall läuft dieselbe Buchung durch.
	ok := simpleEntry("6815", domain.AccountBank, 10000)
	if _, err := env.journal.Post(ctx, ok); err != nil {
		t.Fatalf("mit Steuerfall ist die Buchung buchbar: %v", err)
	}

	// Eine Buchung aus dem Abschluss bleibt ausgenommen: sie trägt eine andere
	// Quelle (Entscheidung 1, „Ausnahme: Source ≠ manual").
	closing := simpleEntry("6815", domain.AccountBank, 10000)
	closing.Source = domain.EntrySourceClosing
	closing.TaxTreatment = ""
	if err := env.journal.ValidatePostable(ctx, closing); err != nil &&
		strings.Contains(err.Error(), "Steuerfall") {
		t.Errorf("die Steuerzeilenpflicht gilt der Handbuchung, nicht dem Abschluss: %v", err)
	}
}

// Mit Dauerfristverlängerung und Nachfrist meldet der Prüflauf einen Monat
// nicht vor dem Tag, an dem die Fristenliste ihn führt.
//
// Sonst widersprächen sich beide: period_not_committed meldete den Januar ab
// dem 31. März, während CommitDueDate ihn erst zum 10. April fällig stellt —
// und commit_overdue, das zwischen beiden Tagen liegen soll, käme nie zum Zug.
func TestNotCommittedFollowsTheCommitDeadline(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.setSettings(t, func(c *domain.CompanySettings) {
		c.PermanentExtension = true
		c.CommitGraceDays = 10
	})

	// Die Frist des Januars: Ende März plus zehn Tage Nachfrist.
	cfg := companySettingsFor(t, env)
	january, err := accounting.ParseVatPeriodKey("2026-01")
	if err != nil {
		t.Fatalf("Zeitraum: %v", err)
	}
	due := CommitDueDate(january, cfg)
	if due != "2026-04-10" {
		t.Fatalf("Frist %s — erwartet den 10. April", due)
	}

	run, err := env.checksOn(t, "2026-04-05").Preview(ctx, CheckRequest{CutoffDate: "2026-04-05"})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
	for _, f := range findingsFor(run, domain.CheckRulePeriodNotCommitted) {
		if f.ObjectID == "2026-01" {
			t.Error("vor der Frist ist der Januar kein Rückstand")
		}
	}

	run, err = env.checksOn(t, "2026-04-11").Preview(ctx, CheckRequest{CutoffDate: "2026-04-11"})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
	found := false
	for _, f := range findingsFor(run, domain.CheckRulePeriodNotCommitted) {
		if f.ObjectID == "2026-01" {
			found = true
		}
	}
	if !found {
		t.Error("nach der Frist gehört der Januar in den Rückstand")
	}
}

// Der Steuersatz wird gegen das Leistungsdatum gehalten (UNV-03 K2).
//
// Eine Rechnung über eine Leistung vom August 2020 mit 19 % passt nicht
// zusammen: damals galten 16 und 5 % (§ 12 Abs. 1 und 2 UStG in der Fassung des
// Zweiten Corona-Steuerhilfegesetzes). Der Vorschlag sagt es und bucht nicht
// stillschweigend.
func TestProposalWarnsAboutARateThatDoesNotFitTheServiceDate(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	receipt := env.filedReceipt(t)

	read := receivedInvoice()
	read.IssueDate = "2020-09-01"
	read.DeliveryDate = "2020-08-15"
	proposal, err := env.einvoicesWith(fakeReader{invoice: read}).Propose(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("Vorschlag: %v", err)
	}
	note := strings.Join(proposal.Notes, " | ")
	if !strings.Contains(note, "16 %") || !strings.Contains(note, "2020-08-15") {
		t.Errorf("der Vorschlag nennt den Satz des Leistungstages nicht: %s", note)
	}

	// Derselbe Beleg mit einem Leistungsdatum von heute: kein Vermerk.
	today := receivedInvoice()
	proposal, err = env.einvoicesWith(fakeReader{invoice: today}).Propose(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("Vorschlag: %v", err)
	}
	for _, n := range proposal.Notes {
		if strings.Contains(n, "passt nicht zum Leistungsdatum") {
			t.Errorf("19 %% passen zum Jahr 2026: %s", n)
		}
	}
}
