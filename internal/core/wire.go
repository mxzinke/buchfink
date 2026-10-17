package core

import (
	"context"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/invoice"
	"github.com/buchfink/buchfink/internal/receiptstore"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/service"
	"gorm.io/gorm"
)

// wire verdrahtet Repositories und Dienste eines Mandanten. Die Reihenfolge
// zählt: ein Dienst, der einen anderen gesetzt bekommt, muss nach ihm
// entstehen.
func wire(db *gorm.DB, info TenantInfo, renderer *invoice.Renderer, o Options) (*Services, stores) {
	accountRepo := repository.NewAccountRepository(db)
	journalRepo := repository.NewJournalRepository(db)
	numberRepo := repository.NewNumberRangeRepository(db)
	bankRepo := repository.NewBankRepository(db)
	contactRepo := repository.NewContactRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	festschreibungRepo := repository.NewFestschreibungRepository(db)
	allocationRepo := repository.NewPaymentAllocationRepository(db)
	receiptRepo := repository.NewReceiptRepository(db)
	assetRepo := repository.NewAssetRepository(db)
	foundationRepo := repository.NewFoundationRepository(db)
	fiscalYearRepo := repository.NewFiscalYearRepository(db)
	vatReturnRepo := repository.NewVatReturnRepository(db)
	zmReturnRepo := repository.NewZMReturnRepository(db)
	checkRunRepo := repository.NewCheckRunRepository(db)
	obligationRepo := repository.NewObligationRepository(db)
	declarationRepo := repository.NewDeclarationRepository(db)
	backupRunRepo := repository.NewBackupRunRepository(db)
	closingStepRepo := repository.NewClosingStepRepository(db)
	accrualRepo := repository.NewAccrualRepository(db)
	provisionRepo := repository.NewProvisionRepository(db)
	discountRateRepo := repository.NewDiscountRateRepository(db)
	notesRepo := repository.NewNotesTextRepository(db)
	appropriationRepo := repository.NewAppropriationRepository(db)
	numberGapRepo := repository.NewNumberGapRepository(db)
	invoiceGroupRepo := repository.NewInvoiceGroupRepository(db)
	vendorAdvanceRepo := repository.NewVendorAdvanceRepository(db)
	inputTaxRepo := repository.NewInputTaxCorrectionRepository(db)
	vatIDCheckRepo := repository.NewVatIDCheckRepository(db)
	exchangeRateRepo := repository.NewExchangeRateRepository(db)
	migrationRepo := repository.NewMigrationRepository(db)
	procDocRepo := repository.NewProcedureDocumentationRepository(db)
	bankRuleRepo := repository.NewBankRuleRepository(db)
	idempotencyRepo := repository.NewIdempotencyRepository(db)
	txRunner := repository.NewTxRunner(db)
	store := receiptstore.New(info.DataDir)

	st := stores{
		journal: journalRepo, audit: auditRepo, settings: settingsRepo,
		festschreibung: festschreibungRepo, fiscalYears: fiscalYearRepo,
		migrations: migrationRepo, examiner: repository.NewExaminerRepository(db),
	}
	s := &Services{
		Tenant: info,
		Repos:  st.readers(contactRepo, invoiceRepo, backupRunRepo, exchangeRateRepo),
	}

	// Der Journaldienst ist der einzige Schreibweg ins Journal und hält die
	// Transaktion aller Vorgänge, die über ihn buchen.
	s.Journal = service.NewJournalService(journalRepo, accountRepo, contactRepo, auditRepo, settingsRepo, txRunner)
	s.Journal.SetFestschreibungRepo(festschreibungRepo)
	s.Journal.SetFiscalYearRepo(fiscalYearRepo)
	s.Journal.SetReceiptRepo(receiptRepo)
	s.Journal.SetBankRepo(bankRepo)
	s.Idempotency = service.NewIdempotencyService(idempotencyRepo, txRunner)

	s.Posting = service.NewPostingService(s.Journal, contactRepo)
	s.Receipts = service.NewReceiptService(receiptRepo, store, auditRepo)
	s.Posting.SetReceiptService(s.Receipts)
	s.Receipts.SetSettingsSource(settingsRepo)
	s.Receipts.SetNumberRepo(numberRepo)
	s.Receipts.SetContactSource(contactRepo)
	s.EInvoice = service.NewEInvoiceService(s.Receipts, contactRepo, invoice.NewReader())
	s.Accounting = service.NewAccountingService(accountRepo, journalRepo, contactRepo, settingsRepo, s.Journal)
	s.Accounting.SetAuditRepo(auditRepo)
	s.Accounts = service.NewAccountService(accountRepo, auditRepo)
	s.Accounts.SetChartInvalidator(s.Journal)
	s.Bank = service.NewBankService(bankRepo, s.Journal, auditRepo)
	s.Bank.SetReceiptService(s.Receipts)
	s.Bank.SetAccountRegistry(repository.NewBankAccountRepository(db), settingsRepo, txRunner)
	s.Payments = service.NewPaymentService(s.Journal, journalRepo, allocationRepo, contactRepo, bankRepo)
	s.Payments.SetAuditRepo(auditRepo)
	s.Posting.SetVendorAdvances(vendorAdvanceRepo)
	s.Posting.SetSettingsSource(settingsRepo)
	s.Vat = service.NewVatService(journalRepo)
	s.Invoices = service.NewInvoiceService(invoiceRepo, contactRepo, settingsRepo, numberRepo, s.Posting, auditRepo)
	s.Invoices.SetDocumentPipeline(s.Receipts, renderer)
	s.Receipts.SetRenderer(renderer)
	s.Invoices.SetRegistry(service.InvoiceRegistry{
		Gaps:        numberGapRepo,
		Groups:      invoiceGroupRepo,
		FiscalYears: fiscalYearRepo,
		Bank:        bankRepo,
	})
	s.Contacts = service.NewContactService(contactRepo, journalRepo, numberRepo, auditRepo)
	s.Assets = service.NewAssetService(
		assetRepo, journalRepo, s.Journal, numberRepo,
		contactRepo, settingsRepo, auditRepo,
	)
	s.Payments.SetAssetRegister(s.Assets)
	s.Assets.SetFiscalYearRepo(fiscalYearRepo)
	s.Audit = service.NewAuditService(auditRepo)
	s.Settings = service.NewSettingsService(settingsRepo, auditRepo)
	s.Settings.SetBookkeeping(journalRepo, festschreibungRepo)
	s.ProcDoc = service.NewProcDocService(
		settingsRepo, numberRepo, procDocRepo, migrationRepo, auditRepo, store)
	s.ProcDoc.SetRenderer(renderer)

	s.Currency = service.NewCurrencyService(exchangeRateRepo, o.Network.Rates, auditRepo)
	s.Currency.SetJournalRepo(journalRepo)
	s.Currency.SetJournalService(s.Journal)

	s.Foundation = service.NewFoundationService(
		foundationRepo, accountRepo, journalRepo, settingsRepo, s.Journal, auditRepo,
	)
	s.Settings.SetFoundationRegistrar(s.Foundation)

	s.Closing = service.NewClosingService(
		fiscalYearRepo, journalRepo, accountRepo, contactRepo, allocationRepo,
		settingsRepo, festschreibungRepo, auditRepo, s.Journal,
	)
	s.Closing.SetFoundationRepo(foundationRepo)
	s.Foundation.SetFoundingYearAligner(s.Closing)
	s.ClosingSettings = service.NewClosingSettingsService(settingsRepo, auditRepo)

	s.Accruals = service.NewAccrualService(
		accrualRepo, journalRepo, s.Journal, settingsRepo, auditRepo, s.Closing,
	)
	s.Accruals.SetReceiptService(s.Receipts)
	s.Closing.SetAccrualCarrier(s.Accruals)

	s.Provisions = service.NewProvisionService(
		provisionRepo, discountRateRepo, journalRepo, s.Journal,
		settingsRepo, auditRepo, s.Closing,
	)
	s.Provisions.SetReceiptService(s.Receipts)
	s.Posting.SetProvisionConsumer(s.Provisions)

	s.ClosingBookings = service.NewClosingBookingService(
		provisionRepo, journalRepo, s.Journal, settingsRepo, auditRepo, s.Closing,
	)
	s.ClosingBookings.SetReceiptService(s.Receipts)
	s.Closing.AddOpeningReverser(s.ClosingBookings.WorkInProgressReverser())

	s.Appropriation = service.NewAppropriationService(
		appropriationRepo, notesRepo, journalRepo, s.Journal,
		settingsRepo, auditRepo, s.Closing,
	)
	s.Appropriation.SetReceiptService(s.Receipts)
	s.Closing.SetReservePosting(s.Receipts)
	s.Closing.SetNotesCopier(s.Appropriation)

	s.InputTax = service.NewInputTaxService(
		inputTaxRepo, s.Journal, journalRepo, settingsRepo, s.Closing, auditRepo)
	s.VatIDs = service.NewVatIDService(vatIDCheckRepo, contactRepo, settingsRepo, auditRepo)
	s.VatIDs.SetClientFactory(o.Network.VatID)
	s.Invoices.SetVatIDConfirmer(s.VatIDs)
	s.Contacts.SetVatIDStatusSource(s.VatIDs)

	s.Gifts = service.NewGiftService(journalRepo, s.Journal, settingsRepo, auditRepo)
	s.Posting.SetGiftRegister(s.Gifts)
	s.Posting.SetCurrencyConverter(s.Currency)
	s.Assets.SetInputTaxRegister(s.InputTax)

	s.Currency.SetOpenItemSource(s.Payments)
	s.Currency.SetClosingService(s.Closing)
	s.Closing.SetCurrencyReverser(s.Currency)

	s.TaxRegister = service.NewTaxRegisterService(
		assetRepo, provisionRepo, journalRepo, settingsRepo, s.Closing)

	s.ClosingSteps = service.NewClosingStepsService(
		closingStepRepo, accrualRepo, provisionRepo,
		appropriationRepo, checkRunRepo, journalRepo, auditRepo, s.Closing,
	)
	s.ClosingSteps.SetDepreciationSource(s.Assets)
	s.ClosingSteps.SetWriteUpSource(s.Assets)
	s.ClosingSteps.SetInputTaxSource(s.InputTax)

	s.Statement = service.NewStatementService(s.Accounting, s.Closing, settingsRepo, auditRepo)
	s.Statement.SetOpenItemSource(s.Payments)
	s.Closing.SetRevenueSource(s.Statement)
	s.Statement.SetRenderer(renderer)
	s.Foundation.SetStatementSource(s.Statement)
	s.Foundation.SetRenderer(renderer)
	s.Foundation.SetReceiptService(s.Receipts)
	s.Closing.SetStatementSource(s.Statement)
	s.ClosingBookings.SetOpenItemSource(s.Payments)
	s.ClosingBookings.SetRevenueSource(s.Statement)
	s.Statement.SetNotesSources(service.NotesSources{
		Provisions:     s.Provisions,
		Reconciliation: s.TaxRegister,
		Texts:          s.Appropriation,
	})

	s.EBilanz = service.NewEBilanzService(s.Statement, settingsRepo, auditRepo)
	s.EBilanz.SetAdjustmentSource(s.TaxRegister)

	s.VatReturns = service.NewVatReturnService(
		journalRepo, receiptRepo, contactRepo, allocationRepo, settingsRepo,
		festschreibungRepo, vatReturnRepo, auditRepo,
	)
	s.ZM = service.NewZMService(
		journalRepo, contactRepo, settingsRepo, festschreibungRepo,
		zmReturnRepo, vatReturnRepo, auditRepo,
	)

	s.Checks = service.NewCheckService(
		journalRepo, receiptRepo, bankRepo, invoiceRepo, numberRepo,
		settingsRepo, festschreibungRepo, vatReturnRepo, checkRunRepo,
		auditRepo,
	)
	s.Checks.SetFiscalYearSource(fiscalYearRepo)
	s.Checks.SetAccountSource(s.Accounting)
	s.Checks.SetOpenItemSource(s.Payments)
	s.Checks.SetDepreciationSource(s.Assets)
	s.Checks.SetProvisionSource(s.Provisions)
	s.Checks.SetClosingStepSource(s.ClosingSteps)
	s.Checks.SetSizeClassSource(s.Statement)
	s.Checks.SetVatIDStatusSource(s.VatIDs)

	s.Obligations = service.NewObligationService(settingsRepo, obligationRepo, auditRepo)
	s.Obligations.SetVatSources(s.VatReturns, vatReturnRepo)
	s.Obligations.SetZMSources(s.ZM, zmReturnRepo)
	s.Obligations.SetCommitments(festschreibungRepo)
	s.Obligations.SetFiscalYearSource(fiscalYearRepo)
	s.Obligations.SetSizeSource(s.Statement)
	s.Obligations.SetAppropriationSource(s.Appropriation)
	s.Obligations.SetJournalSource(journalRepo)
	s.Obligations.SetDeclarationSource(declarationRepo)
	s.Settings.SetEventRecorder(s.Obligations)

	s.Export = service.NewExportService(
		journalRepo, accountRepo, contactRepo, receiptRepo, assetRepo,
		allocationRepo, auditRepo, settingsRepo, festschreibungRepo,
		vatReturnRepo, checkRunRepo, fiscalYearRepo,
		store, info.DataDir,
	)
	s.Export.SetTenantName(info.Name)
	s.Export.SetOpenItemSource(s.Payments)
	s.Export.SetIntegritySource(integrityChecks{journal: s.Journal, receipts: s.Receipts})
	s.Export.SetTaxRegisterSource(s.TaxRegister)
	s.Export.SetProcDocRepo(procDocRepo)

	s.Backup = service.NewBackupService(backupRunRepo, auditRepo, db, info.DataDir, info.ID, info.Name)

	s.Bank.SetOpenItemSource(s.Payments)
	s.Bank.SetCreditCardSource(s.Payments)
	s.Bank.SetRuleRepo(bankRuleRepo)

	s.AuditTrail = service.NewAuditTrailService(receiptRepo, journalRepo, allocationRepo, bankRepo, auditRepo)
	s.AuditTrail.SetRenderer(renderer)

	s.Shareholders = service.NewShareholderService(
		repository.NewShareholderLoanRepository(db), journalRepo, contactRepo, settingsRepo, auditRepo,
		s.Journal, s.Posting, s.Receipts)
	s.Payments.SetLoanSource(s.Shareholders)
	s.Payroll = service.NewPayrollService(
		repository.NewPayrollRepository(db), journalRepo, s.Journal, s.Posting, auditRepo)
	s.Checks.AddFindingSource(s.Shareholders)
	s.Checks.AddFindingSource(s.Payroll)

	s.MonthClose = service.NewMonthCloseService(s.Checks, s.VatReturns, festschreibungRepo)
	s.MonthClose.SetFiscalYearSource(fiscalYearRepo)

	s.Tasks = service.NewTaskService(settingsRepo)
	s.Tasks.SetCheckSource(s.Checks)
	s.Tasks.SetObligationSource(s.Obligations)
	s.Tasks.SetBankSource(s.Bank)
	s.Tasks.SetReceiptSource(s.Receipts)
	s.Tasks.SetOpenItemSource(s.Payments)
	s.Tasks.SetExemptionSource(s.Contacts)
	s.Tasks.SetBackupSource(s.Backup)
	s.Tasks.SetCarryForwardSource(s.Closing)

	s.Declarations = service.NewDeclarationService(declarationRepo, auditRepo,
		service.DeclarationSources{
			Journal: journalRepo, Settings: settingsRepo, FiscalYears: fiscalYearRepo,
			Appropriations: appropriationRepo, Assets: assetRepo, InputTax: inputTaxRepo,
			VatReturns: s.VatReturns, ZM: s.ZM, Statement: s.Statement, TaxRegister: s.TaxRegister,
		})

	return s, st
}

