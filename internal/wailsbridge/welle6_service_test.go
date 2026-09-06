package wailsbridge

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/timestamp"
)

// wiredBridge baut einen vollständig verdrahteten Mandanten auf einem
// Wegwerf-Datenordner. Anders als testBridge geht sie über initTenant: die
// Festschreibung braucht Prüfdienst, Belegablage und Festschreibungskartei, und
// die hängen aneinander.
func wiredBridge(t *testing.T) *BuchfinkBridge {
	t.Helper()
	dataDir := t.TempDir()
	tenant := &domain.TenantConfig{
		ID: "tenant_welle6", Name: "Pfennig Ventures GmbH", DataDir: dataDir,
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	b := &BuchfinkBridge{
		currentYear: 2026,
		appCfgRepo:  repository.NewAppConfigRepository(filepath.Join(t.TempDir(), "config")),
		appConfig: domain.AppConfig{
			Tenants: []domain.TenantConfig{*tenant}, ActiveTenantID: tenant.ID,
			DataDir: dataDir, IsConfigured: true, LastFiscalYear: 2026,
		},
		dataDir: dataDir,
	}
	if err := b.initTenant(tenant); err != nil {
		t.Fatalf("Mandant einrichten: %v", err)
	}
	return b
}

// postSimpleEntry schreibt eine ausgeglichene Buchung über den Journaldienst.
func postSimpleEntry(t *testing.T, b *BuchfinkBridge, bookingDate, description string) *domain.JournalEntry {
	t.Helper()
	entry := &domain.JournalEntry{
		FiscalYear: 2026, BookingDate: bookingDate, DocumentDate: bookingDate,
		ServiceDateFrom: bookingDate, ServiceDateTo: bookingDate,
		Description: description, Source: domain.EntrySourceManual, TaxTreatment: domain.TaxTreatmentNotTaxable,
		Kind: domain.EntryKindNormal, Currency: "EUR", ExchangeRateMicros: 1_000_000,
		Lines: []domain.JournalLine{
			{Position: 1, Side: domain.SideDebit, Account: "6815", Amount: 10000},
			{Position: 2, Side: domain.SideCredit, Account: "1800", Amount: 10000},
		},
	}
	created, err := b.journalSvc.Post(context.Background(), entry)
	if err != nil {
		t.Fatalf("Buchung %q: %v", description, err)
	}
	return created
}

// fakeTSA setzt für die Dauer eines Tests eine Zeitstempelstelle ein, die eine
// feste beglaubigte Zeit liefert.
//
// Ohne diesen Austausch ließe sich der Abgleich der Systemzeit mit der
// beglaubigten Zeit nicht prüfen: der Test müsste entweder die Uhr des Rechners
// verstellen oder eine Stelle im Netz erreichen.
func fakeTSA(t *testing.T, genTime time.Time, name string) {
	t.Helper()
	previous := requestTimestamp
	requestTimestamp = func(ctx context.Context, tsaURL, sha256Hex string) (*timestamp.Result, error) {
		return &timestamp.Result{
			Token:   []byte("test-token"),
			GenTime: genTime,
			TSAName: name,
		}, nil
	}
	t.Cleanup(func() { requestTimestamp = previous })
}

// Weicht die beglaubigte Zeit über die Toleranz von der Systemzeit ab, entsteht
// ein eigener Protokolleintrag mit beiden Zeiten, und die Festschreibung trägt
// den Hinweis.
//
// Der Hinweis allein genügt nicht: er steht dort, wo ohnehin schon jemand
// hinsieht. Als eigener Vorgang im Protokoll fällt die Abweichung auf.
func TestCommitPeriodLogsTimeDriftBeyondTolerance(t *testing.T) {
	b := wiredBridge(t)
	ctx := context.Background()

	postSimpleEntry(t, b, "2026-01-15", "Wareneingang Januar")

	// Neunzig Minuten Rückstand der Systemzeit gegenüber der beglaubigten Zeit:
	// weit über der Toleranz von fünf Minuten.
	fakeTSA(t, time.Now().UTC().Add(90*time.Minute), "Test-TSA")

	rec, err := b.CommitPeriod("month", "Januar 2026", "2026-01-31",
		"Testlauf ohne Belege — die Belegprüfung wird bewusst übergangen")
	if err != nil {
		t.Fatalf("Festschreibung: %v", err)
	}
	if rec.TimeDriftNote == "" {
		t.Fatal("die Festschreibung trägt keinen Hinweis auf die Zeitabweichung")
	}
	if !strings.Contains(rec.TimeDriftNote, "geht nach") {
		t.Errorf("der Hinweis muss die Richtung nennen: %q", rec.TimeDriftNote)
	}

	entries, err := b.auditRepo.FindFiltered(ctx, 0, domain.AuditFilter{EntityType: "TIME_DRIFT"})
	if err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("erwartet ein Protokolleintrag zur Zeitabweichung, gefunden %d", len(entries))
	}
	details := entries[0].Details
	// Beide Zeiten stehen im Eintrag: ohne sie ließe sich die Abweichung später
	// nicht mehr nachrechnen.
	if !strings.Contains(details, "Systemzeit") || !strings.Contains(details, "beglaubigte Zeit") {
		t.Errorf("der Eintrag muss beide Zeiten nennen: %q", details)
	}
	if rec.TSAGenTime == nil ||
		!strings.Contains(details, rec.TSAGenTime.UTC().Format("2006-01-02T15:04")) {
		t.Errorf("die beglaubigte Zeit fehlt im Eintrag: %q", details)
	}
}

