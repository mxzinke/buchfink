package wailsbridge

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/changelog"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Die Nachweise der Welle 6: Änderungsprotokoll mit Kette, Versionshistorie,
// Migrationsprotokoll, Aufbewahrungsfristen, Verfahrensdokumentation,
// Eröffnungsbilanz des Umsteigers und die Altersstruktur der offenen Posten.
//
// Sie stehen in einer eigenen Datei, weil sie eine Sache sind: die Antwort auf
// die Frage, woran ein Prüfer erkennt, dass die Buchführung ordnungsmäßig ist.

// --- Änderungsprotokoll ---------------------------------------------------

// VerifyAuditChain rechnet die Kette des Änderungsprotokolls nach.
func (b *BuchfinkBridge) VerifyAuditChain() (domain.AuditChainResult, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.auditSvc == nil {
		empty := domain.AuditChainResult{IsValid: true, Message: "Kein aktiver Mandant."}
		empty.EnsureLists()
		return empty, nil
	}
	return b.auditSvc.VerifyChain(context.Background())
}

// GetAuditLogsFiltered liefert das Änderungsprotokoll mit Vorher/Nachher,
// eingeschränkt durch den Filter. Ein leerer Filter heißt: alles.
//
// Neben GetAuditLogs und nicht statt dessen: die bisherige Methode liefert die
// jüngsten 200 Einträge ohne Frage und wird von der Übersicht so gebraucht.
func (b *BuchfinkBridge) GetAuditLogsFiltered(limit int, filter domain.AuditFilter) ([]domain.AuditLogEntry, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.auditSvc == nil {
		return []domain.AuditLogEntry{}, nil
	}
	if limit <= 0 {
		limit = 200
	}
	return emptyList(b.auditSvc.GetLogsFiltered(context.Background(), limit, filter))
}

// GetChangeLog liefert die Versionshistorie des Programms als Markdown.
func (b *BuchfinkBridge) GetChangeLog() (string, error) {
	return changelog.Markdown(), nil
}

// GetSchemaMigrations liefert das Protokoll der Schemaänderungen.
func (b *BuchfinkBridge) GetSchemaMigrations() ([]domain.SchemaMigration, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.migrationRepo == nil {
		return []domain.SchemaMigration{}, nil
	}
	return emptyList(b.migrationRepo.FindSchemaMigrations(context.Background()))
}

// GetMigrationRecords liefert das Protokoll der Datenübernahmen.
func (b *BuchfinkBridge) GetMigrationRecords() ([]domain.MigrationRecord, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.migrationRepo == nil {
		return []domain.MigrationRecord{}, nil
	}
	return emptyList(b.migrationRepo.FindMigrationRecords(context.Background()))
}

// --- Aufbewahrung ---------------------------------------------------------

// GetRetentionOverview liefert die Fristenübersicht.
//
// year schränkt auf ein Geschäftsjahr ein; 0 heißt: alle. Das Löschkonzept und
// die Fünfjahresfrist der Systemumstellung stehen unabhängig davon dabei — sie
// gelten nicht je Jahr.
func (b *BuchfinkBridge) GetRetentionOverview(year int) (*service.RetentionOverview, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.retentionSvc == nil {
		empty := &service.RetentionOverview{}
		empty.EnsureLists()
		return empty, nil
	}
	overview, err := b.retentionSvc.Overview(context.Background(), "")
	if err != nil {
		return nil, err
	}
	if year > 0 {
		filtered := make([]service.RetentionYear, 0, 1)
		for _, row := range overview.Years {
			if row.FiscalYear == year {
				filtered = append(filtered, row)
			}
		}
		overview.Years = filtered
	}
	overview.EnsureLists()
	return overview, nil
}

// SetRetentionHold setzt die Aufbewahrungsfrist eines Geschäftsjahres aus.
//
// reason ist einer der Gründe des § 147 Abs. 3 Satz 5 AO ("audit", "appeal",
// "other"), description die Erläuterung — bei „other" Pflicht.
func (b *BuchfinkBridge) SetRetentionHold(year int, reason string, description string) (*domain.RetentionHold, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.retentionSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.retentionSvc.SetHold(context.Background(), year, domain.RetentionHoldReason(reason), description)
}

// ReleaseRetentionHold hebt eine Aussetzung auf.
func (b *BuchfinkBridge) ReleaseRetentionHold(id uint, reason string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return err
	}
	if b.retentionSvc == nil {
		return fmt.Errorf("kein aktiver Mandant")
	}
	return b.retentionSvc.ReleaseHold(context.Background(), id, reason)
}

