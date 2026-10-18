package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// StatementPeriods liefert die benötigten Jahresdaten.
type StatementPeriods interface {
	PeriodSource
	EarliestFiscalYear(context.Context) (int, error)
}

// StatementService stellt Bilanz und Gewinn- und Verlustrechnung auf.
//
// Er beschafft, was die reine Gliederung in internal/accounting nicht beschaffen
// kann: die Salden zweier Geschäftsjahre, den Zeitraum des Jahres, die
// Unternehmensdaten für den Kopf und die offenen Posten für die
// Restlaufzeitengliederung. Gerechnet wird dort, nicht hier — und im Frontend
// gar nicht.
type StatementService struct {
	*StatementFigures
	auditRepo domain.AuditRepository
	openItems OpenItemSource
	notes     NotesSources
	renderer  DocumentRenderer
	deadlines StatementDeadlineSource
}

// StatementDeadlineSource liefert Aufstellungs- und Offenlegungsfrist aus dem
// Pflichtenkalender; der Abschluss rechnet sie nicht selbst.
type StatementDeadlineSource interface {
	DeadlinesOf(ctx context.Context, year int, kinds ...domain.ObligationKind) ([]domain.Deadline, error)
}

// Die drei Quellen des Anhangs. Jede ist der schmalste Ausschnitt des Dienstes,
// der sie führt — der Abschluss soll den Rückstellungsspiegel lesen können,
// ohne die Rückstellungsbuchhaltung mitzubringen.
type (
	// ProvisionMirrorSource liefert den Rückstellungsspiegel (§ 285 HGB).
	ProvisionMirrorSource interface {
		Mirror(ctx context.Context, year int) (*domain.ProvisionMirror, error)
	}
	// ReconciliationSource liefert die Überleitung zur Steuerbilanz.
	ReconciliationSource interface {
		Reconcile(ctx context.Context, year int) (*domain.Reconciliation, error)
	}
	// NotesTextSource liefert die Freitexte des Anhangs.
	NotesTextSource interface {
		NotesTexts(ctx context.Context, year int) ([]domain.NotesSectionText, error)
	}
)

// NotesSources bündelt sie, damit der Aufbau des Dienstes nicht drei Setzer
// braucht. Fehlt eine, bleibt ihr Teil des Anhangs leer statt zu scheitern.
type NotesSources struct {
	Provisions     ProvisionMirrorSource
	Reconciliation ReconciliationSource
	Texts          NotesTextSource
}

// OpenItemSource liefert die offenen Posten für die Restlaufzeitengliederung
// nach § 268 Abs. 4 und 5 HGB. Mehr braucht der Abschluss vom Zahlungsverkehr
// nicht.
//
// Gefragt wird zum Stichtag und nicht nach dem heutigen Stand: ein Abschluss
// ist eine Aussage über einen Tag, und was danach gezahlt wurde, gehört nicht
// hinein.
type OpenItemSource interface {
	OpenItemsAt(ctx context.Context, cutoff string) ([]domain.OpenItem, error)
}

// DocumentRenderer übersetzt eine Typst-Vorlage in ein PDF.
type DocumentRenderer interface {
	RenderDocumentPDF(ctx context.Context, template, ident string) ([]byte, error)
}

// NewStatementService wires the Jahresabschluss-Auswertung.
func NewStatementService(
	accountingSvc *AccountingService,
	periods StatementPeriods,
	settingsRepo domain.SettingsRepository,
	auditRepo domain.AuditRepository,
) *StatementService {
	return &StatementService{
		StatementFigures: &StatementFigures{accountingSvc: accountingSvc, periods: periods, settingsRepo: settingsRepo},
		auditRepo:        auditRepo,
	}
}

// SetOpenItemSource koppelt die offenen Posten an. Ohne sie entsteht der
// Abschluss wie sonst, nur ohne die Restlaufzeiten.
func (s *StatementService) SetOpenItemSource(src OpenItemSource) { s.openItems = src }

// SetNotesSources koppelt den Anhang an: Rückstellungsspiegel, Überleitung und
// Freitexte.
func (s *StatementService) SetNotesSources(src NotesSources) { s.notes = src }

