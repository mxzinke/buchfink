package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

// --- Aussetzung der Aufbewahrungsfrist ------------------------------------

type retentionRepositoryGorm struct {
	db *gorm.DB
}

// NewRetentionRepository liefert die Aussetzungen der Aufbewahrungsfristen.
func NewRetentionRepository(db *gorm.DB) domain.RetentionRepository {
	return &retentionRepositoryGorm{db: db}
}

func (r *retentionRepositoryGorm) CreateHold(ctx context.Context, hold *domain.RetentionHold) error {
	if err := hold.Validate(); err != nil {
		return err
	}
	if hold.SetAt.IsZero() {
		hold.SetAt = time.Now().UTC()
	}
	// Zwei geltende Aussetzungen für dasselbe Jahr wären nicht falsch, aber
	// unbrauchbar: welche gilt, wäre nicht mehr zu beantworten, und die
	// Aufhebung der einen sähe aus wie das Ende der Sperre.
	existing, err := r.FindActiveHold(ctx, hold.FiscalYear)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf(
			"für %d ist die Aufbewahrungsfrist bereits seit dem %s ausgesetzt (%s). Hebe die bestehende Aussetzung auf, bevor du eine neue einträgst",
			hold.FiscalYear, existing.SetAt.Format("02.01.2006"), existing.Reason.Label())
	}
	return dbFrom(ctx, r.db).Create(hold).Error
}

func (r *retentionRepositoryGorm) ReleaseHold(ctx context.Context, id uint, releasedBy, reason string, at time.Time) error {
	var hold domain.RetentionHold
	db := dbFrom(ctx, r.db)
	if err := db.First(&hold, id).Error; err != nil {
		return fmt.Errorf("die Aussetzung wurde nicht gefunden: %w", err)
	}
	if !hold.IsActive() {
		return fmt.Errorf("die Aussetzung für %d ist bereits am %s aufgehoben worden",
			hold.FiscalYear, hold.ReleasedAt.Format("02.01.2006"))
	}
	released := at.UTC()
	hold.ReleasedAt = &released
	hold.ReleasedBy = releasedBy
	hold.ReleaseReason = reason
	// Über den Datensatz und eine Spaltenauswahl, damit der Serializer den
	// verschlüsselten Aufhebungsgrund verschlüsselt.
	return db.Model(&hold).Select("ReleasedAt", "ReleasedBy", "ReleaseReason").Updates(&hold).Error
}

func (r *retentionRepositoryGorm) FindHolds(ctx context.Context) ([]domain.RetentionHold, error) {
	holds := make([]domain.RetentionHold, 0)
	err := dbFrom(ctx, r.db).Order("fiscal_year desc, id desc").Find(&holds).Error
	return holds, err
}

func (r *retentionRepositoryGorm) FindActiveHold(ctx context.Context, fiscalYear int) (*domain.RetentionHold, error) {
	var hold domain.RetentionHold
	err := dbFrom(ctx, r.db).
		Where("fiscal_year = ? AND released_at IS NULL", fiscalYear).
		Order("id desc").First(&hold).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &hold, nil
}

// --- Fassungen der Verfahrensdokumentation --------------------------------

type procDocRepositoryGorm struct {
	db *gorm.DB
}

// NewProcedureDocumentationRepository liefert die Fassungen der
// Verfahrensdokumentation.
func NewProcedureDocumentationRepository(db *gorm.DB) domain.ProcedureDocumentationRepository {
	return &procDocRepositoryGorm{db: db}
}

func (r *procDocRepositoryGorm) Create(ctx context.Context, doc *domain.ProcedureDocumentation) error {
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now().UTC()
	}
	return dbFrom(ctx, r.db).Create(doc).Error
}

func (r *procDocRepositoryGorm) FindAll(ctx context.Context) ([]domain.ProcedureDocumentation, error) {
	docs := make([]domain.ProcedureDocumentation, 0)
	err := dbFrom(ctx, r.db).Order("id desc").Find(&docs).Error
	return docs, err
}

func (r *procDocRepositoryGorm) CountForDay(ctx context.Context, day string) (int64, error) {
	var count int64
	// Gezählt wird über die Fassungsbezeichnung und nicht über den Zeitpunkt:
	// die Bezeichnung beginnt mit dem Tag, und ein LIKE darauf trifft genau
	// die Fassungen, deren Nummer an diesem Tag vergeben wurde — auch dann,
	// wenn die Datei später aus einer anderen Zeitzone gelesen wird.
	err := dbFrom(ctx, r.db).Model(&domain.ProcedureDocumentation{}).
		Where("version LIKE ?", day+"%").Count(&count).Error
	return count, err
}

