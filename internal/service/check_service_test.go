package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// checks liefert den Prüflauf mit einem festen Ausführungstag.
//
// Zwei Regeln messen gegen das spätere von Stichtag und heutigem Tag. Mit der
// echten Uhr hinge das Ergebnis vom Tag des Testlaufs ab; der 1. Januar 2026
// liegt vor jedem Stichtag der Tests, sodass dort der Stichtag gilt. Wer die
// zweite Hälfte prüfen will, nimmt checksOn.
func (e *testEnv) checks(t *testing.T) *CheckService {
	t.Helper()
	return e.checksOn(t, "2026-01-01")
}

func (e *testEnv) checksOn(t *testing.T, today string) *CheckService {
	t.Helper()
	svc := NewCheckService(
		e.journalRepo,
		e.receiptRepo,
		repository.NewBankRepository(e.db),
		repository.NewInvoiceRepository(e.db),
		e.numberRepo,
		repository.NewSettingsRepository(e.db),
		repository.NewFestschreibungRepository(e.db),
		repository.NewVatReturnRepository(e.db),
		repository.NewCheckRunRepository(e.db),
		repository.NewAuditRepository(e.db),
		e.fiscalYear,
	)
	day, err := time.Parse("2006-01-02", today)
	if err != nil {
		t.Fatalf("Ausführungstag %q: %v", today, err)
	}
	svc.now = func() time.Time { return day }
	return svc
}

// setReceivedAt setzt den Belegeingang. Er entscheidet über die Fristen des
// Prüflaufs und wird beim Ablegen aus der Datei nicht erkannt.
func (e *testEnv) setReceivedAt(t *testing.T, receiptID uint, date string) {
	t.Helper()
	if err := e.db.Model(&domain.Receipt{}).Where("id = ?", receiptID).
		Update("received_at", date).Error; err != nil {
		t.Fatalf("Belegeingang setzen: %v", err)
	}
}

// runChecks führt einen Prüflauf zum Stichtag aus.
func runChecks(t *testing.T, svc *CheckService, cutoff string) *domain.CheckRun {
	t.Helper()
	run, err := svc.Run(context.Background(), CheckRequest{CutoffDate: cutoff})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
	return run
}

// findingsFor liefert die Befunde einer Regel.
func findingsFor(run *domain.CheckRun, rule string) []domain.CheckFinding {
	out := []domain.CheckFinding{}
	for _, f := range run.Findings {
		if f.Rule == rule {
			out = append(out, f)
		}
	}
	return out
}

func assertRule(t *testing.T, run *domain.CheckRun, rule string, want int, severity domain.CheckSeverity) {
	t.Helper()
	got := findingsFor(run, rule)
	if len(got) != want {
		t.Fatalf("Regel %s hat %d Befunde, erwartet %d: %+v", rule, len(got), want, got)
	}
	for _, f := range got {
		if f.Severity != severity {
			t.Errorf("Regel %s: Schwere %q, erwartet %q", rule, f.Severity, severity)
		}
	}
}

// openItemStub liefert der Prüfung feste offene Posten.
type openItemStub struct{ items []domain.OpenItem }

func (s openItemStub) OpenItemsAt(context.Context, string) ([]domain.OpenItem, error) {
	return s.items, nil
}

// accountStub liefert der Prüfung einen festen Kontenplan mit Salden.
type accountStub struct{ accounts []domain.Account }

func (s accountStub) AccountsForYear(context.Context, int) ([]domain.Account, error) {
	return s.accounts, nil
}

// depreciationStub meldet eine feste Liste offener Abschreibungen.
type depreciationStub struct{ due []DepreciationDue }

func (s depreciationStub) PendingDepreciation(context.Context, int) ([]DepreciationDue, error) {
	return s.due, nil
}

