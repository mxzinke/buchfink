package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Die Aufgabenliste und der Monatsabschluss.
//
// Geprüft wird die Zusage aus Architektur 6.1: jede Quelle erzeugt genau dann
// eine Zeile, wenn ihr Zustand es verlangt — und keine, wenn nicht. Die Quellen
// sind deshalb Attrappen: eine Aufgabenliste, die nur mit einer vollständig
// verdrahteten Anwendung prüfbar wäre, prüfte die Verdrahtung, nicht die
// Liste.

type stubCheckSource struct {
	run *domain.CheckRun
	err error
}

func (s stubCheckSource) Preview(context.Context, CheckRequest) (*domain.CheckRun, error) {
	return s.run, s.err
}

type stubObligationSource struct{ obligations []domain.Obligation }

func (s stubObligationSource) List(context.Context, ObligationQuery) ([]domain.Obligation, error) {
	return s.obligations, nil
}

type stubBankSource struct{ transactions []domain.BankTransaction }

func (s stubBankSource) GetTransactions(context.Context, int) ([]domain.BankTransaction, error) {
	return s.transactions, nil
}

type stubReceiptSource struct{ receipts []domain.Receipt }

func (s stubReceiptSource) List(context.Context, int, domain.ReceiptStatus) ([]domain.Receipt, error) {
	return s.receipts, nil
}

type stubOpenItemSource struct{ items []domain.OpenItem }

func (s stubOpenItemSource) OpenItems(context.Context) ([]domain.OpenItem, error) {
	return s.items, nil
}

type stubBackupSource struct {
	run  *domain.BackupRun
	runs []domain.BackupRun
}

func (s stubBackupSource) LastSuccessful(context.Context) (*domain.BackupRun, error) {
	return s.run, nil
}

func (s stubBackupSource) GetRuns(_ context.Context, limit int) ([]domain.BackupRun, error) {
	if limit > 0 && len(s.runs) > limit {
		return s.runs[:limit], nil
	}
	return s.runs, nil
}

type stubExemptionSource struct{ warnings []ExemptionCertificateWarning }

func (s stubExemptionSource) ExemptionCertificateWarnings(
	context.Context, string,
) ([]ExemptionCertificateWarning, error) {
	return s.warnings, nil
}

type stubCarryForwardSource struct {
	preview *CarryForwardPreview
}

func (s stubCarryForwardSource) CarryForwardState(context.Context, int) (*CarryForwardPreview, error) {
	return s.preview, nil
}

type stubAppropriationSource struct{ decision *domain.Appropriation }

func (s stubAppropriationSource) Appropriation(context.Context, int) (*domain.Appropriation, error) {
	return s.decision, nil
}

// tasksFor baut die Aufgabenliste über den Attrappen und liefert sie nach
// Schlüssel.
func tasksFor(t *testing.T, svc *TaskService, opts TaskOptions) map[string]domain.Task {
	t.Helper()
	list, err := svc.Tasks(context.Background(), opts)
	if err != nil {
		t.Fatalf("Aufgabenliste: %v", err)
	}
	out := map[string]domain.Task{}
	for _, group := range [][]domain.Task{list.Overdue, list.Open, list.Upcoming} {
		for _, task := range group {
			out[task.Key] = task
		}
	}
	return out
}

// Die leere Buchhaltung erzeugt keine Aufgabe — und liefert trotzdem drei
// Listen statt `null`.
func TestTasksAreEmptyWhenNothingIsDue(t *testing.T) {
	env := newTestEnv(t)
	svc := NewTaskService(repository.NewSettingsRepository(env.db))
	svc.SetBankSource(stubBankSource{})
	svc.SetReceiptSource(stubReceiptSource{})
	svc.SetOpenItemSource(stubOpenItemSource{})
	svc.SetObligationSource(stubObligationSource{})

	list, err := svc.Tasks(context.Background(), TaskOptions{Today: "2026-03-10"})
	if err != nil {
		t.Fatalf("Aufgabenliste: %v", err)
	}
	if list.Total() != 0 {
		t.Errorf("erwartet keine Aufgabe, erhalten %d", list.Total())
	}
	if list.Overdue == nil || list.Open == nil || list.Upcoming == nil {
		t.Error("die drei Gruppen müssen leere Listen sein und nicht nil")
	}
	if list.Today != "2026-03-10" {
		t.Errorf("Stichtag = %q, erwartet den übergebenen Tag", list.Today)
	}
}

