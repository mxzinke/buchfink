package wailsbridge

import (
	"context"
	"fmt"
	"strings"
	"time"

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
// Die Prüfung sitzt an der Bridge und nicht in den Diensten. Die Bridge ist die
// einzige Stelle, an der eine Bedienung ankommt — ein Dienst wird auch vom
// automatischen Sicherungslauf und vom Export gerufen, und die müssen weiter
// arbeiten.

// readOnlyAllowed sind die Methoden, die im Prüfermodus zulässig bleiben.
//
// Die Liste ist die Ausnahme und nicht die Regel: jede exportierte Methode der
// Bridge steht entweder hier oder ruft ensureWritable. Der Test
// TestEveryBridgeMethodIsClassified hält das fest — eine neu hinzugefügte
// schreibende Methode, die niemand eingeordnet hat, fällt dort auf und nicht
// erst dem Prüfer.
//
// Zulässig bleibt, was die Buchführungsdaten des aktiven Mandanten nicht
// verändert: Lesen, Auswerten, Ausgeben, Dateidialoge — und die Verwaltung der
// Mandanten und Schlüssel, die keine Buchung berührt. Der Export gehört
// ausdrücklich dazu: er ist der Zweck des Modus.
var readOnlyAllowed = map[string]bool{
	// Lesen und Auswerten
	"GetAccountByNumber":             true,
	"GetAccountLedger":               true,
	"GetAccountLedgerRange":          true,
	"GetAccounts":                    true,
	"GetActiveTenant":                true,
	"GetAllJournalEntries":           true,
	"GetAnlagenspiegel":              true,
	"GetAppConfig":                   true,
	"GetAssetAccounts":               true,
	"GetAssetAcquisitionCandidates":  true,
	"GetAssetDocumentContent":        true,
	"GetAssetDocumentKinds":          true,
	"GetAssetRules":                  true,
	"GetAssetSummary":                true,
	"GetAuditLogs":                   true,
	"GetAvailableFiscalYears":        true,
	"GetBackupRuns":                  true,
	"GetBankTransactions":            true,
	"GetCarryForwardPreview":         true,
	"GetCheckRuns":                   true,
	"GetClosingState":                true,
	"GetCompanySettings":             true,
	"GetContacts":                    true,
	"GetDeadlines":                   true,
	"GetDepreciationRun":             true,
	"GetDifferenceKinds":             true,
	"GetEBilanzMappingReport":        true,
	"GetEInvoiceRules":               true,
	"GetExpiringAssetDocuments":      true,
	"GetFestschreibungen":            true,
	"GetFinancialSummary":            true,
	"GetFiscalYear":                  true,
	"GetFiscalYears":                 true,
	"GetFixedAsset":                  true,
	"GetFixedAssets":                 true,
	"GetFoundationRules":             true,
	"GetFoundationState":             true,
	"GetInvestmentNoteForIncome":     true,
	"GetInvestmentRules":             true,
	"GetInvoiceDocument":             true,
	"GetInvoices":                    true,
	"GetJournalEntries":              true,
	"GetKeyDirectory":                true,
	"GetLegalForms":                  true,
	"GetOpenItems":                   true,
	"GetOpenItemsAt":                 true,
	"GetPaymentAccounts":             true,
	"GetPaymentAllocations":          true,
	"GetPostingGroups":               true,
	"GetProgramVersion":              true,
	"GetReceipt":                     true,
	"GetReceiptPreview":              true,
	"GetReceipts":                    true,
	"GetRecommendedVatPeriod":        true,
	"GetSKR04Catalog":                true,
	"GetSammelposten":                true,
	"GetSizeClass":                   true,
	"GetSpecialPrepaymentSuggestion": true,
	"GetStatement":                   true,
	"GetStatementDeadlines":          true,
	"GetSuSaOverview":                true,
	"GetSuSaOverviewAt":              true,
	"GetTaxTreatments":               true,
	"GetTenants":                     true,
	"GetUncheckedEInvoiceRules":      true,
	"GetVatPeriods":                  true,
	"GetVatReturn":                   true,
	"GetVatReturns":                  true,
	"GetVatSummary":                  true,
	"GetZMPeriods":                   true,
	"GetZMReturn":                    true,
	"GetZMReturns":                   true,
	"IsLocked":                       true,

	// Rechnen, ohne zu buchen
	"ClassifyAcquisition":       true,
	"ComputeVorabpauschale":     true,
	"PreviewAssetDisposal":      true,
	"PreviewDepreciationPlan":   true,
	"PreviewFoundationPostings": true,
	"PreviewIncomingReceipt":    true,
	"PreviewOutgoingInvoice":    true,
	"ProposeFromEInvoice":       true,
	"ValuateAssetCurrency":      true,

	// Ausgeben. Der Export ist der Zweck des Prüfermodus und muss in ihm
	// laufen — eine Sperre, die die Datenüberlassung verhindert, wäre das
	// Gegenteil dessen, wofür der Modus da ist.
	"ExportArchive":          true,
	"ExportAuditPackage":     true,
	"ExportEBilanzXBRL":      true,
	"ExportJournalCSV":       true,
	"ExportKeyDirectory":     true,
	"ExportStatementCSV":     true,
	"ExportStatementPDF":     true,
	"ExportVatReturnCSV":     true,
	"ExportZ3":               true,
	"ExportZMCSV":            true,
	"GenerateInvoiceZUGFeRD": true,
	"SaveReceiptFileAs":      true,

	// Prüfen
	"VerifyBackup":         true,
	"VerifyFestschreibung": true,
	"VerifyIntegrity":      true,
	"VerifyReceiptFiles":   true,

	// Dateidialoge. Sie wählen einen Pfad und schreiben nichts.
	"SelectAssetDocumentsDialog":  true,
	"SelectBackupDirDialog":       true,
	"SelectBackupFileDialog":      true,
	"SelectDatabaseFileDialog":    true,
	"SelectDirectoryDialog":       true,
	"SelectExportDirectoryDialog": true,
	"SelectReceiptFilesDialog":    true,
	"SelectRecoveryFileDialog":    true,
	"SelectSaveFileDialog":        true,
	"SelectStatementFileDialog":   true,

	// Mandanten und Schlüssel. Sie berühren keine Buchung des gesperrten
	// Mandanten: ein zweiter Mandant darf angelegt, eine Sicherung darf
	// wiederhergestellt und der Schlüssel darf gesichert werden, während hier
	// eine Prüfung läuft.
	"CreateTenant":                true,
	"ExportRecoveryKey":           true,
	"ImportTenant":                true,
	"LoadExistingDatabase":        true,
	"RecoverActiveTenantFromFile": true,
	"RestoreFromBackup":           true,
	"SetupApplication":            true,
	"SwitchTenant":                true,

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

	// Der Haken der Wails-Laufzeit beim Beenden. Er sichert, was da ist, und
	// ändert nichts an den Büchern.
	"ServiceShutdown": true,
}

// readOnlyStateLocked liefert den Zustand des Prüfermodus.
//
// Der Aufrufer hält bereits b.mu — jede schreibende Bridge-Methode nimmt die
// Sperre, bevor sie prüft. Eine zweite Sperre hier wäre ein Deadlock.
func (b *BuchfinkBridge) readOnlyStateLocked() (active bool, until, reason string) {
	tenant := b.activeTenantLocked()
	if tenant == nil {
		return false, "", ""
	}
	today := time.Now().Format("2006-01-02")
	return tenant.ReadOnlyActiveOn(today), tenant.ReadOnlyUntil, tenant.ReadOnlyReason
}

// ensureWritable weist jede schreibende Methode ab, solange der Prüfermodus
// gilt.
//
// Sie steht am Anfang jeder solchen Methode, hinter dem Sperren von b.mu. Eine
// Prüfung, die nur die Oberfläche vornähme — Knöpfe ausblenden —, wäre keine:
// die Bridge ist über die Wails-Laufzeit auch ohne die Oberfläche erreichbar.
func (b *BuchfinkBridge) ensureWritable() error {
	active, until, reason := b.readOnlyStateLocked()
	if !active {
		return nil
	}
	message := fmt.Sprintf("Der Prüfermodus ist bis zum %s aktiv. Bis dahin nimmt die Buchführung keine Änderung auf", formatGermanDate(until))
	if strings.TrimSpace(reason) != "" {
		message += fmt.Sprintf(" (Grund: %s)", reason)
	}
	return fmt.Errorf("%s.", message)
}

// EnableReadOnly schaltet den Prüfermodus bis zu einem Tag ein.
//
// Datum und Grund sind Pflicht. Ohne Ende bliebe der Modus liegen und
// blockierte irgendwann die laufende Buchführung; ohne Grund ließe sich später
// nicht mehr sagen, welche Prüfung ihn veranlasst hat.
func (b *BuchfinkBridge) EnableReadOnly(until, reason string) (domain.AppConfig, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if strings.TrimSpace(until) == "" {
		return b.appConfig, fmt.Errorf("der Prüfermodus braucht ein Datum, bis zu dem er gilt")
	}
	if _, err := time.Parse("2006-01-02", until); err != nil {
		return b.appConfig, fmt.Errorf("%q ist kein gültiges Datum (erwartet JJJJ-MM-TT)", until)
	}
	if until < time.Now().Format("2006-01-02") {
		return b.appConfig, fmt.Errorf("der Prüfermodus kann nicht in der Vergangenheit enden")
	}
	if strings.TrimSpace(reason) == "" {
		return b.appConfig, fmt.Errorf("bitte den Grund angeben, aus dem der Prüfermodus eingeschaltet wird")
	}

	tenant := b.activeTenantLocked()
	if tenant == nil {
		return b.appConfig, fmt.Errorf("kein aktiver Mandant gefunden")
	}
	tenant.ReadOnlyUntil = until
	tenant.ReadOnlyReason = reason
	b.appConfig.SyncActiveTenant(time.Now().Format("2006-01-02"))
	if err := b.appCfgRepo.Save(&b.appConfig); err != nil {
		return b.appConfig, err
	}

	if b.auditRepo != nil {
		_ = b.auditRepo.Log(context.Background(), domain.AuditActionUpdate, "READ_ONLY", tenant.ID,
			fmt.Sprintf("Prüfermodus eingeschaltet bis %s. Grund: %s", until, reason))
	}
	return b.appConfig, nil
}

// DisableReadOnly beendet den Prüfermodus. Der Grund ist Pflicht und steht im
// Änderungsprotokoll: ein vorzeitig beendeter Prüfermodus ist ein Vorgang, den
// jemand zu verantworten hat.
func (b *BuchfinkBridge) DisableReadOnly(reason string) (domain.AppConfig, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if strings.TrimSpace(reason) == "" {
		return b.appConfig, fmt.Errorf("bitte den Grund angeben, aus dem der Prüfermodus beendet wird")
	}
	tenant := b.activeTenantLocked()
	if tenant == nil {
		return b.appConfig, fmt.Errorf("kein aktiver Mandant gefunden")
	}

	was := tenant.ReadOnlyUntil
	tenant.ReadOnlyUntil = ""
	tenant.ReadOnlyReason = ""
	b.appConfig.SyncActiveTenant(time.Now().Format("2006-01-02"))
	if err := b.appCfgRepo.Save(&b.appConfig); err != nil {
		return b.appConfig, err
	}

	if b.auditRepo != nil {
		_ = b.auditRepo.Log(context.Background(), domain.AuditActionUpdate, "READ_ONLY", tenant.ID,
			fmt.Sprintf("Prüfermodus beendet (war bis %s). Grund: %s", was, reason))
	}
	return b.appConfig, nil
}

// GetProgramVersion liefert die Fassung des laufenden Programms.
func (b *BuchfinkBridge) GetProgramVersion() string { return programVersion() }

// formatGermanDate macht aus 2026-06-30 den 30.06.2026.
func formatGermanDate(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return t.Format("02.01.2006")
}
