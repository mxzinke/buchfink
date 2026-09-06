package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// --- Prüferpaket: Protokollkette, Versionshistorie, Verfahrensdokumentation --

// Das Prüferpaket muss die drei Nachweise enthalten, die die Buchführung
// erklären: die Kette des Änderungsprotokolls, die Versionshistorie des
// Programms und die erzeugte Verfahrensdokumentation. Ohne sie ist es ein
// Archivexport mit einem anderen Namen.
func TestAuditPackageCarriesAuditChainChangelogAndProcedureDoc(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.filledBooks(t)

	// Eine erzeugte Fassung der Verfahrensdokumentation liegt im Belegspeicher
	// und nicht als Datei im Datenordner — genau dort holt der Export sie.
	procDocs := newProcDocService(t, env)
	generated, err := procDocs.Generate(ctx, time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("die Verfahrensdokumentation ließ sich nicht erzeugen: %v", err)
	}

	svc := env.exports(t)
	svc.SetIntegritySource(testIntegrity{journal: env.journal, receipts: env.receipts})
	svc.SetProcDocRepo(repository.NewProcedureDocumentationRepository(env.db))

	dir := filepath.Join(t.TempDir(), "pruefer")
	result, err := svc.ExportAuditPackage(ctx, env.fiscalYear, dir)
	if err != nil {
		t.Fatalf("Prüferpaket: %v", err)
	}

	report, err := os.ReadFile(filepath.Join(dir, "integritaet.txt"))
	if err != nil {
		t.Fatalf("integritaet.txt fehlt: %v", err)
	}
	// Die Protokollkette steht als eigener Abschnitt neben der Journalkette.
	for _, want := range []string{"Änderungsprotokoll", "Davon verkettet:", "Letzter Kettenwert:"} {
		if !strings.Contains(string(report), want) {
			t.Errorf("der Nachweis nennt %q nicht:\n%s", want, report)
		}
	}
	if strings.Contains(string(report), "noch keine verketteten Einträge") {
		t.Errorf("die Protokollkette wurde nicht geprüft:\n%s", report)
	}

	changelog, err := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	if err != nil {
		t.Fatalf("CHANGELOG.md fehlt im Prüferpaket: %v", err)
	}
	if len(changelog) == 0 {
		t.Error("die Versionshistorie liegt leer bei")
	}

	docPath := filepath.Join(dir, "verfahrensdokumentation", generated.Document.FileName)
	content, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("die Verfahrensdokumentation fehlt im Prüferpaket: %v", err)
	}
	if !strings.Contains(string(content), "Verfahrensdokumentation") {
		t.Error("die beigelegte Fassung enthält nicht den erzeugten Text")
	}
	for _, note := range result.Notes {
		if strings.Contains(note, "Verfahrensdokumentation liegt nicht") ||
			strings.Contains(note, "noch keine Verfahrensdokumentation") {
			t.Errorf("das Paket vermerkt eine fehlende Verfahrensdokumentation, obwohl sie beiliegt: %q", note)
		}
	}

	// Die Bearbeiterkennung gehört ans Paket selbst und nicht nur an den
	// Protokolleintrag daneben.
	if strings.TrimSpace(result.Actor) == "" {
		t.Error("das Exportergebnis hat keine Bearbeiterkennung")
	}
	if !strings.Contains(string(report), "Bearbeiter:") {
		t.Errorf("der Nachweis nennt den Bearbeiter nicht:\n%s", report)
	}
}

// Ohne erzeugte Fassung sagt das Paket das ausdrücklich. Ein Prüferpaket, das
// ohne Hinweis ohne Verfahrensdokumentation ankommt, sieht vollständig aus.
func TestAuditPackageNotesMissingProcedureDocumentation(t *testing.T) {
	env := newTestEnv(t)
	env.filledBooks(t)

	svc := env.exports(t)
	svc.SetIntegritySource(testIntegrity{journal: env.journal, receipts: env.receipts})
	svc.SetProcDocRepo(repository.NewProcedureDocumentationRepository(env.db))

	dir := filepath.Join(t.TempDir(), "pruefer")
	result, err := svc.ExportAuditPackage(context.Background(), env.fiscalYear, dir)
	if err != nil {
		t.Fatalf("Prüferpaket: %v", err)
	}
	found := false
	for _, note := range result.Notes {
		if strings.Contains(note, "Verfahrensdokumentation") {
			found = true
		}
	}
	if !found {
		t.Errorf("die fehlende Verfahrensdokumentation wurde nicht vermerkt: %v", result.Notes)
	}
}