// Jede Quelle erzeugt ihre Zeile, und zwar in der richtigen Gruppe.
func TestEverySourceProducesItsTask(t *testing.T) {
	env := newTestEnv(t)
	today := "2026-03-10"

	svc := NewTaskService(repository.NewSettingsRepository(env.db))
	svc.SetCheckSource(stubCheckSource{run: &domain.CheckRun{Findings: []domain.CheckFinding{
		{Rule: domain.CheckRuleEntryWithoutReceipt, Severity: domain.CheckBlocking, Message: "Buchung ohne Beleg"},
		{Rule: domain.CheckRuleEntryWithoutReceipt, Severity: domain.CheckBlocking, Message: "noch eine"},
		{Rule: domain.CheckRuleVatIDUnconfirmed, Severity: domain.CheckWarning, Message: "Bestätigung fehlt"},
		// Die Bankumsätze haben ihre eigene, genauere Zeile aus der Quelle
		// selbst; als Befund dürfen sie nicht ein zweites Mal erscheinen.
		{Rule: domain.CheckRuleBankUnmatched, Severity: domain.CheckWarning, Message: "Umsatz offen"},
	}}})
	svc.SetObligationSource(stubObligationSource{obligations: []domain.Obligation{
		{Key: "ustva.2026-02", Kind: domain.ObligationVatReturn, Title: "Umsatzsteuer-Voranmeldung Februar 2026",
			DueDate: "2026-03-10", Description: "Die Voranmeldung ist abzugeben.", Status: domain.ObligationOpen},
		{Key: "ustva.2026-01", Kind: domain.ObligationVatReturn, Title: "Umsatzsteuer-Voranmeldung Januar 2026",
			DueDate: "2026-02-10", Description: "Die Voranmeldung ist abzugeben.", Status: domain.ObligationOverdue},
		{Key: "offenlegung.2025", Kind: domain.ObligationDisclosure, Title: "Offenlegung", DueDate: "2026-12-31"},
		{Key: "erledigt", Title: "Schon erledigt", DueDate: "2026-01-05", Status: domain.ObligationDone},
	}})
	svc.SetBankSource(stubBankSource{transactions: []domain.BankTransaction{
		{ID: 1, Amount: -5000, MatchStatus: domain.MatchStatusUnmatched},
		{ID: 2, Amount: 12000, MatchStatus: domain.MatchStatusMatched},
	}})
	svc.SetReceiptSource(stubReceiptSource{receipts: []domain.Receipt{
		{ID: 1, Direction: domain.DirectionIncoming, Kind: domain.ReceiptKindInvoice,
			Status: domain.ReceiptStatusFiled, ReceivedAt: "2026-03-09", GrossAmount: 11900},
		{ID: 2, Direction: domain.DirectionIncoming, Kind: domain.ReceiptKindInvoice,
			Status: domain.ReceiptStatusFiled, ReceivedAt: "2026-01-05", GrossAmount: 238000},
		// Ein gebuchter Beleg über der Grenze: er zählt nicht mehr als
		// ungebucht, aber sein Leistungsnachweis fehlt trotzdem.
		{ID: 3, Direction: domain.DirectionIncoming, Kind: domain.ReceiptKindInvoice,
			Status: domain.ReceiptStatusSealed, ReceivedAt: "2026-02-01", GrossAmount: 500000,
			JournalEntryID: func() *uint { id := uint(11); return &id }()},
	}})
	svc.SetOpenItemSource(stubOpenItemSource{items: []domain.OpenItem{
		{EntryID: 7, ContactType: domain.ContactTypeCustomer,
			DueDate: "2026-02-01", OpenAmount: 119000},
		{EntryID: 8, ContactType: domain.ContactTypeCustomer,
			DueDate: "2026-04-01", OpenAmount: 50000},
		{EntryID: 9, ContactType: domain.ContactTypeVendor,
			DueDate: "2026-01-01", OpenAmount: 30000},
	}})
	svc.SetBackupSource(stubBackupSource{})

	tasks := tasksFor(t, svc, TaskOptions{Today: today,
		ReadOnlyUntil: "2026-06-30", ReadOnlyReason: "Betriebsprüfung"})

	expect := map[string]domain.TaskGroup{
		"check_findings." + domain.CheckRuleEntryWithoutReceipt: domain.TaskGroupOverdue,
		"check_findings." + domain.CheckRuleVatIDUnconfirmed:    domain.TaskGroupOpen,
		domain.TaskKeyObligation + ".ustva.2026-02":             domain.TaskGroupUpcoming,
		domain.TaskKeyObligation + ".ustva.2026-01":             domain.TaskGroupOverdue,
		domain.TaskKeyBankUnmatched:                             domain.TaskGroupOpen,
		domain.TaskKeyReceiptsUnbooked:                          domain.TaskGroupOpen,
		domain.TaskKeyReceiptsOverdue:                           domain.TaskGroupOverdue,
		domain.TaskKeyOverdueItems:                              domain.TaskGroupOverdue,
		domain.TaskKeyBackupMissing:                             domain.TaskGroupOverdue,
		domain.TaskKeyReadOnly:                                  domain.TaskGroupOpen,
		domain.TaskKeyServiceProof:                              domain.TaskGroupOpen,
	}
	for key, group := range expect {
		task, ok := tasks[key]
		if !ok {
			t.Errorf("die Aufgabe %q fehlt", key)
			continue
		}
		if task.Group != group {
			t.Errorf("%q steht in der Gruppe %q, erwartet %q", key, task.Group, group)
		}
		if task.Title == "" || task.Why == "" {
			t.Errorf("%q hat keinen Titel oder keinen Grund: %+v", key, task)
		}
		if !task.Target.Valid() {
			t.Errorf("%q hat kein gültiges Navigationsziel: %+v", key, task.Target)
		}
		// Vorgangssprache: kein Paragraph im Titel (Architektur 6.4).
		if strings.Contains(task.Title, "§") {
			t.Errorf("der Titel von %q nennt eine Norm: %q", key, task.Title)
		}
	}

	// Was nicht dran ist, steht nicht in der Liste.
	for _, key := range []string{
		domain.TaskKeyObligation + ".erledigt",
		domain.TaskKeyObligation + ".offenlegung.2025",
		"check_findings." + domain.CheckRuleBankUnmatched,
	} {
		if _, ok := tasks[key]; ok {
			t.Errorf("%q darf nicht in der Liste stehen", key)
		}
	}

	// Der Kontext gehört dazu: Anzahl und Betrag.
	if got := tasks[domain.TaskKeyBankUnmatched]; got.Count != 1 || got.Amount != 5000 {
		t.Errorf("Bankumsätze: %d Stück über %s € — erwartet 1 über 50,00", got.Count, got.Amount)
	}
	if got := tasks[domain.TaskKeyServiceProof]; got.Count != 2 {
		t.Errorf("Belege ohne Leistungsnachweis = %d, erwartet 2 (auch der gebuchte)", got.Count)
	}
	// Das Ziel führt auf den Beleg und nicht auf den Zustandsfilter: die Regel
	// trifft gerade auch den gebuchten Beleg, und die Liste „zu buchen" zeigt
	// genau den nicht. Aufgeschlagen wird der älteste ohne Vermerk.
	want := domain.TaskTarget{Area: domain.AreaBooks, Tab: "receipts", ID: "2"}
	if got := tasks[domain.TaskKeyServiceProof]; got.Target != want {
		t.Errorf("Ziel des Leistungsnachweises = %+v, erwartet den Beleg 2", got.Target)
	}
	if got := tasks[domain.TaskKeyReceiptsUnbooked]; got.Count != 2 {
		t.Errorf("ungebuchte Belege = %d, erwartet 2 — der gebuchte zählt nicht mit", got.Count)
	}
	if got := tasks[domain.TaskKeyOverdueItems]; got.Count != 1 || got.Amount != 119000 {
		t.Errorf("überfällige Forderungen: %d über %s € — erwartet 1 über 1.190,00", got.Count, got.Amount)
	}
	if got := tasks["check_findings."+domain.CheckRuleEntryWithoutReceipt]; got.Count != 2 {
		t.Errorf("Befunde je Regel = %d, erwartet 2 zusammengefasste", got.Count)
	}
}

