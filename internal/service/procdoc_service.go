package service

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/buildinfo"
	"github.com/buchfink/buchfink/internal/changelog"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/procdoc"
	"github.com/buchfink/buchfink/internal/receiptstore"
	"github.com/buchfink/buchfink/internal/timestamp"
)

// ProcDocCategory ist der Zweig des Belegspeichers, unter dem die Fassungen
// der Verfahrensdokumentation liegen.
const ProcDocCategory = "verfahrensdokumentation"

// ProcDocEnvironment sind die Angaben, die nicht aus der Datenbank kommen.
//
// Sie werden übergeben und nicht ermittelt: der Datenordner und das
// Sicherungsziel kennt die Bridge, nicht der Dienst — und ein Dienst, der sich
// seine Umgebung selbst zusammensucht, lässt sich nicht prüfen.
type ProcDocEnvironment struct {
	DataDir      string
	BackupDir    string
	BackupRhythm string
	BackupRuns   []domain.BackupRun
}

// ProcDocResult ist die erzeugte Fassung mit ihrem Inhalt.
type ProcDocResult struct {
	Document domain.ProcedureDocumentation `json:"document"`
	// PDFNote sagt, warum kein PDF entstanden ist; leer heißt: es ist
	// entstanden. Ein Satzfehler hält die Fassung nicht auf — das Markdown
	// trägt die Aussage —, aber er wird auch nicht verschwiegen.
	PDFNote string `json:"pdfNote,omitempty"`
	// Markdown ist der erzeugte Text. Er geht mit zurück, damit die Oberfläche
	// ihn anzeigen kann, ohne die abgelegte Datei erneut zu lesen.
	Markdown string `json:"markdown"`
	Message  string `json:"message"`
}

// ProcDocService erzeugt die Verfahrensdokumentation und legt jede Fassung ab.
type ProcDocService struct {
	settingsRepo  domain.SettingsRepository
	numberRepo    domain.NumberRangeRepository
	procDocRepo   domain.ProcedureDocumentationRepository
	migrationRepo domain.MigrationRepository
	auditRepo     domain.AuditRepository
	store         *receiptstore.Store
	renderer      DocumentRenderer
	env           ProcDocEnvironment
	fiscalYear    int
}

// NewProcDocService verdrahtet die Erzeugung der Verfahrensdokumentation.
func NewProcDocService(
	settingsRepo domain.SettingsRepository,
	numberRepo domain.NumberRangeRepository,
	procDocRepo domain.ProcedureDocumentationRepository,
	migrationRepo domain.MigrationRepository,
	auditRepo domain.AuditRepository,
	store *receiptstore.Store,
	fiscalYear int,
) *ProcDocService {
	return &ProcDocService{
		settingsRepo: settingsRepo, numberRepo: numberRepo,
		procDocRepo: procDocRepo, migrationRepo: migrationRepo,
		auditRepo: auditRepo, store: store, fiscalYear: fiscalYear,
	}
}

// SetEnvironment reicht die Angaben nach, die außerhalb der Datenbank liegen.
func (s *ProcDocService) SetEnvironment(env ProcDocEnvironment) { s.env = env }

// SetRenderer hängt den Dokumentensetzer an, mit dem die Fassung zusätzlich als
// PDF entsteht. Ohne ihn bleibt es beim Markdown.
func (s *ProcDocService) SetRenderer(r DocumentRenderer) { s.renderer = r }

// SetFiscalYear setzt das Geschäftsjahr, für das erzeugt wird.
func (s *ProcDocService) SetFiscalYear(year int) { s.fiscalYear = year }

