import { bridge as Bridge } from './bridge';
import type {
  Account,
  AccountLedger,
  Accrual,
  AcquisitionCandidate,
  AdvanceItem,
  AgentBookings,
  AgentClient,
  AgentRules,
  AgentSettings,
  Anlagenspiegel,
  AppConfig,
  AppConsentRequest,
  Appropriation,
  AppropriationPreview,
  AppropriationRequest,
  AssetClass,
  AssetDetail,
  AuditChainResult,
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
  ChangelogEntry,
  CheckRun,
  ClosingSettings,
  ClosingState,
  ClosingSteps,
  CompanySettings,
  ComplianceHints,
  ConnectedApp,
  Contact,
  Deadline,
  DeclarationExport,
  DeclarationInput,
  DeclarationInputRequest,
  DeclarationKind,
  DeclarationRecord,
  DeclarationSubmissionRequest,
  DeclarationView,
  Direction,
  EInvoiceProfileInfo,
  EInvoiceProposal,
  ExportResult,
  Festschreibung,
  FestschreibungVerification,
  FileCheckResult,
  FinancialStatement,
  FiscalYear,
  FiscalYearCandidates,
  FiscalYearStatus,
  FixedAsset,
  FolderCheck,
  ForeignCurrencyValuation,
  Foundation,
  FoundationRules,
  FoundationState,
  InboxAdded,
  InboxFile,
  InboxInfo,
  InputTaxCorrectionYear,
  IntegrityCheckResult,
  Invoice,
  JournalEntry,
  JournalFilter,
  JournalFilterResult,
  MappingReport,
  MCPServerStatus,
  MigrationRecord,
  MonthCloseState,
  NonDeductibleReport,
  NotesSectionText,
  NumberGapReason,
  NumberGapReasonOption,
  NumberGapReport,
  Obligation,
  ObligationConfirmation,
  OpenItem,
  OrganisationTexts,
  PaymentAllocationDetail,
  PaymentRequest,
  PostingGroup,
  PostingPreview,
  ProcDocResult,
  ProcedureDocumentation,
  Provision,
  ProvisionMirror,
  Receipt,
  ReceiptFileInput,
  ReceiptPreview,
  ReceiptRequest,
  ReceiptStatus,
  Reconciliation,
  RetentionRules,
  SchemaMigration,
  SettleAdvanceRequest,
  ShareholderLoanRequest,
  ShareholderLoanView,
  SizeClass,
  StatementDepth,
  SuSaOverview,
  TaxElectionRegister,
  TaxProvisionPreview,
  Task,
  TaskList,
  TaxTreatmentInfo,
  TenantConfig,
  UnitCode,
  VatIDStatus,
  VatPeriodProposal,
  VatReturn,
  VatSettlement,
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

/** Die Hinweise einer Pflicht und ihr Ziel kommen auch leer als Liste und Ziel an. */
function normalizeObligation(o: Obligation): Obligation {
  if (!o) return o;
  return { ...o, hints: list(o.hints), target: o.target ?? { area: '' } };
}

/** Die Listen einer Jahreserklärung bis hinunter zur Herkunft einer Zeile. */
function normalizeDeclaration(view: DeclarationView): DeclarationView {
  if (!view) return view;
  const d = view.declaration;
  return {
    overview: {
      ...view.overview,
      ergebnis: list(view.overview.ergebnis),
      formulare: list(view.overview.formulare),
      zu_klaeren: list(view.overview.zu_klaeren),
    },
    declaration: {
      ...d,
      formulare: list(d.formulare).map((form) => ({
        ...form,
        abschnitte: list(form.abschnitte).map((section) => ({
          ...section,
          zeilen: list(section.zeilen),
        })),
      })),
      abstimmung: list(d.abstimmung),
      eingaben: list(d.eingaben),
      befunde: list(d.befunde),
    },
  };
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

/** Ohne übergangene Datei kommt die Liste auch leer als Liste an. */
export function normalizeInboxAdded(added: InboxAdded): InboxAdded {
  return { ...added, names: list(added?.names), skipped: list(added?.skipped) };
}

/** Die Leseordner kommen auch ohne Eintrag als Liste an. */
function normalizeAgentRules(rules: AgentRules): AgentRules {
  return {
    ...rules,
    stored: { ...rules.stored, extraReadFolders: list(rules.stored?.extraReadFolders) },
    extraReadFolders: list(rules.extraReadFolders),
  };
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

  /** Legt den Mandanten an; ein leerer Eingang heißt `<Datenordner>/Eingang`. */
  setupApplication: (dataDir: string, inboxDir: string, settings: CompanySettings): Promise<void> =>
    call(() => Bridge.SetupApplication(dataDir, inboxDir, settings as any) as Promise<void>),
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
  getPaymentAccounts: (): Promise<Account[]> => call(() => Bridge.GetPaymentAccounts() as Promise<Account[]>),

  // --- Kontierung --------------------------------------------------------

  getPostingGroups: (direction: Direction | ''): Promise<PostingGroup[]> =>
    call(() => Bridge.GetPostingGroups(direction) as Promise<PostingGroup[]>),
  getTaxTreatments: (direction: Direction): Promise<TaxTreatmentInfo[]> =>
    call(() => Bridge.GetTaxTreatments(direction) as Promise<TaxTreatmentInfo[]>),

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
  getReceipts: (status: ReceiptStatus | '' = ''): Promise<Receipt[]> =>
    call(() => Bridge.GetReceipts(status) as Promise<Receipt[]>).then(list),
  getReceiptPreview: (receiptId: number): Promise<ReceiptPreview> =>
    call(() => Bridge.GetReceiptPreview(receiptId) as Promise<ReceiptPreview>),
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
  verifyIntegrity: (): Promise<IntegrityCheckResult> =>
    call(() => Bridge.VerifyIntegrity() as Promise<IntegrityCheckResult>).then(normalizeIntegrity),

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
  /**
   * Bestätigt die Standard-Buchungsgruppe eines Lieferanten; leer hebt sie
   * auf. Nur in der App: erst mit ihr bucht ein Agent ohne Rückfrage.
   */
  confirmDefaultPostingGroup: (contactId: number, group: string): Promise<Contact> =>
    call(() => Bridge.ConfirmDefaultPostingGroup(contactId, group) as Promise<Contact>),
  getInvoices: (): Promise<Invoice[]> =>
    call(() => Bridge.GetInvoices() as Promise<Invoice[]>)
      .then(list)
      .then((rows) => rows.map(normalizeInvoice)),
  issueInvoice: (invoice: Partial<Invoice>): Promise<Invoice> =>
    call(() => Bridge.IssueInvoice(invoice as any) as Promise<Invoice>).then(normalizeInvoice),

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
  /** Die Gründe, mit denen eine Lücke im Nummernkreis begründet wird. */
  getNumberGapReasons: (): Promise<NumberGapReasonOption[]> =>
    call(() => Bridge.GetNumberGapReasons() as Promise<NumberGapReasonOption[]>).then(list),

  // --- Anzahlungen -------------------------------------------------------

  /** Der Zahlungseingang auf einen Abschlag: hier entsteht die Steuer. */
  settleAdvance: (request: SettleAdvanceRequest): Promise<AdvanceItem> =>
    call(() => Bridge.SettleAdvance(request) as Promise<AdvanceItem>),

  // --- Anlagevermögen ----------------------------------------------------

  getFixedAssets: (assetClass: AssetClass | '' = ''): Promise<FixedAsset[]> =>
    call(() => Bridge.GetFixedAssets(assetClass) as Promise<FixedAsset[]>),
  getFixedAsset: (id: number): Promise<AssetDetail> =>
    call(() => Bridge.GetFixedAsset(id) as Promise<AssetDetail>),
  getAnlagenspiegel: (): Promise<Anlagenspiegel> =>
    call(() => Bridge.GetAnlagenspiegel() as Promise<Anlagenspiegel>),
  /** Buchungen auf Anlagekonten, zu denen noch kein Anlagegut erfasst ist. */
  getAssetAcquisitionCandidates: (): Promise<AcquisitionCandidate[]> =>
    call(() => Bridge.GetAssetAcquisitionCandidates() as Promise<AcquisitionCandidate[]>),

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

  // --- E-Bilanz, Audit & Festschreibung ---------------------------------

  /**
   * Erzeugt die E-Bilanz als XBRL-Instanz oder als Summen- und Saldenliste
   * für eBilanz+ und schreibt sie an einen gewählten Ort. Leer heißt: der
   * Dialog wurde abgebrochen.
   */
  saveEBilanzAs: (year: number, format: 'xbrl' | 'susa_csv'): Promise<string> =>
    call(() => Bridge.SaveEBilanzAs(year, format) as Promise<string>),
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

  getAccruals: (year: number): Promise<Accrual[]> =>
    call(() => Bridge.GetAccruals(year) as Promise<Accrual[]>).then((accruals) =>
      list(accruals).map(normalizeAccrual),
    ),

  getProvisions: (year: number): Promise<Provision[]> =>
    call(() => Bridge.GetProvisions(year) as Promise<Provision[]>).then((provisions) =>
      list(provisions).map(normalizeProvision),
    ),
  /** Der Rückstellungsspiegel des Anhangs: er geht per Definition auf. */
  getProvisionMirror: (year: number): Promise<ProvisionMirror> =>
    call(() => Bridge.GetProvisionMirror(year) as Promise<ProvisionMirror>).then((mirror) =>
      mirror ? { ...mirror, rows: list(mirror.rows) } : mirror,
    ),

  /** Vorsteuer, Umsatzsteuer und Vorauszahlungen zu einem Saldo verrechnet. */
  previewVatSettlement: (year: number): Promise<VatSettlement> =>
    call(() => Bridge.PreviewVatSettlement(year) as Promise<VatSettlement>).then((settlement) =>
      settlement
        ? { ...settlement, rows: list(settlement.rows), lines: list(settlement.lines) }
        : settlement,
    ),
  /** Körperschaftsteuer, Solidaritätszuschlag und Gewerbesteuer — eine Schätzung. */
  previewTaxProvision: (year: number): Promise<TaxProvisionPreview> =>
    call(() => Bridge.PreviewTaxProvision(year) as Promise<TaxProvisionPreview>).then((preview) =>
      preview ? { ...preview, lines: list(preview.lines) } : preview,
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
  /** Die Überleitung Handelsbilanz → Steuerbilanz (§ 60 Abs. 2 EStDV). */
  getReconciliation: (year: number): Promise<Reconciliation> =>
    call(() => Bridge.GetReconciliation(year) as Promise<Reconciliation>).then((reconciliation) =>
      reconciliation ? { ...reconciliation, rows: list(reconciliation.rows) } : reconciliation,
    ),

  // --- Umsatzsteuer-Voranmeldung ----------------------------------------

  /**
   * Das Kennziffernblatt eines Zeitraums, neu gerechnet und nicht gespeichert.
   * Der Entwurf ist immer der heutige Stand des Journals.
   */
  getVatReturn: (periodKey: string): Promise<VatReturn> =>
    call(() => Bridge.GetVatReturn(periodKey) as Promise<VatReturn>).then(normalizeVatReturn),
  saveVatReturn: (periodKey: string): Promise<VatReturn> =>
    call(() => Bridge.SaveVatReturn(periodKey) as Promise<VatReturn>).then(normalizeVatReturn),
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

  // --- Zusammenfassende Meldung -----------------------------------------

  /** Die Meldezeiträume folgen den Umsätzen, nicht einer Einstellung. */
  getZMPeriods: (year: number): Promise<ZMPeriodStatus[]> =>
    call(() => Bridge.GetZMPeriods(year) as Promise<ZMPeriodStatus[]>).then(list),
  getZMReturn: (periodKey: string): Promise<ZMReturn> =>
    call(() => Bridge.GetZMReturn(periodKey) as Promise<ZMReturn>).then(normalizeZMReturn),
  saveZMReturn: (periodKey: string): Promise<ZMReturn> =>
    call(() => Bridge.SaveZMReturn(periodKey) as Promise<ZMReturn>).then(normalizeZMReturn),
  confirmZMSubmitted: (id: number, date: string, ticket: string, note = ''): Promise<ZMReturn> =>
    call(() => Bridge.ConfirmZMSubmitted(id, date, ticket, note) as Promise<ZMReturn>).then(
      normalizeZMReturn,
    ),

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

  // --- Pflichtenkalender und Jahreserklärungen ---------------------------

  /** Die Pflichten eines Geschäftsjahres mit Stand, Nachweis und Ziel, die früheste zuerst. */
  getObligations: (year: number): Promise<Obligation[]> =>
    call(() => Bridge.GetObligations(year) as Promise<Obligation[]>).then((items) =>
      list(items).map(normalizeObligation),
    ),
  /** Eine Jahreserklärung, gerechnet aus dem heutigen Stand der Bücher. */
  getDeclaration: (kind: DeclarationKind, year: number): Promise<DeclarationView> =>
    call(() => Bridge.GetDeclaration(kind, year) as Promise<DeclarationView>).then(
      normalizeDeclaration,
    ),
  setDeclarationInput: (request: DeclarationInputRequest): Promise<DeclarationInput> =>
    call(() => Bridge.SetDeclarationInput(request) as Promise<DeclarationInput>),
  /** Bestätigt den Vorschlag eines Agenten. Nur in der App. */
  confirmDeclarationInput: (
    kind: DeclarationKind,
    year: number,
    key: string,
  ): Promise<DeclarationInput> =>
    call(() => Bridge.ConfirmDeclarationInput(kind, year, key) as Promise<DeclarationInput>),
  deleteDeclarationInput: (kind: DeclarationKind, year: number, key: string): Promise<void> =>
    call(() => Bridge.DeleteDeclarationInput(kind, year, key)),
  /** Hält die Übermittlung in Mein ELSTER fest. Nur in der App. */
  submitDeclaration: (request: DeclarationSubmissionRequest): Promise<DeclarationRecord> =>
    call(() => Bridge.SubmitDeclaration(request) as Promise<DeclarationRecord>),
  /** Schreibt die Erklärung als JSON in einen gewählten Ordner; null heißt abgebrochen. */
  exportDeclarationJSON: (kind: DeclarationKind, year: number): Promise<DeclarationExport | null> =>
    call(() => Bridge.ExportDeclarationJSON(kind, year) as Promise<DeclarationExport | null>),

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

  // --- Bestätigung der USt-IdNr. (§ 18e UStG) ----------------------------

  /** Der Stand der Bestätigung, ohne dass dafür gefragt würde. */
  getVatIDStatus: (contactId: number): Promise<VatIDStatus> =>
    call(() => Bridge.GetVatIDStatus(contactId) as Promise<VatIDStatus>),

  // --- Nicht abziehbare Betriebsausgaben (§ 4 Abs. 5 EStG) ---------------

  /** Der Bericht eines Geschäftsjahres: je Kategorie und je Empfänger. */
  getNonDeductibleReport: (year: number): Promise<NonDeductibleReport> =>
    call(() => Bridge.GetNonDeductibleReport(year) as Promise<NonDeductibleReport>).then(
      normalizeNonDeductibleReport,
    ),

  // --- Fremdwährung ------------------------------------------------------

  /** Die Stichtagsbewertung, ohne sie zu buchen (§ 256a HGB). */
  previewCurrencyValuation: (year: number): Promise<ForeignCurrencyValuation> =>
    call(() => Bridge.PreviewCurrencyValuation(year) as Promise<ForeignCurrencyValuation>).then(
      normalizeCurrencyValuation,
    ),

  // --- Anlagen: Wertaufholung -------------------------------------------

  /** Die Anlagegüter, bei denen die Wertaufholung zu prüfen ist (§ 253 Abs. 5 HGB). */
  getWriteUpReport: (year: number): Promise<WriteUpReport> =>
    call(() => Bridge.GetWriteUpReport(year) as Promise<WriteUpReport>).then(normalizeWriteUpReport),

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

  // --- Prüfermodus -------------------------------------------------------

  /** Schaltet den Prüfermodus bis zu einem Tag ein. Datum und Grund sind Pflicht. */
  enableReadOnly: (until: string, reason: string): Promise<AppConfig> =>
    call(() => Bridge.EnableReadOnly(until, reason) as Promise<AppConfig>),
  /** Beendet ihn. Der Grund steht danach im Änderungsprotokoll. */
  disableReadOnly: (reason: string): Promise<AppConfig> =>
    call(() => Bridge.DisableReadOnly(reason) as Promise<AppConfig>),

  // --- Eingang -----------------------------------------------------------

  /** Pfad, Anzahl und Alter der ältesten Datei im Eingang, dazu Lese- und Exportordner. */
  getInbox: (): Promise<InboxInfo> =>
    call(() => Bridge.GetInbox() as Promise<InboxInfo>).then((info) => ({
      ...info,
      readFolders: list(info?.readFolders),
    })),
  /** Die Dateien im Eingang, die älteste zuerst. */
  listInbox: (): Promise<InboxFile[]> =>
    call(() => Bridge.ListInbox() as Promise<InboxFile[]>).then(list),
  /** Öffnet den Eingang im Dateimanager des Systems. */
  openInbox: (): Promise<void> => call(() => Bridge.OpenInbox()),
  /** Kopiert Dateien in den Eingang; abgelegt wird dabei nichts. */
  addToInbox: (paths: string[]): Promise<InboxAdded> =>
    call(() => Bridge.AddToInbox(paths) as Promise<InboxAdded>).then(normalizeInboxAdded),
  /** Wählt den Eingang; leer heißt `<Datenordner>/Eingang`. Nur in der App. */
  setInboxFolder: (path: string): Promise<InboxInfo> =>
    call(() => Bridge.SetInboxFolder(path) as Promise<InboxInfo>).then((info) => ({
      ...info,
      readFolders: list(info?.readFolders),
    })),
  /** Ob ein Ordner synchronisiert wird; geht auch vor der Einrichtung. */
  checkFolder: (path: string): Promise<FolderCheck> =>
    call(() => Bridge.CheckFolder(path) as Promise<FolderCheck>),

  // --- Regeln für verbundene Apps ----------------------------------------

  /** Direktgrenze, Mindestzahl früherer Buchungen, Lese- und Exportordner. */
  getAgentRules: (): Promise<AgentRules> =>
    call(() => Bridge.GetAgentRules() as Promise<AgentRules>).then(normalizeAgentRules),
  /** Speichert die Regeln; 0 und leer heißen Voreinstellung. Nur in der App. */
  saveAgentRules: (settings: AgentSettings): Promise<AgentRules> =>
    call(() => Bridge.SaveAgentRules(settings) as Promise<AgentRules>).then(normalizeAgentRules),

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
  /** Erfasst einen Darlehensvertrag; Claude kann das auch mit `darlehen_erfassen`. */
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

  // --- Übersicht: Pflichten und Buchungen der Apps -----------------------

  /** Die Pflichten, die noch etwas verlangen, nach Fälligkeit; 0 heißt alle. */
  getNextObligations: (limit = 0): Promise<Obligation[]> =>
    call(() => Bridge.GetNextObligations(limit) as Promise<Obligation[]>).then((items) =>
      list(items).map(normalizeObligation),
    ),
  /** Eine Pflicht mit Stand, Vorbereitung und Nachweis. */
  getObligation: (key: string): Promise<Obligation> =>
    call(() => Bridge.GetObligation(key) as Promise<Obligation>).then(normalizeObligation),
  /** Bestätigt eine Pflicht mit Nachweis. Nur in der App. */
  confirmObligation: (key: string, evidence: ObligationConfirmation): Promise<Obligation> =>
    call(() => Bridge.ConfirmObligation(key, evidence) as Promise<Obligation>).then(
      normalizeObligation,
    ),
  /** Kennzeichnet eine Pflicht mit Grund als entfallen. */
  waiveObligation: (key: string, reason: string): Promise<Obligation> =>
    call(() => Bridge.WaiveObligation(key, reason) as Promise<Obligation>).then(
      normalizeObligation,
    ),
  /** Nimmt Bestätigung oder Entfallen mit Grund zurück. */
  reopenObligation: (key: string, reason: string): Promise<Obligation> =>
    call(() => Bridge.ReopenObligation(key, reason) as Promise<Obligation>).then(
      normalizeObligation,
    ),
  /** Was verbundene Apps seit der letzten Festschreibung gebucht haben, die jüngste zuerst. */
  getAgentBookings: (limit = 0): Promise<AgentBookings> =>
    call(() => Bridge.GetAgentBookings(limit) as Promise<AgentBookings>).then((result) => ({
      total: result?.total ?? 0,
      lastCommit: result?.lastCommit,
      entries: list(result?.entries).map((entry) => ({ ...entry, lines: list(entry.lines) })),
    })),

  // --- Belegverweis, Journalfilter, Aufbewahrung ------------------------

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
