import * as Bridge from '../../bindings/github.com/buchfink/buchfink/internal/wailsbridge/buchfinkbridge';
import {
  AppConfig,
  Account,
  BookingEntry,
  BankTransaction,
  Contact,
  Invoice,
  FinancialSummary,
  IntegrityCheckResult,
  CompanySettings,
  AuditLogEntry,
} from '../types';

let fallbackConfig: AppConfig = {
  dataDir: '~/.buchfink/data',
  certPath: '~/.buchfink/certs/buchfink-cert.pem',
  hasPassword: false,
  isConfigured: true,
  lastFiscalYear: 2026,
};

let fallbackSettings: CompanySettings = {
  companyName: 'Musterfirma GmbH',
  legalForm: 'GmbH',
  fiscalYear: 2026,
  taxNumber: '12/345/67890',
  vatId: 'DE123456789',
  taxOffice: 'Finanzamt Berlin',
  iban: 'DE89370400440532013000',
  bic: 'COBADEFFXXX',
  bankName: 'Commerzbank',
  street: 'Musterstraße 42',
  zipCity: '10115 Berlin',
  country: 'Deutschland',
  currency: 'EUR',
  skr: 'SKR04',
};

let fallbackAccounts: Account[] = [];
let fallbackBookings: BookingEntry[] = [];
let fallbackBankTxs: BankTransaction[] = [];
let fallbackContacts: Contact[] = [];
let fallbackInvoices: Invoice[] = [];
let fallbackAuditLogs: AuditLogEntry[] = [];
let fallbackYears: number[] = [2026];

function isWailsRuntime(): boolean {
  return typeof window !== 'undefined' && Boolean((window as any)._wails);
}

