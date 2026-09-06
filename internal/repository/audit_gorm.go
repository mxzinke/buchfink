package repository

import (
	"context"
	"strings"
	"sync"
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
	// Stammdatenmaske: der gelesene Stand trägt sie, der übergebene nicht, und
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

// auditAppendMu serialisiert das Anhängen an die Protokollkette von außen —
// also dort, wo dieses Repository seine Transaktion selbst öffnet.
//
// Die gemeinsame Transaktion allein genügt nicht: GORM beginnt sie als BEGIN
// DEFERRED, und die Schreibsperre entsteht erst beim ersten Schreiben. Zwei
// gleichzeitige Schreiber lesen deshalb denselben Kettenkopf; im WAL-Modus
// scheitert der zweite beim Schreiben mit SQLITE_BUSY_SNAPSHOT, und weil fast
// jede Aufrufstelle den Fehler des Protokollierens verwirft — der Vorgang selbst
// soll an seinem Protokolleintrag nicht scheitern —, ginge der Eintrag
// stillschweigend verloren. Genau das darf ein Änderungsprotokoll nicht.
//
// Ein Mutex im Prozess und keine Datenbanksperre: Buchfink ist ein
// Einzelplatzprogramm, die Datei wird von genau einem Prozess geöffnet, und
// paketweit statt am Objekt, weil zwei Instanzen desselben Repositories auf
// dieselbe Datei zeigen können.
var auditAppendMu sync.Mutex

// append schreibt einen Protokolleintrag und kettet ihn an seinen Vorgänger.
//
// Das Lesen des Kettenkopfes und das Schreiben liegen in derselben Transaktion
// und unter derselben Sperre: zwei gleichzeitige Schreiber, die denselben Kopf
// lesen, erzeugten zwei Einträge mit demselben Vorgängerhash — die Kette hätte
// eine Gabelung, und die Prüfung meldete einen Bruch, den niemand verursacht
// hat.
func (r *auditRepositoryGorm) append(
	ctx context.Context,
	action domain.AuditAction,
	entityType, entityID, details, before, after string,
) error {
	entry := domain.AuditLogEntry{
		// UTC: eine Ortszeit ohne Zone ist in der Nacht der Zeitumstellung
		// mehrdeutig, und die Reihenfolge des Protokolls hängt an ihr.
		Timestamp:  time.Now().UTC(),
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		Details:    details,
		Before:     before,
		After:      after,
		Actor:      actor.Actor(),
		AppVersion: buildinfo.Version,
	}

	write := func(tx *gorm.DB) error {
		var last domain.AuditLogEntry
		prev := domain.GenesisHash
		err := tx.Order("id desc").Limit(1).Take(&last).Error
		switch {
		case err == nil:
			if last.EntryHash != "" {
				prev = last.EntryHash
			}
		case err == gorm.ErrRecordNotFound:
			// Erster Eintrag: die Kette beginnt beim Genesis-Hash.
		default:
			return err
		}
		entry.PreviousHash = prev
		entry.EntryHash = r.chain.CalculateHash(&entry, prev)
		return tx.Create(&entry).Error
	}

	db := dbFrom(ctx, r.db)
	// Läuft schon eine Transaktion — etwa die einer Buchung —, wird sie
	// benutzt: eine eigene daneben hielte fest, was der Aufrufer gleich
	// zurückrollt, und die Kette trüge einen Eintrag über einen Vorgang, den es
	// nicht gegeben hat.
	//
	// Und dann ohne die Sperre: der Aufrufer schreibt bereits in dieser
	// Transaktion, SQLite lässt zu jeder Zeit genau einen Schreiber zu, und der
	// Eintrag ist damit schon serialisiert. Die Sperre hier zu nehmen wäre nicht
	// nur überflüssig, sondern gefährlich — sie schlösse einen Kreis: der
	// Halter der Sperre wartete draußen auf die Schreibsperre der Datei, die
	// diese Transaktion hält, während diese Transaktion auf die Sperre wartete.
	// Beide kämen erst über den busy_timeout wieder frei, und der
	// Protokolleintrag ginge dabei verloren, weil fast jede Aufrufstelle den
	// Fehler des Protokollierens verwirft.
	if _, nested := ctx.Value(txContextKey{}).(*gorm.DB); nested {
		return write(db)
	}

	auditAppendMu.Lock()
	defer auditAppendMu.Unlock()

	// Und mit Wiederholung: hält ein anderer Weg gerade die Schreibsperre der
	// Datei — eine laufende Buchungstransaktion etwa —, weist SQLite diesen
	// Schreiber sofort mit SQLITE_BUSY ab, ohne den busy_timeout abzuwarten
	// (der Wartehandler läuft nicht, wo er eine Verklemmung erzeugen würde).
	// Ohne die Wiederholung ginge der Protokolleintrag dann still verloren,
	// weil fast jede Aufrufstelle den Fehler des Protokollierens verwirft. Eine
	// Buchungstransaktion dauert Millisekunden; eine Sekunde Geduld reicht für
	// sie und lässt einen echten Fehler trotzdem durch.
	var err error
	for attempt := 0; attempt < 20; attempt++ {
		if err = db.Transaction(write); err == nil || !isDatabaseBusy(err) {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
	return err
}

// isDatabaseBusy erkennt die Abweisung durch einen anderen Schreiber.
//
// Über den Text und nicht über einen Fehlerwert: der Treiber (modernc über
// glebarez) reicht seine Fehler nicht als typisierte Werte durch, und ein
// Vergleich gegen einen Code, den es hier nicht gibt, träfe nie zu.
func isDatabaseBusy(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "busy") || strings.Contains(text, "database is locked")
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