// SetRenderer koppelt den PDF-Renderer an.
func (s *StatementService) SetRenderer(r DocumentRenderer) { s.renderer = r }

// SetDeadlineSource koppelt den Pflichtenkalender an. Ohne ihn nennt der
// Abschluss keine Fristen.
func (s *StatementService) SetDeadlineSource(src StatementDeadlineSource) { s.deadlines = src }

// Build stellt den Jahresabschluss eines Geschäftsjahres auf.
//
// depth darf leer sein; dann entscheidet die Größenklasse über die
// Gliederungstiefe, denn das ist die Tiefe, in der der Abschluss offenzulegen
// ist (§ 266 Abs. 1 Sätze 3 und 4 HGB).
func (s *StatementService) Build(ctx context.Context, year int, depth domain.StatementDepth) (*domain.FinancialStatement, error) {
	if err := requireYear(year); err != nil {
		return nil, err
	}

	// Die Größenklasse entsteht immer aus der vollen Gliederung: die Merkmale
	// des § 267 HGB hängen an der Bilanzsumme und den Umsatzerlösen, und die
	// dürfen nicht davon abhängen, wie tief jemand gerade hinsieht.
	full, err := s.statement(ctx, year, domain.DepthFull)
	if err != nil {
		return nil, err
	}
	sizeClass, err := s.sizeClassFrom(ctx, year, full)
	if err != nil {
		return nil, err
	}

	shown := full
	wanted := depth
	if wanted == "" {
		wanted = sizeClass.Obligations.Depth
	}
	if !wanted.Valid() {
		return nil, fmt.Errorf("unbekannte Gliederungstiefe %q", wanted)
	}
	if wanted != domain.DepthFull {
		shown, err = s.statement(ctx, year, wanted)
		if err != nil {
			return nil, err
		}
	}

	header, err := s.header(ctx, year, full)
	if err != nil {
		return nil, err
	}
	maturities, err := s.maturities(ctx, header.ClosingDate)
	if err != nil {
		return nil, err
	}

	out := &domain.FinancialStatement{
		Header:     header,
		Statement:  *shown,
		SizeClass:  sizeClass,
		Maturities: maturities,
		Notes:      s.notesFor(ctx, year),
		Deadlines:  s.deadlinesFor(ctx, year),
	}
	out.Notes.Missing = []string{}
	required := map[domain.NotesSection]bool{domain.NotesSectionContingent: true, domain.NotesSectionBoardLoans: true, domain.NotesSectionAdditional: true}
	if sizeClass.Obligations.NotesRequired {
		required[domain.NotesSectionMethods] = true
	}
	values := map[domain.NotesSection]domain.NotesSectionText{}
	for _, text := range out.Notes.Texts {
		values[text.Section] = text
	}
	for _, def := range domain.AllNotesSections() {
		if !required[def.Section] {
			continue
		}
		text := values[def.Section]
		if strings.TrimSpace(text.Text) == "" {
			out.Notes.Missing = append(out.Notes.Missing, def.Label)
		}
		if !sizeClass.Obligations.NotesRequired && strings.TrimSpace(text.Text) != "" {
			out.Notes.BelowBalance = append(out.Notes.BelowBalance, text)
		}
	}
	// Ohne Bilanzstichtag nennt die Größenklasse keine Frist; die Ansicht läuft
	// trotzdem über die Liste.
	out.EnsureLists()
	return out, nil
}

// Notes liefert den Anhang eines Geschäftsjahres — für die Ausgabewege, die
// nicht den ganzen Abschluss aufbauen, etwa die E-Bilanz.
func (s *StatementService) Notes(ctx context.Context, year int) domain.StatementNotes {
	return s.notesFor(ctx, year)
}

