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
  type: 'asset' | 'liability' | 'equity' | 'revenue' | 'expense' | 'statistical';
  category: string;
  subcategory?: string;
  kontenklasse?: number;
  kontenklasseName?: string;
  positionId?: string;
  posten?: string;
  balanceSide?: 'Aktiva' | 'Passiva' | 'GuV' | 'Statistisch' | string;
  hgbCode?: string;
  statementType?: 'Bilanz' | 'GuV' | 'Statistisch' | string;
  taxRate: number;
  hauptfunktion?: string;
  hauptfunktionDesc?: string;
  zusatzfunktion?: string;
  zusatzfunktionDesc?: string;
  abschlusszweck?: string;
  isRange?: boolean;
  rangeStart?: string;
  rangeEnd?: string;
  isReserved?: boolean;
  description: string;
  isActive: boolean;
  debitSum?: number;
  creditSum?: number;
  balance: number;
  bookingsCount?: number;
}

export interface AccountLedgerBooking {
  booking: BookingEntry;
  counterAccount: string;
  counterName: string;
  direction: 'SOLL' | 'HABEN' | 'SOLL/HABEN' | string;
  debitAmount: number;
  creditAmount: number;
  runningBalance: number;
}

export interface AccountLedger {
  account: Account;
  fiscalYear: number;
  openingBalance: number;
  totalDebit: number;
  totalCredit: number;
  closingBalance: number;
  bookingsCount: number;
  entries: AccountLedgerBooking[];
}

export interface SuSaClassSummary {
  kontenklasse: number;
  kontenklasseName: string;
  totalDebit: number;
  totalCredit: number;
  totalSaldoDebit: number;
  totalSaldoCredit: number;
  accountsCount: number;
  accounts: Account[];
}

export interface SuSaOverview {
  fiscalYear: number;
  totalDebit: number;
  totalCredit: number;
  totalSaldoDebit: number;
  totalSaldoCredit: number;
  isBalanced: boolean;
  difference: number;
  classes: SuSaClassSummary[];
}

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
  accounts: any[];
  hierarchy: Record<string, any>;
}

export interface BookingEntry {
  id: number;
  fiscalYear: number;
  bookingNumber: string;
  date: string;
  valueDate: string;
  description: string;
  debitAccount: string;
  debitAccountName?: string;
  creditAccount: string;
  creditAccountName?: string;
  amount: number;
  currency: string;
  exchangeRate: number;
  taxCode: string;
  taxAmount: number;
  receiptNumber: string;
  receiptHash: string;
  receiptPath: string;
  bankTxId?: number;
  previousHash: string;
  entryHash: string;
  isStorno: boolean;
  stornoForId?: number;
  createdAt: string;
}

export interface BankTransaction {
  id: number;
  fiscalYear?: number;
  accountIban: string;
  bookingDate: string;
  valueDate: string;
  amount: number;
  currency: string;
  counterpartyName: string;
  counterpartyIban: string;
  remittanceInfo: string;
  endToEndId: string;
  matchStatus: 'unmatched' | 'matched' | 'ignored';
  matchedBookingId?: number;
  suggestedAccount?: string;
  suggestedContact?: string;
}

export interface Contact {
  id: number;
  type: 'customer' | 'vendor';
  number: string;
  name: string;
  company: string;
  email: string;
  address: string;
  taxId: string;
  vatId: string;
  iban: string;
  bic: string;
  paymentTermsDays: number;
  openAmount: number;
  createdAt: string;
}

export interface InvoiceItem {
  position: number;
  description: string;
  quantity: number;
  unit: string;
  unitPrice: number;
  taxRate: number;
  totalNet: number;
  totalGross: number;
}

export interface Invoice {
  id: number;
  fiscalYear?: number;
  invoiceNumber: string;
  date: string;
  dueDate: string;
  contactId: number;
  contactName: string;
  items: InvoiceItem[];
  netAmount: number;
  taxAmount: number;
  grossAmount: number;
  currency: string;
  status: 'draft' | 'issued' | 'paid' | 'cancelled';
  zugferdXml?: string;
  pdfPath?: string;
  createdAt: string;
}

export interface FinancialSummary {
  totalRevenue: number;
  totalExpenses: number;
  netIncome: number;
  bankBalance: number;
  openReceivables: number;
  openPayables: number;
  cashflowHistory: {
    month: string;
    inflow: number;
    outflow: number;
    net: number;
  }[];
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
  fiscalYearStartMonth?: number;
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
  isSmallBusiness?: boolean;
  vatPeriod?: 'month' | 'quarter' | 'year' | 'exempt';
  taxationType?: 'IST' | 'SOLL';
}

export interface AuditLogEntry {
  id: number;
  timestamp: string;
  action: string;
  entityType: string;
  entityId: string;
  details: string;
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
