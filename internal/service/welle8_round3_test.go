package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// Das eigene Konto ist sofort bebuchbar, und die Sperre wirkt sofort
// (BEL-06 K2, Entscheidung 4).
//
// Der Buchungsweg hält den Kontenplan zwischengespeichert. Ohne die
// Invalidierung wäre das neue Konto für ihn „im SKR04 nicht vorhanden" und ein
// gesperrtes weiter bebuchbar — beides erst nach einem Neustart richtig, und
// das ist keine Kontenpflege.
func TestCustomAccountIsPostableRightAfterItIsCreated(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Eine erste Buchung füllt den Zwischenspeicher des Kontenplans. Ohne sie
	// prüfte der Test einen Kontenplan, der ohnehin frisch gelesen wird.
	if _, err := env.journal.Post(ctx, simpleEntry("6300", domain.AccountKasse, 1000)); err != nil {
		t.Fatalf("erste Buchung: %v", err)
	}

	accounts := NewAccountService(
		repository.NewAccountRepository(env.db), repository.NewAuditRepository(env.db))
	accounts.SetChartInvalidator(env.journal)

	const position = "guv.guv_8.sonstige_betriebliche_aufwendungen"
	if _, err := accounts.CreateCustom(ctx, CustomAccountRequest{
		Number: "6011", Name: "Fremdleistungen Fotografie", HGBPosition: position,
	}); err != nil {
		t.Fatalf("eigenes Konto: %v", err)
	}

	if _, err := env.journal.Post(ctx, simpleEntry("6011", domain.AccountKasse, 2500)); err != nil {
		t.Fatalf("auf das eigene Konto muss sofort gebucht werden können: %v", err)
	}

	if _, err := accounts.SetBlocked(ctx, "6011", true, "Aufwandsart entfällt"); err != nil {
		t.Fatalf("Konto sperren: %v", err)
	}
	blocked := simpleEntry("6011", domain.AccountKasse, 2500)
	blocked.Description = "Buchung nach der Sperre"
	if _, err := env.journal.Post(ctx, blocked); err == nil {
		t.Error("ein gesperrtes Konto nimmt keine Buchung mehr auf")
	}

	// Und die Freigabe wirkt ebenso sofort: eine Sperre, die sich nicht
	// zurücknehmen ließe, wäre eine Löschung.
	if _, err := accounts.SetBlocked(ctx, "6011", false, ""); err != nil {
		t.Fatalf("Konto freigeben: %v", err)
	}
	again := simpleEntry("6011", domain.AccountKasse, 2500)
	again.Description = "Buchung nach der Freigabe"
	if _, err := env.journal.Post(ctx, again); err != nil {
		t.Errorf("nach der Freigabe ist das Konto wieder bebuchbar: %v", err)
	}
}

// Die Abstimmung der ZM stellt die sonstigen Leistungen bei monatlicher Meldung
// dem ganzen Kalendervierteljahr gegenüber (UST-04 K3).
//
// Gemeldet werden sie im letzten Quartalsmonat, vorangemeldet im Monat der
// Leistung. Verglichen man Monat gegen Monat, meldete die Abstimmung dieselbe
// Januarleistung zweimal als Abweichung.
func TestZMReconciliationFollowsTheServiceQuarter(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Client SARL", "FR", "FR12345678901")

	// Lieferungen über der Grenze des § 18a Abs. 1 Satz 2 UStG: gemeldet wird
	// monatlich.
	env.euInvoice(t, customer.ID, "2026-01-15", 6_000_000, domain.TaxTreatmentIntraCommunitySupply)
	// Die sonstige Leistung aus dem Januar. Sie gehört in die Märzmeldung.
	env.euInvoice(t, customer.ID, "2026-01-20", 500_000, domain.TaxTreatmentReverseChargeSupply)

	// Die Voranmeldungen laufen monatlich; die Januarleistung steht in der
	// Kennziffer 21 des Januars.
	for _, key := range []string{"2026-01", "2026-02", "2026-03"} {
		if _, err := env.vatReturns(t).Save(ctx, key); err != nil {
			t.Fatalf("Voranmeldung %s: %v", key, err)
		}
	}

	differenceOf := func(periodKey string) domain.Cents {
		t.Helper()
		zm, err := env.zmReturns(t).Draft(ctx, periodKey)
		if err != nil {
			t.Fatalf("ZM-Entwurf %s: %v", periodKey, err)
		}
		if zm.Reconciliation == nil {
			t.Fatalf("der Meldung %s fehlt die Abstimmung", periodKey)
		}
		return zm.Reconciliation.ServicesDifference()
	}

	if diff := differenceOf("2026-01"); diff != 0 {
		t.Errorf("Januar: Abweichung %s bei den sonstigen Leistungen — sie gehören erst in die "+
			"Märzmeldung und dürfen im Januar keine Abweichung sein", diff)
	}
	if diff := differenceOf("2026-03"); diff != 0 {
		t.Errorf("März: Abweichung %s bei den sonstigen Leistungen — die Meldung des "+
			"Quartalsendmonats steht der Kennziffer 21 des ganzen Quartals gegenüber", diff)
	}
}

