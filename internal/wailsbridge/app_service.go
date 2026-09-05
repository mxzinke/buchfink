package wailsbridge

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/einvoice"
	"github.com/buchfink/buchfink/internal/einvoice/xrechnung"
	"github.com/buchfink/buchfink/internal/einvoice/zugferd"
	"github.com/buchfink/buchfink/internal/invoice"
	"github.com/buchfink/buchfink/internal/receiptstore"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/security"
	"github.com/buchfink/buchfink/internal/service"
	"github.com/wailsapp/wails/v3/pkg/application"
	"gorm.io/gorm"
)

// BuchfinkBridge bridges between Wails v3 frontend IPC and the decoupled Go domain services.
type BuchfinkBridge struct {
	mu          sync.RWMutex
	appCfgRepo  domain.AppConfigRepository
	appConfig   domain.AppConfig
	dataDir     string
	currentYear int
	db          *gorm.DB
	vault       *security.Vault // active tenant's field-encryption vault (nil = clear text)
	locked      bool            // true when the active tenant's keychain secret is missing (needs recovery)

	// Repositories
	accountRepo        domain.AccountRepository
	journalRepo        domain.JournalRepository
	bankRepo           domain.BankRepository
	contactRepo        domain.ContactRepository
	invoiceRepo        domain.InvoiceRepository
	numberRepo         domain.NumberRangeRepository
	allocationRepo     domain.PaymentAllocationRepository
	receiptRepo        domain.ReceiptRepository
	assetRepo          domain.AssetRepository
	auditRepo          domain.AuditRepository
	settingsRepo       domain.SettingsRepository
	festschreibungRepo domain.FestschreibungRepository
	foundationRepo     domain.FoundationRepository
	fiscalYearRepo     domain.FiscalYearRepository

	// Services
	journalSvc    *service.JournalService
	postingSvc    *service.PostingService
	receiptSvc    *service.ReceiptService
	eInvoiceSvc   *service.EInvoiceService
	paymentSvc    *service.PaymentService
	vatSvc        *service.VatService
	accountingSvc *service.AccountingService
	bankSvc       *service.BankService
	invoiceSvc    *service.InvoiceService
	renderer      *invoice.Renderer
	contactSvc    *service.ContactService
	ebilanzSvc    *service.EBilanzService
	assetSvc      *service.AssetService
	auditSvc      *service.AuditService
	settingsSvc   *service.SettingsService
	currencySvc   *service.CurrencyService
	foundationSvc *service.FoundationService
	closingSvc    *service.ClosingService
	statementSvc  *service.StatementService
}

func NewBuchfinkBridge() (*BuchfinkBridge, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	baseDir := filepath.Join(homeDir, ".buchfink")
	appCfgRepo := repository.NewAppConfigRepository(baseDir)

	cfg, err := appCfgRepo.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load app config: %w", err)
	}

	currentYear := cfg.LastFiscalYear
	if currentYear == 0 {
		currentYear = time.Now().Year() // Dynamic current year (e.g. 2026)
	}

	b := &BuchfinkBridge{
		appCfgRepo:  appCfgRepo,
		appConfig:   *cfg,
		dataDir:     cfg.DataDir,
		currentYear: currentYear,
	}

	// If configured, initialize DB for active tenant
	if cfg.IsConfigured {
		var activeTenant *domain.TenantConfig
		if cfg.ActiveTenantID != "" {
			for _, t := range cfg.Tenants {
				if t.ID == cfg.ActiveTenantID {
					activeTenant = &t
					break
				}
			}
		}
		if activeTenant == nil && len(cfg.Tenants) > 0 {
			activeTenant = &cfg.Tenants[0]
		}
		if activeTenant == nil && cfg.DataDir != "" {
			activeTenant = &domain.TenantConfig{
				ID:        "default",
				Name:      "Hauptmandant",
				DataDir:   cfg.DataDir,
				CreatedAt: time.Now().Format(time.RFC3339),
			}
		}

		if activeTenant != nil {
			if err := b.initTenant(activeTenant); err != nil {
				return nil, fmt.Errorf("failed to init tenant database: %w", err)
			}
			// Fetch any trusted timestamps that couldn't be obtained while offline.
			go b.retryPendingTimestamps()
		}
	}

	return b, nil
}

func (b *BuchfinkBridge) initTenant(t *domain.TenantConfig) error {
	if t == nil {
		return fmt.Errorf("no tenant specified")
	}

	// Unlock field-level encryption for this tenant (transparent via OS keychain).
	// Existing databases without a keyfile fall back to clear text so legacy data
	// keeps working.
	if security.KeyfileExists(t.DataDir) {
		vault, err := security.OpenTenantVault(t.DataDir, t.ID)
		if err != nil {
			if errors.Is(err, security.ErrNoKeyringSecret) {
				// New machine or lost keychain: enter locked state instead of
				// crashing. The frontend shows a recovery screen and calls
				// RecoverActiveTenantFromFile with the external recovery file.
				b.locked = true
				b.vault = nil
				repository.SetActiveVault(nil)
				b.appConfig.ActiveTenantID = t.ID
				b.dataDir = t.DataDir
				return nil
			}
			return fmt.Errorf("failed to unlock tenant encryption at %s: %w", t.DataDir, err)
		}
		b.vault = vault
		repository.SetActiveVault(vault)
		b.locked = false
	} else {
		b.vault = nil
		repository.SetActiveVault(nil)
		b.locked = false
	}

	db, err := repository.InitTenantDB(t.DataDir)
	if err != nil {
		return fmt.Errorf("failed to initialize tenant database at %s: %w", t.DataDir, err)
	}

	b.db = db
	b.dataDir = t.DataDir

	// Wire Repositories
	b.accountRepo = repository.NewAccountRepository(db)
	b.journalRepo = repository.NewJournalRepository(db)
	b.numberRepo = repository.NewNumberRangeRepository(db)
	b.bankRepo = repository.NewBankRepository(db)
	b.contactRepo = repository.NewContactRepository(db)
	b.invoiceRepo = repository.NewInvoiceRepository(db)
	b.auditRepo = repository.NewAuditRepository(db)
	b.settingsRepo = repository.NewSettingsRepository(db)
	b.festschreibungRepo = repository.NewFestschreibungRepository(db)
	b.allocationRepo = repository.NewPaymentAllocationRepository(db)
	b.receiptRepo = repository.NewReceiptRepository(db)
	b.assetRepo = repository.NewAssetRepository(db)
	b.foundationRepo = repository.NewFoundationRepository(db)
	b.fiscalYearRepo = repository.NewFiscalYearRepository(db)

	// Determine active fiscal year from settings or fallback
	fiscalYear := b.currentYear
	if fiscalYear == 0 {
		fiscalYear = time.Now().Year()
	}
	if s, err := b.settingsRepo.GetCompanySettings(context.Background()); err == nil && s != nil {
		if s.FiscalYear > 0 {
			fiscalYear = s.FiscalYear
		} else {
			fiscalYear = domain.GetFiscalYearForDate(time.Now().Format("2006-01-02"), s.FiscalYearStartMonth)
		}
	}
	b.currentYear = fiscalYear

	// Wire Services. JournalService is the single write path; everything that
	// produces a booking goes through it.
	b.journalSvc = service.NewJournalService(b.journalRepo, b.accountRepo, b.contactRepo, b.auditRepo, b.settingsRepo, fiscalYear)
	b.journalSvc.SetFestschreibungRepo(b.festschreibungRepo)
	// Ein festgestellter Jahresabschluss nimmt keine Buchung mehr auf. Der
	// Journaldienst muss das wissen, bevor er schreibt.
	b.journalSvc.SetFiscalYearRepo(b.fiscalYearRepo)
	b.postingSvc = service.NewPostingService(b.journalSvc, b.contactRepo)
	b.receiptSvc = service.NewReceiptService(b.receiptRepo, b.journalRepo, receiptstore.New(t.DataDir), b.auditRepo, fiscalYear)
	b.postingSvc.SetReceiptService(b.receiptSvc)
	b.eInvoiceSvc = service.NewEInvoiceService(b.receiptSvc, b.contactRepo, invoice.NewReader(), fiscalYear)
	b.accountingSvc = service.NewAccountingService(b.accountRepo, b.journalRepo, b.contactRepo, b.settingsRepo, b.journalSvc, fiscalYear)
	b.bankSvc = service.NewBankService(b.bankRepo, b.journalSvc, b.auditRepo)
	b.paymentSvc = service.NewPaymentService(b.journalSvc, b.journalRepo, b.allocationRepo, b.contactRepo, b.bankRepo, fiscalYear)
	b.vatSvc = service.NewVatService(b.journalRepo, fiscalYear)
	b.invoiceSvc = service.NewInvoiceService(b.invoiceRepo, b.contactRepo, b.settingsRepo, b.numberRepo, b.postingSvc, b.auditRepo)
	// Der Renderer eines vorigen Mandanten hält eine WASM-Instanz von
	// zweistelliger Megabyte-Größe. Sie wird im Hintergrund freigegeben — ein
	// noch laufendes Warm würde den Mandantenwechsel sonst blockieren.
	if previous := b.renderer; previous != nil {
		go func() { _ = previous.Close(context.Background()) }()
	}
	b.renderer = invoice.NewRenderer()
	b.invoiceSvc.SetDocumentPipeline(b.receiptSvc, b.renderer)
	// Das WASM-Modul zu übersetzen kostet ein paar Sekunden. Die soll nicht
	// zahlen, wer auf "Rechnung ausstellen" drückt.
	go func(r *invoice.Renderer) { _ = r.Warm(context.Background()) }(b.renderer)
	b.contactSvc = service.NewContactService(b.contactRepo, b.journalRepo, b.numberRepo, b.auditRepo, fiscalYear)
	b.assetSvc = service.NewAssetService(
		b.assetRepo, b.journalRepo, b.journalSvc, b.numberRepo,
		b.contactRepo, b.settingsRepo, b.auditRepo, fiscalYear,
	)
	// Ein Skonto auf eine Anlagenrechnung mindert die Anschaffungskosten
	// (§ 255 Abs. 1 Satz 3 HGB). Damit der Zahlungsflow das erkennen kann,
	// braucht er die Kartei — mehr von ihr nicht.
	b.paymentSvc.SetAssetRegister(b.assetSvc)
	// Verträge, Gutachten und Zulassungen zum Anlagegut liegen im selben
	// inhaltsadressierten Speicher wie die Belege, nur in einem anderen Zweig.
	b.assetSvc.SetDocumentStore(receiptstore.New(t.DataDir))
	b.auditSvc = service.NewAuditService(b.auditRepo)
	b.settingsSvc = service.NewSettingsService(b.settingsRepo, b.auditRepo)
	b.currencySvc = service.NewCurrencyService()
	// Die Gründungsbegleitung liest das Journal und schreibt über den
	// JournalService wie jeder andere Weg auch — eine Gründungsbuchung ist
	// keine Buchung zweiter Klasse.
	b.foundationSvc = service.NewFoundationService(
		b.foundationRepo, b.accountRepo, b.journalRepo, b.settingsRepo,
		b.journalSvc, b.auditRepo, fiscalYear,
	)

	b.closingSvc = service.NewClosingService(
		b.fiscalYearRepo, b.journalRepo, b.accountRepo, b.contactRepo, b.allocationRepo,
		b.settingsRepo, b.festschreibungRepo, b.auditRepo, b.journalSvc, fiscalYear,
	)
	b.closingSvc.SetFoundationRepo(b.foundationRepo)

	// Bilanz und Gewinn- und Verlustrechnung entstehen im Backend, nicht mehr
	// in der Ansicht. Der Dienst braucht dazu das Geschäftsjahr (Stichtag und
	// Zeitraum), die Salden zweier Jahre, die offenen Posten für die
	// Restlaufzeiten und den Renderer für die Ausgabe als PDF.
	b.statementSvc = service.NewStatementService(
		b.accountingSvc, b.closingSvc, b.settingsRepo, b.auditRepo, fiscalYear)
	b.statementSvc.SetOpenItemSource(b.paymentSvc)
	b.statementSvc.SetRenderer(b.renderer)

	// Die E-Bilanz entsteht aus derselben Gliederung wie die Bilanz auf dem
	// Schirm — zwei Wege zu einer Bilanz wären einer zu viel. Den
	// Anlagenspiegel braucht sie als Kontennachweis: die Bilanz zeigt einen
	// Buchwert, und erst er zeigt, woraus er entstanden ist.
	b.ebilanzSvc = service.NewEBilanzService(
		b.statementSvc, b.settingsRepo, b.auditRepo, fiscalYear)
	b.ebilanzSvc.SetAnlagenspiegelSource(b.assetSvc)
	// Bestehende Datenbanken kennen das Geschäftsjahr nur als Zahl an der
	// Buchung. Die Entitäten dazu entstehen beim ersten Start nach der
	// Umstellung; scheitert das, bleibt der Mandant benutzbar und die Ansicht
	// „Jahresabschluss" legt das Jahr beim ersten Aufruf an.
	if err := b.closingSvc.EnsureFiscalYears(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Geschäftsjahre konnten nicht angelegt werden: %v\n", err)
	}

	b.appConfig.ActiveTenantID = t.ID
	b.appConfig.DataDir = t.DataDir
	b.appConfig.LastFiscalYear = fiscalYear
	_ = b.appCfgRepo.Save(&b.appConfig)

	return nil
}