// Die Integritätsprüfung (Welle 4) muss die Protokollkette mitprüfen: eine
// Buchführung, deren Buchungen unverändert sind, während sich die Einträge über
// ihre Änderungen entfernen ließen, belegt nur die halbe Unveränderbarkeit.
func TestVerifyIntegrityIncludesTheAuditChain(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.filledBooks(t)

	result, err := env.journal.VerifyIntegrity(ctx)
	if err != nil {
		t.Fatalf("Integritätsprüfung: %v", err)
	}
	if result.AuditChain == nil {
		t.Fatal("die Integritätsprüfung meldet die Protokollkette nicht")
	}
	if !result.AuditChain.IsValid || result.AuditChain.CheckedEntries == 0 {
		t.Fatalf("die Protokollkette muss geprüft und gültig sein: %+v", result.AuditChain)
	}

	// Ein nachträglich veränderter Protokolleintrag macht das Gesamtergebnis
	// ungültig.
	if err := env.db.Exec(
		"UPDATE audit_log_entries SET details = ? WHERE id = (SELECT MIN(id) FROM audit_log_entries)",
		"nachträglich geändert").Error; err != nil {
		t.Fatalf("Protokolleintrag ändern: %v", err)
	}
	broken, err := env.journal.VerifyIntegrity(ctx)
	if err != nil {
		t.Fatalf("Integritätsprüfung: %v", err)
	}
	if broken.IsValid {
		t.Error("ein verändertes Änderungsprotokoll muss die Integritätsprüfung scheitern lassen")
	}
	if broken.AuditChain == nil || broken.AuditChain.IsValid {
		t.Error("die Protokollkette muss den Bruch melden")
	}
}

// --- Vorher/Nachher an den Stammdatenwegen ---------------------------------

// latestAudit liefert den jüngsten Protokolleintrag zu einer Objektart. Die
// Abfrage liefert absteigend nach Kennung — der erste Eintrag ist der jüngste.
func latestAudit(t *testing.T, env *testEnv, entityType string) domain.AuditLogEntry {
	t.Helper()
	entries, err := repository.NewAuditRepository(env.db).FindFiltered(
		context.Background(), 0, domain.AuditFilter{EntityType: entityType})
	if err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("kein Protokolleintrag der Art %s gefunden", entityType)
	}
	return entries[0]
}

// fields liest das Vorher oder Nachher eines Eintrags als Feldkarte.
func fields(t *testing.T, raw string) map[string]any {
	t.Helper()
	if raw == "" {
		return map[string]any{}
	}
	out := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("Vorher/Nachher ist kein JSON: %v (%s)", err, raw)
	}
	return out
}

// Der tatsächliche Weg, nicht nur die Hilfsfunktion: SaveContact liest den
// bisherigen Stand und protokolliert nur, was sich geändert hat.
func TestSaveContactLogsOnlyTheChangedFields(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	contact := &domain.Contact{
		Type: domain.ContactTypeVendor, Name: "Lieferant GmbH",
		Street: "Alte Straße 1", PostalCode: "20095", City: "Hamburg", CountryCode: "DE",
	}
	if err := env.contacts.SaveContact(ctx, contact); err != nil {
		t.Fatalf("Kontakt anlegen: %v", err)
	}

	contact.Street = "Neue Straße 7"
	if err := env.contacts.SaveContact(ctx, contact); err != nil {
		t.Fatalf("Kontakt ändern: %v", err)
	}

	entry := latestAudit(t, env, "CONTACT")
	before := fields(t, entry.Before)
	after := fields(t, entry.After)

	if before["street"] != "Alte Straße 1" {
		t.Errorf("das Vorher hat die alte Straße nicht: %v", before)
	}
	if after["street"] != "Neue Straße 7" {
		t.Errorf("das Nachher hat die neue Straße nicht: %v", after)
	}
	// Der unveränderte Name gehört nicht ins Protokoll: er verbärge die eine
	// Änderung zwischen den unveränderten Feldern.
	if _, ok := after["name"]; ok {
		t.Errorf("der unveränderte Name steht im Protokoll: %v", after)
	}
	if _, ok := after["vatPeriod"]; ok {
		t.Errorf("ein Feld, das der Repository-Weg normalisiert, erzeugt Rauschen: %v", after)
	}
}