// Die Verfahrensdokumentation nennt die tatsächlich eingestellte Systematik der
// Belegnummern (BEL-02 K4).
func TestProcDocShowsTheConfiguredReceiptNumberFormat(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	settingsRepo := repository.NewSettingsRepository(env.db)
	cfg, err := settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Unternehmensdaten: %v", err)
	}
	cfg.ReceiptNumberFormat = "BEL-{JAHR}-{NR:5}"
	if err := settingsRepo.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatalf("Belegnummernformat setzen: %v", err)
	}

	result, err := newProcDocService(t, env).
		Generate(ctx, time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Verfahrensdokumentation: %v", err)
	}
	if !strings.Contains(result.Markdown, "BEL-{JAHR}-{NR:5}") {
		t.Error("die Nummernkreistabelle nennt die eingestellte Systematik der Belegnummern nicht")
	}
	if strings.Contains(result.Markdown, "ER-{JAHR}-{NR:4}") {
		t.Error("die Verfahrensdokumentation beschreibt eine Systematik, die das Programm nicht " +
			"vergibt")
	}
}

// „Zugriffe" ist eine Abfrage und keine zwei: die Herausgabe von Daten steht als
// EXPORT, der Prüfermodus als Änderung an der Entität READ_ONLY (QUE-02 K2).
func TestAuditAccessFilterFindsBothKindsOfAccess(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	auditRepo := repository.NewAuditRepository(env.db)

	if err := auditRepo.Log(ctx, domain.AuditActionExport, "RECEIPT", "1",
		"Belegdatei herausgegeben"); err != nil {
		t.Fatalf("Protokolleintrag Export: %v", err)
	}
	if err := auditRepo.Log(ctx, domain.AuditActionUpdate, domain.AuditEntityReadOnly,
		"tenant", "Prüfermodus eingeschaltet"); err != nil {
		t.Fatalf("Protokolleintrag Prüfermodus: %v", err)
	}
	if err := auditRepo.Log(ctx, domain.AuditActionUpdate, "CONTACT", "7",
		"Kontakt geändert"); err != nil {
		t.Fatalf("Protokolleintrag Kontakt: %v", err)
	}

	entries, err := auditRepo.FindFiltered(ctx, 0, domain.AuditFilter{Access: true})
	if err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("%d Einträge unter den Zugriffen — erwartet die Herausgabe und den "+
			"Prüfermodus: %+v", len(entries), entries)
	}
	for _, e := range entries {
		if e.EntityType == "CONTACT" {
			t.Error("eine Stammdatenänderung ist kein Zugriff auf personenbezogene Daten")
		}
	}
}

