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
  /**
   * Sitz, Registergericht und Registernummer sind die Pflichtangaben des
   * § 264 Abs. 1a HGB auf jedem Jahresabschluss. Sie standen bisher nur an der
   * Gründung und fehlten damit jedem Mandanten ohne Gründungsweg.
   */
  seat: string;
  registerCourt: string;
  registerNumber: string;
  currency: string;
  skr: string;
  vatPeriod: string;
  taxationType: string;
  /**
   * Legt die Anlegerstellung für § 20 InvStG ausdrücklich fest — normalerweise
   * leer, weil sie aus der Rechtsform folgt.
   *
   * Gebraucht wird sie in zwei Fällen: bei einer Personengesellschaft, wo
   * § 20 Abs. 3a InvStG auf den Gesellschafter abstellt, und bei den Ausnahmen
   * des § 20 Abs. 1 Sätze 4 und 5 — Lebens- und Krankenversicherer,
   * Kreditinstitute mit Handelsbestand, Pensionsfonds.
   */
  investorOverride: InvestorType;
}

/** Eine Rechtsform aus dem Katalog, mit dem, was sie steuerlich nach sich zieht. */
export interface LegalFormInfo {
  name: string;
  /** Die abgeleitete Anlegerstellung. Leer heißt: aus der Rechtsform folgt sie nicht. */
  investor: InvestorType;
  note: string;
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

export type DisposalKind = 'sale' | 'scrapped' | 'repayment';

/** Die Fondsarten, an denen § 20 InvStG die Teilfreistellung festmacht. */
export type FundClass = '' | 'equity' | 'mixed' | 'real_estate' | 'foreign_real_estate' | 'other';

/** Die Anlegerstellung, an der § 20 Abs. 1 InvStG die Höhe des Satzes festmacht. */
export type InvestorType = '' | 'basic' | 'individual_business' | 'corporate' | 'mixed';

/** Was zu einem Anlagegut abgelegt wird, ohne gebucht zu werden. */
export type AssetDocumentKind =
  | 'contract'
  | 'invoice'
  | 'valuation'
  | 'registration'
  | 'insurance'
  | 'maintenance'
  | 'statement'
  | 'photo'
  | 'other';

export interface AssetDocument {
  id: number;
  assetId: number;
  kind: AssetDocumentKind;
  title?: string;
  fileName: string;
  mimeType: string;
  size: number;
  sha256: string;
  storedPath: string;
  documentDate?: string;
  /** Tag, an dem das Dokument abläuft — eine Police, eine Frist. */
  validUntil?: string;
  note?: string;
  createdAt: string;
}

export interface AssetDocumentKindInfo {
  kind: AssetDocumentKind;
  label: string;
}

export interface ExpiringAssetDocument {
  assetId: number;
  inventoryNumber: string;
  assetName: string;
  documentId: number;
  kind: AssetDocumentKind;
  kindLabel: string;
  title: string;
  validUntil: string;
}

export type AssetMovementKind =
  | 'transfer'
  | 'acquisition'
  | 'subsequent_cost'
  | 'cost_reduction'
  | 'depreciation'
  | 'special_depreciation'
  | 'impairment'
  | 'write_up'
  | 'maintenance'
  | 'income'
  | 'vorabpauschale'
  | 'disposal';

/**
 * Stückzahl in Zehntausendstel: 100 Anteile sind 1_000_000.
 *
 * Wie bei den Beträgen eine ganze Zahl, damit die Summe der Zu- und Abgänge
 * nicht vom Bestand abdriftet — Fondsanteile gibt es in Bruchteilen.
 */
export type Units = number;

/** Ein Stück in der Skalierung von {@link Units}. */
export const UNIT_SCALE = 10000;

/** Devisenkurse werden als Fremdwährungseinheiten je Euro mal einer Million geführt. */
export const RATE_SCALE = 1_000_000;

export interface AssetMovement {
  id: number;
  assetId: number;
  kind: AssetMovementKind;
  /** Konto, das diese Bewegung berührt — nach einer Umbuchung nicht das aktuelle. */
  account?: string;
  date: string;
  fiscalYear: number;
  /** Verändert die Anschaffungs- und Herstellungskosten. */
  costAmount: Cents;
  /** Verändert die kumulierten Abschreibungen. */
  depreciationAmount: Cents;
  journalEntryId?: number;
  entryNumber?: string;
  /** Stückzahl, die diese Bewegung bewegt: positiv beim Zugang, negativ beim Abgang. */
  quantity?: Units;
  /** Betrag, der nur steuerlich zählt — die Vorabpauschale wird nicht gebucht. */
  taxAmount?: Cents;
  /** Monate, um die diese Bewegung die Restnutzungsdauer verlängert. */
  lifeExtensionMonths?: number;
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
  /** Tag der Betriebsbereitschaft; ab hier läuft die AfA. Leer = mit der Anschaffung. */
  inServiceDate?: string;
  acquisitionCost: Cents;
  method: DepreciationMethod;
  usefulLifeMonths: number;
  poolYear?: number;
  /** Sonderabschreibung nach § 7g Abs. 5 EStG: Satz in Promille, höchstens 400. */
  specialPermille?: number;
  /** Jahre, auf die der Betrag gleichmäßig verteilt wird — eins bis fünf. */
  specialYears?: number;
  /** Aufwandskonto der Sonderabschreibung: 6242 für Fahrzeuge, sonst 6241. */
  specialAccount?: string;
  /** Pflichtangabe zu den Voraussetzungen des § 7g Abs. 6 EStG. */
  specialReason?: string;
  identifier?: string;
  /** Stückzahl des Zugangs. Null heißt: dieses Anlagegut wird nicht in Stück geführt. */
  quantity?: Units;
  /** Notierungswährung (ISO 4217). Leer heißt Euro. */
  currency?: string;
  /** Anschaffungskosten in der Notierungswährung. */
  foreignCost?: Cents;
  /** Fälligkeit einer Ausleihung. Entscheidet über § 256a Satz 2 HGB. */
  maturityDate?: string;
  /** Fondsart eines Investmentanteils. Leer heißt: kein Investmentanteil. */
  fundClass?: FundClass;
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
  documents?: AssetDocument[];