// Dasselbe für die Firmeneinstellungen: eine geänderte Besteuerungsart steht mit
// beiden Ständen im Protokoll, der Rest nicht.
func TestUpdateCompanySettingsLogsOnlyTheChangedFields(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	settings := NewSettingsService(
		repository.NewSettingsRepository(env.db), repository.NewAuditRepository(env.db))

	current, err := settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Unternehmensdaten lesen: %v", err)
	}
	current.TaxationType = "IST"
	if err := settings.UpdateCompanySettings(ctx, current); err != nil {
		t.Fatalf("Unternehmensdaten ändern: %v", err)
	}

	entry := latestAudit(t, env, "SETTINGS")
	before := fields(t, entry.Before)
	after := fields(t, entry.After)

	if before["taxationType"] != "SOLL" || after["taxationType"] != "IST" {
		t.Errorf("der Wechsel der Besteuerungsart fehlt im Protokoll: %v → %v", before, after)
	}
	if _, ok := after["companyName"]; ok {
		t.Errorf("der unveränderte Firmenname steht im Protokoll: %v", after)
	}
}

// Die Anlagenstammdaten sind der dritte Weg, den die Spezifikation nennt.
func TestSaveFixedAssetLogsBeforeAndAfter(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	assets := env.assets(t)

	asset, err := assets.Save(ctx, &domain.FixedAsset{
		Name: "Fräsmaschine", Class: domain.AssetClassTangible,
		AcquisitionDate: "2026-01-15", AcquisitionCost: 1_200_000, UsefulLifeMonths: 120,
		Method: domain.DepreciationLinear, Account: "0440", DepreciationAccount: "6220",
	})
	if err != nil {
		t.Fatalf("Anlagegut anlegen: %v", err)
	}

	asset.UsefulLifeMonths = 60
	asset.UsefulLifeReason = "Zweischichtbetrieb, amtliche AfA-Tabelle unterschritten"
	if _, err := assets.Save(ctx, asset); err != nil {
		t.Fatalf("Anlagegut ändern: %v", err)
	}

	entry := latestAudit(t, env, "ANLAGE")
	before := fields(t, entry.Before)
	after := fields(t, entry.After)
	if before["usefulLifeMonths"] != float64(120) || after["usefulLifeMonths"] != float64(60) {
		t.Errorf("die geänderte Nutzungsdauer fehlt im Protokoll: %v → %v", before, after)
	}
	if _, ok := after["name"]; ok {
		t.Errorf("der unveränderte Name steht im Protokoll: %v", after)
	}
	if _, ok := after["movements"]; ok {
		t.Errorf("die Bewegungen gehören nicht ins Vorher/Nachher: %v", after)
	}
}

// Der Abschlussstand des Geschäftsjahres ist der vierte Weg.
func TestFiscalYearStatusLogsBeforeAndAfter(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	closing := env.closing(t)

	if _, err := closing.SetAverageEmployees(ctx, 2026, 7); err != nil {
		t.Fatalf("Arbeitnehmerzahl setzen: %v", err)
	}
	entry := latestAudit(t, env, "FISCAL_YEAR")
	before := fields(t, entry.Before)
	after := fields(t, entry.After)
	if after["averageEmployees"] != float64(7) {
		t.Errorf("die neue Arbeitnehmerzahl fehlt im Protokoll: %v", after)
	}
	if _, ok := before["averageEmployees"]; ok && before["averageEmployees"] == float64(7) {
		t.Errorf("das Vorher hat schon den neuen Wert: %v", before)
	}
	if _, ok := after["year"]; ok {
		t.Errorf("das unveränderte Jahr steht im Protokoll: %v", after)
	}
}