// GetRetentionHolds liefert alle Aussetzungen, die geltenden zuerst.
func (b *BuchfinkBridge) GetRetentionHolds() ([]domain.RetentionHold, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.retentionSvc == nil {
		return []domain.RetentionHold{}, nil
	}
	return emptyList(b.retentionSvc.Holds(context.Background()))
}

// GetExpiredObjects liefert die Geschäftsjahre, deren Aufbewahrungsfrist
// abgelaufen ist.
func (b *BuchfinkBridge) GetExpiredObjects() ([]service.RetentionYear, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.retentionSvc == nil {
		return []service.RetentionYear{}, nil
	}
	return emptyList(b.retentionSvc.ExpiredYears(context.Background(), ""))
}

// ArchiveAndDeleteFiscalYear archiviert ein Geschäftsjahr und löscht es.
//
// Der Archivexport läuft hier und nicht beim Aufrufer: die Löschung ohne
// vorheriges Archiv wäre ein Datenverlust, und ob archiviert wurde, darf nicht
// davon abhängen, ob die Oberfläche zwei Aufrufe in der richtigen Reihenfolge
// macht. Scheitert das Archiv, wird nicht gelöscht.
func (b *BuchfinkBridge) ArchiveAndDeleteFiscalYear(year int, confirmation string) (*service.DeleteResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.retentionSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	ctx := context.Background()

	// Erst Bestätigung und Fristprüfung, dann das Archiv: ein Archivexport für
	// ein Jahr, das gar nicht gelöscht werden darf oder dessen Bestätigung
	// falsch getippt ist, wäre vergeudete Zeit und eine Datei, die niemand
	// angefordert hat — und zwar ein vollständiger Abzug der Buchführung eines
	// Jahres, der ungefragt auf der Platte liegen bliebe.
	if err := b.retentionSvc.EnsureDeleteAllowed(ctx, year, confirmation, ""); err != nil {
		return nil, err
	}

	archivePath, err := b.archiveForDeletion(year)
	if err != nil {
		return nil, err
	}

	return b.retentionSvc.ArchiveAndDelete(ctx, service.DeleteRequest{
		FiscalYear:   year,
		Confirmation: confirmation,
		ArchivePath:  archivePath,
	})
}

// --- Verfahrensdokumentation ----------------------------------------------

// GenerateProcedureDocumentation erzeugt eine neue Fassung und legt sie ab.
func (b *BuchfinkBridge) GenerateProcedureDocumentation() (*service.ProcDocResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.procDocSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	// Die Sicherungsläufe kommen frisch dazu: die Betriebsdokumentation soll
	// die letzten Läufe nennen und nicht den Stand beim Programmstart.
	env := service.ProcDocEnvironment{
		DataDir:      b.dataDir,
		BackupDir:    b.appConfig.BackupDir,
		BackupRhythm: b.backupRhythmLabel(),
	}
	if b.backupRunRepo != nil {
		if runs, err := b.backupRunRepo.FindRecent(context.Background(), 5); err == nil {
			env.BackupRuns = runs
		}
	}
	b.procDocSvc.SetEnvironment(env)
	return b.procDocSvc.Generate(context.Background(), time.Time{})
}

// GetProcedureDocumentations liefert die abgelegten Fassungen.
func (b *BuchfinkBridge) GetProcedureDocumentations() ([]domain.ProcedureDocumentation, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.procDocSvc == nil {
		return []domain.ProcedureDocumentation{}, nil
	}
	return emptyList(b.procDocSvc.Documentations(context.Background()))
}

// GetOrganisationTexts liefert die unternehmensindividuellen Freitexte, mit
// Mustern dort, wo noch nichts erfasst ist.
func (b *BuchfinkBridge) GetOrganisationTexts() (domain.OrganisationTexts, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.procDocSvc == nil {
		return domain.DefaultOrganisationTexts(), nil
	}
	return b.procDocSvc.OrganisationTexts(context.Background()), nil
}

// SaveOrganisationTexts schreibt die Freitexte der Organisationsanweisung.
func (b *BuchfinkBridge) SaveOrganisationTexts(texts domain.OrganisationTexts) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return err
	}
	if b.procDocSvc == nil {
		return fmt.Errorf("kein aktiver Mandant")
	}
	return b.procDocSvc.SaveOrganisationTexts(context.Background(), texts)
}