// OrganisationTexts liest die unternehmensindividuellen Freitexte, mit Mustern
// dort, wo noch nichts erfasst ist.
func (s *ProcDocService) OrganisationTexts(ctx context.Context) domain.OrganisationTexts {
	texts := domain.DefaultOrganisationTexts()
	if s.settingsRepo == nil {
		return texts
	}
	read := func(key, fallback string) string {
		if value, err := s.settingsRepo.Get(ctx, key); err == nil && value != "" {
			return value
		}
		return fallback
	}
	texts.Responsibilities = read(domain.SettingOrgResponsibilities, texts.Responsibilities)
	texts.ReceiptFlow = read(domain.SettingOrgReceiptFlow, texts.ReceiptFlow)
	texts.Scanning = read(domain.SettingOrgScanning, texts.Scanning)
	texts.Approval = read(domain.SettingOrgApproval, texts.Approval)
	texts.Substitution = read(domain.SettingOrgSubstitution, texts.Substitution)
	texts.Backup = read(domain.SettingOrgBackup, texts.Backup)
	texts.Notes = read(domain.SettingOrgNotes, texts.Notes)
	return texts
}

// SaveOrganisationTexts schreibt die Freitexte und protokolliert die Änderung.
func (s *ProcDocService) SaveOrganisationTexts(ctx context.Context, texts domain.OrganisationTexts) error {
	before := s.OrganisationTexts(ctx)
	pairs := []struct {
		key   string
		value string
	}{
		{domain.SettingOrgResponsibilities, texts.Responsibilities},
		{domain.SettingOrgReceiptFlow, texts.ReceiptFlow},
		{domain.SettingOrgScanning, texts.Scanning},
		{domain.SettingOrgApproval, texts.Approval},
		{domain.SettingOrgSubstitution, texts.Substitution},
		{domain.SettingOrgBackup, texts.Backup},
		{domain.SettingOrgNotes, texts.Notes},
	}
	for _, pair := range pairs {
		if err := s.settingsRepo.Set(ctx, pair.key, pair.value); err != nil {
			return fmt.Errorf("die Organisationsanweisung konnte nicht gespeichert werden: %w", err)
		}
	}
	if s.auditRepo != nil {
		_ = s.auditRepo.LogChange(ctx, domain.AuditActionUpdate, "SETTINGS", "ORGANISATION",
			"Organisationsanweisung der Verfahrensdokumentation geändert", before, texts)
	}
	return nil
}

// Generate erzeugt eine Fassung, legt sie im Belegspeicher ab und protokolliert
// sie.
//
// now wird übergeben, damit sich die Fassungsbezeichnung im Test festlegen
// lässt; die Nullzeit heißt: jetzt.
//
// Erzeugt werden beide Formen: Markdown und PDF. Das Markdown ist die Quelle —
// maschinenlesbar, vergleichbar, ohne Satzprogramm zu öffnen —, das PDF ist der
// Satz genau dieses Textes über den vorhandenen Typst-Weg (procdoc.Typst,
// internal/invoice). Gesetzt wird das erzeugte Markdown und nicht ein zweites
// Mal die Eingabe: eine eigene Typst-Vorlage neben der Markdown-Vorlage wären
// zwei Fassungen desselben Dokuments, die auseinanderlaufen.
//
// Scheitert der Satz, gilt die Fassung trotzdem: der Nachweis nach GoBD Rz. 151
// hängt am Inhalt und nicht am Layout. Der Grund steht dann in PDFNote und im
// Protokoll, statt die Erzeugung an einer Formfrage scheitern zu lassen.
func (s *ProcDocService) Generate(ctx context.Context, now time.Time) (*ProcDocResult, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()

	input, err := s.buildInput(ctx, now)
	if err != nil {
		return nil, err
	}
	markdown, err := procdoc.Render(input)
	if err != nil {
		return nil, err
	}

	doc := domain.ProcedureDocumentation{
		Version:     input.Version,
		CreatedAt:   now,
		FiscalYear:  s.fiscalYear,
		CompanyName: input.CompanyName,
		AppVersion:  input.AppVersion,
		RuleVersion: input.RuleVersion,
		Actor:       input.Actor,
		FileName:    procdoc.FileName(input.CompanyName, input.Version),
		Size:        int64(len(markdown)),
	}

	// Abgelegt wird im Belegspeicher und nicht irgendwo im Datenordner: die
	// Fassungen sind über die Aufbewahrungsfrist zu halten, und der
	// Belegspeicher ist der Ort, den Sicherung, Archivexport und Prüferpaket
	// mitnehmen.
	if s.store != nil {
		stored, err := s.store.PutDocument(ProcDocCategory, doc.FileName, bytes.NewReader([]byte(markdown)))
		if err != nil {
			return nil, fmt.Errorf("die Verfahrensdokumentation konnte nicht abgelegt werden: %w", err)
		}
		doc.SHA256 = stored.SHA256
		doc.Size = stored.Size
		doc.StoredPath = stored.RelPath
	}

	pdfNote := s.renderPDF(ctx, &doc, markdown, input.CompanyName)

	if err := s.procDocRepo.Create(ctx, &doc); err != nil {
		return nil, fmt.Errorf("die Fassung konnte nicht festgehalten werden: %w", err)
	}
	if s.auditRepo != nil {
		note := fmt.Sprintf("Verfahrensdokumentation %s erzeugt und als %s abgelegt (SHA256 %s)",
			doc.Version, doc.FileName, doc.SHA256)
		if doc.PDFStoredPath != "" {
			note += fmt.Sprintf("; PDF %s (SHA256 %s)", doc.PDFFileName, doc.PDFSHA256)
		} else if pdfNote != "" {
			note += "; " + pdfNote
		}
		_ = s.auditRepo.Log(ctx, domain.AuditActionCreate, "PROCEDURE_DOC", doc.Version, note)
	}

	form := "als Markdown und PDF"
	if doc.PDFStoredPath == "" {
		form = "als Markdown"
	}
	return &ProcDocResult{
		Document: doc,
		PDFNote:  pdfNote,
		Markdown: markdown,
		Message: fmt.Sprintf(
			"Verfahrensdokumentation %s erzeugt. Die Fassung liegt %s im Belegspeicher unter dokumente/%s und bleibt dort über die Aufbewahrungsfrist.",
			doc.Version, form, ProcDocCategory),
	}, nil
}

