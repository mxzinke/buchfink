/**
 * Spiegelt die Go-Domäne (internal/domain). Änderungen dort müssen hier
 * nachgezogen werden.
 */

/**
 * Cents ist ein Geldbetrag in ganzen Cent — 119000 sind 1.190,00 €.
 *
 * Das Backend rechnet ausschließlich in Cent, damit die Grundinvariante
 * Soll = Haben exakt prüfbar bleibt. Im Frontend gilt dasselbe: niemals in
 * Euro-Fließkommazahlen umrechnen, zur Anzeige `formatCents` benutzen.
 */
export type Cents = number;

export type Side = 'S' | 'H';
export type EntryKind = 'normal' | 'reversal';
export type EntrySource =
  | 'manual'
  | 'receipt'
  | 'invoice'
  | 'payment'
  | 'opening'
  | 'depreciation'
  | 'closing';

export type Direction = 'incoming' | 'outgoing';

export type TaxTreatment =
  | 'domestic'
  | 'reverse_charge'
  | 'intra_community_acquisition'
  | 'intra_community_supply'
  | 'export'
  | 'reverse_charge_supply'
  /** Nullsteuersatz § 12 Abs. 3 UStG — steuerpflichtig zum Satz null, nicht steuerfrei. */
  | 'zero_rated'
  | 'exempt'
  | 'not_taxable';

/** Steuersatz in Basispunkten: 1900 = 19 %. */
export type TaxRate = number;

export const TAX_RATE_NONE: TaxRate = 0;
export const TAX_RATE_REDUCED: TaxRate = 700;
export const TAX_RATE_STANDARD: TaxRate = 1900;

export type DifferenceKind = 'none' | 'skonto' | 'bank_fee' | 'rounding' | 'currency';

export type ContactType = 'customer' | 'vendor';
export type Settlement = 'open' | 'paid';
export type AccountType = 'asset' | 'liability' | 'equity' | 'revenue' | 'expense' | 'statistical';

// -------------------------------------------------------------------------

export interface TenantConfig {
  id: string;
  name: string;
  dataDir: string;
  createdAt: string;
}

export interface AppConfig {
  tenants: TenantConfig[];
  activeTenantId: string;
  dataDir: string;
  isConfigured: boolean;
  lastFiscalYear: number;
}

export interface Account {
  id: number;
  number: string;
  name: string;
  type: AccountType;
  category: string;
  subcategory: string;
  kontenklasse: number;
  kontenklasseName: string;
  positionId: string;
  posten: string;
  balanceSide: string;
  hgbCode: string;
  statementType: string;
  taxRate: number;
  hauptfunktion: string;
  hauptfunktionDesc: string;
  zusatzfunktion: string;
  zusatzfunktionDesc: string;
  abschlusszweck: string;
  isRange: boolean;
  rangeStart: string;
  rangeEnd: string;
  isReserved: boolean;
  description: string;
  isActive: boolean;
  debitSum: Cents;
  creditSum: Cents;
  balance: Cents;
  bookingsCount: number;
  /** Zahl der Personenkonten, aus denen diese Bilanzposition verdichtet ist. */
  aggregatedAccounts?: number;
}

// -------------------------------------------------------------------------
// Journal

export interface JournalLine {
  id: number;
  entryId: number;
  position: number;
  side: Side;
  amount: Cents;
  account: string;
  accountName?: string;
  contactId?: number;
  taxKey?: string;
  taxBase?: Cents;
  text?: string;
}

export interface JournalEntry {
  id: number;
  fiscalYear: number;
  entryNumber: string;
  bookingDate: string;
  documentDate: string;
  serviceDateFrom: string;
  serviceDateTo: string;
  valueDate?: string;
  description: string;
  source: EntrySource;
  documentNumber?: string;
  receiptId?: number;
  receiptHash?: string;
  taxTreatment?: TaxTreatment;
  contactId?: number;
  bankTxId?: number;
  kind: EntryKind;
  reversalOfId?: number;
  reversalReason?: string;
  currency: string;
  exchangeRateMicros: number;
  exchangeRateSource?: string;
  exchangeRateDate?: string;
  postingRuleVersion?: string;
  lines: JournalLine[];
  previousHash: string;
  entryHash: string;
  createdAt: string;
}

export interface CounterAccount {
  account: string;
  name: string;
  amount: Cents;
}

