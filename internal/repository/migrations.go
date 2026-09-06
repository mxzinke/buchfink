package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/buildinfo"
	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

// SchemaVersion ist der Stand des Datenbankschemas, den dieser Code erwartet.
//
// Sie wird bei jeder Welle erhöht, die Tabellen oder Spalten hinzufügt. Ihr
// Zweck ist nicht die Migration selbst — AutoMigrate erkennt fehlende Spalten
// von allein —, sondern die Frage, ob überhaupt migriert werden musste: eine
// Anpassung, die bei jedem Start läuft und nichts protokolliert, lässt sich
// später nicht von einer unterscheiden, die etwas verändert hat (ARC-05, GoBD
// Rz. 34).
//
//	1–5  Wellen 1 bis 5c (rückwirkend vergeben; Dateien aus dieser Zeit tragen
//	     keine Versionszeile und werden mit 0 gelesen)
//	6    Welle 6: Änderungsprotokoll mit Kette, Aufbewahrungsfristen,
//	     Belegkopfdaten, Migrationsprotokoll
//	7    Welle 7: Mahnwesen (Schreiben, Posten, Basiszinssatz), gelernte
//	     Bankregeln, Bestellbezug und Leistungsnachweis am Beleg
//	8    Welle 8: überschriebene Aufbewahrungsfrist am Beleg, Kennzeichen des
//	     selbst angelegten Kontos
const SchemaVersion = 8

// migratedTables benennt die Tabellen, die dieser Stand anlegt oder ändert.
// Sie steht im Protokoll, damit sich später beantworten lässt, was ein Lauf
// angefasst hat.
var migratedTables = []string{
	"audit_log_entries", "journal_entries", "receipts", "contacts",
	"festschreibungen", "asset_documents", "retention_holds", "schema_migrations",
	"migration_records",
	"bank_rules", "base_rates", "dunning_notices", "dunning_notice_items",
	"accounts",
}

// ApplyMigrations bringt das Schema auf den Stand des Codes und protokolliert
// den Lauf.
//
// Zurückgegeben wird der Protokolleintrag, oder nil, wenn nichts zu tun war.
// „Nichts zu tun" ist der Regelfall: eine Datei, deren gespeicherte
// Schemaversion der Codeversion entspricht, wird nicht angefasst. Das ist nicht
// nur schneller — es ist die Voraussetzung dafür, dass das Protokoll etwas
// aussagt.
func ApplyMigrations(ctx context.Context, db *gorm.DB) (*domain.SchemaMigration, error) {
	// Die Protokolltabelle selbst muss immer da sein, bevor sich die
	// Schemaversion lesen lässt.
	if err := db.WithContext(ctx).AutoMigrate(&domain.SchemaMigration{}, &domain.MigrationRecord{}); err != nil {
		return nil, fmt.Errorf("das Migrationsprotokoll ließ sich nicht anlegen: %w", err)
	}

	current, err := storedSchemaVersion(ctx, db)
	if err != nil {
		return nil, err
	}
	if current >= SchemaVersion {
		return nil, nil
	}

	record := &domain.SchemaMigration{
		RunAt:       time.Now().UTC(),
		AppVersion:  buildinfo.Version,
		FromVersion: current,
		ToVersion:   SchemaVersion,
		Tables:      strings.Join(migratedTables, ", "),
		Actor:       actor.Actor(),
	}

	migrateErr := AutoMigrate(db)
	if migrateErr == nil {
		migrateErr = runBackfills(db)
	}
	if migrateErr != nil {
		record.Result = domain.SchemaMigrationResultFailed
		record.Message = migrateErr.Error()
		// Der gescheiterte Lauf wird protokolliert, bevor der Fehler nach oben
		// geht: gerade er ist der, den später jemand sehen muss.
		_ = db.WithContext(ctx).Create(record).Error
		return record, migrateErr
	}

	record.Result = domain.SchemaMigrationResultOK
	record.Message = fmt.Sprintf(
		"Schema von Version %d auf %d gebracht.", current, SchemaVersion)
	if current == 0 {
		record.Message = fmt.Sprintf(
			"Schema auf Version %d gebracht. Die Datei trug noch keine Versionsangabe — "+
				"entweder ist sie neu oder sie stammt aus der Zeit vor dem Migrationsprotokoll.",
			SchemaVersion)
	}
	if err := db.WithContext(ctx).Create(record).Error; err != nil {
		return nil, fmt.Errorf("der Migrationslauf ließ sich nicht protokollieren: %w", err)
	}
	return record, nil
}

