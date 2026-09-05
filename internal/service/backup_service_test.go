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
	"github.com/buchfink/buchfink/internal/security"
	"gorm.io/gorm"
)

// backupEnv ist ein Mandant auf der Platte: eine echte Datenbankdatei, ein
// Belegordner, eine Wiederherstellungsdatei. Die Sicherung lässt sich nicht
// gegen eine Datenbank im Arbeitsspeicher prüfen — sie kopiert Dateien.
type backupEnv struct {
	dataDir string
	db      *gorm.DB
	vault   *security.Vault
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
	journal := NewJournalService(
		journalRepo, repository.NewAccountRepository(db), repository.NewContactRepository(db),
		repository.NewAuditRepository(db), settingsRepo, 2026,
	)

	// Ein echter Schlüsselbund und die Wiederherstellungsdatei dazu — kein
	// Platzhalter: die Felder Buchungstext, Belegpfad und Bewirtungsangaben
	// liegen verschlüsselt in der Datenbank, und eine Sicherungsprüfung, die
	// nur den Klartextfall kennt, prüft nicht das, was der Anwender sichert.
	keyfile, vault, err := security.NewKeyfile("geheimnis-des-mandanten")
	if err != nil {
		t.Fatalf("Keyfile erzeugen: %v", err)
	}
	if err := security.SaveKeyfile(dataDir, keyfile); err != nil {
		t.Fatalf("Keyfile schreiben: %v", err)
	}
	repository.SetActiveVault(vault)
	t.Cleanup(func() { repository.SetActiveVault(nil) })

	backups := NewBackupService(
		repository.NewBackupRunRepository(db), repository.NewAuditRepository(db),
		db, dataDir, "tenant_test", "Pfennig Ventures GmbH")
	// Der Schlüsselbund des Betriebssystems steht im Test nicht zur Verfügung.
	// An seine Stelle tritt der Schlüssel dieses Mandanten — so, wie ihn die
	// Anwendung auf dem Rechner des Anwenders bekäme.
	backups.SetVaultOpener(func(string, string) (*security.Vault, error) { return vault, nil })

	return &backupEnv{
		dataDir: dataDir,
		db:      db,
		vault:   vault,
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
		repository.NewReceiptRepository(e.db), repository.NewJournalRepository(e.db),
		store, repository.NewAuditRepository(e.db), 2026,
	)
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("Testdatei schreiben: %v", err)
	}
	receipt, err := svc.File(context.Background(), FileReceiptRequest{
		Direction: domain.DirectionIncoming,
		Files:     []NewFile{{Role: domain.ReceiptRoleOriginal, Path: path}},
	})
	if err != nil {
		t.Fatalf("Beleg ablegen: %v", err)
	}
	return receipt
}

// assetDocument legt ein Anlagegut mit einem Dokument an.
func (e *backupEnv) assetDocument(t *testing.T, name, content string) {
	t.Helper()
	svc := NewAssetService(
		repository.NewAssetRepository(e.db), repository.NewJournalRepository(e.db),
		e.journal, repository.NewNumberRangeRepository(e.db),
		repository.NewContactRepository(e.db), repository.NewSettingsRepository(e.db),
		repository.NewAuditRepository(e.db), 2026,
	)
	svc.SetDocumentStore(receiptstore.New(e.dataDir))
	asset, err := svc.Save(context.Background(), &domain.FixedAsset{
		Name: "Fräsmaschine", Class: domain.AssetClassTangible,
		Account: "0440", DepreciationAccount: "6220",
		AcquisitionDate: "2026-01-10", AcquisitionCost: 1_200_000,
		UsefulLifeMonths: 48, Method: domain.DepreciationLinear,
	})
	if err != nil {
		t.Fatalf("Anlagegut anlegen: %v", err)
	}
	if _, err := svc.AttachDocument(context.Background(), AttachDocumentRequest{
		AssetID: asset.ID, Content: []byte(content), FileName: name,
	}); err != nil {
		t.Fatalf("Anlagendokument ablegen: %v", err)
	}
}

// --- Sichern ---------------------------------------------------------------

