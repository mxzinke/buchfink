package core

import (
	"context"

	"github.com/buchfink/buchfink/internal/declaration"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/ebilanz"
	"github.com/buchfink/buchfink/internal/service"
)

// AgentServices begrenzt MCP und seine Regeln auf die tatsächlich benötigten Dienstmethoden.
// Transaktion, Erfassungsweg und Freigaben prüfen weiterhin Kern und Dienste.
type AgentServices struct {
	Accounting interface {
		Entry(context.Context, uint) (*domain.JournalEntry, error)
		GetAccountLedgerRange(context.Context, string, string, string) (*domain.AccountLedger, error)
		GetAccounts(context.Context, int) ([]domain.Account, error)
		GetAllEntries(context.Context) ([]domain.JournalEntry, error)
		GetEntries(context.Context, int) ([]domain.JournalEntry, error)
		GetSuSaOverview(context.Context, int) (*domain.SuSaOverview, error)
	}
	Accruals interface {
		Book(context.Context, service.AccrualRequest) (*domain.Accrual, error)
		Preview(context.Context, service.AccrualRequest) (*service.AccrualPreview, error)
	}
	Appropriation interface {
		BookAppropriation(context.Context, int, service.AppropriationRequest) (*domain.Appropriation, error)
		CheckAdopted(context.Context, int, string) error
		PreviewAppropriation(context.Context, int, service.AppropriationRequest) (*service.AppropriationPreview, error)
	}
	Assets interface {
		AcquisitionCandidates(context.Context, int) ([]service.AcquisitionCandidate, error)
		BookAssetIncome(context.Context, service.AssetIncomeRequest) (*domain.JournalEntry, error)
		BookDepreciation(context.Context, service.BookDepreciationRequest) (*service.DepreciationResult, error)
		List(context.Context, int, domain.AssetClass) ([]domain.FixedAsset, error)
		PendingDepreciation(context.Context, int) ([]service.DepreciationDue, error)
		Save(context.Context, *domain.FixedAsset) (*domain.FixedAsset, error)
	}
	Bank interface {
		Accounts(context.Context) ([]domain.BankAccount, error)
		FileStatement(context.Context, []byte, string) (*uint, error)
		GetTransactions(context.Context, int) ([]domain.BankTransaction, error)
		IgnoreBecause(context.Context, uint, string) error
		ImportStatementChecked(context.Context, []byte, string, *uint) (service.StatementImport, error)
		SuggestFor(context.Context, *domain.BankTransaction, []domain.OpenItem) (*service.BankSuggestions, error)
		Transaction(context.Context, uint) (*domain.BankTransaction, error)
	}
	Checks interface {
		Preview(context.Context, service.CheckRequest) (*domain.CheckRun, error)
	}
	Closing interface {
		BookLegalReserve(context.Context, int) (*domain.JournalEntry, error)
		FiscalYears(context.Context) ([]domain.FiscalYear, error)
		LegalReserve(context.Context, int) (*service.LegalReserveState, error)
		PeriodOf(context.Context, int) (*domain.FiscalYear, error)
		YearOf(context.Context, int) (*domain.FiscalYear, error)
	}
	ClosingBookings interface {
		BookReceivableAllowance(context.Context, service.ReceivableAllowanceRequest) (*domain.JournalEntry, error)
		BookTaxProvision(context.Context, service.TaxProvisionRequest) ([]domain.Provision, error)
		BookVatSettlement(context.Context, int) (*domain.JournalEntry, error)
		BookWorkInProgress(context.Context, service.WorkInProgressRequest) (*domain.JournalEntry, error)
		PreviewReceivableAllowance(context.Context, service.ReceivableAllowanceRequest) (*service.ReceivableAllowancePreview, error)
		PreviewTaxProvision(context.Context, int, service.TaxProvisionOptions) (*service.TaxProvisionPreview, error)
		PreviewVatSettlement(context.Context, int) (*service.VatSettlement, error)
		PreviewWorkInProgress(context.Context, service.WorkInProgressRequest) (*service.WorkInProgressPreview, error)
	}
	ClosingSteps interface {
		Steps(context.Context, int) (*service.ClosingSteps, error)
	}
	Contacts interface {
		Contact(context.Context, uint) (*domain.Contact, error)
		GetContacts(context.Context, int) ([]domain.Contact, error)
		ProposeDefaultPostingGroup(context.Context, uint, string, string) (*domain.Contact, error)
		SaveContact(context.Context, *domain.Contact) error
	}
	Currency interface {
		BookCurrencyValuation(context.Context, int) (*service.ForeignCurrencyValuation, error)
		PreviewCurrencyValuation(context.Context, int) (*service.ForeignCurrencyValuation, error)
	}
	Declarations interface {
		Build(context.Context, declaration.Kind, int) (*declaration.Declaration, error)
		SetInput(context.Context, service.InputRequest) (*declaration.Input, error)
	}
	EBilanz interface {
		ExportSuSaCSV(context.Context, int, service.EBilanzOptions) ([]byte, error)
		ExportXBRLWith(context.Context, int, service.EBilanzOptions) (string, error)
		Generate(context.Context, int, service.EBilanzOptions) (*ebilanz.Result, error)
	}
	EInvoice interface {
		ExtractStructuredPart(context.Context, uint) (*domain.Receipt, error)
		Propose(context.Context, uint) (*service.EInvoiceProposal, error)
	}
	Foundation interface {
		GetState(context.Context) (*service.FoundationState, error)
	}
	Idempotency interface {
		Run(context.Context, service.IdempotentCall, func(context.Context) (string, any, error)) (*service.IdempotentOutcome, error)
	}
	Invoices interface {
		CreateInvoiceGroup(context.Context, service.AdvanceGroupRequest) (*domain.InvoiceGroup, error)
		Issue(context.Context, *domain.Invoice) error
		IssueAdvanceInvoice(context.Context, service.AdvanceInvoiceRequest) (*domain.Invoice, error)
		IssueFinalInvoice(context.Context, service.FinalInvoiceRequest) (*domain.Invoice, error)
		Preview(context.Context, *domain.Invoice) (*service.PostingPreview, error)
	}
	Journal interface {
		CommittedThrough(context.Context) (string, error)
		CorrectEntry(context.Context, uint, string, *domain.JournalEntry) (*service.CorrectionResult, error)
		CorrectionOf(context.Context, uint) (*domain.JournalEntry, error)
		ReverseOn(context.Context, uint, string, string) (*domain.JournalEntry, error)
	}
	Obligations interface {
		Get(context.Context, string, string) (*domain.Obligation, error)
		List(context.Context, service.ObligationQuery) ([]domain.Obligation, error)
		MarkPrepared(context.Context, string, string) (*domain.Obligation, error)
		RecordEvent(context.Context, domain.ObligationEventKind, string, string) (*domain.ObligationEvent, error)
	}
	Overdue interface {
		Claims(context.Context, string) ([]service.OverdueClaim, error)
		RecordReminder(context.Context, service.ReminderRequest) (*domain.DunningNote, error)
	}
	Payments interface {
		AgeOpenItems([]domain.OpenItem, string) *domain.OpenItemsAging
		BookBankSplit(context.Context, service.BankSplitRequest) (*domain.JournalEntry, error)
		OpenItems(context.Context) ([]domain.OpenItem, error)
		OpenItemsAt(context.Context, string) ([]domain.OpenItem, error)
		Settle(context.Context, service.PaymentRequest) (*domain.JournalEntry, error)
		SettleCreditCard(context.Context, service.CreditCardSettlementRequest) (*domain.JournalEntry, error)
	}
	Payroll interface {
		BookPayroll(context.Context, service.PayrollRequest) (*domain.JournalEntry, error)
	}
	Posting interface {
		PostIncomingReceipt(context.Context, service.ReceiptRequest) (*domain.JournalEntry, error)
		PostIncomingReceiptWithHeader(context.Context, service.ReceiptRequest, string) (*domain.JournalEntry, error)
		PostManualEntry(context.Context, service.ManualEntryRequest) (*domain.JournalEntry, error)
		PreviewIncomingReceipt(context.Context, service.ReceiptRequest) (*service.PostingPreview, error)
	}
	ProcDoc interface {
		OrganisationTexts(context.Context) domain.OrganisationTexts
		SaveOrganisationTexts(context.Context, domain.OrganisationTexts) error
	}
	Provisions interface {
		BookFormation(context.Context, service.ProvisionRequest) (*domain.Provision, error)
		Preview(context.Context, service.ProvisionRequest) (*service.ProvisionPreview, error)
	}
	Receipts interface {
		AddFile(context.Context, uint, service.NewFile) (*domain.Receipt, error)
		Content(context.Context, uint, uint) (*service.FileContent, error)
		Discard(context.Context, uint, string) error
		File(context.Context, service.FileReceiptRequest) (*domain.Receipt, error)
		FindByEInvoiceNumber(context.Context, string) ([]domain.Receipt, error)
		FindByOriginalHash(context.Context, string) (*domain.Receipt, error)
		Findings(context.Context, uint) (*domain.ReceiptFindings, error)
		Get(context.Context, uint) (*domain.Receipt, error)
		GetForPosting(context.Context, uint) (*domain.Receipt, error)
		InputTaxPeriodDifference(context.Context, *domain.Receipt) *service.InputTaxPeriodDifference
		List(context.Context, int, domain.ReceiptStatus) ([]domain.Receipt, error)
		ReceiptIDsByFileHash(context.Context, []string) (map[string]uint, error)
		SaveHeader(context.Context, uint, domain.ReceiptHeader) (*domain.Receipt, error)
	}
	Settings interface {
		AgentSettings(context.Context) (*domain.AgentSettings, error)
		GetCompanySettings(context.Context) (*domain.CompanySettings, error)
		InboxFolder(context.Context) (string, error)
	}
	Shareholders interface {
		BookInterest(context.Context, service.InterestRequest) (*domain.JournalEntry, error)
		CorrectLoan(context.Context, uint, service.LoanRequest, string) (*service.LoanView, error)
		LoanRequestOf(context.Context, uint) (service.LoanRequest, error)
		Loans(context.Context, string) ([]service.LoanView, error)
		PreviewInterest(context.Context, service.InterestRequest) (*service.InterestPreview, error)
		PreviewLoan(context.Context, uint, service.LoanRequest) (*service.LoanView, error)
		RegisterLoan(context.Context, service.LoanRequest) (*service.LoanView, error)
	}
	Statement interface {
		Statement(context.Context, int, domain.StatementDepth) (*domain.Statement, error)
	}
	Tasks interface {
		Tasks(context.Context, service.TaskOptions) (*domain.TaskList, error)
	}
	Vat interface {
		Summary(context.Context, int, string, string) (*domain.VatSummary, error)
	}
	VatReturns interface {
		Draft(context.Context, string) (*domain.VatReturn, error)
		PeriodMismatch(context.Context, string) domain.VatPeriodType
	}
	ZM interface {
		Draft(context.Context, string) (*domain.ZMReturn, error)
	}
}

