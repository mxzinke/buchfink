package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
// das ist Sache der Oberfläche und nicht der Ablage (docs/architektur.md,
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

	// DSN with WAL mode and busy timeout for concurrent safety
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)", dbPath)

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:  logger.Default.LogMode(logger.Warn),
		NowFunc: nowUTC,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to sqlite db at %s: %w", dbPath, err)
	}

	// Die Schemaanpassung läuft nur, wenn der Code neuer ist als die Datei, und
	// sie protokolliert sich (siehe ApplyMigrations). Vorher lief sie bei jedem
	// Start und hinterließ keine Spur — für die Verfahrensdokumentation ist das
	// die Lücke, die ARC-05 meint.
	if _, err := ApplyMigrations(context.Background(), db); err != nil {
		return nil, fmt.Errorf("failed to run database automigrations: %w", err)
	}

	currentYear := time.Now().Year()
	if err := SeedDefaultsIfEmpty(context.Background(), db, currentYear); err != nil {
		return nil, fmt.Errorf("failed to seed initial SKR04 data: %w", err)
	}

	return db, nil
}

// InitDB initializes a GORM SQLite database (backward compatibility alias for InitTenantDB).
func InitDB(dataDir string, year int) (*gorm.DB, error) {
	return InitTenantDB(dataDir)
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
	// Der Kurstabelle muss vor dem Anlegen der Schlüssel gewechselt werden:
	// AutoMigrate ändert Spalten, aber keinen Primärschlüssel.
	if err := migrateExchangeRateTable(db); err != nil {
		return err
	}
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
		&domain.AssetDocument{},
		&domain.Document{},
		&domain.Foundation{},
		&domain.Shareholder{},
		&domain.FoundationTask{},
		&domain.VatReturn{},
		&domain.ZMReturn{},
		&domain.ZMLine{},
		&domain.CheckRun{},
		&domain.CheckFinding{},
		&domain.DeadlineDone{},
		&domain.BackupRun{},
		// Welle 5a: die Abschlussbausteine. Sie stehen am Ende, weil AutoMigrate
		// in dieser Reihenfolge anlegt und die Fremdschlüssel auf Journal und
		// Beleg schon existieren müssen.
		&domain.ClosingStep{},
		&domain.Accrual{},
		&domain.AccrualRelease{},
		&domain.Provision{},
		&domain.ProvisionMovement{},
		&domain.DiscountRate{},
		&domain.InventoryCount{},
		&domain.NotesText{},
		&domain.Appropriation{},
		// Welle 5b: Rechnungswesen. Der Rechnungsverbund und seine Abschläge
		// verweisen auf Rechnung und Buchung, der Lückenvermerk auf den
		// Nummernkreis — beide stehen deshalb hinter ihnen.
		&domain.InvoiceGroup{},
		&domain.AdvanceItem{},
		&domain.VendorAdvance{},
		&domain.NumberGap{},
		// Welle 5c: die steuerlichen Nebenpflichten. Das Verzeichnis nach § 15a
		// UStG verweist auf Anlagegut, Beleg und Buchung, der Belegnachweis auf
		// die Rechnung, die Bestätigungsabfrage auf den Kontakt — alle drei
		// stehen deshalb hinter ihnen.
		&domain.InputTaxCorrection{},
		&domain.InputTaxUsage{},
		&domain.VatIDCheck{},
		&domain.SupplyEvidence{},
		&domain.VatExchangeRate{},
		// Welle 6: die Nachweise. Die Aussetzung der Aufbewahrungsfrist und die
		// beiden Protokolle stehen am Ende — sie verweisen auf nichts und
		// nichts verweist auf sie.
		&domain.RetentionHold{},
		&domain.SchemaMigration{},
		&domain.MigrationRecord{},
		&domain.ProcedureDocumentation{},
		// Welle 7: die Bedienung. Die gelernte Bankregel und der
		// Basiszinssatz hängen an nichts; das Mahnschreiben verweist auf den
		// Kontakt und seine Posten auf Buchungen und steht deshalb dahinter.
		&domain.BankRule{},
		&domain.BaseRate{},
		&domain.DunningNotice{},
		&domain.DunningNoticeItem{},
	)
}

