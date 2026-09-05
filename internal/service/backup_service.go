package service

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/buildinfo"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/receiptstore"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/security"
	"gorm.io/gorm"
)

// Die Namen im Datenordner. Sie stehen hier, weil die Sicherung sie als
// Ganzes kennen muss: eine Sicherung, der eine dieser Dateien fehlt, ist keine.
const (
	dbFileName       = "buchfink.sqlite"
	keyfileName      = "buchfink.keyfile.json"
	receiptsDirName  = "belege"
	documentsDirName = "dokumente"
	// BackupMetaFileName ist die Metadatei in der Sicherung.
	BackupMetaFileName = "backup.json"
)

// AutoBackupInterval ist der Abstand, nach dem eine automatische Sicherung
// fällig wird.
//
// Ein Tag, weil ein Arbeitstag die Einheit ist, in der Buchführung entsteht:
// mehr verlöre im Ernstfall die Arbeit von Tagen, weniger liefe an einem
// gewöhnlichen Tag mehrfach über denselben Bestand.
const AutoBackupInterval = 24 * time.Hour

// BackupFileInfo ist eine Datei in der Sicherung mit ihrer Prüfsumme.
type BackupFileInfo struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

// BackupMeta ist der Inhalt von backup.json.
//
// Ohne sie ist eine ZIP-Datei nur ein Haufen Dateien: die Prüfsummen sind das,
// woran sich später zeigen lässt, dass die Sicherung heil ist — und zwar bevor
// sie gebraucht wird.
type BackupMeta struct {
	CreatedAt      string           `json:"createdAt"`
	TenantID       string           `json:"tenantId"`
	TenantName     string           `json:"tenantName"`
	ProgramVersion string           `json:"programVersion"`
	FileCount      int              `json:"fileCount"`
	Bytes          int64            `json:"bytes"`
	Files          []BackupFileInfo `json:"files"`
}

// BackupService sichert den Datenordner eines Mandanten und stellt ihn wieder
// her.
//
// Die Wiederherstellungsdatei liegt mit in der Sicherung. Das ist eine
// Abwägung und keine Nachlässigkeit: ohne den Schlüssel lässt sich die
// gesicherte Buchführung auf keinem anderen Rechner mehr öffnen, und eine
// Sicherung, die im Ernstfall nicht zu öffnen ist, ist keine. Der Preis ist,
// dass die Sicherung so schützenswert ist wie die Daten selbst — worauf die
// Oberfläche hinweist.
type BackupService struct {
	runRepo   domain.BackupRunRepository
	auditRepo domain.AuditRepository
	db        *gorm.DB

	dataDir    string
	tenantID   string
	tenantName string

	// openVault öffnet den Schlüsselbund eines entpackten Datenordners. Er
	// steht als Feld und nicht als fester Aufruf, damit die Prüfung ohne
	// Schlüsselbund des Betriebssystems getestet werden kann.
	openVault VaultOpener
}

// VaultOpener öffnet den Schlüsselbund zu einem Datenordner und einer
// Mandantenkennung.
type VaultOpener func(dataDir, tenantID string) (*security.Vault, error)

// NewBackupService verdrahtet die Sicherung.
func NewBackupService(
	runRepo domain.BackupRunRepository,
	auditRepo domain.AuditRepository,
	db *gorm.DB,
	dataDir, tenantID, tenantName string,
) *BackupService {
	return &BackupService{
		runRepo: runRepo, auditRepo: auditRepo, db: db,
		dataDir: dataDir, tenantID: tenantID, tenantName: tenantName,
		openVault: security.OpenTenantVault,
	}
}

// SetVaultOpener ersetzt den Weg, auf dem die Prüfung an den Schlüssel einer
// Sicherung kommt.
func (s *BackupService) SetVaultOpener(open VaultOpener) { s.openVault = open }

