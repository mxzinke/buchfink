package service

import (
	"archive/zip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/receiptstore"
	"github.com/buchfink/buchfink/internal/repository"
	"gorm.io/gorm"
)

// backupEnv ist ein Mandant auf der Platte: eine echte Datenbankdatei und ein
// Belegordner. Die Sicherung lässt sich nicht gegen eine Datenbank im
// Arbeitsspeicher prüfen — sie kopiert Dateien.
type backupEnv struct {
	dataDir string
	db      *gorm.DB
	journal *JournalService
	backups *BackupService
}

func newBackupEnv(t *testing.T) *backupEnv {
	t.Helper()
	dataDir := t.TempDir()

	db, err := repository.InitTenantDB(dataDir)
	if err != nil {
		t.Fatalf("Mandantendatenbank anlegen: %v", err)
	}

	settingsRepo := repository.NewSettingsRepository(db)
	if err := settingsRepo.UpdateCompanySettings(context.Background(), &domain.CompanySettings{
		CompanyName: "Pfennig Ventures GmbH", LegalForm: "GmbH",
		FiscalYear: 2026, FiscalYearStartMonth: 1, Currency: "EUR", SKR: "SKR04",
		VatPeriod: "quarter", TaxationType: "SOLL",
	}); err != nil {
		t.Fatalf("Unternehmensdaten setzen: %v", err)
	}

	journalRepo := repository.NewJournalRepository(db)
	journal := newJournalWithTaxRules(
		journalRepo, repository.NewAccountRepository(db), repository.NewContactRepository(db),
		repository.NewAuditRepository(db), settingsRepo, repository.NewTxRunner(db),
	)

	backups := NewBackupService(
		repository.NewBackupRunRepository(db), repository.NewAuditRepository(db),
		db, dataDir, "tenant_test", "Pfennig Ventures GmbH")

	return &backupEnv{
		dataDir: dataDir,
		db:      db,
		journal: journal,
		backups: backups,
	}
}

// book legt eine Buchung an, damit die Kette etwas zu sichern hat.
func (e *backupEnv) book(t *testing.T, amount domain.Cents) *domain.JournalEntry {
	t.Helper()
	entry, err := e.journal.Post(context.Background(), simpleEntry("6815", "1800", amount))
	if err != nil {
		t.Fatalf("Buchung: %v", err)
	}
	return entry
}

// receiptFile legt eine Belegdatei im Datenordner ab und trägt sie ein.
func (e *backupEnv) receiptFile(t *testing.T, name, content string) *domain.Receipt {
	t.Helper()
	store := receiptstore.New(e.dataDir)
	svc := NewReceiptService(
		repository.NewReceiptRepository(e.db), store, repository.NewAuditRepository(e.db),
	)
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("Testdatei schreiben: %v", err)
	}
	receipt, err := svc.File(context.Background(), FileReceiptRequest{
		// Die Kopfdaten gehören zu jedem buchbaren Beleg
		// (BEL-02): ohne Belegdatum, Aussteller und Betrag weist
		// ValidateBookable die Buchung zurück.
		DocumentDate: "2026-03-01", IssuerName: "Lieferant GmbH", GrossAmount: 11900,
		Direction: domain.DirectionIncoming,
		Files:     []NewFile{{Role: domain.ReceiptRoleOriginal, Path: path}},
	})
	if err != nil {
		t.Fatalf("Beleg ablegen: %v", err)
	}
	return receipt
}

// --- Sichern ---------------------------------------------------------------