// storedSchemaVersion liest die höchste erfolgreich protokollierte Zielversion.
//
// Gelesen wird das Maximum der erfolgreichen Läufe und nicht der letzte
// Eintrag: ein fehlgeschlagener Lauf darf die erreichte Version nicht
// zurücksetzen.
func storedSchemaVersion(ctx context.Context, db *gorm.DB) (int, error) {
	var version *int
	err := db.WithContext(ctx).Model(&domain.SchemaMigration{}).
		Where("result = ?", domain.SchemaMigrationResultOK).
		Select("MAX(to_version)").Scan(&version).Error
	if err != nil {
		return 0, fmt.Errorf("die gespeicherte Schemaversion ließ sich nicht lesen: %w", err)
	}
	if version == nil {
		return 0, nil
	}
	return *version, nil
}

// runBackfills füllt Spalten, die AutoMigrate nur anlegt.
func runBackfills(db *gorm.DB) error {
	if err := BackfillReceiptKinds(db); err != nil {
		return fmt.Errorf("die Belegarten ließen sich nicht ergänzen: %w", err)
	}
	if err := BackfillContactAddresses(db); err != nil {
		return fmt.Errorf("die Anschriften ließen sich nicht zerlegen: %w", err)
	}
	if err := BackfillRetention(db); err != nil {
		return fmt.Errorf("die Aufbewahrungsfristen ließen sich nicht nachtragen: %w", err)
	}
	return nil
}

// --- Repository -----------------------------------------------------------

type migrationRepositoryGorm struct {
	db *gorm.DB
}

// NewMigrationRepository liefert die Protokolle über Schemaänderungen und
// Datenübernahmen.
func NewMigrationRepository(db *gorm.DB) domain.MigrationRepository {
	return &migrationRepositoryGorm{db: db}
}

func (r *migrationRepositoryGorm) FindSchemaMigrations(ctx context.Context) ([]domain.SchemaMigration, error) {
	out := make([]domain.SchemaMigration, 0)
	err := dbFrom(ctx, r.db).Order("id desc").Find(&out).Error
	return out, err
}

func (r *migrationRepositoryGorm) CreateMigrationRecord(ctx context.Context, rec *domain.MigrationRecord) error {
	if rec.RunAt.IsZero() {
		rec.RunAt = time.Now().UTC()
	}
	if rec.AppVersion == "" {
		rec.AppVersion = buildinfo.Version
	}
	if rec.Actor == "" {
		rec.Actor = actor.Actor()
	}
	return dbFrom(ctx, r.db).Create(rec).Error
}

func (r *migrationRepositoryGorm) FindMigrationRecords(ctx context.Context) ([]domain.MigrationRecord, error) {
	out := make([]domain.MigrationRecord, 0)
	err := dbFrom(ctx, r.db).Order("id desc").Find(&out).Error
	return out, err
}

// CountForMigration zählt die Objekte einer geöffneten Datenbank.
//
// Die Abstimmung einer Datenübernahme hängt daran: ohne Zählung bliebe die
// Frage, ob alles angekommen ist, unbeantwortet, bis irgendwann eine Bilanz
// nicht mehr aufgeht (ARC-05).
func CountForMigration(ctx context.Context, db *gorm.DB) (domain.MigrationCounts, error) {
	var counts domain.MigrationCounts
	tx := db.WithContext(ctx)

	count := func(model any, into *int) error {
		var n int64
		if err := tx.Model(model).Count(&n).Error; err != nil {
			return err
		}
		*into = int(n)
		return nil
	}
	for _, pair := range []struct {
		model any
		into  *int
	}{
		{&domain.JournalEntry{}, &counts.JournalEntries},
		{&domain.JournalLine{}, &counts.JournalLines},
		{&domain.Receipt{}, &counts.Receipts},
		{&domain.Contact{}, &counts.Contacts},
		{&domain.Account{}, &counts.Accounts},
		{&domain.Invoice{}, &counts.Invoices},
		{&domain.FixedAsset{}, &counts.FixedAssets},
		{&domain.AuditLogEntry{}, &counts.AuditEntries},
	} {
		if err := count(pair.model, pair.into); err != nil {
			return counts, fmt.Errorf("die übernommenen Objekte ließen sich nicht zählen: %w", err)
		}
	}

	// Soll und Haben über alle Buchungen. Stimmen sie nicht überein, ist die
	// Übernahme unvollständig — eine Buchung ist immer ausgeglichen, also ist
	// es auch ihre Summe.
	type sides struct {
		Side  string
		Total int64
	}
	var rows []sides
	if err := tx.Model(&domain.JournalLine{}).
		Select("side, COALESCE(SUM(amount),0) as total").
		Group("side").Scan(&rows).Error; err != nil {
		return counts, fmt.Errorf("die Journalsummen ließen sich nicht bilden: %w", err)
	}
	for _, row := range rows {
		if domain.Side(row.Side) == domain.SideDebit {
			counts.DebitTotal = domain.Cents(row.Total)
		} else {
			counts.CreditTotal = domain.Cents(row.Total)
		}
	}
	return counts, nil
}