// GetRuns liefert die letzten Läufe.
func (s *BackupService) GetRuns(ctx context.Context, limit int) ([]domain.BackupRun, error) {
	if s.runRepo == nil {
		return make([]domain.BackupRun, 0), nil
	}
	if limit <= 0 {
		limit = 20
	}
	return s.runRepo.FindRecent(ctx, limit)
}

// LastSuccessful liefert die jüngste gelungene Sicherung oder nil.
func (s *BackupService) LastSuccessful(ctx context.Context) (*domain.BackupRun, error) {
	if s.runRepo == nil {
		return nil, nil
	}
	return s.runRepo.LatestSuccessful(ctx)
}

// IsDue meldet, ob eine automatische Sicherung fällig ist.
func (s *BackupService) IsDue(ctx context.Context, now time.Time) bool {
	last, err := s.LastSuccessful(ctx)
	if err != nil || last == nil {
		return true
	}
	return now.Sub(last.StartedAt) >= AutoBackupInterval
}

// CreateBackup schreibt eine Sicherung in den Zielordner und liefert den Lauf.
//
// Der Lauf wird auch dann festgehalten, wenn die Sicherung scheitert. Eine
// fehlgeschlagene Sicherung, von der niemand erfährt, ist gefährlicher als gar
// keine: sie wiegt in Sicherheit.
func (s *BackupService) CreateBackup(ctx context.Context, targetDir string, kind domain.BackupKind) (*domain.BackupRun, error) {
	run := &domain.BackupRun{
		Kind:           kind,
		StartedAt:      time.Now(),
		ProgramVersion: buildinfo.Version,
	}

	path, meta, err := s.writeArchive(targetDir)
	run.FinishedAt = time.Now()
	run.Target = path
	if err != nil {
		run.Success = false
		run.Message = err.Error()
		s.record(ctx, run)
		return run, err
	}

	run.Success = true
	run.FileCount = meta.FileCount
	run.Bytes = meta.Bytes
	run.Message = fmt.Sprintf("%d Dateien, %s gesichert", meta.FileCount, formatBytes(meta.Bytes))
	s.record(ctx, run)
	return run, nil
}

// writeArchive baut die ZIP-Datei.
func (s *BackupService) writeArchive(targetDir string) (string, *BackupMeta, error) {
	if strings.TrimSpace(targetDir) == "" {
		return "", nil, fmt.Errorf("es ist kein Sicherungsordner eingerichtet")
	}
	if s.dataDir == "" {
		return "", nil, fmt.Errorf("kein Datenordner geöffnet")
	}
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		return "", nil, fmt.Errorf("der Sicherungsordner konnte nicht angelegt werden: %w", err)
	}

	// Die Kopie der Datenbank entsteht in einem Temporärordner: VACUUM INTO
	// braucht einen Zielpfad, und der darf nicht im Datenordner liegen, weil
	// die Kopie sonst beim nächsten Lauf mitgesichert würde.
	tmpDir, err := os.MkdirTemp("", "buchfink-backup-*")
	if err != nil {
		return "", nil, fmt.Errorf("es konnte kein Arbeitsordner angelegt werden: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	dbCopy := filepath.Join(tmpDir, dbFileName)
	if err := repository.VacuumInto(s.db, dbCopy); err != nil {
		return "", nil, err
	}

	stamp := time.Now().Format("20060102-150405")
	name := fmt.Sprintf("buchfink-%s-%s.zip", safeSlug(s.tenantName, s.tenantID), stamp)
	zipPath := filepath.Join(targetDir, name)

	out, err := os.OpenFile(zipPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return zipPath, nil, fmt.Errorf("die Sicherungsdatei konnte nicht angelegt werden: %w", err)
	}
	writer := zip.NewWriter(out)

	meta := &BackupMeta{
		CreatedAt:      time.Now().Format(time.RFC3339),
		TenantID:       s.tenantID,
		TenantName:     s.tenantName,
		ProgramVersion: buildinfo.Version,
		Files:          make([]BackupFileInfo, 0, 32),
	}

	add := func(rel, absolute string) error {
		info, err := addFileToZip(writer, rel, absolute)
		if err != nil {
			return err
		}
		meta.Files = append(meta.Files, *info)
		meta.FileCount++
		meta.Bytes += info.Bytes
		return nil
	}

	fail := func(err error) (string, *BackupMeta, error) {
		writer.Close()
		out.Close()
		os.Remove(zipPath)
		return zipPath, nil, err
	}

	if err := add(dbFileName, dbCopy); err != nil {
		return fail(err)
	}
	// Die Wiederherstellungsdatei nur, wenn es sie gibt: ein Mandant aus der
	// Zeit vor der Feldverschlüsselung hat keine, und das ist kein Fehler.
	keyfile := filepath.Join(s.dataDir, keyfileName)
	if _, err := os.Stat(keyfile); err == nil {
		if err := add(keyfileName, keyfile); err != nil {
			return fail(err)
		}
	}
	for _, dir := range []string{receiptsDirName, documentsDirName} {
		if err := s.addTree(dir, add); err != nil {
			return fail(err)
		}
	}

	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fail(fmt.Errorf("die Metadaten konnten nicht erzeugt werden: %w", err))
	}
	entry, err := writer.Create(BackupMetaFileName)
	if err != nil {
		return fail(fmt.Errorf("%s konnte nicht angelegt werden: %w", BackupMetaFileName, err))
	}
	if _, err := entry.Write(metaBytes); err != nil {
		return fail(fmt.Errorf("%s konnte nicht geschrieben werden: %w", BackupMetaFileName, err))
	}

	if err := writer.Close(); err != nil {
		out.Close()
		os.Remove(zipPath)
		return zipPath, nil, fmt.Errorf("die Sicherung konnte nicht abgeschlossen werden: %w", err)
	}
	if err := out.Close(); err != nil {
		os.Remove(zipPath)
		return zipPath, nil, fmt.Errorf("die Sicherung konnte nicht geschlossen werden: %w", err)
	}
	return zipPath, meta, nil
}