// notesFor stellt den Anhang zusammen.
//
// Fällt eine Quelle aus, bleibt ihr Abschnitt leer: ein Abschluss, der wegen
// eines fehlenden Anhangtextes gar nicht mehr entsteht, hilft niemandem. Was
// fehlt, sieht man daran, dass die Tabelle leer ist.
func (s *StatementService) notesFor(ctx context.Context, year int) domain.StatementNotes {
	notes := domain.StatementNotes{
		Texts: make([]domain.NotesSectionText, 0),
		ProvisionMirror: domain.ProvisionMirror{
			FiscalYear: year, Rows: make([]domain.ProvisionMirrorRow, 0),
		},
		Reconciliation: domain.Reconciliation{
			FiscalYear: year, Rows: make([]domain.ReconciliationRow, 0),
		},
		Reference: "§§ 284 und 285 HGB; Überleitung nach § 60 Abs. 2 EStDV",
	}
	if s.notes.Texts != nil {
		if texts, err := s.notes.Texts.NotesTexts(ctx, year); err == nil && texts != nil {
			notes.Texts = texts
		}
	}
	if s.notes.Provisions != nil {
		if mirror, err := s.notes.Provisions.Mirror(ctx, year); err == nil && mirror != nil {
			notes.ProvisionMirror = *mirror
		}
	}
	if s.notes.Reconciliation != nil {
		if recon, err := s.notes.Reconciliation.Reconcile(ctx, year); err == nil && recon != nil {
			notes.Reconciliation = *recon
		}
	}
	return notes
}

func hasAnyBalance(accounts []domain.Account) bool {
	for _, acc := range accounts {
		if acc.DebitSum != 0 || acc.CreditSum != 0 {
			return true
		}
	}
	return false
}

// Header sind die Pflichtangaben des § 264 Abs. 1a HGB, ohne eine Gliederung
// daneben.
//
// Die Eröffnungsbilanz braucht sie: sie hat einen eigenen Stichtag und keine
// Vorjahresspalte, füllt ihren Kopf aber aus denselben Unternehmensdaten.
func (s *StatementService) Header(ctx context.Context, year int) (domain.StatementHeader, error) {
	return s.header(ctx, year, &domain.Statement{})
}

// header sind die Pflichtangaben des § 264 Abs. 1a HGB.
func (s *StatementService) header(ctx context.Context, year int, stmt *domain.Statement) (domain.StatementHeader, error) {
	fy, err := s.period(ctx, year)
	if err != nil {
		return domain.StatementHeader{}, err
	}
	settings, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return domain.StatementHeader{}, err
	}

	header := domain.StatementHeader{
		CompanyName: settings.CompanyName, FirmName: firmNameWithLegalForm(settings),
		LegalForm: settings.LegalForm,
		Seat:      settings.Seat, RegisterCourt: settings.RegisterCourt,
		RegisterNumber: settings.RegisterNumber,
		FiscalYear:     year, StartDate: fy.StartDate, ClosingDate: fy.EndDate,
		IsShortYear: fy.IsShort,
		Reference:   "§ 264 Abs. 1a HGB",
		Missing:     []string{},
	}
	if stmt.HasPrior {
		header.PriorYear = stmt.PriorYear
	}

	// Eine fehlende Pflichtangabe bleibt sonst eine Lücke im Kopf, die niemand
	// einordnet. Benannt wird sie hier, damit die Ansicht sie benennen kann.
	for _, missing := range []struct {
		value, label string
	}{
		{settings.CompanyName, "Firma"},
		{settings.Seat, "Sitz"},
		{settings.RegisterCourt, "Registergericht"},
		{settings.RegisterNumber, "Registernummer"},
	} {
		if strings.TrimSpace(missing.value) == "" {
			header.Missing = append(header.Missing, missing.label)
		}
	}
	return header, nil
}

// deadlinesFor sind Aufstellung und Offenlegung aus dem Pflichtenkalender,
// mit dem Stand, den das Geschäftsjahr vermerkt. Fällt der Kalender aus,
// bleibt die Liste leer statt den Abschluss scheitern zu lassen.
func (s *StatementService) deadlinesFor(ctx context.Context, year int) []domain.Deadline {
	if s.deadlines == nil {
		return []domain.Deadline{}
	}
	out, err := s.deadlines.DeadlinesOf(ctx, year, domain.ObligationStatementPrep, domain.ObligationDisclosure)
	if err != nil || out == nil {
		return []domain.Deadline{}
	}
	return out
}

