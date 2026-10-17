package core

import (
	"context"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// TenantInfo beschreibt den offenen Mandanten.
type TenantInfo struct {
	ID      string
	Name    string
	DataDir string
}

// Services ist der eine Satz Dienste eines offenen Mandanten. Bridge und
// MCP-Server bekommen ihn nur über Tenant.Read und Tenant.Write.
type Services struct {
	Tenant TenantInfo
	Repos  Repos

	Journal         *service.JournalService
	Posting         *service.PostingService
	Receipts        *service.ReceiptService
	EInvoice        *service.EInvoiceService
	Payments        *service.PaymentService
	Vat             *service.VatService
	Accounting      *service.AccountingService
	Accounts        *service.AccountService
	Bank            *service.BankService
	Invoices        *service.InvoiceService
	Contacts        *service.ContactService
	EBilanz         *service.EBilanzService
	Assets          *service.AssetService
	Audit           *service.AuditService
	ProcDoc         *service.ProcDocService
	Settings        *service.SettingsService
	Currency        *service.CurrencyService
	Foundation      *service.FoundationService
	Closing         *service.ClosingService
	ClosingSettings *service.ClosingSettingsService
	Statement       *service.StatementService
	VatReturns      *service.VatReturnService
	ZM              *service.ZMService
	Checks          *service.CheckService
	Deadlines       *service.DeadlineService
	Export          *service.ExportService
	Backup          *service.BackupService
	ClosingSteps    *service.ClosingStepsService
	Accruals        *service.AccrualService
	Provisions      *service.ProvisionService
	ClosingBookings *service.ClosingBookingService
	Appropriation   *service.AppropriationService
	TaxRegister     *service.TaxRegisterService
	InputTax        *service.InputTaxService
	VatIDs          *service.VatIDService
	Gifts           *service.GiftService
	Tasks           *service.TaskService
	MonthClose      *service.MonthCloseService
	AuditTrail      *service.AuditTrailService
	Idempotency     *service.IdempotencyService
	Declarations    *service.DeclarationService
}

// Repos sind lesende Zugänge für Aufrufer, die ohne Dienst lesen. Geschrieben
// wird nur über die Dienste und die Vorgänge des Kerns, die prüfen und
// protokollieren. Einzig das Änderungsprotokoll nimmt Einträge an: auch ein
// Lesevorgang protokolliert die Herausgabe von Daten (QUE-02).
type Repos struct {
	Journal        JournalReader
	Contacts       ContactReader
	Invoices       InvoiceReader
	Audit          AuditLog
	Settings       SettingsReader
	Festschreibung FestschreibungReader
	FiscalYears    FiscalYearReader
	Migrations     MigrationReader
	BackupRuns     BackupRunReader
	ExchangeRates  ExchangeRateReader
	Examiner       ExaminerReader
}

// JournalReader liest Buchungen.
type JournalReader interface {
	FindAll(ctx context.Context, fiscalYear int) ([]domain.JournalEntry, error)
	FindByID(ctx context.Context, id uint) (*domain.JournalEntry, error)
	GetLastEntry(ctx context.Context, fiscalYear int) (*domain.JournalEntry, error)
	Count(ctx context.Context, fiscalYear int) (int64, error)
}

// ContactReader liest Geschäftspartner.
type ContactReader interface {
	FindAll(ctx context.Context) ([]domain.Contact, error)
	FindByID(ctx context.Context, id uint) (*domain.Contact, error)
}

// InvoiceReader liest Rechnungen.
type InvoiceReader interface {
	FindAll(ctx context.Context, fiscalYear int) ([]domain.Invoice, error)
	FindByID(ctx context.Context, id uint) (*domain.Invoice, error)
}

// AuditLog liest das Änderungsprotokoll und nimmt Zugriffsvermerke an.
type AuditLog interface {
	Log(ctx context.Context, action domain.AuditAction, entityType, entityID, details string) error
	FindFiltered(ctx context.Context, limit int, filter domain.AuditFilter) ([]domain.AuditLogEntry, error)
}

// SettingsReader liest Einstellungen und Unternehmensdaten.
type SettingsReader interface {
	Get(ctx context.Context, key string) (string, error)
	GetCompanySettings(ctx context.Context) (*domain.CompanySettings, error)
}

// FestschreibungReader liest Festschreibungen.
type FestschreibungReader interface {
	FindByFiscalYear(ctx context.Context, fiscalYear int) ([]domain.Festschreibung, error)
	FindByID(ctx context.Context, id uint) (*domain.Festschreibung, error)
	FindPendingTimestamp(ctx context.Context) ([]domain.Festschreibung, error)
}

// FiscalYearReader liest Geschäftsjahre und ihren Abschlussstand.
type FiscalYearReader interface {
	FindAll(ctx context.Context) ([]domain.FiscalYear, error)
	FindByYear(ctx context.Context, year int) (*domain.FiscalYear, error)
}

// MigrationReader liest Schema- und Übernahmeprotokoll.
type MigrationReader interface {
	FindSchemaMigrations(ctx context.Context) ([]domain.SchemaMigration, error)
	FindMigrationRecords(ctx context.Context) ([]domain.MigrationRecord, error)
}

// BackupRunReader liest die Sicherungsläufe.
type BackupRunReader interface {
	FindRecent(ctx context.Context, limit int) ([]domain.BackupRun, error)
}

// ExchangeRateReader liest die Kurshistorie.
type ExchangeRateReader interface {
	FindRate(ctx context.Context, currency, date string) (*domain.ExchangeRate, error)
}

// ExaminerReader liest den Prüfermodus.
type ExaminerReader interface {
	Latest(ctx context.Context) (*domain.ExaminerPeriod, error)
}

// stores sind die schreibenden Repositories, die nur der Kern selbst anfasst:
// Festschreibung, Prüfermodus und Übernahmeprotokoll.
type stores struct {
	journal        domain.JournalRepository
	audit          domain.AuditRepository
	settings       domain.SettingsRepository
	festschreibung domain.FestschreibungRepository
	fiscalYears    domain.FiscalYearRepository
	migrations     domain.MigrationRepository
	examiner       domain.ExaminerRepository
}

func (st stores) readers(contacts domain.ContactRepository, invoices domain.InvoiceRepository,
	backupRuns domain.BackupRunRepository, rates domain.ExchangeRateRepository) Repos {
	return Repos{
		Journal: st.journal, Contacts: contacts, Invoices: invoices, Audit: st.audit,
		Settings: st.settings, Festschreibung: st.festschreibung, FiscalYears: st.fiscalYears,
		Migrations: st.migrations, BackupRuns: backupRuns, ExchangeRates: rates,
		Examiner: st.examiner,
	}
}