// addTree nimmt einen Unterordner des Datenordners vollständig auf.
func (s *BackupService) addTree(dir string, add func(rel, absolute string) error) error {
	root := filepath.Join(s.dataDir, dir)
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return nil
	}
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(s.dataDir, path)
		if err != nil {
			return err
		}
		return add(filepath.ToSlash(rel), path)
	})
}

func addFileToZip(w *zip.Writer, rel, absolute string) (*BackupFileInfo, error) {
	src, err := os.Open(absolute)
	if err != nil {
		return nil, fmt.Errorf("%s konnte nicht gelesen werden: %w", rel, err)
	}
	defer src.Close()

	entry, err := w.Create(rel)
	if err != nil {
		return nil, fmt.Errorf("%s konnte nicht in die Sicherung aufgenommen werden: %w", rel, err)
	}
	digest := sha256.New()
	written, err := io.Copy(io.MultiWriter(entry, digest), src)
	if err != nil {
		return nil, fmt.Errorf("%s konnte nicht geschrieben werden: %w", rel, err)
	}
	return &BackupFileInfo{
		Path: rel, SHA256: hex.EncodeToString(digest.Sum(nil)), Bytes: written,
	}, nil
}

// --- Prüfen ---------------------------------------------------------------

// VerifyBackup ist der Wiederherstellungstest.
//
// Er entpackt die Sicherung in einen Temporärordner, prüft die Prüfsummen aus
// backup.json, öffnet die Datenbank schreibgeschützt, rechnet die Hash-Chain
// über alle Geschäftsjahre nach, prüft die Belegdateien und räumt den
// Temporärordner wieder ab. Eine Sicherung, die noch nie zurückgespielt wurde,
// ist eine Vermutung; erst dieser Lauf macht eine Zusage daraus.
func (s *BackupService) VerifyBackup(ctx context.Context, zipPath string) (*domain.BackupRun, error) {
	run := &domain.BackupRun{
		Kind:           domain.BackupKindVerify,
		StartedAt:      time.Now(),
		Target:         zipPath,
		ProgramVersion: buildinfo.Version,
	}

	tmpDir, err := os.MkdirTemp("", "buchfink-verify-*")
	if err != nil {
		run.FinishedAt = time.Now()
		run.Message = fmt.Sprintf("es konnte kein Arbeitsordner angelegt werden: %v", err)
		s.record(ctx, run)
		return run, err
	}
	// Der Temporärordner geht in jedem Fall wieder weg — auch wenn die Prüfung
	// scheitert. Eine entpackte Buchführung, die irgendwo liegen bleibt, ist
	// genau das, was die Verschlüsselung verhindern soll.
	defer os.RemoveAll(tmpDir)

	meta, count, err := extractBackup(zipPath, tmpDir)
	run.FinishedAt = time.Now()
	if err != nil {
		run.Message = err.Error()
		s.record(ctx, run)
		return run, err
	}
	run.FileCount = count
	if meta != nil {
		run.Bytes = meta.Bytes
	}

	// Die Sicherung wird mit ihrem eigenen Schlüssel gelesen und nicht mit dem
	// des gerade offenen Mandanten. Sonst scheiterte die Entschlüsselung, sobald
	// jemand die Sicherung eines anderen Mandanten prüft — und eine heile
	// Sicherung käme als „beschädigt" zurück.
	readCtx, keyErr := s.verificationContext(ctx, tmpDir, meta)
	if keyErr != nil {
		run.Success = false
		run.Message = keyErr.Error()
		s.record(ctx, run)
		return run, keyErr
	}

	problems := make([]string, 0)

	chain, chainErr := verifyChainAt(readCtx, filepath.Join(tmpDir, dbFileName))
	switch {
	case chainErr != nil:
		problems = append(problems, fmt.Sprintf("Die Datenbank ließ sich nicht prüfen: %v", chainErr))
	case !chain.IsValid:
		problems = append(problems, chain.Message)
	}

	files, filesErr := verifyReceiptFilesAt(readCtx, tmpDir)
	switch {
	case filesErr != nil:
		problems = append(problems, fmt.Sprintf("Die Belegdateien ließen sich nicht prüfen: %v", filesErr))
	case !files.IsValid:
		problems = append(problems, files.Message)
	}

	run.FinishedAt = time.Now()
	if len(problems) > 0 {
		run.Success = false
		run.Message = strings.Join(problems, " ")
		s.record(ctx, run)
		return run, nil
	}

	run.Success = true
	run.Message = fmt.Sprintf(
		"Sicherung geprüft: %d Dateien, %d Buchungen mit gültiger Kette, %d Belegdateien unversehrt.",
		count, chain.TotalEntries, files.Checked)
	s.record(ctx, run)
	return run, nil
}