// Scheitert die Buchung erst beim Schreiben, bleibt weder der Eigenbeleg noch
// seine Datei zurück (Entscheidung 1).
//
// Der Fall, den der frühere Test nicht traf: dort scheiterte schon die
// Vorprüfung, und der Beleg entstand gar nicht erst. Hier läuft die Anlage
// durch, und erst das Schreiben der Buchung schlägt fehl.
func TestFailedJournalWriteRollsBackTheSelfIssuedReceipt(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.withSelfIssued(t)
	// Wie in der Anwendung: Beleg und Buchung entstehen in einer Transaktion.
	env.posting.SetTxRunner(repository.NewTxRunner(env.db))

	// Der Journalspeicher weist das Schreiben ab. Alles davor — Vorprüfung,
	// Anlage des Eigenbelegs — läuft durch.
	failing := NewJournalService(
		failingJournalRepo{JournalRepository: env.journalRepo},
		repository.NewAccountRepository(env.db),
		env.contactRepo,
		repository.NewAuditRepository(env.db),
		repository.NewSettingsRepository(env.db),
		env.fiscalYear,
	)
	failing.SetReceiptRepo(env.receiptRepo)
	posting := NewPostingService(failing, env.contactRepo)
	posting.SetReceiptService(env.receipts)
	posting.SetTxRunner(repository.NewTxRunner(env.db))

	if _, err := posting.PostManualEntry(ctx, ManualEntryRequest{
		Entry:      *manualEntry(),
		SelfIssued: &SelfIssuedReceiptRequest{Reason: "Parkgebühr"},
	}); err == nil {
		t.Fatal("scheitert das Schreiben der Buchung, darf kein Vorgang entstehen")
	}

	receipts, err := env.receipts.List(ctx, "")
	if err != nil {
		t.Fatalf("Belege lesen: %v", err)
	}
	if len(receipts) != 0 {
		t.Errorf("%d Belege abgelegt — die gescheiterte Buchung darf keinen hinterlassen",
			len(receipts))
	}
	if files := storedFiles(t, env.dataDir); len(files) != 0 {
		t.Errorf("%d Dateien im Belegspeicher — das PDF des zurückgerollten Eigenbelegs bleibt "+
			"sonst als verwaiste Datei liegen: %v", len(files), files)
	}
}

// failingJournalRepo weist das Schreiben einer Buchung ab und reicht alles
// andere durch.
type failingJournalRepo struct {
	domain.JournalRepository
}

func (r failingJournalRepo) Append(
	ctx context.Context, entry *domain.JournalEntry, hash domain.EntryHashFunc,
) error {
	return errWriteRefused
}

var errWriteRefused = errors.New("das Journal nimmt die Buchung nicht an")

// storedFiles listet die Dateien des Belegspeichers unter dem Datenordner.
func storedFiles(t *testing.T, dataDir string) []string {
	t.Helper()
	out := make([]string, 0)
	err := filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			out = append(out, strings.TrimPrefix(path, dataDir))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Belegspeicher lesen: %v", err)
	}
	return out
}

// Die Festschreibung eines Monats steht einmal in der Aufgabenliste: als
// Fristzeile mit ihrem Monat und nicht zusätzlich als Regelzeile des Prüflaufs.
func TestCommitTaskAppearsOnlyOnce(t *testing.T) {
	env := newTestEnv(t)
	svc := NewTaskService(repository.NewSettingsRepository(env.db), 2026)
	svc.SetDeadlineSource(stubDeadlineSource{deadlines: []domain.Deadline{{
		Key: DeadlineKeyCommit + ".2026-01", Title: "Januar 2026 festschreiben",
		DueDate: "2026-02-28", Reference: "GoBD Rz. 107",
		Description: "Nach der Festschreibung nimmt der Monat keine Buchung mehr auf.",
	}}})
	svc.SetCheckSource(stubCheckSource{run: &domain.CheckRun{Findings: []domain.CheckFinding{{
		Rule: domain.CheckRulePeriodNotCommitted, Severity: domain.CheckWarning,
		ObjectID: "2026-01", ObjectName: "Januar 2026",
		Message: "Januar 2026 ist seit dem 2026-03-31 nicht festgeschrieben",
	}}}})

	list, err := svc.Tasks(context.Background(), TaskOptions{Today: "2026-04-01"})
	if err != nil {
		t.Fatalf("Aufgabenliste: %v", err)
	}
	commit := 0
	for _, group := range [][]domain.Task{list.Overdue, list.Open, list.Upcoming} {
		for _, task := range group {
			if strings.Contains(task.Title, "festschreiben") {
				commit++
			}
		}
	}
	if commit != 1 {
		t.Errorf("%d Zeilen zur Festschreibung des Januars — dieselbe Arbeit steht einmal in der "+
			"Liste: %+v", commit, list)
	}
}