// -------------------------------------------------------------
// MULTI-TENANT MANAGEMENT (MANDANTENVERWALTUNG)
// -------------------------------------------------------------

func (b *BuchfinkBridge) GetTenants() []domain.TenantConfig {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.appConfig.Tenants
}

func (b *BuchfinkBridge) GetActiveTenant() (*domain.TenantConfig, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.appConfig.ActiveTenantID != "" {
		for _, t := range b.appConfig.Tenants {
			if t.ID == b.appConfig.ActiveTenantID {
				return &t, nil
			}
		}
	}
	if len(b.appConfig.Tenants) > 0 {
		return &b.appConfig.Tenants[0], nil
	}
	return nil, nil
}

func (b *BuchfinkBridge) SwitchTenant(tenantID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, t := range b.appConfig.Tenants {
		if t.ID == tenantID {
			return b.initTenant(&t)
		}
	}
	return fmt.Errorf("tenant ID %s not found", tenantID)
}

// IsLocked reports whether the active tenant is encrypted but cannot be unlocked
// on this machine (keychain secret missing). The frontend uses this to show a
// recovery screen instead of the normal app.
func (b *BuchfinkBridge) IsLocked() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.locked
}

// activeTenantLocked returns a pointer to the active tenant config in the slice
// (so edits persist). Caller must hold the lock.
func (b *BuchfinkBridge) activeTenantLocked() *domain.TenantConfig {
	for i := range b.appConfig.Tenants {
		if b.appConfig.Tenants[i].ID == b.appConfig.ActiveTenantID {
			return &b.appConfig.Tenants[i]
		}
	}
	if len(b.appConfig.Tenants) > 0 {
		return &b.appConfig.Tenants[0]
	}
	return nil
}

// ExportRecoveryKey writes an external recovery file for the active tenant into a
// user-chosen folder (e.g. a USB stick) and returns the full path. Keep this file
// safe and SEPARATE from your data backup: together they can decrypt the
// accounting data, neither half alone can.
func (b *BuchfinkBridge) ExportRecoveryKey() (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.vault == nil {
		return "", fmt.Errorf("kein entsperrter Mandant aktiv – Recovery-Export nicht möglich")
	}
	active := b.activeTenantLocked()
	if active == nil {
		return "", fmt.Errorf("kein aktiver Mandant gefunden")
	}

	targetDir := ""
	if app := application.Get(); app != nil && app.Dialog != nil {
		dir, err := app.Dialog.OpenFile().
			CanChooseDirectories(true).
			CanChooseFiles(false).
			CanCreateDirectories(true).
			SetTitle("Speicherort für den Recovery-Schlüssel wählen (z. B. USB-Stick)").
			PromptForSingleSelection()
		if err != nil {
			return "", err
		}
		targetDir = dir
	}
	if targetDir == "" {
		return "", fmt.Errorf("kein Zielordner gewählt")
	}

	data, err := security.ExportTenantRecoveryFile(active.DataDir, active.ID, active.Name, b.vault)
	if err != nil {
		return "", err
	}
	fullPath := filepath.Join(targetDir, fmt.Sprintf("buchfink-recovery-%s.json", active.ID))
	if err := os.WriteFile(fullPath, data, 0600); err != nil {
		return "", fmt.Errorf("Recovery-Datei schreiben: %w", err)
	}
	return fullPath, nil
}

// SelectRecoveryFileDialog opens a native picker for a .json recovery file.
func (b *BuchfinkBridge) SelectRecoveryFileDialog(title string) (string, error) {
	if title == "" {
		title = "Recovery-Schlüsseldatei auswählen"
	}
	app := application.Get()
	if app == nil || app.Dialog == nil {
		return "", nil
	}
	return app.Dialog.OpenFile().
		CanChooseFiles(true).
		CanChooseDirectories(false).
		SetTitle(title).
		PromptForSingleSelection()
}

// RecoverActiveTenantFromFile unlocks the active tenant from an external recovery
// file when its keychain secret is missing (new machine / lost keychain). On
// success it re-provisions the local keychain and fully initializes the tenant.
func (b *BuchfinkBridge) RecoverActiveTenantFromFile(recoveryFilePath string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	active := b.activeTenantLocked()
	if active == nil {
		return fmt.Errorf("kein aktiver Mandant gefunden")
	}
	data, err := os.ReadFile(recoveryFilePath)
	if err != nil {
		return fmt.Errorf("Recovery-Datei lesen: %w", err)
	}
	if _, err := security.RecoverTenantFromFile(active.DataDir, active.ID, data); err != nil {
		return err
	}
	// Keychain is re-provisioned; a full init now unlocks transparently.
	b.locked = false
	return b.initTenant(active)
}

