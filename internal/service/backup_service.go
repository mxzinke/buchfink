package service

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/buildinfo"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/receiptstore"
	"github.com/buchfink/buchfink/internal/repository"
	"gorm.io/gorm"
)

// Die Namen im Datenordner. Sie stehen hier, weil die Sicherung sie als
// Ganzes kennen muss: eine Sicherung, der eine dieser Dateien fehlt, ist keine.
const (
	dbFileName      = "buchfink.sqlite"
	receiptsDirName = "belege"
	// documentsDirName hält die Fassungen der Verfahrensdokumentation.
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
// Die Sicherung ist unverschlüsselt und so schützenswert wie die Daten selbst;
// ihr Schutz am Ablageort liegt beim Anwender.
type BackupService struct {
	runRepo   domain.BackupRunRepository
	auditRepo domain.AuditRepository
	db        *gorm.DB

	dataDir    string
	tenantID   string
	tenantName string

	// recorder klammert den Vermerk eines Laufs. Der Kern setzt hier seinen
	// Schreibvorgang ein: der Lauf selbst dauert lange und läuft ohne
	// Schreibsperre, nur der Vermerk braucht sie.
	recorder func(ctx context.Context, fn func(ctx context.Context) error) error
}

// SetRecorder setzt die Klammer, in der Läufe vermerkt werden.
func (s *BackupService) SetRecorder(recorder func(ctx context.Context, fn func(ctx context.Context) error) error) {
	s.recorder = recorder
}

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
	}
}

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

// backupAuditEntityType ist der Eintragstyp der Sicherungsläufe im Protokoll.
// Er wird beim Zählen der Änderungen übergangen: eine Sicherung ist keine
// Änderung am Bestand, und ohne diese Ausnahme wäre nach jeder Sicherung sofort
// wieder eine fällig.
const backupAuditEntityType = "BACKUP"

// auditLookback ist die Zahl der Protokolleinträge, die HasChangesSince
// zurückliest. Gesucht wird der jüngste Eintrag, der keine Sicherung ist;
// mehrere Sicherungsläufe hintereinander sind selten, und eine Zahl statt des
// ganzen Protokolls hält den Aufruf beim Beenden kurz.
const auditLookback = 50

// HasChangesSince meldet, ob seit der letzten gelungenen Sicherung am Bestand
// gearbeitet wurde.
//
// Gefragt wird das Änderungsprotokoll: jede Buchung, jeder Beleg, jede
// Stammdatenänderung steht dort (GoBD Rz. 34), und was dort nicht steht, hat
// den Bestand nicht verändert. Ohne diese Frage hinge die Sicherung beim
// Beenden allein am Abstand von 24 Stunden — wer morgens sichert und den ganzen
// Tag bucht, verlöre den Tag.
//
// Ohne Protokoll und ohne vorherige Sicherung ist die Antwort ja: dann gibt es
// nichts, woran sich das Gegenteil zeigen ließe.
func (s *BackupService) HasChangesSince(ctx context.Context, last *domain.BackupRun) bool {
	if last == nil {
		return true
	}
	if s.auditRepo == nil {
		return false
	}
	entries, err := s.auditRepo.FindAll(ctx, auditLookback)
	if err != nil {
		// Ein Fehler beim Lesen ist kein „nichts geändert": im Zweifel wird
		// gesichert. Eine Sicherung zu viel kostet Platz, eine zu wenig die
		// Arbeit eines Tages.
		return true
	}
	for _, entry := range entries {
		if entry.EntityType == backupAuditEntityType {
			continue
		}
		return entry.Timestamp.After(last.StartedAt)
	}
	return false
}