// Die Reihenfolge innerhalb einer Gruppe: die früheste Frist oben, Aufgaben ohne
// Frist danach.
func TestTasksAreSortedByDueDate(t *testing.T) {
	env := newTestEnv(t)
	svc := NewTaskService(repository.NewSettingsRepository(env.db))
	svc.SetObligationSource(stubObligationSource{obligations: []domain.Obligation{
		{Key: "spaet", Title: "Später", DueDate: "2026-02-20", Status: domain.ObligationOverdue},
		{Key: "frueh", Title: "Früher", DueDate: "2026-01-20", Status: domain.ObligationOverdue},
	}})
	svc.SetOpenItemSource(stubOpenItemSource{items: []domain.OpenItem{
		{EntryID: 1, ContactType: domain.ContactTypeCustomer, DueDate: "2026-01-10", OpenAmount: 100},
	}})

	list, err := svc.Tasks(context.Background(), TaskOptions{Today: "2026-03-01"})
	if err != nil {
		t.Fatalf("Aufgabenliste: %v", err)
	}
	if len(list.Overdue) != 3 {
		t.Fatalf("erwartet drei überfällige Aufgaben, erhalten %d", len(list.Overdue))
	}
	if list.Overdue[0].DueDate != "2026-01-10" || list.Overdue[1].DueDate != "2026-01-20" {
		t.Errorf("Reihenfolge: %s, %s, %s — erwartet aufsteigend nach Fälligkeit",
			list.Overdue[0].DueDate, list.Overdue[1].DueDate, list.Overdue[2].DueDate)
	}
}