// --- Kopfdaten sind Pflicht beim Buchen ------------------------------------

// Die Ausgangsrechnung legt ihren Beleg mit Kopfdaten ab: Rechnungsdatum, eigene
// Firma, Bruttobetrag. Sie sind an der Rechnung bekannt, und ohne sie bliebe die
// Belegliste für Ausgangsrechnungen ohne Datum und Betrag.
func TestOutgoingInvoiceReceiptCarriesHeaderData(t *testing.T) {
	if testing.Short() {
		t.Skip("die WASM-Kompilierung des Dokuments ist zu langsam für -short")
	}
	env := newTestEnv(t)
	ctx := context.Background()
	env.withSellerContact(t)

	customer := env.customer(t, "Kunde GmbH", "DE", "")
	inv := env.simpleInvoice(customer.ID, "2026-03-01", 100000)
	if err := env.invoicesWiredWithDocuments(t).Issue(ctx, inv); err != nil {
		t.Fatalf("Rechnung ausstellen: %v", err)
	}
	if inv.ReceiptID == nil {
		t.Fatal("die ausgestellte Rechnung hat keinen Beleg")
	}
	receipt, err := env.receipts.Get(ctx, *inv.ReceiptID)
	if err != nil {
		t.Fatalf("Beleg lesen: %v", err)
	}

	if receipt.DocumentDate != inv.Date {
		t.Errorf("Belegdatum %q, erwartet das Rechnungsdatum %q", receipt.DocumentDate, inv.Date)
	}
	if receipt.IssuerName != "Pfennig Ventures GmbH" {
		t.Errorf("Aussteller %q, erwartet die eigene Firma", receipt.IssuerName)
	}
	if receipt.GrossAmount != inv.GrossAmount {
		t.Errorf("Bruttobetrag %s, erwartet %s", receipt.GrossAmount, inv.GrossAmount)
	}
	if !strings.Contains(receipt.Subject, inv.InvoiceNumber) {
		t.Errorf("Betreff %q nennt die Rechnungsnummer nicht", receipt.Subject)
	}
	if !receipt.HasHeader() {
		t.Error("der Beleg hat keine Kopfdaten und würde nach der Altform gehasht")
	}
	if err := receipt.ValidateHeader(); err != nil {
		t.Errorf("der gebuchte Beleg muss die Kopfdatenprüfung bestehen: %v", err)
	}
}

// Der Eigenbeleg einer Abschlussbuchung hat Belegdatum, Betreff und Betrag.
func TestClosingSelfIssuedVoucherCarriesHeaderData(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	receipt, err := selfIssuedVoucher(ctx, env.receipts, 2026, closingVoucher{
		Kind: "rueckstellung", FiscalYear: 2026, Date: "2026-12-31",
		Description: "Rückstellung für Abschlusskosten",
		Explanation: "Aufwand des Jahres, Rechnung folgt im Folgejahr",
		Lines: []domain.JournalLine{
			{Position: 1, Side: domain.SideDebit, Account: "6825", Amount: 150000},
			{Position: 2, Side: domain.SideCredit, Account: "3070", Amount: 150000},
		},
	})
	if err != nil {
		t.Fatalf("Eigenbeleg: %v", err)
	}

	if receipt.DocumentDate != "2026-12-31" {
		t.Errorf("Belegdatum %q, erwartet das Buchungsdatum", receipt.DocumentDate)
	}
	if receipt.Subject != "Rückstellung für Abschlusskosten" {
		t.Errorf("Betreff %q, erwartet die Bezeichnung", receipt.Subject)
	}
	if receipt.GrossAmount != 150000 {
		t.Errorf("Betrag %s, erwartet 1500,00 € aus der Sollseite", receipt.GrossAmount)
	}
	if err := receipt.ValidateHeader(); err != nil {
		t.Errorf("der Eigenbeleg muss die Kopfdatenprüfung bestehen: %v", err)
	}
}