  // Abgeleitet vom Backend, nicht gespeichert.
  accountName?: string;
  cost: Cents;
  accumulated: Cents;
  bookValue: Cents;
  yearAmount: Cents;
  dueAmount: Cents;
  /** Noch fällige Sonderabschreibung des Geschäftsjahres. */
  specialDue: Cents;
  /** Gehaltene Stückzahl nach allen Bewegungen. */
  unitsHeld?: Units;
  /** Summe der über die Besitzzeit angesetzten Vorabpauschalen. */
  vorabpauschalen?: Cents;
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
  specialDue: Cents;
  dueCount: number;
}

export interface AssetScheduleYear {
  fiscalYear: number;
  months: number;
  method: DepreciationMethod;
  rateLabel: string;
  openingBookValue: Cents;
  amount: Cents;
  /** Sonderabschreibung des Jahres, getrennt geführt: eigenes Aufwandskonto. */
  specialAmount?: Cents;
  closingBookValue: Cents;
  note?: string;
  booked: Cents;
  due: Cents;
  specialBooked: Cents;
  specialDue: Cents;
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
  /** Anlagen im Bau und geleistete Anzahlungen: von hier wird umgebucht. */
  inProgress?: boolean;
  /** Grund und Boden und alles, was darauf steht — keine degressive AfA, keine Sonderabschreibung. */
  immovable?: boolean;
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
  /** Vielfaches des linearen Satzes in Tausendsteln: 3000 ist das Dreifache. */
  FactorPermille: number;
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
  /** Höchstsatz der Sonderabschreibung in Promille (§ 7g Abs. 5 EStG). */
  specialMaxPermille: number;
  /** Begünstigungszeitraum in Jahren: das Anschaffungsjahr und die vier folgenden. */
  specialPeriodYears: number;
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
  specialAccount?: string;
  specialPlanned: Cents;
  specialBooked: Cents;
  specialDue: Cents;
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
  /** Teil der Anschaffungskosten, der abgeht. Leer = alles. Nur bei Finanzanlagen. */
  costShare?: Cents;
  /** Derselbe Teilabgang in Stück. Hat Vorrang vor costShare. */
  quantity?: Units;
  taxTreatment?: TaxTreatment;
  taxRate?: TaxRate;
  settlement: Settlement;
  paymentAccount?: string;
  contactId?: number;
  note?: string;
}

export interface DisposalPreview {
  catchUpAmount: Cents;
  /** Im Abgangsjahr noch offene Sonderabschreibung, mit demselben Beleg nachgeholt. */
  specialCatchUp: Cents;
  catchUpLines?: JournalLine[];
  partial: boolean;
  costShare: Cents;
  /** Abgehende Stückzahl und der Bestand danach. */
  quantityShare?: Units;
  unitsRemaining?: Units;
  depreciationShare: Cents;
  bookValue: Cents;
  /** Buchgewinn positiv, Buchverlust negativ. */
  result: Cents;
  isGain: boolean;
  accounts: DisposalAccounts;
  lines: JournalLine[];
  gross: Cents;
  tax: Cents;
  /** Steuerliche Nebenrechnung eines Investmentanteils — sie ändert die Buchung nicht. */
  investment?: InvestmentTaxNote;
  warnings?: string[];
}

/** Die Teilfreistellung eines Fonds für einen Anleger (§ 20 InvStG). */
export interface PartialExemption {
  /** Steuerfreier Anteil in Promille: 800 sind 80 %. */
  permille: number;
  determined: boolean;
  source: string;
  explanation: string;
}

/** Was das InvStG neben der Buchung aus einem Betrag macht. */
export interface InvestmentTaxNote {
  fundClass: FundClass;
  fundClassLabel: string;
  exemption: PartialExemption;
  /** Steht, wenn sich kein Satz bestimmen lässt — und sagt warum. */
  exemptionError?: string;
  grossAmount: Cents;
  vorabpauschalen: Cents;
  exemptAmount: Cents;
  taxableAmount: Cents;
  explanation: string;
}

/** Die Vorabpauschale eines Kalenderjahres (§ 18 InvStG), mit jedem Schritt. */
export interface Vorabpauschale {
  year: number;
  basisReturn: Cents;
  growth: Cents;
  capped: boolean;
  distributions: Cents;
  monthsCounted: number;
  amount: Cents;
  accruedOn: string;
  explanation: string;
}

export interface InvestmentRules {
  fundClasses: { class: FundClass; label: string }[];
  investorTypes: { type: InvestorType; label: string }[];
  investorType: InvestorType;
  investorLabel: string;
  /** Woher die Anlegerstellung kommt: aus der Rechtsform oder aus einer Festlegung. */
  investorReason: string;
  legalForm: string;
  exemptions: {
    class: FundClass;
    label: string;
    permille: number;
    source?: string;
    explanation?: string;
    problem?: string;
  }[];
}

export interface DisposalResult {
  catchUpEntry?: JournalEntry;
  disposalEntry?: JournalEntry;
  asset: FixedAsset;
  message: string;
}

/** Was der Devisenkassamittelkurs eines Stichtags für eine Finanzanlage bedeutet. */
export interface CurrencyValuation {
  currency: string;
  foreignAmount: Cents;
  /** Anschaffungskurs, abgeleitet aus Fremdbetrag und Euro-Anschaffungskosten. */
  acquisitionRate: number;
  ratePerEuro: number;
  valueAtRate: Cents;
  bookValue: Cents;
  /** Greift § 256a Satz 2 HGB — Restlaufzeit höchstens ein Jahr, kein Deckel nach oben? */
  shortTerm: boolean;
  /** Unterschied zum Buchwert: negativ, wo der Kurs gefallen ist. */
  difference: Cents;
  /** Was daraus folgt — und mit welchem Betrag er tatsächlich gebucht werden dürfte. */
  proposal: 'impairment' | 'write_up' | 'none';
  proposedAmount: Cents;
  explanation: string;
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
  transfers: Cents;
  costClosing: Cents;
  depreciationOpening: Cents;
  depreciationYear: Cents;
  writeUpsYear: Cents;
  depreciationDisposal: Cents;
  depreciationTransfer: Cents;
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

// -------------------------------------------------------------
// Gründung: von der Beurkundung bis zur Eintragung
// -------------------------------------------------------------

/** Die Phase der Gründung. Sie folgt aus dem Eintragungsdatum, nicht umgekehrt. */
export type FoundationStage = 'vorgesellschaft' | 'eingetragen';

/** Bar- oder Sacheinlage. Der Unterschied entscheidet über die Mindesteinzahlung. */
export type ContributionKind = 'cash' | 'kind';

/** Ein Gesellschafter und der Geschäftsanteil, den er übernommen hat. */
export interface Shareholder {
  id: number;
  foundationId: number;
  name: string;
  /** Nennbetrag des übernommenen Geschäftsanteils. */
  shareCapital: Cents;
  /** Was darauf tatsächlich geleistet wurde. */
  paidIn: Cents;
  kind: ContributionKind;
}

export interface Foundation {
  id: number;
  /** Tag der notariellen Beurkundung. Mit ihm entsteht die Vorgesellschaft. */
  notarizedOn: string;
  /** Tag der Eintragung. Leer heißt: noch Vorgesellschaft. */
  registeredOn: string;
  registerCourt: string;
  registerNumber: string;
  shareCapital: Cents;
  /**
   * Gründungsaufwand laut Gesellschaftsvertrag. Er begrenzt die
   * Vorbelastungshaftung: was die Satzung der Gesellschaft auferlegt, ist
   * zulässig von ihr getragen.
   */
  foundationCostCap: Cents;
  shareholders: Shareholder[];
}

/** Was eine Rechtsform vor der Anmeldung verlangt. */
export interface FoundationRules {
  legalForm: string;
  minShareCapital: Cents;
  paidInPerShareQuota: number;
  paidInFloor: Cents;
  paidInFloorIsFullCapital: boolean;
  cashOnly: boolean;
  legalReserve: boolean;
  reference: string;
  note: string;
}

/** Der Anteil eines Gesellschafters an der Unterbilanzhaftung. */
export interface UnterbilanzShare {
  shareholderId: number;
  name: string;
  shareCapital: Cents;
  amount: Cents;
}

/**
 * Die Vorbelastungsrechnung: was die Gesellschafter der Gesellschaft schulden,
 * weil ihr Reinvermögen hinter dem Stammkapital zurückbleibt.
 */
export interface Unterbilanz {
  asOf: string;
  /** Erst ab der Eintragung steht die Zahl endgültig fest. */
  isFinal: boolean;
  shareCapital: Cents;
  assets: Cents;
  liabilities: Cents;
  netAssets: Cents;
  /** Rohe Unterdeckung, davon durch die Satzungsklausel gedeckt, und der Rest. */
  shortfall: Cents;
  covered: Cents;
  amount: Cents;
  shares: UnterbilanzShare[];
}

export interface ShareholderCheck {
  shareholderId: number;
  name: string;
  kind: ContributionKind;
  shareCapital: Cents;
  requiredPaidIn: Cents;
  paidIn: Cents;
  isSatisfied: boolean;
}

/** Reicht die geleistete Einlage für die Anmeldung zum Handelsregister? */
export interface AnmeldungCheck {
  legalForm: string;
  minShareCapital: Cents;
  shareCapital: Cents;
  requiredPaidIn: Cents;
  actualPaidIn: Cents;
  isSatisfied: boolean;
  /** Was fehlt, je Befund ein Satz mit Fundstelle. Leer heißt: es passt. */
  findings: string[];
  reference: string;
  shareholders: ShareholderCheck[];
}

/** Eine Pflicht aus der Gründung, mit Frist und Erledigung. */
export interface FoundationDuty {
  key: string;
  title: string;
  /** Leer, wo das Gesetz „unverzüglich" sagt statt einer Tagesfrist. */
  dueDate: string;
  deadline: string;
  reference: string;
  description: string;
  doneOn: string;
  isDone: boolean;
}

/** Ein Buchungsvorschlag der Gründung, vor der Freigabe. */
export interface FoundationPosting {
  title: string;
  date: string;
  description: string;
  reference: string;
  lines: JournalLine[];
  amount: Cents;
}

export interface FoundationPostingPreview {
  postings: FoundationPosting[];
  total: Cents;
  alreadyBooked: boolean;
  /** Was nicht gebucht wird, und warum. */
  skipped: string[];
}

/** Alles, was die Gründungsansicht braucht — in einem Aufruf. */
export interface FoundationState {
  /** Falsch, wenn die Rechtsform keine Kapitalgesellschaft ist. */
  applies: boolean;
  hasFoundation: boolean;
  legalForm: string;
  rules: FoundationRules;
  foundation?: Foundation;
  stage: FoundationStage;
  anmeldung?: AnmeldungCheck;
  unterbilanz?: Unterbilanz;
  duties: FoundationDuty[];
  postingsBooked: boolean;
}

// -------------------------------------------------------------
// Jahresabschluss: Geschäftsjahr, Saldenvortrag, Abschlussstand
// (internal/domain/fiscalyear.go, internal/service/closing_service.go)
// -------------------------------------------------------------

/**
 * Die vier Stände sind keine Abstufungen derselben Sache, sondern Vorgänge mit
 * verschiedenen Beteiligten: Aufstellung durch die Geschäftsführung (§ 242,
 * § 264 Abs. 1 HGB), Feststellung durch die Gesellschafter (§ 42a Abs. 2
 * GmbHG), Offenlegung gegenüber dem Bundesanzeiger (§ 325 HGB).
 */
export type FiscalYearStatus = 'open' | 'prepared' | 'adopted' | 'disclosed';

/** Das Geschäftsjahr als Entität: Zeitraum, Rumpfjahr, Abschlussstand. */
export interface FiscalYear {
  year: number;
  startDate: string;
  endDate: string;
  /** Rumpfgeschäftsjahr (§ 8b EStDV): kürzer als zwölf Monate. */
  isShort: boolean;
  status: FiscalYearStatus;
  preparedOn?: string;
  adoptedOn?: string;
  disclosedOn?: string;
  /** Welcher Gesellschafterbeschluss den Abschluss festgestellt hat. */
  adoptionNote?: string;
  /** Zeitpunkt des letzten Saldenvortrags in dieses Jahr. */
  carriedForwardAt?: string;
  /**
   * Die durchschnittliche Zahl der Arbeitnehmer ist das dritte Merkmal des
   * § 267 Abs. 1 HGB. Aus der Buchführung lässt sie sich nicht ableiten,
   * deshalb wird sie erfasst und nicht gerechnet.
   */
  averageEmployees: number;
  createdAt: string;
}

/** Alles, was die Abschlussansicht eines Jahres braucht — in einem Aufruf. */
export interface ClosingState {
  year: number;
  fiscalYear: FiscalYear;
  /** Erträge minus Aufwendungen der GuV-Konten; abgeleitet, nicht gebucht. */
  netIncome: Cents;
  /** Ohne Jahres-Festschreibung lässt sich der Abschluss nicht feststellen. */
  hasYearCommitment: boolean;
  committedUntil?: string;
  nextYear: number;
  carriedForward: boolean;
  /** Falsch, sobald im abgelaufenen Jahr nach dem Vortrag noch gebucht wurde. */
  carryForwardCurrent: boolean;
  carriedForwardAt?: string;
  /** Leer, wenn das Jahr offengelegt ist. */
  nextStatus?: FiscalYearStatus;
  canAdopt: boolean;
  blocker?: string;
}

/** Vortragsart: je eine Buchung gegen 9000, 9008 und 9009. */
export type CarryForwardKind = 'sachkonto' | 'debitor' | 'kreditor';

/**
 * Eine Zeile der Vortragsvorschau. Alle Beträge sind vorzeichenbehaftet in
 * Soll-Richtung: positiv ist ein Sollsaldo, negativ ein Habensaldo.
 */
export interface CarryForwardRow {
  account: string;
  name: string;
  kind: CarryForwardKind;
  closingBalance: Cents;
  carried: Cents;
  difference: Cents;
  /** Zahl der offenen Posten hinter einem Personenkonto. */
  openItems?: number;
  includesNetIncome?: boolean;
}

/** Der Stand des Saldenvortrags in ein Geschäftsjahr. */
export interface CarryForwardPreview {
  fromYear: number;
  toYear: number;
  /** Erster Tag des neuen Jahres, sonst der erste nicht festgeschriebene Tag. */
  bookingDate: string;
  deferred: boolean;
  rows: CarryForwardRow[];
  netIncome: Cents;
  resultAccount: string;
  resultAccountName: string;
  alreadyCarried: boolean;
  needsCorrection: boolean;
  /** Vortragswerte ohne zurücknehmbare Buchung: ein Lauf würde sie verdoppeln. */
  irreversible?: boolean;
  /** Das Vorjahr selbst trägt keinen Saldenvortrag, obwohl es einen bräuchte. */
  priorYearNotCarried?: boolean;
  /** Zahl der Buchungen, die ein Lauf erzeugt; höchstens drei. */
  entries: number;
  /** Probe auf die Bilanzidentität: Summe aller Vortragswerte, muss null sein. */
  balanceDifference: Cents;
  isBalanced: boolean;
}

// -------------------------------------------------------------
// Jahresabschluss: Bilanz und Gewinn- und Verlustrechnung
// (internal/domain/statement.go, internal/domain/sizeclass.go,
//  internal/ebilanz/mapping.go)
//
// Die Gliederung nach den §§ 266 und 275 HGB entsteht vollständig im Backend.
// Hier stehen nur die Formen, in denen sie ankommt: die Ansicht zeigt Zeilen
// an und rechnet an keiner Stelle nach — sonst gäbe es zwei Bilanzen.
// -------------------------------------------------------------

/**
 * Die Gliederungstiefe ist eine Rechtsfolge der Größenklasse und kein
 * Anzeigegeschmack: § 266 Abs. 1 Satz 3 HGB erlaubt der kleinen Gesellschaft
 * Buchstaben und römische Ziffern, Satz 4 der Kleinstgesellschaft allein die
 * Buchstaben.
 */
export type StatementDepth = 'full' | 'short' | 'letters';

/** Abschnitt, in dem eine Gliederungsposition steht. */
export type StatementSection = 'aktiva' | 'passiva' | 'guv' | 'statistisch';

/** Ein Konto unter einer Gliederungsposition — der Weg zurück zum Kontoblatt. */
export interface StatementAccount {
  number: string;
  name: string;
  positionId: string;
  /** Bezeichnung der SKR04-Position, nicht die der Gliederungszeile. */
  position: string;
  /** Erklärt eine Zuordnung, die dem Kontonamen widerspricht. */
  note?: string;
  amount: Cents;
  priorAmount: Cents;
}

/** Eine Gliederungsposition mit ihrem Wert. */
export interface StatementLine {
  /** Stabiler Schlüssel, z. B. "aktiva.A.II.3". */
  key: string;
  /** Ordnungszahl des Gesetzes: "A.", "II.", "3.", "a)". */
  ordinal: string;
  label: string;
  /** 1 Buchstabe, 2 römische Ziffer, 3 arabische Ziffer. */
  level: number;
  section: StatementSection;
  note?: string;
  /** Zwischensumme der Staffel (§ 275 Abs. 2 Nr. 15 und 17 HGB). */
  isSubtotal: boolean;
  /** Auffangposition ("sonstige …"). */
  isFallback: boolean;
  /**
   * Posten ohne Betrag in beiden Jahren; § 265 Abs. 8 HGB lässt ihn entfallen.
   * Die Entscheidung fällt im Backend, damit Ansicht, PDF und CSV dieselben
   * Zeilen zeigen.
   */
  omitted: boolean;
  amount: Cents;
  priorAmount: Cents;
  accounts?: StatementAccount[];
}

/** Was in einer Auffangposition gelandet ist. */
export interface FallbackCount {
  key: string;
  label: string;
  accounts: number;
  amount: Cents;
}

/** Ein Konto, das wegen seines Vorzeichens auf der Gegenposition steht. */
export interface SignSwitch {
  account: string;
  name: string;
  from: string;
  to: string;
  label: string;
  amount: Cents;
}

/** Was die Gliederung nicht oder nur mit Vorbehalt einordnen konnte. */
export interface AssignmentReport {
  /** Konten mit Saldo ohne Gliederungsposition; sie stehen in "Nicht zugeordnet". */
  unassigned: StatementAccount[];
  /** Saldo gegen die Richtung der Position, ohne Gegenposition. */
  wrongSign: StatementAccount[];
  signSwitches: SignSwitch[];
  fallbacks: FallbackCount[];
}

/** Die Gliederung eines Geschäftsjahres mit Vorjahresspalte. */
export interface Statement {
  fiscalYear: number;
  priorYear: number;
  hasPrior: boolean;
  depth: StatementDepth;
  assets: StatementLine[];
  liabilities: StatementLine[];
  income: StatementLine[];
  /** Konten der Klasse 9 — weder Bilanz noch GuV, aber sichtbar. */
  statistical: StatementLine[];
  assignment: AssignmentReport;

