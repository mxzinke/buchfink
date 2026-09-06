package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type receiptRepositoryGorm struct {
	db *gorm.DB
}

// NewReceiptRepository creates a GORM-backed ReceiptRepository.
func NewReceiptRepository(db *gorm.DB) domain.ReceiptRepository {
	return &receiptRepositoryGorm{db: db}
}

func (r *receiptRepositoryGorm) FindByID(ctx context.Context, id uint) (*domain.Receipt, error) {
	var receipt domain.Receipt
	err := dbFrom(ctx, r.db).
		Preload("Files", func(db *gorm.DB) *gorm.DB { return db.Order("receipt_files.position asc") }).
		First(&receipt, id).Error
	if err != nil {
		return nil, err
	}
	receipt.EnsureLists()
	return &receipt, nil
}

func (r *receiptRepositoryGorm) FindAll(ctx context.Context, fiscalYear int) ([]domain.Receipt, error) {
	return r.find(ctx, fiscalYear, "")
}

func (r *receiptRepositoryGorm) FindByStatus(ctx context.Context, fiscalYear int, status domain.ReceiptStatus) ([]domain.Receipt, error) {
	return r.find(ctx, fiscalYear, status)
}

func (r *receiptRepositoryGorm) find(ctx context.Context, fiscalYear int, status domain.ReceiptStatus) ([]domain.Receipt, error) {
	var receipts []domain.Receipt
	q := dbFrom(ctx, r.db).
		Preload("Files", func(db *gorm.DB) *gorm.DB { return db.Order("receipt_files.position asc") }).
		Order("id desc")
	if fiscalYear > 0 {
		q = q.Where("fiscal_year = ?", fiscalYear)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	err := q.Find(&receipts).Error
	// Ein Beleg ohne Datei ist der Regelfall, solange nur die Kopfdaten
	// erfasst sind; als `null` nähme die Dateiliste die Belegansicht mit.
	for i := range receipts {
		receipts[i].EnsureLists()
	}
	return receipts, err
}

func (r *receiptRepositoryGorm) FindByOriginalHash(ctx context.Context, sha256 string) (*domain.Receipt, error) {
	var receipt domain.Receipt
	err := dbFrom(ctx, r.db).
		Preload("Files", func(db *gorm.DB) *gorm.DB { return db.Order("receipt_files.position asc") }).
		Where("id IN (SELECT receipt_id FROM receipt_files WHERE role = ? AND sha256 = ?)",
			domain.ReceiptRoleOriginal, sha256).
		Order("id asc").First(&receipt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	receipt.EnsureLists()
	return &receipt, nil
}

func (r *receiptRepositoryGorm) FindByJournalEntry(ctx context.Context, entryID uint) (*domain.Receipt, error) {
	var receipt domain.Receipt
	err := dbFrom(ctx, r.db).
		Preload("Files", func(db *gorm.DB) *gorm.DB { return db.Order("receipt_files.position asc") }).
		Where("journal_entry_id = ?", entryID).First(&receipt).Error
	if err != nil {
		return nil, err
	}
	return &receipt, nil
}

// Create allocates the Belegnummer, computes the Beleg-Hash and inserts the Beleg
// with its files in one transaction.
//
// Allocating inside the transaction is what keeps the number range gapless: a
// rolled-back insert releases the number again. This is also why the files are
// written to disk *before* this call — their names are their digests, so nothing
// on disk depends on the number.
func (r *receiptRepositoryGorm) Create(ctx context.Context, receipt *domain.Receipt, hash domain.ReceiptHashFunc) error {
	return dbFrom(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		if receipt.ReceiptNumber == "" {
			key, err := numberRangeFor(receipt.Direction)
			if err != nil {
				return err
			}
			seq, err := allocateNumber(tx, key, receipt.FiscalYear)
			if err != nil {
				return err
			}
			receipt.ReceiptNumber = domain.FormatReceiptNumberWith(
				receiptNumberFormat(tx), receipt.FiscalYear, seq)
		}

		for i := range receipt.Files {
			receipt.Files[i].Position = i + 1
			receipt.Files[i].ReceiptID = 0
			receipt.Files[i].ID = 0
		}
		receipt.ReceiptHash = hash(receipt)

		return tx.Create(receipt).Error
	})
}

// SaveHeader schreibt die Kopfdaten und rechnet den Beleg-Hash neu.
//
// Geschrieben wird über den Datensatz und eine Spaltenauswahl, nicht über eine
// Map: GORM wendet den Feld-Serializer nur auf dem Struct-Weg an, und ein
// Map-Update legte Aussteller und Betreff im Klartext in Spalten, die als
// `serializer:encrypted` deklariert sind — beim nächsten Lesen scheiterte die
// Entschlüsselung, und der Beleg wäre aus der Liste verschwunden.
func (r *receiptRepositoryGorm) SaveHeader(
	ctx context.Context, receiptID uint, header domain.ReceiptHeader, hash domain.ReceiptHashFunc,
) (*domain.Receipt, error) {
	var updated domain.Receipt
	err := dbFrom(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		var receipt domain.Receipt
		if err := tx.Preload("Files").First(&receipt, receiptID).Error; err != nil {
			return err
		}
		// Ein gebuchter Beleg ist versiegelt: seine Kopfdaten stehen im
		// Beleg-Hash, der Beleg-Hash in der Buchung und die Buchung in der
		// Kette. Eine Änderung hier bräche sie.
		if receipt.Status != domain.ReceiptStatusFiled {
			return fmt.Errorf("Beleg %s ist %s — seine Kopfdaten lassen sich nicht mehr ändern",
				receipt.ReceiptNumber, statusLabel(receipt.Status))
		}

		if header.Kind != "" {
			receipt.Kind = header.Kind
		}
		receipt.DocumentDate = header.DocumentDate
		receipt.IssuerName = header.IssuerName
		receipt.GrossAmount = header.GrossAmount
		receipt.TaxAmount = header.TaxAmount
		receipt.Currency = header.Currency
		receipt.Subject = header.Subject
		receipt.RetentionClass = header.RetentionClass
		receipt.RetentionUntil = header.RetentionUntil
		receipt.ReceiptHash = hash(&receipt)
		receipt.UpdatedAt = time.Now().UTC()

		if err := tx.Model(&receipt).Select(
			"Kind", "DocumentDate", "IssuerName", "GrossAmount", "TaxAmount",
			"Currency", "Subject", "RetentionClass", "RetentionUntil",
			"ReceiptHash", "UpdatedAt",
		).Updates(&receipt).Error; err != nil {
			return err
		}

		updated = receipt
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

// ReplaceFiles swaps the file list of an unsealed Beleg and recomputes the
// Beleg-Hash.
//
// A sealed Beleg is refused: its hash sits in a booked journal entry, and
// changing the list would break the chain. What arrives afterwards is a Beleg of
// its own.
func (r *receiptRepositoryGorm) ReplaceFiles(ctx context.Context, receiptID uint, files []domain.ReceiptFile, hash domain.ReceiptHashFunc) (*domain.Receipt, error) {
	var updated domain.Receipt
	err := dbFrom(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		var receipt domain.Receipt
		if err := tx.First(&receipt, receiptID).Error; err != nil {
			return err
		}
		if receipt.Status != domain.ReceiptStatusFiled {
			return fmt.Errorf("Beleg %s ist %s und seine Dateien lassen sich nicht mehr ändern",
				receipt.ReceiptNumber, statusLabel(receipt.Status))
		}

		if err := tx.Where("receipt_id = ?", receiptID).Delete(&domain.ReceiptFile{}).Error; err != nil {
			return err
		}
		for i := range files {
			files[i].ID = 0
			files[i].ReceiptID = receiptID
			files[i].Position = i + 1
		}
		if len(files) > 0 {
			if err := tx.Create(&files).Error; err != nil {
				return err
			}
		}

		receipt.Files = files
		receipt.ReceiptHash = hash(&receipt)
		if err := tx.Model(&domain.Receipt{}).Where("id = ?", receiptID).
			Updates(map[string]any{"receipt_hash": receipt.ReceiptHash, "updated_at": time.Now().UTC()}).Error; err != nil {
			return err
		}

		updated = receipt
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

// Seal marks a Beleg as booked.
//
// It is idempotent for the same journal entry on purpose. The seal is written
// after the journal transaction has committed, so a crash in between leaves an
// entry pointing at an unsealed Beleg; repeating the seal has to repair that
// rather than fail.
func (r *receiptRepositoryGorm) Seal(ctx context.Context, receiptID uint, entryID uint) error {
	return dbFrom(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		var receipt domain.Receipt
		if err := tx.First(&receipt, receiptID).Error; err != nil {
			return err
		}
		if receipt.Status == domain.ReceiptStatusDiscarded {
			return fmt.Errorf("Beleg %s wurde verworfen und kann nicht gebucht werden", receipt.ReceiptNumber)
		}
		if receipt.JournalEntryID != nil && *receipt.JournalEntryID != entryID {
			return fmt.Errorf("Beleg %s ist bereits mit einer anderen Buchung versiegelt", receipt.ReceiptNumber)
		}
		if receipt.Status == domain.ReceiptStatusSealed && receipt.JournalEntryID != nil {
			return nil
		}
		return tx.Model(&domain.Receipt{}).Where("id = ?", receiptID).
			Updates(map[string]any{
				"status":           domain.ReceiptStatusSealed,
				"journal_entry_id": entryID,
				"updated_at":       time.Now().UTC(),
			}).Error
	})
}

// SaveInputTaxOverride schreibt den Grund der Übersteuerung an den Beleg.
//
// Über den Datensatz und nicht über eine Map: der Grund liegt verschlüsselt in
// der Spalte, und GORM wendet den Feld-Serializer nur auf dem Struct-Weg an. Ein
// Map-Update legte den Klartext dorthin, wo Geheimtext erwartet wird, und beim
// nächsten Lesen scheiterte die Entschlüsselung des ganzen Belegs.
func (r *receiptRepositoryGorm) SaveInputTaxOverride(
	ctx context.Context, receiptID uint, reason, at string,
) error {
	receipt := domain.Receipt{ID: receiptID, InputTaxOverride: reason, InputTaxOverrideAt: at}
	return dbFrom(ctx, r.db).Model(&receipt).
		Select("InputTaxOverride", "InputTaxOverrideAt").Updates(&receipt).Error
}

// SaveRetention schreibt die überschriebene Aufbewahrungsfrist an den Beleg.
//
// Auch sie geht über den Datensatz und eine Spaltenauswahl: der Grund liegt
// verschlüsselt in seiner Spalte, und GORM wendet den Feld-Serializer nur auf
// dem Struct-Weg an. Ein gebuchter Beleg ist ausdrücklich nicht ausgenommen —
// die Frist steht außerhalb des Beleg-Hashes, und gerade der gebuchte Beleg ist
// der, dessen Aufbewahrung ein Rechtsstreit verlängert.
func (r *receiptRepositoryGorm) SaveRetention(
	ctx context.Context, receiptID uint, class domain.RetentionClass, until, reason, at string,
) error {
	receipt := domain.Receipt{
		ID: receiptID, RetentionClass: class, RetentionUntil: until,
		RetentionOverrideReason: reason, RetentionOverrideAt: at,
	}
	return dbFrom(ctx, r.db).Model(&receipt).
		Select("RetentionClass", "RetentionUntil", "RetentionOverrideReason", "RetentionOverrideAt").
		Updates(&receipt).Error
}

// SaveAuditTrail schreibt Bestellbezug und Leistungsnachweis an den Beleg.
//
// Über den Datensatz und eine Spaltenauswahl wie die Übersteuerung: der
// Leistungsnachweis liegt verschlüsselt in seiner Spalte, und GORM wendet den
// Feld-Serializer nur auf dem Struct-Weg an.
func (r *receiptRepositoryGorm) SaveAuditTrail(
	ctx context.Context, receiptID uint, orderReference, proof, proofAt string,
) error {
	receipt := domain.Receipt{
		ID: receiptID, OrderReference: orderReference,
		ServiceProof: proof, ServiceProofAt: proofAt,
	}
	return dbFrom(ctx, r.db).Model(&receipt).
		Select("OrderReference", "ServiceProof", "ServiceProofAt").Updates(&receipt).Error
}

// Discard retires a filed Beleg without deleting it. It already carries a
// Belegnummer, and a received document must stay findable.
func (r *receiptRepositoryGorm) Discard(ctx context.Context, receiptID uint, reason string) error {
	return dbFrom(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		var receipt domain.Receipt
		if err := tx.First(&receipt, receiptID).Error; err != nil {
			return err
		}
		if receipt.Status == domain.ReceiptStatusSealed {
			return fmt.Errorf("Beleg %s ist gebucht. Eine Korrektur läuft über Storno der Buchung, nicht über den Beleg",
				receipt.ReceiptNumber)
		}
		return tx.Model(&domain.Receipt{}).Where("id = ?", receiptID).
			Updates(map[string]any{
				"status":         domain.ReceiptStatusDiscarded,
				"discard_reason": reason,
				"updated_at":     time.Now().UTC(),
			}).Error
	})
}

// SaveValidation records the outcome of checking the structured part.
//
// It is allowed on a sealed Beleg: the check touches no file, so the Beleg-Hash
// and the journal chain are unaffected, and a rule set updated later must be able
// to write a fresh result against an already booked document.
func (r *receiptRepositoryGorm) SaveValidation(ctx context.Context, receiptID uint, v domain.ReceiptValidation) error {
	return dbFrom(ctx, r.db).Model(&domain.Receipt{}).Where("id = ?", receiptID).
		Updates(map[string]any{
			"detected_format":     v.Format,
			"detected_profile":    v.Profile,
			"validated_at":        v.At,
			"validation_ruleset":  v.Ruleset,
			"validation_version":  v.Version,
			"validation_coverage": v.Coverage,
			"validation_errors":   v.Errors,
			"validation_findings": v.Findings,
			"updated_at":          time.Now().UTC(),
		}).Error
}

// receiptNumberFormat liest die eingestellte Systematik des Belegnummernkreises
// (BEL-02 K4).
//
// Gelesen wird in derselben Transaktion, in der die Nummer vergeben wird: das
// Format entscheidet über die Nummer, und eine zwischen Lesen und Vergeben
// geänderte Einstellung ergäbe sonst eine Nummer nach dem alten Format mit
// einem Zähler des neuen. Ein fehlender oder untauglicher Eintrag ergibt die
// Voreinstellung — ein Beleg darf an einer Einstellung nicht scheitern, er ist
// aufzubewahren, sobald er da ist (GoBD Rz. 131).
func receiptNumberFormat(tx *gorm.DB) string {
	var item domain.SettingItem
	if err := tx.Where("key = ?", "receipt_number_format").First(&item).Error; err != nil {
		return domain.DefaultReceiptNumberFormat
	}
	if domain.ValidateNumberFormat(item.Value) != nil {
		return domain.DefaultReceiptNumberFormat
	}
	return item.Value
}

// numberRangeFor picks the counter a Beleg draws its number from. Incoming
// documents get their own series; an outgoing Beleg carries the Rechnungsnummer
// the invoice already allocated, so it never reaches this function.
func numberRangeFor(direction domain.Direction) (domain.NumberRangeKey, error) {
	switch direction {
	case domain.DirectionIncoming:
		return domain.NumberRangeReceipt, nil
	case domain.DirectionOutgoing:
		return domain.NumberRangeInvoice, nil
	default:
		return "", fmt.Errorf("unbekannte Belegrichtung %q", direction)
	}
}

func statusLabel(s domain.ReceiptStatus) string {
	switch s {
	case domain.ReceiptStatusSealed:
		return "gebucht und versiegelt"
	case domain.ReceiptStatusDiscarded:
		return "verworfen"
	default:
		return string(s)
	}
}

// IsNotFound reports a missing record without leaking GORM into the services.
func IsNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }
