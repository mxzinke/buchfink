import { bridge as Bridge } from './bridge';
import skr04CatalogData from '../assets/skr04_2026.json';
import type {
  Account,
  AccountLedger,
  AcquisitionAdvice,
  AcquisitionCandidate,
  Anlagenspiegel,
  AppConfig,
  AssetAccountInfo,
  AssetClass,
  AssetDetail,
  AssetRules,
  AssetScheduleYear,
  AssetSummary,
  AuditLogEntry,
  BankTransaction,
  Cents,
  CompanySettings,
  Contact,
  DepreciationMethod,
  DepreciationResult,
  DepreciationRun,
  DifferenceKindInfo,
  Direction,
  DisposalPreview,
  DisposalRequest,
  DisposalResult,
  EInvoiceProposal,
  Festschreibung,
  FestschreibungVerification,
  FinancialSummary,
  FixedAsset,
  IntegrityCheckResult,
  Invoice,
  JournalEntry,
  OpenItem,
  PaymentRequest,
  PostingGroup,
  PostingPreview,
  Receipt,
  ReceiptFileInput,
  ReceiptPreview,
  ReceiptRequest,
  ReceiptStatus,
  SKR04Catalog,
  SuSaOverview,
  TaxTreatmentInfo,
  TenantConfig,
  ValidationResult,
  VatSummary,
} from '../types';

/**
 * Dünne Schicht über die Wails-Bindings.
 *
 * Diese Datei hatte früher für jeden Aufruf einen Fallback auf Beispieldaten.
 * In einer Buchhaltung ist das gefährlich: eine fehlgeschlagene Buchung sah
 * damit aus wie eine erfolgreiche. Fehler werden deshalb durchgereicht und in
 * der Oberfläche angezeigt. Ohne Backend — etwa beim Aufruf im Browser über
 * `npm run dev` — meldet jeder Aufruf das ehrlich.
 */

export class BackendUnavailableError extends Error {
  constructor() {
    super(
      'Keine Verbindung zum Buchfink-Backend. Die Oberfläche läuft ohne die Desktop-App ' +
        'nur als Vorschau — starte sie mit „wails3 dev".'
    );
    this.name = 'BackendUnavailableError';
  }
}

function isWailsRuntime(): boolean {
  return typeof window !== 'undefined' && Boolean((window as any)._wails);
}

/**
 * call führt einen Bridge-Aufruf aus und übersetzt eine fehlende Laufzeit in
 * eine verständliche Meldung. Fachliche Fehler des Backends — „Konto 8400 ist
 * nicht bebuchbar", „Zeitraum ist festgeschrieben" — kommen unverändert an.
 */
async function call<T>(fn: () => Promise<T>): Promise<T> {
  if (!isWailsRuntime()) throw new BackendUnavailableError();
  return fn();
}

const catalog = skr04CatalogData as unknown as SKR04Catalog;

