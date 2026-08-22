// SPDX-FileCopyrightText: 2026 Maximilian P.
// SPDX-License-Identifier: EUPL-1.2

package wailsbridge

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
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
	auditRepo          domain.AuditRepository
	settingsRepo       domain.SettingsRepository
	festschreibungRepo domain.FestschreibungRepository

	// Services
	journalSvc    *service.JournalService
	postingSvc    *service.PostingService
	paymentSvc    *service.PaymentService
	vatSvc        *service.VatService
	accountingSvc *service.AccountingService
	bankSvc       *service.BankService
	invoiceSvc    *service.InvoiceService
	contactSvc    *service.ContactService
	ebilanzSvc    *service.EBilanzService
	auditSvc      *service.AuditService
	settingsSvc   *service.SettingsService
	currencySvc   *service.CurrencyService
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
	b.postingSvc = service.NewPostingService(b.journalSvc, b.contactRepo)
	b.accountingSvc = service.NewAccountingService(b.accountRepo, b.journalRepo, b.contactRepo, b.settingsRepo, b.journalSvc, fiscalYear)
	b.bankSvc = service.NewBankService(b.bankRepo, b.journalSvc, b.auditRepo)
	b.paymentSvc = service.NewPaymentService(b.journalSvc, b.journalRepo, b.allocationRepo, b.contactRepo, b.bankRepo, fiscalYear)
	b.vatSvc = service.NewVatService(b.journalRepo, fiscalYear)
	b.invoiceSvc = service.NewInvoiceService(b.invoiceRepo, b.contactRepo, b.settingsRepo, b.numberRepo, b.postingSvc, b.auditRepo)
	b.contactSvc = service.NewContactService(b.contactRepo, b.journalRepo, b.numberRepo, b.auditRepo, fiscalYear)
	b.ebilanzSvc = service.NewEBilanzService(b.accountingSvc, b.settingsRepo, b.auditRepo)
	b.auditSvc = service.NewAuditService(b.auditRepo)
	b.settingsSvc = service.NewSettingsService(b.settingsRepo, b.auditRepo)
	b.currencySvc = service.NewCurrencyService()

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

func (b *BuchfinkBridge) GetAvailableFiscalYears() []int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accountingSvc == nil {
		return repository.DiscoverAvailableFiscalYears(b.dataDir)
	}
	return b.accountingSvc.GetAvailableFiscalYears(context.Background())
}

func (b *BuchfinkBridge) CreateFiscalYear(year int) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.setFiscalYearLocked(year)
	return nil
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

func (b *BuchfinkBridge) ExportEBilanzXBRL() (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.ebilanzSvc == nil {
		return "", fmt.Errorf("ebilanz service not initialized")
	}
	return b.ebilanzSvc.ExportXBRL(context.Background())
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
