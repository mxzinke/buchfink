package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// StatementService stellt Bilanz und Gewinn- und Verlustrechnung auf.
//
// Er beschafft, was die reine Gliederung in internal/accounting nicht beschaffen
// kann: die Salden zweier Geschäftsjahre, den Zeitraum des Jahres, die
// Unternehmensdaten für den Kopf und die offenen Posten für die
// Restlaufzeitengliederung. Gerechnet wird dort, nicht hier — und im Frontend
// gar nicht.
type StatementService struct {
	accountingSvc *AccountingService
	closingSvc    *ClosingService
	settingsRepo  domain.SettingsRepository
	auditRepo     domain.AuditRepository
	openItems     OpenItemSource
	notes         NotesSources
	renderer      DocumentRenderer
	fiscalYear    int
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
	closingSvc *ClosingService,
	settingsRepo domain.SettingsRepository,
	auditRepo domain.AuditRepository,
	fiscalYear int,
) *StatementService {
	return &StatementService{
		accountingSvc: accountingSvc,
		closingSvc:    closingSvc,
		settingsRepo:  settingsRepo,
		auditRepo:     auditRepo,
		fiscalYear:    fiscalYear,
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

// SetFiscalYear updates the active fiscal year.
func (s *StatementService) SetFiscalYear(year int) { s.fiscalYear = year }

// Build stellt den Jahresabschluss eines Geschäftsjahres auf.
//
// depth darf leer sein; dann entscheidet die Größenklasse über die
// Gliederungstiefe, denn das ist die Tiefe, in der der Abschluss offenzulegen
// ist (§ 266 Abs. 1 Sätze 3 und 4 HGB).
func (s *StatementService) Build(ctx context.Context, year int, depth domain.StatementDepth) (*domain.FinancialStatement, error) {
	if year <= 0 {
		year = s.fiscalYear
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

	return &domain.FinancialStatement{
		Header:     header,
		Statement:  *shown,
		SizeClass:  sizeClass,
		Maturities: maturities,
		Notes:      s.notesFor(ctx, year),
		Deadlines:  s.deadlinesFor(ctx, year, header.ClosingDate, sizeClass),
	}, nil
}

// Notes liefert den Anhang eines Geschäftsjahres — für die Ausgabewege, die
// nicht den ganzen Abschluss aufbauen, etwa die E-Bilanz.
func (s *StatementService) Notes(ctx context.Context, year int) domain.StatementNotes {
	if year <= 0 {
		year = s.fiscalYear
	}
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

// SizeClassFor beurteilt die Größenklasse eines Geschäftsjahres.
func (s *StatementService) SizeClassFor(ctx context.Context, year int) (*domain.SizeClass, error) {
	if year <= 0 {
		year = s.fiscalYear
	}
	stmt, err := s.statement(ctx, year, domain.DepthFull)
	if err != nil {
		return nil, err
	}
	sizeClass, err := s.sizeClassFrom(ctx, year, stmt)
	if err != nil {
		return nil, err
	}
	return &sizeClass, nil
}

// Deadlines liefert die Termine des Jahresabschlusses: Aufstellung und
// Offenlegung.
func (s *StatementService) Deadlines(ctx context.Context, year int) ([]domain.Deadline, error) {
	if year <= 0 {
		year = s.fiscalYear
	}
	sizeClass, err := s.SizeClassFor(ctx, year)
	if err != nil {
		return nil, err
	}
	return s.deadlinesFor(ctx, year, sizeClass.ClosingDate, *sizeClass), nil
}

// Statement liefert allein die Gliederung — die E-Bilanz braucht sie ohne Kopf,
// Größenklasse und Fristen.
func (s *StatementService) Statement(ctx context.Context, year int, depth domain.StatementDepth) (*domain.Statement, error) {
	if year <= 0 {
		year = s.fiscalYear
	}
	if depth == "" {
		depth = domain.DepthFull
	}
	return s.statement(ctx, year, depth)
}

// Accounts liefert den Kontenplan mit den Salden eines Geschäftsjahres.
func (s *StatementService) Accounts(ctx context.Context, year int) ([]domain.Account, error) {
	if year <= 0 {
		year = s.fiscalYear
	}
	return s.accountingSvc.AccountsForYear(ctx, year)
}

// statement baut die Gliederung eines Jahres mit der Vorjahresspalte.
func (s *StatementService) statement(ctx context.Context, year int, depth domain.StatementDepth) (*domain.Statement, error) {
	current, err := s.accountingSvc.AccountsForYear(ctx, year)
	if err != nil {
		return nil, err
	}
	prior, err := s.accountingSvc.AccountsForYear(ctx, year-1)
	if err != nil {
		return nil, err
	}
	// § 265 Abs. 2 HGB verlangt die Vorjahreszahl zu jedem Posten — aber nur,
	// wo es ein Vorjahr gibt. Ein Jahr ohne jede Buchung ergäbe eine Spalte aus
	// lauter Nullen, und die läse sich wie eine Aussage über das Vorjahr.
	if !hasAnyBalance(prior) {
		prior = nil
	}

	stmt, err := accounting.BuildStatement(current, prior, depth)
	if err != nil {
		return nil, err
	}
	stmt.FiscalYear = year
	stmt.PriorYear = year - 1
	return stmt, nil
}

func hasAnyBalance(accounts []domain.Account) bool {
	for _, acc := range accounts {
		if acc.DebitSum != 0 || acc.CreditSum != 0 {
			return true
		}
	}
	return false
}

// maxSizeHistoryYears begrenzt, wie weit die Zweijahresregel zurückgeht.
//
// Zurückgegangen wird nur, solange kein Stichtagspaar übereinstimmt — bei einer
// Gesellschaft, die zehn Jahre lang bei jedem Abschluss die Klasse wechselt.
// Das kommt nicht vor; die Grenze verhindert allein, dass ein kaputter
// Datenbestand die Aufstellung des Abschlusses in eine lange Schleife schickt.
const maxSizeHistoryYears = 10

// sizeClassFrom beurteilt so viele Stichtage, wie die Zweijahresregel braucht.
//
// Zwei genügen nicht: § 267 Abs. 4 Satz 1 HGB vergleicht die Beurteilung des
// Stichtags mit der des Vorjahres, und was gilt, wenn beide auseinandergehen,
// ist die *wirksame* Klasse des Vorjahres. Die kann aus einem noch früheren
// Paar stammen. Deshalb werden Vorjahre so lange beurteilt, bis zwei
// aufeinanderfolgende Stichtage dieselbe Klasse ergeben oder das erste erfasste
// Geschäftsjahr erreicht ist.
func (s *StatementService) sizeClassFrom(ctx context.Context, year int, stmt *domain.Statement) (domain.SizeClass, error) {
	fy, err := s.period(ctx, year)
	if err != nil {
		return domain.SizeClass{}, err
	}

	current, err := accounting.AssessSize(year, fy.EndDate, fy.StartDate, domain.SizeCriteria{
		BalanceSheetTotal: stmt.BalanceSheetTotal,
		Revenue:           stmt.Revenue,
		Employees:         fy.AverageEmployees,
	})
	if err != nil {
		return domain.SizeClass{}, err
	}

	earliest := 0
	if s.closingSvc != nil {
		if e, err := s.closingSvc.EarliestFiscalYear(ctx); err == nil {
			earliest = e
		}
	}
	// § 267 Abs. 4 Satz 2 HGB: beim ersten Abschluss nach der Gründung
	// entscheidet dieser Stichtag allein.
	isFirstYear := earliest == year || earliest == 0

	history := []domain.SizeAssessment{current}
	if !isFirstYear {
		for y := year - 1; y >= year-maxSizeHistoryYears; y-- {
			if earliest > 0 && y < earliest {
				break
			}
			assessment, ok, err := s.assessYear(ctx, y)
			if err != nil {
				return domain.SizeClass{}, err
			}
			if !ok {
				break
			}
			history = append([]domain.SizeAssessment{assessment}, history...)
			if assessment.Class == history[1].Class {
				// Ab hier steht die wirksame Klasse fest; ältere Stichtage
				// ändern an ihr nichts mehr.
				break
			}
		}
	}

	return accounting.ClassifySize(history, isFirstYear), nil
}

// assessYear beurteilt einen zurückliegenden Abschlussstichtag aus den Zahlen
// dieses Jahres allein.
//
// Die zweite Rückgabe ist falsch, wenn sich das Jahr nicht beurteilen lässt:
// ohne Buchung gibt es keine Merkmale, und eine Bilanz, die nicht aufgeht, gibt
// keine Bilanzsumme her. Beides ist kein Fehler der Beurteilung des laufenden
// Jahres — es beendet nur die Kette der Vorjahre.
func (s *StatementService) assessYear(ctx context.Context, year int) (domain.SizeAssessment, bool, error) {
	accounts, err := s.accountingSvc.AccountsForYear(ctx, year)
	if err != nil {
		return domain.SizeAssessment{}, false, err
	}
	if !hasAnyBalance(accounts) {
		return domain.SizeAssessment{}, false, nil
	}
	// Ohne Vorjahresspalte: für die Merkmale des § 267 HGB zählt allein dieser
	// Stichtag, und die zweite Spalte kostete eine weitere Abfrage je Jahr.
	stmt, err := accounting.BuildStatement(accounts, nil, domain.DepthFull)
	if err != nil {
		return domain.SizeAssessment{}, false, nil
	}
	fy, err := s.period(ctx, year)
	if err != nil {
		return domain.SizeAssessment{}, false, err
	}
	assessment, err := accounting.AssessSize(year, fy.EndDate, fy.StartDate, domain.SizeCriteria{
		BalanceSheetTotal: stmt.BalanceSheetTotal,
		Revenue:           stmt.Revenue,
		Employees:         fy.AverageEmployees,
	})
	if err != nil {
		return domain.SizeAssessment{}, false, err
	}
	return assessment, true, nil
}

// period liefert den Zeitraum eines Geschäftsjahres, ohne ihn anzulegen.
func (s *StatementService) period(ctx context.Context, year int) (*domain.FiscalYear, error) {
	if s.closingSvc == nil {
		return domain.NewFiscalYear(year, fmt.Sprintf("%d-01-01", year), fmt.Sprintf("%d-12-31", year)), nil
	}
	return s.closingSvc.PeriodOf(ctx, year)
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
		CompanyName: settings.CompanyName, LegalForm: settings.LegalForm,
		Seat: settings.Seat, RegisterCourt: settings.RegisterCourt,
		RegisterNumber: settings.RegisterNumber,
		FiscalYear:     year, StartDate: fy.StartDate, ClosingDate: fy.EndDate,
		IsShortYear: fy.IsShort,
		Reference:   "§ 264 Abs. 1a HGB",
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

// deadlinesFor setzt die Termine und trägt ein, was das Geschäftsjahr bereits
// als erledigt vermerkt.
//
// Der Abschlussstand aus Welle 1 sagt das: aufgestellt ist aufgestellt, und
// eine Frist, die erledigt ist, gehört nicht mehr als Mahnung in eine Liste.
func (s *StatementService) deadlinesFor(
	ctx context.Context, year int, closingDate string, sizeClass domain.SizeClass,
) []domain.Deadline {
	deadlines := accounting.StatementDeadlines(year, closingDate, sizeClass.Obligations)
	fy, err := s.period(ctx, year)
	if err != nil {
		return deadlines
	}
	for i := range deadlines {
		switch deadlines[i].Key {
		case "abschluss.aufstellung":
			if fy.PreparedOn != "" {
				deadlines[i].IsDone, deadlines[i].DoneOn = true, fy.PreparedOn
			}
		case "abschluss.offenlegung":
			if fy.DisclosedOn != "" {
				deadlines[i].IsDone, deadlines[i].DoneOn = true, fy.DisclosedOn
			}
		}
	}
	return deadlines
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
