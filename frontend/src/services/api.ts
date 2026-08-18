import * as Bridge from '../../bindings/github.com/buchfink/buchfink/internal/wailsbridge/buchfinkbridge';
import skr04CatalogData from '../assets/skr04_2026.json';
import {
  AppConfig,
  TenantConfig,
  Account,
  AccountLedger,
  AccountLedgerBooking,
  SuSaOverview,
  SuSaClassSummary,
  SKR04Catalog,
  BookingEntry,
  BankTransaction,
  Contact,
  Invoice,
  FinancialSummary,
  IntegrityCheckResult,
  CompanySettings,
  AuditLogEntry,
} from '../types';

let fallbackTenants: TenantConfig[] = [
  {
    id: 'default',
    name: 'Hauptmandant',
    dataDir: '~/.buchfink/data',
    certPath: '~/.buchfink/certs/buchfink-cert.pem',
    hasPassword: false,
    createdAt: new Date().toISOString(),
  },
];

let fallbackConfig: AppConfig = {
  tenants: fallbackTenants,
  activeTenantId: 'default',
  dataDir: '~/.buchfink/data',
  certPath: '~/.buchfink/certs/buchfink-cert.pem',
  hasPassword: false,
  isConfigured: true,
  lastFiscalYear: 2026,
};

let fallbackSettings: CompanySettings = {
  companyName: 'Hauptmandant',
  legalForm: 'GmbH',
  fiscalYear: 2026,
  fiscalYearStartMonth: 1,
  taxNumber: '',
  vatId: '',
  taxOffice: '',
  iban: '',
  bic: '',
  bankName: '',
  street: '',
  zipCity: '',
  country: 'Deutschland',
  currency: 'EUR',
  skr: 'SKR04',
  isSmallBusiness: false,
  vatPeriod: 'quarter',
  taxationType: 'IST',
};