// CountObjects zählt, was ein Geschäftsjahr an aufzubewahrenden Daten trägt.
func (r *retentionRepositoryGorm) CountObjects(ctx context.Context, fiscalYear int) (domain.RetentionCounts, error) {
	var counts domain.RetentionCounts
	db := dbFrom(ctx, r.db)

	countWhere := func(model any, where string, args []any, into *int) error {
		var n int64
		q := db.Model(model)
		if where != "" {
			q = q.Where(where, args...)
		}
		if err := q.Count(&n).Error; err != nil {
			return err
		}
		*into = int(n)
		return nil
	}

	year := []any{fiscalYear}
	for _, item := range []struct {
		model any
		where string
		args  []any
		into  *int
	}{
		{&domain.JournalEntry{}, "fiscal_year = ?", year, &counts.JournalEntries},
		{&domain.Receipt{}, "fiscal_year = ?", year, &counts.Receipts},
		{&domain.Festschreibung{}, "fiscal_year = ?", year, &counts.Festschreibungen},
		{&domain.CheckRun{}, "fiscal_year = ?", year, &counts.CheckRuns},
		{&domain.VatReturn{}, "fiscal_year = ?", year, &counts.VatReturns},
		{&domain.Invoice{}, "fiscal_year = ?", year, &counts.Invoices},
		{&domain.BankTransaction{}, "fiscal_year = ?", year, &counts.BankTransactions},
		{&domain.AssetMovement{}, "fiscal_year = ?", year, &counts.AssetMovements},
		{&domain.Accrual{}, "fiscal_year = ?", year, &counts.Accruals},
		{&domain.Provision{}, "fiscal_year = ?", year, &counts.Provisions},
		{&domain.InventoryCount{}, "fiscal_year = ?", year, &counts.InventoryCounts},
		{&domain.ZMReturn{}, "fiscal_year = ?", year, &counts.ZMReturns},
		{&domain.InputTaxUsage{}, "fiscal_year = ?", year, &counts.InputTaxUsages},
		{&domain.NumberGap{}, "fiscal_year = ?", year, &counts.NumberGaps},
	} {
		if err := countWhere(item.model, item.where, item.args, item.into); err != nil {
			return counts, fmt.Errorf("die Objekte des Geschäftsjahres %d ließen sich nicht zählen: %w", fiscalYear, err)
		}
	}

	// Zeilen und Dateien hängen an ihren Köpfen und werden über sie gezählt.
	entryIDs := db.Model(&domain.JournalEntry{}).Select("id").Where("fiscal_year = ?", fiscalYear)
	receiptIDs := db.Model(&domain.Receipt{}).Select("id").Where("fiscal_year = ?", fiscalYear)
	if err := countWhere(&domain.JournalLine{}, "entry_id IN (?)", []any{entryIDs},
		&counts.JournalLines); err != nil {
		return counts, err
	}
	if err := countWhere(&domain.ReceiptFile{}, "receipt_id IN (?)", []any{receiptIDs},
		&counts.ReceiptFiles); err != nil {
		return counts, err
	}
	return counts, nil
}