// Liegt die beglaubigte Zeit innerhalb der Toleranz, entsteht kein Eintrag. Eine
// Meldung über jede Sekunde Abweichung wäre Rauschen und niemand läse sie.
func TestCommitPeriodStaysSilentWithinTolerance(t *testing.T) {
	b := wiredBridge(t)
	postSimpleEntry(t, b, "2026-01-15", "Wareneingang Januar")
	fakeTSA(t, time.Now().UTC().Add(time.Minute), "Test-TSA")

	rec, err := b.CommitPeriod("month", "Januar 2026", "2026-01-31", "Testlauf ohne Belege")
	if err != nil {
		t.Fatalf("Festschreibung: %v", err)
	}
	if rec.TimeDriftNote != "" {
		t.Errorf("eine Minute liegt in der Toleranz, bekommen: %q", rec.TimeDriftNote)
	}
	entries, _ := b.auditRepo.FindFiltered(context.Background(), 0,
		domain.AuditFilter{EntityType: "TIME_DRIFT"})
	if len(entries) != 0 {
		t.Errorf("ohne Abweichung darf kein Eintrag entstehen, gefunden %d", len(entries))
	}
}

// Nach der Festschreibung tragen alle Buchungen bis zum Stichtag ihren
// Festschreibungszeitpunkt und den Verweis auf die Festschreibung; die
// Buchungen danach nicht.
func TestCommitPeriodStampsEntriesUpToTheCutoff(t *testing.T) {
	b := wiredBridge(t)
	ctx := context.Background()

	inPeriod := postSimpleEntry(t, b, "2026-01-15", "Wareneingang Januar")
	afterPeriod := postSimpleEntry(t, b, "2026-02-03", "Wareneingang Februar")

	fakeTSA(t, time.Now().UTC(), "Test-TSA")
	rec, err := b.CommitPeriod("month", "Januar 2026", "2026-01-31", "Testlauf ohne Belege")
	if err != nil {
		t.Fatalf("Festschreibung: %v", err)
	}
	if rec.EntriesStamped != 1 {
		t.Errorf("gestempelt wurden %d Buchungen, erwartet 1", rec.EntriesStamped)
	}

	stamped, err := b.journalRepo.FindByID(ctx, inPeriod.ID)
	if err != nil {
		t.Fatalf("Buchung lesen: %v", err)
	}
	if stamped.CommittedAt == nil {
		t.Error("die Buchung bis zum Stichtag trägt keinen Festschreibungszeitpunkt")
	}
	if stamped.FestschreibungID == nil || *stamped.FestschreibungID != rec.ID {
		t.Errorf("die Buchung verweist nicht auf die Festschreibung %d: %v",
			rec.ID, stamped.FestschreibungID)
	}

	open, err := b.journalRepo.FindByID(ctx, afterPeriod.ID)
	if err != nil {
		t.Fatalf("Buchung lesen: %v", err)
	}
	if open.CommittedAt != nil || open.FestschreibungID != nil {
		t.Error("eine Buchung nach dem Stichtag darf nicht festgeschrieben sein")
	}
}

