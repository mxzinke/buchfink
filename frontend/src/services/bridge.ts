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
  GenerateInvoiceZUGFeRD: <T>(invoiceId: number) => invoke<T>('GenerateInvoiceZUGFeRD', invoiceId),
  GetInvoiceDocument: <T>(invoiceId: number) => invoke<T>('GetInvoiceDocument', invoiceId),

  // Korrektur, Storno, Versand und Nummernkreis der Ausgangsrechnungen
  RegenerateInvoiceDocument: <T>(invoiceId: number) =>
    invoke<T>('RegenerateInvoiceDocument', invoiceId),
  CancelInvoiceWithDocument: <T>(invoiceId: number, reason: string) =>
    invoke<T>('CancelInvoiceWithDocument', invoiceId, reason),
  CorrectInvoice: <T>(invoiceId: number, reason: string, replacement: unknown) =>
    invoke<T>('CorrectInvoice', invoiceId, reason, replacement),
  MarkInvoiceSent: <T>(invoiceId: number, date: string, via: string, note: string) =>
    invoke<T>('MarkInvoiceSent', invoiceId, date, via, note),
  GetInvoiceNumberGaps: <T>(year: number) => invoke<T>('GetInvoiceNumberGaps', year),
  RecordInvoiceNumberGapReason: (
    year: number,
    sequence: number,
    reason: string,
    detail: string,
  ) => invoke<void>('RecordInvoiceNumberGapReason', year, sequence, reason, detail),
  GetUnitCodes: <T>() => invoke<T>('GetUnitCodes'),
  GetEInvoiceProfiles: <T>() => invoke<T>('GetEInvoiceProfiles'),
  GetInvoiceSentViaOptions: <T>() => invoke<T>('GetInvoiceSentViaOptions'),
  GetNumberGapReasons: <T>() => invoke<T>('GetNumberGapReasons'),

  // Anzahlungen: Rechnungsverbund, Abschlag, Schlussrechnung, Ausbuchung
  GetInvoiceGroups: <T>() => invoke<T>('GetInvoiceGroups'),
  CreateInvoiceGroup: <T>(request: unknown) => invoke<T>('CreateInvoiceGroup', request),
  IssueAdvanceInvoice: <T>(request: unknown) => invoke<T>('IssueAdvanceInvoice', request),
  SettleAdvance: <T>(request: unknown) => invoke<T>('SettleAdvance', request),
  RefundAdvance: <T>(request: unknown) => invoke<T>('RefundAdvance', request),
  IssueFinalInvoice: <T>(request: unknown) => invoke<T>('IssueFinalInvoice', request),
  GetOpenAdvances: <T>() => invoke<T>('GetOpenAdvances'),
  GetAdvanceTargets: <T>() => invoke<T>('GetAdvanceTargets'),
  GetOpenVendorAdvances: <T>(contactId: number) => invoke<T>('GetOpenVendorAdvances', contactId),
  WriteOffOpenItem: <T>(request: unknown) => invoke<T>('WriteOffOpenItem', request),

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
  GetLegacySpecialDepreciations: <T>() => invoke<T>('GetLegacySpecialDepreciations'),
  GetAssetAcquisitionCandidates: <T>() => invoke<T>('GetAssetAcquisitionCandidates'),
  GetSammelposten: <T>(fiscalYear: number) => invoke<T>('GetSammelposten', fiscalYear),

  // Bilanz und Gewinn- und Verlustrechnung
  GetStatement: <T>(year: number, depth: string) => invoke<T>('GetStatement', year, depth),
  GetSizeClass: <T>(year: number) => invoke<T>('GetSizeClass', year),
  GetStatementDeadlines: <T>(year: number) => invoke<T>('GetStatementDeadlines', year),
  ExportStatementPDF: (year: number) => invoke<string>('ExportStatementPDF', year),
  ExportStatementCSV: (year: number) => invoke<string>('ExportStatementCSV', year),
  SetAverageEmployees: <T>(year: number, count: number) =>
    invoke<T>('SetAverageEmployees', year, count),
  SetPriorYearRevenue: <T>(year: number, amount: number) =>
    invoke<T>('SetPriorYearRevenue', year, amount),

  // E-Bilanz, Audit & Festschreibung
  ExportEBilanzXBRL: () => invoke<string>('ExportEBilanzXBRL'),
  GetEBilanzMappingReport: <T>(year: number) => invoke<T>('GetEBilanzMappingReport', year),
  GetAuditLogs: <T>() => invoke<T>('GetAuditLogs'),
  GetFestschreibungen: <T>() => invoke<T>('GetFestschreibungen'),
  CommitPeriod: <T>(
    periodType: string,
    periodLabel: string,
    cutoffDate: string,
    overrideReason: string,
  ) => invoke<T>('CommitPeriod', periodType, periodLabel, cutoffDate, overrideReason),
  VerifyFestschreibung: <T>(id: number) => invoke<T>('VerifyFestschreibung', id),

  // Jahresabschluss
  GetFiscalYears: <T>() => invoke<T>('GetFiscalYears'),
  CreateFiscalYear: (year: number) => invoke<void>('CreateFiscalYear', year),
  GetClosingState: <T>(year: number) => invoke<T>('GetClosingState', year),
  GetCarryForwardPreview: <T>(toYear: number) => invoke<T>('GetCarryForwardPreview', toYear),
  CarryForward: <T>(toYear: number) => invoke<T>('CarryForward', toYear),
  SetFiscalYearStatus: <T>(year: number, status: string, date: string, note: string) =>
    invoke<T>('SetFiscalYearStatus', year, status, date, note),
  ReopenFiscalYear: <T>(year: number, reason: string) => invoke<T>('ReopenFiscalYear', year, reason),

  // Abschlussbausteine: Schritte, Abgrenzung, Rückstellungen, Vorräte,
  // Umsatzsteuer-Verrechnung, Steuerrückstellung, Ergebnisverwendung, Anhang
  GetClosingSteps: <T>(year: number) => invoke<T>('GetClosingSteps', year),
  SkipClosingStep: <T>(year: number, key: string, reason: string) =>
    invoke<T>('SkipClosingStep', year, key, reason),
  MarkClosingStepDone: <T>(year: number, key: string) =>
    invoke<T>('MarkClosingStepDone', year, key),

  ProposeAccruals: <T>(year: number) => invoke<T>('ProposeAccruals', year),
  PreviewAccrual: <T>(request: unknown) => invoke<T>('PreviewAccrual', request),
  BookAccrual: <T>(request: unknown) => invoke<T>('BookAccrual', request),
  GetAccruals: <T>(year: number) => invoke<T>('GetAccruals', year),
  GetAccrualReport: <T>(cutoff: string) => invoke<T>('GetAccrualReport', cutoff),

  GetProvisions: <T>(year: number) => invoke<T>('GetProvisions', year),
  PreviewProvision: <T>(request: unknown) => invoke<T>('PreviewProvision', request),
  BookProvisionFormation: <T>(request: unknown) => invoke<T>('BookProvisionFormation', request),
  BookProvisionIncrease: <T>(request: unknown) => invoke<T>('BookProvisionIncrease', request),
  BookProvisionRelease: <T>(request: unknown) => invoke<T>('BookProvisionRelease', request),
  BookProvisionConsumption: <T>(request: unknown) => invoke<T>('BookProvisionConsumption', request),
  BookProvisionUnwinding: <T>(request: unknown) => invoke<T>('BookProvisionUnwinding', request),
  SettleProvision: <T>(provisionId: number, date: string, reason: string) =>
    invoke<T>('SettleProvision', provisionId, date, reason),
  GetProvisionMirror: <T>(year: number) => invoke<T>('GetProvisionMirror', year),
  GetDiscountRates: <T>(month: string) => invoke<T>('GetDiscountRates', month),
  GetDiscountRateMonths: () => invoke<string[]>('GetDiscountRateMonths'),
  SaveDiscountRates: (rows: unknown[]) => invoke<void>('SaveDiscountRates', rows),
  ImportDiscountRatesCSV: (path: string, month: string, average: number) =>
    invoke<number>('ImportDiscountRatesCSV', path, month, average),

  GetInventoryAccounts: <T>(year: number) => invoke<T>('GetInventoryAccounts', year),
  PreviewInventory: <T>(request: unknown) => invoke<T>('PreviewInventory', request),
  BookInventory: <T>(request: unknown) => invoke<T>('BookInventory', request),

  PreviewVatSettlement: <T>(year: number) => invoke<T>('PreviewVatSettlement', year),
  BookVatSettlement: <T>(year: number) => invoke<T>('BookVatSettlement', year),
  PreviewTaxProvision: <T>(year: number) => invoke<T>('PreviewTaxProvision', year),
  BookTaxProvision: <T>(request: unknown) => invoke<T>('BookTaxProvision', request),

  PreviewAppropriation: <T>(year: number, request: unknown) =>
    invoke<T>('PreviewAppropriation', year, request),
  BookAppropriation: <T>(year: number, request: unknown) =>
    invoke<T>('BookAppropriation', year, request),
  GetAppropriation: <T>(year: number) => invoke<T>('GetAppropriation', year),

  GetNotesTexts: <T>(year: number) => invoke<T>('GetNotesTexts', year),
  SaveNotesText: <T>(year: number, section: string, text: string) =>
    invoke<T>('SaveNotesText', year, section, text),

  GetClosingSettings: <T>() => invoke<T>('GetClosingSettings'),
  SaveClosingSettings: <T>(settings: unknown) => invoke<T>('SaveClosingSettings', settings),

  GetTaxElectionRegister: <T>(year: number) => invoke<T>('GetTaxElectionRegister', year),
  ExportTaxElectionRegisterCSV: (year: number) =>
    invoke<string>('ExportTaxElectionRegisterCSV', year),
  GetReconciliation: <T>(year: number) => invoke<T>('GetReconciliation', year),

  // Umsatzsteuer-Voranmeldung
  GetVatPeriods: <T>(year: number) => invoke<T>('GetVatPeriods', year),
  GetVatReturn: <T>(periodKey: string) => invoke<T>('GetVatReturn', periodKey),
  SaveVatReturn: <T>(periodKey: string) => invoke<T>('SaveVatReturn', periodKey),
  GetVatReturns: <T>(year: number) => invoke<T>('GetVatReturns', year),
  ConfirmVatReturnSubmitted: <T>(id: number, date: string, ticket: string, note: string) =>
    invoke<T>('ConfirmVatReturnSubmitted', id, date, ticket, note),
  CreateVatCorrection: <T>(periodKey: string) => invoke<T>('CreateVatCorrection', periodKey),
  ExportVatReturnCSV: (id: number) => invoke<string>('ExportVatReturnCSV', id),
  GetSpecialPrepaymentSuggestion: <T>(year: number) =>
    invoke<T>('GetSpecialPrepaymentSuggestion', year),

  // Zusammenfassende Meldung
  GetZMPeriods: <T>(year: number) => invoke<T>('GetZMPeriods', year),
  GetZMReturn: <T>(periodKey: string) => invoke<T>('GetZMReturn', periodKey),
  SaveZMReturn: <T>(periodKey: string) => invoke<T>('SaveZMReturn', periodKey),
  GetZMReturns: <T>(year: number) => invoke<T>('GetZMReturns', year),
  ConfirmZMSubmitted: <T>(id: number, date: string, ticket: string, note: string) =>
    invoke<T>('ConfirmZMSubmitted', id, date, ticket, note),
  CreateZMCorrection: <T>(periodKey: string) => invoke<T>('CreateZMCorrection', periodKey),
  ExportZMCSV: (id: number) => invoke<string>('ExportZMCSV', id),

  // Prüfläufe und Fristen
  RunChecks: <T>(cutoffDate: string, periodType: string) =>
    invoke<T>('RunChecks', cutoffDate, periodType),
  GetCheckRuns: <T>(year: number) => invoke<T>('GetCheckRuns', year),
  GetDeadlines: <T>(year: number) => invoke<T>('GetDeadlines', year),
  MarkDeadlineDone: (key: string, date: string) => invoke<void>('MarkDeadlineDone', key, date),

  // Gründung
  GetFoundationState: <T>() => invoke<T>('GetFoundationState'),
  SaveFoundation: <T>(foundation: unknown) => invoke<T>('SaveFoundation', foundation),
  GetFoundationRules: <T>() => invoke<T>('GetFoundationRules'),
  GetRecommendedVatPeriod: <T>(foundingYear: number) =>
    invoke<T>('GetRecommendedVatPeriod', foundingYear),
  PreviewFoundationPostings: <T>() => invoke<T>('PreviewFoundationPostings'),
  BookFoundationPostings: <T>() => invoke<T>('BookFoundationPostings'),
  RegisterCompany: <T>(date: string, court: string, number: string) =>
    invoke<T>('RegisterCompany', date, court, number),
  CompleteFoundationDuty: (key: string, doneOn: string, note: string) =>
    invoke<void>('CompleteFoundationDuty', key, doneOn, note),

  // Datenüberlassung nach § 147 Abs. 6 AO
  ExportZ3: <T>(year: number, targetDir: string) => invoke<T>('ExportZ3', year, targetDir),
  ExportArchive: <T>(year: number, targetDir: string) => invoke<T>('ExportArchive', year, targetDir),
  ExportAuditPackage: <T>(year: number, targetDir: string) =>
    invoke<T>('ExportAuditPackage', year, targetDir),
  ExportJournalCSV: (from: string, to: string) => invoke<string>('ExportJournalCSV', from, to),
  ExportKeyDirectory: () => invoke<string>('ExportKeyDirectory'),
  GetKeyDirectory: <T>() => invoke<T>('GetKeyDirectory'),
  SelectExportDirectoryDialog: (title: string) =>
    invoke<string>('SelectExportDirectoryDialog', title),
  SelectSaveFileDialog: (title: string, suggestedName: string) =>
    invoke<string>('SelectSaveFileDialog', title, suggestedName),
  SaveReceiptFileAs: (receiptId: number, fileId: number) =>
    invoke<string>('SaveReceiptFileAs', receiptId, fileId),

  // Prüfläufe über Kette und Dateien
  VerifyReceiptFiles: <T>() => invoke<T>('VerifyReceiptFiles'),

  // Stichtagsauswertungen
  GetSuSaOverviewAt: <T>(cutoff: string) => invoke<T>('GetSuSaOverviewAt', cutoff),
  GetAccountLedgerRange: <T>(accountNumber: string, from: string, to: string) =>
    invoke<T>('GetAccountLedgerRange', accountNumber, from, to),
  GetOpenItemsAt: <T>(cutoff: string) => invoke<T>('GetOpenItemsAt', cutoff),
  GetPaymentAllocations: <T>(entryId: number) => invoke<T>('GetPaymentAllocations', entryId),

  // Bankimport über den Dateipfad — die Datei wird als Beleg archiviert
  ImportCAMT053File: (path: string, ledgerAccount: string) =>
    invoke<number>('ImportCAMT053File', path, ledgerAccount),
  SelectStatementFileDialog: (title: string) => invoke<string>('SelectStatementFileDialog', title),

  // Sicherung und Wiederherstellung
  GetBackupRuns: <T>() => invoke<T>('GetBackupRuns'),
  SetBackupDir: <T>(dir: string) => invoke<T>('SetBackupDir', dir),
  CreateBackup: <T>() => invoke<T>('CreateBackup'),
  VerifyBackup: <T>(zipPath: string) => invoke<T>('VerifyBackup', zipPath),
  RestoreFromBackup: <T>(zipPath: string, targetDir: string) =>
    invoke<T>('RestoreFromBackup', zipPath, targetDir),
  SelectBackupDirDialog: (title: string) => invoke<string>('SelectBackupDirDialog', title),
  SelectBackupFileDialog: (title: string) => invoke<string>('SelectBackupFileDialog', title),

  // Steuerliche Nebenpflichten: Vorsteuerberichtigung § 15a UStG
  GetInputTaxCorrections: <T>(year: number) => invoke<T>('GetInputTaxCorrections', year),
  PreviewInputTaxCorrection: <T>(year: number) => invoke<T>('PreviewInputTaxCorrection', year),
  RegisterInputTaxCorrection: <T>(request: unknown) =>
    invoke<T>('RegisterInputTaxCorrection', request),
  CloseInputTaxCorrection: <T>(id: number, reason: string, date: string) =>
    invoke<T>('CloseInputTaxCorrection', id, reason, date),
  SaveInputTaxUsage: <T>(request: unknown) => invoke<T>('SaveInputTaxUsage', request),
  BookInputTaxCorrection: <T>(year: number) => invoke<T>('BookInputTaxCorrection', year),

  // Bestätigung der USt-IdNr. (§ 18e UStG)
  GetVatIDChecks: <T>(contactId: number) => invoke<T>('GetVatIDChecks', contactId),
  GetVatIDStatus: <T>(contactId: number) => invoke<T>('GetVatIDStatus', contactId),
  CheckVatID: <T>(contactId: number) => invoke<T>('CheckVatID', contactId),

  // Belegnachweis der innergemeinschaftlichen Lieferung
  GetSupplyEvidenceKinds: <T>() => invoke<T>('GetSupplyEvidenceKinds'),
  GetSupplyEvidence: <T>(invoiceId: number, transport: string) =>
    invoke<T>('GetSupplyEvidence', invoiceId, transport),
  AddSupplyEvidence: <T>(request: unknown) => invoke<T>('AddSupplyEvidence', request),
  SetSupplyTransport: <T>(invoiceId: number, transport: string) =>
    invoke<T>('SetSupplyTransport', invoiceId, transport),
  RemoveSupplyEvidence: <T>(invoiceId: number, evidenceId: number, transport: string) =>
    invoke<T>('RemoveSupplyEvidence', invoiceId, evidenceId, transport),
  GetSupplyEvidenceReport: <T>(year: number) => invoke<T>('GetSupplyEvidenceReport', year),

  // Nicht abziehbare Betriebsausgaben (§ 4 Abs. 5 EStG)
  GetNonDeductibleCategories: <T>() => invoke<T>('GetNonDeductibleCategories'),
  GetNonDeductibleReport: <T>(year: number) => invoke<T>('GetNonDeductibleReport', year),
  RebookGiftsForRecipient: <T>(request: unknown) => invoke<T>('RebookGiftsForRecipient', request),

  // Fremdwährung: Tageskurse, Umsatzsteuerkurse, Stichtagsbewertung
  GetExchangeRate: <T>(currency: string, date: string) =>
    invoke<T>('GetExchangeRate', currency, date),
  GetExchangeRates: <T>(currency: string, from: string, to: string) =>
    invoke<T>('GetExchangeRates', currency, from, to),
  SaveExchangeRate: <T>(rate: unknown) => invoke<T>('SaveExchangeRate', rate),
  GetVatExchangeRates: <T>(from: string, to: string) =>
    invoke<T>('GetVatExchangeRates', from, to),
  SaveVatExchangeRate: <T>(rate: unknown) => invoke<T>('SaveVatExchangeRate', rate),
  ImportVatExchangeRatesCSV: <T>(path: string) => invoke<T>('ImportVatExchangeRatesCSV', path),
  PreviewCurrencyValuation: <T>(year: number) => invoke<T>('PreviewCurrencyValuation', year),
  BookCurrencyValuation: <T>(year: number) => invoke<T>('BookCurrencyValuation', year),

  // Anlagen: Regelsätze, Wertaufholung, Sammelposten, anschaffungsnahe Kosten
  GetAfaRules: <T>() => invoke<T>('GetAfaRules'),
  GetWriteUpReport: <T>(year: number) => invoke<T>('GetWriteUpReport', year),
  ConfirmImpairmentPersists: <T>(assetId: number, year: number, note: string) =>
    invoke<T>('ConfirmImpairmentPersists', assetId, year, note),
  GetPoolConsistencyReport: <T>(year: number) => invoke<T>('GetPoolConsistencyReport', year),
  CheckNearAcquisitionCost: <T>(assetId: number, date: string, amount: number) =>
    invoke<T>('CheckNearAcquisitionCost', assetId, date, amount),
  CapitalizeNearAcquisitionCost: <T>(request: unknown) =>
    invoke<T>('CapitalizeNearAcquisitionCost', request),
  GetExemptionCertificateWarnings: <T>(today: string) =>
    invoke<T>('GetExemptionCertificateWarnings', today),

  // Adressen der Netzdienste
  GetServiceEndpoints: <T>() => invoke<T>('GetServiceEndpoints'),
  SaveServiceEndpoints: <T>(endpoints: unknown) => invoke<T>('SaveServiceEndpoints', endpoints),

  // Prüfermodus
  EnableReadOnly: <T>(until: string, reason: string) => invoke<T>('EnableReadOnly', until, reason),
  DisableReadOnly: <T>(reason: string) => invoke<T>('DisableReadOnly', reason),
  GetProgramVersion: () => invoke<string>('GetProgramVersion'),
};