// Eine Quelle, die ausfällt, nimmt der Liste ihre Zeile — und nicht die Liste.
func TestFailingSourceDoesNotBreakTheTaskList(t *testing.T) {
	env := newTestEnv(t)
	svc := NewTaskService(repository.NewSettingsRepository(env.db))
	svc.SetCheckSource(stubCheckSource{err: errors.New("Prüflauf kaputt")})
	svc.SetBankSource(stubBankSource{transactions: []domain.BankTransaction{
		{ID: 1, Amount: -100, MatchStatus: domain.MatchStatusUnmatched},
	}})

	tasks := tasksFor(t, svc, TaskOptions{Today: "2026-03-10"})
	if _, ok := tasks[domain.TaskKeyBankUnmatched]; !ok {
		t.Error("die übrigen Quellen müssen weiterlaufen, wenn eine ausfällt")
	}
}

// Eine erfolgreiche Sicherung von gestern ist keine Aufgabe; eine von vor einer
// Woche schon.
func TestBackupTaskFollowsTheLastSuccessfulRun(t *testing.T) {
	env := newTestEnv(t)
	settings := repository.NewSettingsRepository(env.db)
	today := time.Now().Format("2006-01-02")

	fresh := NewTaskService(settings)
	fresh.SetBackupSource(stubBackupSource{run: &domain.BackupRun{StartedAt: time.Now()}})
	if _, ok := tasksFor(t, fresh, TaskOptions{Today: today})[domain.TaskKeyBackupMissing]; ok {
		t.Error("nach einer frischen Sicherung darf keine Aufgabe entstehen")
	}

	stale := NewTaskService(settings)
	stale.SetBackupSource(stubBackupSource{run: &domain.BackupRun{
		StartedAt: time.Now().AddDate(0, 0, -7),
	}})
	if _, ok := tasksFor(t, stale, TaskOptions{Today: today})[domain.TaskKeyBackupMissing]; !ok {
		t.Error("eine Woche ohne Sicherung muss auffallen")
	}
}

// --- Die übrigen Quellen der Aufgabenliste --------------------------------
//
// Für jede Quelle beide Seiten: die Zeile entsteht, wenn ihr Zustand es
// verlangt, und sie entsteht nicht, wenn er es nicht verlangt. Eine Prüfung nur
// der Positivseite ließe offen, ob die Quelle überhaupt einen Zustand ansieht.

// Freistellungsbescheinigungen: abgelaufen ist überfällig, ablaufend steht unter
// „demnächst".
func TestExemptionSourceProducesItsTask(t *testing.T) {
	env := newTestEnv(t)
	today := "2026-03-10"

	svc := NewTaskService(repository.NewSettingsRepository(env.db))
	svc.SetExemptionSource(stubExemptionSource{warnings: []ExemptionCertificateWarning{
		{ContactID: 1, Name: "Bau Meier GmbH", ValidUntil: "2026-02-01", State: "expired"},
		{ContactID: 2, Name: "Dach Schulz GmbH", ValidUntil: "2026-03-31", State: "expiring"},
	}})

	tasks := tasksFor(t, svc, TaskOptions{Today: today})
	expired, ok := tasks[domain.TaskKeyExemption+".expired"]
	if !ok {
		t.Fatal("die abgelaufene Bescheinigung erzeugt keine Aufgabe")
	}
	if expired.Group != domain.TaskGroupOverdue || expired.Count != 1 || expired.Target.Tab != "contacts" {
		t.Errorf("abgelaufene Bescheinigung: %+v", expired)
	}
	expiring, ok := tasks[domain.TaskKeyExemption]
	if !ok {
		t.Fatal("die ablaufende Bescheinigung erzeugt keine Aufgabe")
	}
	if expiring.Group != domain.TaskGroupUpcoming || expiring.Count != 1 {
		t.Errorf("ablaufende Bescheinigung: %+v", expiring)
	}

	quiet := NewTaskService(repository.NewSettingsRepository(env.db))
	quiet.SetExemptionSource(stubExemptionSource{})
	for _, key := range []string{domain.TaskKeyExemption, domain.TaskKeyExemption + ".expired"} {
		if _, ok := tasksFor(t, quiet, TaskOptions{Today: today})[key]; ok {
			t.Errorf("%q darf ohne ablaufende Bescheinigung nicht entstehen", key)
		}
	}
}