// Der Sicherungsordner ist eine Einstellung des Mandanten: sein Wechsel gehört
// mit beiden Ständen ins Protokoll.
func TestSetBackupDirLogsBeforeAndAfter(t *testing.T) {
	b := wiredBridge(t)
	ctx := context.Background()

	first := t.TempDir()
	second := t.TempDir()
	if _, err := b.SetBackupDir(first); err != nil {
		t.Fatalf("Sicherungsordner setzen: %v", err)
	}
	if _, err := b.SetBackupDir(second); err != nil {
		t.Fatalf("Sicherungsordner ändern: %v", err)
	}

	entries, err := b.auditRepo.FindFiltered(ctx, 0, domain.AuditFilter{EntityType: "BACKUP_DIR"})
	if err != nil || len(entries) == 0 {
		t.Fatalf("Protokoll lesen: %v (%d Einträge)", err, len(entries))
	}
	before := auditFields(t, entries[0].Before)
	after := auditFields(t, entries[0].After)
	firstAbs, _ := filepath.Abs(first)
	secondAbs, _ := filepath.Abs(second)
	if before["backupDir"] != firstAbs {
		t.Errorf("das Vorher trägt den alten Ordner nicht: %v", before)
	}
	if after["backupDir"] != secondAbs {
		t.Errorf("das Nachher trägt den neuen Ordner nicht: %v", after)
	}
}

// Der Prüfermodus schaltet die Schreibwege ab. Wann er galt und wie lange, ist
// Teil des Nachweises über den Zeitraum, in dem der Prüfer gesehen hat, was er
// gesehen hat.
func TestReadOnlyModeLogsBeforeAndAfter(t *testing.T) {
	b := wiredBridge(t)
	ctx := context.Background()

	until := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	if _, err := b.EnableReadOnly(until, "Außenprüfung 2020 bis 2023"); err != nil {
		t.Fatalf("Prüfermodus einschalten: %v", err)
	}
	entries, err := b.auditRepo.FindFiltered(ctx, 0, domain.AuditFilter{EntityType: "READ_ONLY"})
	if err != nil || len(entries) == 0 {
		t.Fatalf("Protokoll lesen: %v (%d Einträge)", err, len(entries))
	}
	after := auditFields(t, entries[0].After)
	if after["readOnlyUntil"] != until {
		t.Errorf("das Nachher trägt das Enddatum nicht: %v", after)
	}
	if after["readOnlyReason"] != "Außenprüfung 2020 bis 2023" {
		t.Errorf("das Nachher trägt den Grund nicht: %v", after)
	}

	if _, err := b.DisableReadOnly("Prüfung abgeschlossen"); err != nil {
		t.Fatalf("Prüfermodus beenden: %v", err)
	}
	entries, _ = b.auditRepo.FindFiltered(ctx, 0, domain.AuditFilter{EntityType: "READ_ONLY"})
	before := auditFields(t, entries[0].Before)
	if before["readOnlyUntil"] != until {
		t.Errorf("das Vorher des Abschaltens trägt das Enddatum nicht: %v", before)
	}
}

// Das Prüferpaket der Bridge trägt die Fassungen der Verfahrensdokumentation:
// die Verdrahtung muss sie an den Export reichen.
func TestBridgeAuditPackageCarriesTheGeneratedProcedureDocumentation(t *testing.T) {
	b := wiredBridge(t)

	result, err := b.GenerateProcedureDocumentation()
	if err != nil {
		t.Fatalf("Verfahrensdokumentation erzeugen: %v", err)
	}

	dir := filepath.Join(t.TempDir(), "pruefer")
	if _, err := b.ExportAuditPackage(2026, dir); err != nil {
		t.Fatalf("Prüferpaket: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "verfahrensdokumentation", result.Document.FileName)); err != nil {
		t.Errorf("die erzeugte Verfahrensdokumentation liegt dem Paket nicht bei: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "CHANGELOG.md")); err != nil {
		t.Errorf("die Versionshistorie liegt dem Paket nicht bei: %v", err)
	}
}

// auditFields liest das Vorher oder Nachher eines Protokolleintrags.
func auditFields(t *testing.T, raw string) map[string]any {
	t.Helper()
	if raw == "" {
		return map[string]any{}
	}
	out := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("Vorher/Nachher ist kein JSON: %v (%s)", err, raw)
	}
	return out
}
