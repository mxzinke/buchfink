package wailsbridge

import (
	"bytes"
	"context"
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

	// Repositories
	accountRepo  domain.AccountRepository
	bookingRepo  domain.BookingRepository
	bankRepo     domain.BankRepository
	contactRepo  domain.ContactRepository
	invoiceRepo  domain.InvoiceRepository
	auditRepo    domain.AuditRepository
	settingsRepo domain.SettingsRepository

	// Services
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
				ID:          "default",
				Name:        "Hauptmandant",
				DataDir:     cfg.DataDir,
				CertPath:    cfg.CertPath,
				HasPassword: cfg.HasPassword,
				CreatedAt:   time.Now().Format(time.RFC3339),
			}
		}

		if activeTenant != nil {
			if err := b.initTenant(activeTenant); err != nil {
				return nil, fmt.Errorf("failed to init tenant database: %w", err)
			}
		}
	}

	return b, nil
}

func (b *BuchfinkBridge) initTenant(t *domain.TenantConfig) error {
	if t == nil {
		return fmt.Errorf("no tenant specified")
	}

	db, err := repository.InitTenantDB(t.DataDir)
	if err != nil {
		return fmt.Errorf("failed to initialize tenant database at %s: %w", t.DataDir, err)
	}

	b.db = db
	b.dataDir = t.DataDir

	// Wire Repositories
	b.accountRepo = repository.NewAccountRepository(db)
	b.bookingRepo = repository.NewBookingRepository(db)
	b.bankRepo = repository.NewBankRepository(db)
	b.contactRepo = repository.NewContactRepository(db)
	b.invoiceRepo = repository.NewInvoiceRepository(db)
	b.auditRepo = repository.NewAuditRepository(db)
	b.settingsRepo = repository.NewSettingsRepository(db)

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

	// Wire Services
	b.accountingSvc = service.NewAccountingService(b.accountRepo, b.bookingRepo, b.settingsRepo, b.auditRepo, fiscalYear)
	b.bankSvc = service.NewBankService(b.bankRepo, b.accountingSvc, b.auditRepo)
	b.invoiceSvc = service.NewInvoiceService(b.invoiceRepo, b.contactRepo, b.settingsRepo, b.auditRepo)
	b.contactSvc = service.NewContactService(b.contactRepo, b.auditRepo)
	b.ebilanzSvc = service.NewEBilanzService(b.accountingSvc, b.settingsRepo, b.auditRepo)
	b.auditSvc = service.NewAuditService(b.auditRepo)
	b.settingsSvc = service.NewSettingsService(b.settingsRepo, b.auditRepo)
	b.currencySvc = service.NewCurrencyService()

	b.appConfig.ActiveTenantID = t.ID
	b.appConfig.DataDir = t.DataDir
	b.appConfig.CertPath = t.CertPath
	b.appConfig.HasPassword = t.HasPassword
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

func (b *BuchfinkBridge) CreateTenant(
	name string,
	dataDir string,
	certDir string,
	password string,
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
	if certDir == "" {
		certDir = filepath.Join(homeDir, ".buchfink", "tenants", tenantID, "keys")
	}

	// Persist absolute paths so tenant locations are stable regardless of the
	// process working directory (relative paths would break e.g. in dev mode).
	if abs, err := filepath.Abs(dataDir); err == nil {
		dataDir = abs
	}

	certPath, _, err := security.GenerateCertificate(certDir, name, password)
	if err != nil {
		return nil, fmt.Errorf("certificate generation failed: %w", err)
	}

	fiscalYear := settings.FiscalYear
	if fiscalYear == 0 {
		fiscalYear = time.Now().Year()
	}

	t := domain.TenantConfig{
		ID:          tenantID,
		Name:        name,
		DataDir:     dataDir,
		CertPath:    certPath,
		HasPassword: password != "",
		CreatedAt:   time.Now().Format(time.RFC3339),
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

func (b *BuchfinkBridge) ImportTenant(dbFilePath string, certPath string, password string) (*domain.TenantConfig, error) {
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

	if certPath == "" {
		homeDir, _ := os.UserHomeDir()
		certPath = filepath.Join(homeDir, ".buchfink", "certs", "buchfink-cert.pem")
	}

	t := domain.TenantConfig{
		ID:          tenantID,
		Name:        name,
		DataDir:     dataDir,
		CertPath:    certPath,
		HasPassword: password != "",
		CreatedAt:   time.Now().Format(time.RFC3339),
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

// SetupApplication handles initial setup: generates certificate in chosen folder, establishes data directory, and configures company.
func (b *BuchfinkBridge) SetupApplication(
	dataDir string,
	certDir string,
	password string,
	settings domain.CompanySettings,
) error {
	name := settings.CompanyName
	if name == "" {
		name = "Hauptmandant"
	}
	_, err := b.CreateTenant(name, dataDir, certDir, password, settings)
	return err
}

// LoadExistingDatabase loads an existing SQLite database file from disk.
func (b *BuchfinkBridge) LoadExistingDatabase(dbFilePath string) error {
	_, err := b.ImportTenant(dbFilePath, "", "")
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

	b.currentYear = year
	if b.accountingSvc != nil {
		b.accountingSvc.SetFiscalYear(year)
	}
	b.appConfig.LastFiscalYear = year
	_ = b.appCfgRepo.Save(&b.appConfig)
	return nil
}

func (b *BuchfinkBridge) GetFiscalYear() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.currentYear
}

func (b *BuchfinkBridge) SetFiscalYear(year int) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.currentYear = year
	if b.accountingSvc != nil {
		b.accountingSvc.SetFiscalYear(year)
	}
	b.appConfig.LastFiscalYear = year
	_ = b.appCfgRepo.Save(&b.appConfig)
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

func (b *BuchfinkBridge) GetAccountBookings(accountNumber string) ([]domain.BookingEntry, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accountingSvc == nil {
		return nil, nil
	}
	return b.accountingSvc.GetAccountBookings(context.Background(), accountNumber)
}

func (b *BuchfinkBridge) GetAccountLedger(accountNumber string) (*domain.AccountLedger, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accountingSvc == nil {
		return nil, fmt.Errorf("accounting service not initialized")
	}
	return b.accountingSvc.GetAccountLedger(context.Background(), accountNumber)
}

func (b *BuchfinkBridge) GetSuSaOverview() (*domain.SuSaOverview, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accountingSvc == nil {
		return nil, fmt.Errorf("accounting service not initialized")
	}
	return b.accountingSvc.GetSuSaOverview(context.Background())
}

func (b *BuchfinkBridge) GetSKR04Catalog() (*accounting.SKR04Catalog, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accountingSvc == nil {
		return accounting.GetSKR04Catalog()
	}
	return b.accountingSvc.GetSKR04Catalog(context.Background())
}

func (b *BuchfinkBridge) GetBookings() ([]domain.BookingEntry, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accountingSvc == nil {
		return nil, nil
	}
	return b.accountingSvc.GetBookings(context.Background())
}

func (b *BuchfinkBridge) GetAllBookings() ([]domain.BookingEntry, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accountingSvc == nil {
		return nil, nil
	}
	return b.accountingSvc.GetAllBookings(context.Background())
}

func (b *BuchfinkBridge) CreateBooking(entry domain.BookingEntry) (*domain.BookingEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.accountingSvc == nil {
		return nil, fmt.Errorf("accounting service not initialized")
	}
	return b.accountingSvc.CreateBooking(context.Background(), &entry)
}

func (b *BuchfinkBridge) StornoBooking(bookingID uint, reason string) (*domain.BookingEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.accountingSvc == nil {
		return nil, fmt.Errorf("accounting service not initialized")
	}
	return b.accountingSvc.StornoBooking(context.Background(), bookingID, reason)
}

func (b *BuchfinkBridge) VerifyIntegrity() (domain.IntegrityCheckResult, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.accountingSvc == nil {
		return domain.IntegrityCheckResult{IsValid: true, Message: "Bereit"}, nil
	}
	return b.accountingSvc.VerifyIntegrity(context.Background())
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
	return b.bankSvc.GetTransactions(context.Background())
}

func (b *BuchfinkBridge) ImportCAMT053XML(xmlContent string) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.bankSvc == nil {
		return 0, fmt.Errorf("bank service not initialized")
	}
	return b.bankSvc.ImportCAMT053(context.Background(), bytes.NewBufferString(xmlContent))
}

func (b *BuchfinkBridge) MatchAndBookTransaction(
	bankTxID uint,
	debitAcc, creditAcc, receiptNr, desc string,
) (*domain.BookingEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.bankSvc == nil {
		return nil, fmt.Errorf("bank service not initialized")
	}
	return b.bankSvc.MatchAndBook(context.Background(), bankTxID, debitAcc, creditAcc, receiptNr, desc)
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
		return nil, fmt.Errorf("contact service not initialized")
	}
	if err := b.contactSvc.SaveContact(context.Background(), &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (b *BuchfinkBridge) GetInvoices() ([]domain.Invoice, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.invoiceSvc == nil {
		return nil, nil
	}
	return b.invoiceSvc.GetInvoices(context.Background())
}

func (b *BuchfinkBridge) CreateInvoice(inv domain.Invoice) (*domain.Invoice, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.invoiceSvc == nil {
		return nil, fmt.Errorf("invoice service not initialized")
	}
	if err := b.invoiceSvc.CreateInvoice(context.Background(), &inv); err != nil {
		return nil, err
	}
	return &inv, nil
}

func (b *BuchfinkBridge) GenerateInvoiceZUGFeRD(invoiceID uint) (string, string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.invoiceSvc == nil {
		return "", "", fmt.Errorf("invoice service not initialized")
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

