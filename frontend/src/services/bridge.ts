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
  DeleteTenant: (tenantId: string) => invoke<void>('DeleteTenant', tenantId),

  GetBankAccounts: <T>() => invoke<T>('GetBankAccounts'),
  PreviewBankStatement: <T>(path: string) => invoke<T>('PreviewBankStatement', path),
  ConfigureBankAccounts: (accounts: unknown) => invoke<void>('ConfigureBankAccounts', accounts),
  SetInvoiceBankAccount: (iban: string) => invoke<void>('SetInvoiceBankAccount', iban),

  // Einrichtung
  ValidateTenantDirectory: (dataDir: string) => invoke<void>('ValidateTenantDirectory', dataDir),
  GetAppConfig: <T>() => invoke<T>('GetAppConfig'),
  SetupApplication: (dataDir: string, inboxDir: string, settings: unknown) =>
    invoke<void>('SetupApplication', dataDir, inboxDir, settings),
  LoadExistingDatabase: (dbFilePath: string) => invoke<void>('LoadExistingDatabase', dbFilePath),
  SelectDirectoryDialog: (title: string) => invoke<string>('SelectDirectoryDialog', title),
  SelectDatabaseFileDialog: (title: string) => invoke<string>('SelectDatabaseFileDialog', title),

  // Geschäftsjahr & Stammdaten
  GetFiscalYear: () => invoke<number>('GetFiscalYear'),
  SetFiscalYear: (year: number) => invoke<void>('SetFiscalYear', year),
  GetAvailableFiscalYears: () => invoke<number[]>('GetAvailableFiscalYears'),
  GetCompanySettings: <T>() => invoke<T>('GetCompanySettings'),
  UpdateCompanySettings: (settings: unknown) => invoke<void>('UpdateCompanySettings', settings),

  // Konten
  GetAccounts: <T>() => invoke<T>('GetAccounts'),
  GetAccountLedger: <T>(accountNumber: string) => invoke<T>('GetAccountLedger', accountNumber),
  GetSuSaOverview: <T>() => invoke<T>('GetSuSaOverview'),
  GetPaymentAccounts: <T>() => invoke<T>('GetPaymentAccounts'),

  // Kontierung
  GetPostingGroups: <T>(direction: string) => invoke<T>('GetPostingGroups', direction),
  GetTaxTreatments: <T>(direction: string) => invoke<T>('GetTaxTreatments', direction),

  // Journal
  GetJournalEntries: <T>() => invoke<T>('GetJournalEntries'),
  GetAllJournalEntries: <T>() => invoke<T>('GetAllJournalEntries'),
  PostIncomingReceipt: <T>(request: unknown) => invoke<T>('PostIncomingReceipt', request),
  VerifyIntegrity: <T>() => invoke<T>('VerifyIntegrity'),

  // Belege
  SelectReceiptFilesDialog: (title: string) => invoke<string[]>('SelectReceiptFilesDialog', title),
  FileIncomingReceipt: <T>(receivedAt: string, receivedVia: string, files: unknown[]) =>
    invoke<T>('FileIncomingReceipt', receivedAt, receivedVia, files),
  GetReceipts: <T>(status: string) => invoke<T>('GetReceipts', status),
  GetReceiptPreview: <T>(receiptId: number) => invoke<T>('GetReceiptPreview', receiptId),
  ProposeFromEInvoice: <T>(receiptId: number) => invoke<T>('ProposeFromEInvoice', receiptId),
  GetEInvoiceRules: () => invoke<string[]>('GetEInvoiceRules'),
  PreviewIncomingReceipt: <T>(request: unknown) => invoke<T>('PreviewIncomingReceipt', request),
  PreviewOutgoingInvoice: <T>(invoice: unknown) => invoke<T>('PreviewOutgoingInvoice', invoice),

  // Bank & Zahlungen
  GetBankTransactions: <T>() => invoke<T>('GetBankTransactions'),
  BookBankTransactionDirect: <T>(bankTxId: number, counterAccount: string, description: string) =>
    invoke<T>('BookBankTransactionDirect', bankTxId, counterAccount, description),
  SettlePayment: <T>(request: unknown) => invoke<T>('SettlePayment', request),

  // Kontakte & Rechnungen
  GetContacts: <T>() => invoke<T>('GetContacts'),
  SaveContact: <T>(contact: unknown) => invoke<T>('SaveContact', contact),
  GetInvoices: <T>() => invoke<T>('GetInvoices'),
  IssueInvoice: <T>(invoice: unknown) => invoke<T>('IssueInvoice', invoice),
  GetInvoiceDocument: <T>(invoiceId: number) => invoke<T>('GetInvoiceDocument', invoiceId),

  // Korrektur, Storno, Versand und Nummernkreis der Ausgangsrechnungen
  RegenerateInvoiceDocument: <T>(invoiceId: number) =>
    invoke<T>('RegenerateInvoiceDocument', invoiceId),
  GetInvoiceNumberGaps: <T>(year: number) => invoke<T>('GetInvoiceNumberGaps', year),
  RecordInvoiceNumberGapReason: (
    year: number,
    sequence: number,
    reason: string,
    detail: string,
  ) => invoke<void>('RecordInvoiceNumberGapReason', year, sequence, reason, detail),
  GetUnitCodes: <T>() => invoke<T>('GetUnitCodes'),
  GetEInvoiceProfiles: <T>() => invoke<T>('GetEInvoiceProfiles'),
  GetNumberGapReasons: <T>() => invoke<T>('GetNumberGapReasons'),

  // Anzahlungen
  SettleAdvance: <T>(request: unknown) => invoke<T>('SettleAdvance', request),

  // Anlagevermögen
  GetFixedAssets: <T>(assetClass: string) => invoke<T>('GetFixedAssets', assetClass),
  GetFixedAsset: <T>(id: number) => invoke<T>('GetFixedAsset', id),
  GetAnlagenspiegel: <T>() => invoke<T>('GetAnlagenspiegel'),
  GetAssetAcquisitionCandidates: <T>() => invoke<T>('GetAssetAcquisitionCandidates'),

  // Bilanz und Gewinn- und Verlustrechnung
  GetStatement: <T>(year: number, depth: string) => invoke<T>('GetStatement', year, depth),
  GetSizeClass: <T>(year: number) => invoke<T>('GetSizeClass', year),
  ExportStatementPDF: (year: number) => invoke<string>('ExportStatementPDF', year),
  ExportStatementCSV: (year: number) => invoke<string>('ExportStatementCSV', year),
  SetAverageEmployees: <T>(year: number, count: number) =>
    invoke<T>('SetAverageEmployees', year, count),

  // E-Bilanz, Audit & Festschreibung
  SaveEBilanzAs: (year: number, format: string) => invoke<string>('SaveEBilanzAs', year, format),
  GetEBilanzMappingReport: <T>(year: number) => invoke<T>('GetEBilanzMappingReport', year),
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
  GetFiscalYearCandidates: <T>() => invoke<T>('GetFiscalYearCandidates'),
  CreateFiscalYear: (year: number) => invoke<void>('CreateFiscalYear', year),
  BookLegalReserve: <T>(year: number) => invoke<T>('BookLegalReserve', year),
  GetClosingState: <T>(year: number) => invoke<T>('GetClosingState', year),
  GetCarryForwardPreview: <T>(toYear: number) => invoke<T>('GetCarryForwardPreview', toYear),
  CarryForward: <T>(toYear: number) => invoke<T>('CarryForward', toYear),
  SetFiscalYearStatus: <T>(year: number, status: string, date: string, note: string) =>
    invoke<T>('SetFiscalYearStatus', year, status, date, note),
  ReopenFiscalYear: <T>(year: number, reason: string) => invoke<T>('ReopenFiscalYear', year, reason),

  // Abschlussbausteine
  GetClosingSteps: <T>(year: number) => invoke<T>('GetClosingSteps', year),
  SkipClosingStep: <T>(year: number, key: string, reason: string) =>
    invoke<T>('SkipClosingStep', year, key, reason),
  ReopenClosingStep: <T>(year: number, key: string, reason: string) =>
    invoke<T>('ReopenClosingStep', year, key, reason),

  GetAccruals: <T>(year: number) => invoke<T>('GetAccruals', year),

  GetProvisions: <T>(year: number) => invoke<T>('GetProvisions', year),
  GetProvisionMirror: <T>(year: number) => invoke<T>('GetProvisionMirror', year),

  PreviewVatSettlement: <T>(year: number) => invoke<T>('PreviewVatSettlement', year),
  PreviewTaxProvision: <T>(year: number) => invoke<T>('PreviewTaxProvision', year),

  PreviewAppropriation: <T>(year: number, request: unknown) =>
    invoke<T>('PreviewAppropriation', year, request),
  BookAppropriation: <T>(year: number, request: unknown) =>
    invoke<T>('BookAppropriation', year, request),
  GetAppropriation: <T>(year: number) => invoke<T>('GetAppropriation', year),

  GetNotesTexts: <T>(year: number) => invoke<T>('GetNotesTexts', year),

  GetClosingSettings: <T>() => invoke<T>('GetClosingSettings'),
  SaveClosingSettings: <T>(settings: unknown) => invoke<T>('SaveClosingSettings', settings),

  GetTaxElectionRegister: <T>(year: number) => invoke<T>('GetTaxElectionRegister', year),
  GetReconciliation: <T>(year: number) => invoke<T>('GetReconciliation', year),

  // Umsatzsteuer-Voranmeldung
  GetVatReturn: <T>(periodKey: string) => invoke<T>('GetVatReturn', periodKey),
  SaveVatReturn: <T>(periodKey: string) => invoke<T>('SaveVatReturn', periodKey),
  ConfirmVatReturnSubmitted: <T>(id: number, date: string, ticket: string, note: string) =>
    invoke<T>('ConfirmVatReturnSubmitted', id, date, ticket, note),
  CreateVatCorrection: <T>(periodKey: string) => invoke<T>('CreateVatCorrection', periodKey),
  ExportVatReturnCSV: (id: number) => invoke<string>('ExportVatReturnCSV', id),

  // Zusammenfassende Meldung
  GetZMPeriods: <T>(year: number) => invoke<T>('GetZMPeriods', year),
  GetZMReturn: <T>(periodKey: string) => invoke<T>('GetZMReturn', periodKey),
  SaveZMReturn: <T>(periodKey: string) => invoke<T>('SaveZMReturn', periodKey),
  ConfirmZMSubmitted: <T>(id: number, date: string, ticket: string, note: string) =>
    invoke<T>('ConfirmZMSubmitted', id, date, ticket, note),

  // Prüfläufe und Fristen
  RunChecks: <T>(cutoffDate: string, periodType: string) =>
    invoke<T>('RunChecks', cutoffDate, periodType),
  GetCheckRuns: <T>(year: number) => invoke<T>('GetCheckRuns', year),
  GetDeadlines: <T>(year: number) => invoke<T>('GetDeadlines', year),
  MarkDeadlineDone: (key: string, date: string) => invoke<void>('MarkDeadlineDone', key, date),

  // Pflichtenkalender und Jahreserklärungen (internal/wailsbridge/closings_service.go);
  // Bestätigen und Entfallen stehen bei der Übersicht.
  GetObligations: <T>(year: number) => invoke<T>('GetObligations', year),
  GetDeclaration: <T>(kind: string, year: number) => invoke<T>('GetDeclaration', kind, year),
  SetDeclarationInput: <T>(request: unknown) => invoke<T>('SetDeclarationInput', request),
  ConfirmDeclarationInput: <T>(kind: string, year: number, key: string) =>
    invoke<T>('ConfirmDeclarationInput', kind, year, key),
  DeleteDeclarationInput: (kind: string, year: number, key: string) =>
    invoke<void>('DeleteDeclarationInput', kind, year, key),
  SubmitDeclaration: <T>(request: unknown) => invoke<T>('SubmitDeclaration', request),
  ExportDeclarationJSON: <T>(kind: string, year: number) =>
    invoke<T>('ExportDeclarationJSON', kind, year),

  // Gründung
  GetFoundationState: <T>() => invoke<T>('GetFoundationState'),
  SaveFoundation: <T>(foundation: unknown) => invoke<T>('SaveFoundation', foundation),
  GetFoundationRules: <T>() => invoke<T>('GetFoundationRules'),
  GetCountries: <T>() => invoke<T>('GetCountries'),

  // Datenüberlassung nach § 147 Abs. 6 AO
  ExportZ3: <T>(year: number, targetDir: string) => invoke<T>('ExportZ3', year, targetDir),
  ExportArchive: <T>(year: number, targetDir: string) => invoke<T>('ExportArchive', year, targetDir),
  ExportAuditPackage: <T>(year: number, targetDir: string) =>
    invoke<T>('ExportAuditPackage', year, targetDir),
  SelectExportDirectoryDialog: (title: string) =>
    invoke<string>('SelectExportDirectoryDialog', title),
  SelectSaveFileDialog: (title: string, suggestedName: string) =>
    invoke<string>('SelectSaveFileDialog', title, suggestedName),
  SaveReceiptFileAs: (receiptId: number, fileId: number) =>
    invoke<string>('SaveReceiptFileAs', receiptId, fileId),

  // Prüfläufe über Kette und Dateien
  VerifyReceiptFiles: <T>() => invoke<T>('VerifyReceiptFiles'),

  // Stichtagsauswertungen
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

  // Bestätigung der USt-IdNr. (§ 18e UStG)
  GetVatIDStatus: <T>(contactId: number) => invoke<T>('GetVatIDStatus', contactId),

  // Nicht abziehbare Betriebsausgaben (§ 4 Abs. 5 EStG)
  GetNonDeductibleReport: <T>(year: number) => invoke<T>('GetNonDeductibleReport', year),

  // Fremdwährung: Stichtagsbewertung
  PreviewCurrencyValuation: <T>(year: number) => invoke<T>('PreviewCurrencyValuation', year),

  // Anlagen: Wertaufholung
  GetWriteUpReport: <T>(year: number) => invoke<T>('GetWriteUpReport', year),

  // Nachweise: Änderungsprotokoll, Versionen, Migrationen
  VerifyAuditChain: <T>() => invoke<T>('VerifyAuditChain'),
  GetAuditLogsFiltered: <T>(limit: number, filter: unknown) =>
    invoke<T>('GetAuditLogsFiltered', limit, filter),
  GetChangeLog: <T>() => invoke<T>('GetChangeLog'),
  GetSchemaMigrations: <T>() => invoke<T>('GetSchemaMigrations'),
  GetMigrationRecords: <T>() => invoke<T>('GetMigrationRecords'),
  SetSystemChangeDate: (date: string) => invoke<void>('SetSystemChangeDate', date),

  // Verfahrensdokumentation und Hinweise zum Funktionsumfang
  GenerateProcedureDocumentation: <T>() => invoke<T>('GenerateProcedureDocumentation'),
  GetProcedureDocumentations: <T>() => invoke<T>('GetProcedureDocumentations'),
  GetOrganisationTextDefaults: <T>() => invoke<T>('GetOrganisationTextDefaults'),
  OpenReleasesPage: () => invoke<void>('OpenReleasesPage'),
  SaveProcedureDocumentationAs: (id: number, wantPdf: boolean) =>
    invoke<string>('SaveProcedureDocumentationAs', id, wantPdf),
  GetOrganisationTexts: <T>() => invoke<T>('GetOrganisationTexts'),
  SaveOrganisationTexts: (texts: unknown) => invoke<void>('SaveOrganisationTexts', texts),
  GetComplianceHints: <T>() => invoke<T>('GetComplianceHints'),

  // Kontakte
  BlockContact: <T>(id: number, reason: string) => invoke<T>('BlockContact', id, reason),
  GetSelectableContacts: <T>() => invoke<T>('GetSelectableContacts'),

  // Die Bedienung der Welle 7: Aufgabenliste, Monatsabschluss,
  // Zuordnungsvorschlag und Prüfpfad. Die Reihenfolge der Argumente ist gegen
  // internal/wailsbridge/welle7_service.go geprüft; `invoke` ist untypisiert
  // und würde eine vertauschte Reihenfolge erst zur Laufzeit melden.
  GetTasks: <T>() => invoke<T>('GetTasks'),
  GetMonthCloseState: <T>(month: string) => invoke<T>('GetMonthCloseState', month),
  SuggestBankMatches: <T>(bankTxId: number) => invoke<T>('SuggestBankMatches', bankTxId),
  GetBankRules: <T>() => invoke<T>('GetBankRules'),
  DeleteBankRule: (id: number) => invoke<void>('DeleteBankRule', id),

  // Übersicht (internal/wailsbridge/overview_service.go)
  GetNextObligations: <T>(limit: number) => invoke<T>('GetNextObligations', limit),
  GetObligation: <T>(key: string) => invoke<T>('GetObligation', key),
  ConfirmObligation: <T>(key: string, evidence: unknown) =>
    invoke<T>('ConfirmObligation', key, evidence),
  WaiveObligation: <T>(key: string, reason: string) => invoke<T>('WaiveObligation', key, reason),
  ReopenObligation: <T>(key: string, reason: string) =>
    invoke<T>('ReopenObligation', key, reason),
  GetAgentBookings: <T>(limit: number) => invoke<T>('GetAgentBookings', limit),

  // Belegverweis, Fristen, Journalfilter. Jede Signatur ist gegen
  // internal/wailsbridge/welle8_service.go geprüft — Anzahl und Reihenfolge der
  // Argumente —, weil `invoke` untypisiert ist und eine vertauschte Reihenfolge
  // erst zur Laufzeit auffiele.
  GetRetentionRules: <T>() => invoke<T>('GetRetentionRules'),
  GetEntriesForReceipt: <T>(receiptId: number) => invoke<T>('GetEntriesForReceipt', receiptId),
  GetVatPeriodProposal: <T>(year: number) => invoke<T>('GetVatPeriodProposal', year),
  GetFilteredJournal: <T>(filter: unknown) => invoke<T>('GetFilteredJournal', filter),
  SaveFilteredJournalCSV: (filter: unknown, path: string) =>
    invoke<string>('SaveFilteredJournalCSV', filter, path),

  // Prüfermodus
  EnableReadOnly: <T>(until: string, reason: string) => invoke<T>('EnableReadOnly', until, reason),
  DisableReadOnly: <T>(reason: string) => invoke<T>('DisableReadOnly', reason),

  // Verbundene Apps (internal/wailsbridge/mcp_service.go)
  GetMCPServerStatus: <T>() => invoke<T>('GetMCPServerStatus'),
  SetMCPPort: <T>(port: number) => invoke<T>('SetMCPPort', port),
  GetConnectedApps: <T>() => invoke<T>('GetConnectedApps'),
  RevokeConnectedApp: (clientId: string) => invoke<void>('RevokeConnectedApp', clientId),
  GetPendingAppRequests: <T>() => invoke<T>('GetPendingAppRequests'),
  ResolveAppRequest: (id: string, nonce: string, approve: boolean) =>
    invoke<void>('ResolveAppRequest', id, nonce, approve),

  // Stammdaten des Unternehmens (internal/wailsbridge/company_service.go)
  RegisterCompany: <T>(date: string, court: string, number: string) =>
    invoke<T>('RegisterCompany', date, court, number),
  GetShareholderLoans: <T>() => invoke<T>('GetShareholderLoans'),
  RegisterShareholderLoan: <T>(request: unknown) => invoke<T>('RegisterShareholderLoan', request),
  CorrectShareholderLoan: <T>(id: number, request: unknown, reason: string) =>
    invoke<T>('CorrectShareholderLoan', id, request, reason),
  FormatElsterTaxNumber: (taxNumber: string, federalState: string) =>
    invoke<string>('FormatElsterTaxNumber', taxNumber, federalState),
  OpenAgentGuide: (client: string) => invoke<void>('OpenAgentGuide', client),
};