// SetSystemChangeDate hält den Umstellungszeitpunkt der Datenübernahme fest.
//
// An ihm hängt die Fünfjahresfrist des § 147 Abs. 6 Satz 6 AO: so lange ist das
// Altsystem für den Datenzugriff verfügbar zu halten.
func (b *BuchfinkBridge) SetSystemChangeDate(date string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return err
	}
	if b.settingsSvc == nil {
		return fmt.Errorf("kein aktiver Mandant")
	}
	trimmed := strings.TrimSpace(date)
	if trimmed != "" {
		if _, err := time.Parse("2006-01-02", trimmed); err != nil {
			return fmt.Errorf("der Umstellungszeitpunkt wird als Datum erwartet (JJJJ-MM-TT)")
		}
	}
	return b.settingsSvc.SetValue(context.Background(), domain.SettingSystemChangeDate, trimmed,
		"Umstellungszeitpunkt der Datenübernahme aus einem Altsystem gesetzt")
}

// --- Hinweise -------------------------------------------------------------

// ComplianceHints sind die Hinweise, die zur Einrichtung und zu den
// Einstellungen gehören.
type ComplianceHints struct {
	// LegalFormNote ist der Hinweis auf Kapitalkonten und Entnahmen, leer bei
	// Kapitalgesellschaften.
	LegalFormNote string `json:"legalFormNote"`
	// CloudWarning ist belegt, wenn der Datenordner in einem
	// Synchronisationsordner liegt.
	CloudWarning string `json:"cloudWarning"`
	// DataDir ist der Speicherort, damit die Oberfläche ihn nennen kann.
	DataDir string `json:"dataDir"`
	// TaxCaseHints nennt die Steuerfälle außerhalb des Funktionsumfangs.
	TaxCaseHints []string `json:"taxCaseHints"`
	// SystemChangeDate und SystemChangeNote sind der Umstellungszeitpunkt und
	// die Fünfjahresfrist daraus.
	SystemChangeDate string `json:"systemChangeDate"`
	SystemChangeNote string `json:"systemChangeNote"`
}

// GetComplianceHints liefert die Hinweise zu Rechtsform, Speicherort und
// Steuerfällen.
func (b *BuchfinkBridge) GetComplianceHints() (*ComplianceHints, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	hints := &ComplianceHints{
		DataDir:      b.dataDir,
		CloudWarning: domain.CloudFolderWarning(b.dataDir),
		TaxCaseHints: domain.TaxCaseHints(),
	}
	if b.settingsRepo != nil {
		ctx := context.Background()
		if settings, err := b.settingsRepo.GetCompanySettings(ctx); err == nil && settings != nil {
			hints.LegalFormNote = domain.LegalFormLimitationNote(settings.LegalForm)
		}
		if date, err := b.settingsRepo.Get(ctx, domain.SettingSystemChangeDate); err == nil {
			hints.SystemChangeDate = date
		}
	}
	if hints.SystemChangeDate != "" && b.retentionSvc != nil {
		if overview, err := b.retentionSvc.Overview(context.Background(), ""); err == nil {
			hints.SystemChangeNote = overview.SystemChangeNote
		}
	}
	return hints, nil
}

// --- Kontakte -------------------------------------------------------------

// BlockContactResult ist das Ergebnis einer Sperre nach einem Löschverlangen.
type BlockContactResult struct {
	Contact *domain.Contact `json:"contact"`
	// Answer ist der Antworttext an die betroffene Person, mit den Normen, auf
	// die sich die Ablehnung des Löschverlangens stützt.
	Answer string `json:"answer"`
}

// BlockContact sperrt einen Geschäftspartner nach einem Löschverlangen.
func (b *BuchfinkBridge) BlockContact(id uint, reason string) (*BlockContactResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.contactSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	contact, answer, err := b.contactSvc.BlockContact(context.Background(), id, reason)
	if err != nil {
		return nil, err
	}
	return &BlockContactResult{Contact: contact, Answer: answer}, nil
}

// GetSelectableContacts liefert die Geschäftspartner, die sich noch auswählen
// lassen — gesperrte fehlen.
func (b *BuchfinkBridge) GetSelectableContacts() ([]domain.Contact, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.contactSvc == nil {
		return []domain.Contact{}, nil
	}
	return emptyList(b.contactSvc.SelectableContacts(context.Background()))
}

// --- Belege ---------------------------------------------------------------

// SaveReceiptHeader schreibt die Kopfdaten eines abgelegten Belegs nach.
func (b *BuchfinkBridge) SaveReceiptHeader(receiptID uint, header domain.ReceiptHeader) (*domain.Receipt, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.receiptSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.receiptSvc.SaveHeader(context.Background(), receiptID, header)
}

