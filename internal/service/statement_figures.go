package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// StatementFigures berechnet Abschlusszahlen und Größenklasse aus den Büchern.
// Abschluss und Pflichtenkalender nutzen dieselbe Auswertung ohne Fristen.
type StatementFigures struct {
	accountingSvc *AccountingService
	periods       StatementPeriods
	settingsRepo  domain.SettingsRepository
}

// Figures liefert die gemeinsame Auswertung ohne Anhang und Pflichtenkalender.
func (s *StatementService) Figures() *StatementFigures { return s.StatementFigures }

// SizeClassFor beurteilt die Größenklasse eines Geschäftsjahres.
func (s *StatementFigures) SizeClassFor(ctx context.Context, year int) (*domain.SizeClass, error) {
	if err := requireYear(year); err != nil {
		return nil, err
	}
	stmt, err := s.statement(ctx, year, domain.DepthFull, "")
	if err != nil {
		return nil, err
	}
	sizeClass, err := s.sizeClassFrom(ctx, year, stmt)
	if err != nil {
		return nil, err
	}
	return &sizeClass, nil
}

// Statement liefert allein die Gliederung — die E-Bilanz braucht sie ohne Kopf,
// Größenklasse und Fristen.
func (s *StatementFigures) Statement(ctx context.Context, year int, depth domain.StatementDepth) (*domain.Statement, error) {
	if err := requireYear(year); err != nil {
		return nil, err
	}
	if depth == "" {
		depth = domain.DepthFull
	}
	return s.statement(ctx, year, depth, "")
}

// StatementUntil ist die Gliederung eines Geschäftsjahres bis zu einem
// Stichtag in ihm, etwa zum Monatsabschluss. Die Vorjahresspalte bleibt der
// Stand am Ende des Vorjahres.
func (s *StatementFigures) StatementUntil(
	ctx context.Context, year int, cutoff string, depth domain.StatementDepth,
) (*domain.Statement, error) {
	if err := requireYear(year); err != nil {
		return nil, err
	}
	if len(cutoff) != 10 {
		return nil, fmt.Errorf("der Stichtag fehlt oder ist unvollständig (erwartet JJJJ-MM-TT)")
	}
	if depth == "" {
		depth = domain.DepthFull
	}
	return s.statement(ctx, year, depth, cutoff)
}

// Accounts liefert den Kontenplan mit den Salden eines Geschäftsjahres.
func (s *StatementFigures) Accounts(ctx context.Context, year int) ([]domain.Account, error) {
	if err := requireYear(year); err != nil {
		return nil, err
	}
	return s.accountingSvc.AccountsForYear(ctx, year)
}