// Der Journaldienst ist der einzige Schreibweg: er weist eine Buchung ab, deren
// Beleg keine Kopfdaten hat — auf jedem Weg und nicht nur im Dialog.
func TestPostRejectsAnEntryWhoseReceiptHasNoHeader(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.journal.SetReceiptRepo(env.receiptRepo)

	// Ein Beleg ohne Kopfdaten — der Papierscan, den jemand abgelegt, aber noch
	// nicht erfasst hat.
	receipt := filePDFReceipt(t, env, FileReceiptRequest{})
	if receipt.HasHeader() {
		t.Fatal("der Beleg des Falls darf keine Kopfdaten haben")
	}

	entry := &domain.JournalEntry{
		FiscalYear: 2026, BookingDate: "2026-03-01", DocumentDate: "2026-03-01",
		Description: "Wareneingang", Source: domain.EntrySourceManual, TaxTreatment: domain.TaxTreatmentNotTaxable,
		Currency: "EUR", ExchangeRateMicros: 1_000_000,
		ReceiptID: &receipt.ID, ReceiptHash: receipt.ReceiptHash,
		Lines: []domain.JournalLine{
			{Position: 1, Side: domain.SideDebit, Account: "6815", Amount: 10000},
			{Position: 2, Side: domain.SideCredit, Account: "1800", Amount: 10000},
		},
	}
	if _, err := env.journal.Post(ctx, entry); err == nil {
		t.Fatal("eine Buchung auf einen Beleg ohne Kopfdaten muss abgewiesen werden")
	} else if !strings.Contains(err.Error(), "Belegdatum") {
		t.Errorf("die Meldung muss sagen, was fehlt: %v", err)
	}

	// Mit erfassten Kopfdaten geht dieselbe Buchung durch.
	if _, err := env.receipts.SaveHeader(ctx, receipt.ID, domain.ReceiptHeader{
		DocumentDate: "2026-03-01", IssuerName: "Lieferant GmbH",
		GrossAmount: 11900, TaxAmount: 1900, Currency: "EUR",
	}); err != nil {
		t.Fatalf("Kopfdaten erfassen: %v", err)
	}
	filled, _ := env.receipts.Get(ctx, receipt.ID)
	entry.ReceiptHash = filled.ReceiptHash
	if _, err := env.journal.Post(ctx, entry); err != nil {
		t.Fatalf("mit Kopfdaten muss die Buchung durchgehen: %v", err)
	}
}

// Die Kopfdatenerfassung selbst wird mit Vorher/Nachher protokolliert: die
// Kopfdaten gehen in den Beleg-Hash und damit in jede Buchung darauf.
func TestSaveHeaderLogsBeforeAndAfter(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	receipt := filePDFReceipt(t, env, FileReceiptRequest{})
	if _, err := env.receipts.SaveHeader(ctx, receipt.ID, domain.ReceiptHeader{
		DocumentDate: "2026-03-01", IssuerName: "Lieferant GmbH",
		GrossAmount: 11900, TaxAmount: 1900, Currency: "EUR",
	}); err != nil {
		t.Fatalf("Kopfdaten erfassen: %v", err)
	}

	entry := latestAudit(t, env, "RECEIPT")
	after := fields(t, entry.After)
	if after["documentDate"] != "2026-03-01" || after["issuerName"] != "Lieferant GmbH" {
		t.Errorf("die erfassten Kopfdaten fehlen im Protokoll: %v", after)
	}
	if _, ok := after["receiptHash"]; !ok {
		t.Errorf("der neu gerechnete Beleg-Hash gehört ins Protokoll: %v", after)
	}
	if _, ok := after["files"]; ok {
		t.Errorf("die Dateiliste gehört nicht ins Vorher/Nachher der Kopfdaten: %v", after)
	}
}

// --- Abstände als Kennzahlen des Prüflaufs ---------------------------------