func (b *BuchfinkBridge) CreateTenant(
	name string,
	dataDir string,
	settings domain.CompanySettings,
) (*domain.TenantConfig, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if name == "" {
		if settings.CompanyName != "" {
			name = settings.CompanyName
		} else {
			name = "Neuer Mandant"
		}
	}

	tenantID := fmt.Sprintf("tenant_%d", time.Now().UnixNano())

	homeDir, _ := os.UserHomeDir()
	if dataDir == "" {
		dataDir = filepath.Join(homeDir, ".buchfink", "tenants", tenantID, "data")
	}

	// Persist absolute paths so tenant locations are stable regardless of the
	// process working directory (relative paths would break e.g. in dev mode).
	if abs, err := filepath.Abs(dataDir); err == nil {
		dataDir = abs
	}

	// Provision transparent field encryption: generates the envelope keyfile in
	// the data dir and stores the wrapping secret in the OS keychain. initTenant
	// then opens it. Fails closed — no tenant without encryption provisioned.
	if _, err := security.CreateTenantVault(dataDir, tenantID); err != nil {
		return nil, fmt.Errorf("failed to provision tenant encryption: %w", err)
	}

	fiscalYear := settings.FiscalYear
	if fiscalYear == 0 {
		fiscalYear = time.Now().Year()
	}

	t := domain.TenantConfig{
		ID:        tenantID,
		Name:      name,
		DataDir:   dataDir,
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	b.appConfig.Tenants = append(b.appConfig.Tenants, t)
	b.appConfig.ActiveTenantID = tenantID
	b.appConfig.IsConfigured = true
	b.appConfig.LastFiscalYear = fiscalYear

	if err := b.initTenant(&t); err != nil {
		return nil, fmt.Errorf("failed to init tenant DB: %w", err)
	}

	// Update company profile in new database
	if settings.CompanyName == "" {
		settings.CompanyName = name
	}
	settings.FiscalYear = fiscalYear
	if err := b.settingsSvc.UpdateCompanySettings(context.Background(), &settings); err != nil {
		return nil, fmt.Errorf("failed to save initial company settings: %w", err)
	}

	_ = b.appCfgRepo.Save(&b.appConfig)
	return &t, nil
}

func (b *BuchfinkBridge) ImportTenant(dbFilePath string) (*domain.TenantConfig, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, err := os.Stat(dbFilePath); err != nil {
		return nil, fmt.Errorf("database file not found: %s", dbFilePath)
	}

	dataDir := filepath.Dir(dbFilePath)
	tenantID := fmt.Sprintf("tenant_%d", time.Now().UnixNano())
	name := filepath.Base(dataDir)
	if name == "data" || name == "." || name == "" {
		name = fmt.Sprintf("Mandant (%s)", filepath.Base(dbFilePath))
	}

	// An imported database keeps whatever encryption state it shipped with: if a
	// keyfile sits next to it (and the keychain secret is present) initTenant
	// unlocks it, otherwise it is treated as clear text.
	t := domain.TenantConfig{
		ID:        tenantID,
		Name:      name,
		DataDir:   dataDir,
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	b.appConfig.Tenants = append(b.appConfig.Tenants, t)
	b.appConfig.ActiveTenantID = tenantID
	b.appConfig.IsConfigured = true

	if err := b.initTenant(&t); err != nil {
		return nil, err
	}

	// Try reading company name from settings
	if s, err := b.settingsSvc.GetCompanySettings(context.Background()); err == nil && s != nil && s.CompanyName != "" {
		t.Name = s.CompanyName
		for i := range b.appConfig.Tenants {
			if b.appConfig.Tenants[i].ID == tenantID {
				b.appConfig.Tenants[i].Name = s.CompanyName
				break
			}
		}
		_ = b.appCfgRepo.Save(&b.appConfig)
	}

	return &t, nil
}

func (b *BuchfinkBridge) DeleteTenant(tenantID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	var newTenants []domain.TenantConfig
	for _, t := range b.appConfig.Tenants {
		if t.ID != tenantID {
			newTenants = append(newTenants, t)
		}
	}
	b.appConfig.Tenants = newTenants

	// Remove the tenant's wrapping secret from the OS keychain. The keyfile and
	// data on disk are left untouched (deletion of user data stays explicit).
	_ = security.DeleteTenantSecret(tenantID)

	if b.appConfig.ActiveTenantID == tenantID {
		if len(newTenants) > 0 {
			b.appConfig.ActiveTenantID = newTenants[0].ID
			_ = b.initTenant(&newTenants[0])
		} else {
			b.appConfig.ActiveTenantID = ""
			b.appConfig.IsConfigured = false
			b.db = nil
		}
	}

	return b.appCfgRepo.Save(&b.appConfig)
}

// -------------------------------------------------------------
// SETUP & ONBOARDING ASSISTANT (FIRST LAUNCH & VAULT)
// -------------------------------------------------------------

func (b *BuchfinkBridge) GetAppConfig() domain.AppConfig {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.appConfig
}

// SetupApplication handles initial setup: establishes the data directory,
// provisions transparent field encryption, and configures the company.
func (b *BuchfinkBridge) SetupApplication(
	dataDir string,
	settings domain.CompanySettings,
) error {
	name := settings.CompanyName
	if name == "" {
		name = "Hauptmandant"
	}
	_, err := b.CreateTenant(name, dataDir, settings)
	return err
}

// LoadExistingDatabase loads an existing SQLite database file from disk.
func (b *BuchfinkBridge) LoadExistingDatabase(dbFilePath string) error {
	_, err := b.ImportTenant(dbFilePath)
	return err
}

// SelectDirectoryDialog opens a native OS folder picker.
func (b *BuchfinkBridge) SelectDirectoryDialog(title string) (string, error) {
	if title == "" {
		title = "Buchfink Datenordner auswählen"
	}
	app := application.Get()
	if app == nil || app.Dialog == nil {
		return "", nil
	}
	return app.Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		CanCreateDirectories(true).
		SetTitle(title).
		PromptForSingleSelection()
}

// SelectDatabaseFileDialog opens a native OS file picker for SQLite database files.
func (b *BuchfinkBridge) SelectDatabaseFileDialog(title string) (string, error) {
	if title == "" {
		title = "Buchfink Buchhaltungsdatei auswählen"
	}
	app := application.Get()
	if app == nil || app.Dialog == nil {
		return "", nil
	}
	return app.Dialog.OpenFile().
		CanChooseFiles(true).
		CanChooseDirectories(false).
		AddFilter("Buchfink Buchhaltungsdateien (*.sqlite, *.db)", "*.sqlite;*.db").
		SetTitle(title).
		PromptForSingleSelection()
}

// -------------------------------------------------------------
// DYNAMIC FISCAL YEARS & FILTERING
// -------------------------------------------------------------

// GetAvailableFiscalYears lists the fiscal years the user can switch to: the
// union of the recorded fiscal years and those that only appear in the journal.
//
// Beide Quellen, weil beide unvollständig sind. Ein neu angelegtes Geschäftsjahr
// hat noch keine Buchung, und eine gewachsene Datenbank hat Buchungen in Jahren,
// zu denen die Entität erst nachträglich entsteht.
func (b *BuchfinkBridge) GetAvailableFiscalYears() []int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accountingSvc == nil {
		return repository.DiscoverAvailableFiscalYears(b.dataDir)
	}

	years := b.accountingSvc.GetAvailableFiscalYears(context.Background())
	if b.closingSvc == nil {
		return years
	}
	known := make(map[int]bool, len(years))
	for _, y := range years {
		known[y] = true
	}
	recorded, err := b.closingSvc.FiscalYears(context.Background())
	if err != nil {
		return years
	}
	for _, fy := range recorded {
		if fy.Year > 0 && !known[fy.Year] {
			known[fy.Year] = true
			years = append(years, fy.Year)
		}
	}
	sort.Ints(years)
	return years
}

// setFiscalYearLocked switches the active year across every service that
// filters by it. Caller must hold the lock.
func (b *BuchfinkBridge) setFiscalYearLocked(year int) {
	b.currentYear = year
	if b.accountingSvc != nil {
		b.accountingSvc.SetFiscalYear(year)
	}
	if b.journalSvc != nil {
		b.journalSvc.SetFiscalYear(year)
	}
	if b.paymentSvc != nil {
		b.paymentSvc.SetFiscalYear(year)
	}
	if b.contactSvc != nil {
		b.contactSvc.SetFiscalYear(year)
	}
	if b.vatSvc != nil {
		b.vatSvc.SetFiscalYear(year)
	}
	if b.receiptSvc != nil {
		b.receiptSvc.SetFiscalYear(year)
	}
	if b.eInvoiceSvc != nil {
		b.eInvoiceSvc.SetFiscalYear(year)
	}
	if b.assetSvc != nil {
		b.assetSvc.SetFiscalYear(year)
	}
	if b.foundationSvc != nil {
		b.foundationSvc.SetFiscalYear(year)
	}
	if b.statementSvc != nil {
		b.statementSvc.SetFiscalYear(year)
	}
	if b.ebilanzSvc != nil {
		b.ebilanzSvc.SetFiscalYear(year)
	}
	if b.closingSvc != nil {
		b.closingSvc.SetFiscalYear(year)
	}
	b.appConfig.LastFiscalYear = year
	_ = b.appCfgRepo.Save(&b.appConfig)
}

func (b *BuchfinkBridge) GetFiscalYear() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.currentYear
}

