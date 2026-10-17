import { bridge as Bridge } from './bridge';
import type {
  Account,
  AccountLedger,
  Accrual,
  AccrualPreview,
  AccrualProposal,
  AccrualReport,
  AccrualRequest,
  AcquisitionAdvice,
  AcquisitionCandidate,
  AdvanceGroupRequest,
  AdvanceInvoiceRequest,
  AdvanceItem,
  AdvanceTargetOption,
  AfaRules,
  AgentClient,
  Anlagenspiegel,
  AppConfig,
  AppConsentRequest,
  Appropriation,
  AppropriationPreview,
  AppropriationRequest,
  AssetAccountInfo,
  AssetClass,
  AssetDetail,
  AssetRules,
  AssetScheduleYear,
  AuditChainResult,
  AuditTrail,
  AuditFilter,
  AuditLogEntry,
  BackupRun,
  BankAccount,
  Country,
  BankImportPreview,
  BankRule,
  BankSuggestions,
  BankTransaction,
  BlockContactResult,
  CarryForwardPreview,
  Cents,
  ChangelogEntry,
  CheckRun,
  ClosingSettings,
  ClosingState,
  ClosingSteps,
  CompanySettings,
  ComplianceHints,
  ConnectedApp,
  Contact,
  CorrectionResult,
  CustomAccountRequest,
  CurrencyValuation,
  Deadline,
  DepreciationMethod,
  DepreciationResult,
  DepreciationRun,
  DifferenceKindInfo,
  Direction,
  DiscountRate,
  DisposalPreview,
  DisposalRequest,
  DisposalResult,
  EInvoiceProfileInfo,
  EInvoiceProposal,
  ExchangeRate,
  ExemptionCertificateWarning,
  ExportResult,
  Festschreibung,
  FestschreibungVerification,
  FileCheckResult,
  FinalInvoiceRequest,
  FinancialStatement,
  FinancialSummary,
  FiscalYear,
  FiscalYearCandidates,
  FiscalYearStatus,
  FixedAsset,
  ForeignCurrencyValuation,
  Foundation,
  FoundationRules,
  FoundationState,
  GiftRebooking,
  InputTaxCorrection,
  InputTaxCorrectionYear,
  IntegrityCheckResult,
  InvestmentRules,
  Invoice,
  InvoiceGroup,
  InvoiceSentVia,
  InvoiceSentViaOption,
  JournalEntry,
  JournalFilter,
  JournalFilterResult,
  LegalFormInfo,
  ManualEntryRequest,
  MappingReport,
  MCPServerStatus,
  MigrationRecord,
  MonthCloseState,
  NonDeductibleReport,
  NotesSection,
  NotesSectionText,
  NumberGapReason,
  NumberGapReasonOption,
  NumberGapReport,
  OpeningBalancePreview,
  OpeningBalanceRequest,
  OpenItem,
  OpenItemsAging,
  OrganisationTexts,
  PaymentAllocationDetail,
  PaymentRequest,
  PoolConsistencyReport,
  PostingGroup,
  PostingPreview,
  ProcDocResult,
  ProcedureDocumentation,
  Provision,
  ProvisionChangeRequest,
  ProvisionMirror,
  ProvisionPreview,
  ProvisionRequest,
  RebookGiftsRequest,
  Receipt,
  ReceiptFileInput,
  ReceiptFindings,
  ReceiptHeader,
  ReceiptPreview,
  ReceiptRequest,
  ReceiptStatus,
  Reconciliation,
  RefundAdvanceRequest,
  RegisterInputTaxRequest,
  RetentionClass,
  RetentionRules,
  SaveInputTaxUsageRequest,
  SchemaMigration,
  SelfIssuedReceiptRequest,
  SettleAdvanceRequest,
  ShareholderLoanRequest,
  ShareholderLoanView,
  Settlement,
  SizeClass,
  SpecialPrepaymentSuggestion,
  StatementDepth,
  StatementPositionOption,
  SuSaOverview,
  TaxElectionRegister,
  TaxProvisionPreview,
  TaxProvisionRequest,
  Task,
  TaskList,
  TaxTreatment,
  TaxTreatmentInfo,
  TenantConfig,
  UnitCode,
  Units,
  VatExchangeRate,
  VatIDCheck,
  VatIDStatus,
  VatPeriodStatus,
  VatPeriodProposal,
  VatRateImport,
  VatReturn,
  VatSettlement,
  VendorAdvance,
  Vorabpauschale,
  WriteOffRequest,
  WriteUpReport,
  ZMPeriodStatus,
  ZMReturn,
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

export const COMPANY_SETTINGS_CHANGED = 'buchfink:company-settings-changed';

/**
 * call führt einen Bridge-Aufruf aus und übersetzt eine fehlende Laufzeit in
 * eine verständliche Meldung. Fachliche Fehler des Backends — „Konto 8400 ist
 * nicht bebuchbar", „Zeitraum ist festgeschrieben" — kommen unverändert an.
 */
async function call<T>(fn: () => Promise<T>): Promise<T> {
  if (!isWailsRuntime()) throw new BackendUnavailableError();
  return fn();
}

/**
 * Go serialisiert eine Liste, die nie befüllt wurde, als `null`. Die Oberfläche
 * rechnet dagegen überall mit einer Liste, und `null.length` reißt beim Rendern
 * die ganze Ansicht mit — ausgerechnet im Regelfall: kein Nachtrag, kein Befund,
 * keine Zeile. Was der Typ als Liste ankündigt, kommt deshalb an dieser Grenze
 * als Liste an, statt in jeder Ansicht einzeln abgefangen zu werden.
 */
function list<T>(value: T[] | null | undefined): T[] {
  return value ?? [];
}

/**
 * Die Parameter eines Navigationsziels. Go liefert eine leere Map als `null`,
 * und die Ansicht liest sie ohne weitere Prüfung.
 */
function normalizeTask(task: Task): Task {
  if (!task) return task;
  return { ...task, target: task.target ?? { area: '' } };
}

/** Die Listen einer Voranmeldung: Kennziffern mit Drill-down und Nachträge. */
function normalizeVatReturn(r: VatReturn): VatReturn {
  if (!r) return r;
  return {
    ...r,
    figures: list(r.figures).map((line) => ({ ...line, entryIds: list(line.entryIds) })),
    lateEntries: list(r.lateEntries),
  };
}

/** Dasselbe für die Zusammenfassende Meldung: Zeilen, Befunde, Nachträge. */
function normalizeZMReturn(r: ZMReturn): ZMReturn {
  if (!r) return r;
  return {
    ...r,
    lines: list(r.lines),
    findings: list(r.findings),
    lateEntries: list(r.lateEntries),
  };
}

/** Ein Prüflauf ohne Befund ist der gute Fall — und der mit der leeren Liste. */
function normalizeCheckRun(run: CheckRun): CheckRun {
  if (!run) return run;
  return { ...run, findings: list(run.findings) };
}

/** Die Kette des Änderungsprotokolls: ohne Bruch ist die Liste leer. */
function normalizeAuditChain(result: AuditChainResult): AuditChainResult {
  if (!result) return result;
  return { ...result, breaks: list(result.breaks) };
}

/** Dasselbe für die Prüfläufe über Kette und Dateien: kein Befund ist der Regelfall. */
function normalizeIntegrity(result: IntegrityCheckResult): IntegrityCheckResult {
  if (!result) return result;
  return {
    ...result,
    breaks: list(result.breaks),
    fiscalYears: list(result.fiscalYears),
    // Die Protokollkette hängt seit Welle 6 am selben Ergebnis. Sie kann
    // fehlen — dann bleibt das Feld leer und behauptet nichts.
    auditChain: result.auditChain ? normalizeAuditChain(result.auditChain) : undefined,
  };
}

function normalizeFileCheck(result: FileCheckResult): FileCheckResult {
  if (!result) return result;
  return { ...result, issues: list(result.issues) };
}

/** Ein Export ohne Hinweise ist der gute Fall — und der mit den leeren Listen. */
function normalizeExport(result: ExportResult): ExportResult {
  if (!result) return result;
  return {
    ...result,
    tables: list(result.tables),
    files: list(result.files),
    notes: list(result.notes),
  };
}

/**
 * Die Listen der Abschlussbausteine.
 *
 * Sie sind der Regelfall des leeren Jahres: eine Rückstellung ohne Bewegung
 * gibt es nicht, wohl aber eine Abgrenzung, deren Auflösungsplan noch leer ist,
 * und einen Abschluss, in dem weder das eine noch das andere vorkommt. Die
 * Normalisierung steht hier und nicht in der Ansicht, weil sonst jede Tabelle
 * ihre eigene Absicherung trüge.
 */
function normalizeAccrual(accrual: Accrual): Accrual {
  if (!accrual) return accrual;
  return { ...accrual, releases: list(accrual.releases) };
}

function normalizeProvision(provision: Provision): Provision {
  if (!provision) return provision;
  return { ...provision, movements: list(provision.movements) };
}

/**
 * Der Anhang des Abschlusses. Seine drei Listen sind der Regelfall des ersten
 * Jahres: kein Text geschrieben, keine Rückstellung gebildet, keine Abweichung
 * zur Steuerbilanz. Sie hier zu sichern kostet nichts und hält die Ansicht
 * davon ab, an einer fehlenden Liste den ganzen Baum zu verlieren.
 */
function normalizeStatement(statement: FinancialStatement): FinancialStatement {
  if (!statement) return statement;
  const notes = statement.notes;
  if (!notes) return statement;
  return {
    ...statement,
    notes: {
      ...notes,
      texts: list(notes.texts),
      provisionMirror: notes.provisionMirror
        ? { ...notes.provisionMirror, rows: list(notes.provisionMirror.rows) }
        : notes.provisionMirror,
      reconciliation: notes.reconciliation
        ? { ...notes.reconciliation, rows: list(notes.reconciliation.rows) }
        : notes.reconciliation,
    },
  };
}

/**
 * Die Listen einer Rechnung: Positionen und die Bezüge auf vorausgegangene
 * Rechnungen (BG-3). Eine gewöhnliche Rechnung hat keine Bezüge, und ohne
 * diese Sicherung stünde in jeder Ansicht, die sie liest, ein eigener
 * Standardwert — oder eben keiner.
 */
function normalizeInvoice(invoice: Invoice): Invoice {
  if (!invoice) return invoice;
  return { ...invoice, items: list(invoice.items), precedingRefs: list(invoice.precedingRefs) };
}

/** Ein Verbund ohne Abschlagsrechnung ist der Zustand direkt nach dem Anlegen. */
function normalizeInvoiceGroup(group: InvoiceGroup): InvoiceGroup {
  if (!group) return group;
  return {
    ...group,
    advances: list(group.advances),
    // Der Fortschritt kommt aus dem Backend. Fehlt er (ein Verbund aus einem
    // älteren Aufruf), ist der Stand der eines Verbunds ohne Abschlag: nichts
    // abgerechnet, alles offen. Ein `undefined` an dieser Stelle würde die
    // Seite beim ersten Zugriff zerlegen.
    progress: group.progress ?? {
      agreedNet: group.totalNet ?? 0,
      billedNet: 0,
      receivedNet: 0,
      receivedTax: 0,
      receivedGross: 0,
      openNet: group.totalNet ?? 0,
      closed: Boolean(group.closed),
    },
  };
}

/**
 * Die Listen der steuerlichen Nebenpflichten (Welle 5c).
 *
 * Alle vier sind der Regelfall des unbelasteten Jahres: kein Wirtschaftsgut im
 * Verzeichnis, kein Nachweisbeleg, kein Geschenk, kein Posten in Fremdwährung.
 * Ein `null` aus Go an einer dieser Stellen reißt die Ansicht ausgerechnet dort
 * mit, wo nichts zu beanstanden ist.
 */
function normalizeInputTaxYear(year: InputTaxCorrectionYear): InputTaxCorrectionYear {
  if (!year) return year;
  return {
    ...year,
    rows: list(year.rows).map((row) => ({
      ...row,
      correction: row.correction
        ? { ...row.correction, usages: list(row.correction.usages) }
        : row.correction,
    })),
  };
}

function normalizeNonDeductibleReport(report: NonDeductibleReport): NonDeductibleReport {
  if (!report) return report;
  return {
    ...report,
    categories: list(report.categories),
    recipients: list(report.recipients).map((row) => ({
      ...row,
      toRebook: list(row.toRebook),
      bookings: list(row.bookings),
    })),
  };
}

function normalizeCurrencyValuation(
  valuation: ForeignCurrencyValuation,
): ForeignCurrencyValuation {
  if (!valuation) return valuation;
  return { ...valuation, items: list(valuation.items) };
}

function normalizeWriteUpReport(report: WriteUpReport): WriteUpReport {
  if (!report) return report;
  return {
    ...report,
    candidates: list(report.candidates).map((candidate) => ({
      ...candidate,
      impairments: list(candidate.impairments),
    })),
  };
}

/**
 * Die Listen der Buchungsvorschau.
 *
 * Der gute Fall ist hier der leere: eine Rechnung ohne Beanstandung hat keine
 * Warnung und keinen Befund. Käme dafür ein `null` aus Go, risse ausgerechnet
 * der gute Fall die Vorschau beim ersten `.length`.
 */
function normalizePostingPreview(preview: PostingPreview): PostingPreview {
  if (!preview) return preview;
  return {
    ...preview,
    lines: list(preview.lines),
    warnings: list(preview.warnings),
    inputTaxFindings: list(preview.inputTaxFindings),
  };
}

/** Altersstruktur und Restlaufzeiten: ohne offenen Posten sind beide leer. */
function normalizeOpenItemsAging(aging: OpenItemsAging): OpenItemsAging {
  if (!aging) return aging;
  return {
    ...aging,
    sides: list(aging.sides).map((side) => ({
      ...side,
      buckets: list(side.buckets),
      maturities: list(side.maturities),
    })),
  };
}

/** Die Eröffnungsbuchungen des Umsteigers samt ihren Hinweisen. */
function normalizeOpeningBalance(preview: OpeningBalancePreview): OpeningBalancePreview {
  if (!preview) return preview;
  return { ...preview, entries: list(preview.entries), messages: list(preview.messages) };
}

export const Api = {
  // --- Mandanten ---------------------------------------------------------

  getTenants: (): Promise<TenantConfig[]> => call(() => Bridge.GetTenants() as Promise<TenantConfig[]>),
  getActiveTenant: (): Promise<TenantConfig | null> =>
    call(() => Bridge.GetActiveTenant() as Promise<TenantConfig | null>),
  switchTenant: (tenantId: string): Promise<void> => call(() => Bridge.SwitchTenant(tenantId) as Promise<void>),
  deleteTenant: (tenantId: string): Promise<void> => call(() => Bridge.DeleteTenant(tenantId) as Promise<void>),

  // --- Einrichtung -------------------------------------------------------

  getAppConfig: (): Promise<AppConfig> => call(() => Bridge.GetAppConfig() as Promise<AppConfig>),
  validateTenantDirectory: (dataDir: string): Promise<void> =>
    call(() => Bridge.ValidateTenantDirectory(dataDir)),

  setupApplication: (dataDir: string, settings: CompanySettings): Promise<void> =>
    call(() => Bridge.SetupApplication(dataDir, settings as any) as Promise<void>),
  loadExistingDatabase: (dbFilePath: string): Promise<void> =>
    call(() => Bridge.LoadExistingDatabase(dbFilePath) as Promise<void>),
  selectDirectoryDialog: (title: string): Promise<string> =>
    call(() => Bridge.SelectDirectoryDialog(title) as Promise<string>),
  selectDatabaseFileDialog: (title: string): Promise<string> =>
    call(() => Bridge.SelectDatabaseFileDialog(title) as Promise<string>),

  // --- Geschäftsjahr & Stammdaten ---------------------------------------

  getFiscalYear: (): Promise<number> => call(() => Bridge.GetFiscalYear() as Promise<number>),
  setFiscalYear: (year: number): Promise<void> => call(() => Bridge.SetFiscalYear(year) as Promise<void>),
  getAvailableFiscalYears: (): Promise<number[]> =>
    call(() => Bridge.GetAvailableFiscalYears() as Promise<number[]>),
  getCompanySettings: (): Promise<CompanySettings> =>
    call(() => Bridge.GetCompanySettings() as Promise<CompanySettings>),
  updateCompanySettings: async (settings: CompanySettings): Promise<void> => {
    await call(() => Bridge.UpdateCompanySettings(settings as any) as Promise<void>);
    window.dispatchEvent(new Event(COMPANY_SETTINGS_CHANGED));
  },

  // --- Konten ------------------------------------------------------------

  getAccounts: (): Promise<Account[]> => call(() => Bridge.GetAccounts() as Promise<Account[]>),
  getAccountLedger: (accountNumber: string): Promise<AccountLedger> =>
    call(() => Bridge.GetAccountLedger(accountNumber) as Promise<AccountLedger>),
  getSuSaOverview: (): Promise<SuSaOverview> => call(() => Bridge.GetSuSaOverview() as Promise<SuSaOverview>),
  /** Summen- und Salden zu einem Stichtag. Leerer Stichtag heißt: ganzes Jahr. */
  getSuSaOverviewAt: (cutoff = ''): Promise<SuSaOverview> =>
    call(() => Bridge.GetSuSaOverviewAt(cutoff) as Promise<SuSaOverview>),
  getPaymentAccounts: (): Promise<Account[]> => call(() => Bridge.GetPaymentAccounts() as Promise<Account[]>),

  // --- Kontierung --------------------------------------------------------

  getPostingGroups: (direction: Direction | ''): Promise<PostingGroup[]> =>
    call(() => Bridge.GetPostingGroups(direction) as Promise<PostingGroup[]>),
  getTaxTreatments: (direction: Direction): Promise<TaxTreatmentInfo[]> =>
    call(() => Bridge.GetTaxTreatments(direction) as Promise<TaxTreatmentInfo[]>),
  getDifferenceKinds: (): Promise<DifferenceKindInfo[]> =>
    call(() => Bridge.GetDifferenceKinds() as Promise<DifferenceKindInfo[]>),

  // --- Journal -----------------------------------------------------------

  getJournalEntries: (): Promise<JournalEntry[]> =>
    call(() => Bridge.GetJournalEntries() as Promise<JournalEntry[]>).then(list),
  getAllJournalEntries: (): Promise<JournalEntry[]> =>
    call(() => Bridge.GetAllJournalEntries() as Promise<JournalEntry[]>).then(list),
  postIncomingReceipt: (request: ReceiptRequest): Promise<JournalEntry> =>
    call(() => Bridge.PostIncomingReceipt(request as any) as Promise<JournalEntry>),
  /** Der Buchungssatz, wie er gebucht würde. Das Frontend rechnet ihn nicht nach. */
  previewIncomingReceipt: (request: ReceiptRequest): Promise<PostingPreview> =>
    call(() => Bridge.PreviewIncomingReceipt(request as any) as Promise<PostingPreview>).then(
      normalizePostingPreview,
    ),
  previewOutgoingInvoice: (invoice: Partial<Invoice>): Promise<PostingPreview> =>
    call(() => Bridge.PreviewOutgoingInvoice(invoice as any) as Promise<PostingPreview>).then(
      normalizePostingPreview,
    ),

  // --- Belege ------------------------------------------------------------

  // Der Wails-Dateidialog liefert beim Abbruch null; ohne Normalisierung liefe jeder
  // Aufrufer in einen TypeError statt in den leeren Fall.
  selectReceiptFiles: (title = 'Belegdateien auswählen'): Promise<string[]> =>
    call(() => Bridge.SelectReceiptFilesDialog(title)).then(list),
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
    call(() => Bridge.GetReceipts(status) as Promise<Receipt[]>).then(list),
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
  /** Die Regeln, die Buchfink prüft — der Prüfumfang ist Teil des Ergebnisses. */
  getEInvoiceRules: (): Promise<string[]> => call(() => Bridge.GetEInvoiceRules()),
  /**
   * Schreibt eine Belegdatei unter ihrem Originalnamen an einen gewählten Ort.
   * Leerer Pfad heißt: der Dialog wurde abgebrochen.
   */
  saveReceiptFileAs: (receiptId: number, fileId: number): Promise<string> =>
    call(() => Bridge.SaveReceiptFileAs(receiptId, fileId)),
  /** Das archivierte Rechnungsdokument — dasselbe PDF, das der Kunde bekommen hat. */
  getInvoiceDocument: (invoiceId: number): Promise<ReceiptPreview> =>
    call(() => Bridge.GetInvoiceDocument(invoiceId) as Promise<ReceiptPreview>),
  reverseJournalEntry: (entryId: number, reason: string): Promise<JournalEntry> =>
    call(() => Bridge.ReverseJournalEntry(entryId, reason) as Promise<JournalEntry>),
  verifyIntegrity: (): Promise<IntegrityCheckResult> =>
    call(() => Bridge.VerifyIntegrity() as Promise<IntegrityCheckResult>).then(normalizeIntegrity),
  getFinancialSummary: (): Promise<FinancialSummary> =>
    call(() => Bridge.GetFinancialSummary() as Promise<FinancialSummary>),

  // --- Bank & Zahlungen --------------------------------------------------

  getBankAccounts: (): Promise<BankAccount[]> => call(() => Bridge.GetBankAccounts() as Promise<BankAccount[]>).then(list),
  previewBankStatement: (path: string): Promise<BankImportPreview> => call(() => Bridge.PreviewBankStatement(path) as Promise<BankImportPreview>),
  configureBankAccounts: (accounts: BankAccount[]): Promise<void> => call(() => Bridge.ConfigureBankAccounts(accounts)),
  setInvoiceBankAccount: (iban: string): Promise<void> => call(() => Bridge.SetInvoiceBankAccount(iban)),
  getBankTransactions: (): Promise<BankTransaction[]> =>
    call(() => Bridge.GetBankTransactions() as Promise<BankTransaction[]>).then(list),
  /**
   * Import über den Dateipfad. Nur so lässt sich die CAMT-Datei selbst als
   * Beleg archivieren — der Inhalt allein wäre nach dem Parsen verloren
   * (GoBD Rz. 130 f.). Den Import über den Inhalt gibt es hier bewusst nicht
   * mehr: er ließe den Weg offen, der den Kontoauszug nicht aufbewahrt
   * (ARC-03).
   */
  importCAMTFile: (path: string, ledgerAccount: string): Promise<number> =>
    call(() => Bridge.ImportCAMT053File(path, ledgerAccount) as Promise<number>),
  selectStatementFile: (title = 'Kontoauszug (CAMT.053) auswählen'): Promise<string> =>
    call(() => Bridge.SelectStatementFileDialog(title)),
  bookBankTransactionDirect: (
    bankTxId: number,
    counterAccount: string,
    description: string
  ): Promise<JournalEntry> =>
    call(() => Bridge.BookBankTransactionDirect(bankTxId, counterAccount, description) as Promise<JournalEntry>),
  ignoreBankTransaction: (bankTxId: number): Promise<void> =>
    call(() => Bridge.IgnoreBankTransaction(bankTxId) as Promise<void>),
  getOpenItems: (): Promise<OpenItem[]> => call(() => Bridge.GetOpenItems() as Promise<OpenItem[]>),
  /**
   * Die offenen Posten zu einem Stichtag: Zahlungen nach dem Stichtag zählen
   * nicht. Leerer Stichtag heißt heute.
   */
  getOpenItemsAt: (cutoff = ''): Promise<OpenItem[]> =>
    call(() => Bridge.GetOpenItemsAt(cutoff) as Promise<OpenItem[]>).then(list),
  /** Die Einzelposten einer Zahlungsbuchung — wogegen die Zahlung lief. */
  getPaymentAllocations: (entryId: number): Promise<PaymentAllocationDetail[]> =>
    call(() => Bridge.GetPaymentAllocations(entryId) as Promise<PaymentAllocationDetail[]>).then(list),
  settlePayment: (request: PaymentRequest): Promise<JournalEntry> =>
    call(() => Bridge.SettlePayment(request as any) as Promise<JournalEntry>),

  // --- Kontakte & Rechnungen --------------------------------------------

  getContacts: (): Promise<Contact[]> =>
    call(() => Bridge.GetContacts() as Promise<Contact[]>).then(list),
  saveContact: (contact: Partial<Contact>): Promise<Contact> =>
    call(() => Bridge.SaveContact(contact as any) as Promise<Contact>),
  getInvoices: (): Promise<Invoice[]> =>
    call(() => Bridge.GetInvoices() as Promise<Invoice[]>)
      .then(list)
      .then((rows) => rows.map(normalizeInvoice)),
  issueInvoice: (invoice: Partial<Invoice>): Promise<Invoice> =>
    call(() => Bridge.IssueInvoice(invoice as any) as Promise<Invoice>).then(normalizeInvoice),
  generateInvoiceZUGFeRD: (invoiceId: number): Promise<[string, string]> =>
    call(() => Bridge.GenerateInvoiceZUGFeRD(invoiceId) as Promise<[string, string]>),

  // --- Korrektur, Storno, Versand, Nummernkreis --------------------------

  /**
   * Holt ein fehlendes Rechnungsdokument nach. Der Fall: Nummer und Buchung
   * stehen, das Erzeugen des PDF ist gescheitert — ohne diesen Weg wäre die
   * Nummer verloren.
   */
  regenerateInvoiceDocument: (invoiceId: number): Promise<Invoice> =>
    call(() => Bridge.RegenerateInvoiceDocument(invoiceId) as Promise<Invoice>).then(
      normalizeInvoice,
    ),
  /** Storniert eine Rechnung und stellt die Stornorechnung aus; zurück kommt sie. */
  cancelInvoiceWithDocument: (invoiceId: number, reason: string): Promise<Invoice> =>
    call(() => Bridge.CancelInvoiceWithDocument(invoiceId, reason) as Promise<Invoice>).then(
      normalizeInvoice,
    ),
  /** Storniert und stellt die berichtigte Rechnung aus; zurück kommt die neue. */
  correctInvoice: (
    invoiceId: number,
    reason: string,
    replacement: Partial<Invoice>,
  ): Promise<Invoice> =>
    call(() => Bridge.CorrectInvoice(invoiceId, reason, replacement) as Promise<Invoice>).then(
      normalizeInvoice,
    ),
  /** Vermerkt, wann und wie die Rechnung hinausgegangen ist. */
  markInvoiceSent: (
    invoiceId: number,
    date: string,
    via: InvoiceSentVia,
    note = '',
  ): Promise<Invoice> =>
    call(() => Bridge.MarkInvoiceSent(invoiceId, date, via, note) as Promise<Invoice>).then(
      normalizeInvoice,
    ),
  /** Der Lückenbericht: Zählerstand gegen die vergebenen Nummern. */
  getInvoiceNumberGaps: (year = 0): Promise<NumberGapReport> =>
    call(() => Bridge.GetInvoiceNumberGaps(year) as Promise<NumberGapReport>).then((report) =>
      report ? { ...report, gaps: list(report.gaps) } : report,
    ),
  /** Dokumentiert, warum zu einer Nummer keine Rechnung gehört. */
  recordInvoiceNumberGapReason: (
    year: number,
    sequence: number,
    reason: NumberGapReason,
    detail = '',
  ): Promise<void> => call(() => Bridge.RecordInvoiceNumberGapReason(year, sequence, reason, detail)),
  /** Die Mengeneinheiten nach UN/ECE Rec. 20, die eine Position haben kann. */
  getUnitCodes: (): Promise<UnitCode[]> =>
    call(() => Bridge.GetUnitCodes() as Promise<UnitCode[]>).then(list),
  /** Die Zielformate, in denen eine Rechnung ausgestellt werden kann. */
  getEInvoiceProfiles: (): Promise<EInvoiceProfileInfo[]> =>
    call(() => Bridge.GetEInvoiceProfiles() as Promise<EInvoiceProfileInfo[]>).then(list),
  /**
   * Die Versandwege des Vermerks „Als versendet vermerken".
   *
   * Wie Einheiten und Profile aus dem Backend: die Beschriftungen stehen in
   * `domain.InvoiceSentViaOptions` und nicht ein zweites Mal in der Seite.
   */
  getInvoiceSentViaOptions: (): Promise<InvoiceSentViaOption[]> =>
    call(() => Bridge.GetInvoiceSentViaOptions() as Promise<InvoiceSentViaOption[]>).then(list),
  /** Die Gründe, mit denen eine Lücke im Nummernkreis begründet wird. */
  getNumberGapReasons: (): Promise<NumberGapReasonOption[]> =>
    call(() => Bridge.GetNumberGapReasons() as Promise<NumberGapReasonOption[]>).then(list),

  // --- Anzahlungen -------------------------------------------------------

  getInvoiceGroups: (): Promise<InvoiceGroup[]> =>
    call(() => Bridge.GetInvoiceGroups() as Promise<InvoiceGroup[]>)
      .then(list)
      .then((rows) => rows.map(normalizeInvoiceGroup)),
  createInvoiceGroup: (request: AdvanceGroupRequest): Promise<InvoiceGroup> =>
    call(() => Bridge.CreateInvoiceGroup(request) as Promise<InvoiceGroup>).then(
      normalizeInvoiceGroup,
    ),
  /** Die Abschlagsrechnung wird beim Ausstellen nicht gebucht — erst bei Zahlung. */
  issueAdvanceInvoice: (request: AdvanceInvoiceRequest): Promise<Invoice> =>
    call(() => Bridge.IssueAdvanceInvoice(request) as Promise<Invoice>).then(normalizeInvoice),
  /** Der Zahlungseingang auf einen Abschlag: hier entsteht die Steuer. */
  settleAdvance: (request: SettleAdvanceRequest): Promise<AdvanceItem> =>
    call(() => Bridge.SettleAdvance(request) as Promise<AdvanceItem>),
  /** Die Rückzahlung einer vereinnahmten Anzahlung (§ 17 Abs. 2 Nr. 2 UStG). */
  refundAdvance: (request: RefundAdvanceRequest): Promise<AdvanceItem> =>
    call(() => Bridge.RefundAdvance(request) as Promise<AdvanceItem>),
  /** Die Schlussrechnung setzt die vereinnahmten Anzahlungen ab (BT-113). */
  issueFinalInvoice: (request: FinalInvoiceRequest): Promise<Invoice> =>
    call(() => Bridge.IssueFinalInvoice(request) as Promise<Invoice>).then(normalizeInvoice),
  /** Die gestellten, noch nicht vereinnahmten Abschläge als offene Posten. */
  getOpenAdvances: (): Promise<OpenItem[]> =>
    call(() => Bridge.GetOpenAdvances() as Promise<OpenItem[]>).then(list),
  /** Die Verwendungen einer geleisteten Anzahlung mit ihrem Konto. */
  getAdvanceTargets: (): Promise<AdvanceTargetOption[]> =>
    call(() => Bridge.GetAdvanceTargets() as Promise<AdvanceTargetOption[]>).then(list),
  /** Die geleisteten Anzahlungen an einen Lieferanten; 0 heißt alle. */
  getOpenVendorAdvances: (contactId = 0): Promise<VendorAdvance[]> =>
    call(() => Bridge.GetOpenVendorAdvances(contactId) as Promise<VendorAdvance[]>).then(list),
  /** Bucht eine uneinbringliche Forderung aus; die Begründung ist Pflicht. */
  writeOffOpenItem: (request: WriteOffRequest): Promise<JournalEntry> =>
    call(() => Bridge.WriteOffOpenItem(request) as Promise<JournalEntry>),

  // --- Anlagevermögen ----------------------------------------------------

  getFixedAssets: (assetClass: AssetClass | '' = ''): Promise<FixedAsset[]> =>
    call(() => Bridge.GetFixedAssets(assetClass) as Promise<FixedAsset[]>),
  getFixedAsset: (id: number): Promise<AssetDetail> =>
    call(() => Bridge.GetFixedAsset(id) as Promise<AssetDetail>),
  saveFixedAsset: (asset: Partial<FixedAsset>): Promise<FixedAsset> =>
    call(() => Bridge.SaveFixedAsset(asset as any) as Promise<FixedAsset>),
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
  getAssetRules: (): Promise<AssetRules> =>
    call(() => Bridge.GetAssetRules() as Promise<AssetRules>).then((rules) =>
      rules
        ? {
            ...rules,
            degressiveWindows: list(rules.degressiveWindows),
            electricVehicleWindows: list(rules.electricVehicleWindows),
            methods: list(rules.methods),
          }
        : rules,
    ),
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
    call(() => Bridge.GetStatement(year, depth) as Promise<FinancialStatement>).then(
      normalizeStatement,
    ),
  getSizeClass: (year: number): Promise<SizeClass> =>
    call(() => Bridge.GetSizeClass(year) as Promise<SizeClass>),
  /** Bilanz und GuV als PDF, Base64 wie der Rechnungsexport. */
  exportStatementPDF: (year: number): Promise<string> =>
    call(() => Bridge.ExportStatementPDF(year) as Promise<string>),
  /** Dieselbe Gliederung als CSV-Text (UTF-8, Semikolon). */
  exportStatementCSV: (year: number): Promise<string> =>
    call(() => Bridge.ExportStatementCSV(year) as Promise<string>),
  /** Das dritte Merkmal des § 267 Abs. 1 HGB; aus Buchungen nicht ableitbar. */
  setAverageEmployees: (year: number, count: number): Promise<FiscalYear> =>
    call(() => Bridge.SetAverageEmployees(year, count) as Promise<FiscalYear>),
  /**
   * Der Gesamtumsatz des Vorjahres. An ihm hängt, ob 2027 noch eine sonstige
   * Rechnung ohne strukturierten Datensatz ausgestellt werden darf
   * (§ 27 Abs. 38 Nr. 2 UStG).
   */
  setPriorYearRevenue: (year: number, amount: Cents): Promise<FiscalYear> =>
    call(() => Bridge.SetPriorYearRevenue(year, amount) as Promise<FiscalYear>),

  // --- E-Bilanz, Audit & Festschreibung ---------------------------------

  exportEBilanzXBRL: (): Promise<string> => call(() => Bridge.ExportEBilanzXBRL() as Promise<string>),
  /**
   * Welches Konto unter welcher Gliederungsposition und welchem
   * Taxonomie-Element erscheint — und was die Erzeugung verhindert.
   */
  getEBilanzMappingReport: (year: number): Promise<MappingReport> =>
    call(() => Bridge.GetEBilanzMappingReport(year) as Promise<MappingReport>),
  getFestschreibungen: (): Promise<Festschreibung[]> =>
    call(() => Bridge.GetFestschreibungen() as Promise<Festschreibung[]>).then(list),
  /**
   * Schreibt einen Zeitraum fest. Der Prüflauf läuft im Backend davor; ein
   * blockierender Befund lässt sich nur mit Begründung übergehen, und die
   * Begründung steht danach am Prüflauf und im Protokoll.
   */
  commitPeriod: (
    periodType: string,
    periodLabel: string,
    cutoffDate: string,
    overrideReason = '',
  ): Promise<Festschreibung> =>
    call(
      () =>
        Bridge.CommitPeriod(
          periodType,
          periodLabel,
          cutoffDate,
          overrideReason,
        ) as Promise<Festschreibung>,
    ),
  verifyFestschreibung: (id: number): Promise<FestschreibungVerification> =>
    call(() => Bridge.VerifyFestschreibung(id) as Promise<FestschreibungVerification>),

  // --- Jahresabschluss ---------------------------------------------------

  /** Die Geschäftsjahre als Entitäten: Zeitraum, Rumpfjahr, Abschlussstand. */
  getFiscalYears: (): Promise<FiscalYear[]> => call(() => Bridge.GetFiscalYears() as Promise<FiscalYear[]>),
  getFiscalYearCandidates: (): Promise<FiscalYearCandidates> =>
    call(() => Bridge.GetFiscalYearCandidates() as Promise<FiscalYearCandidates>),
  /** Legt das Geschäftsjahr an und schaltet auf es um. */
  createFiscalYear: (year: number): Promise<void> => call(() => Bridge.CreateFiscalYear(year)),
  bookLegalReserve: (year: number): Promise<JournalEntry> => call(() => Bridge.BookLegalReserve(year) as Promise<JournalEntry>),
  getClosingState: (year: number): Promise<ClosingState> =>
    call(() => Bridge.GetClosingState(year) as Promise<ClosingState>),
  /**
   * Der Vortragsstand ins Zieljahr: je Konto Schlusssaldo des Vorjahres,
   * bereits vorgetragener Wert und Differenz. Bucht nichts.
   */
  getCarryForwardPreview: (toYear: number): Promise<CarryForwardPreview> =>
    call(() => Bridge.GetCarryForwardPreview(toYear) as Promise<CarryForwardPreview>).then(
      (preview) =>
        preview
          ? {
              ...preview,
              rows: list(preview.rows),
              accrualReleases: list(preview.accrualReleases),
            }
          : preview,
    ),
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

  // --- Abschlussbausteine ------------------------------------------------

  /**
   * Die elf Bausteine des Abschlusses mit ihrem Zustand. Er ist zur Hälfte
   * abgeleitet — eine gebuchte AfA ist erledigt —, zur Hälfte gespeichert:
   * ein übersprungener Schritt ist eine Aussage und kein Versehen.
   */
  getClosingSteps: (year: number): Promise<ClosingSteps> =>
    call(() => Bridge.GetClosingSteps(year) as Promise<ClosingSteps>).then((steps) =>
      steps ? { ...steps, steps: list(steps.steps) } : steps,
    ),
  /** Übergeht einen Baustein. Der Grund ist Pflicht und bleibt am Schritt. */
  skipClosingStep: (year: number, key: string, reason: string): Promise<ClosingSteps> =>
    call(() => Bridge.SkipClosingStep(year, key, reason) as Promise<ClosingSteps>).then((steps) =>
      steps ? { ...steps, steps: list(steps.steps) } : steps,
    ),
  /**
   * Nimmt das Überspringen zurück: der Baustein steht wieder offen. Möglich,
   * solange das Jahr weder festgeschrieben noch festgestellt ist; der Grund ist
   * Pflicht, weil er an die Stelle des Grundes tritt, mit dem übergangen wurde.
   */
  reopenClosingStep: (year: number, key: string, reason: string): Promise<ClosingSteps> =>
    call(() => Bridge.ReopenClosingStep(year, key, reason) as Promise<ClosingSteps>).then((steps) =>
      steps ? { ...steps, steps: list(steps.steps) } : steps,
    ),
  markClosingStepDone: (year: number, key: string): Promise<ClosingSteps> =>
    call(() => Bridge.MarkClosingStepDone(year, key) as Promise<ClosingSteps>).then((steps) =>
      steps ? { ...steps, steps: list(steps.steps) } : steps,
    ),

  /** Buchungen, deren Leistung über den Bilanzstichtag hinausreicht (§ 250 HGB). */
  proposeAccruals: (year: number): Promise<AccrualProposal> =>
    call(() => Bridge.ProposeAccruals(year) as Promise<AccrualProposal>).then((proposal) =>
      proposal ? { ...proposal, items: list(proposal.items) } : proposal,
    ),
  /** Rechnet den Posten samt Auflösungsplan, ohne ihn zu buchen. */
  previewAccrual: (request: AccrualRequest): Promise<AccrualPreview> =>
    call(() => Bridge.PreviewAccrual(request as any) as Promise<AccrualPreview>).then((preview) =>
      preview
        ? {
            ...preview,
            accrual: normalizeAccrual(preview.accrual),
            lines: list(preview.lines),
            releases: list(preview.releases),
            warnings: list(preview.warnings),
          }
        : preview,
    ),
  bookAccrual: (request: AccrualRequest): Promise<Accrual> =>
    call(() => Bridge.BookAccrual(request as any) as Promise<Accrual>).then(normalizeAccrual),
  getAccruals: (year: number): Promise<Accrual[]> =>
    call(() => Bridge.GetAccruals(year) as Promise<Accrual[]>).then((accruals) =>
      list(accruals).map(normalizeAccrual),
    ),
  /** Der Bestand aller Abgrenzungen zu einem Stichtag; leer heißt Bilanzstichtag. */
  getAccrualReport: (cutoff = ''): Promise<AccrualReport> =>
    call(() => Bridge.GetAccrualReport(cutoff) as Promise<AccrualReport>).then((report) =>
      report ? { ...report, rows: list(report.rows) } : report,
    ),

  getProvisions: (year: number): Promise<Provision[]> =>
    call(() => Bridge.GetProvisions(year) as Promise<Provision[]>).then((provisions) =>
      list(provisions).map(normalizeProvision),
    ),
  /** Rechnet Abzinsung und Buchungssatz, ohne zu buchen (§ 253 Abs. 2 HGB). */
  previewProvision: (request: ProvisionRequest): Promise<ProvisionPreview> =>
    call(() => Bridge.PreviewProvision(request as any) as Promise<ProvisionPreview>).then(
      (preview) =>
        preview
          ? {
              ...preview,
              provision: normalizeProvision(preview.provision),
              lines: list(preview.lines),
              findings: list(preview.findings),
            }
          : preview,
    ),
  bookProvisionFormation: (request: ProvisionRequest): Promise<Provision> =>
    call(() => Bridge.BookProvisionFormation(request as any) as Promise<Provision>).then(
      normalizeProvision,
    ),
  bookProvisionIncrease: (request: ProvisionRequest): Promise<Provision> =>
    call(() => Bridge.BookProvisionIncrease(request as any) as Promise<Provision>).then(
      normalizeProvision,
    ),
  /** Auflösung nur mit Grund: der Rückstellungsgrund muss entfallen sein. */
  bookProvisionRelease: (request: ProvisionChangeRequest): Promise<Provision> =>
    call(() => Bridge.BookProvisionRelease(request as any) as Promise<Provision>).then(
      normalizeProvision,
    ),
  bookProvisionConsumption: (request: ProvisionChangeRequest): Promise<Provision> =>
    call(() => Bridge.BookProvisionConsumption(request as any) as Promise<Provision>).then(
      normalizeProvision,
    ),
  bookProvisionUnwinding: (request: ProvisionChangeRequest): Promise<Provision> =>
    call(() => Bridge.BookProvisionUnwinding(request as any) as Promise<Provision>).then(
      normalizeProvision,
    ),
  /** Erledigt die Rückstellung; ein offener Rest wird mit Grund aufgelöst. */
  settleProvision: (provisionId: number, date: string, reason: string): Promise<Provision> =>
    call(() => Bridge.SettleProvision(provisionId, date, reason) as Promise<Provision>).then(
      normalizeProvision,
    ),
  /** Der Rückstellungsspiegel des Anhangs: er geht per Definition auf. */
  getProvisionMirror: (year: number): Promise<ProvisionMirror> =>
    call(() => Bridge.GetProvisionMirror(year) as Promise<ProvisionMirror>).then((mirror) =>
      mirror ? { ...mirror, rows: list(mirror.rows) } : mirror,
    ),
  /** Die Abzinsungssätze eines Monats; leerer Monat heißt: die jüngsten. */
  getDiscountRates: (month = ''): Promise<DiscountRate[]> =>
    call(() => Bridge.GetDiscountRates(month) as Promise<DiscountRate[]>).then(list),
  getDiscountRateMonths: (): Promise<string[]> =>
    call(() => Bridge.GetDiscountRateMonths()).then(list),
  saveDiscountRates: (rows: DiscountRate[]): Promise<void> =>
    call(() => Bridge.SaveDiscountRates(rows as any[])),
  /**
   * Liest die Veröffentlichung der Bundesbank: zwei Spalten, Restlaufzeit und
   * Satz. Der Monat gehört dazu — ohne ihn ließe sich der Satz keinem Stichtag
   * zuordnen. Sieben Jahre Mittelung sind die Rückstellungen, zehn die
   * Altersversorgung.
   */
  importDiscountRatesCSV: (path: string, month: string, average = 7): Promise<number> =>
    call(() => Bridge.ImportDiscountRatesCSV(path, month, average)),

  /** Vorsteuer, Umsatzsteuer und Vorauszahlungen zu einem Saldo verrechnet. */
  previewVatSettlement: (year: number): Promise<VatSettlement> =>
    call(() => Bridge.PreviewVatSettlement(year) as Promise<VatSettlement>).then((settlement) =>
      settlement
        ? { ...settlement, rows: list(settlement.rows), lines: list(settlement.lines) }
        : settlement,
    ),
  bookVatSettlement: (year: number): Promise<JournalEntry> =>
    call(() => Bridge.BookVatSettlement(year) as Promise<JournalEntry>),
  /** Körperschaftsteuer, Solidaritätszuschlag und Gewerbesteuer — eine Schätzung. */
  previewTaxProvision: (year: number): Promise<TaxProvisionPreview> =>
    call(() => Bridge.PreviewTaxProvision(year) as Promise<TaxProvisionPreview>).then((preview) =>
      preview ? { ...preview, lines: list(preview.lines) } : preview,
    ),
  bookTaxProvision: (request: TaxProvisionRequest): Promise<Provision[]> =>
    call(() => Bridge.BookTaxProvision(request as any) as Promise<Provision[]>).then((provisions) =>
      list(provisions).map(normalizeProvision),
    ),

  /**
   * Der Beschluss über die Ergebnisverwendung. Er gehört zum Jahr, dessen
   * Ergebnis verwendet wird, gebucht wird er im Folgejahr.
   */
  previewAppropriation: (
    year: number,
    request: AppropriationRequest,
  ): Promise<AppropriationPreview> =>
    call(
      () => Bridge.PreviewAppropriation(year, request as any) as Promise<AppropriationPreview>,
    ).then((preview) =>
      preview
        ? { ...preview, lines: list(preview.lines), warnings: list(preview.warnings) }
        : preview,
    ),
  bookAppropriation: (year: number, request: AppropriationRequest): Promise<Appropriation> =>
    call(() => Bridge.BookAppropriation(year, request as any) as Promise<Appropriation>),
  /** Null, solange kein Beschluss gefasst ist. */
  getAppropriation: (year: number): Promise<Appropriation | null> =>
    call(() => Bridge.GetAppropriation(year) as Promise<Appropriation | null>),

  /** Die Abschnitte des Anhangs, auch die leeren: der Anhang ist eine Gliederung. */
  getNotesTexts: (year: number): Promise<NotesSectionText[]> =>
    call(() => Bridge.GetNotesTexts(year) as Promise<NotesSectionText[]>).then(list),
  saveNotesText: (year: number, section: NotesSection, text: string): Promise<NotesSectionText[]> =>
    call(() => Bridge.SaveNotesText(year, section, text) as Promise<NotesSectionText[]>).then(list),

  /**
   * Hebesatz, Abgrenzungsmethode, Vorschlagsschwelle und Auflösungstakt. Ohne
   * sie rechnete jede Installation mit den Voreinstellungen weiter, als wären
   * sie gewählt worden.
   */
  getClosingSettings: (): Promise<ClosingSettings> =>
    call(() => Bridge.GetClosingSettings() as Promise<ClosingSettings>),
  /** Der Dienst prüft die Grenzen und protokolliert Vorher- und Nachherwert. */
  saveClosingSettings: (settings: ClosingSettings): Promise<ClosingSettings> =>
    call(() => Bridge.SaveClosingSettings(settings) as Promise<ClosingSettings>),

  /** Das Verzeichnis der steuerlichen Wahlrechte (§ 5 Abs. 1 Satz 2 EStG). */
  getTaxElectionRegister: (year: number): Promise<TaxElectionRegister> =>
    call(() => Bridge.GetTaxElectionRegister(year) as Promise<TaxElectionRegister>).then(
      (register) =>
        register
          ? {
              ...register,
              rows: list(register.rows).map((row) => ({ ...row, years: list(row.years) })),
            }
          : register,
    ),
  /** Dasselbe Verzeichnis als CSV-Text; wohin es gehört, entscheidet der Anwender. */
  exportTaxElectionRegisterCSV: (year: number): Promise<string> =>
    call(() => Bridge.ExportTaxElectionRegisterCSV(year)),
  /** Die Überleitung Handelsbilanz → Steuerbilanz (§ 60 Abs. 2 EStDV). */
  getReconciliation: (year: number): Promise<Reconciliation> =>
    call(() => Bridge.GetReconciliation(year) as Promise<Reconciliation>).then((reconciliation) =>
      reconciliation ? { ...reconciliation, rows: list(reconciliation.rows) } : reconciliation,
    ),

  // --- Umsatzsteuer-Voranmeldung ----------------------------------------

  /** Die Zeiträume eines Jahres mit Fälligkeit, Festschreibung und Stand. */
  getVatPeriods: (year: number): Promise<VatPeriodStatus[]> =>
    call(() => Bridge.GetVatPeriods(year) as Promise<VatPeriodStatus[]>).then(list),
  /**
   * Das Kennziffernblatt eines Zeitraums, neu gerechnet und nicht gespeichert.
   * Der Entwurf ist immer der heutige Stand des Journals.
   */
  getVatReturn: (periodKey: string): Promise<VatReturn> =>
    call(() => Bridge.GetVatReturn(periodKey) as Promise<VatReturn>).then(normalizeVatReturn),
  saveVatReturn: (periodKey: string): Promise<VatReturn> =>
    call(() => Bridge.SaveVatReturn(periodKey) as Promise<VatReturn>).then(normalizeVatReturn),
  getVatReturns: (year: number): Promise<VatReturn[]> =>
    call(() => Bridge.GetVatReturns(year) as Promise<VatReturn[]>).then((returns) =>
      list(returns).map(normalizeVatReturn),
    ),
  /**
   * Bestätigt die Übermittlung in Mein ELSTER. Der Zeitraum muss
   * festgeschrieben sein, und ohne Transferticket gibt es keine Bestätigung.
   */
  confirmVatReturnSubmitted: (
    id: number,
    date: string,
    ticket: string,
    note = '',
  ): Promise<VatReturn> =>
    call(() => Bridge.ConfirmVatReturnSubmitted(id, date, ticket, note) as Promise<VatReturn>).then(
      normalizeVatReturn,
    ),
  /** Legt eine berichtigte Voranmeldung an (Kennziffer 10 des Vordrucks). */
  createVatCorrection: (periodKey: string): Promise<VatReturn> =>
    call(() => Bridge.CreateVatCorrection(periodKey) as Promise<VatReturn>).then(normalizeVatReturn),
  /** Das Kennziffernblatt als Text zum Abtippen in Mein ELSTER. */
  exportVatReturnCSV: (id: number): Promise<string> =>
    call(() => Bridge.ExportVatReturnCSV(id)),
  /** Ein Elftel der Vorauszahlungen des Vorjahres — ein Vorschlag, kein Wert. */
  getSpecialPrepaymentSuggestion: (year: number): Promise<SpecialPrepaymentSuggestion> =>
    call(() =>
      Bridge.GetSpecialPrepaymentSuggestion(year) as Promise<SpecialPrepaymentSuggestion>,
    ).then((s) => (s ? { ...s, periods: list(s.periods) } : s)),

  // --- Zusammenfassende Meldung -----------------------------------------

  /** Die Meldezeiträume folgen den Umsätzen, nicht einer Einstellung. */
  getZMPeriods: (year: number): Promise<ZMPeriodStatus[]> =>
    call(() => Bridge.GetZMPeriods(year) as Promise<ZMPeriodStatus[]>).then(list),
  getZMReturn: (periodKey: string): Promise<ZMReturn> =>
    call(() => Bridge.GetZMReturn(periodKey) as Promise<ZMReturn>).then(normalizeZMReturn),
  saveZMReturn: (periodKey: string): Promise<ZMReturn> =>
    call(() => Bridge.SaveZMReturn(periodKey) as Promise<ZMReturn>).then(normalizeZMReturn),
  getZMReturns: (year: number): Promise<ZMReturn[]> =>
    call(() => Bridge.GetZMReturns(year) as Promise<ZMReturn[]>).then((returns) =>
      list(returns).map(normalizeZMReturn),
    ),
  confirmZMSubmitted: (id: number, date: string, ticket: string, note = ''): Promise<ZMReturn> =>
    call(() => Bridge.ConfirmZMSubmitted(id, date, ticket, note) as Promise<ZMReturn>).then(
      normalizeZMReturn,
    ),
  createZMCorrection: (periodKey: string): Promise<ZMReturn> =>
    call(() => Bridge.CreateZMCorrection(periodKey) as Promise<ZMReturn>).then(normalizeZMReturn),
  /** Die Meldedatei im Spaltenformat des BZSt-Online-Portals. */
  exportZMCSV: (id: number): Promise<string> => call(() => Bridge.ExportZMCSV(id)),

  // --- Prüfläufe und Fristen --------------------------------------------

  /**
   * Der Prüfbericht bis zu einem Stichtag als Vorschau — er wird nicht
   * gespeichert. Abgelegt wird der Lauf, den die Festschreibung selbst
   * ausführt; sonst stünden je Festschreibung zwei Läufe im Protokoll. Der
   * Zeitraumtyp schaltet die Regeln zu, die vor der Jahresfestschreibung gelten.
   */
  runChecks: (cutoffDate: string, periodType = ''): Promise<CheckRun> =>
    call(() => Bridge.RunChecks(cutoffDate, periodType) as Promise<CheckRun>).then(
      normalizeCheckRun,
    ),
  getCheckRuns: (year: number): Promise<CheckRun[]> =>
    call(() => Bridge.GetCheckRuns(year) as Promise<CheckRun[]>).then((runs) =>
      list(runs).map(normalizeCheckRun),
    ),
  /**
   * Alle Termine eines Jahres. „Erledigt" ergibt sich aus den Daten — der
   * übermittelten Voranmeldung, der Festschreibung —, nicht aus einem Haken.
   */
  getDeadlines: (year: number): Promise<Deadline[]> =>
    call(() => Bridge.GetDeadlines(year) as Promise<Deadline[]>).then(list),
  /** Der Haken für das, was Buchfink nicht sieht. Leeres Datum nimmt ihn zurück. */
  markDeadlineDone: (key: string, date: string): Promise<void> =>
    call(() => Bridge.MarkDeadlineDone(key, date)),

  // --- Gründung ---------------------------------------------------------

  /** Regeln, Gründung, Anmeldungsbefund und Unterbilanz in einem Aufruf. */
  getFoundationState: (): Promise<FoundationState> =>
    call(() => Bridge.GetFoundationState() as Promise<FoundationState>),
  saveFoundation: (foundation: Partial<Foundation>): Promise<Foundation> =>
    call(() => Bridge.SaveFoundation(foundation as any) as Promise<Foundation>),
  /** Die Kapitalaufbringungsregeln der Kapitalgesellschaften. */
  getFoundationRules: (): Promise<FoundationRules[]> =>
    call(() => Bridge.GetFoundationRules() as Promise<FoundationRules[]>),
  /** Die Länder, die für die Geschäftsanschrift wählbar sind. */
  getCountries: (): Promise<Country[]> => call(() => Bridge.GetCountries() as Promise<Country[]>).then(list),
  /**
   * Voranmeldungszeitraum einer Gründung in diesem Jahr, mit Begründung.
   * § 18 Abs. 2 UStG hat dafür ein Stichjahr — deshalb wird gefragt statt geraten.
   */
  getRecommendedVatPeriod: (foundingYear: number): Promise<{ period: string; reason: string }> =>
    call(
      () =>
        Bridge.GetRecommendedVatPeriod(foundingYear) as Promise<{ period: string; reason: string }>
    ),
  // --- Datenüberlassung nach § 147 Abs. 6 AO -----------------------------

  /** Zielordner für einen Export. Leerer Pfad heißt: abgebrochen. */
  selectExportDirectory: (title = 'Zielordner für den Export wählen'): Promise<string> =>
    call(() => Bridge.SelectExportDirectoryDialog(title)),
  /** Die Tabellen eines Geschäftsjahres (Z3): CSV, index.xml, Feldbeschreibung. */
  exportZ3: (year: number, targetDir: string): Promise<ExportResult> =>
    call(() => Bridge.ExportZ3(year, targetDir) as Promise<ExportResult>).then(normalizeExport),
  /** Dasselbe samt Belegdateien. */
  exportArchive: (year: number, targetDir: string): Promise<ExportResult> =>
    call(() => Bridge.ExportArchive(year, targetDir) as Promise<ExportResult>).then(normalizeExport),
  /** Das Prüferpaket: Archiv, Integritätsnachweis, Verfahrensdokumentation. */
  exportAuditPackage: (year: number, targetDir: string): Promise<ExportResult> =>
    call(() => Bridge.ExportAuditPackage(year, targetDir) as Promise<ExportResult>).then(
      normalizeExport,
    ),
  /** Das Journal eines Zeitraums als CSV. Leerer Pfad heißt: abgebrochen. */
  exportJournalCSV: (from = '', to = ''): Promise<string> =>
    call(() => Bridge.ExportJournalCSV(from, to)),

  // --- Prüfläufe über Kette und Dateien ---------------------------------

  /** Prüft jede Belegdatei gegen ihre Prüfsumme. */
  verifyReceiptFiles: (): Promise<FileCheckResult> =>
    call(() => Bridge.VerifyReceiptFiles() as Promise<FileCheckResult>).then(normalizeFileCheck),

  // --- Sicherung und Wiederherstellung -----------------------------------

  getBackupRuns: (): Promise<BackupRun[]> =>
    call(() => Bridge.GetBackupRuns() as Promise<BackupRun[]>).then(list),
  /** Setzt den Sicherungsordner und liefert die aktualisierte Konfiguration. */
  setBackupDir: (dir: string): Promise<AppConfig> =>
    call(() => Bridge.SetBackupDir(dir) as Promise<AppConfig>),
  createBackup: (): Promise<BackupRun> => call(() => Bridge.CreateBackup() as Promise<BackupRun>),
  /** Der Wiederherstellungstest: entpacken, prüfen, Temporärordner löschen. */
  verifyBackup: (zipPath: string): Promise<BackupRun> =>
    call(() => Bridge.VerifyBackup(zipPath) as Promise<BackupRun>),
  /** Entpackt eine Sicherung in einen leeren Ordner und meldet ihn als Mandanten an. */
  restoreFromBackup: (zipPath: string, targetDir: string): Promise<TenantConfig> =>
    call(() => Bridge.RestoreFromBackup(zipPath, targetDir) as Promise<TenantConfig>),
  selectBackupDir: (title = 'Ordner für die Sicherung wählen'): Promise<string> =>
    call(() => Bridge.SelectBackupDirDialog(title)),
  selectBackupFile: (title = 'Buchfink-Sicherung auswählen'): Promise<string> =>
    call(() => Bridge.SelectBackupFileDialog(title)),

  // --- Vorsteuerberichtigung nach § 15a UStG -----------------------------

  /** Das Verzeichnis mit Blick auf ein Geschäftsjahr. */
  getInputTaxCorrections: (year: number): Promise<InputTaxCorrectionYear> =>
    call(() => Bridge.GetInputTaxCorrections(year) as Promise<InputTaxCorrectionYear>).then(
      normalizeInputTaxYear,
    ),
  /** Nimmt ein Wirtschaftsgut ins Verzeichnis auf. */
  registerInputTaxCorrection: (request: RegisterInputTaxRequest): Promise<InputTaxCorrection> =>
    call(() => Bridge.RegisterInputTaxCorrection(request as any) as Promise<InputTaxCorrection>),
  /** Schließt einen Eintrag vorzeitig ab — Abgang, Entnahme, Fehleintrag. */
  closeInputTaxCorrection: (id: number, reason: string, date = ''): Promise<InputTaxCorrection> =>
    call(() => Bridge.CloseInputTaxCorrection(id, reason, date) as Promise<InputTaxCorrection>),
  /** Bestätigt oder ändert den Verwendungsanteil eines Jahres. */
  saveInputTaxUsage: (request: SaveInputTaxUsageRequest): Promise<InputTaxCorrectionYear> =>
    call(() => Bridge.SaveInputTaxUsage(request as any) as Promise<InputTaxCorrectionYear>).then(
      normalizeInputTaxYear,
    ),
  /** Bucht die Berichtigungen eines Geschäftsjahres; der Betrag geht in Kz 64. */
  bookInputTaxCorrection: (year: number): Promise<InputTaxCorrectionYear> =>
    call(() => Bridge.BookInputTaxCorrection(year) as Promise<InputTaxCorrectionYear>).then(
      normalizeInputTaxYear,
    ),

  // --- Bestätigung der USt-IdNr. (§ 18e UStG) ----------------------------

  /** Der Verlauf der Bestätigungsabfragen eines Kontakts. */
  getVatIDChecks: (contactId: number): Promise<VatIDCheck[]> =>
    call(() => Bridge.GetVatIDChecks(contactId) as Promise<VatIDCheck[]>).then(list),
  /** Der Stand der Bestätigung, ohne dass dafür gefragt würde. */
  getVatIDStatus: (contactId: number): Promise<VatIDStatus> =>
    call(() => Bridge.GetVatIDStatus(contactId) as Promise<VatIDStatus>),
  /** Die qualifizierte Bestätigungsanfrage. Ihr Ergebnis bleibt am Kontakt. */
  checkVatID: (contactId: number): Promise<VatIDCheck> =>
    call(() => Bridge.CheckVatID(contactId) as Promise<VatIDCheck>),

  // --- Nicht abziehbare Betriebsausgaben (§ 4 Abs. 5 EStG) ---------------

  /** Der Bericht eines Geschäftsjahres: je Kategorie und je Empfänger. */
  getNonDeductibleReport: (year: number): Promise<NonDeductibleReport> =>
    call(() => Bridge.GetNonDeductibleReport(year) as Promise<NonDeductibleReport>).then(
      normalizeNonDeductibleReport,
    ),
  /** Bucht die noch abziehbar stehenden Geschenke eines Empfängers um. */
  rebookGiftsForRecipient: (request: RebookGiftsRequest): Promise<GiftRebooking> =>
    call(() => Bridge.RebookGiftsForRecipient(request as any) as Promise<GiftRebooking>).then(
      (result) =>
        result
          ? { ...result, reversals: list(result.reversals), rebookings: list(result.rebookings) }
          : result,
    ),

  // --- Fremdwährung ------------------------------------------------------

  /** Der Kurs eines Tages. Fehlt er, wird er geholt und in der Historie abgelegt. */
  getExchangeRate: (currency: string, date: string): Promise<ExchangeRate> =>
    call(() => Bridge.GetExchangeRate(currency, date) as Promise<ExchangeRate>),
  /** Die Kurshistorie einer Währung. */
  getExchangeRates: (currency: string, from = '', to = ''): Promise<ExchangeRate[]> =>
    call(() => Bridge.GetExchangeRates(currency, from, to) as Promise<ExchangeRate[]>).then(list),
  /** Nimmt einen von Hand erfassten Kurs auf. Ohne Quelle ist er eine Behauptung. */
  saveExchangeRate: (rate: Partial<ExchangeRate>): Promise<ExchangeRate> =>
    call(() => Bridge.SaveExchangeRate(rate as any) as Promise<ExchangeRate>),
  /** Die Umsatzsteuer-Durchschnittskurse eines Zeitraums (§ 16 Abs. 6 UStG). */
  getVatExchangeRates: (from = '', to = ''): Promise<VatExchangeRate[]> =>
    call(() => Bridge.GetVatExchangeRates(from, to) as Promise<VatExchangeRate[]>).then(list),
  /** Liest die Durchschnittskurse aus einer CSV-Datei. */
  importVatExchangeRatesCSV: (path: string): Promise<VatRateImport> =>
    call(() => Bridge.ImportVatExchangeRatesCSV(path) as Promise<VatRateImport>).then((result) =>
      result ? { ...result, problems: list(result.problems) } : result,
    ),
  /** Die Stichtagsbewertung, ohne sie zu buchen (§ 256a HGB). */
  previewCurrencyValuation: (year: number): Promise<ForeignCurrencyValuation> =>
    call(() => Bridge.PreviewCurrencyValuation(year) as Promise<ForeignCurrencyValuation>).then(
      normalizeCurrencyValuation,
    ),
  /** Bucht die Bewertung und ihre Auflösung im Folgejahr. */
  bookCurrencyValuation: (year: number): Promise<ForeignCurrencyValuation> =>
    call(() => Bridge.BookCurrencyValuation(year) as Promise<ForeignCurrencyValuation>).then(
      normalizeCurrencyValuation,
    ),

  // --- Anlagen: Regelsätze, Wertaufholung, Sammelposten ------------------

  /** Die Abschreibungsregeln aus der Ressource, aus der auch gerechnet wird. */
  getAfaRules: (): Promise<AfaRules> =>
    call(() => Bridge.GetAfaRules() as Promise<AfaRules>).then((rules) =>
      rules
        ? {
            ...rules,
            parameterSets: list(rules.parameterSets),
            degressiveWindows: list(rules.degressiveWindows),
            electricVehicleWindows: list(rules.electricVehicleWindows),
            buildingRates: list(rules.buildingRates),
          }
        : rules,
    ),
  /** Die Anlagegüter, bei denen die Wertaufholung zu prüfen ist (§ 253 Abs. 5 HGB). */
  getWriteUpReport: (year: number): Promise<WriteUpReport> =>
    call(() => Bridge.GetWriteUpReport(year) as Promise<WriteUpReport>).then(normalizeWriteUpReport),
  /** Hält fest, dass der Grund einer außerplanmäßigen Abschreibung fortbesteht. */
  confirmImpairmentPersists: (assetId: number, year: number, note: string): Promise<WriteUpReport> =>
    call(
      () => Bridge.ConfirmImpairmentPersists(assetId, year, note) as Promise<WriteUpReport>,
    ).then(normalizeWriteUpReport),
  /** Die Einheitlichkeit des Wahlrechts nach § 6 Abs. 2a Satz 5 EStG. */
  getPoolConsistencyReport: (year: number): Promise<PoolConsistencyReport> =>
    call(() => Bridge.GetPoolConsistencyReport(year) as Promise<PoolConsistencyReport>).then(
      (report) =>
        report
          ? { ...report, pooled: list(report.pooled), immediate: list(report.immediate) }
          : report,
    ),
  /** Freistellungsbescheinigungen § 48b EStG, die ablaufen oder abgelaufen sind. */
  getExemptionCertificateWarnings: (today = ''): Promise<ExemptionCertificateWarning[]> =>
    call(
      () => Bridge.GetExemptionCertificateWarnings(today) as Promise<ExemptionCertificateWarning[]>,
    ).then(list),

  // --- Änderungsprotokoll, Versionen und Datenübernahme ------------------

  /** Rechnet die Kette des Änderungsprotokolls nach (UNV-03, QUE-04). */
  verifyAuditChain: (): Promise<AuditChainResult> =>
    call(() => Bridge.VerifyAuditChain() as Promise<AuditChainResult>).then(normalizeAuditChain),
  /**
   * Das Protokoll mit Vorher/Nachher, eingeschränkt durch den Filter.
   *
   * Neben `getAuditLogs` und nicht statt dessen: die Übersicht braucht die
   * jüngsten Einträge ohne Frage, die Suche einen Filter.
   */
  getAuditLogsFiltered: (limit = 200, filter: AuditFilter = {}): Promise<AuditLogEntry[]> =>
    call(() => Bridge.GetAuditLogsFiltered(limit, filter) as Promise<AuditLogEntry[]>).then(list),
  /** Die Versionshistorie des Programms als Markdown (UNV-06). */
  /** Die Versionshistorie als Zeilen, die neueste zuerst. */
  getChangeLog: (): Promise<ChangelogEntry[]> =>
    call(() => Bridge.GetChangeLog() as Promise<ChangelogEntry[]>).then(list),
  /** Öffnet die Fassungsübersicht des Projekts im Browser. */
  openReleasesPage: (): Promise<void> => call(() => Bridge.OpenReleasesPage()),
  /** Das Protokoll der Schemaänderungen. */
  getSchemaMigrations: (): Promise<SchemaMigration[]> =>
    call(() => Bridge.GetSchemaMigrations() as Promise<SchemaMigration[]>).then(list),
  /** Das Protokoll der Datenübernahmen mit ihren Zählungen (ARC-05). */
  getMigrationRecords: (): Promise<MigrationRecord[]> =>
    call(() => Bridge.GetMigrationRecords() as Promise<MigrationRecord[]>).then(list),
  /**
   * Hält den Umstellungszeitpunkt der Übernahme aus einem Altsystem fest.
   * An ihm hängt die Fünfjahresfrist des § 147 Abs. 6 Satz 6 AO.
   */
  setSystemChangeDate: (date: string): Promise<void> =>
    call(() => Bridge.SetSystemChangeDate(date)),

  // --- Verfahrensdokumentation (PRF-03) ----------------------------------

  /** Erzeugt eine neue Fassung und legt sie im Belegspeicher ab. */
  generateProcedureDocumentation: (): Promise<ProcDocResult> =>
    call(() => Bridge.GenerateProcedureDocumentation() as Promise<ProcDocResult>),
  /** Die abgelegten Fassungen, neueste zuerst. */
  getProcedureDocumentations: (): Promise<ProcedureDocumentation[]> =>
    call(() => Bridge.GetProcedureDocumentations() as Promise<ProcedureDocumentation[]>).then(list),
  /**
   * Schreibt eine abgelegte Fassung an einen gewählten Ort; leer heißt: der
   * Dialog wurde abgebrochen. Die Herausgabe steht danach im Protokoll.
   */
  saveProcedureDocumentationAs: (id: number, wantPdf: boolean): Promise<string> =>
    call(() => Bridge.SaveProcedureDocumentationAs(id, wantPdf)),
  /** Die Freitexte der Organisationsanweisung, mit Mustern wo nichts erfasst ist. */
  getOrganisationTexts: (): Promise<OrganisationTexts> =>
    call(() => Bridge.GetOrganisationTexts() as Promise<OrganisationTexts>),
  /**
   * Die Muster selbst. Daran erkennt die Ansicht, welcher Abschnitt beschrieben
   * und welcher nur vorbelegt ist — die Texte allein sagen es nicht.
   */
  getOrganisationTextDefaults: (): Promise<OrganisationTexts> =>
    call(() => Bridge.GetOrganisationTextDefaults() as Promise<OrganisationTexts>),
  saveOrganisationTexts: (texts: OrganisationTexts): Promise<void> =>
    call(() => Bridge.SaveOrganisationTexts(texts as any)),
  /** Die Hinweise zu Rechtsform, Speicherort und Steuerfällen (BEW-13, UST-08). */
  getComplianceHints: (): Promise<ComplianceHints> =>
    call(() => Bridge.GetComplianceHints() as Promise<ComplianceHints>).then((hints) =>
      hints ? { ...hints, taxCaseHints: list(hints.taxCaseHints) } : hints,
    ),

  // --- Kontakte, Belege, Journal und offene Posten der Welle 6 -----------

  /**
   * Sperrt einen Geschäftspartner nach einem Löschverlangen und liefert die
   * Antwort an die betroffene Person gleich mit.
   */
  blockContact: (id: number, reason: string): Promise<BlockContactResult> =>
    call(() => Bridge.BlockContact(id, reason) as Promise<BlockContactResult>),
  /** Die Geschäftspartner, die sich noch auswählen lassen — gesperrte fehlen. */
  getSelectableContacts: (): Promise<Contact[]> =>
    call(() => Bridge.GetSelectableContacts() as Promise<Contact[]>).then(list),
  /** Schreibt die Kopfdaten eines abgelegten Belegs nach (BEL-02). */
  saveReceiptHeader: (receiptId: number, header: ReceiptHeader): Promise<Receipt> =>
    call(() => Bridge.SaveReceiptHeader(receiptId, header as any) as Promise<Receipt>),
  /** Storniert eine Buchung und bucht sie richtig neu, verknüpft (BEL-09). */
  correctEntry: (
    entryId: number,
    reason: string,
    newEntry: Partial<JournalEntry>,
  ): Promise<CorrectionResult> =>
    call(() => Bridge.CorrectEntry(entryId, reason, newEntry as any) as Promise<CorrectionResult>),
  /** Die Eröffnungsbuchungen des Umsteigers, ohne sie zu schreiben. */
  previewOpeningBalance: (request: OpeningBalanceRequest): Promise<OpeningBalancePreview> =>
    call(
      () => Bridge.PreviewOpeningBalance(request as any) as Promise<OpeningBalancePreview>,
    ).then(normalizeOpeningBalance),
  /** Bucht die Eröffnungsbilanz — alles oder nichts. */
  bookOpeningBalance: (request: OpeningBalanceRequest): Promise<OpeningBalancePreview> =>
    call(() => Bridge.BookOpeningBalance(request as any) as Promise<OpeningBalancePreview>).then(
      normalizeOpeningBalance,
    ),
  /** Altersstruktur und Restlaufzeiten zum Stichtag; leer heißt heute (BEL-07). */
  getOpenItemsAging: (cutoff = ''): Promise<OpenItemsAging> =>
    call(() => Bridge.GetOpenItemsAging(cutoff) as Promise<OpenItemsAging>).then(
      normalizeOpenItemsAging,
    ),

  // --- Prüfermodus -------------------------------------------------------

  /** Schaltet den Prüfermodus bis zu einem Tag ein. Datum und Grund sind Pflicht. */
  enableReadOnly: (until: string, reason: string): Promise<AppConfig> =>
    call(() => Bridge.EnableReadOnly(until, reason) as Promise<AppConfig>),
  /** Beendet ihn. Der Grund steht danach im Änderungsprotokoll. */
  disableReadOnly: (reason: string): Promise<AppConfig> =>
    call(() => Bridge.DisableReadOnly(reason) as Promise<AppConfig>),

  // --- Verbundene Apps ---------------------------------------------------

  /** Läuft der MCP-Server, auf welchem Port, und wenn nicht: warum. */
  getMCPServerStatus: (): Promise<MCPServerStatus> =>
    call(() => Bridge.GetMCPServerStatus() as Promise<MCPServerStatus>),
  /** Neuer Port; 0 heißt Vorgabe. Ein belegter Port wird gemeldet, nicht umgangen. */
  setMCPPort: (port: number): Promise<MCPServerStatus> =>
    call(() => Bridge.SetMCPPort(port) as Promise<MCPServerStatus>),
  /** Die Apps, denen je zugestimmt wurde, mit Mandanten und letztem Zugriff. */
  getConnectedApps: (): Promise<ConnectedApp[]> =>
    call(() => Bridge.GetConnectedApps() as Promise<ConnectedApp[]>).then((apps) =>
      list(apps).map((app) => ({
        ...app,
        tenants: list(app.tenants).map((t) => ({ ...t, scopes: list(t.scopes) })),
      })),
    ),
  /** Entzieht einer App jeden Zugriff; ihr nächster Aufruf scheitert. */
  revokeConnectedApp: (clientId: string): Promise<void> =>
    call(() => Bridge.RevokeConnectedApp(clientId)),
  /** Offene Zustimmungsanfragen, falls ein Ereignis verpasst wurde. */
  getPendingAppRequests: (): Promise<AppConsentRequest[]> =>
    call(() => Bridge.GetPendingAppRequests() as Promise<AppConsentRequest[]>).then((requests) =>
      list(requests).map((r) => ({ ...r, scopes: list(r.scopes) })),
    ),
  /** Entscheidet eine Anfrage; die Nonce stammt aus der Anfrage und gilt einmal. */
  resolveAppRequest: (id: string, nonce: string, approve: boolean): Promise<void> =>
    call(() => Bridge.ResolveAppRequest(id, nonce, approve)),
  /** Öffnet die Anleitung zu einer KI-App im Browser; ohne Angabe die Übersicht. */
  openAgentGuide: (client: AgentClient | '' = ''): Promise<void> =>
    call(() => Bridge.OpenAgentGuide(client)),

  // --- Stammdaten: Gesellschafterdarlehen, Steuernummer -------------------

  /** Hält die Eintragung ins Handelsregister fest; damit endet die Vorgesellschaft. */
  registerCompany: (date: string, court: string, number: string): Promise<Foundation> =>
    call(() => Bridge.RegisterCompany(date, court, number) as Promise<Foundation>),
  /** Die Darlehensverträge mit Gesellschaftern und ihrem gebuchten Stand von heute. */
  getShareholderLoans: (): Promise<ShareholderLoanView[]> =>
    call(() => Bridge.GetShareholderLoans() as Promise<ShareholderLoanView[]>).then((loans) =>
      list(loans).map((view) => ({ ...view, findings: list(view.findings) })),
    ),
  /** Erfasst einen Darlehensvertrag; nur in der App, nie über eine KI-App. */
  registerShareholderLoan: (request: ShareholderLoanRequest): Promise<ShareholderLoanView> =>
    call(() => Bridge.RegisterShareholderLoan(request) as Promise<ShareholderLoanView>),
  /** Berichtigt einen Darlehensvertrag; der Grund steht im Änderungsprotokoll. */
  correctShareholderLoan: (
    id: number,
    request: ShareholderLoanRequest,
    reason: string,
  ): Promise<ShareholderLoanView> =>
    call(() => Bridge.CorrectShareholderLoan(id, request, reason) as Promise<ShareholderLoanView>),
  /** Die Steuernummer im 13-stelligen Bundesschema; leer, solange sie unvollständig ist. */
  formatElsterTaxNumber: (taxNumber: string, federalState: string): Promise<string> =>
    call(() => Bridge.FormatElsterTaxNumber(taxNumber, federalState)),

  // --- Die Bedienung: Aufgaben, Monatsabschluss, Prüfpfad ----------------

  /**
   * Die Aufgabenliste in ihren drei Gruppen (Architektur 6.1).
   *
   * Der Regelfall einer aufgeräumten Buchführung ist die leere Gruppe, und
   * genau dann käme aus Go ein `null` an. Die Listen werden deshalb hier
   * gesetzt und nicht in der Ansicht — sonst trüge jede Gruppe ihre eigene
   * Absicherung.
   */
  getTasks: (): Promise<TaskList> =>
    call(() => Bridge.GetTasks() as Promise<TaskList>).then((tasks) =>
      tasks
        ? {
            ...tasks,
            overdue: list(tasks.overdue).map(normalizeTask),
            open: list(tasks.open).map(normalizeTask),
            upcoming: list(tasks.upcoming).map(normalizeTask),
          }
        : tasks,
    ),
  /** Der Stand eines Monats („JJJJ-MM") in seinen drei Schritten (6.2). */
  getMonthCloseState: (month: string): Promise<MonthCloseState> =>
    call(() => Bridge.GetMonthCloseState(month) as Promise<MonthCloseState>).then((state) =>
      state ? { ...state, steps: list(state.steps), findings: list(state.findings) } : state,
    ),
  /** Die wahrscheinlichsten Zuordnungen zu einem Bankumsatz. Gebucht wird nichts. */
  suggestBankMatches: (bankTxId: number): Promise<BankSuggestions> =>
    call(() => Bridge.SuggestBankMatches(bankTxId) as Promise<BankSuggestions>).then(
      (suggestions) =>
        suggestions
          ? {
              ...suggestions,
              suggestions: list(suggestions.suggestions).map((s) => ({
                ...s,
                reasons: list(s.reasons),
                items: list(s.items),
              })),
            }
          : suggestions,
    ),
  /** Die gelernten Zuordnungen wiederkehrender Umsätze. */
  getBankRules: (): Promise<BankRule[]> =>
    call(() => Bridge.GetBankRules() as Promise<BankRule[]>).then(list),
  /** Entfernt eine gelernte Zuordnung; der nächste Umsatz wird wieder gefragt. */
  deleteBankRule: (id: number): Promise<void> => call(() => Bridge.DeleteBankRule(id)),
  /** Der Prüfpfad eines Belegs zur Anzeige: Beleg, Buchung, Zahlung, Bankumsatz. */
  getAuditTrail: (receiptId: number): Promise<AuditTrail> =>
    call(() => Bridge.GetAuditTrail(receiptId) as Promise<AuditTrail>).then((trail) =>
      trail ? { ...trail, steps: list(trail.steps) } : trail,
    ),
  /** Schreibt denselben Prüfpfad in eine Datei; leerer Pfad heißt: abgebrochen. */
  exportAuditTrail: (receiptId: number, format: 'csv' | 'pdf' = 'pdf'): Promise<string> =>
    call(() => Bridge.ExportAuditTrail(receiptId, format)),
  /** Schreibt den Leistungsnachweis an einen Eingangsbeleg (RECH-08). */
  saveServiceProof: (receiptId: number, text: string, date: string): Promise<Receipt> =>
    call(() => Bridge.SaveServiceProof(receiptId, text, date) as Promise<Receipt>),
  /**
   * Der Hinweis zu einem langen Zahlungsziel; leer heißt: unauffällig.
   *
   * Er kommt aus dem Fachbereich und wird hier nicht nachgebaut: eine Grenze,
   * die in der Maske ein zweites Mal stünde, wird bei der nächsten Änderung an
   * einer der beiden Stellen vergessen.
   */
  getPaymentTermNotice: (dueDays: number): Promise<string> =>
    call(() => Bridge.GetPaymentTermNotice(dueDays)),

  // --- Welle 8: keine Buchung ohne Beleg ---------------------------------

  /**
   * Bucht einen von Hand erfassten Buchungssatz mit seinem Beleg (BEL-01).
   *
   * Entweder `receiptId` oder `selfIssued`: gibt es keinen fremden Beleg,
   * entsteht der Eigenbeleg im selben Vorgang. Welche der beiden Angaben fehlt
   * oder doppelt ist, sagt das Backend — die Maske baut die Regel nicht nach.
   */
  postManualEntry: (request: ManualEntryRequest): Promise<JournalEntry> =>
    call(() => Bridge.PostManualEntry(request as any) as Promise<JournalEntry>),
  /** Erzeugt einen Eigenbeleg mit PDF und Kopfdaten und legt ihn ab. */
  createSelfIssuedReceipt: (request: SelfIssuedReceiptRequest): Promise<Receipt> =>
    call(() => Bridge.CreateSelfIssuedReceipt(request as any) as Promise<Receipt>),
  /** Verlängert die Aufbewahrungsfrist eines Belegs; nur nach oben (ARC-01). */
  overrideReceiptRetention: (
    receiptId: number,
    retentionClass: RetentionClass,
    reason: string,
  ): Promise<Receipt> =>
    call(
      () => Bridge.OverrideReceiptRetention(receiptId, retentionClass, reason) as Promise<Receipt>,
    ),
  /** Die Fristentabelle mit Quelle und Rechtsstand. Anzeige, keine Einstellung. */
  getRetentionRules: (): Promise<RetentionRules> =>
    call(() => Bridge.GetRetentionRules() as Promise<RetentionRules>).then((rules) =>
      rules
        ? {
            ...rules,
            classes: list(rules.classes).map((entry) => ({ ...entry, kinds: list(entry.kinds) })),
          }
        : rules,
    ),
  /** Die Buchungen zu einem Beleg — der Weg von der Ablage ins Journal (GOB-02). */
  getEntriesForReceipt: (receiptId: number): Promise<JournalEntry[]> =>
    call(() => Bridge.GetEntriesForReceipt(receiptId) as Promise<JournalEntry[]>).then((entries) =>
      list(entries).map((entry) => ({ ...entry, lines: list(entry.lines) })),
    ),
  /** Die Beanstandungsliste eines Belegs, nach Fehlerklassen getrennt (RECH-02). */
  getReceiptFindings: (receiptId: number): Promise<ReceiptFindings> =>
    call(() => Bridge.GetReceiptFindings(receiptId) as Promise<ReceiptFindings>).then((findings) =>
      findings
        ? {
            ...findings,
            groups: list(findings.groups).map((group) => ({
              ...group,
              findings: list(group.findings),
            })),
          }
        : findings,
    ),
  /** Legt ein eigenes Konto im freien Bereich des SKR04 an (BEL-06). */
  createCustomAccount: (request: CustomAccountRequest): Promise<Account> =>
    call(() => Bridge.CreateCustomAccount(request as any) as Promise<Account>),
  /**
   * Sperrt ein eigenes Konto für neue Buchungen oder gibt es wieder frei.
   * Gesperrt statt gelöscht: die Buchungen zeigen auf die Nummer.
   */
  setAccountBlocked: (number: string, blocked: boolean, reason: string): Promise<Account> =>
    call(() => Bridge.SetAccountBlocked(number, blocked, reason) as Promise<Account>),
  /** Die selbst angelegten Konten. */
  getCustomAccounts: (): Promise<Account[]> =>
    call(() => Bridge.GetCustomAccounts() as Promise<Account[]>).then(list),
  /** Die Gliederungspositionen, unter denen ein eigenes Konto stehen darf. */
  getStatementPositions: (): Promise<StatementPositionOption[]> =>
    call(() => Bridge.GetStatementPositions() as Promise<StatementPositionOption[]>).then(list),
  /** Der Voranmeldungszeitraum, der sich aus der Steuer des Vorjahres ergibt. */
  getVatPeriodProposal: (year: number): Promise<VatPeriodProposal | null> =>
    call(() => Bridge.GetVatPeriodProposal(year) as Promise<VatPeriodProposal>),
  /** Die gefilterte Menge der Journalzeilen mit ihrer Summenzeile (PRF-01). */
  getFilteredJournal: (filter: JournalFilter): Promise<JournalFilterResult> =>
    call(() => Bridge.GetFilteredJournal(filter as any) as Promise<JournalFilterResult>).then(
      (result) => (result ? { ...result, rows: list(result.rows) } : result),
    ),
  /** Schreibt dieselbe Menge in eine gewählte Datei; leerer Pfad heißt: abgebrochen. */
  saveFilteredJournalCSV: (filter: JournalFilter, path: string): Promise<string> =>
    call(() => Bridge.SaveFilteredJournalCSV(filter as any, path)),
  /**
   * Fragt den Zielpfad einer Ausgabe ab; leerer Pfad heißt: abgebrochen.
   *
   * Der Dienst schreibt erst danach: ein Dienst, der den Ort selbst wählte,
   * schriebe irgendwohin. Deshalb hier der Dialog und dort das Schreiben.
   */
  selectSaveFilePath: (title: string, suggestedName: string): Promise<string> =>
    call(() => Bridge.SelectSaveFileDialog(title, suggestedName)),
};