// renderPDF setzt die Fassung als PDF und legt es neben dem Markdown ab.
//
// Der Rückgabewert ist der Grund, warum kein PDF entstanden ist; leer heißt: es
// ist entstanden. Ein Fehler wird nicht durchgereicht — er nähme der
// Verfahrensdokumentation die Erzeugung, obwohl ihr Inhalt fertig ist —, aber
// er wird auch nicht verschwiegen: er steht im Ergebnis und im Protokoll.
func (s *ProcDocService) renderPDF(
	ctx context.Context, doc *domain.ProcedureDocumentation, markdown, company string,
) string {
	if s.renderer == nil {
		return "Der Dokumentensetzer ist nicht verfügbar; die Fassung liegt nur als Markdown vor."
	}
	if s.store == nil {
		return "Ohne Belegspeicher lässt sich das PDF nicht ablegen; die Fassung liegt nur als Markdown vor."
	}
	// Die Fassungsbezeichnung als Kennung: zwei Läufe derselben Fassung sind
	// damit byteweise vergleichbar (GoBD Rz. 76 Abs. 2).
	pdf, err := s.renderer.RenderDocumentPDF(ctx,
		procdoc.Typst(markdown, fmt.Sprintf("Verfahrensdokumentation %s — %s", doc.Version, company), doc.CreatedAt),
		"Verfahrensdokumentation "+doc.Version)
	if err != nil {
		return fmt.Sprintf("Das PDF konnte nicht gesetzt werden: %v. Die Fassung liegt als Markdown vor.", err)
	}
	name := procdoc.PDFFileName(company, doc.Version)
	stored, err := s.store.PutDocument(ProcDocCategory, name, bytes.NewReader(pdf))
	if err != nil {
		return fmt.Sprintf("Das PDF konnte nicht abgelegt werden: %v. Die Fassung liegt als Markdown vor.", err)
	}
	doc.PDFFileName = name
	doc.PDFStoredPath = stored.RelPath
	doc.PDFSHA256 = stored.SHA256
	doc.PDFSize = stored.Size
	return ""
}

// Documentations liefert die abgelegten Fassungen, die neueste zuerst.
func (s *ProcDocService) Documentations(ctx context.Context) ([]domain.ProcedureDocumentation, error) {
	return s.procDocRepo.FindAll(ctx)
}