// Die Abstände Belegdatum → Erfassung → Festschreibung stehen als Kennzahlen am
// Prüflauf. Der Journalexport beantwortet die Frage je Buchung; über den
// Zeitraum beantwortet sie nur der Lauf.
func TestCheckRunReportsCaptureAndCommitmentIntervals(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Drei Buchungen mit festen Abständen: 2, 4 und 30 Tage zwischen
	// Belegdatum und Erfassung. Der Median ist damit 4, das Maximum 30, und
	// über der Frist von zehn Tagen liegt genau eine.
	created := time.Date(2026, 3, 31, 9, 0, 0, 0, time.UTC)
	for _, documentDate := range []string{"2026-03-29", "2026-03-27", "2026-03-01"} {
		entry := &domain.JournalEntry{
			FiscalYear: 2026, BookingDate: "2026-03-31", DocumentDate: documentDate,
			ServiceDateFrom: documentDate, ServiceDateTo: documentDate,
			Description: "Wareneingang", Source: domain.EntrySourceManual, TaxTreatment: domain.TaxTreatmentNotTaxable,
			Kind: domain.EntryKindNormal, Currency: "EUR", ExchangeRateMicros: 1_000_000,
			CreatedAt: created,
			Lines: []domain.JournalLine{
				{Position: 1, Side: domain.SideDebit, Account: "6815", Amount: 10000},
				{Position: 2, Side: domain.SideCredit, Account: "1800", Amount: 10000},
			},
		}
		if err := env.journalRepo.Append(ctx, entry, accounting.NewHashChain().CalculateHash); err != nil {
			t.Fatalf("Buchung anlegen: %v", err)
		}
	}

	// Eine davon ist festgeschrieben, und zwar sieben Tage nach der Erfassung.
	committed := created.AddDate(0, 0, 7)
	if _, err := env.journalRepo.MarkCommitted(ctx, 2026, "2026-03-31", 0, committed); err != nil {
		t.Fatalf("Festschreibungsstempel: %v", err)
	}

	checks := env.checks(t)
	run, err := checks.Preview(ctx, CheckRequest{CutoffDate: "2026-03-31"})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}

	tl := run.Timeliness
	if tl.MeasuredEntries != 3 {
		t.Fatalf("gemessene Buchungen %d, erwartet 3", tl.MeasuredEntries)
	}
	if tl.CaptureDaysMedian != 4 {
		t.Errorf("Median Belegdatum → Erfassung %d, erwartet 4", tl.CaptureDaysMedian)
	}
	if tl.CaptureDaysMax != 30 {
		t.Errorf("Maximum Belegdatum → Erfassung %d, erwartet 30", tl.CaptureDaysMax)
	}
	if tl.CaptureLimitDays != 10 || tl.LateEntries != 1 {
		t.Errorf("über der Frist von %d Tagen liegen %d Buchungen, erwartet 1 bei 10 Tagen",
			tl.CaptureLimitDays, tl.LateEntries)
	}
	if tl.CommittedEntries != 3 || tl.UncommittedEntries != 0 {
		t.Errorf("festgeschrieben %d, offen %d, erwartet 3 und 0",
			tl.CommittedEntries, tl.UncommittedEntries)
	}
	if tl.CommitDaysMedian != 7 || tl.CommitDaysMax != 7 {
		t.Errorf("Abstand Erfassung → Festschreibung %d/%d Tage, erwartet 7/7",
			tl.CommitDaysMedian, tl.CommitDaysMax)
	}
}

// --- UTC in den Persistenzpfaden -------------------------------------------