export interface AccountLedgerRow {
  entryId: number;
  entryNumber: string;
  bookingDate: string;
  documentDate: string;
  documentNumber?: string;
  description: string;
  kind: EntryKind;
  side: Side;
  debitAmount: Cents;
  creditAmount: Cents;
  runningBalance: Cents;
  counterAccounts: CounterAccount[] | null;
  taxKey?: string;
}

export interface AccountLedger {
  account: Account;
  fiscalYear: number;
  openingBalance: Cents;
  totalDebit: Cents;
  totalCredit: Cents;
  closingBalance: Cents;
  rowCount: number;
  rows: AccountLedgerRow[] | null;
}

export interface SuSaClassSummary {
  kontenklasse: number;
  kontenklasseName: string;
  totalDebit: Cents;
  totalCredit: Cents;
  totalSaldoDebit: Cents;
  totalSaldoCredit: Cents;
  accountsCount: number;
  accounts: Account[];
}

export interface SuSaOverview {
  fiscalYear: number;
  totalDebit: Cents;
  totalCredit: Cents;
  totalSaldoDebit: Cents;
  totalSaldoCredit: Cents;
  isBalanced: boolean;
  difference: Cents;
  classes: SuSaClassSummary[];
}

// -------------------------------------------------------------------------
// Kontierung

export interface PostingGroup {
  key: string;
  label: string;
  category: string;
  hint?: string;
  direction: Direction;
  account: string;
  defaultRate: TaxRate;
  /** Der Steuerfall, den die Gruppe vorschlägt. Leer = Inland, steuerpflichtig. */
  defaultTreatment?: TaxTreatment;
  /** Konto für den nicht abzugsfähigen Anteil, z. B. 6644 bei Bewirtung. */
  nonDeductibleAccount?: string;
  /** Gesetzliche Abzugsquote, die für diese Gruppe gilt. */
  deductibleQuota?: string;
}

export interface TaxTreatmentInfo {
  treatment: TaxTreatment;
  label: string;
  hint: string;
  direction: Direction;
  requiresRate: boolean;
  requiresVatId: boolean;
}

export interface DifferenceKindInfo {
  kind: DifferenceKind;
  label: string;
  hint: string;
}

export interface ReceiptPosition {
  postingGroup: string;
  account?: string;
  net: Cents;
  taxRate: TaxRate;
  text?: string;
}

export interface ReceiptRequest {
  contactId: number;
  /** Der abgelegte Beleg. Pflicht: keine Buchung ohne Beleg. */
  receiptId: number;
  bookingDate: string;
  documentDate: string;
  serviceDateFrom: string;
  serviceDateTo: string;
  description: string;
  taxTreatment: TaxTreatment;
  positions: ReceiptPosition[];
  settlement: Settlement;
  paymentAccount?: string;
  currency?: string;
  /** Pflicht, sobald auf ein Bewirtungskonto gebucht wird. */
  entertainment?: EntertainmentDetail;
}

/**
 * Der Buchungssatz, wie das Backend ihn berechnet — ohne ihn zu schreiben.
 *
 * Die Oberfläche zeigt diese Zahlen an und rechnet sie nicht selbst nach. Eine
 * zweite Steuerrechnung im Frontend wäre eine zweite Wahrheit, die auseinander
 * läuft, sobald ein Steuerfall dazukommt.
 */
export interface PostingPreview {
  lines: JournalLine[];
  /** Summe der Aufwands- bzw. Ertragszeilen. */
  net: Cents;
  /** Differenz zwischen Brutto und Netto. Bei § 13b null: gezahlt wird netto. */
  tax: Cents;
  /** Was tatsächlich gezahlt oder vereinnahmt wird — die Gegenzeile. */
  gross: Cents;
  balanced: boolean;
  warnings?: PostingWarning[];
}

// -------------------------------------------------------------------------
// Belege

export type ReceiptFileRole = 'original' | 'structured' | 'rendering' | 'attachment';
export type ReceiptStatus = 'filed' | 'sealed' | 'discarded';

export interface ReceiptFile {
  id: number;
  receiptId: number;
  position: number;
  role: ReceiptFileRole;
  fileName: string;
  mimeType: string;
  size: number;
  sha256: string;
  /** Aus einer anderen Datei erzeugt statt empfangen. */
  derived: boolean;
  storedPath: string;
  createdAt: string;
}