// buildInput sammelt die veränderlichen Angaben.
func (s *ProcDocService) buildInput(ctx context.Context, now time.Time) (procdoc.Input, error) {
	settings, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return procdoc.Input{}, fmt.Errorf("die Stammdaten konnten nicht gelesen werden: %w", err)
	}

	version, err := s.nextVersion(ctx, now)
	if err != nil {
		return procdoc.Input{}, err
	}

	year := s.fiscalYear
	if year == 0 {
		year = settings.FiscalYear
	}
	from, to := fiscalYearBounds(year, settings.FiscalYearStartMonth)

	input := procdoc.Input{
		CompanyName:      settings.CompanyName,
		LegalForm:        settings.LegalForm,
		Street:           settings.Street,
		ZipCity:          settings.ZipCity,
		Country:          settings.Country,
		TaxNumber:        settings.TaxNumber,
		VatID:            settings.VatID,
		TaxOffice:        settings.TaxOffice,
		FiscalYear:       year,
		FiscalYearFrom:   from,
		FiscalYearTo:     to,
		DataDir:          s.env.DataDir,
		CloudWarning:     domain.CloudFolderWarning(s.env.DataDir),
		SKR:              settings.SKR,
		TaxCases:         coveredTaxCases(),
		ExcludedCases:    domain.TaxCaseHints(),
		CaptureDays:      settings.ReceiptCaptureDays,
		TaxationType:     taxationLabel(settings.TaxationType),
		VatPeriod:        vatPeriodLabel(settings.VatPeriod),
		LegalFormNote:    domain.LegalFormLimitationNote(settings.LegalForm),
		AppVersion:       buildinfo.Version,
		RuleVersion:      accounting.PostingRuleVersion,
		TSAName:          timestamp.DefaultTSA,
		ExportFormats:    exportFormats(),
		BackupDir:        s.env.BackupDir,
		BackupRhythm:     s.env.BackupRhythm,
		ChangelogTable:   changelog.Markdown(),
		CheckRules:       checkRuleCatalog(),
		Version:          version,
		CreatedAt:        now,
		Actor:            actor.Actor(),
		SystemChangeDate: s.systemChangeDate(ctx),
	}

	input.Organisation = procdoc.Organisation(s.OrganisationTexts(ctx))
	input.NumberRanges = s.numberRanges(ctx, settings, year)
	input.RetentionRows = retentionRows()
	input.Migrations = s.migrations(ctx)
	for i := range s.env.BackupRuns {
		run := &s.env.BackupRuns[i]
		result := "fehlgeschlagen"
		if run.Success {
			result = "erfolgreich"
		}
		input.BackupRuns = append(input.BackupRuns, procdoc.BackupRun{
			At:      run.StartedAt.UTC().Format("02.01.2006 15:04"),
			Result:  result,
			Target:  run.Target,
			SizeKiB: run.Bytes / 1024,
		})
	}
	return input, nil
}

// nextVersion bildet die Fassungsbezeichnung "YYYY-MM-DD-n".
func (s *ProcDocService) nextVersion(ctx context.Context, now time.Time) (string, error) {
	day := now.Format("2006-01-02")
	count, err := s.procDocRepo.CountForDay(ctx, day)
	if err != nil {
		return "", fmt.Errorf("die bisherigen Fassungen ließen sich nicht zählen: %w", err)
	}
	return fmt.Sprintf("%s-%d", day, count+1), nil
}

func (s *ProcDocService) systemChangeDate(ctx context.Context) string {
	if s.settingsRepo == nil {
		return ""
	}
	value, err := s.settingsRepo.Get(ctx, domain.SettingSystemChangeDate)
	if err != nil {
		return ""
	}
	return value
}

func (s *ProcDocService) migrations(ctx context.Context) []procdoc.Migration {
	if s.migrationRepo == nil {
		return nil
	}
	records, err := s.migrationRepo.FindSchemaMigrations(ctx)
	if err != nil {
		return nil
	}
	out := make([]procdoc.Migration, 0, len(records))
	for i := range records {
		out = append(out, procdoc.Migration{
			At:          records[i].RunAt.UTC().Format("02.01.2006 15:04"),
			AppVersion:  records[i].AppVersion,
			FromVersion: records[i].FromVersion,
			ToVersion:   records[i].ToVersion,
			Result:      records[i].Result,
		})
	}
	return out
}