// Die Sicherung muss alles enthalten, was einen Mandanten ausmacht: die
// Datenbank, die Belege — und backup.json mit den Prüfsummen. Fehlt eines
// davon, ist der Bestand im Ernstfall nicht wiederherstellbar.
func TestCreateBackupContainsEverything(t *testing.T) {
	env := newBackupEnv(t)
	env.book(t, 11900)
	env.receiptFile(t, "rechnung.pdf", "%PDF-1.4 Rechnung")

	target := t.TempDir()
	run, err := env.backups.CreateBackup(context.Background(), target, domain.BackupKindManual)
	if err != nil {
		t.Fatalf("Sicherung: %v", err)
	}
	if !run.Success {
		t.Fatalf("die Sicherung ist fehlgeschlagen: %s", run.Message)
	}
	if !strings.HasSuffix(run.Target, ".zip") || !strings.Contains(run.Target, "pfennig") {
		t.Errorf("der Dateiname der Sicherung nennt den Mandanten nicht: %s", run.Target)
	}

	names := zipEntries(t, run.Target)
	for _, want := range []string{"buchfink.sqlite", BackupMetaFileName} {
		if !names[want] {
			t.Errorf("die Sicherung enthält %s nicht", want)
		}
	}
	belege := 0
	for name := range names {
		if strings.HasPrefix(name, "belege/") {
			belege++
		}
	}
	if belege == 0 {
		t.Error("die Sicherung enthält keine Belegdatei")
	}

	meta := readBackupMeta(t, run.Target)
	if meta.FileCount != run.FileCount || meta.FileCount == 0 {
		t.Errorf("backup.json zählt %d Dateien, der Lauf %d", meta.FileCount, run.FileCount)
	}
	for _, f := range meta.Files {
		if len(f.SHA256) != 64 {
			t.Errorf("%s: keine brauchbare Prüfsumme in backup.json", f.Path)
		}
	}
	if meta.ProgramVersion == "" {
		t.Error("backup.json nennt die Programmversion nicht")
	}
}

// Ein Sicherungslauf wird auch dann festgehalten, wenn er scheitert. Eine
// fehlgeschlagene Sicherung, von der niemand erfährt, wiegt in Sicherheit.
func TestFailedBackupIsRecorded(t *testing.T) {
	env := newBackupEnv(t)
	run, err := env.backups.CreateBackup(context.Background(), "", domain.BackupKindManual)
	if err == nil {
		t.Fatal("eine Sicherung ohne Zielordner muss scheitern")
	}
	if run.Success {
		t.Error("der Lauf ist als gelungen vermerkt, obwohl er scheiterte")
	}

	runs, err := env.backups.GetRuns(context.Background(), 10)
	if err != nil {
		t.Fatalf("Läufe lesen: %v", err)
	}
	if len(runs) != 1 || runs[0].Success {
		t.Fatalf("erwartet einen festgehaltenen Fehlschlag, erhalten %d Läufe", len(runs))
	}
	if _, err := env.backups.LastSuccessful(context.Background()); err != nil {
		t.Fatalf("letzte gelungene Sicherung: %v", err)
	}
	last, _ := env.backups.LastSuccessful(context.Background())
	if last != nil {
		t.Error("ein Fehlschlag gilt als letzte gelungene Sicherung — das wäre eine falsche Zusage")
	}
}

// Die automatische Sicherung ist fällig, solange es keine gelungene gibt, und
// wieder nach einem Tag.
func TestBackupIsDueUntilOneSucceeded(t *testing.T) {
	env := newBackupEnv(t)
	ctx := context.Background()

	if !env.backups.IsDue(ctx, time.Now()) {
		t.Error("ohne jede Sicherung muss eine fällig sein")
	}
	if _, err := env.backups.CreateBackup(ctx, t.TempDir(), domain.BackupKindAutomatic); err != nil {
		t.Fatalf("Sicherung: %v", err)
	}
	if env.backups.IsDue(ctx, time.Now()) {
		t.Error("unmittelbar nach einer Sicherung ist keine fällig")
	}
	if !env.backups.IsDue(ctx, time.Now().Add(25*time.Hour)) {
		t.Error("nach mehr als einem Tag ist wieder eine fällig")
	}
}

// --- Wiederherstellen ------------------------------------------------------

// Die Wiederherstellung in einen leeren Ordner muss eine Datenbank ergeben, die
// sich öffnen lässt und deren Kette gültig ist. Alles andere wäre eine
// Sicherung, die niemand gebrauchen kann.
func TestRestoreIntoEmptyDirectoryYieldsAValidChain(t *testing.T) {
	env := newBackupEnv(t)
	env.book(t, 11900)
	env.book(t, 4200)
	env.receiptFile(t, "beleg.pdf", "%PDF-1.4 Beleg")

	run, err := env.backups.CreateBackup(context.Background(), t.TempDir(), domain.BackupKindManual)
	if err != nil {
		t.Fatalf("Sicherung: %v", err)
	}

	target := filepath.Join(t.TempDir(), "wiederhergestellt")
	restored, err := env.backups.RestoreFromBackup(context.Background(), run.Target, target)
	if err != nil {
		t.Fatalf("Wiederherstellung: %v", err)
	}
	if !restored.Success {
		t.Fatalf("die Wiederherstellung ist fehlgeschlagen: %s", restored.Message)
	}

	chain, err := verifyChainAt(context.Background(), filepath.Join(target, "buchfink.sqlite"))
	if err != nil {
		t.Fatalf("die wiederhergestellte Datenbank ließ sich nicht prüfen: %v", err)
	}
	if !chain.IsValid {
		t.Errorf("die Kette der wiederhergestellten Datenbank ist gebrochen: %s", chain.Message)
	}
	if chain.TotalEntries != 2 {
		t.Errorf("erwartet 2 wiederhergestellte Buchungen, gefunden %d", chain.TotalEntries)
	}

	files, err := verifyReceiptFilesAt(context.Background(), target)
	if err != nil {
		t.Fatalf("Belegprüflauf: %v", err)
	}
	if !files.IsValid || files.Checked != 1 {
		t.Errorf("die wiederhergestellten Belegdateien sind zu beanstanden: %s", files.Message)
	}
}

