package wailsbridge

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/service"
	"gorm.io/gorm"
)

// BuchfinkBridge bridges between Wails v3 frontend IPC and the decoupled Go domain services.
type BuchfinkBridge struct {
	mu          sync.RWMutex
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
	dataDir := filepath.Join(homeDir, ".buchfink", "data")
	currentYear := time.Now().Year()

	b := &BuchfinkBridge{
		dataDir:     dataDir,
		currentYear: currentYear,
	}

	if err := b.initYearDatabase(currentYear); err != nil {
		return nil, fmt.Errorf("failed to init database for year %d: %w", currentYear, err)
	}

	return b, nil
}

func (b *BuchfinkBridge) initYearDatabase(year int) error {
	db, err := repository.InitDB(b.dataDir, year)
	if err != nil {
		return err
	}

	b.db = db
	b.currentYear = year

	// Wire Repositories
	b.accountRepo = repository.NewAccountRepository(db)
	b.bookingRepo = repository.NewBookingRepository(db)
	b.bankRepo = repository.NewBankRepository(db)
	b.contactRepo = repository.NewContactRepository(db)
	b.invoiceRepo = repository.NewInvoiceRepository(db)
	b.auditRepo = repository.NewAuditRepository(db)
	b.settingsRepo = repository.NewSettingsRepository(db)

	// Wire Services
	b.accountingSvc = service.NewAccountingService(b.accountRepo, b.bookingRepo, b.auditRepo, year)
	b.bankSvc = service.NewBankService(b.bankRepo, b.accountingSvc, b.auditRepo)
	b.invoiceSvc = service.NewInvoiceService(b.invoiceRepo, b.contactRepo, b.settingsRepo, b.auditRepo)
	b.contactSvc = service.NewContactService(b.contactRepo, b.auditRepo)
	b.ebilanzSvc = service.NewEBilanzService(b.accountingSvc, b.settingsRepo, b.auditRepo)
	b.auditSvc = service.NewAuditService(b.auditRepo)
	b.settingsSvc = service.NewSettingsService(b.settingsRepo, b.auditRepo)
	b.currencySvc = service.NewCurrencyService()

	return nil
}

// -------------------------------------------------------------
// APP & FISCAL YEAR MANAGEMENT
// -------------------------------------------------------------

func (b *BuchfinkBridge) GetFiscalYear() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.currentYear
}

func (b *BuchfinkBridge) SetFiscalYear(year int) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if year == b.currentYear {
		return nil
	}
	return b.initYearDatabase(year)
}

func (b *BuchfinkBridge) GetCompanySettings() (*domain.CompanySettings, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.settingsSvc.GetCompanySettings(context.Background())
}

func (b *BuchfinkBridge) UpdateCompanySettings(settings domain.CompanySettings) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.settingsSvc.UpdateCompanySettings(context.Background(), &settings)
}

// -------------------------------------------------------------
// ACCOUNTING & JOURNAL
// -------------------------------------------------------------

func (b *BuchfinkBridge) GetAccounts() ([]domain.Account, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.accountingSvc.GetAccounts(context.Background())
}

func (b *BuchfinkBridge) GetBookings() ([]domain.BookingEntry, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.accountingSvc.GetBookings(context.Background())
}

func (b *BuchfinkBridge) CreateBooking(entry domain.BookingEntry) (*domain.BookingEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.accountingSvc.CreateBooking(context.Background(), &entry)
}

func (b *BuchfinkBridge) StornoBooking(bookingID uint, reason string) (*domain.BookingEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.accountingSvc.StornoBooking(context.Background(), bookingID, reason)
}

func (b *BuchfinkBridge) VerifyIntegrity() (domain.IntegrityCheckResult, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.accountingSvc.VerifyIntegrity(context.Background())
}

func (b *BuchfinkBridge) GetFinancialSummary() (*domain.FinancialSummary, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.accountingSvc.GetFinancialSummary(context.Background())
}

// -------------------------------------------------------------
// BANK IMPORT & RECONCILIATION
// -------------------------------------------------------------

func (b *BuchfinkBridge) GetBankTransactions() ([]domain.BankTransaction, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.bankSvc.GetTransactions(context.Background())
}

