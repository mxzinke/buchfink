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
  AssetDocumentKind,
  AssetDocumentKindInfo,
  AssetRules,
  AssetScheduleYear,
  AssetSummary,
  AuditLogEntry,
  BankTransaction,
  CarryForwardPreview,
  Cents,
  ClosingState,
  CompanySettings,
  Contact,
  CurrencyValuation,
  Deadline,
  DepreciationMethod,
  DepreciationResult,
  DepreciationRun,
  DifferenceKindInfo,
  Direction,
  DisposalPreview,
  DisposalRequest,
  DisposalResult,
  EInvoiceProposal,
  ExpiringAssetDocument,
  Festschreibung,
  FestschreibungVerification,
  FinancialStatement,
  FinancialSummary,
  FiscalYear,
  FiscalYearStatus,
  FixedAsset,
  Foundation,
  FoundationPostingPreview,
  FoundationRules,
  FoundationState,
  IntegrityCheckResult,
  InvestmentRules,
  InvestmentTaxNote,
  Invoice,
  JournalEntry,
  LegalFormInfo,
  MappingReport,
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
  Settlement,
  SizeClass,
  StatementDepth,
  SuSaOverview,
  TaxRate,
  TaxTreatment,
  TaxTreatmentInfo,
  TenantConfig,
  Units,
  ValidationResult,
  VatSummary,
  Vorabpauschale,
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
    /** Zugekaufte Stückzahl bei einer Finanzanlage. */
    quantity?: Units;
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
    /** Sonderabschreibung nach § 7g Abs. 5 EStG, in Promille und über so viele Jahre. */
    specialPermille?: number;
    specialYears?: number;
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
  /**
   * Erhaltungsaufwand: eine Buchung, die zum Anlagegut gehört, seinen Wert aber
   * nicht ändert. Genau das unterscheidet sie von den nachträglichen
   * Herstellungskosten (§ 255 Abs. 2 Satz 1 HGB).
   */
  bookAssetMaintenance: (request: {
    assetId: number;
    date: string;
    amount: Cents;
    account?: string;
    taxTreatment?: TaxTreatment;
    taxRate?: TaxRate;
    settlement: Settlement;
    paymentAccount?: string;
    contactId?: number;
    note: string;
  }): Promise<JournalEntry> =>
    call(() => Bridge.BookAssetMaintenance(request as any) as Promise<JournalEntry>),
  /** Dividende, Ausschüttung oder Zins, verknüpft mit dem Anteil, aus dem sie stammt. */
  bookAssetIncome: (request: {
    assetId: number;
    date: string;
    amount: Cents;
    account?: string;
    taxTreatment?: TaxTreatment;
    settlement: Settlement;
    paymentAccount?: string;
    contactId?: number;
    /** Einbehaltene Kapitalertragsteuer: sie mindert den Zufluss, nicht den Ertrag. */
    withholdingTax?: Cents;
    note?: string;
  }): Promise<JournalEntry> =>
    call(() => Bridge.BookAssetIncome(request as any) as Promise<JournalEntry>),
  /**
   * Umrechnung zum Devisenkassamittelkurs des Stichtags (§ 256a HGB). Bucht
   * nichts — es nennt den Betrag, den eine Abschreibung oder Zuschreibung hätte.
   */
  valuateAssetCurrency: (request: {
    assetId: number;
    date: string;
    /** Fremdwährungseinheiten je Euro, mal einer Million (RATE_SCALE). */
    ratePerEuro: number;
  }): Promise<CurrencyValuation> =>
    call(() => Bridge.ValuateAssetCurrency(request as any) as Promise<CurrencyValuation>),
  /** Bucht die Umrechnungsdifferenz auf die Konten der Währungsumrechnung (6880/4840). */
  bookAssetCurrencyValuation: (request: {
    assetId: number;
    date: string;
    ratePerEuro: number;
  }): Promise<JournalEntry> =>
    call(() => Bridge.BookAssetCurrencyValuation(request as any) as Promise<JournalEntry>),

  // --- Dokumente am Anlagegut -------------------------------------------

  /** Öffnet den nativen Dateidialog. Dateien reisen als Pfad, nicht als Inhalt. */
  selectAssetDocumentsDialog: (title = ''): Promise<string[]> =>
    call(() => Bridge.SelectAssetDocumentsDialog(title) as Promise<string[]>),
  attachAssetDocument: (request: {
    assetId: number;
    kind: AssetDocumentKind;
    path: string;
    title?: string;
    documentDate?: string;
    validUntil?: string;
    note?: string;
  }): Promise<FixedAsset> =>
    call(() => Bridge.AttachAssetDocument(request as any) as Promise<FixedAsset>),
  removeAssetDocument: (assetId: number, documentId: number): Promise<FixedAsset> =>
    call(() => Bridge.RemoveAssetDocument(assetId, documentId) as Promise<FixedAsset>),
  getAssetDocumentContent: (documentId: number): Promise<ReceiptPreview> =>
    call(() => Bridge.GetAssetDocumentContent(documentId) as Promise<ReceiptPreview>),
  getAssetDocumentKinds: (): Promise<AssetDocumentKindInfo[]> =>
    call(() => Bridge.GetAssetDocumentKinds() as Promise<AssetDocumentKindInfo[]>),
  /** Was bis zu einem Stichtag ausläuft. Leer heißt: bis zum Ende des Geschäftsjahres. */
  getExpiringAssetDocuments: (until = ''): Promise<ExpiringAssetDocument[]> =>
    call(() => Bridge.GetExpiringAssetDocuments(until) as Promise<ExpiringAssetDocument[]>),

  // --- Investmentanteile ------------------------------------------------

  /** Der Rechtsformkatalog, je mit dem, was die Form steuerlich nach sich zieht. */
  getLegalForms: (): Promise<LegalFormInfo[]> =>
    call(() => Bridge.GetLegalForms() as Promise<LegalFormInfo[]>),
  /** Fondsarten, Anlegerstellungen und der Satz, der sich aus beidem ergibt. */
  getInvestmentRules: (): Promise<InvestmentRules> =>
    call(() => Bridge.GetInvestmentRules() as Promise<InvestmentRules>),
  /**
   * Rechnet die Vorabpauschale eines Kalenderjahres (§ 18 InvStG) — und hält
   * sie fest, wenn `record` gesetzt ist. Gebucht wird nichts.
   */
  computeVorabpauschale: (request: {
    assetId: number;
    year: number;
    openingPrice: Cents;
    closingPrice: Cents;
    distributions?: Cents;
    /** Basiszins in Basispunkten: 253 sind 2,53 %. */
    basisPoints: number;
    record?: boolean;
    note?: string;
  }): Promise<Vorabpauschale> =>
    call(() => Bridge.ComputeVorabpauschale(request as any) as Promise<Vorabpauschale>),
  /** Die Teilfreistellung einer Ausschüttung, bevor sie gebucht wird. */
  getInvestmentNoteForIncome: (assetId: number, amount: Cents): Promise<InvestmentTaxNote> =>
    call(() => Bridge.GetInvestmentNoteForIncome(assetId, amount) as Promise<InvestmentTaxNote>),

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

  // --- Bilanz und Gewinn- und Verlustrechnung ---------------------------

  /**
   * Der fertige Abschluss eines Geschäftsjahres: Gliederung nach den §§ 266 und
   * 275 HGB mit Vorjahresspalte, Größenklasse, Angaben unter der Bilanz,
   * Fristen und Zuordnungsbericht.
   *
   * Leere Tiefe heißt „die Tiefe, die die Größenklasse vorgibt" — nicht die
   * volle Gliederung. Den Unterschied kennt nur das Backend.
   */
  getStatement: (year: number, depth: StatementDepth | '' = ''): Promise<FinancialStatement> =>
    call(() => Bridge.GetStatement(year, depth) as Promise<FinancialStatement>),
  getSizeClass: (year: number): Promise<SizeClass> =>
    call(() => Bridge.GetSizeClass(year) as Promise<SizeClass>),
  /** Aufstellung und Offenlegung mit Datum und Norm (§ 264 Abs. 1, § 325 HGB). */
  getStatementDeadlines: (year: number): Promise<Deadline[]> =>
    call(() => Bridge.GetStatementDeadlines(year) as Promise<Deadline[]>),
  /** Bilanz und GuV als PDF, Base64 wie der Rechnungsexport. */
  exportStatementPDF: (year: number): Promise<string> =>
    call(() => Bridge.ExportStatementPDF(year) as Promise<string>),
  /** Dieselbe Gliederung als CSV-Text (UTF-8, Semikolon). */
  exportStatementCSV: (year: number): Promise<string> =>
    call(() => Bridge.ExportStatementCSV(year) as Promise<string>),
  /** Das dritte Merkmal des § 267 Abs. 1 HGB; aus Buchungen nicht ableitbar. */
  setAverageEmployees: (year: number, count: number): Promise<FiscalYear> =>
    call(() => Bridge.SetAverageEmployees(year, count) as Promise<FiscalYear>),

  // --- E-Bilanz, Audit & Festschreibung ---------------------------------

  exportEBilanzXBRL: (): Promise<string> => call(() => Bridge.ExportEBilanzXBRL() as Promise<string>),
  /**
   * Welches Konto unter welcher Gliederungsposition und welchem
   * Taxonomie-Element erscheint — und was die Erzeugung verhindert.
   */
  getEBilanzMappingReport: (year: number): Promise<MappingReport> =>
    call(() => Bridge.GetEBilanzMappingReport(year) as Promise<MappingReport>),
  getAuditLogs: (): Promise<AuditLogEntry[]> => call(() => Bridge.GetAuditLogs() as Promise<AuditLogEntry[]>),
  getFestschreibungen: (): Promise<Festschreibung[]> =>
    call(() => Bridge.GetFestschreibungen() as Promise<Festschreibung[]>),
  commitPeriod: (periodType: string, periodLabel: string, cutoffDate: string): Promise<Festschreibung> =>
    call(() => Bridge.CommitPeriod(periodType, periodLabel, cutoffDate) as Promise<Festschreibung>),
  verifyFestschreibung: (id: number): Promise<FestschreibungVerification> =>
    call(() => Bridge.VerifyFestschreibung(id) as Promise<FestschreibungVerification>),

  // --- Jahresabschluss ---------------------------------------------------

  /** Die Geschäftsjahre als Entitäten: Zeitraum, Rumpfjahr, Abschlussstand. */
  getFiscalYears: (): Promise<FiscalYear[]> => call(() => Bridge.GetFiscalYears() as Promise<FiscalYear[]>),
  /** Legt das Geschäftsjahr an und schaltet auf es um. */
  createFiscalYear: (year: number): Promise<void> => call(() => Bridge.CreateFiscalYear(year)),
  getClosingState: (year: number): Promise<ClosingState> =>
    call(() => Bridge.GetClosingState(year) as Promise<ClosingState>),
  /**
   * Der Vortragsstand ins Zieljahr: je Konto Schlusssaldo des Vorjahres,
   * bereits vorgetragener Wert und Differenz. Bucht nichts.
   */
  getCarryForwardPreview: (toYear: number): Promise<CarryForwardPreview> =>
    call(() => Bridge.GetCarryForwardPreview(toYear) as Promise<CarryForwardPreview>),
  /** Bucht den Saldenvortrag; ein erneuter Lauf nimmt den bestehenden zurück. */
  carryForward: (toYear: number): Promise<JournalEntry[]> =>
    call(() => Bridge.CarryForward(toYear) as Promise<JournalEntry[]>),
  setFiscalYearStatus: (
    year: number,
    status: FiscalYearStatus,
    date: string,
    note = ''
  ): Promise<FiscalYear> =>
    call(() => Bridge.SetFiscalYearStatus(year, status, date, note) as Promise<FiscalYear>),
  /** Nimmt die Feststellung zurück; der Grund ist Pflicht und wird protokolliert. */
  reopenFiscalYear: (year: number, reason: string): Promise<FiscalYear> =>
    call(() => Bridge.ReopenFiscalYear(year, reason) as Promise<FiscalYear>),

  // --- Gründung ---------------------------------------------------------

  /**
   * Regeln, Gründung, Anmeldungsbefund, Unterbilanz und Fristen in einem Aufruf.
   * `applies` ist falsch, wenn die Rechtsform keine Kapitalgesellschaft ist.
   */
  getFoundationState: (): Promise<FoundationState> =>
    call(() => Bridge.GetFoundationState() as Promise<FoundationState>),
  saveFoundation: (foundation: Partial<Foundation>): Promise<Foundation> =>
    call(() => Bridge.SaveFoundation(foundation as any) as Promise<Foundation>),
  /** Die Kapitalaufbringungsregeln der Rechtsformen, die der Gründungsweg abdeckt. */
  getFoundationRules: (): Promise<FoundationRules[]> =>
    call(() => Bridge.GetFoundationRules() as Promise<FoundationRules[]>),
  /**
   * Voranmeldungszeitraum einer Gründung in diesem Jahr, mit Begründung.
   * § 18 Abs. 2 UStG hat dafür ein Stichjahr — deshalb wird gefragt statt geraten.
   */
  getRecommendedVatPeriod: (foundingYear: number): Promise<{ period: string; reason: string }> =>
    call(
      () =>
        Bridge.GetRecommendedVatPeriod(foundingYear) as Promise<{ period: string; reason: string }>
    ),
  previewFoundationPostings: (): Promise<FoundationPostingPreview> =>
    call(() => Bridge.PreviewFoundationPostings() as Promise<FoundationPostingPreview>),
  bookFoundationPostings: (): Promise<JournalEntry[]> =>
    call(() => Bridge.BookFoundationPostings() as Promise<JournalEntry[]>),
  /** Die Eintragung ins Handelsregister — sie beendet die Vorgesellschaft. */
  registerCompany: (date: string, court: string, number: string): Promise<Foundation> =>
    call(() => Bridge.RegisterCompany(date, court, number) as Promise<Foundation>),
  /** Erledigte Gründungspflicht mit ihrem Datum; leeres Datum nimmt sie zurück. */
  completeFoundationDuty: (key: string, doneOn: string, note = ''): Promise<void> =>
    call(() => Bridge.CompleteFoundationDuty(key, doneOn, note)),
};