// numberRanges beschreibt die Nummernkreise mit ihrer Systematik.
func (s *ProcDocService) numberRanges(ctx context.Context, settings *domain.CompanySettings, year int) []procdoc.NumberRange {
	invoiceFormat := settings.InvoiceNumberFormat
	if invoiceFormat == "" {
		invoiceFormat = domain.DefaultInvoiceNumberFormat
	}
	ranges := []struct {
		name   string
		key    domain.NumberRangeKey
		format string
		year   int
		scope  string
	}{
		{"Buchungsnummer", domain.NumberRangeJournal, "{JAHR}-{NR:6}", year, "je Geschäftsjahr"},
		{"Eingangsbeleg", domain.NumberRangeReceipt, "ER-{JAHR}-{NR:4}", year, "je Geschäftsjahr"},
		{"Ausgangsrechnung", domain.NumberRangeInvoice, invoiceFormat, year, "je Geschäftsjahr"},
		{"Debitorenkonto", domain.NumberRangeDebitor, "10000–69999", 0, "jahresübergreifend"},
		{"Kreditorenkonto", domain.NumberRangeCreditor, "70000–99999", 0, "jahresübergreifend"},
	}

	out := make([]procdoc.NumberRange, 0, len(ranges))
	for _, r := range ranges {
		next := "—"
		if s.numberRepo != nil {
			if value, err := s.numberRepo.Peek(ctx, r.key, r.year); err == nil {
				next = fmt.Sprintf("%d", value)
			}
		}
		out = append(out, procdoc.NumberRange{Name: r.name, Format: r.format, Next: next, Scope: r.scope})
	}
	return out
}

// retentionRows bringt das Löschkonzept in die Form der Vorlage.
func retentionRows() []procdoc.RetentionRow {
	concept := accounting.DeletionConcept()
	out := make([]procdoc.RetentionRow, 0, len(concept))
	for _, row := range concept {
		out = append(out, procdoc.RetentionRow{
			Category:   row.Category,
			Class:      row.Class.Label(),
			Years:      row.Years,
			LegalBasis: row.LegalBasis,
			Note:       row.Note,
		})
	}
	return out
}

// checkRuleCatalog ist das interne Kontrollsystem in Regelform.
//
// Der Katalog steht hier und nicht im Prüfdienst, weil er dort als Verhalten
// existiert und nicht als Liste: jede Regel ist eine Funktion. Die
// Verfahrensdokumentation braucht sie als Aufzählung mit ihrem Zweck, und die
// Schlüssel sind dieselben Konstanten — ein Tippfehler fiele beim Übersetzen
// auf.
func checkRuleCatalog() []procdoc.CheckRule {
	return []procdoc.CheckRule{
		{Key: domain.CheckRuleEntryWithoutReceipt, Severity: "Blockierend", Purpose: "Keine Buchung ohne Beleg (§ 146 Abs. 1 AO, GoBD Rz. 61)."},
		{Key: domain.CheckRuleReceiptUnbooked, Severity: "Hinweis", Purpose: "Abgelegte, noch nicht gebuchte Belege."},
		{Key: domain.CheckRuleReceiptOverdue, Severity: "Blockierend", Purpose: "Belege, die länger als die Erfassungsfrist offen sind (GoBD Rz. 47)."},
		{Key: domain.CheckRuleBankUnmatched, Severity: "Hinweis", Purpose: "Bankumsätze ohne Buchung."},
		{Key: domain.CheckRuleInterimBalance, Severity: "Blockierend", Purpose: "Saldo auf dem Geldtransit- oder Interimskonto."},
		{Key: domain.CheckRuleDuplicateReceipt, Severity: "Hinweis", Purpose: "Derselbe Beleg zweimal abgelegt."},
		{Key: domain.CheckRuleDuplicatePayment, Severity: "Hinweis", Purpose: "Dieselbe Zahlung zweimal gebucht."},
		{Key: domain.CheckRuleNumberGap, Severity: "Blockierend", Purpose: "Lücke im Nummernkreis ohne Vermerk."},
		{Key: domain.CheckRuleAccountUnmapped, Severity: "Blockierend", Purpose: "Konto ohne Zuordnung zu einer Bilanz- oder GuV-Position."},
		{Key: domain.CheckRuleDepreciationMissing, Severity: "Blockierend", Purpose: "Fällige Abschreibung noch nicht gebucht."},
		{Key: domain.CheckRuleVatReturnMissing, Severity: "Blockierend", Purpose: "Voranmeldung des Zeitraums fehlt."},
		{Key: domain.CheckRuleCommitOverdue, Severity: "Hinweis", Purpose: "Zeitraum überfällig festzuschreiben."},
		{Key: domain.CheckRuleProvisionDiscount, Severity: "Hinweis", Purpose: "Rückstellung ohne Abzinsungssatz des Stichtagsmonats (§ 253 Abs. 2 HGB)."},
		{Key: domain.CheckRuleClosingStepSkipped, Severity: "Hinweis", Purpose: "Übersprungener Abschlussbaustein mit seinem Grund."},
		{Key: domain.CheckRuleICSupplyEvidenceMissing, Severity: "Blockierend", Purpose: "Innergemeinschaftliche Lieferung ohne Belegnachweis (§§ 17a bis 17c UStDV)."},
		{Key: domain.CheckRuleICSupplyUnconfirmed, Severity: "Hinweis", Purpose: "USt-IdNr. beim Ausstellen nicht bestätigt (§ 18e UStG)."},
	}
}