// Die Sicherung muss alles enthalten, was einen Mandanten ausmacht: die
// Datenbank, den Schlüssel, die Belege — und backup.json mit den Prüfsummen.
// Fehlt eine der vier, ist der Bestand im Ernstfall nicht wiederherstellbar.
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
	for _, want := range []string{"buchfink.sqlite", "buchfink.keyfile.json", BackupMetaFileName} {
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
// danach wieder abräumen. Eine entpackte Buchführung, die liegen bleibt, ist
// genau das, was die Verschlüsselung verhindern soll.
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

// Die Prüfung einer Sicherung muss den Schlüssel benutzen, der in ihr liegt —
// nicht den des gerade offenen Mandanten.
//
// Sonst scheitert die Entschlüsselung, sobald jemand die Sicherung eines
// anderen Mandanten prüft, und eine vollständig heile Sicherung käme als
// „beschädigt" zurück. Genau darauf verließe sich niemand mehr.
func TestVerifyBackupUsesTheKeyOfTheBackedUpTenant(t *testing.T) {
	env := newBackupEnv(t)
	ownVault := env.vault

	env.book(t, 11900)
	env.book(t, 4200)
	env.receiptFile(t, "beleg.pdf", "%PDF-1.4 Beleg")

	run, err := env.backups.CreateBackup(context.Background(), t.TempDir(), domain.BackupKindManual)
	if err != nil {
		t.Fatalf("Sicherung: %v", err)
	}

	// Jetzt ist ein anderer Mandant offen — mit einem anderen Schlüssel.
	_, otherVault, err := security.NewKeyfile("geheimnis-eines-anderen")
	if err != nil {
		t.Fatalf("zweites Keyfile: %v", err)
	}
	repository.SetActiveVault(otherVault)

	// Die Prüfung holt sich den Schlüssel aus der Sicherung. Der Schlüsselbund
	// des Betriebssystems steht im Test nicht zur Verfügung; geprüft wird, dass
	// die Prüfung ihn zur Mandantenkennung aus backup.json erfragt und benutzt.
	asked := ""
	env.backups.SetVaultOpener(func(dataDir, tenantID string) (*security.Vault, error) {
		asked = tenantID
		if _, err := os.Stat(filepath.Join(dataDir, "buchfink.keyfile.json")); err != nil {
			t.Errorf("die Prüfung sucht das Keyfile nicht im entpackten Ordner: %v", err)
		}
		return ownVault, nil
	})

	verify, err := env.backups.VerifyBackup(context.Background(), run.Target)
	if err != nil {
		t.Fatalf("Sicherung prüfen: %v", err)
	}
	if asked != "tenant_test" {
		t.Errorf("die Prüfung fragte nach dem Schlüssel des Mandanten %q, erwartet tenant_test", asked)
	}
	if !verify.Success {
		t.Fatalf("die heile Sicherung eines anderen Mandanten wurde beanstandet: %s", verify.Message)
	}
	if !strings.Contains(verify.Message, "2 Buchungen") {
		t.Errorf("die Prüfung hat die Kette nicht über beide Buchungen gerechnet: %s", verify.Message)
	}
}

// Fehlt der Schlüssel zu einer verschlüsselten Sicherung, ist das keine
// Beschädigung. Die Meldung muss den Unterschied benennen — sonst wirft jemand
// eine heile Sicherung weg.
func TestVerifyBackupSaysSoWhenTheKeyIsMissing(t *testing.T) {
	env := newBackupEnv(t)
	env.book(t, 11900)
	run, err := env.backups.CreateBackup(context.Background(), t.TempDir(), domain.BackupKindManual)
	if err != nil {
		t.Fatalf("Sicherung: %v", err)
	}

	env.backups.SetVaultOpener(func(string, string) (*security.Vault, error) {
		return nil, security.ErrNoKeyringSecret
	})

	verify, err := env.backups.VerifyBackup(context.Background(), run.Target)
	if err == nil {
		t.Fatal("ohne Schlüssel muss die Prüfung abgelehnt werden")
	}
	if verify.Success {
		t.Error("der Lauf ist als gelungen vermerkt, obwohl nichts geprüft werden konnte")
	}
	if strings.Contains(verify.Message, "beschädigt") {
		t.Errorf("die Meldung nennt die Sicherung beschädigt, obwohl nur der Schlüssel fehlt: %s", verify.Message)
	}
	if !strings.Contains(verify.Message, "tenant_test") || !strings.Contains(verify.Message, "aufschließen") {
		t.Errorf("die Meldung sagt nicht, dass der Schlüssel des Mandanten fehlt: %s", verify.Message)
	}
}

// Ohne Wiederherstellungsdatei in der Sicherung liegen die Felder im Klartext.
// Die Prüfung muss dann ausdrücklich ohne Schlüssel lesen: griffe sie auf den
// des offenen Mandanten zurück, entschlüsselte sie Klartext und scheiterte.
func TestVerificationContextReadsPlainBackupsWithoutAnyKey(t *testing.T) {
	env := newBackupEnv(t)
	empty := t.TempDir()

	ctx, err := env.backups.verificationContext(context.Background(), empty,
		&BackupMeta{TenantID: "tenant_test"})
	if err != nil {
		t.Fatalf("Sicherung ohne Keyfile: %v", err)
	}
	// Der Mandant dieses Tests hat einen Schlüssel — die Prüfung darf ihn hier
	// gerade nicht benutzen.
	if got := repository.VaultForTest(ctx); got != nil {
		t.Error("die Prüfung einer unverschlüsselten Sicherung bindet einen Schlüssel")
	}
}

// Der Wiederherstellungstest prüft auch die Anlagendokumente. Sie liegen in
// derselben Sicherung und sind genauso aufbewahrungspflichtig.
func TestVerifyBackupChecksAssetDocumentsToo(t *testing.T) {
	env := newBackupEnv(t)
	env.book(t, 11900)
	env.receiptFile(t, "beleg.pdf", "%PDF-1.4 Beleg")
	env.assetDocument(t, "kaufvertrag.pdf", "%PDF-1.4 Kaufvertrag")

	run, err := env.backups.CreateBackup(context.Background(), t.TempDir(), domain.BackupKindManual)
	if err != nil {
		t.Fatalf("Sicherung: %v", err)
	}

	verify, err := env.backups.VerifyBackup(context.Background(), run.Target)
	if err != nil {
		t.Fatalf("Sicherung prüfen: %v", err)
	}
	if !verify.Success {
		t.Fatalf("die Sicherung wurde beanstandet: %s", verify.Message)
	}
	// Zwei Dateien: die Belegdatei und das Anlagendokument. Ohne das Dokument
	// stünde hier eine 1 — und der Vertrag wäre ungeprüft.
	if !strings.Contains(verify.Message, "2 Belegdateien") {
		t.Errorf("das Anlagendokument wurde nicht mitgeprüft: %s", verify.Message)
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
