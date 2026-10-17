package core

import (
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

	// fiscalYear ist das Geschäftsjahr, nach dem die Dienste filtern, wenn ein
	// Aufruf keines nennt. Geändert wird es nur unter Tenant.stateMu.
	fiscalYear int
}

// Repos sind die Repositories, die Aufrufer ohne Dienst lesen.
type Repos struct {
	Journal        domain.JournalRepository
	Accounts       domain.AccountRepository
	Contacts       domain.ContactRepository
	Invoices       domain.InvoiceRepository
	Receipts       domain.ReceiptRepository
	Audit          domain.AuditRepository
	Settings       domain.SettingsRepository
	Festschreibung domain.FestschreibungRepository
	FiscalYears    domain.FiscalYearRepository
	Migrations     domain.MigrationRepository
	BackupRuns     domain.BackupRunRepository
	ExchangeRates  domain.ExchangeRateRepository
}

// FiscalYear ist das Geschäftsjahr, das die Dienste ohne Angabe annehmen.
func (s *Services) FiscalYear() int { return s.fiscalYear }

// setFiscalYear stellt jeden Dienst mit Jahreszustand auf ein Jahr um.
func (s *Services) setFiscalYear(year int) {
	s.fiscalYear = year
	for _, svc := range []interface{ SetFiscalYear(int) }{
		s.Journal, s.Receipts, s.EInvoice, s.Payments, s.Vat, s.Accounting,
		s.Contacts, s.EBilanz, s.Assets, s.ProcDoc, s.Closing, s.Statement,
		s.VatReturns, s.ZM, s.Checks, s.Deadlines, s.Export, s.ClosingSteps,
		s.Accruals, s.Provisions, s.ClosingBookings, s.Appropriation,
		s.TaxRegister, s.InputTax, s.Gifts, s.Tasks, s.MonthClose,
	} {
		svc.SetFiscalYear(year)
	}
}
