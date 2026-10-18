package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// nowUTC ist die Uhr, aus der GORM `CreatedAt` und `UpdatedAt` füllt.
//
// Sie muss gesetzt werden, weil GORM sonst `time.Now()` in Ortszeit nimmt. Der
// SQLite-Treiber schreibt den Zonenversatz mit und liest ihn zurück, sodass ein
// Zeitpunkt in derselben Zone wieder herauskommt, in der er entstand — und wer
// daraus ein Datum bildet, bekommt vor Mitternacht UTC den falschen Tag. Das
// traf das Datum, ab dem der Leistungsnachweis verlangt wird
// (settings_gorm.go): abends gesetzt, stand dort der Folgetag.
//
// Gespeichert wird deshalb überall UTC. Angezeigt wird weiterhin in Ortszeit;
// das ist Sache der Oberfläche und nicht der Ablage (docs/entwicklung/architektur.md,
// Zeitpunkte).
func nowUTC() time.Time {
	return time.Now().UTC()
}

// InitTenantDB initializes a GORM SQLite database for a tenant.
// Master data (accounts, contacts, settings) is overarching, and bookings exist within fiscal years.
func InitTenantDB(dataDir string) (*gorm.DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("could not create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "buchfink.sqlite")

	db, err := gorm.Open(sqlite.Open(tenantDSN(dbPath)), &gorm.Config{
		Logger:  logger.Default.LogMode(logger.Warn),
		NowFunc: nowUTC,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to sqlite db at %s: %w", dbPath, err)
	}

	// Die Schemaanpassung läuft nur, wenn der Code neuer ist als die Datei, und
	// sie protokolliert sich (siehe ApplyMigrations, ARC-05).
	if _, err := ApplyMigrations(context.Background(), db); err != nil {
		_ = CloseDB(db)
		return nil, fmt.Errorf("failed to run database automigrations: %w", err)
	}

	currentYear := time.Now().Year()
	if err := SeedDefaultsIfEmpty(context.Background(), db, currentYear); err != nil {
		_ = CloseDB(db)
		return nil, fmt.Errorf("failed to seed initial SKR04 data: %w", err)
	}

	return db, nil
}

// BusyTimeout ist die Wartezeit eines Schreibers auf die Schreibsperre der
// Datei. Sie deckt auch lange Vorgänge wie das Erzeugen eines
// Rechnungsdokuments in der Transaktion ab.
const BusyTimeout = 15 * time.Second

// tenantDSN öffnet die Mandantendatei im WAL-Modus. `_txlock=immediate` lässt
// jede Transaktion mit BEGIN IMMEDIATE beginnen: die Schreibsperre entsteht beim
// Beginn und nicht erst beim ersten Schreiben, ein zweiter Schreiber wartet
// deshalb im busy_timeout, statt beim Hochstufen mit SQLITE_BUSY_SNAPSHOT
// abgewiesen zu werden.
func tenantDSN(dbPath string) string {
	return fmt.Sprintf(
		"%s?_pragma=journal_mode(wal)&_pragma=busy_timeout(%d)&_pragma=secure_delete(ON)&_txlock=immediate",
		dbPath, BusyTimeout.Milliseconds())
}

// InitInMemoryDB initializes an ephemeral SQLite database for unit and integration testing.
func InitInMemoryDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:  logger.Default.LogMode(logger.Silent),
		NowFunc: nowUTC,
	})
	if err != nil {
		return nil, err
	}

	if _, err := ApplyMigrations(context.Background(), db); err != nil {
		return nil, err
	}

	return db, nil
}

// CloseDB gibt die Verbindungen einer Datenbank frei.
//
// Für die Tests: eine Datenbank im Arbeitsspeicher lebt, solange ihre
// Verbindung offen ist, und `database/sql` hält den Pool bis zum Ende des
// Prozesses. Ein Testlauf über einige hundert Fälle hielte damit einige hundert
// Datenbanken zugleich — bis SQLite keinen Speicher mehr bekommt und die Fälle,
// die zuletzt laufen, an einer Meldung scheitern, die mit ihnen nichts zu tun
// hat.
func CloseDB(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// OpenReadOnlyDB öffnet eine vorhandene Datenbankdatei, ohne sie zu verändern.
//
// Der Wiederherstellungstest braucht das: er prüft eine Sicherung und darf sie
// dabei nicht anfassen. Ohne den Nur-Lese-Modus liefe AutoMigrate über die
// Kopie, und die geprüfte Datei wäre nicht mehr die gesicherte — ein Test, der
// seinen Gegenstand verändert, prüft ihn nicht.
func OpenReadOnlyDB(dbPath string) (*gorm.DB, error) {
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("die Datenbankdatei %s wurde nicht gefunden: %w", filepath.Base(dbPath), err)
	}
	dsn := fmt.Sprintf("file:%s?mode=ro&_pragma=busy_timeout(5000)", dbPath)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:  logger.Default.LogMode(logger.Silent),
		NowFunc: nowUTC,
	})
	if err != nil {
		return nil, fmt.Errorf("die Datenbank konnte nicht schreibgeschützt geöffnet werden: %w", err)
	}
	return db, nil
}