export interface Receipt {
  id: number;
  fiscalYear: number;
  receiptNumber: string;
  direction: Direction;
  status: ReceiptStatus;
  files: ReceiptFile[];
  /** Über die geordnete Dateiliste; steht so in der Buchung. */
  receiptHash: string;
  receivedAt?: string;
  receivedVia?: string;
  journalEntryId?: number;
  discardReason?: string;

  // E-Rechnung: leer bei einem Scan oder einer gewöhnlichen PDF-Rechnung.
  detectedFormat?: string;
  detectedProfile?: string;
  validatedAt?: string;
  validationRuleset?: string;
  validationVersion?: string;
  validationCoverage?: string;
  validationErrors: number;
  /** Die Befunde als JSON, siehe ValidationFinding. */
  validationFindings?: string;

  createdAt: string;
  updatedAt: string;
}

export interface ReceiptFileInput {
  path: string;
  role: ReceiptFileRole;
}

/**
 * Was Buchfink aus einer empfangenen E-Rechnung liest.
 *
 * Ein Vorschlag, keine Buchung. Die Buchungsgruppe bleibt leer — welches
 * Aufwandskonto zutrifft, sagt keine Rechnung.
 */
export interface EInvoiceProposal {
  request: ReceiptRequest;
  format: string;
  profile: string;
  /** Was der Datensatz zu sein angibt — entscheidet über das Vorzeichen. */
  kind: string;
  kindLabel: string;
  supplierName: string;
  supplierVatId?: string;
  supplierTaxId?: string;
  invoiceNumber: string;
  grossAmount: Cents;
  matchedContact: boolean;
  /** Rechnungen, auf die sich dieser Beleg bezieht (BG-3). */
  precedingInvoices?: string[];
  /** Was nicht gefüllt werden konnte und warum. */
  notes?: string[];
}

/**
 * Das Ergebnis der EN-16931-Prüfung.
 *
 * `coverage` kennt bewusst keinen Wert für „vollständig geprüft": die
 * Referenzumsetzung ist ein Schematron-Regelwerk, das kein Go-Prozessor
 * ausführt. Die geprüften Regeln sind über `Api.getEInvoiceRules()` abrufbar.
 */
export interface ValidationFinding {
  rule: string;
  /** Der Schweregrad stammt aus der Spezifikation, nicht aus Buchfink. */
  severity: 'fatal' | 'warning' | 'information';
  /** Wo im Beleg, z. B. "Position 3". Leer bei Dokumentebene. */
  where?: string;
  /** Die betroffenen Geschäftsbegriffe (BT/BG). */
  terms?: string[];
  message: string;
}

export interface ValidationResult {
  ruleset: string;
  version: string;
  format: string;
  profile: string;
  coverage: 'partial';
  findings?: ValidationFinding[];
}

export interface ReceiptPreview {
  dataUrl: string;
  fileName: string;
  mimeType: string;
  /** false, wenn die Datei auf der Platte nicht mehr zu ihrer Prüfsumme passt. */
  intact: boolean;
}

// -------------------------------------------------------------------------
// Offene Posten & Zahlungen

export interface OpenItem {
  entryId: number;
  entryNumber: string;
  contactId: number;
  contactName: string;
  contactType: ContactType;
  ledgerAccount: string;
  documentNumber: string;
  documentDate: string;
  dueDate: string;
  grossAmount: Cents;
  settledAmount: Cents;
  openAmount: Cents;
  taxRate: TaxRate;
  /**
   * Der Steuerfall der ursprünglichen Buchung. Er entscheidet, wie ein Skonto
   * berichtigt wird: nur beim steuerpflichtigen Inlandsumsatz steckt die Steuer
   * im offenen Betrag, und § 13b und der innergemeinschaftliche Erwerb haben
   * zwei Steuerzeilen statt einer (§ 17 Abs. 1 Satz 5 UStG).
   */
  taxTreatment?: TaxTreatment;
}

export interface AllocationRequest {
  openItemEntryId: number;
  settledAmount: Cents;
  differenceKind: DifferenceKind;
  differenceAmount: Cents;
}

export interface PaymentRequest {
  bankTxId?: number;
  paymentAccount: string;
  paymentDate: string;
  valueDate?: string;
  description?: string;
  allocations: AllocationRequest[];
}

// -------------------------------------------------------------------------

export interface BankTransaction {
  id: number;
  fiscalYear: number;
  accountIban: string;
  bookingDate: string;
  valueDate: string;
  /** Positiv = Eingang, negativ = Ausgang. */
  amount: Cents;
  currency: string;
  counterpartyName: string;
  counterpartyIban: string;
  remittanceInfo: string;
  endToEndId: string;
  matchStatus: 'unmatched' | 'matched' | 'ignored';
  ledgerAccount: string;
  matchedAmount: Cents;
}