// RestoreFromBackup entpackt eine Sicherung in einen leeren Zielordner.
//
// Niemals über bestehende Daten: eine Wiederherstellung, die einen belegten
// Ordner überschreibt, macht aus einem Datenverlust zwei. Der Anwender bekommt
// stattdessen die Meldung und wählt einen leeren Ordner.
func (s *BackupService) RestoreFromBackup(ctx context.Context, zipPath, targetDir string) (*domain.BackupRun, error) {
	run := &domain.BackupRun{
		Kind:           domain.BackupKindRestore,
		StartedAt:      time.Now(),
		Target:         zipPath,
		ProgramVersion: buildinfo.Version,
	}

	if err := ensureEmptyDir(targetDir); err != nil {
		run.FinishedAt = time.Now()
		run.Message = err.Error()
		s.record(ctx, run)
		return run, err
	}

	meta, count, err := extractBackup(zipPath, targetDir)
	run.FinishedAt = time.Now()
	run.FileCount = count
	if meta != nil {
		run.Bytes = meta.Bytes
	}
	if err != nil {
		run.Message = err.Error()
		s.record(ctx, run)
		return run, err
	}

	run.Success = true
	run.Message = fmt.Sprintf("%d Dateien nach %s wiederhergestellt", count, targetDir)
	s.record(ctx, run)
	return run, nil
}