// Der Saldenvortrag: eine Differenz ist überfällig, ein ausgeglichener und
// gebuchter Vortrag ist keine Aufgabe.
func TestCarryForwardSourceProducesItsTask(t *testing.T) {
	env := newTestEnv(t)
	today := "2026-03-10"
	settings := repository.NewSettingsRepository(env.db)

	unbalanced := NewTaskService(settings)
	unbalanced.SetCarryForwardSource(stubCarryForwardSource{
		preview: &CarryForwardPreview{
			FromYear: 2025, ToYear: 2026, IsBalanced: false, BalanceDifference: 12345,
			Rows: []CarryForwardRow{{Account: "0400"}},
		},
	})
	task, ok := tasksFor(t, unbalanced, TaskOptions{Today: today})[domain.TaskKeyCarryForward+".unbalanced"]
	if !ok {
		t.Fatal("die Vortragsdifferenz erzeugt keine Aufgabe")
	}
	if task.Group != domain.TaskGroupOverdue || task.Amount != 12345 || task.Target.Tab != "year" {
		t.Errorf("Vortragsdifferenz: %+v", task)
	}

	// Ausgeglichen, aber noch nicht gebucht: die Aufgabe „vortragen".
	pending := NewTaskService(settings)
	pending.SetCarryForwardSource(stubCarryForwardSource{
		preview: &CarryForwardPreview{
			FromYear: 2025, ToYear: 2026, IsBalanced: true, AlreadyCarried: false,
			Rows: []CarryForwardRow{{Account: "0400"}, {Account: "1200"}},
		},
	})
	carry, ok := tasksFor(t, pending, TaskOptions{Today: today})[domain.TaskKeyCarryForward]
	if !ok {
		t.Fatal("der ausstehende Saldenvortrag erzeugt keine Aufgabe")
	}
	if carry.Group != domain.TaskGroupOpen || carry.Count != 2 {
		t.Errorf("ausstehender Vortrag: %+v", carry)
	}

	// Ausgeglichen und gebucht: nichts zu tun.
	done := NewTaskService(settings)
	done.SetCarryForwardSource(stubCarryForwardSource{
		preview: &CarryForwardPreview{
			FromYear: 2025, ToYear: 2026, IsBalanced: true, AlreadyCarried: true,
			Rows: []CarryForwardRow{{Account: "0400"}},
		},
	})
	for _, key := range []string{
		domain.TaskKeyCarryForward, domain.TaskKeyCarryForward + ".unbalanced",
		domain.TaskKeyCarryForward + ".correction",
	} {
		if _, ok := tasksFor(t, done, TaskOptions{Today: today})[key]; ok {
			t.Errorf("%q darf beim ausgeglichenen und gebuchten Vortrag nicht entstehen", key)
		}
	}
}