export interface Contact {
  id: number;
  type: ContactType;
  ledgerAccount: string;
  name: string;
  company: string;
  email: string;
  address: string;
  taxId: string;
  vatId: string;
  countryCode: string;
  iban: string;
  bic: string;
  paymentTermsDays: number;
  /** Keine Unternehmerin/kein Unternehmer — dann greift keine E-Rechnungspflicht. */
  isPrivate: boolean;
  /** Kleinunternehmer nach § 19 UStG: darf immer eine sonstige Rechnung stellen. */
  isSmallBusiness: boolean;
  openAmount: Cents;
  createdAt: string;
}

/** Aufzeichnung zu einer Bewirtung, § 4 Abs. 5 Satz 1 Nr. 2 EStG. */
export interface EntertainmentDetail {
  place: string;
  day: string;
  participants: string;
  occasion: string;
}

/** Ein Hinweis zur Buchung. Blockiert nie — was folgt, ist eine Rechtsfrage. */
export interface PostingWarning {
  code: string;
  severity: 'info' | 'warning';
  title: string;
  detail: string;
  /** Text zum Weitergeben an den Lieferanten. */
  supplierNote?: string;
}

export interface InvoiceItem {
  id?: number;
  invoiceId?: number;
  position: number;
  description: string;
  /** Menge mit drei Nachkommastellen: 1500 = 1,5. */
  quantityMilli: number;
  unit: string;
  unitPrice: Cents;
  taxRate: TaxRate;
  postingGroup?: string;
}

export interface Invoice {
  id: number;
  fiscalYear: number;
  invoiceNumber: string;
  date: string;
  serviceDateFrom: string;
  serviceDateTo: string;
  dueDate: string;
  contactId: number;
  contactName: string;
  items: InvoiceItem[];
  taxTreatment: TaxTreatment;
  netAmount: Cents;
  taxAmount: Cents;
  grossAmount: Cents;
  currency: string;
  status: 'draft' | 'issued' | 'paid' | 'cancelled';
  journalEntryId?: number;
  /** Der Beleg mit dem hybriden PDF und dem ZUGFeRD-XML. */
  receiptId?: number;
  paidAmount: Cents;
  createdAt: string;
}

export interface VatFigure {
  rate: TaxRate;
  net: Cents;
  tax: Cents;
}

export interface VatSummary {
  fiscalYear: number;
  periodFrom: string;
  periodTo: string;
  taxableRevenue: VatFigure[] | null;
  exemptRevenue: Cents;
  intraCommunitySupply: Cents;
  export: Cents;
  reverseChargeSupply: Cents;
  outputTax: Cents;
  reverseChargeTax: Cents;
  reverseChargeBase: Cents;
  intraCommunityAcquisitionTax: Cents;
  intraCommunityAcquisitionBase: Cents;
  totalOwedTax: Cents;
  inputTax: Cents;
  payable: Cents;
}

export interface CashflowDataPoint {
  month: string;
  label: string;
  inflow: Cents;
  outflow: Cents;
  net: Cents;
}

export interface FinancialSummary {
  totalRevenue: Cents;
  totalExpenses: Cents;
  netIncome: Cents;
  bankBalance: Cents;
  openReceivables: Cents;
  openPayables: Cents;
  cashflowHistory: CashflowDataPoint[] | null;
}

export interface IntegrityCheckResult {
  isValid: boolean;
  totalEntries: number;
  checkedEntries: number;
  firstBrokenId?: number;
  message: string;
  lastVerifiedHash: string;
  checkedAt: string;
}

export interface CompanySettings {
  companyName: string;
  legalForm: string;
  fiscalYear: number;
  fiscalYearStartMonth: number;
  taxNumber: string;
  vatId: string;
  taxOffice: string;
  iban: string;
  bic: string;
  bankName: string;
  street: string;
  zipCity: string;
  country: string;
  currency: string;
  skr: string;
  vatPeriod: string;
  taxationType: string;
}

export interface AuditLogEntry {
  id: number;
  timestamp: string;
  action: string;
  entityType: string;
  entityId: string;
  details: string;
  previousHash?: string;
  entryHash?: string;
}