// Eine Buchung ohne Beleg blockiert die Festschreibung: was danach fehlt, fehlt
// für immer.
func TestCheckEntryWithoutReceipt(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Negativ: der gebuchte Eingangsbeleg hängt an seiner Buchung.
	vendor := env.vendor(t, "Lieferant", "DE", "")
	if _, err := env.posting.PostIncomingReceipt(ctx,
		env.receipt(t, vendor.ID, "fremdleistungen", 100000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)); err != nil {
		t.Fatalf("Beleg buchen: %v", err)
	}
	assertRule(t, runChecks(t, env.checks(t), "2026-03-31"), domain.CheckRuleEntryWithoutReceipt, 0, domain.CheckBlocking)

	// Positiv: eine Handbuchung ohne Beleg.
	if _, err := env.journal.Post(ctx, simpleEntry(domain.AccountKasse, domain.AccountBank, 10000)); err != nil {
		t.Fatalf("Handbuchung: %v", err)
	}
	assertRule(t, runChecks(t, env.checks(t), "2026-03-31"), domain.CheckRuleEntryWithoutReceipt, 1, domain.CheckBlocking)
}

// Folgebuchungen brauchen keinen eigenen Beleg: ihr Nachweis ist die Buchung,
// auf die sie sich beziehen.
func TestCheckEntryWithoutReceiptSkipsDerivedBookings(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	entry := simpleEntry(domain.AccountKasse, domain.AccountBank, 10000)
	entry.Source = domain.EntrySourcePayment
	if _, err := env.journal.Post(ctx, entry); err != nil {
		t.Fatalf("Zahlungsbuchung: %v", err)
	}
	assertRule(t, runChecks(t, env.checks(t), "2026-03-31"), domain.CheckRuleEntryWithoutReceipt, 0, domain.CheckBlocking)
}

// Ein abgelegter, aber nicht gebuchter Beleg blockiert, wenn er in den
// festzuschreibenden Zeitraum gehört — danach nimmt der Zeitraum keine Buchung
// mehr auf.
func TestCheckReceiptUnbooked(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Negativ: ein gebuchter Beleg.
	vendor := env.vendor(t, "Lieferant", "DE", "")
	if _, err := env.posting.PostIncomingReceipt(ctx,
		env.receipt(t, vendor.ID, "fremdleistungen", 100000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)); err != nil {
		t.Fatalf("Beleg buchen: %v", err)
	}
	assertRule(t, runChecks(t, env.checks(t), "2026-03-31"), domain.CheckRuleReceiptUnbooked, 0, domain.CheckBlocking)

	// Positiv: ein abgelegter Beleg mit Eingang im Zeitraum, aber ohne Buchung.
	// Er blockiert, weil der Zeitraum nach der Festschreibung keine Buchung mehr
	// aufnimmt.
	env.setReceivedAt(t, env.fileIncoming(t, "offen.pdf").ID, "2026-03-05")
	assertRule(t, runChecks(t, env.checks(t), "2026-03-31"), domain.CheckRuleReceiptUnbooked, 1, domain.CheckBlocking)

	// Derselbe Beleg, ein Stichtag davor: sein Eingang liegt nach dem Stichtag,
	// er gehört also noch nicht in den festzuschreibenden Zeitraum. Dann ist er
	// ein Hinweis und keine Sperre — die Buchung ist noch möglich.
	found := findingsFor(runChecks(t, env.checks(t), "2026-03-01"), domain.CheckRuleReceiptUnbooked)
	if len(found) != 1 || found[0].Severity != domain.CheckWarning {
		t.Fatalf("erwartet einen Hinweis (warning) für den Beleg nach dem Stichtag, erhalten %+v", found)
	}
}

// Ein Beleg, der länger als die Erfassungsfrist unbearbeitet liegt, ist ein
// eigener Hinweis: GoBD Rz. 47 nennt zehn Tage für unbare Geschäftsvorfälle.
func TestCheckReceiptOverdue(t *testing.T) {
	env := newTestEnv(t)
	filed := env.fileIncoming(t, "offen.pdf")

	env.setReceivedAt(t, filed.ID, "2026-03-01")

	// Negativ: fünf Tage danach ist die Frist nicht abgelaufen.
	assertRule(t, runChecks(t, env.checks(t), "2026-03-06"), domain.CheckRuleReceiptOverdue, 0, domain.CheckWarning)

	// Positiv: zwanzig Tage danach schon.
	assertRule(t, runChecks(t, env.checks(t), "2026-03-21"), domain.CheckRuleReceiptOverdue, 1, domain.CheckWarning)

	// Und der späte Lauf zu einem zurückliegenden Stichtag meldet ihn auch:
	// gemessen wird gegen den Ausführungstag, sonst bliebe ein Beleg, der seit
	// zwei Monaten liegt, unbemerkt, weil der Stichtag kurz nach seinem Eingang
	// liegt.
	assertRule(t, runChecks(t, env.checksOn(t, "2026-05-15"), "2026-03-06"), domain.CheckRuleReceiptOverdue, 1, domain.CheckWarning)
}