// maturities ist die Restlaufzeitengliederung nach § 268 Abs. 4 und 5 HGB.
//
// Sie entsteht aus den offenen Posten und ihrer Fälligkeit, gemessen vom
// Abschlussstichtag. Ein Posten ohne Fälligkeit bekommt keine Restlaufzeit
// zugeschrieben: eine erfundene Frist wäre eine Angabe, die niemand gemacht hat.
//
// Gefragt wird die Stichtagssicht (OpenItemsAt), nicht die heutige OP-Liste.
// Sonst schrumpfte die Angabe eines abgeschlossenen Jahres mit jeder Zahlung,
// die danach gebucht wird.
func (s *StatementService) maturities(ctx context.Context, closingDate string) (domain.MaturityTable, error) {
	table := domain.MaturityTable{
		ClosingDate: closingDate,
		Reference:   "§ 268 Abs. 4 und 5 HGB",
		Rows: []domain.MaturityRow{
			{Key: "receivables", Label: "Forderungen aus Lieferungen und Leistungen",
				Note: "Anzugeben ist der Betrag mit einer Restlaufzeit von mehr als einem Jahr (§ 268 Abs. 4 Satz 1 HGB)."},
			{Key: "liabilities", Label: "Verbindlichkeiten aus Lieferungen und Leistungen",
				Note: "Anzugeben sind die Beträge mit einer Restlaufzeit bis zu einem Jahr und von mehr als fünf Jahren (§ 268 Abs. 5 Satz 1 HGB)."},
		},
	}
	if s.openItems == nil || closingDate == "" {
		return table, nil
	}

	items, err := s.openItems.OpenItemsAt(ctx, closingDate)
	if err != nil {
		return table, err
	}
	oneYear := addMonthsISO(closingDate, 12)
	fiveYears := addMonthsISO(closingDate, 60)

	for _, item := range items {
		// Was nach dem Stichtag entstanden ist, stand am Stichtag nicht offen.
		if item.DocumentDate > closingDate {
			continue
		}
		idx := 0
		if item.ContactType != domain.ContactTypeCustomer {
			idx = 1
		}
		row := &table.Rows[idx]
		amount := item.OpenAmount
		if amount < 0 {
			amount = -amount
		}
		row.Total += amount
		row.Items++
		switch {
		case item.DueDate == "":
			row.Undated += amount
		case item.DueDate <= oneYear:
			row.UpToOneYear += amount
		case item.DueDate <= fiveYears:
			row.OverOneYear += amount
		default:
			row.OverOneYear += amount
			row.OverFiveYears += amount
		}
	}
	return table, nil
}

// addMonthsISO verschiebt ein ISO-Datum um ganze Monate und kappt auf das
// Monatsende — sonst würde aus dem 31. Januar plus einem Monat der 3. März.
func addMonthsISO(iso string, months int) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	year, month, day := t.Date()
	target := time.Date(year, month+time.Month(months), 1, 0, 0, 0, 0, time.UTC)
	last := target.AddDate(0, 1, -1).Day()
	if day > last {
		day = last
	}
	return time.Date(target.Year(), target.Month(), day, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
}

// firmNameWithLegalForm ist die Firma für den Kopf des Abschlusses.
//
// Der Name, ergänzt um die Rechtsform, wo sie im Namen fehlt — viele erfassen
// „Muster Ventures" und wählen die Rechtsform daneben, andere schreiben sie in
// den Namen. Der Zusatz „i. G." steht dabei ganz hinten und nicht zwischen
// Name und Rechtsform: „Muster Ventures i. G. GmbH" wäre keine Firma.
func firmNameWithLegalForm(settings *domain.CompanySettings) string {
	name := strings.TrimSpace(settings.CompanyName)
	if name == "" {
		return ""
	}
	if form := strings.TrimSpace(settings.LegalForm); form != "" && !strings.Contains(name, form) {
		name += " " + form
	}
	if settings.InGruendung {
		name += " i. G."
	}
	return name
}