// statement baut die Gliederung eines Jahres mit der Vorjahresspalte; mit
// Stichtag bis zu ihm.
func (s *StatementFigures) statement(
	ctx context.Context, year int, depth domain.StatementDepth, cutoff string,
) (*domain.Statement, error) {
	current, err := s.accountingSvc.AccountsForYearAt(ctx, year, cutoff)
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

// StatementAt baut die Gliederung auf einen Stichtag statt auf ein
// Geschäftsjahr.
//
// Der Fall ist die Eröffnungsbilanz: sie steht auf den Tag der Beurkundung
// (§ 242 Abs. 1 HGB) und nicht auf den Schluss eines Geschäftsjahres. Gelesen
// wird über alle Jahre bis zu diesem Tag — die Jahresgrenze fällt weg, weil eine
// Beurkundung im November und ein aktives Geschäftsjahr im Januar auseinander
// liegen können.
//
// Ohne Vorjahresspalte: § 265 Abs. 2 HGB verlangt sie zu jedem Posten, aber vor
// dem ersten Tag des Unternehmens gibt es kein Vorjahr, über das sich etwas
// sagen ließe.
func (s *StatementFigures) StatementAt(
	ctx context.Context, cutoff string, depth domain.StatementDepth,
) (*domain.Statement, error) {
	if len(cutoff) != 10 {
		return nil, fmt.Errorf("der Stichtag fehlt oder ist unvollständig (erwartet JJJJ-MM-TT)")
	}
	if depth == "" {
		depth = domain.DepthFull
	}
	accounts, err := s.accountingSvc.AccountsForYearAt(ctx, 0, cutoff)
	if err != nil {
		return nil, err
	}
	stmt, err := accounting.BuildStatement(accounts, nil, depth)
	if err != nil {
		return nil, err
	}
	// Das Geschäftsjahr, in das der Stichtag fällt: die Gliederung trägt es als
	// Kennung, gerechnet wird über den Stichtag.
	startMonth := 1
	if cfg, err := s.settingsRepo.GetCompanySettings(ctx); err == nil && cfg != nil &&
		cfg.FiscalYearStartMonth >= 1 && cfg.FiscalYearStartMonth <= 12 {
		startMonth = cfg.FiscalYearStartMonth
	}
	stmt.FiscalYear = domain.GetFiscalYearForDate(cutoff, startMonth)
	stmt.PriorYear = stmt.FiscalYear - 1
	return stmt, nil
}

// AccountsAt liefert den Kontenplan mit den Salden zu einem Stichtag, über alle
// Geschäftsjahre hinweg.
func (s *StatementFigures) AccountsAt(ctx context.Context, cutoff string) ([]domain.Account, error) {
	return s.accountingSvc.AccountsForYearAt(ctx, 0, cutoff)
}

// sizeClassFrom beurteilt so viele Stichtage, wie die Zweijahresregel braucht.
//
// Zwei genügen nicht: § 267 Abs. 4 Satz 1 HGB vergleicht die Beurteilung des
// Stichtags mit der des Vorjahres, und was gilt, wenn beide auseinandergehen,
// ist die *wirksame* Klasse des Vorjahres. Die kann aus einem noch früheren
// Paar stammen. Deshalb werden Vorjahre so lange beurteilt, bis zwei
// aufeinanderfolgende Stichtage dieselbe Klasse ergeben oder das erste erfasste
// Geschäftsjahr erreicht ist.
func (s *StatementFigures) sizeClassFrom(ctx context.Context, year int, stmt *domain.Statement) (domain.SizeClass, error) {
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
	if s.periods != nil {
		if e, err := s.periods.EarliestFiscalYear(ctx); err == nil {
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

// RevenueOf liefert die Umsatzerlöse eines Geschäftsjahres — die Zeile Nr. 1 der
// Gewinn- und Verlustrechnung (§ 275 Abs. 2 HGB).
//
// Sie entsteht aus derselben Gliederung wie die GuV auf dem Schirm und nicht
// aus einer zweiten Kontenauswahl: die Erlösschmälerungen sind nach § 277
// Abs. 1 HGB abzusetzen, und wer sie in einer eigenen Summe vergisst, meldet
// einen zu hohen Umsatz. Ein Jahr ohne Buchungen oder mit einer Bilanz, die
// nicht aufgeht, liefert null. Das ist die Auskunft, dass sich nichts ableiten
// lässt, kein Fehler.
func (s *StatementFigures) RevenueOf(ctx context.Context, year int) (domain.Cents, error) {
	if year <= 0 {
		return 0, nil
	}
	accounts, err := s.accountingSvc.AccountsForYear(ctx, year)
	if err != nil {
		return 0, err
	}
	if !hasAnyBalance(accounts) {
		return 0, nil
	}
	stmt, err := accounting.BuildStatement(accounts, nil, domain.DepthFull)
	if err != nil {
		return 0, nil
	}
	return stmt.Revenue, nil
}

// assessYear beurteilt einen zurückliegenden Abschlussstichtag aus den Zahlen
// dieses Jahres allein.
//
// Die zweite Rückgabe ist falsch, wenn sich das Jahr nicht beurteilen lässt:
// ohne Buchung gibt es keine Merkmale, und eine Bilanz, die nicht aufgeht, gibt
// keine Bilanzsumme her. Beides ist kein Fehler der Beurteilung des laufenden
// Jahres — es beendet nur die Kette der Vorjahre.
func (s *StatementFigures) assessYear(ctx context.Context, year int) (domain.SizeAssessment, bool, error) {
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
func (s *StatementFigures) period(ctx context.Context, year int) (*domain.FiscalYear, error) {
	if s.periods == nil {
		return domain.NewFiscalYear(year, fmt.Sprintf("%d-01-01", year), fmt.Sprintf("%d-12-31", year)), nil
	}
	return s.periods.PeriodOf(ctx, year)
}

// maxSizeHistoryYears begrenzt, wie weit die Zweijahresregel zurückgeht.
//
// Zurückgegangen wird nur, solange kein Stichtagspaar übereinstimmt — bei einer
// Gesellschaft, die zehn Jahre lang bei jedem Abschluss die Klasse wechselt.
// Das kommt nicht vor; die Grenze verhindert allein, dass ein kaputter
// Datenbestand die Aufstellung des Abschlusses in eine lange Schleife schickt.
const maxSizeHistoryYears = 10