// DeleteFiscalYear löscht die Daten eines Geschäftsjahres.
//
// In einer Transaktion, weil eine halb gelöschte Buchführung schlimmer ist als
// eine ungelöschte: eine Buchung ohne Zeilen oder ein Beleg ohne Dateien wäre
// ein Datensatz, der eine Vollständigkeit behauptet, die er nicht hat.
//
// Die Prüfung, ob überhaupt gelöscht werden darf — Frist abgelaufen, keine
// Aussetzung, Archivexport erstellt —, steht nicht hier, sondern im
// RetentionService. Das Repository führt aus, was entschieden wurde.
func (r *retentionRepositoryGorm) DeleteFiscalYear(ctx context.Context, fiscalYear int) (domain.RetentionCounts, []string, error) {
	counts, err := r.CountObjects(ctx, fiscalYear)
	if err != nil {
		return counts, nil, err
	}

	orphans := make([]string, 0)
	var cleared int
	err = dbFrom(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		// Die verwaisten Dateien werden vor dem Löschen bestimmt: danach ist
		// nicht mehr festzustellen, welche Prüfsumme zu diesem Jahr gehörte.
		//
		// Gelesen wird über den Datensatz, weil der Ablagepfad verschlüsselt in
		// der Spalte liegt und ein Rohwert der Geheimtext wäre.
		var files []domain.ReceiptFile
		yearReceipts := tx.Model(&domain.Receipt{}).Select("id").Where("fiscal_year = ?", fiscalYear)
		if err := tx.Where("receipt_id IN (?)", yearReceipts).
			Find(&files).Error; err != nil {
			return err
		}
		for i := range files {
			var stillUsed int64
			sameYear := tx.Model(&domain.Receipt{}).Select("id").Where("fiscal_year = ?", fiscalYear)
			if err := tx.Model(&domain.ReceiptFile{}).
				Where("sha256 = ?", files[i].SHA256).
				Where("receipt_id NOT IN (?)", sameYear).
				Count(&stillUsed).Error; err != nil {
				return err
			}
			// Nur was kein anderer Beleg mehr braucht: der Speicher ist
			// inhaltsadressiert, und dieselbe Datei kann an zwei Belegen hängen.
			if stillUsed == 0 && files[i].StoredPath != "" {
				orphans = append(orphans, files[i].StoredPath)
			}
		}

		// Gelöscht wird über die Modelle und nicht über Tabellennamen: den
		// Namen bildet GORM, und eine getippte Zeichenkette daneben ginge beim
		// nächsten Umbenennen still ins Leere.
		//
		// Die Reihenfolge ist die der Abhängigkeiten: erst die Kinder, dann die
		// Köpfe. Ein verwaistes Kind wäre ein Datensatz, der auf nichts mehr
		// zeigt und in keiner Auswertung mehr auftaucht.
		entryIDs := tx.Model(&domain.JournalEntry{}).Select("id").Where("fiscal_year = ?", fiscalYear)
		receiptIDs := tx.Model(&domain.Receipt{}).Select("id").Where("fiscal_year = ?", fiscalYear)
		checkRunIDs := tx.Model(&domain.CheckRun{}).Select("id").Where("fiscal_year = ?", fiscalYear)
		invoiceIDs := tx.Model(&domain.Invoice{}).Select("id").Where("fiscal_year = ?", fiscalYear)
		groupIDs := tx.Model(&domain.InvoiceGroup{}).Select("id").Where("fiscal_year = ?", fiscalYear)
		zmIDs := tx.Model(&domain.ZMReturn{}).Select("id").Where("fiscal_year = ?", fiscalYear)
		accrualIDs := tx.Model(&domain.Accrual{}).Select("id").Where("fiscal_year = ?", fiscalYear)
		provisionIDs := tx.Model(&domain.Provision{}).Select("id").Where("fiscal_year = ?", fiscalYear)
		// Das Verzeichnis nach § 15a UStG läuft über mehrere Jahre. Gelöscht
		// wird nur, was in diesem Jahr entstanden ist *und* mit ihm endet: ein
		// Eintrag, dessen Berichtigungszeitraum weiterläuft, wird noch
		// gebraucht, und einer aus einem früheren Jahr gehört zu dessen Frist
		// und nicht zu dieser. Was bleibt, verliert unten nur seine Verweise.
		endedCorrectionIDs := tx.Model(&domain.InputTaxCorrection{}).Select("id").
			Where("first_fiscal_year = ? AND last_fiscal_year <= ?", fiscalYear, fiscalYear)

		deletes := []struct {
			model any
			where string
			args  []any
		}{
			// Die Rechnungen des Jahres mit allem, was an ihnen hängt.
			{&domain.SupplyEvidence{}, "invoice_id IN (?)", []any{invoiceIDs}},
			{&domain.InvoiceItem{}, "invoice_id IN (?)", []any{invoiceIDs}},
			{&domain.InvoiceReference{}, "invoice_id IN (?)", []any{invoiceIDs}},
			{&domain.AdvanceItem{}, "invoice_id IN (?) OR group_id IN (?)", []any{invoiceIDs, groupIDs}},
			{&domain.Invoice{}, "fiscal_year = ?", []any{fiscalYear}},
			{&domain.InvoiceGroup{}, "fiscal_year = ?", []any{fiscalYear}},
			// Anzahlungen an Lieferanten hängen an Beleg und Buchung: ohne
			// beide wäre der Datensatz eine Behauptung über eine Zahlung, die
			// nicht mehr nachzuweisen ist.
			{&domain.VendorAdvance{}, "entry_id IN (?) OR receipt_id IN (?)", []any{entryIDs, receiptIDs}},
			// Der Ausgleich verbindet zwei Buchungen. Fällt eine, ist er
			// gegenstandslos — und ein bezahlter offener Posten lebte sonst
			// wieder auf, weil seine Zahlungszuordnung ins Leere zeigte.
			{&domain.PaymentAllocation{}, "open_item_entry_id IN (?) OR payment_entry_id IN (?)",
				[]any{entryIDs, entryIDs}},
			{&domain.BankTransaction{}, "fiscal_year = ?", []any{fiscalYear}},
			{&domain.AssetMovement{}, "fiscal_year = ?", []any{fiscalYear}},
			{&domain.AccrualRelease{}, "accrual_id IN (?) OR fiscal_year = ?", []any{accrualIDs, fiscalYear}},
			{&domain.Accrual{}, "fiscal_year = ?", []any{fiscalYear}},
			{&domain.ProvisionMovement{}, "provision_id IN (?) OR fiscal_year = ?", []any{provisionIDs, fiscalYear}},
			{&domain.Provision{}, "fiscal_year = ?", []any{fiscalYear}},
			{&domain.InventoryCount{}, "fiscal_year = ?", []any{fiscalYear}},
			{&domain.Appropriation{}, "year = ?", []any{fiscalYear}},
			{&domain.ZMLine{}, "zm_return_id IN (?)", []any{zmIDs}},
			{&domain.ZMReturn{}, "fiscal_year = ?", []any{fiscalYear}},
			{&domain.InputTaxUsage{}, "correction_id IN (?) OR fiscal_year = ?",
				[]any{endedCorrectionIDs, fiscalYear}},
			{&domain.InputTaxCorrection{}, "first_fiscal_year = ? AND last_fiscal_year <= ?",
				[]any{fiscalYear, fiscalYear}},
			// Nummernkreise und Lückenvermerke des Jahres: der Kreis ist
			// jahresbezogen, und ein Vermerk über eine Lücke in einer nicht
			// mehr vorhandenen Nummernfolge erklärt nichts mehr.
			{&domain.NumberGap{}, "fiscal_year = ?", []any{fiscalYear}},
			{&domain.NumberRange{}, "fiscal_year = ?", []any{fiscalYear}},
			// Der Abschluss selbst: Bausteine und Anhangtexte des Jahres.
			{&domain.ClosingStep{}, "year = ?", []any{fiscalYear}},
			{&domain.NotesText{}, "year = ?", []any{fiscalYear}},
			// Zuletzt das Journal mit seinen Anhängen und die Belege.
			{&domain.ReceiptFile{}, "receipt_id IN (?)", []any{receiptIDs}},
			{&domain.Receipt{}, "fiscal_year = ?", []any{fiscalYear}},
			{&domain.JournalLine{}, "entry_id IN (?)", []any{entryIDs}},
			{&domain.EntertainmentDetail{}, "entry_id IN (?)", []any{entryIDs}},
			{&domain.GiftRecord{}, "entry_id IN (?) OR fiscal_year = ?", []any{entryIDs, fiscalYear}},
			{&domain.JournalEntry{}, "fiscal_year = ?", []any{fiscalYear}},
			{&domain.CheckFinding{}, "check_run_id IN (?)", []any{checkRunIDs}},
			{&domain.CheckRun{}, "fiscal_year = ?", []any{fiscalYear}},
			{&domain.Festschreibung{}, "fiscal_year = ?", []any{fiscalYear}},
			{&domain.VatReturn{}, "fiscal_year = ?", []any{fiscalYear}},
		}
		for _, d := range deletes {
			if err := tx.Where(d.where, d.args...).Delete(d.model).Error; err != nil {
				return fmt.Errorf("das Geschäftsjahr %d ließ sich nicht vollständig löschen: %w", fiscalYear, err)
			}
		}
		cleared, err = clearDanglingReferences(tx, fiscalYear)
		return err
	})
	if err != nil {
		return counts, nil, err
	}
	counts.ClearedReferences = cleared
	return counts, orphans, nil
}