// Ein Bankumsatz ohne Zuordnung blockiert: der Geschäftsvorfall steht auf dem
// Kontoauszug, aber in keiner Buchung.
func TestCheckBankUnmatched(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	bankRepo := repository.NewBankRepository(env.db)

	if _, err := bankRepo.CreateBatch(ctx, []domain.BankTransaction{{
		FiscalYear: 2026, AccountIBAN: "DE02120300000000202051",
		BookingDate: "2026-02-10", ValueDate: "2026-02-10",
		Amount: 50000, CounterpartyName: "Kunde", MatchStatus: domain.MatchStatusUnmatched,
	}}); err != nil {
		t.Fatalf("Bankumsatz: %v", err)
	}
	assertRule(t, runChecks(t, env.checks(t), "2026-03-31"), domain.CheckRuleBankUnmatched, 1, domain.CheckBlocking)

	// Negativ: nach der Zuordnung ist der Befund weg.
	transactions, err := bankRepo.FindAll(ctx, 2026)
	if err != nil {
		t.Fatalf("Bankumsätze lesen: %v", err)
	}
	if err := bankRepo.SetMatchStatus(ctx, transactions[0].ID, domain.MatchStatusMatched); err != nil {
		t.Fatalf("Zuordnung: %v", err)
	}
	assertRule(t, runChecks(t, env.checks(t), "2026-03-31"), domain.CheckRuleBankUnmatched, 0, domain.CheckBlocking)
}

// Geldtransit und durchlaufende Posten sind Durchgangskonten: ein Saldo darauf
// heißt, dass die Gegenbuchung fehlt.
func TestCheckInterimBalance(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Positiv: Geld ist auf den Transit gebucht und nicht wieder herunter.
	if _, err := env.journal.Post(ctx, simpleEntry(domain.AccountGeldtransit, domain.AccountBank, 50000)); err != nil {
		t.Fatalf("Transitbuchung: %v", err)
	}
	assertRule(t, runChecks(t, env.checks(t), "2026-03-31"), domain.CheckRuleInterimBalance, 1, domain.CheckBlocking)

	// Negativ: die Gegenbuchung gleicht den Transit aus.
	if _, err := env.journal.Post(ctx, simpleEntry(domain.AccountKasse, domain.AccountGeldtransit, 50000)); err != nil {
		t.Fatalf("Gegenbuchung: %v", err)
	}
	assertRule(t, runChecks(t, env.checks(t), "2026-03-31"), domain.CheckRuleInterimBalance, 0, domain.CheckBlocking)
}

// Gleicher Partner, gleiche Belegnummer, gleicher Betrag: dieselbe Rechnung
// zweimal erfasst — und mit ihr ein zweiter Vorsteuerabzug.
func TestCheckDuplicateReceipt(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Lieferant", "DE", "")

	book := func() {
		t.Helper()
		entry := simpleEntry("6815", vendor.LedgerAccount, 100000)
		entry.DocumentNumber = "RG-4711"
		entry.ContactID = &vendor.ID
		entry.Lines[1].ContactID = &vendor.ID
		if _, err := env.journal.Post(ctx, entry); err != nil {
			t.Fatalf("Buchung: %v", err)
		}
	}

	book()
	assertRule(t, runChecks(t, env.checks(t), "2026-03-31"), domain.CheckRuleDuplicateReceipt, 0, domain.CheckWarning)

	book()
	assertRule(t, runChecks(t, env.checks(t), "2026-03-31"), domain.CheckRuleDuplicateReceipt, 1, domain.CheckWarning)
}