export const Api = {
  isBackendAvailable: isWailsRuntime,

  // --- Mandanten ---------------------------------------------------------

  getTenants: (): Promise<TenantConfig[]> => call(() => Bridge.GetTenants() as Promise<TenantConfig[]>),
  getActiveTenant: (): Promise<TenantConfig | null> =>
    call(() => Bridge.GetActiveTenant() as Promise<TenantConfig | null>),
  switchTenant: (tenantId: string): Promise<void> => call(() => Bridge.SwitchTenant(tenantId) as Promise<void>),
  createTenant: (name: string, dataDir: string, settings: CompanySettings): Promise<TenantConfig> =>
    call(() => Bridge.CreateTenant(name, dataDir, settings as any) as Promise<TenantConfig>),
  importTenant: (dbFilePath: string): Promise<TenantConfig> =>
    call(() => Bridge.ImportTenant(dbFilePath) as Promise<TenantConfig>),
  deleteTenant: (tenantId: string): Promise<void> => call(() => Bridge.DeleteTenant(tenantId) as Promise<void>),
  isLocked: (): Promise<boolean> => call(() => Bridge.IsLocked() as Promise<boolean>),

  // --- Einrichtung -------------------------------------------------------

  getAppConfig: (): Promise<AppConfig> => call(() => Bridge.GetAppConfig() as Promise<AppConfig>),
  setupApplication: (dataDir: string, settings: CompanySettings): Promise<void> =>
    call(() => Bridge.SetupApplication(dataDir, settings as any) as Promise<void>),
  loadExistingDatabase: (dbFilePath: string): Promise<void> =>
    call(() => Bridge.LoadExistingDatabase(dbFilePath) as Promise<void>),
  selectDirectoryDialog: (title: string): Promise<string> =>
    call(() => Bridge.SelectDirectoryDialog(title) as Promise<string>),
  selectDatabaseFileDialog: (title: string): Promise<string> =>
    call(() => Bridge.SelectDatabaseFileDialog(title) as Promise<string>),
  selectRecoveryFile: (title: string): Promise<string> =>
    call(() => Bridge.SelectRecoveryFileDialog(title) as Promise<string>),
  exportRecoveryKey: (): Promise<string> => call(() => Bridge.ExportRecoveryKey() as Promise<string>),
  recoverFromFile: (path: string): Promise<void> =>
    call(() => Bridge.RecoverActiveTenantFromFile(path) as Promise<void>),

  // --- Geschäftsjahr & Stammdaten ---------------------------------------

  getFiscalYear: (): Promise<number> => call(() => Bridge.GetFiscalYear() as Promise<number>),
  setFiscalYear: (year: number): Promise<void> => call(() => Bridge.SetFiscalYear(year) as Promise<void>),
  getAvailableFiscalYears: (): Promise<number[]> =>
    call(() => Bridge.GetAvailableFiscalYears() as Promise<number[]>),
  getCompanySettings: (): Promise<CompanySettings> =>
    call(() => Bridge.GetCompanySettings() as Promise<CompanySettings>),
  updateCompanySettings: (settings: CompanySettings): Promise<void> =>
    call(() => Bridge.UpdateCompanySettings(settings as any) as Promise<void>),

  // --- Konten ------------------------------------------------------------

  getAccounts: (): Promise<Account[]> => call(() => Bridge.GetAccounts() as Promise<Account[]>),
  getAccountByNumber: (number: string): Promise<Account> =>
    call(() => Bridge.GetAccountByNumber(number) as Promise<Account>),
  getAccountLedger: (accountNumber: string): Promise<AccountLedger> =>
    call(() => Bridge.GetAccountLedger(accountNumber) as Promise<AccountLedger>),
  getSuSaOverview: (): Promise<SuSaOverview> => call(() => Bridge.GetSuSaOverview() as Promise<SuSaOverview>),
  getPaymentAccounts: (): Promise<Account[]> => call(() => Bridge.GetPaymentAccounts() as Promise<Account[]>),

  /** Der SKR04-Katalog ist statisch und liegt dem Frontend als Datei bei. */
  getSKR04Catalog: (): SKR04Catalog => catalog,

  // --- Kontierung --------------------------------------------------------

  getPostingGroups: (direction: Direction | ''): Promise<PostingGroup[]> =>
    call(() => Bridge.GetPostingGroups(direction) as Promise<PostingGroup[]>),
  getTaxTreatments: (direction: Direction): Promise<TaxTreatmentInfo[]> =>
    call(() => Bridge.GetTaxTreatments(direction) as Promise<TaxTreatmentInfo[]>),
  getDifferenceKinds: (): Promise<DifferenceKindInfo[]> =>
    call(() => Bridge.GetDifferenceKinds() as Promise<DifferenceKindInfo[]>),

  // --- Journal -----------------------------------------------------------

  getJournalEntries: (): Promise<JournalEntry[]> =>
    call(() => Bridge.GetJournalEntries() as Promise<JournalEntry[]>),
  getAllJournalEntries: (): Promise<JournalEntry[]> =>
    call(() => Bridge.GetAllJournalEntries() as Promise<JournalEntry[]>),
  postJournalEntry: (entry: Partial<JournalEntry>): Promise<JournalEntry> =>
    call(() => Bridge.PostJournalEntry(entry as any) as Promise<JournalEntry>),
  postIncomingReceipt: (request: ReceiptRequest): Promise<JournalEntry> =>
    call(() => Bridge.PostIncomingReceipt(request as any) as Promise<JournalEntry>),
  /** Der Buchungssatz, wie er gebucht würde. Das Frontend rechnet ihn nicht nach. */
  previewIncomingReceipt: (request: ReceiptRequest): Promise<PostingPreview> =>
    call(() => Bridge.PreviewIncomingReceipt(request as any) as Promise<PostingPreview>),
  previewOutgoingInvoice: (invoice: Partial<Invoice>): Promise<PostingPreview> =>
    call(() => Bridge.PreviewOutgoingInvoice(invoice as any) as Promise<PostingPreview>),

  // --- Belege ------------------------------------------------------------

  selectReceiptFiles: (title = 'Belegdateien auswählen'): Promise<string[]> =>
    call(() => Bridge.SelectReceiptFilesDialog(title)),
  fileIncomingReceipt: (
    files: ReceiptFileInput[],
    receivedAt = '',
    receivedVia = 'upload',
  ): Promise<Receipt> =>
    call(() => Bridge.FileIncomingReceipt(receivedAt, receivedVia, files) as Promise<Receipt>),
  addReceiptFile: (receiptId: number, file: ReceiptFileInput): Promise<Receipt> =>
    call(() => Bridge.AddReceiptFile(receiptId, file) as Promise<Receipt>),
  removeReceiptFile: (receiptId: number, fileId: number): Promise<Receipt> =>
    call(() => Bridge.RemoveReceiptFile(receiptId, fileId) as Promise<Receipt>),
  getReceipts: (status: ReceiptStatus | '' = ''): Promise<Receipt[]> =>
    call(() => Bridge.GetReceipts(status) as Promise<Receipt[]>),
  getReceipt: (id: number): Promise<Receipt> => call(() => Bridge.GetReceipt(id) as Promise<Receipt>),
  discardReceipt: (id: number, reason: string): Promise<void> =>
    call(() => Bridge.DiscardReceipt(id, reason)),
  getReceiptPreview: (receiptId: number): Promise<ReceiptPreview> =>
    call(() => Bridge.GetReceiptPreview(receiptId) as Promise<ReceiptPreview>),
  /** Zieht den strukturierten Rechnungsdatensatz aus einem abgelegten Beleg. */
  extractStructuredPart: (receiptId: number): Promise<Receipt> =>
    call(() => Bridge.ExtractStructuredPart(receiptId) as Promise<Receipt>),
  /** Buchungsvorschlag aus dem strukturierten Teil. Das Konto bleibt offen. */
  proposeFromEInvoice: (receiptId: number): Promise<EInvoiceProposal> =>
    call(() => Bridge.ProposeFromEInvoice(receiptId) as Promise<EInvoiceProposal>),
  /** Prüft den strukturierten Teil erneut. Das Regelwerk ist versioniert. */
  validateEInvoice: (receiptId: number): Promise<ValidationResult> =>
    call(() => Bridge.ValidateEInvoice(receiptId) as Promise<ValidationResult>),
  /** Die Regeln, die Buchfink prüft — der Prüfumfang ist Teil des Ergebnisses. */
  getEInvoiceRules: (): Promise<string[]> => call(() => Bridge.GetEInvoiceRules()),
  /** Die Regeln, die Buchfink nicht prüft, je mit Begründung. */
  getUncheckedEInvoiceRules: (): Promise<Record<string, string>> =>
    call(() => Bridge.GetUncheckedEInvoiceRules()),
  /** Das archivierte Rechnungsdokument — dasselbe PDF, das der Kunde bekommen hat. */
  getInvoiceDocument: (invoiceId: number): Promise<ReceiptPreview> =>
    call(() => Bridge.GetInvoiceDocument(invoiceId) as Promise<ReceiptPreview>),
  reverseJournalEntry: (entryId: number, reason: string): Promise<JournalEntry> =>
    call(() => Bridge.ReverseJournalEntry(entryId, reason) as Promise<JournalEntry>),
  verifyIntegrity: (): Promise<IntegrityCheckResult> =>
    call(() => Bridge.VerifyIntegrity() as Promise<IntegrityCheckResult>),
  getFinancialSummary: (): Promise<FinancialSummary> =>
    call(() => Bridge.GetFinancialSummary() as Promise<FinancialSummary>),
  getVatSummary: (from = '', to = ''): Promise<VatSummary> =>
    call(() => Bridge.GetVatSummary(from, to) as Promise<VatSummary>),

  // --- Bank & Zahlungen --------------------------------------------------

  getBankTransactions: (): Promise<BankTransaction[]> =>
    call(() => Bridge.GetBankTransactions() as Promise<BankTransaction[]>),
  importCAMT: (xmlContent: string, ledgerAccount: string): Promise<number> =>
    call(() => Bridge.ImportCAMT053XML(xmlContent, ledgerAccount) as Promise<number>),
  bookBankTransactionDirect: (
    bankTxId: number,
    counterAccount: string,
    description: string
  ): Promise<JournalEntry> =>
    call(() => Bridge.BookBankTransactionDirect(bankTxId, counterAccount, description) as Promise<JournalEntry>),
  ignoreBankTransaction: (bankTxId: number): Promise<void> =>
    call(() => Bridge.IgnoreBankTransaction(bankTxId) as Promise<void>),
  getOpenItems: (): Promise<OpenItem[]> => call(() => Bridge.GetOpenItems() as Promise<OpenItem[]>),
  settlePayment: (request: PaymentRequest): Promise<JournalEntry> =>
    call(() => Bridge.SettlePayment(request as any) as Promise<JournalEntry>),

  // --- Kontakte & Rechnungen --------------------------------------------

  getContacts: (): Promise<Contact[]> => call(() => Bridge.GetContacts() as Promise<Contact[]>),
  saveContact: (contact: Partial<Contact>): Promise<Contact> =>
    call(() => Bridge.SaveContact(contact as any) as Promise<Contact>),
  deleteContact: (id: number): Promise<void> => call(() => Bridge.DeleteContact(id) as Promise<void>),
  getInvoices: (): Promise<Invoice[]> => call(() => Bridge.GetInvoices() as Promise<Invoice[]>),
  issueInvoice: (invoice: Partial<Invoice>): Promise<Invoice> =>
    call(() => Bridge.IssueInvoice(invoice as any) as Promise<Invoice>),
  cancelInvoice: (invoiceId: number, reason: string): Promise<void> =>
    call(() => Bridge.CancelInvoice(invoiceId, reason) as Promise<void>),
  generateInvoiceZUGFeRD: (invoiceId: number): Promise<[string, string]> =>
    call(() => Bridge.GenerateInvoiceZUGFeRD(invoiceId) as Promise<[string, string]>),

  // --- Anlagevermögen ----------------------------------------------------

  getFixedAssets: (assetClass: AssetClass | '' = ''): Promise<FixedAsset[]> =>
    call(() => Bridge.GetFixedAssets(assetClass) as Promise<FixedAsset[]>),
  getAssetSummary: (assetClass: AssetClass | '' = ''): Promise<AssetSummary> =>
    call(() => Bridge.GetAssetSummary(assetClass) as Promise<AssetSummary>),
  getFixedAsset: (id: number): Promise<AssetDetail> =>
    call(() => Bridge.GetFixedAsset(id) as Promise<AssetDetail>),
  saveFixedAsset: (asset: Partial<FixedAsset>): Promise<FixedAsset> =>
    call(() => Bridge.SaveFixedAsset(asset as any) as Promise<FixedAsset>),
  deleteFixedAsset: (id: number): Promise<void> => call(() => Bridge.DeleteFixedAsset(id)),
  /** Nachträgliche Anschaffungskosten oder eine Minderung — ohne eigene Buchung. */
  recordAssetCostAdjustment: (request: {
    assetId: number;
    date: string;
    amount: Cents;
    reduction?: boolean;
    /** Monate, um die sich die Restnutzungsdauer ab diesem Jahr verlängert. */
    extendLifeMonths?: number;
    note?: string;
    journalEntryId?: number;
  }): Promise<FixedAsset> =>
    call(() => Bridge.RecordAssetCostAdjustment(request as any) as Promise<FixedAsset>),
  /** Der kuratierte Katalog der Anlagekonten, je mit seinem AfA-Konto. */
  getAssetAccounts: (assetClass: AssetClass | '' = ''): Promise<AssetAccountInfo[]> =>
    call(() => Bridge.GetAssetAccounts(assetClass) as Promise<AssetAccountInfo[]>),
  /** Wertgrenzen, Zeitfenster der degressiven AfA und zulässige Methoden. */
  getAssetRules: (): Promise<AssetRules> => call(() => Bridge.GetAssetRules() as Promise<AssetRules>),
  /**
   * Sofortabzug, Sammelposten oder aktivieren? Die Antwort kommt aus dem
   * Backend, damit die Wertgrenzen nur an einer Stelle stehen.
   */
  classifyAcquisition: (netCost: Cents, date: string, selfUsable: boolean): Promise<AcquisitionAdvice> =>
    call(() => Bridge.ClassifyAcquisition(netCost, date, selfUsable) as Promise<AcquisitionAdvice>),
  /** Der Abschreibungsplan für eine Eingabe, die noch kein Anlagegut ist. */
  previewDepreciationPlan: (request: {
    acquisitionDate: string;
    cost: Cents;
    usefulLifeMonths: number;
    method: DepreciationMethod;
    poolYear?: number;
  }): Promise<AssetScheduleYear[]> =>
    call(() => Bridge.PreviewDepreciationPlan(request as any) as Promise<AssetScheduleYear[]>),
  getDepreciationRun: (): Promise<DepreciationRun> =>
    call(() => Bridge.GetDepreciationRun() as Promise<DepreciationRun>),
  bookDepreciationRun: (request: {
    fiscalYear: number;
    bookingDate?: string;
    assetIds?: number[];
  }): Promise<DepreciationResult> =>
    call(() => Bridge.BookDepreciationRun(request as any) as Promise<DepreciationResult>),
  bookAssetImpairment: (request: {
    assetId: number;
    date: string;
    amount: Cents;
    permanent: boolean;
    reason: string;
  }): Promise<JournalEntry> =>
    call(() => Bridge.BookAssetImpairment(request as any) as Promise<JournalEntry>),
  bookAssetWriteUp: (request: {
    assetId: number;
    date: string;
    amount: Cents;
    reason: string;
  }): Promise<JournalEntry> =>
    call(() => Bridge.BookAssetWriteUp(request as any) as Promise<JournalEntry>),
  /** Fertigstellung: Umbuchung von der Anlage im Bau auf ihr endgültiges Konto. */
  transferFixedAsset: (request: {
    assetId: number;
    date: string;
    account: string;
    depreciationAccount?: string;
    method: DepreciationMethod;
    usefulLifeMonths: number;
    note?: string;
  }): Promise<FixedAsset> =>
    call(() => Bridge.TransferFixedAsset(request as any) as Promise<FixedAsset>),
  previewAssetDisposal: (request: DisposalRequest): Promise<DisposalPreview> =>
    call(() => Bridge.PreviewAssetDisposal(request as any) as Promise<DisposalPreview>),
  disposeFixedAsset: (request: DisposalRequest): Promise<DisposalResult> =>
    call(() => Bridge.DisposeFixedAsset(request as any) as Promise<DisposalResult>),
  getAnlagenspiegel: (): Promise<Anlagenspiegel> =>
    call(() => Bridge.GetAnlagenspiegel() as Promise<Anlagenspiegel>),
  /** Buchungen auf Anlagekonten, zu denen noch kein Anlagegut erfasst ist. */
  getAssetAcquisitionCandidates: (): Promise<AcquisitionCandidate[]> =>
    call(() => Bridge.GetAssetAcquisitionCandidates() as Promise<AcquisitionCandidate[]>),
  getSammelposten: (fiscalYear = 0): Promise<FixedAsset | null> =>
    call(() => Bridge.GetSammelposten(fiscalYear) as Promise<FixedAsset | null>),

  // --- E-Bilanz, Audit & Festschreibung ---------------------------------

  exportEBilanzXBRL: (): Promise<string> => call(() => Bridge.ExportEBilanzXBRL() as Promise<string>),
  getAuditLogs: (): Promise<AuditLogEntry[]> => call(() => Bridge.GetAuditLogs() as Promise<AuditLogEntry[]>),
  getFestschreibungen: (): Promise<Festschreibung[]> =>
    call(() => Bridge.GetFestschreibungen() as Promise<Festschreibung[]>),
  commitPeriod: (periodType: string, periodLabel: string, cutoffDate: string): Promise<Festschreibung> =>
    call(() => Bridge.CommitPeriod(periodType, periodLabel, cutoffDate) as Promise<Festschreibung>),
  verifyFestschreibung: (id: number): Promise<FestschreibungVerification> =>
    call(() => Bridge.VerifyFestschreibung(id) as Promise<FestschreibungVerification>),
};