// migrateExchangeRateTable bringt die Kurstabelle auf ihre datierte Form.
//
// Die alte Fassung hatte die Währung als einzigen Primärschlüssel und konnte
// damit nur einen Kurs je Währung halten — eine Zwischenablage, keine Historie.
// AutoMigrate ändert Spalten, aber keinen Primärschlüssel: die Tabelle muss neu
// angelegt werden, und der Inhalt muss mit.
//
// Der Inhalt ist in der Praxis leer. `domain.ExchangeRate` war zwar migriert,
// aber es gab keinen Weg, der je einen Kurs geschrieben hätte. Trotzdem wird
// umkopiert statt gelöscht: eine Annahme über fremde Daten, die man auch prüfen
// kann, prüft man.
func migrateExchangeRateTable(db *gorm.DB) error {
	if !db.Migrator().HasTable("exchange_rates") {
		return nil
	}
	if db.Migrator().HasColumn(&domain.ExchangeRate{}, "id") {
		return nil
	}

	type legacyRate struct {
		Currency string
		Rate     float64
		Date     string
		Source   string
	}
	var legacy []legacyRate
	if err := db.Table("exchange_rates").Find(&legacy).Error; err != nil {
		return fmt.Errorf("die bisherigen Umrechnungskurse ließen sich nicht lesen: %w", err)
	}
	if err := db.Migrator().DropTable("exchange_rates"); err != nil {
		return fmt.Errorf("die alte Kurstabelle ließ sich nicht ablösen: %w", err)
	}
	if err := db.AutoMigrate(&domain.ExchangeRate{}); err != nil {
		return fmt.Errorf("die neue Kurstabelle ließ sich nicht anlegen: %w", err)
	}
	for _, l := range legacy {
		if l.Currency == "" || l.Date == "" || l.Rate <= 0 {
			continue
		}
		source := l.Source
		if source == "" {
			source = "aus der früheren Kurstabelle übernommen"
		}
		rate := domain.ExchangeRate{
			Currency:   l.Currency,
			Date:       l.Date,
			RateMicros: int64(l.Rate*float64(domain.RateScale) + 0.5),
			Source:     source,
		}
		if err := db.Create(&rate).Error; err != nil {
			return fmt.Errorf("der übernommene Kurs %s zum %s ließ sich nicht schreiben: %w",
				l.Currency, l.Date, err)
		}
	}
	return nil
}

// BackfillContactAddresses zerlegt die einzeilige Anschrift alter Kontakte in
// ihre Bestandteile.
//
// § 14 Abs. 4 Nr. 1 UStG verlangt die vollständige Anschrift des Empfängers,
// und EN 16931 verlangt sie in Feldern. Bestandsdaten haben sie als eine
// Zeile; ohne diesen Lauf ließe sich zu keinem übernommenen Kunden mehr eine
// Rechnung ausstellen.
//
// Der Parser ist bewusst streng: was er nicht sicher trennen kann, lässt er
// stehen. Die Kontaktseite meldet den Datensatz dann als unvollständig, und
// jemand sieht ihn an. Eine geratene Straße auf einer Rechnung wäre ein
// Formfehler, den der Empfänger mit seinem Vorsteuerabzug bezahlt.
func BackfillContactAddresses(db *gorm.DB) error {
	// Gelesen wird ohne Filter auf die Anschrift: sie liegt verschlüsselt in der
	// Spalte, und ein SQL-Vergleich auf den leeren String träfe den Geheimtext
	// und nicht den Inhalt.
	var contacts []domain.Contact
	if err := db.Find(&contacts).Error; err != nil {
		return err
	}
	for i := range contacts {
		c := &contacts[i]
		if !c.MigrateAddress() {
			continue
		}
		// Geschrieben wird über den Datensatz und nicht über eine Map.
		//
		// GORM wendet den Feld-Serializer nur auf dem Struct-Weg an
		// (schema.Field.ValueOf); ein Map-Update ginge an ihm vorbei und legte
		// den Klartext in die Spalten, die als `serializer:encrypted`
		// deklariert sind. Beim nächsten Lesen scheiterte die Entschlüsselung,
		// und der migrierte Kontakt wäre verschwunden — aus der Kontaktliste
		// und aus dem Rechnungsdialog.
		if err := db.Model(c).Select("Street", "PostalCode", "City").Updates(c).Error; err != nil {
			return err
		}
	}
	return nil
}

// BackfillReceiptKinds gibt Belegen aus der Zeit vor der Belegart ihren Wert.
//
// AutoMigrate legt die Spalte an, füllt sie aber nicht: bestehende Zeilen
// bekommen NULL oder den leeren String, und ein Beleg ohne Art fiele durch die
// Strukturprüfung und aus dem Schlüsselverzeichnis. Der Regelfall ist die
// Rechnung — ein Kontoauszug kann vor dieser Welle gar nicht abgelegt worden
// sein, weil der Bankimport die Datei bisher verworfen hat.
func BackfillReceiptKinds(db *gorm.DB) error {
	return db.Model(&domain.Receipt{}).
		Where("kind IS NULL OR kind = ''").
		Update("kind", domain.ReceiptKindInvoice).Error
}

