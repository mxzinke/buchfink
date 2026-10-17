package repository

import (
	"context"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/buildinfo"
	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type auditRepositoryGorm struct {
	db    *gorm.DB
	chain *accounting.AuditChain
}

// NewAuditRepository creates a new GORM-backed AuditRepository.
func NewAuditRepository(db *gorm.DB) domain.AuditRepository {
	return &auditRepositoryGorm{db: db, chain: accounting.NewAuditChain()}
}

func (r *auditRepositoryGorm) Log(ctx context.Context, action domain.AuditAction, entityType, entityID, details string) error {
	return r.append(ctx, action, entityType, entityID, details, "", "")
}

func (r *auditRepositoryGorm) LogChange(
	ctx context.Context,
	action domain.AuditAction,
	entityType, entityID, details string,
	before, after any,
) error {
	// Berechnete Felder gehören nicht ins Protokoll: sie werden nicht
	// gespeichert, stehen je nach Aufrufweg belegt oder leer da und erschienen
	// sonst als Änderung, die niemand vorgenommen hat.
	beforeJSON, afterJSON := domain.ChangedFields(before, after, ignoredAuditFields...)
	return r.append(ctx, action, entityType, entityID, details, beforeJSON, afterJSON)
}

// ignoredAuditFields sind die JSON-Namen der abgeleiteten Felder der
// Stammdaten. Sie stehen zentral und nicht an jeder Aufrufstelle, weil ein
// vergessener Name dort ein Protokoll erzeugte, das bei jedem Speichern eine
// Änderung meldet.
var ignoredAuditFields = []string{
	"openAmount",   // Saldo des Personenkontos, beim Lesen gerechnet
	"vatIdNotice",  // Hinweis zur Bestätigungsabfrage, nie gespeichert
	"updatedAt",    // ändert sich bei jedem Speichern und sagt nichts
	"createdAt",    //
	"accountName",  // Kontobezeichnung, beim Lesen ergänzt
	"bookValue",    // Buchwert eines Anlageguts, gerechnet
	"depreciation", //
	// Die Bewegungen eines Anlageguts gehören den Buchungen und nicht der
	// Stammdatenmaske: der gelesene Stand hat sie, der übergebene nicht, und
	// sie erschienen sonst bei jedem Speichern als Änderung, die niemand
	// vorgenommen hat.
	"movements",
	// Die Kennziffern und Nachträge einer Voranmeldung stehen in eigenen
	// Feldern und werden aus dem Journal gerechnet; sie im Protokoll zu
	// wiederholen ersetzte die Änderung durch das ganze Formular.
	"figures",
	"lateEntries",
	// Die Dateiliste eines Belegs hängt am Ladeweg — der eine Aufruf lädt sie
	// mit, der andere nicht — und ihre Änderung wird auf ihrem eigenen Weg
	// protokolliert (ReceiptService.replaceFiles). Im Vorher/Nachher der
	// Kopfdaten wäre sie Rauschen.
	"files",
}

// append schreibt einen Protokolleintrag und kettet ihn an seinen Vorgänger.
//
// Lesen des Kettenkopfes und Schreiben liegen in einer Transaktion. Sie beginnt
// mit BEGIN IMMEDIATE (siehe tenantDSN), zwei Schreiber lesen deshalb nie
// denselben Kopf. Läuft schon eine Transaktion, etwa die einer Buchung, gehört
// der Eintrag zu ihr und fällt mit ihr.
func (r *auditRepositoryGorm) append(
	ctx context.Context,
	action domain.AuditAction,
	entityType, entityID, details, before, after string,
) error {
	entry := domain.AuditLogEntry{
		// UTC: eine Ortszeit ohne Zone ist in der Nacht der Zeitumstellung
		// mehrdeutig, und die Reihenfolge des Protokolls richtet sich nach ihr.
		Timestamp:  time.Now().UTC(),
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		Details:    details,
		Before:     before,
		After:      after,
		Actor:      actor.Actor(),
		Channel:    string(actor.RecordedChannel(ctx)),
		AppVersion: buildinfo.Version,
	}

	return dbFrom(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		var last domain.AuditLogEntry
		prev := domain.GenesisHash
		err := tx.Order("id desc").Limit(1).Take(&last).Error
		switch {
		case err == nil:
			// Auch ein leerer Eigenhash wird übernommen: die Prüfung meldet den
			// Bruch am betroffenen Eintrag, statt dass die Kette hier neu beginnt.
			prev = last.EntryHash
		case err == gorm.ErrRecordNotFound:
		default:
			return err
		}
		entry.PreviousHash = prev
		entry.EntryHash = r.chain.CalculateHash(&entry, prev)
		return tx.Create(&entry).Error
	})
}

func (r *auditRepositoryGorm) FindAll(ctx context.Context, limit int) ([]domain.AuditLogEntry, error) {
	return r.FindFiltered(ctx, limit, domain.AuditFilter{})
}

func (r *auditRepositoryGorm) FindFiltered(ctx context.Context, limit int, filter domain.AuditFilter) ([]domain.AuditLogEntry, error) {
	entries := make([]domain.AuditLogEntry, 0)
	query := dbFrom(ctx, r.db).Model(&domain.AuditLogEntry{})

	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.EntityType != "" {
		query = query.Where("entity_type = ?", filter.EntityType)
	}
	if filter.EntityID != "" {
		query = query.Where("entity_id = ?", filter.EntityID)
	}
	if filter.Actor != "" {
		query = query.Where("actor = ?", filter.Actor)
	}
	if filter.Channel != "" {
		query = query.Where("channel = ?", filter.Channel)
	}
	// „Zugriffe" ist eine Kategorie und keine Aktion: die Herausgabe von Daten
	// steht als EXPORT, der Prüfermodus als Änderung an der Entität READ_ONLY.
	// Beide Formen in einer Bedingung, damit die Oberfläche eine Abfrage
	// stellt und nicht zwei Listen mischen muss.
	if filter.Access {
		query = query.Where("action = ? OR entity_type = ?",
			domain.AuditActionExport, domain.AuditEntityReadOnly)
	}
	// Die Grenzen sind Tage und werden auf UTC-Zeitpunkte gebracht: die Spalte
	// hält Zeitpunkte, und ein Vergleich gegen „2026-03-01" träfe sonst nur
	// Mitternacht.
	if from, err := time.Parse("2006-01-02", filter.From); err == nil {
		query = query.Where("timestamp >= ?", from.UTC())
	}
	if to, err := time.Parse("2006-01-02", filter.To); err == nil {
		query = query.Where("timestamp < ?", to.AddDate(0, 0, 1).UTC())
	}

	query = query.Order("id desc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&entries).Error
	return entries, err
}

func (r *auditRepositoryGorm) Count(ctx context.Context) (int64, error) {
	var count int64
	err := dbFrom(ctx, r.db).Model(&domain.AuditLogEntry{}).Count(&count).Error
	return count, err
}

// FindAllAscending liefert das Protokoll in Schreibreihenfolge.
//
// Die Kettenprüfung braucht genau diese Richtung: sie läuft vom ersten Eintrag
// vorwärts. Die Anzeige braucht die umgekehrte, deshalb stehen beide da.
func (r *auditRepositoryGorm) FindAllAscending(ctx context.Context) ([]domain.AuditLogEntry, error) {
	entries := make([]domain.AuditLogEntry, 0)
	err := dbFrom(ctx, r.db).Order("id asc").Find(&entries).Error
	return entries, err
}