// VacuumInto schreibt eine in sich stimmige Kopie der Datenbank.
//
// Ein einfaches Kopieren der Datei genügt im WAL-Modus nicht: die zuletzt
// geschriebenen Seiten stehen dann noch im Write-Ahead-Log daneben, und die
// Kopie wäre auf einem Stand, den es nie gab. VACUUM INTO schreibt eine Kopie
// des vollständigen, in sich geschlossenen Standes (SQLite ab 3.27).
func VacuumInto(db *gorm.DB, targetPath string) error {
	if db == nil {
		return fmt.Errorf("keine Datenbank geöffnet")
	}
	// VACUUM INTO weigert sich, eine vorhandene Datei zu überschreiben.
	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("die vorherige Kopie konnte nicht entfernt werden: %w", err)
	}
	if err := db.Exec("VACUUM INTO ?", targetPath).Error; err != nil {
		return fmt.Errorf("die Datenbank konnte nicht gesichert werden: %w", err)
	}
	return nil
}

// AutoMigrate applies schema changes for all domain entities.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&domain.Account{},
		&domain.JournalEntry{},
		&domain.JournalLine{},
		&domain.EntertainmentDetail{},
		&domain.GiftRecord{},
		&domain.Receipt{},
		&domain.ReceiptFile{},
		&domain.NumberRange{},
		&domain.PaymentAllocation{},
		&domain.BankTransaction{},
		&domain.BankAccount{},
		&domain.Contact{},
		&domain.Invoice{},
		&domain.InvoiceItem{},
		&domain.InvoiceReference{},
		&domain.AuditLogEntry{},
		&domain.SettingItem{},
		&domain.ExchangeRate{},
		&domain.Festschreibung{},
		&domain.FiscalYear{},
		&domain.FixedAsset{},
		&domain.AssetMovement{},
		&domain.Foundation{},
		&domain.Shareholder{},
		&domain.CompanyShareholder{},
		&domain.VatReturn{},
		&domain.ZMReturn{},
		&domain.ZMLine{},
		&domain.CheckRun{},
		&domain.CheckFinding{},
		&domain.ObligationRecord{},
		&domain.ObligationEvent{},
		&domain.BackupRun{},
		// Die Abschlussbausteine stehen am Ende, weil AutoMigrate
		// in dieser Reihenfolge anlegt und die Fremdschlüssel auf Journal und
		// Beleg schon existieren müssen.
		&domain.ClosingStep{},
		&domain.Accrual{},
		&domain.AccrualRelease{},
		&domain.Provision{},
		&domain.ProvisionMovement{},
		&domain.DiscountRate{},
		&domain.NotesText{},
		&domain.Appropriation{},
		// Der Rechnungsverbund und seine Abschläge
		// verweisen auf Rechnung und Buchung, der Lückenvermerk auf den
		// Nummernkreis — beide stehen deshalb hinter ihnen.
		&domain.InvoiceGroup{},
		&domain.AdvanceItem{},
		&domain.VendorAdvance{},
		&domain.NumberGap{},
		// Das Verzeichnis nach § 15a
		// UStG verweist auf Anlagegut, Beleg und Buchung, die Bestätigungsabfrage
		// auf den Kontakt — beide stehen deshalb hinter ihnen.
		&domain.InputTaxCorrection{},
		&domain.InputTaxUsage{},
		&domain.VatIDCheck{},
		&domain.VatExchangeRate{},
		// Die beiden Protokolle stehen am Ende — sie
		// verweisen auf nichts und nichts verweist auf sie.
		&domain.SchemaMigration{},
		&domain.MigrationRecord{},
		&domain.ProcedureDocumentation{},
		&domain.BankRule{},
		&domain.IdempotencyRecord{},
		&domain.ExaminerPeriod{},
		&domain.ShareholderLoan{},
		&domain.ShareholderLoanInterest{},
		&domain.PayrollRun{},
		&domain.DeclarationInput{},
		&domain.DeclarationRecord{},
		&domain.DunningNote{},
	)
}