// BackfillRetention trägt Aufbewahrungsklasse und Fristende an Belegen und
// Anlagendokumenten nach, die vor Welle 6 abgelegt wurden.
//
// Ohne den Lauf stünden in der Belegliste für den ganzen Altbestand leere
// Fristen, obwohl sie aus Belegart und Entstehungsjahr genauso zu rechnen sind
// wie bei jedem neu abgelegten Beleg — und der Bericht über abgelaufene
// Objekte, der über die Löschung entscheidet, sähe die alten Belege überhaupt
// nicht. Ein leeres Feld hieße dort „keine Frist", und das ist die falsche
// Auskunft.
//
// Gelesen wird über eine schmale Auswahl und nicht über den Datensatz: die
// Belegtabelle hat verschlüsselte Spalten (Aussteller, Betreff, Dateipfade),
// und die Frist richtet sich nach keiner von ihnen. So braucht der Lauf den
// Schlüssel nicht.
func BackfillRetention(db *gorm.DB) error {
	type receiptRow struct {
		ID           uint
		Kind         string
		FiscalYear   int
		DocumentDate string
	}
	var receipts []receiptRow
	if err := db.Model(&domain.Receipt{}).
		Select("id", "kind", "fiscal_year", "document_date").
		Where("retention_class IS NULL OR retention_class = ''").
		Scan(&receipts).Error; err != nil {
		return err
	}
	for _, row := range receipts {
		info := accounting.RetentionFor(
			domain.RetentionKindOf(domain.ReceiptKind(row.Kind)),
			retentionOriginYear(row.DocumentDate, row.FiscalYear))
		if info.Class == domain.RetentionClassNone {
			continue
		}
		if err := db.Model(&domain.Receipt{}).Where("id = ?", row.ID).
			Updates(map[string]any{
				"retention_class": info.Class,
				"retention_until": info.RetentionEnd,
			}).Error; err != nil {
			return err
		}
	}

	type documentRow struct {
		ID           uint
		DocumentDate string
		CreatedAt    time.Time
	}
	var documents []documentRow
	if err := db.Model(&domain.AssetDocument{}).
		Select("id", "document_date", "created_at").
		Where("retention_class IS NULL OR retention_class = ''").
		Scan(&documents).Error; err != nil {
		return err
	}
	for _, row := range documents {
		info := accounting.RetentionFor(domain.RetentionKindAssetDocument,
			retentionOriginYear(row.DocumentDate, row.CreatedAt.Year()))
		if info.Class == domain.RetentionClassNone {
			continue
		}
		if err := db.Model(&domain.AssetDocument{}).Where("id = ?", row.ID).
			Updates(map[string]any{
				"retention_class": info.Class,
				"retention_until": info.RetentionEnd,
			}).Error; err != nil {
			return err
		}
	}
	return nil
}

// retentionOriginYear liefert das Entstehungsjahr: das Jahr des Dokumentdatums,
// hilfsweise das mitgegebene Ersatzjahr (Geschäftsjahr bzw. Ablagejahr).
func retentionOriginYear(documentDate string, fallback int) int {
	if len(documentDate) >= 4 {
		if year, err := strconv.Atoi(documentDate[:4]); err == nil && year > 1900 {
			return year
		}
	}
	return fallback
}

// SeedDefaultsIfEmpty populates initial SKR04 chart of accounts and default company settings if database is newly created.
func SeedDefaultsIfEmpty(ctx context.Context, db *gorm.DB, year int) error {
	// Fehlende Konten werden ergänzt, vorhandene nie gelöscht.
	//
	// Vorher stand hier `DELETE FROM accounts`, sobald weniger als hundert
	// Konten dastanden. Das ist die gefährlichste Zeile, die eine Buchhaltung
	// haben kann: ein Kontenplan, den jemand bewusst gekürzt hat, wurde beim
	// nächsten Start weggeworfen, und mit ihm jede Änderung an einem
	// Kontonamen. Buchungen verweisen über die Kontonummer, also überlebte die
	// Buchung ihr Konto — bis zur nächsten Auswertung, die das Konto nicht mehr
	// fand.
	//
	// Ergänzt wird deshalb, was fehlt, und zwar an der Kontonummer erkannt —
	// und ohne Schwellwert. Der frühere Schwellwert von hundert Konten stammte
	// aus der Zeit, in der die Tabelle geleert und neu gefüllt wurde; für das
	// Ergänzen ist er falsch: er ist genau die Bedingung, unter der ein neuer
	// SKR04-Stand nicht in eine bestehende, vollständige Datei käme. Der Lauf
	// ist ein Vergleich der Kontonummern und billig genug, um ihn bei jedem
	// Öffnen zu machen.
	added, err := seedMissingAccounts(ctx, db)
	if err != nil {
		return err
	}
	if added > 0 {
		_ = NewAuditRepository(db).Log(ctx, domain.AuditActionImport, "ACCOUNT", "SKR04",
			fmt.Sprintf("%d fehlende Konten aus dem Kontenrahmen SKR04 ergänzt; vorhandene Konten wurden nicht verändert.", added))
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
			{Key: "iban", Value: ""},
			{Key: "bic", Value: ""},
			{Key: "bank_name", Value: ""},
			{Key: "street", Value: ""},
			{Key: "zip_city", Value: ""},
			{Key: "country", Value: "Deutschland"},
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
