package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/buildinfo"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/receiptstore"
)

// RetentionYear ist die Fristenlage eines Geschäftsjahres.
type RetentionYear struct {
	FiscalYear int                    `json:"fiscalYear"`
	Counts     domain.RetentionCounts `json:"counts"`
	// Classes sind die Fristen der Objektarten dieses Jahres, je Klasse eine
	// Zeile. Ein Jahr hat mehrere Fristen: die Bücher zehn Jahre, die Belege
	// acht, die Handelsbriefe sechs — eine einzige Zahl je Jahr wäre falsch.
	Classes []RetentionClassRow `json:"classes"`
	// EarliestDeletion ist der erste Tag, an dem das ganze Jahr gelöscht werden
	// darf: das Maximum über alle Klassen. Es gilt die längste Frist, weil das
	// Löschen das ganze Jahr trifft.
	EarliestDeletion string `json:"earliestDeletion"`
	Expired          bool   `json:"expired"`
	// Hold ist die geltende Aussetzung oder nil.
	Hold *domain.RetentionHold `json:"hold,omitempty"`
	// Deletable meldet, ob das Jahr am Stichtag gelöscht werden dürfte.
	Deletable bool   `json:"deletable"`
	Note      string `json:"note"`
}

// RetentionClassRow ist die Frist einer Aufbewahrungsklasse innerhalb eines
// Jahres.
type RetentionClassRow struct {
	Class            domain.RetentionClass `json:"class"`
	Label            string                `json:"label"`
	Years            int                   `json:"years"`
	RetentionEnd     string                `json:"retentionEnd"`
	EarliestDeletion string                `json:"earliestDeletion"`
	LegalBasis       string                `json:"legalBasis"`
}

// RetentionOverview ist die Fristenübersicht über alle Geschäftsjahre.
type RetentionOverview struct {
	Today string          `json:"today"`
	Years []RetentionYear `json:"years"`
	// Concept ist das Löschkonzept: Datenkategorie, Frist, Rechtsgrundlage.
	Concept []accounting.DeletionConceptRow `json:"concept"`
	// SystemChangeDate und SystemChangeNote sind der Umstellungszeitpunkt bei
	// der Übernahme aus einem Altsystem und die Fünfjahresfrist daraus.
	SystemChangeDate string `json:"systemChangeDate,omitempty"`
	SystemChangeNote string `json:"systemChangeNote,omitempty"`
}

// EnsureLists ersetzt nicht belegte Listen durch leere.
func (o *RetentionOverview) EnsureLists() {
	if o.Years == nil {
		o.Years = make([]RetentionYear, 0)
	}
	if o.Concept == nil {
		o.Concept = make([]accounting.DeletionConceptRow, 0)
	}
	for i := range o.Years {
		if o.Years[i].Classes == nil {
			o.Years[i].Classes = make([]RetentionClassRow, 0)
		}
	}
}

// RetentionService führt die Aufbewahrungsfristen, ihre Aussetzung und die
// Löschung abgelaufener Jahrgänge.
//
// Gelöscht wird nie von selbst. Eine Buchführung, die sich selbst aufräumt,
// löscht irgendwann etwas, das noch gebraucht wird — und § 147 Abs. 3 Satz 5 AO
// nennt vier Gründe, aus denen eine Frist nicht abläuft, die alle außerhalb des
// Programms liegen. Der Dienst sagt deshalb, was abgelaufen ist, und führt die
// Löschung nur auf ausdrückliche Anweisung aus.
type RetentionService struct {
	retentionRepo domain.RetentionRepository
	journalRepo   domain.JournalRepository
	settingsRepo  domain.SettingsRepository
	auditRepo     domain.AuditRepository
	store         *receiptstore.Store
}

// NewRetentionService verdrahtet die Fristenverwaltung.
func NewRetentionService(
	retentionRepo domain.RetentionRepository,
	journalRepo domain.JournalRepository,
	settingsRepo domain.SettingsRepository,
	auditRepo domain.AuditRepository,
	store *receiptstore.Store,
) *RetentionService {
	return &RetentionService{
		retentionRepo: retentionRepo,
		journalRepo:   journalRepo,
		settingsRepo:  settingsRepo,
		auditRepo:     auditRepo,
		store:         store,
	}
}