// Mehr zugeordnet als der Posten ausmacht: der übliche Weg, auf dem eine
// Rechnung zweimal überwiesen wird.
func TestCheckDuplicatePayment(t *testing.T) {
	env := newTestEnv(t)

	svc := env.checks(t)
	svc.SetOpenItemSource(openItemStub{items: []domain.OpenItem{
		{EntryID: 1, EntryNumber: "2026-000001", ContactName: "Lieferant", GrossAmount: 119000, SettledAmount: 119000},
	}})
	assertRule(t, runChecks(t, svc, "2026-03-31"), domain.CheckRuleDuplicatePayment, 0, domain.CheckWarning)

	svc = env.checks(t)
	svc.SetOpenItemSource(openItemStub{items: []domain.OpenItem{
		{EntryID: 1, EntryNumber: "2026-000001", ContactName: "Lieferant", GrossAmount: 119000, SettledAmount: 238000},
	}})
	found := findingsFor(runChecks(t, svc, "2026-03-31"), domain.CheckRuleDuplicatePayment)
	if len(found) != 1 {
		t.Fatalf("erwartet einen Befund, erhalten %d", len(found))
	}
	if !strings.Contains(found[0].Message, "1.190,00") {
		t.Errorf("der Befund sollte die Überzahlung beziffern: %s", found[0].Message)
	}
}

// Eine Lücke im Nummernkreis muss erklärbar sein: § 14 Abs. 4 Nr. 4 UStG
// verlangt eine fortlaufende Rechnungsnummer.
func TestCheckNumberGap(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	env.fileIncoming(t, "eins.pdf")
	env.fileIncoming(t, "zwei.pdf")
	assertRule(t, runChecks(t, env.checks(t), "2026-03-31"), domain.CheckRuleNumberGap, 0, domain.CheckWarning)

	// Eine verbrauchte Nummer ohne Beleg — so sieht ein zurückgerollter
	// Anlagevorgang aus.
	if _, err := env.numberRepo.Allocate(ctx, domain.NumberRangeReceipt, 2026); err != nil {
		t.Fatalf("Nummer vergeben: %v", err)
	}
	env.fileIncoming(t, "vier.pdf")

	found := findingsFor(runChecks(t, env.checks(t), "2026-03-31"), domain.CheckRuleNumberGap)
	if len(found) != 1 {
		t.Fatalf("erwartet einen Befund, erhalten %d: %+v", len(found), found)
	}
	if !strings.Contains(found[0].Message, "ER-2026-0003") {
		t.Errorf("der Befund sollte die fehlende Nummer nennen: %s", found[0].Message)
	}
}

// Ein Konto mit Saldo ohne Gliederungsposition erschiene in keinem Posten des
// Abschlusses. Vor der Jahresfestschreibung blockiert das.
func TestCheckAccountUnmapped(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	unmapped := domain.Account{
		Number: "1234", Name: "Konto ohne Position", Type: domain.AccountTypeAsset,
		StatementType: "Bilanz", PositionID: "gibt.es.nicht", DebitSum: 10000,
	}

	svc := env.checks(t)
	svc.SetAccountSource(accountStub{accounts: []domain.Account{unmapped}})
	run, err := svc.Run(ctx, CheckRequest{CutoffDate: "2026-12-31", PeriodType: "year"})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
	assertRule(t, run, domain.CheckRuleAccountUnmapped, 1, domain.CheckBlocking)

	// Negativ: dieselbe Prüfung ohne Saldo und ohne Jahresanlass.
	unmapped.DebitSum = 0
	svc = env.checks(t)
	svc.SetAccountSource(accountStub{accounts: []domain.Account{unmapped}})
	run, err = svc.Run(ctx, CheckRequest{CutoffDate: "2026-12-31", PeriodType: "year"})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
	assertRule(t, run, domain.CheckRuleAccountUnmapped, 0, domain.CheckBlocking)
}

