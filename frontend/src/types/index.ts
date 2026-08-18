export interface Account {
  id: number;
  number: string;
  name: string;
  type: 'asset' | 'liability' | 'revenue' | 'expense';
  category: string;
  taxRate: number;
  description: string;
  isActive: boolean;
  balance: number;
}

export interface BookingEntry {
  id: number;
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
}

export interface AuditLogEntry {
  id: number;
  timestamp: string;
  action: string;
  entityType: string;
  entityId: string;
  details: string;
}