// Niemals über bestehende Daten: eine Wiederherstellung, die einen belegten
// Ordner überschreibt, macht aus einem Datenverlust zwei.
func TestRestoreRefusesANonEmptyDirectory(t *testing.T) {
	env := newBackupEnv(t)
	env.book(t, 11900)

	run, err := env.backups.CreateBackup(context.Background(), t.TempDir(), domain.BackupKindManual)
	if err != nil {
		t.Fatalf("Sicherung: %v", err)
	}

	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, "wichtig.txt"), []byte("nicht überschreiben"), 0o600); err != nil {
		t.Fatalf("Testdatei: %v", err)
	}

	if _, err := env.backups.RestoreFromBackup(context.Background(), run.Target, target); err == nil {
		t.Fatal("die Wiederherstellung in einen belegten Ordner muss abgewiesen werden")
	} else if !strings.Contains(err.Error(), "nicht leer") {
		t.Errorf("die Fehlermeldung nennt den Grund nicht: %v", err)
	}

	if data, err := os.ReadFile(filepath.Join(target, "wichtig.txt")); err != nil ||
		string(data) != "nicht überschreiben" {
		t.Error("die vorhandene Datei wurde angetastet")
	}
}

// --- Prüfen ----------------------------------------------------------------

// Der Wiederherstellungstest muss die Sicherung prüfen und den Temporärordner
// danach wieder abräumen. Eine entpackte Buchführung soll nicht liegen bleiben.
func TestVerifyBackupSucceedsAndLeavesNothingBehind(t *testing.T) {
	env := newBackupEnv(t)
	env.book(t, 11900)
	env.receiptFile(t, "beleg.pdf", "%PDF-1.4 Beleg")

	run, err := env.backups.CreateBackup(context.Background(), t.TempDir(), domain.BackupKindManual)
	if err != nil {
		t.Fatalf("Sicherung: %v", err)
	}

	before := tempDirNames(t)
	verify, err := env.backups.VerifyBackup(context.Background(), run.Target)
	if err != nil {
		t.Fatalf("Sicherung prüfen: %v", err)
	}
	if !verify.Success {
		t.Fatalf("die geprüfte Sicherung wurde beanstandet: %s", verify.Message)
	}
	if verify.Kind != domain.BackupKindVerify {
		t.Errorf("der Lauf ist als %q vermerkt, erwartet %q", verify.Kind, domain.BackupKindVerify)
	}

	for name := range tempDirNames(t) {
		if !before[name] && strings.HasPrefix(name, "buchfink-verify-") {
			t.Errorf("der Temporärordner %s wurde nicht abgeräumt", name)
		}
	}
}

// Eine manipulierte Datei in der Sicherung muss auffallen — und zwar mit Namen.
// „Die Sicherung ist kaputt" hilft niemandem weiter.
func TestVerifyBackupFindsATamperedFile(t *testing.T) {
	env := newBackupEnv(t)
	env.book(t, 11900)
	env.receiptFile(t, "beleg.pdf", "%PDF-1.4 Beleg")

	run, err := env.backups.CreateBackup(context.Background(), t.TempDir(), domain.BackupKindManual)
	if err != nil {
		t.Fatalf("Sicherung: %v", err)
	}

	tampered := filepath.Join(t.TempDir(), "manipuliert.zip")
	changed := rewriteZip(t, run.Target, tampered, func(name string, data []byte) []byte {
		if strings.HasPrefix(name, "belege/") {
			return []byte("%PDF-1.4 etwas ganz anderes")
		}
		return data
	})
	if changed == "" {
		t.Fatal("in der Sicherung war keine Belegdatei zu ändern")
	}

	verify, err := env.backups.VerifyBackup(context.Background(), tampered)
	if err == nil && verify.Success {
		t.Fatal("die manipulierte Sicherung wurde nicht beanstandet")
	}
	message := verify.Message
	if err != nil {
		message = err.Error()
	}
	if !strings.Contains(message, changed) {
		t.Errorf("die Meldung nennt die betroffene Datei %q nicht: %s", changed, message)
	}
}

