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

/**
 * Wails registriert gebundene Methoden unter ihrem voll qualifizierten Namen:
 * dem vollständigen Go-Importpfad des Pakets, dem Typnamen und dem
 * Methodennamen. Der kurze Paketname `wailsbridge` reicht nicht — die Laufzeit
 * schlägt den Namen unverändert in einer Map nach und antwortet sonst mit
 * „unknown bound method name". Die Konstante muss also mit dem Modulpfad aus
 * go.mod und dem Verzeichnis des Pakets übereinstimmen;
 * `scripts/check_bridge_bindings.py` prüft das.
 */
const SERVICE = 'github.com/buchfink/buchfink/internal/wailsbridge.BuchfinkBridge';

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

  // Belege
  SelectReceiptFilesDialog: (title: string) => invoke<string[]>('SelectReceiptFilesDialog', title),
  FileIncomingReceipt: <T>(receivedAt: string, receivedVia: string, files: unknown[]) =>
    invoke<T>('FileIncomingReceipt', receivedAt, receivedVia, files),
  AddReceiptFile: <T>(receiptId: number, file: unknown) => invoke<T>('AddReceiptFile', receiptId, file),
  RemoveReceiptFile: <T>(receiptId: number, fileId: number) =>
    invoke<T>('RemoveReceiptFile', receiptId, fileId),
  GetReceipts: <T>(status: string) => invoke<T>('GetReceipts', status),
  GetReceipt: <T>(id: number) => invoke<T>('GetReceipt', id),
  DiscardReceipt: (id: number, reason: string) => invoke<void>('DiscardReceipt', id, reason),
  GetReceiptPreview: <T>(receiptId: number) => invoke<T>('GetReceiptPreview', receiptId),
  ExtractStructuredPart: <T>(receiptId: number) => invoke<T>('ExtractStructuredPart', receiptId),
  ProposeFromEInvoice: <T>(receiptId: number) => invoke<T>('ProposeFromEInvoice', receiptId),
  ValidateEInvoice: <T>(receiptId: number) => invoke<T>('ValidateEInvoice', receiptId),
  GetEInvoiceRules: () => invoke<string[]>('GetEInvoiceRules'),
  GetUncheckedEInvoiceRules: () => invoke<Record<string, string>>('GetUncheckedEInvoiceRules'),
  PreviewIncomingReceipt: <T>(request: unknown) => invoke<T>('PreviewIncomingReceipt', request),
  PreviewOutgoingInvoice: <T>(invoice: unknown) => invoke<T>('PreviewOutgoingInvoice', invoice),

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
  GetInvoiceDocument: <T>(invoiceId: number) => invoke<T>('GetInvoiceDocument', invoiceId),

  // Anlagevermögen
  GetFixedAssets: <T>(assetClass: string) => invoke<T>('GetFixedAssets', assetClass),
  GetAssetSummary: <T>(assetClass: string) => invoke<T>('GetAssetSummary', assetClass),
  GetFixedAsset: <T>(id: number) => invoke<T>('GetFixedAsset', id),
  SaveFixedAsset: <T>(asset: unknown) => invoke<T>('SaveFixedAsset', asset),
  DeleteFixedAsset: (id: number) => invoke<void>('DeleteFixedAsset', id),
  RecordAssetCostAdjustment: <T>(request: unknown) => invoke<T>('RecordAssetCostAdjustment', request),
  GetAssetAccounts: <T>(assetClass: string) => invoke<T>('GetAssetAccounts', assetClass),
  GetAssetRules: <T>() => invoke<T>('GetAssetRules'),
  ClassifyAcquisition: <T>(netCost: number, date: string, selfUsable: boolean) =>
    invoke<T>('ClassifyAcquisition', netCost, date, selfUsable),
  PreviewDepreciationPlan: <T>(request: unknown) => invoke<T>('PreviewDepreciationPlan', request),
  GetDepreciationRun: <T>() => invoke<T>('GetDepreciationRun'),
  BookDepreciationRun: <T>(request: unknown) => invoke<T>('BookDepreciationRun', request),
  BookAssetImpairment: <T>(request: unknown) => invoke<T>('BookAssetImpairment', request),
  BookAssetWriteUp: <T>(request: unknown) => invoke<T>('BookAssetWriteUp', request),
  BookAssetMaintenance: <T>(request: unknown) => invoke<T>('BookAssetMaintenance', request),
  BookAssetIncome: <T>(request: unknown) => invoke<T>('BookAssetIncome', request),
  ValuateAssetCurrency: <T>(request: unknown) => invoke<T>('ValuateAssetCurrency', request),
  BookAssetCurrencyValuation: <T>(request: unknown) =>
    invoke<T>('BookAssetCurrencyValuation', request),
  SelectAssetDocumentsDialog: <T>(title: string) => invoke<T>('SelectAssetDocumentsDialog', title),
  AttachAssetDocument: <T>(request: unknown) => invoke<T>('AttachAssetDocument', request),
  RemoveAssetDocument: <T>(assetId: number, documentId: number) =>
    invoke<T>('RemoveAssetDocument', assetId, documentId),
  GetAssetDocumentContent: <T>(documentId: number) =>
    invoke<T>('GetAssetDocumentContent', documentId),
  GetAssetDocumentKinds: <T>() => invoke<T>('GetAssetDocumentKinds'),
  GetExpiringAssetDocuments: <T>(until: string) => invoke<T>('GetExpiringAssetDocuments', until),
  GetLegalForms: <T>() => invoke<T>('GetLegalForms'),
  GetInvestmentRules: <T>() => invoke<T>('GetInvestmentRules'),
  ComputeVorabpauschale: <T>(request: unknown) => invoke<T>('ComputeVorabpauschale', request),
  GetInvestmentNoteForIncome: <T>(assetId: number, amount: number) =>
    invoke<T>('GetInvestmentNoteForIncome', assetId, amount),
  TransferFixedAsset: <T>(request: unknown) => invoke<T>('TransferFixedAsset', request),
  PreviewAssetDisposal: <T>(request: unknown) => invoke<T>('PreviewAssetDisposal', request),
  DisposeFixedAsset: <T>(request: unknown) => invoke<T>('DisposeFixedAsset', request),
  GetAnlagenspiegel: <T>() => invoke<T>('GetAnlagenspiegel'),
  GetAssetAcquisitionCandidates: <T>() => invoke<T>('GetAssetAcquisitionCandidates'),
  GetSammelposten: <T>(fiscalYear: number) => invoke<T>('GetSammelposten', fiscalYear),

  // E-Bilanz, Audit & Festschreibung
  ExportEBilanzXBRL: () => invoke<string>('ExportEBilanzXBRL'),
  GetAuditLogs: <T>() => invoke<T>('GetAuditLogs'),
  GetFestschreibungen: <T>() => invoke<T>('GetFestschreibungen'),
  CommitPeriod: <T>(periodType: string, periodLabel: string, cutoffDate: string) =>
    invoke<T>('CommitPeriod', periodType, periodLabel, cutoffDate),
  VerifyFestschreibung: <T>(id: number) => invoke<T>('VerifyFestschreibung', id),
};