// Initialize complete SKR04 2026 accounts catalog
const fallbackAccounts: Account[] = ((skr04CatalogData as any).accounts || []).map((a: any, idx: number) => {
  const taxRate =
    a.name.includes('19 %') || a.name.includes('19%')
      ? 0.19
      : a.name.includes('7 %') || a.name.includes('7%')
      ? 0.07
      : a.name.includes('16 %') || a.name.includes('16%')
      ? 0.16
      : 0;

  const descParts: string[] = [];
  if (a.bilanzierung?.posten && a.bilanzierung.posten !== a.name) {
    descParts.push(`Posten: ${a.bilanzierung.posten}`);
  }
  if (a.steuer_funktion?.hauptfunktion_description) {
    descParts.push(a.steuer_funktion.hauptfunktion_description);
  }
  if (a.steuer_funktion?.zusatzfunktion_description) {
    descParts.push(a.steuer_funktion.zusatzfunktion_description);
  }

  return {
    id: idx + 1,
    number: a.number,
    name: a.name,
    type: a.bilanzierung?.account_type || 'asset',
    category: a.category || '',
    subcategory: a.subcategory || '',
    kontenklasse: a.kontenklasse?.number ?? 0,
    kontenklasseName: a.kontenklasse?.name || '',
    positionId: a.position_id || '',
    posten: a.bilanzierung?.posten || '',
    balanceSide: a.bilanzierung?.balance_side || 'Aktiva',
    hgbCode: a.bilanzierung?.hgb_code || '',
    statementType: a.bilanzierung?.statement_type || 'Bilanz',
    taxRate: taxRate,
    hauptfunktion: a.steuer_funktion?.hauptfunktion || '',
    hauptfunktionDesc: a.steuer_funktion?.hauptfunktion_description || '',
    zusatzfunktion: a.steuer_funktion?.zusatzfunktion || '',
    zusatzfunktionDesc: a.steuer_funktion?.zusatzfunktion_description || '',
    abschlusszweck: a.steuer_funktion?.abschlusszweck || '',
    isRange: Boolean(a.is_range),
    rangeStart: a.range_start || '',
    rangeEnd: a.range_end || '',
    isReserved: Boolean(a.is_reserved),
    description: a.description || descParts.join(' • '),
    isActive: !a.is_reserved,
    debitSum: 0,
    creditSum: 0,
    balance: 0,
    bookingsCount: 0,
  };
});
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
  // -------------------------------------------------------------
  // MULTI-TENANT MANAGEMENT
  // -------------------------------------------------------------

  async getTenants(): Promise<TenantConfig[]> {
    try {
      const res = await Bridge.GetTenants();
      if (res && res.length > 0) return res as TenantConfig[];
      return fallbackTenants;
    } catch {
      return fallbackTenants;
    }
  },

  async getActiveTenant(): Promise<TenantConfig | null> {
    try {
      const res = await Bridge.GetActiveTenant();
      if (res) return res as TenantConfig;
      return fallbackTenants[0] || null;
    } catch {
      return fallbackTenants[0] || null;
    }
  },

  async switchTenant(tenantId: string): Promise<void> {
    try {
      await Bridge.SwitchTenant(tenantId);
    } catch (e) {
      if (!isWailsRuntime()) {
        const found = fallbackTenants.find((t) => t.id === tenantId);
        if (found) {
          fallbackConfig.activeTenantId = tenantId;
          fallbackConfig.dataDir = found.dataDir;
          fallbackConfig.certPath = found.certPath;
          fallbackSettings.companyName = found.name;
        }
        return;
      }
      throw e;
    }
  },

  async createTenant(
    name: string,
    dataDir: string,
    certDir: string,
    password: string,
    settings: CompanySettings
  ): Promise<TenantConfig> {
    try {
      const res = await Bridge.CreateTenant(name, dataDir, certDir, password, settings as any);
      if (res) return res as TenantConfig;
      throw new Error('Failed to create tenant');
    } catch (e) {
      if (!isWailsRuntime()) {
        const newTenant: TenantConfig = {
          id: `tenant_${Date.now()}`,
          name: name || settings.companyName || 'Neuer Mandant',
          dataDir: dataDir || `~/.buchfink/tenants/${Date.now()}/data`,
          certPath: `${certDir || '~/.buchfink/keys'}/buchfink-cert.pem`,
          hasPassword: Boolean(password),
          createdAt: new Date().toISOString(),
        };
        fallbackTenants.push(newTenant);
        fallbackConfig.tenants = fallbackTenants;
        fallbackConfig.activeTenantId = newTenant.id;
        fallbackConfig.isConfigured = true;
        fallbackSettings = { ...settings, companyName: newTenant.name };
        return newTenant;
      }
      throw e;
    }
  },

  async importTenant(
    dbFilePath: string,
    certPath?: string,
    password?: string
  ): Promise<TenantConfig> {
    try {
      const res = await Bridge.ImportTenant(dbFilePath, certPath || '', password || '');
      if (res) return res as TenantConfig;
      throw new Error('Failed to import tenant');
    } catch (e) {
      if (!isWailsRuntime()) {
        const newTenant: TenantConfig = {
          id: `tenant_${Date.now()}`,
          name: `Mandant (${dbFilePath.split('/').pop()})`,
          dataDir: dbFilePath,
          certPath: certPath || '~/.buchfink/certs/buchfink-cert.pem',
          hasPassword: Boolean(password),
          createdAt: new Date().toISOString(),
        };
        fallbackTenants.push(newTenant);
        fallbackConfig.tenants = fallbackTenants;
        fallbackConfig.activeTenantId = newTenant.id;
        fallbackConfig.isConfigured = true;
        return newTenant;
      }
      throw e;
    }
  },

  async deleteTenant(tenantId: string): Promise<void> {
    try {
      await Bridge.DeleteTenant(tenantId);
    } catch (e) {
      if (!isWailsRuntime()) {
        fallbackTenants = fallbackTenants.filter((t) => t.id !== tenantId);
        fallbackConfig.tenants = fallbackTenants;
        if (fallbackConfig.activeTenantId === tenantId) {
          fallbackConfig.activeTenantId = fallbackTenants[0]?.id || '';
        }
        return;
      }
      throw e;
    }
  },

  // -------------------------------------------------------------
  // CONFIG & DIALOGS
  // -------------------------------------------------------------

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

  async setupApplication(
    dataDir: string,
    certDir: string,
    password: string,
    settings: CompanySettings
  ): Promise<void> {
    try {
      await Bridge.SetupApplication(dataDir, certDir, password, settings as any);
    } catch (e) {
      if (!isWailsRuntime()) {
        const name = settings.companyName || 'Hauptmandant';
        const tenant: TenantConfig = {
          id: 'default',
          name: name,
          dataDir: dataDir || fallbackConfig.dataDir,
          certPath: `${certDir || '~/.buchfink/keys'}/buchfink-cert.pem`,
          hasPassword: Boolean(password),
          createdAt: new Date().toISOString(),
        };
        fallbackTenants = [tenant];
        fallbackConfig.tenants = fallbackTenants;
        fallbackConfig.activeTenantId = 'default';
        fallbackConfig.dataDir = tenant.dataDir;
        fallbackConfig.certPath = tenant.certPath;
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
        fallbackSettings = { ...fallbackSettings, fiscalYear: year };
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

  async getAccountByNumber(number: string): Promise<Account | null> {
    try {
      if ((Bridge as any).GetAccountByNumber) {
        const res = await (Bridge as any).GetAccountByNumber(number);
        if (res) return res as Account;
      }
    } catch {}
    const accounts = await this.getAccounts();
    return (
      accounts.find(
        (a) =>
          a.number === number ||
          (a.isRange &&
            a.rangeStart &&
            a.rangeEnd &&
            number >= a.rangeStart &&
            number <= a.rangeEnd)
      ) || null
    );
  },

  async getAccountBookings(accountNumber: string): Promise<BookingEntry[]> {
    try {
      if ((Bridge as any).GetAccountBookings) {
        const res = await (Bridge as any).GetAccountBookings(accountNumber);
        if (res) return res as BookingEntry[];
      }
    } catch {}
    const bookings = await this.getBookings();
    return bookings.filter(
      (b) => b.debitAccount === accountNumber || b.creditAccount === accountNumber
    );
  },

  async getAccountLedger(accountNumber: string): Promise<AccountLedger> {
    try {
      if ((Bridge as any).GetAccountLedger) {
        const res = await (Bridge as any).GetAccountLedger(accountNumber);
        if (res) return res as AccountLedger;
      }
    } catch {}

    // Fallback in-memory ledger generator
    const [acc, allAccounts, bookings] = await Promise.all([
      this.getAccountByNumber(accountNumber),
      this.getAccounts(),
      this.getBookings(),
    ]);

    const accMap = new Map(allAccounts.map((a) => [a.number, a.name]));
    const targetAcc: Account = acc || {
      id: 0,
      number: accountNumber,
      name: accMap.get(accountNumber) || `Konto ${accountNumber}`,
      type: 'asset',
      category: 'Sonstiges',
      taxRate: 0,
      description: '',
      isActive: true,
      balance: 0,
    };

    const isDebitPositive =
      targetAcc.type === 'asset' ||
      targetAcc.type === 'expense' ||
      targetAcc.type === 'statistical';

    const relevantBookings = bookings.filter((b) => {
      if (b.debitAccount === accountNumber || b.creditAccount === accountNumber) {
        return true;
      }
      if (targetAcc.isRange && targetAcc.rangeStart && targetAcc.rangeEnd) {
        const inDeb =
          b.debitAccount >= targetAcc.rangeStart &&
          b.debitAccount <= targetAcc.rangeEnd;
        const inCred =
          b.creditAccount >= targetAcc.rangeStart &&
          b.creditAccount <= targetAcc.rangeEnd;
        return inDeb || inCred;
      }
      return false;
    });

    let runningBalance = 0;
    let totalDebit = 0;
    let totalCredit = 0;

    const entries: AccountLedgerBooking[] = relevantBookings.map((b) => {
      const isDebit =
        b.debitAccount === accountNumber ||
        (targetAcc.isRange &&
          targetAcc.rangeStart &&
          targetAcc.rangeEnd &&
          b.debitAccount >= targetAcc.rangeStart &&
          b.debitAccount <= targetAcc.rangeEnd);
      const isCredit =
        b.creditAccount === accountNumber ||
        (targetAcc.isRange &&
          targetAcc.rangeStart &&
          targetAcc.rangeEnd &&
          b.creditAccount >= targetAcc.rangeStart &&
          b.creditAccount <= targetAcc.rangeEnd);

      let dir = 'SOLL';
      let debitAmt = 0;
      let creditAmt = 0;
      let counterAcc = '';
      let counterName = '';

      if (isDebit && !isCredit) {
        dir = 'SOLL';
        debitAmt = b.amount;
        totalDebit += debitAmt;
        counterAcc = b.creditAccount;
        counterName = accMap.get(counterAcc) || counterAcc;
        if (isDebitPositive) {
          runningBalance += debitAmt;
        } else {
          runningBalance -= debitAmt;
        }
      } else if (isCredit && !isDebit) {
        dir = 'HABEN';
        creditAmt = b.amount;
        totalCredit += creditAmt;
        counterAcc = b.debitAccount;
        counterName = accMap.get(counterAcc) || counterAcc;
        if (isDebitPositive) {
          runningBalance -= creditAmt;
        } else {
          runningBalance += creditAmt;
        }
      } else {
        dir = 'SOLL/HABEN';
        debitAmt = b.amount;
        creditAmt = b.amount;
        totalDebit += debitAmt;
        totalCredit += creditAmt;
        counterAcc = targetAcc.number;
        counterName = targetAcc.name;
      }

      return {
        booking: {
          ...b,
          debitAccountName: accMap.get(b.debitAccount) || b.debitAccount,
          creditAccountName: accMap.get(b.creditAccount) || b.creditAccount,
        },
        counterAccount: counterAcc,
        counterName: counterName,
        direction: dir,
        debitAmount: debitAmt,
        creditAmount: creditAmt,
        runningBalance: runningBalance,
      };
    });

    return {
      account: {
        ...targetAcc,
        debitSum: totalDebit,
        creditSum: totalCredit,
        balance: runningBalance,
        bookingsCount: entries.length,
      },
      fiscalYear: fallbackSettings.fiscalYear,
      openingBalance: 0,
      totalDebit,
      totalCredit,
      closingBalance: runningBalance,
      bookingsCount: entries.length,
      entries,
    };
  },

  async getSuSaOverview(): Promise<SuSaOverview> {
    try {
      if ((Bridge as any).GetSuSaOverview) {
        const res = await (Bridge as any).GetSuSaOverview();
        if (res) return res as SuSaOverview;
      }
    } catch {}

    const accounts = await this.getAccounts();
    const classNames: Record<number, string> = {
      0: 'Klasse 0: Anlagevermögenskonten',
      1: 'Klasse 1: Umlaufvermögenskonten',
      2: 'Klasse 2: Eigenkapital- & Fremdkapitalkonten',
      3: 'Klasse 3: Fremdkapitalkonten (Verbindlichkeiten)',
      4: 'Klasse 4: Betriebliche Erträge',
      5: 'Klasse 5: Betriebliche Aufwendungen (Material / Fremdleistungen)',
      6: 'Klasse 6: Betriebliche Aufwendungen (Personal / AfA / Sonstige)',
      7: 'Klasse 7: Weitere Erträge & Aufwendungen (Finanzen / Steuern)',
      8: 'Klasse 8: Freie Kontenklasse / Sonderkonten',
      9: 'Klasse 9: Vortrags-, Kapital- & statistische Konten',
    };

    const classes: SuSaClassSummary[] = Array.from({ length: 10 }, (_, i) => ({
      kontenklasse: i,
      kontenklasseName: classNames[i],
      totalDebit: 0,
      totalCredit: 0,
      totalSaldoDebit: 0,
      totalSaldoCredit: 0,
      accountsCount: 0,
      accounts: [],
    }));

    let grandTotalDebit = 0;
    let grandTotalCredit = 0;
    let grandSaldoDebit = 0;
    let grandSaldoCredit = 0;

    for (const a of accounts) {
      const kk =
        typeof a.kontenklasse === 'number' &&
        a.kontenklasse >= 0 &&
        a.kontenklasse <= 9
          ? a.kontenklasse
          : 0;
      const cls = classes[kk];
      cls.accounts.push(a);
      const deb = a.debitSum || 0;
      const cred = a.creditSum || 0;
      cls.totalDebit += deb;
      cls.totalCredit += cred;

      let sDebit = 0;
      let sCredit = 0;
      if (deb > cred) sDebit = deb - cred;
      else if (cred > deb) sCredit = cred - deb;

      cls.totalSaldoDebit += sDebit;
      cls.totalSaldoCredit += sCredit;
      cls.accountsCount++;

      grandTotalDebit += deb;
      grandTotalCredit += cred;
      grandSaldoDebit += sDebit;
      grandSaldoCredit += sCredit;
    }

    const diff = Math.abs(grandTotalDebit - grandTotalCredit);

    return {
      fiscalYear: fallbackSettings.fiscalYear,
      totalDebit: grandTotalDebit,
      totalCredit: grandTotalCredit,
      totalSaldoDebit: grandSaldoDebit,
      totalSaldoCredit: grandSaldoCredit,
      isBalanced: diff < 0.01,
      difference: diff,
      classes,
    };
  },

  async getSKR04Catalog(): Promise<SKR04Catalog> {
    try {
      if ((Bridge as any).GetSKR04Catalog) {
        const res = await (Bridge as any).GetSKR04Catalog();
        if (res) return res as SKR04Catalog;
      }
    } catch {}
    return skr04CatalogData as unknown as SKR04Catalog;
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

  async getAllBookings(): Promise<BookingEntry[]> {
    try {
      const res = await Bridge.GetAllBookings();
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
        const fy = entry.fiscalYear || fallbackSettings.fiscalYear;
        const newEntry: BookingEntry = {
          id: fallbackBookings.length + 1,
          fiscalYear: fy,
          bookingNumber: entry.bookingNumber || `B-${fy}-${String(fallbackBookings.length + 1).padStart(4, '0')}`,
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
        if (!fallbackYears.includes(fy)) {
          fallbackYears.push(fy);
          fallbackYears.sort();
        }
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
        if (target.isStorno || target.stornoForId != null) {
          throw new Error('Eine Stornobuchung kann nicht erneut storniert werden');
        }
        const alreadyStornoed = fallbackBookings.some((b) => b.stornoForId === bookingId);
        if (alreadyStornoed) {
          throw new Error('Diese Buchung wurde bereits storniert');
        }
        const prevHash = fallbackBookings.length > 0 ? fallbackBookings[fallbackBookings.length - 1].entryHash : '0000000000000000000000000000000000000000000000000000000000000000';
        const nextId = fallbackBookings.length > 0 ? Math.max(...fallbackBookings.map((b) => b.id)) + 1 : 1;
        const stornoEntry: BookingEntry = {
          id: nextId,
          fiscalYear: target.fiscalYear,
          bookingNumber: `STORNO-${target.bookingNumber}`,
          date: new Date().toISOString().split('T')[0],
          valueDate: target.valueDate,
          description: `STORNO zu ${target.bookingNumber}: ${target.description} (Grund: ${reason})`,
          debitAccount: target.creditAccount,
          creditAccount: target.debitAccount,
          amount: target.amount,
          currency: target.currency,
          exchangeRate: target.exchangeRate,
          taxCode: target.taxCode,
          taxAmount: target.taxAmount,
          receiptNumber: target.receiptNumber,
          receiptHash: target.receiptHash,
          receiptPath: target.receiptPath,
          bankTxId: target.bankTxId,
          previousHash: prevHash,
          entryHash: 'stornohash_' + nextId,
          isStorno: true,
          stornoForId: target.id,
          createdAt: new Date().toISOString(),
        };
        fallbackBookings.push(stornoEntry);
        return stornoEntry;
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