// Eine ZIP-Datei ohne backup.json ist keine Buchfink-Sicherung. Sie darf nicht
// halb entpackt und dann für gut befunden werden.
func TestVerifyBackupRejectsAForeignArchive(t *testing.T) {
	env := newBackupEnv(t)

	path := filepath.Join(t.TempDir(), "fremd.zip")
	out, err := os.Create(path)
	if err != nil {
		t.Fatalf("Testarchiv: %v", err)
	}
	w := zip.NewWriter(out)
	entry, _ := w.Create("irgendwas.txt")
	_, _ = entry.Write([]byte("kein Buchfink"))
	_ = w.Close()
	_ = out.Close()

	run, err := env.backups.VerifyBackup(context.Background(), path)
	if err == nil {
		t.Fatal("ein fremdes Archiv muss abgewiesen werden")
	}
	if !strings.Contains(err.Error(), BackupMetaFileName) {
		t.Errorf("die Meldung nennt die fehlende Metadatei nicht: %v", err)
	}
	if run.Success {
		t.Error("der Lauf ist als gelungen vermerkt")
	}
}

// --- Hilfen ----------------------------------------------------------------

func zipEntries(t *testing.T, path string) map[string]bool {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("Sicherung öffnen: %v", err)
	}
	defer reader.Close()
	names := map[string]bool{}
	for _, f := range reader.File {
		names[f.Name] = true
	}
	return names
}

func readBackupMeta(t *testing.T, path string) BackupMeta {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("Sicherung öffnen: %v", err)
	}
	defer reader.Close()
	for _, f := range reader.File {
		if f.Name != BackupMetaFileName {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("%s öffnen: %v", BackupMetaFileName, err)
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("%s lesen: %v", BackupMetaFileName, err)
		}
		var meta BackupMeta
		if err := json.Unmarshal(data, &meta); err != nil {
			t.Fatalf("%s lesen: %v", BackupMetaFileName, err)
		}
		return meta
	}
	t.Fatalf("%s fehlt in der Sicherung", BackupMetaFileName)
	return BackupMeta{}
}