// Die AfA ist eine Abschlussbuchung zum Bilanzstichtag und lässt sich nach der
// Jahresfestschreibung nicht nachholen. Im Monatslauf ist sie nicht fällig.
func TestCheckDepreciationMissingOnlyBeforeTheYearCommit(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	pending := depreciationStub{due: []DepreciationDue{
		{AssetID: 1, InventoryNumber: "AV-2026-0001", Name: "Maschine", Due: 120000},
	}}

	svc := env.checks(t)
	svc.SetDepreciationSource(pending)
	monthly, err := svc.Run(ctx, CheckRequest{CutoffDate: "2026-03-31", PeriodType: "month"})
	if err != nil {
		t.Fatalf("Monatslauf: %v", err)
	}
	assertRule(t, monthly, domain.CheckRuleDepreciationMissing, 0, domain.CheckBlocking)

	svc = env.checks(t)
	svc.SetDepreciationSource(pending)
	yearly, err := svc.Run(ctx, CheckRequest{CutoffDate: "2026-12-31", PeriodType: "year"})
	if err != nil {
		t.Fatalf("Jahreslauf: %v", err)
	}
	assertRule(t, yearly, domain.CheckRuleDepreciationMissing, 1, domain.CheckBlocking)
}

// Ein abgelaufener Voranmeldungszeitraum ohne bestätigte Übermittlung ist ein
// Hinweis — die Anmeldung geschieht außerhalb von Buchfink, aber sie fehlt.
func TestCheckVatReturnMissing(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde", "DE", "")
	env.domesticInvoice(t, customer.ID, "2026-02-10", 100000)

	found := findingsFor(runChecks(t, env.checks(t), "2026-04-30"), domain.CheckRuleVatReturnMissing)
	if len(found) != 1 || found[0].ObjectID != "2026-Q1" {
		t.Fatalf("erwartet einen Befund zu 2026-Q1, erhalten %+v", found)
	}

	// Negativ: am letzten Tag des Quartals ist die Anmeldung noch nicht fällig
	// (10. April). Ein Hinweis beim Festschreiben zum Quartalsende beanstandete
	// die Reihenfolge, die Buchfink selbst verlangt — bestätigt werden kann die
	// Anmeldung erst nach der Festschreibung.
	assertRule(t, runChecks(t, env.checks(t), "2026-03-31"), domain.CheckRuleVatReturnMissing, 0, domain.CheckWarning)

	// Negativ: nach der Bestätigung ist der Zeitraum erledigt.
	env.commitUntil(t, "2026-03-31")
	vat := env.vatReturns(t)
	saved, err := vat.Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("Anmeldung speichern: %v", err)
	}
	if _, err := vat.ConfirmSubmitted(ctx, saved.ID, "2026-04-09", "TT-1", ""); err != nil {
		t.Fatalf("bestätigen: %v", err)
	}
	assertRule(t, runChecks(t, env.checks(t), "2026-04-30"), domain.CheckRuleVatReturnMissing, 0, domain.CheckWarning)
}

// Wer vom Voranmeldungsverfahren befreit ist (§ 18 Abs. 2 Satz 3 UStG), gibt
// keine Voranmeldung ab — der Prüflauf darf ihm keine fehlende anmahnen.
//
// Die Fristenliste schließt denselben Fall aus (DeadlineService.vatDeadlines).
// Widersprächen sich beide, stünde im Prüfbericht eine Pflicht, die kein Termin
// kennt — und niemand wüsste, welche der beiden Stellen recht hat.
func TestCheckVatReturnMissingSkipsAnnualFilers(t *testing.T) {
	env := newTestEnv(t)
	env.setSettings(t, func(c *domain.CompanySettings) { c.VatPeriod = "year" })
	customer := env.customer(t, "Kunde", "DE", "")
	env.domesticInvoice(t, customer.ID, "2026-02-10", 100000)

	assertRule(t, runChecks(t, env.checks(t), "2027-06-30"), domain.CheckRuleVatReturnMissing, 0, domain.CheckWarning)
}