// Zeitpunkte, die gespeichert werden, stehen in UTC. Eine Ortszeit ohne Zone ist
// in der Nacht der Zeitumstellung mehrdeutig, und die Reihenfolge der
// Aufzeichnungen richtet sich nach ihr.
func TestPersistedTimestampsAreUTC(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Sicherungslauf.
	backupRepo := repository.NewBackupRunRepository(env.db)
	run := &domain.BackupRun{
		Kind: domain.BackupKindManual, StartedAt: time.Now().UTC(),
		FinishedAt: time.Now().UTC(), Success: true, Target: "/media/sicherung/x.zip",
	}
	if err := backupRepo.Create(ctx, run); err != nil {
		t.Fatalf("Sicherungslauf schreiben: %v", err)
	}
	runs, err := backupRepo.FindRecent(ctx, 1)
	if err != nil || len(runs) == 0 {
		t.Fatalf("Sicherungslauf lesen: %v", err)
	}
	if _, offset := runs[0].StartedAt.Zone(); offset != 0 {
		t.Errorf("der Startzeitpunkt des Sicherungslaufs hat den Zonenversatz %d", offset)
	}

	// Voranmeldung.
	vatRepo := repository.NewVatReturnRepository(env.db)
	rec := &domain.VatReturn{
		FiscalYear: 2026, PeriodType: domain.VatPeriodMonth, PeriodKey: "2026-03",
		PeriodFrom: "2026-03-01", PeriodTo: "2026-03-31", Status: domain.VatReturnDraft,
	}
	if err := vatRepo.Create(ctx, rec); err != nil {
		t.Fatalf("Voranmeldung schreiben: %v", err)
	}
	if _, offset := rec.CreatedAt.Zone(); offset != 0 {
		t.Errorf("der Zeitpunkt der Voranmeldung hat den Zonenversatz %d", offset)
	}

	// Prüflauf.
	checkRepo := repository.NewCheckRunRepository(env.db)
	checkRun := &domain.CheckRun{FiscalYear: 2026, CutoffDate: "2026-03-31"}
	if err := checkRepo.Create(ctx, checkRun); err != nil {
		t.Fatalf("Prüflauf schreiben: %v", err)
	}
	if _, offset := checkRun.CreatedAt.Zone(); offset != 0 {
		t.Errorf("der Zeitpunkt des Prüflaufs hat den Zonenversatz %d", offset)
	}

	// Einstellung.
	settingsRepo := repository.NewSettingsRepository(env.db)
	if err := settingsRepo.Set(ctx, "test_key", "wert"); err != nil {
		t.Fatalf("Einstellung schreiben: %v", err)
	}
	var item domain.SettingItem
	if err := env.db.Where("key = ?", "test_key").Take(&item).Error; err != nil {
		t.Fatalf("Einstellung lesen: %v", err)
	}
	if _, offset := item.UpdatedAt.Zone(); offset != 0 {
		t.Errorf("der Zeitpunkt der Einstellung hat den Zonenversatz %d", offset)
	}

	// Zeitpunkte, die als Zeichenkette gespeichert oder ausgegeben werden,
	// führen ihre Zone mit: eine Ortszeit ohne Zonenangabe ist außerhalb des
	// Rechners, auf dem sie entstand, nicht mehr eindeutig.
	utcString := func(label, value string) {
		t.Helper()
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			t.Errorf("%s: %q ist kein RFC-3339-Zeitpunkt: %v", label, value, err)
			return
		}
		if _, offset := parsed.Zone(); offset != 0 {
			t.Errorf("%s: %q hat den Zonenversatz %d", label, value, offset)
		}
	}

	// Lückenvermerk am Nummernkreis.
	if err := env.invoicesWired(t).RecordNumberGapReason(
		ctx, 2026, 7, domain.NumberGapAborted, "Erfassung abgebrochen"); err != nil {
		t.Fatalf("Lückengrund festhalten: %v", err)
	}
	gaps, err := repository.NewNumberGapRepository(env.db).FindByYear(ctx, domain.NumberRangeInvoice, 2026)
	if err != nil || len(gaps) == 0 {
		t.Fatalf("Lückenvermerk lesen: %v (%d)", err, len(gaps))
	}
	utcString("Lückenvermerk", gaps[0].RecordedAt)

	// Prüfzeitpunkte der beiden Integritätsprüfungen.
	fileCheck, err := env.receipts.VerifyReceiptFiles(ctx)
	if err != nil {
		t.Fatalf("Belegdateien prüfen: %v", err)
	}
	utcString("Belegdateiprüfung", fileCheck.CheckedAt)

	integrity, err := env.journal.VerifyIntegrity(ctx)
	if err != nil {
		t.Fatalf("Kette prüfen: %v", err)
	}
	utcString("Kettenprüfung", integrity.CheckedAt)
}

// --- Fehlende SKR04-Konten werden auch in einer vollen Datei ergänzt -------