// rewriteZip schreibt eine Kopie der Sicherung, in der eine Datei verändert
// wurde, und liefert deren Namen.
func rewriteZip(t *testing.T, src, dst string, change func(name string, data []byte) []byte) string {
	t.Helper()
	reader, err := zip.OpenReader(src)
	if err != nil {
		t.Fatalf("Sicherung öffnen: %v", err)
	}
	defer reader.Close()

	out, err := os.Create(dst)
	if err != nil {
		t.Fatalf("Kopie anlegen: %v", err)
	}
	defer out.Close()
	writer := zip.NewWriter(out)

	changed := ""
	for _, f := range reader.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("%s lesen: %v", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("%s lesen: %v", f.Name, err)
		}
		replaced := change(f.Name, data)
		if string(replaced) != string(data) {
			changed = f.Name
		}
		entry, err := writer.Create(f.Name)
		if err != nil {
			t.Fatalf("%s schreiben: %v", f.Name, err)
		}
		if _, err := entry.Write(replaced); err != nil {
			t.Fatalf("%s schreiben: %v", f.Name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Kopie schließen: %v", err)
	}
	return changed
}

func tempDirNames(t *testing.T) map[string]bool {
	t.Helper()
	entries, err := os.ReadDir(os.TempDir())
	if err != nil {
		t.Fatalf("Temporärordner lesen: %v", err)
	}
	names := map[string]bool{}
	for _, e := range entries {
		names[e.Name()] = true
	}
	return names
}

// Scheitert die Wiederherstellung, bleibt der Zielordner leer.
//
// Sonst stünde dort eine halbe Buchführung: die Dateien vor dem
// Prüfsummenfehler, ohne die dahinter. Der Ordner sähe aus wie ein Mandant, und
// der zweite Versuch fände ihn nicht mehr leer.
func TestRestoreLeavesAnEmptyDirectoryAfterAChecksumError(t *testing.T) {
	env := newBackupEnv(t)
	env.book(t, 11900)
	env.receiptFile(t, "beleg.pdf", "%PDF-1.4 Beleg")

	run, err := env.backups.CreateBackup(context.Background(), t.TempDir(), domain.BackupKindManual)
	if err != nil {
		t.Fatalf("Sicherung: %v", err)
	}

	tampered := filepath.Join(t.TempDir(), "manipuliert.zip")
	if changed := rewriteZip(t, run.Target, tampered, func(name string, data []byte) []byte {
		if strings.HasPrefix(name, "belege/") {
			return []byte("%PDF-1.4 etwas ganz anderes")
		}
		return data
	}); changed == "" {
		t.Fatal("in der Sicherung war keine Belegdatei zu ändern")
	}

	target := t.TempDir()
	restored, err := env.backups.RestoreFromBackup(context.Background(), tampered, target)
	if err == nil {
		t.Fatal("eine beschädigte Sicherung darf nicht wiederhergestellt werden")
	}
	if restored.Success {
		t.Error("der Lauf ist als gelungen vermerkt")
	}

	entries, err := os.ReadDir(target)
	if err != nil {
		t.Fatalf("Zielordner: %v", err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("der Zielordner hat nach dem Fehlschlag %d Einträge: %v", len(entries), names)
	}

	// Und der zweite Versuch — mit der heilen Sicherung — geht durch, weil der
	// Ordner leer geblieben ist.
	if _, err := env.backups.RestoreFromBackup(context.Background(), run.Target, target); err != nil {
		t.Fatalf("die Wiederherstellung in den aufgeräumten Ordner muss durchgehen: %v", err)
	}
}

// Beim Beenden zählt, ob gearbeitet wurde, nicht der zeitliche Abstand.
//
// Wer morgens sichert und den ganzen Tag bucht, verlöre sonst den Tag, weil die
// 24 Stunden noch nicht um sind. Ein Tag ohne Änderung wird dagegen nicht ein
// zweites Mal gesichert — die Sicherung selbst zählt nicht als Änderung.
func TestHasChangesSinceIgnoresTheBackupItself(t *testing.T) {
	env := newBackupEnv(t)
	ctx := context.Background()
	env.book(t, 11900)

	run, err := env.backups.CreateBackup(ctx, t.TempDir(), domain.BackupKindManual)
	if err != nil {
		t.Fatalf("Sicherung: %v", err)
	}
	last, err := env.backups.LastSuccessful(ctx)
	if err != nil || last == nil {
		t.Fatalf("letzte Sicherung: %v", err)
	}

	// Unmittelbar nach der Sicherung ist nichts geschehen — der eigene
	// Protokolleintrag des Laufs zählt nicht.
	if env.backups.HasChangesSince(ctx, last) {
		t.Error("nach der Sicherung darf keine Änderung gemeldet werden")
	}
	// Und ohne vorherige Sicherung gibt es nichts, woran sich das Gegenteil
	// zeigen ließe.
	if !env.backups.HasChangesSince(ctx, nil) {
		t.Error("ohne vorherige Sicherung muss gesichert werden")
	}

	// Eine Buchung danach ist eine Änderung, obwohl die 24 Stunden nicht um sind.
	env.book(t, 4200)
	if env.backups.IsDue(ctx, run.StartedAt.Add(time.Hour)) {
		t.Error("eine Stunde nach der Sicherung ist der Abstand noch nicht erreicht")
	}
	if !env.backups.HasChangesSince(ctx, last) {
		t.Error("die Buchung nach der Sicherung muss als Änderung gelten")
	}
}

// Zwei Sicherungen in derselben Sekunde überschreiben einander nicht; die
// zweite bekommt eine Nummer im Namen.
func TestTwoBackupsInOneSecondKeepBothFiles(t *testing.T) {
	env := newBackupEnv(t)
	ctx := context.Background()
	dir := t.TempDir()
	first, err := env.backups.CreateBackup(ctx, dir, domain.BackupKindManual)
	if err != nil {
		t.Fatal(err)
	}
	second, err := env.backups.CreateBackup(ctx, dir, domain.BackupKindManual)
	if err != nil {
		t.Fatalf("zweite Sicherung derselben Sekunde: %v", err)
	}
	if first.Target == second.Target {
		t.Errorf("beide Sicherungen liegen unter %s", first.Target)
	}
}