  totalAssets: Cents;
  totalAssetsPrior: Cents;
  totalLiabilities: Cents;
  totalLiabilitiesPrior: Cents;
  /** Bilanzsumme des § 267 Abs. 4a HGB: Posten A bis E der Aktivseite. */
  balanceSheetTotal: Cents;
  balanceSheetTotalPrior: Cents;

  netIncome: Cents;
  netIncomePrior: Cents;
  /** Nummer 1 der Staffel — das Merkmal "Umsatzerlöse" des § 267 HGB. */
  revenue: Cents;
  revenuePrior: Cents;
}

/** Eine Zeile der Restlaufzeitengliederung. */
export interface MaturityRow {
  key: string;
  label: string;
  total: Cents;
  upToOneYear: Cents;
  overOneYear: Cents;
  overFiveYears: Cents;
  items: number;
  /** Posten ohne Fälligkeit: ohne sie gibt es keine Restlaufzeit. */
  undated: Cents;
  note?: string;
}

/** Angabe unter der Bilanz nach § 268 Abs. 4 und 5 HGB. */
export interface MaturityTable {
  closingDate: string;
  rows: MaturityRow[];
  reference: string;
}

/** Ein Termin des Jahresabschlusses mit seiner Norm. */
export interface Deadline {
  key: string;
  title: string;
  dueDate: string;
  period: string;
  reference: string;
  description: string;
  fiscalYear: number;
  isDone: boolean;
  doneOn?: string;
}

/** Die Pflichtangaben des § 264 Abs. 1a HGB im Kopf des Abschlusses. */
export interface StatementHeader {
  companyName: string;
  legalForm: string;
  seat: string;
  registerCourt: string;
  registerNumber: string;
  fiscalYear: number;
  startDate: string;
  closingDate: string;
  priorYear: number;
  isShortYear: boolean;
  reference: string;
  /** Pflichtangaben, die in den Einstellungen fehlen. */
  missing: string[];
}

/** Größenklasse nach den §§ 267, 267a HGB. */
export type SizeClassKind = 'micro' | 'small' | 'medium' | 'large';

/** Die drei Merkmale des § 267 Abs. 1 HGB zu einem Stichtag. */
export interface SizeCriteria {
  balanceSheetTotal: Cents;
  revenue: Cents;
  employees: number;
}

/** Die Schwellenwerte einer Fassung, datiert nach dem Beginn des Jahres. */
export interface SizeThresholdSet {
  validFrom: string;
  reference: string;
  micro: SizeCriteria;
  small: SizeCriteria;
  medium: SizeCriteria;
}

/** Die Beurteilung eines einzelnen Abschlussstichtags (§ 267 Abs. 4 HGB). */
export interface SizeAssessment {
  year: number;
  closingDate: string;
  criteria: SizeCriteria;
  class: SizeClassKind;
  /** Die Merkmale, die für die Klasse sprechen — zwei von drei genügen. */
  met: string[];
  thresholds: SizeThresholdSet;
}

/** Die Folgen der Größenklasse, je mit ihrer Norm. */
export interface SizeObligations {
  depth: StatementDepth;
  depthReference: string;
  notesRequired: boolean;
  notesReference: string;
  managementReport: boolean;
  managementReportReference: string;
  auditRequired: boolean;
  auditReference: string;
  preparationMonths: number;
  preparationReference: string;
  disclosureMonths: number;
  disclosureReference: string;
  disclosureScope: string;
  disclosureScopeReference: string;
}

/** Die Einordnung eines Geschäftsjahres samt Begründung. */
export interface SizeClass {
  year: number;
  closingDate: string;
  class: SizeClassKind;
  criteria: SizeCriteria;
  current: SizeAssessment;
  prior?: SizeAssessment;
  /** Die Stichtage, die in die Zweijahresregel eingegangen sind. */
  history?: SizeAssessment[];
  /** § 267 Abs. 4 Satz 2 HGB: bei Neugründung gilt schon der erste Stichtag. */
  isFirstYear: boolean;
  reason: string;
  obligations: SizeObligations;
}

/** Der Jahresabschluss, wie ihn die Ansicht zeigt. */
export interface FinancialStatement {
  header: StatementHeader;
  statement: Statement;
  sizeClass: SizeClass;
  maturities: MaturityTable;
  deadlines: Deadline[];
}

/** Ein Konto mit Saldo, seine Gliederungsposition und sein Taxonomie-Element. */
export interface MappingRow {
  account: string;
  name: string;
  balance: Cents;
  positionKey: string;
  positionLabel: string;
  element: string;
  verified: boolean;
  /** Benennt, was fehlt, wenn etwas fehlt. */
  finding?: string;
}

/** Der Zuordnungsbericht vor dem E-Bilanz-Export. */
export interface MappingReport {
  fiscalYear: number;
  taxonomyVersion: string;
  taxonomyDate: string;
  taxonomyNote: string;
  rows: MappingRow[];
  /** Konten ohne Gliederungsposition oder ohne Taxonomie-Element. */
  blocking: MappingRow[];
  fallbacks: FallbackCount[];
  /** Elemente, deren Name noch gegen die amtliche Taxonomie zu prüfen ist. */
  unverified: number;
  canExport: boolean;
}