// integrityChecks bündelt die beiden Prüfläufe für den Export, der weder den
// Journaldienst noch die Belegablage kennen soll.
type integrityChecks struct {
	journal  *service.JournalService
	receipts *service.ReceiptService
}

func (c integrityChecks) VerifyIntegrity(ctx context.Context) (domain.IntegrityCheckResult, error) {
	return c.journal.VerifyIntegrity(ctx)
}

func (c integrityChecks) VerifyReceiptFiles(ctx context.Context) (*domain.FileCheckResult, error) {
	return c.receipts.VerifyReceiptFiles(ctx)
}

// procDocEnvironment beschreibt Ablage und Sicherung für die
// Verfahrensdokumentation.
func procDocEnvironment(dataDir, backupDir string) service.ProcDocEnvironment {
	return service.ProcDocEnvironment{
		DataDir:      dataDir,
		BackupDir:    backupDir,
		BackupRhythm: backupRhythmLabel(backupDir),
	}
}

// backupRhythmLabel beschreibt den Sicherungsrhythmus als Satz: die
// Verfahrensdokumentation muss ihn beschreiben und kann auf keine
// Einstellung verweisen.
func backupRhythmLabel(backupDir string) string {
	if backupDir == "" {
		return "nicht eingerichtet — es wird nicht gesichert"
	}
	return "täglich: beim Öffnen des Mandanten und stündlich während der Arbeit, sobald die letzte erfolgreiche Sicherung länger als einen Tag zurückliegt, und beim Beenden, wenn seitdem gearbeitet wurde"
}