export interface Festschreibung {
  id: number;
  fiscalYear: number;
  periodType: string;
  periodLabel: string;
  cutoffDate: string;
  chainHead: string;
  entryCount: number;
  tsaName: string;
  tsaGenTime?: string;
  timestampStatus: string;
  createdAt: string;
}

export interface FestschreibungVerification {
  id: number;
  hasTimestamp: boolean;
  isValid: boolean;
  coversCurrent: boolean;
  genTime?: string;
  tsaName: string;
  message: string;
}

// -------------------------------------------------------------------------
// SKR04-Katalog (statisch, aus skr04_2026.json)

export interface SKR04Position {
  id: string;
  name: string;
  statement_type: string;
  balance_side: string;
  hgb_code: string;
  group: string;
  main_group: string;
  account_type: string;
  kontenklasse: { number: number; name: string };
  account_numbers: string[];
  accounts_count: number;
}

export interface SKR04Metadata {
  title: string;
  subtitle: string;
  validity_from: string;
  version: string;
  article_number: string;
  source_file: string;
  description: string;
  generated_at: string;
}

export interface SKR04Legend {
  hauptfunktionen: Record<string, string>;
  zusatzfunktionen: Record<string, string>;
  abschlusszweck: Record<string, string>;
  programmverbindung: Record<string, string>;
  footnotes: Record<string, string>;
}

export interface SKR04Statistics {
  total_accounts: number;
  active_accounts: number;
  reserved_accounts: number;
  range_accounts: number;
  total_positions: number;
  accounts_by_type: Record<string, number>;
  accounts_by_kontenklasse: Record<string, number>;
  positions_by_side: Record<string, number>;
}

export interface SKR04Catalog {
  metadata: SKR04Metadata;
  legend: SKR04Legend;
  statistics: SKR04Statistics;
  positions: SKR04Position[];
}

// -------------------------------------------------------------------------
// Anlagevermögen (internal/domain/asset.go)

/** Die drei Blöcke des Anlagevermögens nach § 266 Abs. 2 A HGB. */
export type AssetClass = 'intangible' | 'tangible' | 'financial';

export type DepreciationMethod = 'linear' | 'degressive' | 'pool' | 'immediate' | 'none';

export type AssetStatus =
  | 'active'
  | 'fully_written'
  | 'disposed'
  | 'unbooked'
  | 'depreciate_due';

export type DisposalKind = 'sale' | 'scrapped';

export type AssetMovementKind =
  | 'acquisition'
  | 'subsequent_cost'
  | 'cost_reduction'
  | 'depreciation'
  | 'impairment'
  | 'write_up'
  | 'disposal';

export interface AssetMovement {
  id: number;
  assetId: number;
  kind: AssetMovementKind;
  date: string;
  fiscalYear: number;
  /** Verändert die Anschaffungs- und Herstellungskosten. */
  costAmount: Cents;
  /** Verändert die kumulierten Abschreibungen. */
  depreciationAmount: Cents;
  journalEntryId?: number;
  entryNumber?: string;
  note?: string;
  createdAt: string;
}

export interface FixedAsset {
  id: number;
  inventoryNumber: string;
  name: string;
  description?: string;
  class: AssetClass;
  account: string;
  depreciationAccount?: string;
  acquisitionDate: string;
  acquisitionCost: Cents;
  method: DepreciationMethod;
  usefulLifeMonths: number;
  poolYear?: number;
  identifier?: string;
  /** Beteiligungsquote in Promille: 200 sind 20 %. */
  holdingPermille?: number;
  taxPrivileged?: boolean;
  contactId?: number;
  acquisitionEntryId?: number;
  disposalDate?: string;
  disposalKind?: DisposalKind;
  disposalProceeds?: Cents;
  disposalEntryId?: number;
  notes?: string;
  createdAt: string;
  updatedAt: string;
  movements?: AssetMovement[];

  // Abgeleitet vom Backend, nicht gespeichert.
  accountName?: string;
  cost: Cents;
  accumulated: Cents;
  bookValue: Cents;
  yearAmount: Cents;
  dueAmount: Cents;
  status: AssetStatus;
  statusNote?: string;
}

export interface AssetSummary {
  fiscalYear: number;
  count: number;
  cost: Cents;
  accumulated: Cents;
  bookValue: Cents;
  yearAmount: Cents;
  dueAmount: Cents;
  dueCount: number;
}