// Ein fehlgeschlagener jüngster Lauf fällt sofort auf — auch wenn die letzte
// erfolgreiche Sicherung noch innerhalb der Frist liegt.
func TestFailedBackupRunIsATaskOfItsOwn(t *testing.T) {
	env := newTestEnv(t)
	settings := repository.NewSettingsRepository(env.db)
	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1)

	failed := NewTaskService(settings)
	failed.SetBackupSource(stubBackupSource{
		run: &domain.BackupRun{StartedAt: yesterday, Success: true},
		runs: []domain.BackupRun{{
			Kind: domain.BackupKindAutomatic, StartedAt: time.Now(), Success: false,
			Message: "Zielordner nicht beschreibbar",
		}},
	})
	tasks := tasksFor(t, failed, TaskOptions{Today: today})
	task, ok := tasks[domain.TaskKeyBackupFailed]
	if !ok {
		t.Fatal("ein fehlgeschlagener Lauf muss eine Aufgabe erzeugen")
	}
	if task.Group != domain.TaskGroupOverdue {
		t.Errorf("der Fehlschlag steht in der Gruppe %q, erwartet überfällig", task.Group)
	}
	if !strings.Contains(task.Why, "Zielordner nicht beschreibbar") {
		t.Errorf("der Fehlertext fehlt im Grund: %q", task.Why)
	}
	// Die Frist ist noch nicht abgelaufen: ohne diese Regel entstünde gar nichts.
	if _, ok := tasks[domain.TaskKeyBackupMissing]; ok {
		t.Error("nach einer frischen erfolgreichen Sicherung darf keine Fristaufgabe entstehen")
	}

	// Ein gelungener jüngster Lauf ist keine Aufgabe.
	fine := NewTaskService(settings)
	fine.SetBackupSource(stubBackupSource{
		run:  &domain.BackupRun{StartedAt: time.Now(), Success: true},
		runs: []domain.BackupRun{{Kind: domain.BackupKindAutomatic, StartedAt: time.Now(), Success: true}},
	})
	if _, ok := tasksFor(t, fine, TaskOptions{Today: today})[domain.TaskKeyBackupFailed]; ok {
		t.Error("nach einem gelungenen Lauf darf keine Aufgabe entstehen")
	}
}

// Im Januar stehen die Pflichten des Vorjahres auf der Liste, soweit sie in den
// letzten dreißig Tagen fällig wurden oder noch fällig werden: die
// Voranmeldung für Dezember am 10. Januar, nicht der ganze Rückstand.
func TestTasksIncludeLastYearsDeadlinesUntilTheyExpire(t *testing.T) {
	env := newTestEnv(t)
	svc := NewTaskService(repository.NewSettingsRepository(env.db))
	svc.SetBankSource(stubBankSource{})
	svc.SetReceiptSource(stubReceiptSource{})
	svc.SetOpenItemSource(stubOpenItemSource{})
	svc.SetObligationSource(stubObligationSource{obligations: []domain.Obligation{
		{Key: "ustva.2025-12", Kind: domain.ObligationVatReturn, Title: "Umsatzsteuer-Voranmeldung Dezember 2025",
			DueDate: "2026-01-10", Description: "Die Voranmeldung ist abzugeben.", Status: domain.ObligationOpen},
		// Weit verstrichen: das gehört in den Kalender und nicht auf den ersten
		// Bildschirm.
		{Key: "ustva.2025-03", Kind: domain.ObligationVatReturn, Title: "Umsatzsteuer-Voranmeldung März 2025",
			DueDate: "2025-04-10", Status: domain.ObligationOverdue},
		{Key: "zm.2026-Q1", Kind: domain.ObligationECSalesList, Title: "Zusammenfassende Meldung 1. Quartal 2026",
			DueDate: "2026-01-25", Status: domain.ObligationOpen},
	}})

	tasks := tasksFor(t, svc, TaskOptions{Today: "2026-01-05"})
	december, ok := tasks[domain.TaskKeyObligation+".ustva.2025-12"]
	if !ok {
		t.Fatalf("die Voranmeldung Dezember 2025 fehlt am 05.01.2026: %v", keysOf(tasks))
	}
	if december.Group != domain.TaskGroupUpcoming || december.DueDate != "2026-01-10" {
		t.Errorf("Dezember-Voranmeldung: Gruppe %q, fällig %s — erwartet demnächst zum 10.01.2026",
			december.Group, december.DueDate)
	}
	if _, ok := tasks[domain.TaskKeyObligation+".ustva.2025-03"]; ok {
		t.Error("eine seit Monaten verstrichene Frist des Vorjahres gehört nicht auf die Aufgabenliste")
	}
	if _, ok := tasks[domain.TaskKeyObligation+".zm.2026-Q1"]; !ok {
		t.Errorf("die Frist des laufenden Jahres fehlt: %v", keysOf(tasks))
	}
}

// keysOf nennt die Schlüssel einer Aufgabenliste für die Fehlermeldung.
func keysOf(tasks map[string]domain.Task) []string {
	keys := make([]string, 0, len(tasks))
	for key := range tasks {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