// Der frühere Schwellwert von hundert Konten war die Bedingung, unter der
// ein neuer SKR04-Stand nicht mehr in eine bestehende Datei kam.
func TestSeedAddsMissingAccountsEvenInAFullChart(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	accountRepo := repository.NewAccountRepository(env.db)
	all, err := accountRepo.FindAll(ctx)
	if err != nil {
		t.Fatalf("Kontenplan lesen: %v", err)
	}
	if len(all) < 100 {
		t.Fatalf("der Test braucht einen vollen Kontenplan, gefunden %d Konten", len(all))
	}

	// Ein einzelnes Konto verschwindet — so, wie es fehlte, wenn der
	// Kontenrahmen später um es erweitert wird.
	victim := all[len(all)/2].Number
	if err := env.db.Exec("DELETE FROM accounts WHERE number = ?", victim).Error; err != nil {
		t.Fatalf("Konto entfernen: %v", err)
	}

	if err := repository.SeedDefaultsIfEmpty(ctx, env.db, 2026); err != nil {
		t.Fatalf("Konten ergänzen: %v", err)
	}
	if _, err := accountRepo.FindByNumber(ctx, victim); err != nil {
		t.Errorf("das fehlende Konto %s wurde nicht ergänzt: %v", victim, err)
	}

	after, err := accountRepo.FindAll(ctx)
	if err != nil {
		t.Fatalf("Kontenplan lesen: %v", err)
	}
	if len(after) != len(all) {
		t.Errorf("nach dem Ergänzen stehen %d Konten da, vorher waren es %d", len(after), len(all))
	}
}

// --- Vorher/Nachher an der Voranmeldung ------------------------------------

// Die Bestätigung einer Voranmeldung ist ein Statuswechsel und wird mit beiden
// Ständen protokolliert; die Bearbeiterkennung steht an der Anmeldung selbst.
func TestVatReturnSubmissionLogsBeforeAndAfter(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	vatRepo := repository.NewVatReturnRepository(env.db)
	rec := &domain.VatReturn{
		FiscalYear: 2026, PeriodType: domain.VatPeriodMonth, PeriodKey: "2026-03",
		PeriodFrom: "2026-03-01", PeriodTo: "2026-03-31", Status: domain.VatReturnDraft,
	}
	if err := vatRepo.Create(ctx, rec); err != nil {
		t.Fatalf("Entwurf anlegen: %v", err)
	}

	before := *rec
	rec.Status = domain.VatReturnSubmitted
	rec.SubmittedAt = "2026-04-10"
	rec.TransferTicket = "TT-4711"
	if err := vatRepo.Update(ctx, rec); err != nil {
		t.Fatalf("Bestätigung speichern: %v", err)
	}
	if err := repository.NewAuditRepository(env.db).LogChange(ctx, domain.AuditActionUpdate,
		"VAT_RETURN", "1", "Voranmeldung übermittelt", &before, rec); err != nil {
		t.Fatalf("Protokolleintrag: %v", err)
	}

	entry := latestAudit(t, env, "VAT_RETURN")
	beforeFields := fields(t, entry.Before)
	afterFields := fields(t, entry.After)
	if beforeFields["status"] != string(domain.VatReturnDraft) ||
		afterFields["status"] != string(domain.VatReturnSubmitted) {
		t.Errorf("der Statuswechsel fehlt im Protokoll: %v → %v", beforeFields, afterFields)
	}
	if _, ok := afterFields["figures"]; ok {
		t.Errorf("die Kennziffern gehören nicht ins Vorher/Nachher: %v", afterFields)
	}
}

// Die Kennzahlen werden mit dem Lauf abgelegt und wieder gelesen: sie stehen als
// Spalten am Prüflauf, nicht nur im Speicher des Aufrufs.
func TestCheckRunTimelinessSurvivesTheRepository(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	checks := env.checks(t)
	if _, err := checks.Run(ctx, CheckRequest{CutoffDate: "2026-03-31"}); err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
	runs, err := checks.Runs(ctx, 2026)
	if err != nil || len(runs) == 0 {
		t.Fatalf("Läufe lesen: %v (%d Läufe)", err, len(runs))
	}
	if runs[0].Timeliness.CaptureLimitDays != 10 {
		t.Errorf("die Erfassungsfrist überlebt die Ablage nicht: %+v", runs[0].Timeliness)
	}
}
