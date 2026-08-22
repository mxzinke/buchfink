// SPDX-FileCopyrightText: 2026 Maximilian Pfennig
// SPDX-License-Identifier: EUPL-1.2

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
  documentHash?: string;
  documentPath?: string;
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
  bookingDate: string;
  documentDate: string;
  serviceDateFrom: string;
  serviceDateTo: string;
  documentNumber: string;
  documentHash?: string;
  documentPath?: string;
  description: string;
  taxTreatment: TaxTreatment;
  positions: ReceiptPosition[];
  settlement: Settlement;
  paymentAccount?: string;
  currency?: string;
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
  openAmount: Cents;
  createdAt: string;
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
  isSmallBusiness: boolean;
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