// AgentServices liefert die begrenzte Sicht innerhalb eines Kernvorgangs.
func (s *Services) AgentServices() AgentServices {
	return AgentServices{
		Accounting:      s.Accounting,
		Accruals:        s.Accruals,
		Appropriation:   s.Appropriation,
		Assets:          s.Assets,
		Bank:            s.Bank,
		Checks:          s.Checks,
		Closing:         s.Closing,
		ClosingBookings: s.ClosingBookings,
		ClosingSteps:    s.ClosingSteps,
		Contacts:        s.Contacts,
		Currency:        s.Currency,
		Declarations:    s.Declarations,
		EBilanz:         s.EBilanz,
		EInvoice:        s.EInvoice,
		Foundation:      s.Foundation,
		Idempotency:     s.Idempotency,
		Invoices:        s.Invoices,
		Journal:         s.Journal,
		Obligations:     s.Obligations,
		Overdue:         s.Overdue,
		Payments:        s.Payments,
		Payroll:         s.Payroll,
		Posting:         s.Posting,
		ProcDoc:         s.ProcDoc,
		Provisions:      s.Provisions,
		Receipts:        s.Receipts,
		Settings:        s.Settings,
		Shareholders:    s.Shareholders,
		Statement:       s.Statement,
		Tasks:           s.Tasks,
		Vat:             s.Vat,
		VatReturns:      s.VatReturns,
		ZM:              s.ZM,
	}
}