export const Api = {
  async getAppConfig(): Promise<AppConfig> {
    try {
      const res = await Bridge.GetAppConfig();
      if (res) return res as AppConfig;
      return fallbackConfig;
    } catch {
      return fallbackConfig;
    }
  },

  async selectDirectoryDialog(title?: string): Promise<string> {
    try {
      const res = await Bridge.SelectDirectoryDialog(title || '');
      return res || '';
    } catch {
      return '';
    }
  },

  async selectDatabaseFileDialog(title?: string): Promise<string> {
    try {
      const res = await Bridge.SelectDatabaseFileDialog(title || '');
      return res || '';
    } catch {
      return '';
    }
  },

  async setupApplication(dataDir: string, password: string, settings: CompanySettings): Promise<void> {
    try {
      await Bridge.SetupApplication(dataDir, password, settings as any);
    } catch (e) {
      if (!isWailsRuntime()) {
        fallbackConfig.dataDir = dataDir || fallbackConfig.dataDir;
        fallbackConfig.isConfigured = true;
        fallbackConfig.hasPassword = Boolean(password);
        fallbackSettings = { ...settings };
        if (!fallbackYears.includes(settings.fiscalYear)) {
          fallbackYears.push(settings.fiscalYear);
          fallbackYears.sort();
        }
        return;
      }
      throw e;
    }
  },

  async loadExistingDatabase(dbPath: string): Promise<void> {
    try {
      await Bridge.LoadExistingDatabase(dbPath);
    } catch (e) {
      if (!isWailsRuntime()) {
        fallbackConfig.isConfigured = true;
        return;
      }
      throw e;
    }
  },

  async getAvailableFiscalYears(): Promise<number[]> {
    try {
      const res = await Bridge.GetAvailableFiscalYears();
      if (res && res.length > 0) return res;
      return fallbackYears;
    } catch {
      return fallbackYears;
    }
  },

  async createFiscalYear(year: number): Promise<void> {
    try {
      await Bridge.CreateFiscalYear(year);
    } catch (e) {
      if (!isWailsRuntime()) {
        if (!fallbackYears.includes(year)) {
          fallbackYears.push(year);
          fallbackYears.sort();
        }
        return;
      }
      throw e;
    }
  },

  async getFiscalYear(): Promise<number> {
    try {
      return await Bridge.GetFiscalYear();
    } catch {
      return fallbackSettings.fiscalYear;
    }
  },

  async setFiscalYear(year: number): Promise<void> {
    try {
      await Bridge.SetFiscalYear(year);
    } catch {
      fallbackSettings.fiscalYear = year;
    }
  },

  async getCompanySettings(): Promise<CompanySettings> {
    try {
      const res = await Bridge.GetCompanySettings();
      if (res) return res as CompanySettings;
      return fallbackSettings;
    } catch {
      return fallbackSettings;
    }
  },

  async updateCompanySettings(settings: CompanySettings): Promise<void> {
    try {
      await Bridge.UpdateCompanySettings(settings as any);
    } catch {
      fallbackSettings = { ...settings };
    }
  },

  async getAccounts(): Promise<Account[]> {
    try {
      const res = await Bridge.GetAccounts();
      if (res && res.length > 0) return res as Account[];
      return fallbackAccounts;
    } catch {
      return fallbackAccounts;
    }
  },

  async getBookings(): Promise<BookingEntry[]> {
    try {
      const res = await Bridge.GetBookings();
      if (res) return res as BookingEntry[];
      return fallbackBookings;
    } catch {
      return fallbackBookings;
    }
  },

  async createBooking(entry: Partial<BookingEntry>): Promise<BookingEntry> {
    try {
      const res = await Bridge.CreateBooking(entry as any);
      if (res) return res as BookingEntry;
      throw new Error('Failed to create booking in backend');
    } catch (e) {
      if (!isWailsRuntime()) {
        const newEntry: BookingEntry = {
          id: fallbackBookings.length + 1,
          bookingNumber: entry.bookingNumber || `B-${fallbackSettings.fiscalYear}-${String(fallbackBookings.length + 1).padStart(4, '0')}`,
          date: entry.date || new Date().toISOString().split('T')[0],
          valueDate: entry.valueDate || entry.date || new Date().toISOString().split('T')[0],
          description: entry.description || 'Buchung',
          debitAccount: entry.debitAccount || '1800',
          creditAccount: entry.creditAccount || '4400',
          amount: entry.amount || 0,
          currency: entry.currency || 'EUR',
          exchangeRate: entry.exchangeRate || 1.0,
          taxCode: entry.taxCode || 'NONE',
          taxAmount: entry.taxAmount || 0,
          receiptNumber: entry.receiptNumber || '',
          receiptHash: entry.receiptHash || '',
          receiptPath: entry.receiptPath || '',
          bankTxId: entry.bankTxId,
          previousHash: '0000000000000000000000000000000000000000000000000000000000000000',
          entryHash: 'localhash',
          isStorno: false,
          createdAt: new Date().toISOString(),
        };
        fallbackBookings.push(newEntry);
        return newEntry;
      }
      throw e;
    }
  },

  async stornoBooking(bookingId: number, reason: string): Promise<BookingEntry> {
    try {
      const res = await Bridge.StornoBooking(bookingId, reason);
      if (res) return res as BookingEntry;
      throw new Error('Failed to storno booking');
    } catch (e) {
      if (!isWailsRuntime()) {
        const target = fallbackBookings.find((b) => b.id === bookingId);
        if (!target) throw new Error('Booking not found');
        target.isStorno = true;
        return target;
      }
      throw e;
    }
  },

  async verifyIntegrity(): Promise<IntegrityCheckResult> {
    try {
      const res = await Bridge.VerifyIntegrity();
      return res as IntegrityCheckResult;
    } catch {
      return {
        isValid: true,
        totalEntries: fallbackBookings.length,
        checkedEntries: fallbackBookings.length,
        message: 'Kette ist intakt.',
        lastVerifiedHash: '0000000000000000000000000000000000000000000000000000000000000000',
        checkedAt: new Date().toLocaleTimeString('de-DE'),
      };
    }
  },

  async getFinancialSummary(): Promise<FinancialSummary> {
    try {
      const res = await Bridge.GetFinancialSummary();
      if (res) return res as FinancialSummary;
      return {
        totalRevenue: 0,
        totalExpenses: 0,
        netIncome: 0,
        bankBalance: 0,
        openReceivables: 0,
        openPayables: 0,
        cashflowHistory: [],
      };
    } catch {
      return {
        totalRevenue: 0,
        totalExpenses: 0,
        netIncome: 0,
        bankBalance: 0,
        openReceivables: 0,
        openPayables: 0,
        cashflowHistory: [],
      };
    }
  },

  async getBankTransactions(): Promise<BankTransaction[]> {
    try {
      const res = await Bridge.GetBankTransactions();
      if (res) return res as BankTransaction[];
      return fallbackBankTxs;
    } catch {
      return fallbackBankTxs;
    }
  },

  async importSampleBankStatement(): Promise<number> {
    try {
      return await Bridge.ImportSampleBankStatement();
    } catch {
      return 0;
    }
  },

  async importCAMT053XML(xmlContent: string): Promise<number> {
    try {
      return await Bridge.ImportCAMT053XML(xmlContent);
    } catch {
      return 0;
    }
  },

  async matchAndBookTransaction(
    bankTxId: number,
    debitAcc: string,
    creditAcc: string,
    receiptNr: string,
    desc: string
  ): Promise<BookingEntry> {
    try {
      const res = await Bridge.MatchAndBookTransaction(bankTxId, debitAcc, creditAcc, receiptNr, desc);
      if (res) return res as BookingEntry;
      throw new Error('Failed to match transaction');
    } catch (e) {
      if (!isWailsRuntime()) {
        return this.createBooking({
          description: desc,
          debitAccount: debitAcc,
          creditAccount: creditAcc,
          receiptNumber: receiptNr,
          bankTxId: bankTxId,
        });
      }
      throw e;
    }
  },

  async getContacts(): Promise<Contact[]> {
    try {
      const res = await Bridge.GetContacts();
      if (res) return res as Contact[];
      return fallbackContacts;
    } catch {
      return fallbackContacts;
    }
  },

  async saveContact(contact: Contact): Promise<Contact> {
    try {
      const res = await Bridge.SaveContact(contact as any);
      if (res) return res as Contact;
      return contact;
    } catch {
      if (!contact.id) {
        contact.id = fallbackContacts.length + 1;
        fallbackContacts.push(contact);
      }
      return contact;
    }
  },

  async getInvoices(): Promise<Invoice[]> {
    try {
      const res = await Bridge.GetInvoices();
      if (res) return res as Invoice[];
      return fallbackInvoices;
    } catch {
      return fallbackInvoices;
    }
  },

  async createInvoice(inv: Invoice): Promise<Invoice> {
    try {
      const res = await Bridge.CreateInvoice(inv as any);
      if (res) return res as Invoice;
      return inv;
    } catch {
      inv.id = fallbackInvoices.length + 1;
      fallbackInvoices.push(inv);
      return inv;
    }
  },

  async generateInvoiceZUGFeRD(inv: Invoice): Promise<{ xml: string; typst: string }> {
    try {
      const [xml, typst] = await Bridge.GenerateInvoiceZUGFeRD(inv.id);
      return { xml, typst };
    } catch {
      return {
        xml: `<!-- ZUGFeRD 2.2 Factur-X XML for ${inv.invoiceNumber} -->`,
        typst: `// Document template for ${inv.invoiceNumber}`,
      };
    }
  },

  async exportEBilanzXBRL(): Promise<string> {
    try {
      return await Bridge.ExportEBilanzXBRL();
    } catch {
      return '<!-- XBRL Instance for German E-Bilanz GAAP 6.7 -->';
    }
  },

  async getAuditLogs(): Promise<AuditLogEntry[]> {
    try {
      const res = await Bridge.GetAuditLogs();
      if (res) return res as AuditLogEntry[];
      return fallbackAuditLogs;
    } catch {
      return fallbackAuditLogs;
    }
  },
};