// coveredTaxCases sind die Steuerfälle im Funktionsumfang.
func coveredTaxCases() []string {
	return []string{
		"Steuerpflichtige Umsätze zu 19 % und 7 % sowie steuerfreie Umsätze mit und ohne Vorsteuerabzug",
		"Innergemeinschaftliche Lieferungen und Erwerbe, Zusammenfassende Meldung",
		"Reverse-Charge nach § 13b UStG auf der Eingangs- und der Ausgangsseite",
		"Ausfuhrlieferungen in das Drittland und Einfuhrumsatzsteuer",
		"Anzahlungen mit Steuerentstehung nach § 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG",
		"Vorsteuerberichtigung nach § 15a UStG",
		"Nicht abziehbare Betriebsausgaben nach § 4 Abs. 5 EStG (Geschenke, Bewirtung)",
		"Anlagevermögen mit linearer Abschreibung, GWG und Sammelposten",
		"Fremdwährungsgeschäfte mit datierten Umrechnungskursen",
	}
}

// exportFormats nennt die Ausgabeformate mit ihrem Stand.
func exportFormats() []string {
	return []string{
		"Datenzugriff Z3: CSV je Tabelle mit index.xml nach dem Beschreibungsstandard, einschließlich Feldbeschreibung und Schlüsselverzeichnis",
		"Archivexport: derselbe Datenbestand mit den Belegdateien",
		"E-Bilanz: XBRL-Instanz aus der HGB-Gliederung",
		"Umsatzsteuer-Voranmeldung und Zusammenfassende Meldung als Datensatz mit Programmfassung und Regelstand",
		"Ausgangsrechnungen als ZUGFeRD (EN 16931) oder XRechnung",
	}
}

func taxationLabel(kind string) string {
	if kind == "IST" {
		return "Istversteuerung (§ 20 UStG)"
	}
	return "Sollversteuerung (§ 16 Abs. 1 Satz 1 UStG)"
}

func vatPeriodLabel(period string) string {
	switch period {
	case "month":
		return "monatlich"
	case "year":
		return "jährlich"
	default:
		return "vierteljährlich"
	}
}

// fiscalYearBounds liefert Anfang und Ende des Geschäftsjahres als ISO-Daten.
func fiscalYearBounds(year, startMonth int) (string, string) {
	if startMonth < 1 || startMonth > 12 {
		startMonth = 1
	}
	start := time.Date(year, time.Month(startMonth), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(1, 0, -1)
	return start.Format("2006-01-02"), end.Format("2006-01-02")
}