func (b *BuchfinkBridge) SetFiscalYear(year int) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.setFiscalYearLocked(year)
	return nil
}

// -------------------------------------------------------------
// COMPANY SETTINGS & STAMMDATEN
// -------------------------------------------------------------

func (b *BuchfinkBridge) GetCompanySettings() (*domain.CompanySettings, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.settingsSvc == nil {
		return &domain.CompanySettings{FiscalYear: time.Now().Year(), FiscalYearStartMonth: 1, Currency: "EUR", SKR: "SKR04", TaxationType: "SOLL"}, nil
	}
	return b.settingsSvc.GetCompanySettings(context.Background())
}

func (b *BuchfinkBridge) UpdateCompanySettings(settings domain.CompanySettings) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.settingsSvc == nil {
		return nil
	}
	// Also sync tenant name if active
	if settings.CompanyName != "" && b.appConfig.ActiveTenantID != "" {
		for i := range b.appConfig.Tenants {
			if b.appConfig.Tenants[i].ID == b.appConfig.ActiveTenantID {
				b.appConfig.Tenants[i].Name = settings.CompanyName
				_ = b.appCfgRepo.Save(&b.appConfig)
				break
			}
		}
	}
	return b.settingsSvc.UpdateCompanySettings(context.Background(), &settings)
}

// -------------------------------------------------------------
// ACCOUNTING & JOURNAL
// -------------------------------------------------------------

func (b *BuchfinkBridge) GetAccounts() ([]domain.Account, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accountingSvc == nil {
		return nil, nil
	}
	return b.accountingSvc.GetAccounts(context.Background())
}

func (b *BuchfinkBridge) GetAccountByNumber(number string) (*domain.Account, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accountingSvc == nil {
		return nil, fmt.Errorf("accounting service not initialized")
	}
	return b.accountingSvc.GetAccountByNumber(context.Background(), number)
}

func (b *BuchfinkBridge) GetAccountLedger(accountNumber string) (*domain.AccountLedger, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accountingSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.accountingSvc.GetAccountLedger(context.Background(), accountNumber)
}

func (b *BuchfinkBridge) GetSuSaOverview() (*domain.SuSaOverview, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accountingSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.accountingSvc.GetSuSaOverview(context.Background())
}

func (b *BuchfinkBridge) GetSKR04Catalog() (*accounting.SKR04Catalog, error) {
	return accounting.GetSKR04Catalog()
}

// GetPostingGroups returns the fachliche Gruppen the user picks from instead of
// account numbers. "incoming" for Eingangsbelege, "outgoing" for Erlöse, empty
// for both.
func (b *BuchfinkBridge) GetPostingGroups(direction string) []accounting.PostingGroup {
	return accounting.PostingGroups(domain.Direction(direction))
}

// GetTaxTreatments returns the Steuerfälle valid for a direction, with the hints
// the UI shows next to them.
func (b *BuchfinkBridge) GetTaxTreatments(direction string) []domain.TaxTreatmentInfo {
	return domain.TaxTreatments(domain.Direction(direction))
}

// GetPaymentAccounts lists the accounts that can settle a document immediately.
func (b *BuchfinkBridge) GetPaymentAccounts() ([]domain.Account, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accountingSvc == nil {
		return nil, nil
	}
	all, err := b.accountingSvc.GetAccounts(context.Background())
	if err != nil {
		return nil, err
	}
	liquid := map[string]bool{}
	for _, a := range domain.LiquidAccounts() {
		liquid[a] = true
	}
	var result []domain.Account
	for _, a := range all {
		if liquid[a.Number] {
			result = append(result, a)
		}
	}
	return result, nil
}

func (b *BuchfinkBridge) GetJournalEntries() ([]domain.JournalEntry, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accountingSvc == nil {
		return nil, nil
	}
	return b.accountingSvc.GetEntries(context.Background())
}

func (b *BuchfinkBridge) GetAllJournalEntries() ([]domain.JournalEntry, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accountingSvc == nil {
		return nil, nil
	}
	return b.accountingSvc.GetAllEntries(context.Background())
}

// PostJournalEntry books a manually composed Buchungssatz. The journal enforces
// the rules; the frontend only collects the input.
func (b *BuchfinkBridge) PostJournalEntry(entry domain.JournalEntry) (*domain.JournalEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.journalSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.journalSvc.Post(context.Background(), &entry)
}

// PostIncomingReceipt books an Eingangsbeleg from the fachliche Gruppe, the
// Steuerfall and the payment state.
func (b *BuchfinkBridge) PostIncomingReceipt(req service.ReceiptRequest) (*domain.JournalEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.postingSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.postingSvc.PostIncomingReceipt(context.Background(), req)
}

// ReverseJournalEntry cancels a booking by Generalumkehr.
func (b *BuchfinkBridge) ReverseJournalEntry(entryID uint, reason string) (*domain.JournalEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.journalSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.journalSvc.Reverse(context.Background(), entryID, reason)
}

func (b *BuchfinkBridge) VerifyIntegrity() (domain.IntegrityCheckResult, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.journalSvc == nil {
		return domain.IntegrityCheckResult{IsValid: true, Message: "Bereit"}, nil
	}
	return b.journalSvc.VerifyIntegrity(context.Background())
}

func (b *BuchfinkBridge) GetFinancialSummary() (*domain.FinancialSummary, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accountingSvc == nil {
		return &domain.FinancialSummary{}, nil
	}
	return b.accountingSvc.GetFinancialSummary(context.Background())
}