// clearDanglingReferences nullt die Verweise, die nach dem Löschen eines
// Geschäftsjahres ins Leere zeigen, und liefert ihre Zahl.
//
// Nicht alles, was auf eine Buchung oder einen Beleg des gelöschten Jahres
// zeigt, gehört selbst zu diesem Jahr: ein Anlagegut lebt über seine
// Nutzungsdauer, ein Eintrag des Verzeichnisses nach § 15a UStG über zehn Jahre,
// ein Beleg eines späteren Jahres kann auf eine frühere Buchung versiegelt sein.
// Diese Objekte bleiben — ihre eigene Frist läuft ja noch —, aber ihr Verweis
// darf nicht auf eine Zeile zeigen, die es nicht mehr gibt: aus einer bezahlten
// Rechnung würde sonst wieder ein offener Posten, und der Anlagenspiegel
// verwiese auf eine Zugangsbuchung, die niemand mehr nachlesen kann.
//
// Gearbeitet wird über „zeigt auf keine vorhandene Zeile mehr" und nicht über
// „gehörte zum gelöschten Jahr": das trifft auch Verweise, die aus einem
// früheren Löschlauf stammen, und braucht keine zweite Liste, die mit der ersten
// auseinanderlaufen kann.
//
// Das Journal selbst wird nicht angefasst. Seine Felder — der Verweis auf die
// stornierte Buchung voran — gehen in die kanonische Form und damit in die
// Hash-Kette ein; sie zu nullen hieße, die Kette der verbleibenden Jahre zu
// brechen, um einen Verweis aufzuräumen.
func clearDanglingReferences(tx *gorm.DB, fiscalYear int) (int, error) {
	entries := func() *gorm.DB { return tx.Model(&domain.JournalEntry{}).Select("id") }
	receipts := func() *gorm.DB { return tx.Model(&domain.Receipt{}).Select("id") }
	invoices := func() *gorm.DB { return tx.Model(&domain.Invoice{}).Select("id") }

	refs := []struct {
		model  any
		column string
		target func() *gorm.DB
	}{
		{&domain.FixedAsset{}, "acquisition_entry_id", entries},
		{&domain.FixedAsset{}, "disposal_entry_id", entries},
		{&domain.AssetMovement{}, "journal_entry_id", entries},
		{&domain.InputTaxCorrection{}, "entry_id", entries},
		{&domain.InputTaxCorrection{}, "receipt_id", receipts},
		{&domain.InputTaxUsage{}, "entry_id", entries},
		{&domain.Appropriation{}, "journal_entry_id", entries},
		{&domain.Appropriation{}, "receipt_id", receipts},
		{&domain.InventoryCount{}, "journal_entry_id", entries},
		{&domain.InventoryCount{}, "receipt_id", receipts},
		{&domain.Accrual{}, "source_entry_id", entries},
		{&domain.Accrual{}, "formation_entry_id", entries},
		{&domain.AccrualRelease{}, "journal_entry_id", entries},
		{&domain.ProvisionMovement{}, "journal_entry_id", entries},
		{&domain.VendorAdvance{}, "settled_by_entry_id", entries},
		{&domain.AdvanceItem{}, "settlement_entry_id", entries},
		{&domain.Invoice{}, "journal_entry_id", entries},
		{&domain.Invoice{}, "receipt_id", receipts},
		{&domain.Invoice{}, "corrects_invoice_id", invoices},
		{&domain.Invoice{}, "cancelled_by_invoice_id", invoices},
		{&domain.InvoiceGroup{}, "final_invoice_id", invoices},
		{&domain.SupplyEvidence{}, "receipt_id", receipts},
		{&domain.BankTransaction{}, "statement_receipt_id", receipts},
		{&domain.Receipt{}, "journal_entry_id", entries},
	}

	var cleared int64
	for _, ref := range refs {
		result := tx.Model(ref.model).
			Where(ref.column+" IS NOT NULL").
			Where(ref.column+" NOT IN (?)", ref.target()).
			Update(ref.column, gorm.Expr("NULL"))
		if result.Error != nil {
			return 0, fmt.Errorf(
				"die Verweise auf das gelöschte Geschäftsjahr %d ließen sich nicht auflösen (%s): %w",
				fiscalYear, ref.column, result.Error)
		}
		cleared += result.RowsAffected
	}
	return int(cleared), nil
}