export interface AssetScheduleYear {
  fiscalYear: number;
  months: number;
  method: DepreciationMethod;
  rateLabel: string;
  openingBookValue: Cents;
  amount: Cents;
  closingBookValue: Cents;
  note?: string;
  booked: Cents;
  due: Cents;
  status: 'gebucht' | 'offen' | 'teilweise' | 'geplant';
}

export interface AssetDetail {
  asset: FixedAsset;
  schedule: AssetScheduleYear[];
  movements: AssetMovement[];
  /** Höchstbetrag einer Zuschreibung (§ 253 Abs. 5 Satz 1 HGB), vom Backend gerechnet. */
  writeUpCeiling: Cents;
  /** Die Sätze, die zu genau diesem Anlagegut gehören — vom Backend gerechnet. */
  notes: string[];
}

export interface AssetAccountInfo {
  number: string;
  name: string;
  class: AssetClass;
  group: string;
  hint?: string;
  depreciationAccount?: string;
  depreciable: boolean;
  defaultUsefulLifeMonths?: number;
  usefulLifeSource?: string;
}

export type AcquisitionOption = 'immediate' | 'pool' | 'activate';

export interface AcquisitionAdvice {
  recommended: AcquisitionOption;
  allowed: AcquisitionOption[];
  reason: string;
  poolNote?: string;
  limits: {
    immediate: Cents;
    recordFrom: Cents;
    poolLowerLimit: Cents;
    poolUpperLimit: Cents;
  };
}

export interface DegressiveWindow {
  From: string;
  Until: string;
  FactorTimes: number;
  MaxPermille: number;
  Source: string;
}

export interface AssetMethodInfo {
  method: DepreciationMethod;
  label: string;
  classes: AssetClass[];
  hint: string;
}

export interface AssetRules {
  fiscalYear: number;
  gwgImmediateLimit: Cents;
  gwgRecordFrom: Cents;
  poolLowerLimit: Cents;
  poolUpperLimit: Cents;
  poolYears: number;
  degressiveWindows: DegressiveWindow[];
  methods: AssetMethodInfo[];
}

export interface DepreciationDue {
  assetId: number;
  inventoryNumber: string;
  name: string;
  account: string;
  expenseAccount: string;
  method: string;
  rateLabel: string;
  months: number;
  planned: Cents;
  booked: Cents;
  due: Cents;
  bookValueBefore: Cents;
  bookValueAfter: Cents;
  note?: string;
}

export interface DepreciationRun {
  fiscalYear: number;
  bookingDate: string;
  due: DepreciationDue[];
  total: Cents;
  missingPriorYears?: number[];
}

export interface DepreciationResult {
  entries: JournalEntry[];
  total: Cents;
  skipped?: string[];
}

export interface DisposalAccounts {
  revenue?: string;
  bookValue: string;
  explanation: string;
}

export interface DisposalRequest {
  assetId: number;
  date: string;
  kind: DisposalKind;
  proceeds: Cents;
  taxTreatment?: TaxTreatment;
  taxRate?: TaxRate;
  settlement: Settlement;
  paymentAccount?: string;
  contactId?: number;
  note?: string;
}

export interface DisposalPreview {
  catchUpAmount: Cents;
  catchUpLines?: JournalLine[];
  bookValue: Cents;
  /** Buchgewinn positiv, Buchverlust negativ. */
  result: Cents;
  isGain: boolean;
  accounts: DisposalAccounts;
  lines: JournalLine[];
  gross: Cents;
  tax: Cents;
  warnings?: string[];
}

export interface DisposalResult {
  catchUpEntry?: JournalEntry;
  disposalEntry?: JournalEntry;
  asset: FixedAsset;
  message: string;
}

export interface AcquisitionCandidate {
  entryId: number;
  entryNumber: string;
  bookingDate: string;
  description: string;
  account: string;
  accountName: string;
  amount: Cents;
  contactId?: number;
}

export interface AnlagenspiegelRow {
  class: AssetClass;
  account: string;
  accountName: string;
  assetCount: number;
  costOpening: Cents;
  additions: Cents;
  disposals: Cents;
  costClosing: Cents;
  depreciationOpening: Cents;
  depreciationYear: Cents;
  writeUpsYear: Cents;
  depreciationDisposal: Cents;
  depreciationClosing: Cents;
  bookValueOpening: Cents;
  bookValueClosing: Cents;
}

export interface Anlagenspiegel {
  fiscalYear: number;
  rows: AnlagenspiegelRow[];
  totals: AnlagenspiegelRow;
  classTotals: AnlagenspiegelRow[];
}