// -------------------------------------------------------------
// BANK IMPORT & RECONCILIATION
// -------------------------------------------------------------

func (b *BuchfinkBridge) GetBankTransactions() ([]domain.BankTransaction, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.bankSvc == nil {
		return nil, nil
	}
	return b.bankSvc.GetTransactions(context.Background(), b.currentYear)
}

// ImportCAMT053XML imports a statement for one liquid account. The account is an
// explicit parameter: a company usually has more than one.
func (b *BuchfinkBridge) ImportCAMT053XML(xmlContent string, ledgerAccount string) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.bankSvc == nil {
		return 0, fmt.Errorf("Bankimport ist noch nicht initialisiert")
	}
	return b.bankSvc.ImportCAMT053(context.Background(), bytes.NewBufferString(xmlContent), ledgerAccount)
}

// BookBankTransactionDirect books a statement line that has no document behind
// it (fees, interest, transfers between own accounts).
func (b *BuchfinkBridge) BookBankTransactionDirect(bankTxID uint, counterAccount, description string) (*domain.JournalEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.bankSvc == nil {
		return nil, fmt.Errorf("Bankimport ist noch nicht initialisiert")
	}
	return b.bankSvc.BookDirect(context.Background(), bankTxID, counterAccount, description)
}

func (b *BuchfinkBridge) IgnoreBankTransaction(bankTxID uint) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.bankSvc == nil {
		return fmt.Errorf("Bankimport ist noch nicht initialisiert")
	}
	return b.bankSvc.Ignore(context.Background(), bankTxID)
}

// -------------------------------------------------------------
// CONTACTS & INVOICES
// -------------------------------------------------------------

func (b *BuchfinkBridge) GetContacts() ([]domain.Contact, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.contactSvc == nil {
		return nil, nil
	}
	return b.contactSvc.GetContacts(context.Background())
}