// ensureEmptyDir besteht auf einem leeren oder noch nicht vorhandenen Ordner.
func ensureEmptyDir(dir string) error {
	if strings.TrimSpace(dir) == "" {
		return fmt.Errorf("kein Zielordner für die Wiederherstellung angegeben")
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return os.MkdirAll(dir, 0o700)
	}
	if err != nil {
		return fmt.Errorf("der Zielordner konnte nicht gelesen werden: %w", err)
	}
	if len(entries) > 0 {
		return fmt.Errorf(
			"der Ordner %s ist nicht leer. Eine Wiederherstellung darf bestehende Daten nicht überschreiben — bitte einen leeren Ordner wählen",
			dir)
	}
	return nil
}

// ReadBackupMeta liest backup.json aus einer Sicherung, ohne sie zu entpacken.
//
// Die Wiederherstellung braucht die Mandantenkennung, bevor sie den Ordner
// anmeldet: der Schlüsselbund führt das Geheimnis unter der Kennung, unter der
// die Sicherung entstanden ist. Meldete die Wiederherstellung den Ordner unter
// einer neuen Kennung an, fände sie den Schlüssel nicht — und ein
// wiederhergestellter Mandant bliebe auf demselben Rechner gesperrt, auf dem
// sein Geheimnis liegt.
func ReadBackupMeta(zipPath string) (*BackupMeta, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("die Sicherung %s konnte nicht geöffnet werden: %w", filepath.Base(zipPath), err)
	}
	defer reader.Close()
	return readMetaFromZip(reader.File, zipPath)
}

// readMetaFromZip liest die Metadatei aus den Einträgen einer Sicherung.
func readMetaFromZip(files []*zip.File, zipPath string) (*BackupMeta, error) {
	var meta *BackupMeta
	for _, entry := range files {
		if entry.Name != BackupMetaFileName {
			continue
		}
		rc, err := entry.Open()
		if err != nil {
			return nil, fmt.Errorf("%s konnte nicht gelesen werden: %w", BackupMetaFileName, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("%s konnte nicht gelesen werden: %w", BackupMetaFileName, err)
		}
		meta = &BackupMeta{}
		if err := json.Unmarshal(data, meta); err != nil {
			return nil, fmt.Errorf("%s ist unlesbar: %w", BackupMetaFileName, err)
		}
	}
	if meta == nil {
		return nil, fmt.Errorf("die Datei %s fehlt — %s ist keine Buchfink-Sicherung",
			BackupMetaFileName, filepath.Base(zipPath))
	}
	return meta, nil
}

// extractBackup entpackt eine Sicherung und prüft dabei jede Datei gegen die
// Prüfsumme aus backup.json.
func extractBackup(zipPath, targetDir string) (*BackupMeta, int, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, 0, fmt.Errorf("die Sicherung %s konnte nicht geöffnet werden: %w", filepath.Base(zipPath), err)
	}
	defer reader.Close()

	meta, err := readMetaFromZip(reader.File, zipPath)
	if err != nil {
		return nil, 0, err
	}

	expected := make(map[string]BackupFileInfo, len(meta.Files))
	for _, f := range meta.Files {
		expected[f.Path] = f
	}

	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		return meta, 0, fmt.Errorf("der Zielordner konnte nicht angelegt werden: %w", err)
	}

	written := 0
	damaged := make([]string, 0)
	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() || entry.Name == BackupMetaFileName {
			continue
		}
		sum, err := extractOne(entry, targetDir)
		if err != nil {
			return meta, written, err
		}
		written++
		want, ok := expected[entry.Name]
		if !ok {
			damaged = append(damaged, fmt.Sprintf("%s steht nicht in %s", entry.Name, BackupMetaFileName))
			continue
		}
		if want.SHA256 != sum {
			damaged = append(damaged, fmt.Sprintf("%s stimmt nicht mit seiner Prüfsumme überein", entry.Name))
		}
		delete(expected, entry.Name)
	}
	for path := range expected {
		damaged = append(damaged, fmt.Sprintf("%s fehlt in der Sicherung", path))
	}

	if len(damaged) > 0 {
		sort.Strings(damaged)
		return meta, written, fmt.Errorf("die Sicherung ist beschädigt: %s", strings.Join(damaged, "; "))
	}
	return meta, written, nil
}