// CreateBackup schreibt eine Sicherung in den Zielordner und liefert den Lauf.
//
// Der Lauf wird auch dann festgehalten, wenn die Sicherung scheitert. Eine
// fehlgeschlagene Sicherung, von der niemand erfährt, ist gefährlicher als gar
// keine: sie wiegt in Sicherheit.
func (s *BackupService) CreateBackup(ctx context.Context, targetDir string, kind domain.BackupKind) (*domain.BackupRun, error) {
	run := &domain.BackupRun{
		Kind:           kind,
		StartedAt:      time.Now().UTC(),
		Actor:          actor.Actor(),
		ProgramVersion: buildinfo.Version,
	}

	path, meta, err := s.writeArchive(targetDir)
	run.FinishedAt = time.Now().UTC()
	run.Target = path
	if err != nil {
		run.Success = false
		run.Message = err.Error()
		if recErr := s.record(ctx, run); recErr != nil {
			return run, errors.Join(err, recErr)
		}
		return run, err
	}

	run.Success = true
	run.FileCount = meta.FileCount
	run.Bytes = meta.Bytes
	run.Message = fmt.Sprintf("%d Dateien, %s gesichert", meta.FileCount, formatBytes(meta.Bytes))
	if err := s.record(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

// createUnique legt base.zip im Ordner an, ohne Vorhandenes zu überschreiben.
// Gibt es die Datei derselben Sekunde schon, bekommt die neue eine Nummer.
func createUnique(dir, base string) (string, *os.File, error) {
	var last error
	for n := 1; n <= 99; n++ {
		name := base + ".zip"
		if n > 1 {
			name = fmt.Sprintf("%s-%d.zip", base, n)
		}
		path := filepath.Join(dir, name)
		out, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			return path, out, nil
		}
		if !os.IsExist(err) {
			return path, nil, err
		}
		last = err
	}
	return filepath.Join(dir, base+".zip"), nil, last
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

	// Der Stempel im Dateinamen steht in Ortszeit, der Zeitpunkt im Beipackzettel
	// (BackupMeta.CreatedAt) und im Protokoll in UTC.
	//
	// Der Dateiname wird von einem Menschen gelesen, der im Zielordner nach der
	// Sicherung von gestern Abend sucht; eine Sicherung um 23:30 Uhr, die
	// „20260906-2130" hieße, fände er dort nicht. Verglichen und sortiert wird
	// nach dem Zeitpunkt im Beipackzettel, und der ist eindeutig.
	stamp := time.Now().Format("20060102-150405")
	base := fmt.Sprintf("buchfink-%s-%s", safeSlug(s.tenantName, s.tenantID), stamp)
	zipPath, out, err := createUnique(targetDir, base)
	if err != nil {
		return zipPath, nil, fmt.Errorf("die Sicherungsdatei konnte nicht angelegt werden: %w", err)
	}
	writer := zip.NewWriter(out)

	meta := &BackupMeta{
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
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
		StartedAt:      time.Now().UTC(),
		Actor:          actor.Actor(),
		Target:         zipPath,
		ProgramVersion: buildinfo.Version,
	}

	tmpDir, err := os.MkdirTemp("", "buchfink-verify-*")
	if err != nil {
		run.FinishedAt = time.Now().UTC()
		run.Message = fmt.Sprintf("es konnte kein Arbeitsordner angelegt werden: %v", err)
		if recErr := s.record(ctx, run); recErr != nil {
			return run, errors.Join(err, recErr)
		}
		return run, err
	}
	// Der Temporärordner geht in jedem Fall wieder weg — auch wenn die Prüfung
	// scheitert. Eine entpackte Buchführung soll nicht irgendwo liegen bleiben.
	defer os.RemoveAll(tmpDir)

	meta, count, err := extractBackup(zipPath, tmpDir)
	run.FinishedAt = time.Now().UTC()
	if err != nil {
		run.Message = err.Error()
		if recErr := s.record(ctx, run); recErr != nil {
			return run, errors.Join(err, recErr)
		}
		return run, err
	}
	run.FileCount = count
	if meta != nil {
		run.Bytes = meta.Bytes
	}

	problems := make([]string, 0)

	chain, chainErr := verifyChainAt(ctx, filepath.Join(tmpDir, dbFileName))
	switch {
	case chainErr != nil:
		problems = append(problems, fmt.Sprintf("Die Datenbank ließ sich nicht prüfen: %v", chainErr))
	case !chain.IsValid:
		problems = append(problems, chain.Message)
	}

	files, filesErr := verifyReceiptFilesAt(ctx, tmpDir)
	switch {
	case filesErr != nil:
		problems = append(problems, fmt.Sprintf("Die Belegdateien ließen sich nicht prüfen: %v", filesErr))
	case !files.IsValid:
		problems = append(problems, files.Message)
	}

	run.FinishedAt = time.Now().UTC()
	if len(problems) > 0 {
		run.Success = false
		run.Message = strings.Join(problems, " ")
		if err := s.record(ctx, run); err != nil {
			return nil, err
		}
		return run, nil
	}

	run.Success = true
	run.Message = fmt.Sprintf(
		"Sicherung geprüft: %d Dateien, %d Buchungen mit gültiger Kette, %d Belegdateien unversehrt.",
		count, chain.TotalEntries, files.Checked)
	if err := s.record(ctx, run); err != nil {
		return nil, err
	}
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
		StartedAt:      time.Now().UTC(),
		Actor:          actor.Actor(),
		Target:         zipPath,
		ProgramVersion: buildinfo.Version,
	}

	if err := ensureEmptyDir(targetDir); err != nil {
		run.FinishedAt = time.Now().UTC()
		run.Message = err.Error()
		if recErr := s.record(ctx, run); recErr != nil {
			return run, errors.Join(err, recErr)
		}
		return run, err
	}

	meta, count, err := extractBackup(zipPath, targetDir)
	run.FinishedAt = time.Now().UTC()
	run.FileCount = count
	if meta != nil {
		run.Bytes = meta.Bytes
	}
	if err != nil {
		// Der Zielordner bleibt leer, wenn die Wiederherstellung scheitert.
		//
		// Sonst stünde dort eine halbe Buchführung: die Dateien, die vor dem
		// Prüfsummenfehler geschrieben wurden, ohne die dahinter. Ein zweiter
		// Versuch fände den Ordner nicht mehr leer und verlangte einen neuen
		// (ensureEmptyDir), und der Bestand daneben sähe aus wie ein Mandant.
		// Ein Ordner mit halben Büchern ist gefährlicher als keiner.
		if removeErr := clearDir(targetDir); removeErr != nil {
			err = fmt.Errorf(
				"%w. Außerdem ließ sich der halb entpackte Zielordner nicht aufräumen: %v", err, removeErr)
		}
		run.FileCount = 0
		run.Message = err.Error()
		if recErr := s.record(ctx, run); recErr != nil {
			return run, errors.Join(err, recErr)
		}
		return run, err
	}

	run.Success = true
	run.Message = fmt.Sprintf("%d Dateien nach %s wiederhergestellt", count, targetDir)
	if err := s.record(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

// clearDir räumt den Inhalt eines Ordners weg und lässt ihn selbst stehen.
//
// Der Ordner selbst bleibt, weil ihn der Anwender ausgewählt hat und weil
// ensureEmptyDir ihn ohnehin wieder anlegen würde; entfernt wird, was die
// gescheiterte Wiederherstellung hineingeschrieben hat.
func clearDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
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

// verifyReceiptFilesAt prüft die Belegdateien eines entpackten Datenordners —
// dieselbe Auskunft wie der Prüflauf im laufenden Betrieb.
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
	return checkReceiptFiles(receipts, receiptstore.New(dataDir)), nil
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
func (s *BackupService) RecordRun(ctx context.Context, run *domain.BackupRun) error {
	if run == nil {
		return nil
	}
	copied := *run
	copied.ID = 0
	return s.record(ctx, &copied)
}

// record hält den Lauf in der Datenbank und im Änderungsprotokoll fest.
func (s *BackupService) record(ctx context.Context, run *domain.BackupRun) error {
	if s.recorder != nil {
		return s.recorder(ctx, func(ctx context.Context) error { return s.writeRecord(ctx, run) })
	}
	return s.writeRecord(ctx, run)
}

func (s *BackupService) writeRecord(ctx context.Context, run *domain.BackupRun) error {
	if s.runRepo != nil {
		if err := s.runRepo.Create(ctx, run); err != nil {
			return fmt.Errorf("der Sicherungslauf ließ sich nicht vermerken: %w", err)
		}
	}
	if s.auditRepo == nil {
		return nil
	}
	outcome := "gelungen"
	if !run.Success {
		outcome = "fehlgeschlagen"
	}
	return logAudit(ctx, s.auditRepo, domain.AuditActionExport, "BACKUP", string(run.Kind),
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