// Overview liefert die Fristenlage aller Geschäftsjahre zum Stichtag.
//
// today wird übergeben und nicht aus der Uhr genommen: eine Frist, die sich
// beim Testen nicht auf einen Tag festlegen lässt, ist eine Frist, die nicht
// geprüft wird. Leer heißt: heute.
func (s *RetentionService) Overview(ctx context.Context, today string) (*RetentionOverview, error) {
	if today == "" {
		today = time.Now().UTC().Format("2006-01-02")
	}
	overview := &RetentionOverview{
		Today:   today,
		Concept: accounting.DeletionConcept(),
	}

	years, err := s.journalRepo.GetAvailableFiscalYears(ctx)
	if err != nil {
		return nil, fmt.Errorf("die Geschäftsjahre konnten nicht gelesen werden: %w", err)
	}
	sort.Ints(years)

	for _, year := range years {
		row, err := s.yearRow(ctx, year, today)
		if err != nil {
			return nil, err
		}
		overview.Years = append(overview.Years, row)
	}

	if s.settingsRepo != nil {
		if date, err := s.settingsRepo.Get(ctx, domain.SettingSystemChangeDate); err == nil && date != "" {
			overview.SystemChangeDate = date
			overview.SystemChangeNote = systemChangeNote(date, today)
		}
	}

	overview.EnsureLists()
	return overview, nil
}

// yearRow bildet die Fristenlage eines einzelnen Jahres.
func (s *RetentionService) yearRow(ctx context.Context, year int, today string) (RetentionYear, error) {
	row := RetentionYear{FiscalYear: year}

	counts, err := s.retentionRepo.CountObjects(ctx, year)
	if err != nil {
		return row, err
	}
	row.Counts = counts

	// Die Klassen eines Jahres: Bücher (Journal, Abschlüsse, Festschreibungen),
	// Belege und Handelsbriefe. Alle drei stehen da, auch wo das Jahr keine
	// Objekte der Klasse trägt — die Frist ist eine Eigenschaft des Jahres und
	// keine Zählung.
	for _, kind := range []domain.RetentionKind{
		domain.RetentionKindJournal,
		domain.RetentionKindReceiptInvoice,
		domain.RetentionKindReceiptLetter,
	} {
		info := accounting.RetentionFor(kind, year)
		row.Classes = append(row.Classes, RetentionClassRow{
			Class:            info.Class,
			Label:            info.Class.Label(),
			Years:            info.Years,
			RetentionEnd:     info.RetentionEnd,
			EarliestDeletion: info.EarliestDeletion,
			LegalBasis:       info.LegalBasis,
		})
		if info.EarliestDeletion > row.EarliestDeletion {
			row.EarliestDeletion = info.EarliestDeletion
		}
	}
	row.Expired = row.EarliestDeletion != "" && today >= row.EarliestDeletion

	hold, err := s.retentionRepo.FindActiveHold(ctx, year)
	if err != nil {
		return row, err
	}
	row.Hold = hold
	row.Deletable = row.Expired && hold == nil

	switch {
	case hold != nil:
		row.Note = fmt.Sprintf(
			"Die Aufbewahrungsfrist ist seit dem %s ausgesetzt (%s). Sie läuft nicht ab, solange die Unterlagen für das Verfahren von Bedeutung sind (§ 147 Abs. 3 Satz 5 AO).",
			hold.SetAt.Format("02.01.2006"), hold.Reason.Label())
	case row.Expired:
		row.Note = fmt.Sprintf(
			"Die Aufbewahrungsfrist ist am %s abgelaufen. Das Geschäftsjahr darf archiviert und gelöscht werden.",
			row.EarliestDeletion)
	default:
		row.Note = fmt.Sprintf("Aufzubewahren bis zum %s.", row.EarliestDeletion)
	}
	return row, nil
}

// systemChangeNote beschreibt die Fünfjahresfrist des § 147 Abs. 6 Satz 6 AO.
func systemChangeNote(changeDate, today string) string {
	end := accounting.AddYearsISO(changeDate, 5)
	if end == "" {
		return ""
	}
	if today > end {
		return fmt.Sprintf(
			"Die Umstellung aus dem Altsystem war am %s. Die Fünfjahresfrist des § 147 Abs. 6 Satz 6 AO ist am %s abgelaufen; das Altsystem muss für den Datenzugriff nicht mehr verfügbar gehalten werden.",
			changeDate, end)
	}
	return fmt.Sprintf(
		"Die Umstellung aus dem Altsystem war am %s. Bis zum %s ist das Altsystem für den Datenzugriff verfügbar zu halten (§ 147 Abs. 6 Satz 6 AO).",
		changeDate, end)
}

