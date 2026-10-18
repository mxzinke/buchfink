package wailsbridge

import (
	"github.com/buchfink/buchfink/internal/domain"
)

// Der Prüfermodus.
//
// Während einer Außenprüfung wird der Datenbestand eingefroren: der Prüfer
// bekommt Zugriff auf die Daten des Prüfungszeitraums (§ 147 Abs. 6 AO), und
// was er sieht, darf sich unter seinen Händen nicht ändern. Buchfink kennt
// keine Benutzerkonten und kann den Zugriff deshalb nicht nach Personen
// trennen; die Sperre gilt für die ganze Anwendung und ist befristet.
//
// Die Prüfung sitzt im Kern (Tenant.Write): jeder Schreibvorgang geht dort
// durch, gleich ob er aus der Oberfläche oder von einem Agenten kommt.

// readOnlyAllowed sind die Methoden, die im Prüfermodus zulässig bleiben.
//
// Die Liste ist die Ausnahme und nicht die Regel: jede exportierte Methode der
// Bridge steht entweder hier oder schreibt über write/change. Der Test
// TestEveryBridgeMethodIsClassified hält das fest — eine neu hinzugefügte
// schreibende Methode, die niemand eingeordnet hat, fällt dort auf und nicht
// erst dem Prüfer.
//
// Zulässig bleibt, was die Buchführungsdaten des aktiven Mandanten nicht
// verändert: Lesen, Auswerten, Ausgeben, Dateidialoge — und die Verwaltung der
// Mandanten, die keine Buchung berührt. Der Export gehört
// ausdrücklich dazu: er ist der Zweck des Modus.
var readOnlyAllowed = map[string]bool{
	// Die Auswertungen der steuerlichen Nebenpflichten lesen und
	// rechnen. GetExchangeRate und PreviewCurrencyValuation holen im laufenden
	// Betrieb einen fehlenden Kurs und legen ihn ab; im Prüfermodus schalten
	// beide auf den nur lesenden Kursdienst um und stehen deshalb hier. Ein
	// Prüfer muss die Stichtagsbewertung eines abgeschlossenen Jahres ansehen
	// können — die Kurse dazu stehen dann längst in der Historie.
	"GetAfaRules": true,
	// Die Regeln für verbundene Apps lesen.
	"GetAgentRules": true,
	// Die lesenden Auswertungen der laufenden Buchhaltung.
	// SaveFilteredJournalCSV gibt Journalzeilen heraus und protokolliert das
	// als Zugriff (QUE-02 K2); das Protokoll ist der eine Schreibweg eines
	// Lesevorgangs.
	"GetEntriesForReceipt":          true,
	"GetReceiptFindings":            true,
	"GetRetentionRules":             true,
	"GetVatPeriodProposal":          true,
	"GetFilteredJournal":            true,
	"SaveFilteredJournalCSV":        true,
	"GetExchangeRate":               true,
	"PreviewCurrencyValuation":      true,
	"GetInputTaxCorrections":        true,
	"GetNonDeductibleReport":        true,
	"GetPoolConsistencyReport":      true,
	"GetVatExchangeRates":           true,
	"GetVatIDStatus":                true,
	"GetWriteUpReport":              true,
	"GetClosingSettings":            true,
	"GetAccountLedger":              true,
	"GetAccounts":                   true,
	"GetActiveTenant":               true,
	"GetAllJournalEntries":          true,
	"GetAnlagenspiegel":             true,
	"GetAppConfig":                  true,
	"GetAssetAcquisitionCandidates": true,
	"GetAssetRules":                 true,
	"GetAvailableFiscalYears":       true,
	"GetBackupRuns":                 true,
	"GetBankTransactions":           true,
	"GetCarryForwardPreview":        true,
	"GetCheckRuns":                  true,
	"GetClosingState":               true,
	"GetCompanySettings":            true,
	"GetContacts":                   true,
	"GetDeadlines":                  true,
	"GetObligations":                true,
	"GetDeclaration":                true,
	"GetEBilanzMappingReport":       true,
	"GetEInvoiceProfiles":           true,
	"GetEInvoiceRules":              true,
	"GetFestschreibungen":           true,
	"GetFiscalYear":                 true,
	"GetFiscalYears":                true,
	"GetFiscalYearCandidates":       true,
	"GetFixedAsset":                 true,
	"GetFixedAssets":                true,
	"GetFoundationRules":            true,
	"GetCountries":                  true,
	"GetFoundationState":            true,
	// Mandantenverzeichnis, Rechnungswesen und Nachschlagewerte
	"ValidateTenantDirectory": true,
	"GetInvoiceNumberGaps":    true,
	"GetNumberGapReasons":     true,
	"GetUnitCodes":            true,
	"GetInvestmentRules":      true,
	"GetInvoiceDocument":      true,
	"GetInvoices":             true,
	"GetJournalEntries":       true,
	"GetOpenItemsAt":          true,
	"GetPaymentAccounts":      true,
	"GetPaymentAllocations":   true,
	"GetPostingGroups":        true,
	"GetReceiptPreview":       true,
	"GetReceipts":             true,
	"GetSizeClass":            true,
	"GetStatement":            true,
	"GetSuSaOverview":         true,
	"GetTaxTreatments":        true,
	"GetTenants":              true,
	"GetVatReturn":            true,
	"GetZMPeriods":            true,
	"GetZMReturn":             true,

	// Die Abschlussbausteine: Schrittliste, Rückstellungen, Abgrenzung,
	// Verzeichnis und Anhang. Lesen und rechnen sie nur, bleiben sie
	// im Prüfermodus offen — gerade das Verzeichnis nach § 5 Abs. 1 Satz 2
	// EStG und die Überleitung sind das, wonach ein Prüfer als Erstes fragt.
	"GetAccruals":            true,
	"GetAppropriation":       true,
	"GetClosingSteps":        true,
	"GetNotesTexts":          true,
	"GetProvisionMirror":     true,
	"GetProvisions":          true,
	"GetReconciliation":      true,
	"GetTaxElectionRegister": true,

	// Rechnen, ohne zu buchen
	"PreviewAppropriation":   true,
	"PreviewIncomingReceipt": true,
	"PreviewOutgoingInvoice": true,
	"PreviewTaxProvision":    true,
	"PreviewVatSettlement":   true,
	"ProposeFromEInvoice":    true,

	// Ausgeben. Der Export ist der Zweck des Prüfermodus und muss in ihm
	// laufen — eine Sperre, die die Datenüberlassung verhindert, wäre das
	// Gegenteil dessen, wofür der Modus da ist.
	"ExportArchive":                true,
	"ExportAuditPackage":           true,
	"SaveEBilanzAs":                true,
	"ExportStatementCSV":           true,
	"ExportStatementPDF":           true,
	"ExportVatReturnCSV":           true,
	"ExportZ3":                     true,
	"SaveProcedureDocumentationAs": true,
	// Der Verweis nach draußen. Er öffnet den Browser und rührt keine Daten an;
	// gerade während einer Prüfung darf nachgesehen werden, was eine Fassung
	// geändert hat.
	"OpenReleasesPage":  true,
	"SaveReceiptFileAs": true,

	// Die Nachweise lesen Protokoll, Historie, Migrationen
	// und die Altersstruktur. Gerade im Prüfermodus sind sie das,
	// wonach gefragt wird.
	"GetAuditLogsFiltered":        true,
	"GetChangeLog":                true,
	"GetComplianceHints":          true,
	"GetMigrationRecords":         true,
	"GetOpenItemsAging":           true,
	"GetOrganisationTextDefaults": true,
	"GetOrganisationTexts":        true,
	"GetProcedureDocumentations":  true,
	"GetSchemaMigrations":         true,
	"GetSelectableContacts":       true,
	"VerifyAuditChain":            true,

	// Aufgabenliste, Monatsabschluss,
	// Zuordnungsvorschlag und Prüfpfad lesen und
	// rechnen — gebucht und geschrieben wird an keiner dieser Stellen. Gerade
	// der Prüfpfad gehört in den Prüfermodus: er ist die Antwort auf die Frage,
	// die eine Prüfung stellt.
	"GetTasks":           true,
	"GetMonthCloseState": true,
	"SuggestBankMatches": true,
	"GetBankRules":       true,
	"GetAuditTrail":      true,

	// Die Übersicht: Pflichten des Kalenders und die Buchungen der
	// verbundenen Apps seit der letzten Festschreibung.
	"GetNextObligations": true,
	"GetObligation":      true,
	"GetAgentBookings":   true,

	// Prüfen
	"VerifyBackup":         true,
	"VerifyFestschreibung": true,
	"VerifyIntegrity":      true,
	"VerifyReceiptFiles":   true,

	// Dateidialoge. Sie wählen einen Pfad und schreiben nichts.
	"SelectBackupDirDialog":       true,
	"SelectBackupFileDialog":      true,
	"SelectDatabaseFileDialog":    true,
	"SelectDirectoryDialog":       true,
	"SelectExportDirectoryDialog": true,
	"SelectReceiptFilesDialog":    true,
	"SelectSaveFileDialog":        true,
	"SelectStatementFileDialog":   true,

	// Mandanten. Sie berühren keine Buchung des gesperrten Mandanten: ein
	// zweiter Mandant darf angelegt und eine Sicherung darf wiederhergestellt
	// werden, während hier eine Prüfung läuft. Den geprüften Mandanten selbst
	// nimmt der Kern nicht aus der Liste.
	"DeleteTenant":         true,
	"GetBankAccounts":      true,
	"PreviewBankStatement": true,
	"LoadExistingDatabase": true,
	"RestoreFromBackup":    true,
	"SetupApplication":     true,
	"SwitchTenant":         true,

	// Ansichtseinstellungen. Das gewählte Geschäftsjahr ist ein Filter der
	// Oberfläche und keine Buchführungsangabe.
	"SetFiscalYear": true,

	// Der Modus selbst. Er muss sich ein- und ausschalten lassen, während er
	// gilt — sonst ließe er sich nie beenden.
	"DisableReadOnly": true,
	"EnableReadOnly":  true,

	// Die Sicherung. Sie liest die Daten und schreibt eine Datei; der
	// Sicherungslauf im Protokoll ist ein Nachweis und keine Buchung. Gerade
	// während einer Prüfung soll gesichert werden dürfen.
	"CreateBackup": true,
	"SetBackupDir": true,

	// Die Haken der Wails-Laufzeit. Der Start öffnet den MCP-Server, das Ende
	// sichert, was da ist; keiner ändert etwas an den Büchern.
	"ServiceStartup":  true,
	"ServiceShutdown": true,

	// Verbundene Apps. Zulassungen und Tokens liegen in einer eigenen Datei
	// neben der App-Konfiguration, nicht in den Büchern; was eine App
	// schreibt, prüft der Kern bei jedem Vorgang selbst. Gerade im
	// Prüfermodus muss sich ein Zugriff entziehen lassen.
	"GetMCPServerStatus":    true,
	"SetMCPPort":            true,
	"GetConnectedApps":      true,
	"RevokeConnectedApp":    true,
	"GetPendingAppRequests": true,
	"ResolveAppRequest":     true,
	"OpenAgentGuide":        true,

	// Der Eingang. Er gehört nicht zu den Büchern: zeigen, öffnen und
	// hineinziehen bleibt auch während einer Prüfung möglich, und ob ein
	// Ordner synchronisiert wird, fragt nur das Dateisystem.
	"GetInbox":    true,
	"ListInbox":   true,
	"OpenInbox":   true,
	"AddToInbox":  true,
	"CheckFolder": true,

	// Stammdaten: der Stand der Gesellschafterdarlehen und die Umrechnung der
	// Steuernummer lesen nur.
	"GetShareholderLoans":   true,
	"FormatElsterTaxNumber": true,
}

// EnableReadOnly schaltet den Prüfermodus bis zu einem Tag ein. Datum und
// Grund sind Pflicht.
func (b *BuchfinkBridge) EnableReadOnly(until, reason string) (domain.AppConfig, error) {
	return b.host.EnableExaminer(uiContext(), until, reason)
}

// DisableReadOnly beendet den Prüfermodus. Der Grund ist Pflicht und steht im
// Änderungsprotokoll.
func (b *BuchfinkBridge) DisableReadOnly(reason string) (domain.AppConfig, error) {
	return b.host.DisableExaminer(uiContext(), reason)
}