func (b *BuchfinkBridge) SaveContact(c domain.Contact) (*domain.Contact, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.contactSvc == nil {
		return nil, fmt.Errorf("Stammdaten sind noch nicht initialisiert")
	}
	if err := b.contactSvc.SaveContact(context.Background(), &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (b *BuchfinkBridge) DeleteContact(id uint) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.contactSvc == nil {
		return fmt.Errorf("Stammdaten sind noch nicht initialisiert")
	}
	return b.contactSvc.DeleteContact(context.Background(), id)
}

func (b *BuchfinkBridge) GetInvoices() ([]domain.Invoice, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.invoiceSvc == nil {
		return nil, nil
	}
	return b.invoiceSvc.GetInvoices(context.Background(), b.currentYear)
}

// IssueInvoice assigns the consecutive invoice number and books the receivable
// in one step — an invoice never exists on paper without being in the journal.
func (b *BuchfinkBridge) IssueInvoice(inv domain.Invoice) (*domain.Invoice, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.invoiceSvc == nil {
		return nil, fmt.Errorf("Rechnungswesen ist noch nicht initialisiert")
	}
	if err := b.invoiceSvc.Issue(context.Background(), &inv); err != nil {
		return nil, err
	}
	return &inv, nil
}

func (b *BuchfinkBridge) CancelInvoice(invoiceID uint, reason string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.invoiceSvc == nil {
		return fmt.Errorf("Rechnungswesen ist noch nicht initialisiert")
	}
	return b.invoiceSvc.Cancel(context.Background(), invoiceID, reason)
}

func (b *BuchfinkBridge) GenerateInvoiceZUGFeRD(invoiceID uint) (string, string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.invoiceSvc == nil {
		return "", "", fmt.Errorf("Rechnungswesen ist noch nicht initialisiert")
	}
	return b.invoiceSvc.GenerateZUGFeRDAndTypst(context.Background(), invoiceID)
}

// -------------------------------------------------------------
// E-BILANZ & AUDIT TRAIL
// -------------------------------------------------------------

// -------------------------------------------------------------
// BELEGE
// -------------------------------------------------------------

// ReceiptFileInput is one file on its way into a Beleg.
//
// Files travel as paths, not as content: the native dialog and drag & drop both
// hand over paths, and a multi-megabyte scan has no business crossing the IPC
// boundary base64-encoded.
type ReceiptFileInput struct {
	Path string `json:"path"`
	Role string `json:"role"`
}

// SelectReceiptFilesDialog opens the native picker for Belegdateien.
func (b *BuchfinkBridge) SelectReceiptFilesDialog(title string) ([]string, error) {
	if title == "" {
		title = "Belegdateien auswählen"
	}
	app := application.Get()
	if app == nil || app.Dialog == nil {
		return nil, nil
	}
	return app.Dialog.OpenFile().
		CanChooseFiles(true).
		CanChooseDirectories(false).
		AddFilter("Belege (PDF, Bild, XML)", "*.pdf;*.png;*.jpg;*.jpeg;*.tif;*.tiff;*.webp;*.xml").
		SetTitle(title).
		PromptForMultipleSelection()
}

// FileIncomingReceipt files an incoming Beleg away without booking it. The two
// are separate steps: an XRechnung has to be storable the moment it arrives, and
// only becomes bookable once a rendering exists.
func (b *BuchfinkBridge) FileIncomingReceipt(receivedAt, receivedVia string, files []ReceiptFileInput) (*domain.Receipt, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.receiptSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.receiptSvc.File(context.Background(), service.FileReceiptRequest{
		Direction:   domain.DirectionIncoming,
		ReceivedAt:  receivedAt,
		ReceivedVia: receivedVia,
		Files:       toNewFiles(files),
	})
}

// AddReceiptFile appends a file to a Beleg that is not yet booked.
func (b *BuchfinkBridge) AddReceiptFile(receiptID uint, file ReceiptFileInput) (*domain.Receipt, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.receiptSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.receiptSvc.AddFile(context.Background(), receiptID, toNewFiles([]ReceiptFileInput{file})[0])
}

// RemoveReceiptFile drops a file from a Beleg that is not yet booked.
func (b *BuchfinkBridge) RemoveReceiptFile(receiptID, fileID uint) (*domain.Receipt, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.receiptSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.receiptSvc.RemoveFile(context.Background(), receiptID, fileID)
}

// GetReceipts lists the Belege of the active fiscal year. An empty status
// returns all of them.
func (b *BuchfinkBridge) GetReceipts(status string) ([]domain.Receipt, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.receiptSvc == nil {
		return []domain.Receipt{}, nil
	}
	return b.receiptSvc.List(context.Background(), domain.ReceiptStatus(status))
}

// GetReceipt returns one Beleg with its files.
// ReceiptService.Get repariert dabei ein fehlendes Siegel und schreibt deshalb —
// darum die Schreibsperre.
func (b *BuchfinkBridge) GetReceipt(id uint) (*domain.Receipt, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.receiptSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.receiptSvc.Get(context.Background(), id)
}

// DiscardReceipt retires a filed Beleg. It keeps its number and stays findable —
// a received document must not vanish without a trace.
func (b *BuchfinkBridge) DiscardReceipt(id uint, reason string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.receiptSvc == nil {
		return fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.receiptSvc.Discard(context.Background(), id, reason)
}

// ReceiptPreview is what the frontend shows for a Beleg.
type ReceiptPreview struct {
	// DataURL carries the displayable file inline so the frontend can render it
	// without a second transport.
	DataURL  string `json:"dataUrl"`
	FileName string `json:"fileName"`
	MimeType string `json:"mimeType"`
	// Intact is false when the file on disk no longer matches the checksum it
	// was filed under. It is shown rather than hidden.
	Intact bool `json:"intact"`
}

// maxPreviewBytes caps what is inlined into the IPC response. Larger files are
// reported rather than silently truncated.
const maxPreviewBytes = 32 << 20

// GetReceiptPreview returns the file the user is shown for a Beleg — the
// original for a PDF or an image, the generated rendering for an XRechnung.
//
// On a hybrid Beleg this is the image part, which is display only: booking reads
// the structured part. A difference between the two is potentially a second
// invoice with § 14c consequences and is shown, not judged.
// ReceiptService.Get repariert dabei ein fehlendes Siegel und schreibt deshalb —
// darum die Schreibsperre.
func (b *BuchfinkBridge) GetReceiptPreview(receiptID uint) (*ReceiptPreview, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.receiptSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	content, err := b.receiptSvc.DisplayContent(context.Background(), receiptID)
	if err != nil {
		return nil, err
	}
	if len(content.Data) > maxPreviewBytes {
		return nil, fmt.Errorf("die Belegdatei %s ist zu groß für die Vorschau (%d MB)", content.FileName, len(content.Data)>>20)
	}
	return &ReceiptPreview{
		DataURL:  "data:" + content.MimeType + ";base64," + base64.StdEncoding.EncodeToString(content.Data),
		FileName: content.FileName,
		MimeType: content.MimeType,
		Intact:   content.Intact,
	}, nil
}

// GetInvoiceDocument returns the issued invoice as a data URL for display and
// download. It is the same hybrid PDF that was archived when the invoice was
// issued, not a fresh rendering — what the customer received is what is shown.
// ReceiptService.Get repariert dabei ein fehlendes Siegel und schreibt deshalb —
// darum die Schreibsperre.
func (b *BuchfinkBridge) GetInvoiceDocument(invoiceID uint) (*ReceiptPreview, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.invoiceSvc == nil || b.receiptSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	inv, err := b.invoiceRepo.FindByID(context.Background(), invoiceID)
	if err != nil {
		return nil, err
	}
	if inv.ReceiptID == nil {
		return nil, fmt.Errorf("zu Rechnung %s liegt kein Belegdokument vor", inv.InvoiceNumber)
	}
	content, err := b.receiptSvc.DisplayContent(context.Background(), *inv.ReceiptID)
	if err != nil {
		return nil, err
	}
	if len(content.Data) > maxPreviewBytes {
		return nil, fmt.Errorf("das Rechnungsdokument ist zu groß für die Anzeige")
	}
	return &ReceiptPreview{
		DataURL:  "data:" + content.MimeType + ";base64," + base64.StdEncoding.EncodeToString(content.Data),
		FileName: content.FileName,
		MimeType: content.MimeType,
		Intact:   content.Intact,
	}, nil
}

// ExtractStructuredPart pulls the invoice data out of a filed Beleg and attaches
// it under the role `structured`.
//
// It is a step of its own because filing must not depend on it: a Beleg is kept
// in the form it arrived, and only then is it examined.
func (b *BuchfinkBridge) ExtractStructuredPart(receiptID uint) (*domain.Receipt, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.eInvoiceSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.eInvoiceSvc.ExtractStructuredPart(context.Background(), receiptID)
}

// ProposeFromEInvoice turns the structured part of a Beleg into a prefilled
// booking. The fachliche Gruppe stays open — no invoice knows which expense
// account a supply belongs on.
// ReceiptService.Get repariert dabei ein fehlendes Siegel und schreibt deshalb —
// darum die Schreibsperre.
func (b *BuchfinkBridge) ProposeFromEInvoice(receiptID uint) (*service.EInvoiceProposal, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.eInvoiceSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.eInvoiceSvc.Propose(context.Background(), receiptID)
}

// ValidateEInvoice re-runs the EN-16931 rule check against the structured part of
// a Beleg and records the result.
//
// The rule set is versioned, so a document checked under an older version can be
// checked again without being re-filed. The check touches no file, which is why
// it is allowed on a booked Beleg too.
func (b *BuchfinkBridge) ValidateEInvoice(receiptID uint) (*domain.ReceiptValidation, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.eInvoiceSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.eInvoiceSvc.Validate(context.Background(), receiptID)
}

// GetEInvoiceRules lists the rules Buchfink checks. It is deliberately exposed:
// "validated" without the list of what was checked tells a user nothing they
// can act on.
//
// The list covers EN 16931 completely and the German CIUS as far as its rules
// can be decided on the semantic model. GetUncheckedEInvoiceRules names the
// rest, with the reason for each.
func (b *BuchfinkBridge) GetEInvoiceRules() []string {
	rules := einvoice.RulesChecked()
	rules = append(rules, xrechnung.CheckedRules()...)
	rules = append(rules, zugferd.CheckedRules()...)
	sort.Strings(rules)
	return rules
}

// GetUncheckedEInvoiceRules names the rules of the specifications Buchfink does
// not check, each with the reason.
//
// Silence about a gap is worse than the gap: a user told "geprüft" deserves to
// know what was not looked at.
func (b *BuchfinkBridge) GetUncheckedEInvoiceRules() map[string]string {
	out := map[string]string{}
	for _, rule := range einvoice.RulesUnchecked() {
		out[rule] = "keine Regel der Norm bleibt ungeprüft"
	}
	for rule, reason := range xrechnung.UncheckedRules() {
		out[rule] = reason
	}
	return out
}

// PreviewIncomingReceipt computes the booking without writing it, so the form can
// show the Buchungssatz instead of re-deriving it.
// ReceiptService.Get repariert dabei ein fehlendes Siegel und schreibt deshalb —
// darum die Schreibsperre.
func (b *BuchfinkBridge) PreviewIncomingReceipt(req service.ReceiptRequest) (*service.PostingPreview, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.postingSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.postingSvc.PreviewIncomingReceipt(context.Background(), req)
}

// PreviewOutgoingInvoice computes what an invoice would book, without issuing it.
func (b *BuchfinkBridge) PreviewOutgoingInvoice(inv domain.Invoice) (*service.PostingPreview, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.invoiceSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.invoiceSvc.Preview(context.Background(), &inv)
}

func toNewFiles(files []ReceiptFileInput) []service.NewFile {
	out := make([]service.NewFile, 0, len(files))
	for _, f := range files {
		role := domain.ReceiptFileRole(f.Role)
		if role == "" {
			role = domain.ReceiptRoleOriginal
		}
		out = append(out, service.NewFile{
			Path: f.Path,
			Role: role,
			// A rendering is the only role the user can pick that is by
			// definition produced rather than received.
			Derived: role == domain.ReceiptRoleRendering,
		})
	}
	return out
}

// GetLegalForms hands the Rechtsform catalog to the UI, jede mit dem, was sie
// steuerlich nach sich zieht.
//
// Die Rechtsform wird ohnehin eingetragen. Sie mitzuliefern statt die
// Anlegerstellung für § 20 InvStG gesondert abzufragen ist der Unterschied
// zwischen einer Einstellung und einer Prüfungsfrage.
func (b *BuchfinkBridge) GetLegalForms() []domain.LegalFormInfo {
	return domain.LegalFormCatalog()
}

func (b *BuchfinkBridge) ExportEBilanzXBRL() (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.ebilanzSvc == nil {
		return "", fmt.Errorf("ebilanz service not initialized")
	}
	return b.ebilanzSvc.ExportXBRL(context.Background(), b.currentYear)
}

func (b *BuchfinkBridge) GetAuditLogs() ([]domain.AuditLogEntry, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.auditSvc == nil {
		return nil, nil
	}
	return b.auditSvc.GetLogs(context.Background(), 200)
}

// -------------------------------------------------------------
// OFFENE POSTEN & ZAHLUNGSZUORDNUNG
// -------------------------------------------------------------

// GetOpenItems lists the unsettled receivables and payables. The amounts are
// derived from the journal and the recorded allocations, never stored, so they
// cannot drift apart from the bookings.
func (b *BuchfinkBridge) GetOpenItems() ([]domain.OpenItem, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.paymentSvc == nil {
		return nil, nil
	}
	return b.paymentSvc.OpenItems(context.Background())
}

// GetDifferenceKinds returns the payment difference kinds with their hints.
func (b *BuchfinkBridge) GetDifferenceKinds() []domain.DifferenceKindInfo {
	return domain.DifferenceKinds()
}

// SettlePayment books a payment against one or more open items, including
// Skonto, bank fees and rounding differences.
func (b *BuchfinkBridge) SettlePayment(req service.PaymentRequest) (*domain.JournalEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.paymentSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.paymentSvc.Settle(context.Background(), req)
}

// GetVatSummary aggregates the VAT figures of a period from the journal's tax
// lines. Empty bounds mean the whole fiscal year.
//
// This is an orientation figure, not an Umsatzsteuer-Voranmeldung — but it is
// derived from the actual Steuerschlüssel and Bemessungsgrundlagen rather than
// reconstructed from account numbers.
func (b *BuchfinkBridge) GetVatSummary(from, to string) (*domain.VatSummary, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.vatSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.vatSvc.Summary(context.Background(), from, to)
}

// -------------------------------------------------------------
// GRÜNDUNG (VORGESELLSCHAFT BIS EINTRAGUNG)
// -------------------------------------------------------------

// GetFoundationState returns everything the Gründungsansicht needs: die Regeln
// der Rechtsform, die erfasste Gründung, den Anmeldungsbefund, die Unterbilanz
// und die Fristen.
//
// Ein Aufruf, nicht fünf: die Teile hängen voneinander ab, und fünf Aufrufe
// könnten einen Stand zeigen, den es so nie gab.
func (b *BuchfinkBridge) GetFoundationState() (*service.FoundationState, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.foundationSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.foundationSvc.GetState(context.Background())
}

// SaveFoundation records or updates the Gründung of the active tenant.
func (b *BuchfinkBridge) SaveFoundation(f domain.Foundation) (*domain.Foundation, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.foundationSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.foundationSvc.Save(context.Background(), &f)
}

// GetFoundationRules returns the Kapitalaufbringungsregeln of every legal form
// the Gründungsweg covers. Der Einrichtungsassistent verzweigt daran, statt
// Rechtsformnamen zu vergleichen.
func (b *BuchfinkBridge) GetFoundationRules() []accounting.FoundationRules {
	return accounting.FoundationLegalForms()
}

// GetRecommendedVatPeriod is the Voranmeldungszeitraum a company founded in this
// year starts with, plus the reason. § 18 Abs. 2 UStG hat dafür ein Stichjahr,
// deshalb kommt die Antwort aus dem Code und steht nicht als Satz in der Maske.
func (b *BuchfinkBridge) GetRecommendedVatPeriod(foundingYear int) map[string]string {
	return map[string]string{
		"period": accounting.RecommendedVatPeriod(foundingYear),
		"reason": accounting.VatPeriodReason(foundingYear),
	}
}

// PreviewFoundationPostings shows the Gründungsbuchungen before they are written.
func (b *BuchfinkBridge) PreviewFoundationPostings() (*service.FoundationPostingPreview, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.foundationSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.foundationSvc.PreviewPostings(context.Background())
}

// BookFoundationPostings writes the released Gründungsbuchungen.
func (b *BuchfinkBridge) BookFoundationPostings() ([]domain.JournalEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.foundationSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.foundationSvc.BookPostings(context.Background())
}

// RegisterCompany records the entry in the Handelsregister and ends the
// Vorgesellschaft. Ab hier steht die Unterbilanz auf diesen Tag fest.
func (b *BuchfinkBridge) RegisterCompany(date, court, number string) (*domain.Foundation, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.foundationSvc == nil {
		return nil, fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.foundationSvc.Register(context.Background(), date, court, number)
}

// CompleteFoundationDuty records a fulfilled Gründungspflicht with the day it
// happened, or takes it back when the date is empty.
func (b *BuchfinkBridge) CompleteFoundationDuty(key, doneOn, note string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.foundationSvc == nil {
		return fmt.Errorf("Buchhaltung ist noch nicht initialisiert")
	}
	return b.foundationSvc.CompleteDuty(context.Background(), key, doneOn, note)
}