// SetHold setzt die Aufbewahrungsfrist eines Geschäftsjahres aus.
func (s *RetentionService) SetHold(
	ctx context.Context, year int, reason domain.RetentionHoldReason, description string,
) (*domain.RetentionHold, error) {
	counts, err := s.retentionRepo.CountObjects(ctx, year)
	if err != nil {
		return nil, err
	}
	hold := &domain.RetentionHold{
		FiscalYear:  year,
		Reason:      reason,
		Description: strings.TrimSpace(description),
		SetAt:       time.Now().UTC(),
		SetBy:       actor.Actor(),
		AppVersion:  buildinfo.Version,
		AffectedNote: fmt.Sprintf("%d Buchungen, %d Belege, %d Dateien",
			counts.JournalEntries, counts.Receipts, counts.ReceiptFiles),
	}
	if err := s.retentionRepo.CreateHold(ctx, hold); err != nil {
		return nil, err
	}
	if s.auditRepo != nil {
		_ = s.auditRepo.LogChange(ctx, domain.AuditActionUpdate, "RETENTION_HOLD",
			fmt.Sprintf("%d", hold.ID),
			fmt.Sprintf("Aufbewahrungsfrist für %d ausgesetzt (%s): %s. Betroffen: %s.",
				year, hold.Reason.Label(), hold.Description, hold.AffectedNote),
			nil, hold)
	}
	return hold, nil
}

// ReleaseHold hebt eine Aussetzung auf.
func (s *RetentionService) ReleaseHold(ctx context.Context, id uint, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("zum Aufheben gehört ein Grund — er ist der Nachweis, dass das Verfahren beendet ist")
	}
	if err := s.retentionRepo.ReleaseHold(ctx, id, actor.Actor(), strings.TrimSpace(reason), time.Now().UTC()); err != nil {
		return err
	}
	if s.auditRepo != nil {
		_ = s.auditRepo.Log(ctx, domain.AuditActionUpdate, "RETENTION_HOLD", fmt.Sprintf("%d", id),
			fmt.Sprintf("Aussetzung der Aufbewahrungsfrist aufgehoben: %s", strings.TrimSpace(reason)))
	}
	return nil
}

// Holds liefert alle Aussetzungen, die geltenden zuerst.
func (s *RetentionService) Holds(ctx context.Context) ([]domain.RetentionHold, error) {
	return s.retentionRepo.FindHolds(ctx)
}

// ExpiredYears sind die Geschäftsjahre, deren Frist abgelaufen ist und für die
// keine Aussetzung gilt.
func (s *RetentionService) ExpiredYears(ctx context.Context, today string) ([]RetentionYear, error) {
	overview, err := s.Overview(ctx, today)
	if err != nil {
		return nil, err
	}
	out := make([]RetentionYear, 0)
	for _, year := range overview.Years {
		if year.Expired {
			out = append(out, year)
		}
	}
	return out, nil
}

// EnsureDeletable prüft, ob ein Geschäftsjahr gelöscht werden darf.
//
// Die Prüfung ist getrennt von der Löschung, weil die Oberfläche sie vor der
// Bestätigung stellen muss: ein Knopf, der erst nach dem zweiten Bestätigen
// sagt, dass er nicht darf, ist ein schlechter Knopf.
func (s *RetentionService) EnsureDeletable(ctx context.Context, year int, today string) error {
	if today == "" {
		today = time.Now().UTC().Format("2006-01-02")
	}
	row, err := s.yearRow(ctx, year, today)
	if err != nil {
		return err
	}
	if !row.Expired {
		return fmt.Errorf(
			"das Geschäftsjahr %d ist bis zum %s aufzubewahren und lässt sich vorher nicht löschen (§ 257 Abs. 4 HGB, § 147 Abs. 3 AO)",
			year, row.EarliestDeletion)
	}
	if row.Hold != nil {
		return fmt.Errorf(
			"für das Geschäftsjahr %d ist die Aufbewahrungsfrist seit dem %s ausgesetzt (%s). Solange die Unterlagen für das Verfahren von Bedeutung sind, läuft die Frist nicht ab (§ 147 Abs. 3 Satz 5 AO)",
			year, row.Hold.SetAt.Format("02.01.2006"), row.Hold.Reason.Label())
	}
	return nil
}

// DeleteRequest ist der Auftrag, ein Geschäftsjahr zu archivieren und zu
// löschen.
type DeleteRequest struct {
	FiscalYear int `json:"fiscalYear"`
	// Confirmation muss die Jahreszahl als Text tragen. Eine Bestätigung, die
	// sich mit einem Klick geben lässt, ist bei einem unumkehrbaren Vorgang
	// keine.
	Confirmation string `json:"confirmation"`
	// ArchivePath ist der Archivexport, der vor der Löschung erstellt wurde.
	// Ohne ihn wird nicht gelöscht: die Aufbewahrungsfrist endet, die Pflicht
	// zur Nachvollziehbarkeit einer einmal geführten Buchführung endet damit
	// nicht automatisch — und ein Archiv kostet nichts.
	ArchivePath string `json:"archivePath"`
	// Today ist der Stichtag; leer heißt heute.
	Today string `json:"today,omitempty"`
}