// --- Journal --------------------------------------------------------------

// CorrectEntry storniert eine Buchung und bucht sie richtig neu.
func (b *BuchfinkBridge) CorrectEntry(entryID uint, reason string, newEntry *domain.JournalEntry) (*service.CorrectionResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.journalSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.journalSvc.CorrectEntry(context.Background(), entryID, reason, newEntry)
}

// GetCorrectionOf liefert die Neubuchung, die eine stornierte Buchung ersetzt,
// oder nil.
func (b *BuchfinkBridge) GetCorrectionOf(entryID uint) (*domain.JournalEntry, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.journalSvc == nil {
		return nil, nil
	}
	entry, err := b.journalSvc.CorrectionOf(context.Background(), entryID)
	entry.EnsureLists()
	return entry, err
}

// PreviewOpeningBalance zeigt die Eröffnungsbuchungen des Umsteigers, ohne sie
// zu schreiben.
func (b *BuchfinkBridge) PreviewOpeningBalance(req service.OpeningBalanceRequest) (*service.OpeningBalancePreview, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.journalSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.journalSvc.PreviewOpeningBalance(context.Background(), req)
}

// BookOpeningBalance bucht die Eröffnungsbilanz des Umsteigers.
func (b *BuchfinkBridge) BookOpeningBalance(req service.OpeningBalanceRequest) (*service.OpeningBalancePreview, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.journalSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.journalSvc.BookOpeningBalance(context.Background(), req)
}

// --- Offene Posten --------------------------------------------------------

// GetOpenItemsAging liefert Altersstruktur und Restlaufzeiten der offenen
// Posten zum Stichtag. Leerer Stichtag heißt: heute.
func (b *BuchfinkBridge) GetOpenItemsAging(cutoff string) (*domain.OpenItemsAging, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.paymentSvc == nil {
		empty := &domain.OpenItemsAging{Cutoff: cutoff}
		empty.EnsureLists()
		return empty, nil
	}
	if cutoff == "" {
		cutoff = time.Now().UTC().Format("2006-01-02")
	}
	items, err := b.paymentSvc.OpenItemsAt(context.Background(), cutoff)
	if err != nil {
		return nil, err
	}
	aging := accounting.AgeOpenItems(items, cutoff)
	// Die Zusicherung steht hier und nicht nur bei der leeren Antwort: eine
	// Liste, die an einer Stelle `null` sein darf, ist keine Zusicherung.
	aging.EnsureLists()
	return &aging, nil
}

// --- Hilfen ---------------------------------------------------------------

// archiveForDeletion schreibt den Archivexport, der der Löschung vorausgeht.
//
// In den Datenordner unter „archiv/<Jahr>" und nicht an einen vom Anwender
// gewählten Ort: der Ordner wird mitgesichert, und ein Archiv, das nur auf
// einem Wechseldatenträger liegt, ist beim nächsten Rechner weg. Wer es
// woanders haben will, kopiert es dorthin — der Pfad steht im Protokoll.
func (b *BuchfinkBridge) archiveForDeletion(year int) (string, error) {
	if b.exportSvc == nil {
		return "", fmt.Errorf("ohne Export lässt sich vor dem Löschen kein Archiv erstellen")
	}
	targetDir := filepath.Join(b.dataDir, "archiv", fmt.Sprintf("%d", year))
	result, err := b.exportSvc.ExportArchive(context.Background(), year, targetDir)
	if err != nil {
		return "", fmt.Errorf("das Archiv des Geschäftsjahres %d konnte nicht erstellt werden, deshalb wurde nicht gelöscht: %w", year, err)
	}
	return result.Dir, nil
}

// backupRhythmLabel beschreibt den Sicherungsrhythmus für die
// Verfahrensdokumentation.
//
// Buchfink sichert beim Beenden und beim Start, wenn die letzte gelungene
// Sicherung länger als einen Tag her ist. Das ist der Rhythmus, und er steht
// hier als Satz, weil eine Verfahrensdokumentation ihn beschreiben muss und
// nicht auf eine Einstellung verweisen kann, die es nicht gibt.
func (b *BuchfinkBridge) backupRhythmLabel() string {
	if b.appConfig.BackupDir == "" {
		return "nicht eingerichtet — es wird nicht gesichert"
	}
	return "täglich, ausgelöst beim Beenden und beim Start des Programms, wenn die letzte erfolgreiche Sicherung länger als einen Tag zurückliegt"
}