// Der Zeitpunkt des Laufs kommt aus derselben Uhr wie seine Fristen.
//
// Der Lauf ist eine Aussage über einen Zeitpunkt, und er trägt ihn selbst. Käme
// er aus time.Now(), während die Regeln nach der gestellten Uhr rechnen, trüge
// ein Lauf ein anderes Datum als seine eigenen Befunde — und das gespeicherte
// Protokoll wäre nicht mehr nachzurechnen.
func TestCheckRunTakesItsTimeFromTheServiceClock(t *testing.T) {
	env := newTestEnv(t)
	run := runChecks(t, env.checksOn(t, "2026-05-20"), "2026-04-30")
	if got := run.CreatedAt.Format("2006-01-02"); got != "2026-05-20" {
		t.Errorf("Zeitpunkt des Laufs = %s, erwartet 2026-05-20", got)
	}

	stored, err := env.checks(t).Runs(context.Background(), 2026)
	if err != nil || len(stored) == 0 {
		t.Fatalf("gespeicherte Läufe lesen: %v (%d)", err, len(stored))
	}
	if got := stored[0].CreatedAt.Format("2006-01-02"); got != "2026-05-20" {
		t.Errorf("gespeicherter Zeitpunkt = %s, erwartet 2026-05-20", got)
	}
}

// Ein Monat, dessen Folgemonat abgelaufen ist und der immer noch nicht
// festgeschrieben wurde, gehört auf die Aufgabenliste.
func TestCheckCommitOverdue(t *testing.T) {
	env := newTestEnv(t)

	// Negativ: Ende Februar ist der Januar noch nicht überfällig.
	assertRule(t, runChecks(t, env.checks(t), "2026-02-28"), domain.CheckRuleCommitOverdue, 0, domain.CheckWarning)

	// Positiv: Mitte März ist er es.
	found := findingsFor(runChecks(t, env.checks(t), "2026-03-15"), domain.CheckRuleCommitOverdue)
	if len(found) != 1 || found[0].ObjectID != "2026-01" {
		t.Fatalf("erwartet einen Befund zum Januar 2026, erhalten %+v", found)
	}

	// Nach der Festschreibung ist er weg.
	env.commitUntil(t, "2026-02-28")
	assertRule(t, runChecks(t, env.checks(t), "2026-03-15"), domain.CheckRuleCommitOverdue, 0, domain.CheckWarning)
}

// Blockierende Befunde verhindern die Festschreibung. Übergangen wird nur mit
// Begründung, und die wird gespeichert und protokolliert.
func TestCommitIsBlockedUntilOverriddenWithAReason(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	if _, err := env.journal.Post(ctx, simpleEntry(domain.AccountKasse, domain.AccountBank, 10000)); err != nil {
		t.Fatalf("Handbuchung ohne Beleg: %v", err)
	}

	svc := env.checks(t)
	run, err := svc.EnsureCommittable(ctx, CheckRequest{CutoffDate: "2026-03-31", PeriodType: "month"})
	if err == nil {
		t.Fatal("ein blockierender Befund muss die Festschreibung verhindern")
	}
	if !run.HasBlocking() {
		t.Error("der Lauf sollte blockierende Befunde führen")
	}
	if !strings.Contains(err.Error(), "Begründung") {
		t.Errorf("die Meldung sollte den Weg über die Begründung nennen: %v", err)
	}

	const reason = "Der Beleg ist beim Lieferanten angefordert und kommt nicht mehr."
	run, err = svc.EnsureCommittable(ctx, CheckRequest{
		CutoffDate: "2026-03-31", PeriodType: "month", OverrideReason: reason,
	})
	if err != nil {
		t.Fatalf("mit Begründung muss die Festschreibung durchgehen: %v", err)
	}
	if run.OverrideReason != reason {
		t.Errorf("die Begründung am Lauf = %q, erwartet %q", run.OverrideReason, reason)
	}

	stored, err := repository.NewCheckRunRepository(env.db).FindByID(ctx, run.ID)
	if err != nil {
		t.Fatalf("Lauf laden: %v", err)
	}
	if stored.OverrideReason != reason {
		t.Errorf("die gespeicherte Begründung = %q", stored.OverrideReason)
	}
	if len(stored.Findings) != len(run.Findings) {
		t.Errorf("der gespeicherte Lauf führt %d Befunde, der ausgeführte %d", len(stored.Findings), len(run.Findings))
	}

	logs, err := repository.NewAuditRepository(env.db).FindAll(ctx, 200)
	if err != nil {
		t.Fatalf("Protokoll: %v", err)
	}
	logged := false
	for _, l := range logs {
		if l.EntityType == "CHECK_RUN" && strings.Contains(l.Details, reason) {
			logged = true
		}
	}
	if !logged {
		t.Error("das Übergehen gehört ins Protokoll — sonst ist es keine Kontrolle, sondern ihre Umgehung")
	}
}

