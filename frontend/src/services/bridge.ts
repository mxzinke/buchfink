import { Call } from '@wailsio/runtime';

/**
 * Aufruf der Go-Methoden über die Wails-Laufzeit.
 *
 * Wails kann Bindings generieren, die dann unter `frontend/bindings/` liegen.
 * Diese Dateien werden nicht eingecheckt, wodurch das Frontend ohne einen Lauf
 * des Generators nicht typprüfbar war. Der namensbasierte Aufruf über die
 * Laufzeit kommt ohne Codegenerierung aus: hier steht die eine Stelle, an der
 * Methodennamen und Signaturen der Bridge festgehalten sind.
 */

const SERVICE = 'wailsbridge.BuchfinkBridge';

function invoke<T>(method: string, ...args: unknown[]): Promise<T> {
  return Call.ByName(`${SERVICE}.${method}`, ...args) as unknown as Promise<T>;
}

export const bridge = {
  // Mandanten
  GetTenants: <T>() => invoke<T>('GetTenants'),
  GetActiveTenant: <T>() => invoke<T>('GetActiveTenant'),
  SwitchTenant: (tenantId: string) => invoke<void>('SwitchTenant', tenantId),
  CreateTenant: <T>(name: string, dataDir: string, settings: unknown) =>
    invoke<T>('CreateTenant', name, dataDir, settings),
  ImportTenant: <T>(dbFilePath: string) => invoke<T>('ImportTenant', dbFilePath),
  DeleteTenant: (tenantId: string) => invoke<void>('DeleteTenant', tenantId),
  IsLocked: () => invoke<boolean>('IsLocked'),

  // Einrichtung
  GetAppConfig: <T>() => invoke<T>('GetAppConfig'),
  SetupApplication: (dataDir: string, settings: unknown) =>
    invoke<void>('SetupApplication', dataDir, settings),
  LoadExistingDatabase: (dbFilePath: string) => invoke<void>('LoadExistingDatabase', dbFilePath),
  SelectDirectoryDialog: (title: string) => invoke<string>('SelectDirectoryDialog', title),
  SelectDatabaseFileDialog: (title: string) => invoke<string>('SelectDatabaseFileDialog', title),
  SelectRecoveryFileDialog: (title: string) => invoke<string>('SelectRecoveryFileDialog', title),
  ExportRecoveryKey: () => invoke<string>('ExportRecoveryKey'),
  RecoverActiveTenantFromFile: (path: string) => invoke<void>('RecoverActiveTenantFromFile', path),

  // Geschäftsjahr & Stammdaten
  GetFiscalYear: () => invoke<number>('GetFiscalYear'),
  SetFiscalYear: (year: number) => invoke<void>('SetFiscalYear', year),
  GetAvailableFiscalYears: () => invoke<number[]>('GetAvailableFiscalYears'),
  GetCompanySettings: <T>() => invoke<T>('GetCompanySettings'),
  UpdateCompanySettings: (settings: unknown) => invoke<void>('UpdateCompanySettings', settings),

  // Konten
  GetAccounts: <T>() => invoke<T>('GetAccounts'),
  GetAccountByNumber: <T>(number: string) => invoke<T>('GetAccountByNumber', number),
  GetAccountLedger: <T>(accountNumber: string) => invoke<T>('GetAccountLedger', accountNumber),
  GetSuSaOverview: <T>() => invoke<T>('GetSuSaOverview'),
  GetPaymentAccounts: <T>() => invoke<T>('GetPaymentAccounts'),

  // Kontierung
  GetPostingGroups: <T>(direction: string) => invoke<T>('GetPostingGroups', direction),
  GetTaxTreatments: <T>(direction: string) => invoke<T>('GetTaxTreatments', direction),
  GetDifferenceKinds: <T>() => invoke<T>('GetDifferenceKinds'),

  // Journal
  GetJournalEntries: <T>() => invoke<T>('GetJournalEntries'),
  GetAllJournalEntries: <T>() => invoke<T>('GetAllJournalEntries'),
  PostJournalEntry: <T>(entry: unknown) => invoke<T>('PostJournalEntry', entry),
  PostIncomingReceipt: <T>(request: unknown) => invoke<T>('PostIncomingReceipt', request),
  ReverseJournalEntry: <T>(entryId: number, reason: string) =>
    invoke<T>('ReverseJournalEntry', entryId, reason),
  VerifyIntegrity: <T>() => invoke<T>('VerifyIntegrity'),
  GetFinancialSummary: <T>() => invoke<T>('GetFinancialSummary'),
  GetVatSummary: <T>(from: string, to: string) => invoke<T>('GetVatSummary', from, to),

  // Bank & Zahlungen
  GetBankTransactions: <T>() => invoke<T>('GetBankTransactions'),
  ImportCAMT053XML: (xmlContent: string, ledgerAccount: string) =>
    invoke<number>('ImportCAMT053XML', xmlContent, ledgerAccount),
  BookBankTransactionDirect: <T>(bankTxId: number, counterAccount: string, description: string) =>
    invoke<T>('BookBankTransactionDirect', bankTxId, counterAccount, description),
  IgnoreBankTransaction: (bankTxId: number) => invoke<void>('IgnoreBankTransaction', bankTxId),
  GetOpenItems: <T>() => invoke<T>('GetOpenItems'),
  SettlePayment: <T>(request: unknown) => invoke<T>('SettlePayment', request),

  // Kontakte & Rechnungen
  GetContacts: <T>() => invoke<T>('GetContacts'),
  SaveContact: <T>(contact: unknown) => invoke<T>('SaveContact', contact),
  DeleteContact: (id: number) => invoke<void>('DeleteContact', id),
  GetInvoices: <T>() => invoke<T>('GetInvoices'),
  IssueInvoice: <T>(invoice: unknown) => invoke<T>('IssueInvoice', invoice),
  CancelInvoice: (invoiceId: number, reason: string) => invoke<void>('CancelInvoice', invoiceId, reason),
  GenerateInvoiceZUGFeRD: <T>(invoiceId: number) => invoke<T>('GenerateInvoiceZUGFeRD', invoiceId),

  // E-Bilanz, Audit & Festschreibung
  ExportEBilanzXBRL: () => invoke<string>('ExportEBilanzXBRL'),
  GetAuditLogs: <T>() => invoke<T>('GetAuditLogs'),
  GetFestschreibungen: <T>() => invoke<T>('GetFestschreibungen'),
  CommitPeriod: <T>(periodType: string, periodLabel: string, cutoffDate: string) =>
    invoke<T>('CommitPeriod', periodType, periodLabel, cutoffDate),
  VerifyFestschreibung: <T>(id: number) => invoke<T>('VerifyFestschreibung', id),
};