// SeedDefaultsIfEmpty populates initial SKR04 chart of accounts and default company settings if database is newly created.
func SeedDefaultsIfEmpty(ctx context.Context, db *gorm.DB, year int) error {
	// Fehlende Konten werden ergänzt, an der Kontonummer erkannt; vorhandene
	// werden nie gelöscht oder geändert, denn Buchungen verweisen über die
	// Kontonummer. So kommt ein neuer Stand des Kontenrahmens mit einer neuen
	// Programmfassung in jede Datei. Der Vergleich ist billig genug für jedes
	// Öffnen.
	added, err := seedMissingAccounts(ctx, db)
	if err != nil {
		return err
	}
	if added > 0 {
		if err := NewAuditRepository(db).Log(ctx, domain.AuditActionImport, "ACCOUNT", "SKR04",
			fmt.Sprintf("%d fehlende Konten aus dem Kontenrahmen SKR04 ergänzt; vorhandene Konten wurden nicht verändert.", added)); err != nil {
			return fmt.Errorf("die Ergänzung des Kontenrahmens ließ sich nicht protokollieren: %w", err)
		}
	}

	// 2. Seed Default Company Settings
	var sCount int64
	if err := db.WithContext(ctx).Model(&domain.SettingItem{}).Count(&sCount).Error; err == nil && sCount == 0 {
		defaultSettings := []domain.SettingItem{
			{Key: "company_name", Value: ""},
			{Key: "legal_form", Value: "GmbH"},
			{Key: "fiscal_year", Value: fmt.Sprintf("%d", year)},
			{Key: "fiscal_year_start_month", Value: "1"},
			{Key: "tax_number", Value: ""},
			{Key: "vat_id", Value: ""},
			{Key: "tax_office", Value: ""},
			{Key: "street", Value: ""},
			{Key: "postal_code", Value: ""},
			{Key: "city", Value: ""},
			{Key: "country_code", Value: domain.DefaultCountryCode},
			// Sitz, Registergericht und Registernummer sind Pflichtangaben des
			// § 264 Abs. 1a HGB im Kopf des Jahresabschlusses.
			{Key: "seat", Value: ""},
			{Key: "register_court", Value: ""},
			{Key: "register_number", Value: ""},
			{Key: "currency", Value: "EUR"},
			{Key: "skr", Value: "SKR04"},
			{Key: "vat_period", Value: "quarter"},
			{Key: "taxation_type", Value: "SOLL"},
			// Dauerfristverlängerung und Sondervorauszahlung sind eine
			// Entscheidung des Unternehmers (§§ 46 ff. UStDV) und deshalb
			// zunächst aus.
			{Key: "permanent_extension", Value: "false"},
			{Key: "special_prepayment", Value: "0"},
			// GoBD Rz. 47: unbare Geschäftsvorfälle sind innerhalb von zehn
			// Tagen zu erfassen. Der Wert ist einstellbar, weil die Rz. eine
			// Obergrenze nennt und kein Gesetz.
			{Key: "receipt_capture_days", Value: "10"},
			{Key: "commit_grace_days", Value: "0"},
			// Die Systematik des Rechnungsnummernkreises gehört in die
			// Verfahrensdokumentation und ist deshalb eine Einstellung.
			{Key: "invoice_number_format", Value: domain.DefaultInvoiceNumberFormat},
		}

		for _, s := range defaultSettings {
			if err := db.WithContext(ctx).Save(&s).Error; err != nil {
				return fmt.Errorf("failed to seed setting %s: %w", s.Key, err)
			}
		}
	}

	return nil
}

// seedMissingAccounts ergänzt die Konten des Kontenrahmens, die in der Datei
// fehlen, und liefert ihre Zahl.
//
// Verglichen wird über die Kontonummer: sie ist der eindeutige Schlüssel des
// Kontenplans und das, worüber eine Buchung auf ihr Konto zeigt.
func seedMissingAccounts(ctx context.Context, db *gorm.DB) (int, error) {
	defaults := accounting.DefaultSKR04Accounts()
	if len(defaults) == 0 {
		return 0, nil
	}

	var existing []string
	if err := db.WithContext(ctx).Model(&domain.Account{}).Pluck("number", &existing).Error; err != nil {
		return 0, fmt.Errorf("der vorhandene Kontenplan ließ sich nicht lesen: %w", err)
	}
	known := make(map[string]bool, len(existing))
	for _, number := range existing {
		known[number] = true
	}

	missing := make([]domain.Account, 0, len(defaults))
	for i := range defaults {
		if !known[defaults[i].Number] {
			missing = append(missing, defaults[i])
		}
	}
	if len(missing) == 0 {
		return 0, nil
	}
	if err := db.WithContext(ctx).CreateInBatches(&missing, 100).Error; err != nil {
		return 0, fmt.Errorf("failed to seed SKR04 accounts: %w", err)
	}
	return len(missing), nil
}