// DeleteResult ist das Ergebnis der Löschung.
type DeleteResult struct {
	FiscalYear   int                    `json:"fiscalYear"`
	Deleted      domain.RetentionCounts `json:"deleted"`
	FilesRemoved int                    `json:"filesRemoved"`
	FilesKept    int                    `json:"filesKept"`
	ArchivePath  string                 `json:"archivePath"`
	Message      string                 `json:"message"`
}

// ArchiveAndDelete löscht ein Geschäftsjahr, dessen Frist abgelaufen ist.
//
// Der Vorgang ist unumkehrbar, und deshalb steht vor ihm alles, was sich davor
// stellen lässt: die Fristprüfung, die Prüfung auf eine Aussetzung, die
// ausgeschriebene Bestätigung und der Archivexport. Der Protokolleintrag wird
// zuletzt geschrieben und bleibt — er ist das, was von dem Jahr übrig bleibt.
func (s *RetentionService) ArchiveAndDelete(ctx context.Context, req DeleteRequest) (*DeleteResult, error) {
	if strings.TrimSpace(req.Confirmation) != fmt.Sprintf("%d", req.FiscalYear) {
		return nil, fmt.Errorf(
			"zum Löschen des Geschäftsjahres %d ist die Jahreszahl als Bestätigung einzugeben. Der Vorgang lässt sich nicht rückgängig machen",
			req.FiscalYear)
	}
	if strings.TrimSpace(req.ArchivePath) == "" {
		return nil, fmt.Errorf(
			"vor dem Löschen ist ein Archivexport des Geschäftsjahres %d zu erstellen. Ohne ihn wäre die Buchführung dieses Jahres danach nicht mehr nachvollziehbar",
			req.FiscalYear)
	}
	if err := s.EnsureDeletable(ctx, req.FiscalYear, req.Today); err != nil {
		return nil, err
	}

	deleted, orphans, err := s.retentionRepo.DeleteFiscalYear(ctx, req.FiscalYear)
	if err != nil {
		return nil, err
	}

	result := &DeleteResult{
		FiscalYear:  req.FiscalYear,
		Deleted:     deleted,
		ArchivePath: req.ArchivePath,
		FilesKept:   deleted.ReceiptFiles - len(orphans),
	}
	// Die Dateien werden erst nach der Datenbanktransaktion entfernt: eine
	// gelöschte Datei zu einem noch vorhandenen Beleg wäre der schlechtere
	// Zwischenzustand als ein verwaister Dateirest.
	if s.store != nil {
		for _, path := range orphans {
			if err := s.store.Delete(path); err == nil {
				result.FilesRemoved++
			}
		}
	}

	// Das Protokoll nennt jede Objektart einzeln. Es ist das, was von dem Jahr
	// übrig bleibt, und muss deshalb auch das nennen, was neben Journal und
	// Belegen verschwunden ist — Rechnungen, Bankumsätze, Anlagenbewegungen —
	// und die Verweise, die aus überdauernden Objekten genullt werden mussten.
	result.Message = fmt.Sprintf(
		"Geschäftsjahr %d gelöscht: %d Buchungen mit %d Zeilen, %d Belege mit %d Dateien, "+
			"%d Festschreibungen, %d Prüfläufe, %d Voranmeldungen, %d Zusammenfassende Meldungen, "+
			"%d Rechnungen, %d Bankumsätze, %d Anlagenbewegungen, %d Abgrenzungen, %d Rückstellungen, "+
			"%d Inventurwerte, %d Nummernlücken, %d Verwendungsanteile nach § 15a UStG. "+
			"%d Verweise überdauernder Objekte auf gelöschte Buchungen und Belege wurden aufgelöst. "+
			"%d Dateien von der Platte entfernt, %d bleiben, weil andere Belege auf sie zeigen. Archiv: %s.",
		req.FiscalYear, deleted.JournalEntries, deleted.JournalLines,
		deleted.Receipts, deleted.ReceiptFiles, deleted.Festschreibungen,
		deleted.CheckRuns, deleted.VatReturns, deleted.ZMReturns,
		deleted.Invoices, deleted.BankTransactions, deleted.AssetMovements,
		deleted.Accruals, deleted.Provisions, deleted.InventoryCounts,
		deleted.NumberGaps, deleted.InputTaxUsages,
		deleted.ClearedReferences,
		result.FilesRemoved, result.FilesKept, req.ArchivePath)

	if s.auditRepo != nil {
		_ = s.auditRepo.Log(ctx, domain.AuditActionUpdate, "FISCAL_YEAR_DELETED",
			fmt.Sprintf("%d", req.FiscalYear), result.Message)
	}
	return result, nil
}