// Ohne blockierende Befunde braucht es keine Begründung.
func TestCommitPassesWithoutFindings(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	vendor := env.vendor(t, "Lieferant", "DE", "")
	if _, err := env.posting.PostIncomingReceipt(ctx,
		env.receipt(t, vendor.ID, "fremdleistungen", 100000, domain.TaxRateStandard, domain.TaxTreatmentDomestic)); err != nil {
		t.Fatalf("Beleg buchen: %v", err)
	}

	run, err := env.checks(t).EnsureCommittable(ctx, CheckRequest{CutoffDate: "2026-03-31", PeriodType: "month"})
	if err != nil {
		t.Fatalf("ohne blockierende Befunde muss die Festschreibung durchgehen: %v", err)
	}
	if run.HasBlocking() {
		t.Errorf("unerwartete blockierende Befunde: %+v", run.Blocking())
	}
}

// Die Läufe werden gespeichert und sind in der Oberfläche einsehbar.
func TestCheckRunsArePersisted(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.checks(t)

	runChecks(t, svc, "2026-01-31")
	runChecks(t, svc, "2026-02-28")

	runs, err := svc.Runs(ctx, 2026)
	if err != nil {
		t.Fatalf("Läufe lesen: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("erwartet zwei Läufe, erhalten %d", len(runs))
	}
	latest, err := svc.Latest(ctx)
	if err != nil || latest == nil {
		t.Fatalf("jüngster Lauf: %v", err)
	}
	if latest.CutoffDate != "2026-02-28" {
		t.Errorf("jüngster Lauf zum %s, erwartet 2026-02-28", latest.CutoffDate)
	}
}

// Die Vorschau vor der Festschreibung wird nicht abgelegt.
//
// Sie zeigt denselben Bericht, ohne ihn ins Protokoll zu schreiben. Läge sie
// dort, stünden je Festschreibung zwei Läufe — der aus dem Dialog und der aus
// der Festschreibung —, und die Begründung für ein Übergehen hinge an dem Lauf,
// den der Anwender gar nicht gesehen hat.
func TestCheckPreviewIsNotPersisted(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.checks(t)
	// Ein Bankumsatz ohne Zuordnung: der Bericht hat etwas zu melden, sonst
	// prüfte der Test nur eine leere Liste.
	if _, err := repository.NewBankRepository(env.db).CreateBatch(ctx, []domain.BankTransaction{{
		FiscalYear: 2026, AccountIBAN: "DE02120300000000202051",
		BookingDate: "2026-01-20", ValueDate: "2026-01-20",
		Amount: 50000, CounterpartyName: "Kunde", MatchStatus: domain.MatchStatusUnmatched,
	}}); err != nil {
		t.Fatalf("Bankumsatz: %v", err)
	}

	preview, err := svc.Preview(ctx, CheckRequest{CutoffDate: "2026-01-31"})
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if preview.ID != 0 {
		t.Errorf("die Vorschau bekommt keine Kennung, erhalten %d", preview.ID)
	}
	if !preview.HasBlocking() {
		t.Error("die Vorschau muss den nicht zugeordneten Bankumsatz melden")
	}

	runs, err := svc.Runs(ctx, 2026)
	if err != nil {
		t.Fatalf("Läufe lesen: %v", err)
	}
	if len(runs) != 0 {
		t.Fatalf("die Vorschau darf keinen Lauf ablegen, erhalten %d", len(runs))
	}

	// Der Lauf der Festschreibung dagegen steht im Protokoll.
	if _, err := svc.EnsureCommittable(ctx, CheckRequest{
		CutoffDate: "2026-01-31", PeriodType: "month", OverrideReason: "Kontoauszug fehlt noch",
	}); err != nil {
		t.Fatalf("Festschreibung mit Begründung: %v", err)
	}
	runs, err = svc.Runs(ctx, 2026)
	if err != nil {
		t.Fatalf("Läufe lesen: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("erwartet genau einen abgelegten Lauf je Festschreibung, erhalten %d", len(runs))
	}
	if runs[0].OverrideReason != "Kontoauszug fehlt noch" {
		t.Errorf("die Begründung steht am abgelegten Lauf, erhalten %q", runs[0].OverrideReason)
	}
}

// Eine ausgestellte Ausgangsrechnung hat einen Beleg — über den Beleg.
//
// Seit Nummer, Rechnung und Buchung in einer Transaktion entstehen, kommt das
// Dokument danach: die Buchung kann den Beleg nicht mehr tragen, weil der
// Beleg-Hash in ihrem Kettenhash steht und ein Nachtrag die Kette bräche. Der
// Nachweis läuft deshalb vom Beleg zur Buchung, und der Prüflauf zählt beide
// Richtungen. Ohne das meldete er jede Ausgangsrechnung als „Buchung ohne
// Beleg" und keine Festschreibung käme durch.
func TestCheckIssuedInvoiceCarriesItsReceipt(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde", "DE", "")

	inv := &domain.Invoice{
		ContactID: customer.ID, Date: "2026-02-10",
		ServiceDateFrom: "2026-02-10", ServiceDateTo: "2026-02-10",
		TaxTreatment: domain.TaxTreatmentDomestic,
		Items: []domain.InvoiceItem{{
			Description: "Leistung", QuantityMilli: 1000, UnitPrice: 200000, TaxRate: domain.TaxRateStandard,
		}},
	}
	if err := env.invoicesWithDocuments(t).Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}

	entry, err := env.journalRepo.FindByID(ctx, *inv.JournalEntryID)
	if err != nil {
		t.Fatalf("Buchung laden: %v", err)
	}
	if inv.ReceiptID == nil {
		t.Fatal("die ausgestellte Rechnung muss auf ihren Beleg zeigen")
	}
	receipt, err := env.receipts.Get(ctx, *inv.ReceiptID)
	if err != nil {
		t.Fatalf("Beleg laden: %v", err)
	}
	if receipt.JournalEntryID == nil || *receipt.JournalEntryID != entry.ID {
		t.Errorf("der Beleg zeigt auf die Buchung %v, gebucht wurde aber %d", receipt.JournalEntryID, entry.ID)
	}
	assertRule(t, runChecks(t, env.checks(t), "2026-03-31"), domain.CheckRuleEntryWithoutReceipt, 0, domain.CheckBlocking)
}

// Die Begründung steht nur an einem Lauf, der etwas zu übergehen hatte. Sonst
// behauptete das Protokoll ein Übergehen, das nie stattgefunden hat.
func TestCheckOverrideReasonOnlyWithBlockingFindings(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	svc := env.checks(t)
	run, err := svc.Run(ctx, CheckRequest{
		CutoffDate: "2026-03-31", PeriodType: "month",
		OverrideReason: "Begründung ohne Anlass",
	})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
	if run.HasBlocking() {
		t.Fatalf("der Lauf sollte ohne blockierende Befunde sein: %+v", run.Blocking())
	}
	if run.OverrideReason != "" {
		t.Errorf("die Begründung am Lauf = %q, erwartet keine", run.OverrideReason)
	}

	logs, err := repository.NewAuditRepository(env.db).FindAll(ctx, 200)
	if err != nil {
		t.Fatalf("Protokoll: %v", err)
	}
	for _, l := range logs {
		if l.EntityType == "CHECK_RUN" && strings.Contains(l.Details, "übergangen") {
			t.Errorf("das Protokoll behauptet ein Übergehen: %s", l.Details)
		}
	}
}