// extractOne schreibt einen Eintrag und liefert seine Prüfsumme.
func extractOne(entry *zip.File, targetDir string) (string, error) {
	rel := filepath.Clean(filepath.FromSlash(entry.Name))
	if filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		// Ein Eintrag, dessen Name aus dem Zielordner herausführt. Eine
		// fremde ZIP-Datei darf nicht ins Dateisystem schreiben, wohin sie
		// will.
		return "", fmt.Errorf("die Sicherung enthält einen unzulässigen Pfad: %q", entry.Name)
	}
	abs := filepath.Join(targetDir, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o700); err != nil {
		return "", fmt.Errorf("der Ordner für %s konnte nicht angelegt werden: %w", entry.Name, err)
	}

	rc, err := entry.Open()
	if err != nil {
		return "", fmt.Errorf("%s konnte nicht gelesen werden: %w", entry.Name, err)
	}
	defer rc.Close()

	dst, err := os.OpenFile(abs, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return "", fmt.Errorf("%s konnte nicht angelegt werden: %w", entry.Name, err)
	}
	defer dst.Close()

	digest := sha256.New()
	if _, err := io.Copy(io.MultiWriter(dst, digest), rc); err != nil {
		return "", fmt.Errorf("%s konnte nicht entpackt werden: %w", entry.Name, err)
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

// verificationContext bindet den Schlüsselbund der Sicherung an den Kontext,
// mit dem die entpackte Datenbank gelesen wird.
//
// Drei Fälle: ohne Keyfile in der Sicherung steht alles im Klartext — der
// Kontext bindet dann ausdrücklich keinen Schlüssel, damit nicht der des
// offenen Mandanten einspringt. Mit Keyfile wird der Schlüssel zur
// Mandantenkennung aus backup.json geöffnet. Liegt er auf diesem Rechner nicht
// vor, wird die Prüfung mit klarer Meldung abgelehnt: „ich habe den Schlüssel
// nicht" ist eine andere Auskunft als „die Sicherung ist beschädigt".
func (s *BackupService) verificationContext(
	ctx context.Context, dataDir string, meta *BackupMeta,
) (context.Context, error) {
	if !security.KeyfileExists(dataDir) {
		return repository.WithVault(ctx, nil), nil
	}
	tenantID := ""
	if meta != nil {
		tenantID = meta.TenantID
	}
	if tenantID == "" {
		return nil, fmt.Errorf(
			"die Sicherung trägt eine Wiederherstellungsdatei, aber %s nennt keinen Mandanten — "+
				"ohne ihn lässt sich der Schlüssel nicht zuordnen", BackupMetaFileName)
	}
	open := s.openVault
	if open == nil {
		open = security.OpenTenantVault
	}
	vault, err := open(dataDir, tenantID)
	if err != nil {
		return nil, fmt.Errorf(
			"die Sicherung des Mandanten %s ließ sich nicht aufschließen: %v. "+
				"Die Dateien sind vollständig; geprüft werden können sie erst auf einem Rechner, "+
				"auf dem der Schlüssel dieses Mandanten liegt, oder nach einer Wiederherstellung "+
				"aus der Wiederherstellungsdatei", tenantID, err)
	}
	return repository.WithVault(ctx, vault), nil
}

// verifyChainAt rechnet die Hash-Chain einer Datenbankdatei nach.
func verifyChainAt(ctx context.Context, dbPath string) (domain.IntegrityCheckResult, error) {
	db, err := repository.OpenReadOnlyDB(dbPath)
	if err != nil {
		return domain.IntegrityCheckResult{}, err
	}
	defer closeDB(db)

	journalRepo := repository.NewJournalRepository(db)
	years, err := journalRepo.GetAvailableFiscalYears(ctx)
	if err != nil {
		return domain.IntegrityCheckResult{}, err
	}
	byYear := make(map[int][]domain.JournalEntry, len(years))
	for _, year := range years {
		entries, err := journalRepo.FindAll(ctx, year)
		if err != nil {
			return domain.IntegrityCheckResult{}, err
		}
		byYear[year] = entries
	}
	return accounting.NewHashChain().VerifyYears(byYear), nil
}

// verifyReceiptFilesAt prüft die Belegdateien und Anlagendokumente eines
// entpackten Datenordners.
//
// Beides und nicht nur die Belege: ein Vertrag zum Anlagegut liegt in derselben
// Sicherung und ist genauso aufbewahrungspflichtig. Der Prüflauf im laufenden
// Betrieb sieht beide — der Wiederherstellungstest muss dieselbe Auskunft geben.
func verifyReceiptFilesAt(ctx context.Context, dataDir string) (*domain.FileCheckResult, error) {
	db, err := repository.OpenReadOnlyDB(filepath.Join(dataDir, dbFileName))
	if err != nil {
		return nil, err
	}
	defer closeDB(db)

	receipts, err := repository.NewReceiptRepository(db).FindAll(ctx, 0)
	if err != nil {
		return nil, err
	}
	documents, err := repository.NewAssetRepository(db).FindAllDocuments(ctx)
	if err != nil {
		return nil, err
	}
	return checkReceiptFiles(receipts, documents, receiptstore.New(dataDir)), nil
}

func closeDB(db *gorm.DB) {
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

// RecordRun hält einen anderswo ausgeführten Lauf in dieser Datenbank fest.
//
// Die Wiederherstellung läuft mit dem Dienst des Mandanten, der gerade offen
// ist — oder ganz ohne einen, wenn sie vom Startbildschirm aus angestoßen wird.
// Der Lauf gehört aber auch in die Bücher des wiederhergestellten Mandanten:
// die Oberfläche zeigt die letzten Läufe je Mandant, und ein Mandant, der aus
// einer Sicherung entstanden ist, hätte dort sonst keinen einzigen — mit der
// Auskunft „noch nie gesichert" ausgerechnet für den Bestand, dessen Herkunft
// eine Sicherung ist.
//
// Eine Kopie und kein Verschieben: der ausführende Mandant hat den Lauf
// tatsächlich ausgeführt und protokolliert das zu Recht.
func (s *BackupService) RecordRun(ctx context.Context, run *domain.BackupRun) {
	if run == nil {
		return
	}
	copied := *run
	copied.ID = 0
	s.record(ctx, &copied)
}

// record hält den Lauf in der Datenbank und im Änderungsprotokoll fest.
func (s *BackupService) record(ctx context.Context, run *domain.BackupRun) {
	if s.runRepo != nil {
		_ = s.runRepo.Create(ctx, run)
	}
	if s.auditRepo == nil {
		return
	}
	outcome := "gelungen"
	if !run.Success {
		outcome = "fehlgeschlagen"
	}
	_ = s.auditRepo.Log(ctx, domain.AuditActionExport, "BACKUP", string(run.Kind),
		fmt.Sprintf("Sicherungslauf %s %s: %s", run.Kind, outcome, run.Message))
}

// safeSlug macht aus dem Mandantennamen einen Bestandteil eines Dateinamens.
func safeSlug(name, fallback string) string {
	out := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		case r == '-' || r == '_':
			return r
		}
		return '-'
	}, name)
	out = strings.Trim(strings.ReplaceAll(out, "--", "-"), "-")
	if out == "" {
		out = strings.Trim(fallback, "-")
	}
	if out == "" {
		return "mandant"
	}
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}

func formatBytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/float64(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/float64(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f kB", float64(n)/float64(1<<10))
	default:
		return fmt.Sprintf("%d Bytes", n)
	}
}