func (b *BuchfinkBridge) ImportSampleBankStatement() (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	sampleXML := `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
  <BkToCstmrStmt>
    <Stmt>
      <Acct><Id><IBAN>DE89370400440532013000</IBAN></Id></Acct>
      <Ntry>
        <Amt Ccy="EUR">2856.00</Amt>
        <CdtDbtInd>CRDT</CdtDbtInd>
        <BookgDt><Dt>2024-04-15</Dt></BookgDt>
        <ValDt><Dt>2024-04-15</Dt></ValDt>
        <NtryDtls><TxDtls>
          <Refs><EndToEndId>E2E-20240415-001</EndToEndId></Refs>
          <RltdPties>
            <Dbtr><Nm>Acme Corp GmbH</Nm></Dbtr>
            <DbtrAcct><Id><IBAN>DE12500105170648489890</IBAN></Id></DbtrAcct>
          </RltdPties>
          <RmtInf><Ustrd>Rechnung RE-2024-042 Webentwicklung</Ustrd></RmtInf>
        </TxDtls></NtryDtls>
      </Ntry>
      <Ntry>
        <Amt Ccy="EUR">89.25</Amt>
        <CdtDbtInd>DBIT</CdtDbtInd>
        <BookgDt><Dt>2024-04-12</Dt></BookgDt>
        <ValDt><Dt>2024-04-12</Dt></ValDt>
        <NtryDtls><TxDtls>
          <Refs><EndToEndId>E2E-20240412-002</EndToEndId></Refs>
          <RltdPties>
            <Cdtr><Nm>Hetzner Online GmbH</Nm></Cdtr>
            <CdtrAcct><Id><IBAN>DE45700202700015762901</IBAN></Id></CdtrAcct>
          </RltdPties>
          <RmtInf><Ustrd>Server Hosting Invoice 2024-4412</Ustrd></RmtInf>
        </TxDtls></NtryDtls>
      </Ntry>
      <Ntry>
        <Amt Ccy="EUR">650.00</Amt>
        <CdtDbtInd>DBIT</CdtDbtInd>
        <BookgDt><Dt>2024-04-10</Dt></BookgDt>
        <ValDt><Dt>2024-04-10</Dt></ValDt>
        <NtryDtls><TxDtls>
          <Refs><EndToEndId>E2E-20240410-003</EndToEndId></Refs>
          <RltdPties>
            <Cdtr><Nm>Immobilienverwaltung Schmidt</Nm></Cdtr>
            <CdtrAcct><Id><IBAN>DE33200411550123456789</IBAN></Id></CdtrAcct>
          </RltdPties>
          <RmtInf><Ustrd>Büromiete April 2024</Ustrd></RmtInf>
        </TxDtls></NtryDtls>
      </Ntry>
    </Stmt>
  </BkToCstmrStmt>
</Document>`
	return b.bankSvc.ImportCAMT053(context.Background(), strings.NewReader(sampleXML))
}

func (b *BuchfinkBridge) ImportCAMT053XML(xmlContent string) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.bankSvc.ImportCAMT053(context.Background(), bytes.NewBufferString(xmlContent))
}

func (b *BuchfinkBridge) MatchAndBookTransaction(
	bankTxID uint,
	debitAcc, creditAcc, receiptNr, desc string,
) (*domain.BookingEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.bankSvc.MatchAndBook(context.Background(), bankTxID, debitAcc, creditAcc, receiptNr, desc)
}

// -------------------------------------------------------------
// CONTACTS & INVOICES
// -------------------------------------------------------------

func (b *BuchfinkBridge) GetContacts() ([]domain.Contact, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.contactSvc.GetContacts(context.Background())
}

func (b *BuchfinkBridge) SaveContact(c domain.Contact) (*domain.Contact, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.contactSvc.SaveContact(context.Background(), &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (b *BuchfinkBridge) GetInvoices() ([]domain.Invoice, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.invoiceSvc.GetInvoices(context.Background())
}

func (b *BuchfinkBridge) CreateInvoice(inv domain.Invoice) (*domain.Invoice, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.invoiceSvc.CreateInvoice(context.Background(), &inv); err != nil {
		return nil, err
	}
	return &inv, nil
}

func (b *BuchfinkBridge) GenerateInvoiceZUGFeRD(invoiceID uint) (string, string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.invoiceSvc.GenerateZUGFeRDAndTypst(context.Background(), invoiceID)
}

// -------------------------------------------------------------
// E-BILANZ & AUDIT TRAIL
// -------------------------------------------------------------

func (b *BuchfinkBridge) ExportEBilanzXBRL() (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.ebilanzSvc.ExportXBRL(context.Background())
}

func (b *BuchfinkBridge) GetAuditLogs() ([]domain.AuditLogEntry, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.auditSvc.GetLogs(context.Background(), 200)
}
