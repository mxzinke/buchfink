package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"sort"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// TaxRegisterService führt das Verzeichnis nach § 5 Abs. 1 Satz 2 EStG und die
// Überleitungsrechnung zur Steuerbilanz.
//
// Buchfink führt eine Einheitsbilanz: die Handelsbilanz ist zugleich Grundlage
// der Steuerbilanz. Ganz aufgeht das an zwei Stellen nicht, und beide sind
// hier zusammengefasst.
//
// Die erste ist § 7g Abs. 5 EStG. Die Sonderabschreibung ist ein steuerliches
// Wahlrecht; wer es ausübt, muss das Wirtschaftsgut nach § 5 Abs. 1 Satz 2 EStG
// in ein besonderes, laufend zu führendes Verzeichnis aufnehmen — mit Tag der
// Anschaffung, Anschaffungskosten, der Vorschrift und den vorgenommenen
// Abschreibungen. Genau das ist dieses Verzeichnis.
//
// Die zweite ist die Abzinsung der Rückstellungen: handelsrechtlich mit dem
// laufzeitkongruenten Durchschnittszins (§ 253 Abs. 2 HGB), steuerlich mit
// 5,5 % (§ 6 Abs. 1 Nr. 3a Buchst. e EStG). Dazu kommt die
// Drohverlustrückstellung, die § 5 Abs. 4a EStG steuerlich verbietet.
//
// Gebucht wird in beiden Fällen nichts. Die Überleitung ist eine Rechnung neben
// der Bilanz, keine zweite Buchführung.
type TaxRegisterService struct {
	assetRepo     domain.AssetRepository
	provisionRepo domain.ProvisionRepository
	// journalRepo wird nur gelesen, und nur für eine Frage: ob die Buchung
	// einer Rückstellungsbewegung per Generalumkehr zurückgenommen wurde. Eine
	// stornierte Bildung gehört in keine Überleitung — sie steht auch in keiner
	// Bilanz mehr.
	journalRepo  domain.JournalRepository
	settingsRepo domain.SettingsRepository
	closingSvc   *ClosingService
	fiscalYear   int
}

// NewTaxRegisterService wires das Verzeichnis und die Überleitung.
func NewTaxRegisterService(
	assetRepo domain.AssetRepository,
	provisionRepo domain.ProvisionRepository,
	journalRepo domain.JournalRepository,
	settingsRepo domain.SettingsRepository,
	closingSvc *ClosingService,
	fiscalYear int,
) *TaxRegisterService {
	return &TaxRegisterService{
		assetRepo: assetRepo, provisionRepo: provisionRepo, journalRepo: journalRepo,
		settingsRepo: settingsRepo, closingSvc: closingSvc, fiscalYear: fiscalYear,
	}
}

// SetFiscalYear schaltet das aktive Geschäftsjahr um.
func (s *TaxRegisterService) SetFiscalYear(year int) { s.fiscalYear = year }

// TaxElectionYear ist ein Jahr im Verzeichnis: handelsrechtliche und
// steuerliche Abschreibung nebeneinander.
type TaxElectionYear struct {
	FiscalYear int          `json:"fiscalYear"`
	Commercial domain.Cents `json:"commercial"`
	Tax        domain.Cents `json:"tax"`
	Difference domain.Cents `json:"difference"`
}

// TaxElectionRow ist ein Wirtschaftsgut im Verzeichnis.
type TaxElectionRow struct {
	AssetID         uint         `json:"assetId"`
	InventoryNumber string       `json:"inventoryNumber"`
	Name            string       `json:"name"`
	AcquisitionDate string       `json:"acquisitionDate"`
	Cost            domain.Cents `json:"cost"`
	// Provision nennt die Vorschrift, auf die sich das Wahlrecht stützt.
	Provision string `json:"provision"`
	Reason    string `json:"reason,omitempty"`

	Years           []TaxElectionYear `json:"years"`
	TotalCommercial domain.Cents      `json:"totalCommercial"`
	TotalTax        domain.Cents      `json:"totalTax"`
	TotalDifference domain.Cents      `json:"totalDifference"`

	// BookValue und TaxBookValue sind die Buchwerte am Ende des
	// Geschäftsjahres: handelsrechtlich, was in den Büchern steht, steuerlich
	// derselbe Wert abzüglich der kumulierten Mehr-AfA. Die Überleitung
	// braucht sie, weil sie Wertansätze gegenüberstellt und keine Differenzen.
	BookValue    domain.Cents `json:"bookValue"`
	TaxBookValue domain.Cents `json:"taxBookValue"`
}

// TaxElectionRegister ist das Verzeichnis nach § 5 Abs. 1 Satz 2 EStG.
type TaxElectionRegister struct {
	FiscalYear      int              `json:"fiscalYear"`
	Rows            []TaxElectionRow `json:"rows"`
	TotalDifference domain.Cents     `json:"totalDifference"`
	// TotalBookValue und TotalTaxBookValue sind die Buchwerte der im
	// Verzeichnis geführten Wirtschaftsgüter am Ende des Geschäftsjahres.
	TotalBookValue    domain.Cents `json:"totalBookValue"`
	TotalTaxBookValue domain.Cents `json:"totalTaxBookValue"`
	Note              string       `json:"note"`
}

// Register stellt das Verzeichnis bis zum Ende eines Geschäftsjahres zusammen.
func (s *TaxRegisterService) Register(ctx context.Context, year int) (*TaxElectionRegister, error) {
	if year == 0 {
		year = s.fiscalYear
	}
	assets, err := s.assetRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	startMonth := s.fiscalYearStartMonth(ctx)
	register := &TaxElectionRegister{
		FiscalYear: year, Rows: make([]TaxElectionRow, 0),
		Note: "Wer die Sonderabschreibung nach § 7g Abs. 5 EStG in Anspruch nimmt, muss das " +
			"Wirtschaftsgut nach § 5 Abs. 1 Satz 2 EStG in ein besonderes, laufend zu führendes " +
			"Verzeichnis aufnehmen. Gebucht wird die Sonderabschreibung nicht: handelsrechtlich ist " +
			"sie seit dem BilMoG unzulässig (§ 254 HGB a. F. entfallen).",
	}

	for i := range assets {
		asset := &assets[i]
		if asset.SpecialPermille <= 0 {
			continue
		}
		row := TaxElectionRow{
			AssetID: asset.ID, InventoryNumber: asset.InventoryNumber, Name: asset.Name,
			AcquisitionDate: asset.AcquisitionDate, Cost: asset.Cost,
			Provision: "§ 7g Abs. 5 EStG", Reason: asset.SpecialReason,
			Years: make([]TaxElectionYear, 0),
		}
		// Die steuerliche Spalte kommt aus dem Plan und nicht aus den
		// Bewegungen: steuerlich wird nichts gebucht, und nach dem
		// Begünstigungszeitraum weicht die steuerliche AfA auch ohne
		// Sonderabschreibung von der handelsrechtlichen ab (§ 7a Abs. 9 EStG).
		// Die handelsrechtliche Spalte bleibt dagegen das, was in den Büchern
		// steht — das Verzeichnis stellt beide nebeneinander.
		planByYear := map[int]accounting.AfAYear{}
		if rows, err := accounting.BuildAfASchedule(afaPlanFor(asset, startMonth)); err == nil {
			for _, r := range rows {
				planByYear[r.FiscalYear] = r
			}
		}

		byYear := map[int]*TaxElectionYear{}
		var years []int
		for _, m := range asset.Movements {
			if m.FiscalYear > year {
				continue
			}
			switch m.Kind {
			case domain.AssetMovementDepreciation, domain.AssetMovementSpecialDepreciation:
			default:
				continue
			}
			entry, ok := byYear[m.FiscalYear]
			if !ok {
				years = append(years, m.FiscalYear)
				byYear[m.FiscalYear] = &TaxElectionYear{FiscalYear: m.FiscalYear}
				entry = byYear[m.FiscalYear]
			}
			// Alte Bewegungen tragen die Sonderabschreibung noch als gebuchte
			// Abschreibung; sie zählt handelsrechtlich dort, wo sie steht.
			entry.Commercial += m.DepreciationAmount
		}
		sort.Ints(years)
		for _, y := range years {
			entry := byYear[y]
			if planned, ok := planByYear[y]; ok {
				entry.Tax = planned.TaxAmount + planned.SpecialAmount
			} else {
				entry.Tax = entry.Commercial
			}
			entry.Difference = entry.Tax - entry.Commercial
			row.Years = append(row.Years, *entry)
			row.TotalCommercial += entry.Commercial
			row.TotalTax += entry.Tax
			row.TotalDifference += entry.Difference
		}
		// Der Buchwert am Ende des Jahres ist die Summe aller Bewegungen bis
		// dahin — Zugänge minus Abschreibungen, wie ihn auch die Anlagenkartei
		// rechnet. Der steuerliche Buchwert ist derselbe Wert abzüglich der
		// kumulierten Mehr-AfA: steuerlich wird nichts gebucht, die Differenz
		// ist der ganze Unterschied.
		for _, m := range asset.Movements {
			if m.FiscalYear > year {
				continue
			}
			row.BookValue += m.BookValueEffect()
		}
		row.TaxBookValue = row.BookValue - row.TotalDifference
		if row.TaxBookValue < 0 {
			row.TaxBookValue = 0
		}
		register.Rows = append(register.Rows, row)
		register.TotalDifference += row.TotalDifference
		register.TotalBookValue += row.BookValue
		register.TotalTaxBookValue += row.TaxBookValue
	}
	sort.Slice(register.Rows, func(i, j int) bool {
		return register.Rows[i].InventoryNumber < register.Rows[j].InventoryNumber
	})
	return register, nil
}

func (s *TaxRegisterService) fiscalYearStartMonth(ctx context.Context) int {
	if s.settingsRepo == nil {
		return 1
	}
	settings, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil || settings == nil || settings.FiscalYearStartMonth <= 0 {
		return 1
	}
	return settings.FiscalYearStartMonth
}

// RegisterCSV gibt das Verzeichnis als CSV aus — eine Zeile je Wirtschaftsgut
// und Jahr, weil das Verzeichnis „laufend zu führen" ist und der Prüfer die
// Jahre nebeneinander sehen will.
func (s *TaxRegisterService) RegisterCSV(ctx context.Context, year int) (string, error) {
	register, err := s.Register(ctx, year)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	writer := csv.NewWriter(&out)
	writer.Comma = ';'
	_ = writer.Write([]string{
		"Inventarnummer", "Wirtschaftsgut", "Tag der Anschaffung", "Anschaffungskosten",
		"Vorschrift", "Geschäftsjahr", "AfA handelsrechtlich", "AfA steuerlich", "Differenz",
	})
	for _, row := range register.Rows {
		for _, y := range row.Years {
			_ = writer.Write([]string{
				row.InventoryNumber, row.Name, row.AcquisitionDate, row.Cost.Decimal(),
				row.Provision, fmt.Sprintf("%d", y.FiscalYear),
				y.Commercial.Decimal(), y.Tax.Decimal(), y.Difference.Decimal(),
			})
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("das Verzeichnis konnte nicht als CSV geschrieben werden: %w", err)
	}
	return out.String(), nil
}

// ReconciliationRow und Reconciliation stehen in internal/domain: die
// Überleitung ist Bestandteil des Anhangs und geht als Block in die E-Bilanz.
// Gerechnet wird sie hier.
type (
	ReconciliationRow = domain.ReconciliationRow
	Reconciliation    = domain.Reconciliation
)

// Reconcile stellt die Überleitungsrechnung zusammen.
func (s *TaxRegisterService) Reconcile(ctx context.Context, year int) (*Reconciliation, error) {
	if year == 0 {
		year = s.fiscalYear
	}
	fy, err := s.closingSvc.PeriodOf(ctx, year)
	if err != nil {
		return nil, err
	}
	out := &Reconciliation{
		FiscalYear: year, Cutoff: fy.EndDate, Rows: make([]ReconciliationRow, 0),
		Note: "Buchfink führt eine Einheitsbilanz. Die Überleitung nennt die Stellen, an denen " +
			"Handels- und Steuerbilanz zwingend auseinanderfallen; gebucht wird nichts.",
	}

	// 1. Anlagevermögen: die Sonderabschreibung nach § 7g Abs. 5 EStG mindert
	// nur den steuerlichen Wertansatz.
	register, err := s.Register(ctx, year)
	if err != nil {
		return nil, err
	}
	if register.TotalDifference != 0 {
		// In den Spalten stehen Wertansätze und nicht Differenzen: die
		// kumulierten Buchwerte der Wirtschaftsgüter, die das Wahlrecht in
		// Anspruch nehmen — handelsrechtlich, was in den Büchern steht,
		// steuerlich derselbe Bestand nach der Mehr-AfA. Stünde hier wie bei
		// den Rückstellungen einmal eine Differenz und einmal ein Wertansatz,
		// trügen zwei Zeilen unter denselben Spaltenköpfen zwei verschiedene
		// Bedeutungen.
		out.Rows = append(out.Rows, ReconciliationRow{
			Position: "Anlagevermögen", Basis: "§ 7g Abs. 5 EStG, § 5 Abs. 1 Satz 2 EStG",
			Commercial: register.TotalBookValue, Tax: register.TotalTaxBookValue,
			Difference: register.TotalTaxBookValue - register.TotalBookValue,
			Explanation: fmt.Sprintf(
				"Kumulierte Sonderabschreibungen von %s € mindern den steuerlichen Wertansatz des "+
					"Anlagevermögens; handelsrechtlich sind sie seit dem BilMoG nicht ansetzbar. "+
					"Gegenübergestellt sind die Buchwerte der betroffenen Wirtschaftsgüter zum %s.",
				register.TotalDifference, fy.EndDate),
		})
		out.EquityEffect += register.TotalTaxBookValue - register.TotalBookValue
	}

	// 2. Rückstellungen: Abzinsung und das Ansatzverbot für Drohverluste.
	stored, err := s.provisionRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	provisions, err := liveProvisions(ctx, s.journalRepo, stored)
	if err != nil {
		return nil, err
	}
	var commercialProvisions, taxProvisions, pendingLoss domain.Cents
	for i := range provisions {
		p := &provisions[i]
		balance := p.BalanceAt(year)
		if balance <= 0 {
			continue
		}
		if p.Kind == domain.ProvisionPendingLoss {
			// § 5 Abs. 4a Satz 1 EStG: Rückstellungen für drohende Verluste aus
			// schwebenden Geschäften dürfen steuerlich nicht gebildet werden.
			pendingLoss += balance
			continue
		}
		commercialProvisions += balance
		taxProvisions += s.taxValueOf(p, fy.EndDate, balance)
	}
	if diff := taxProvisions - commercialProvisions; diff != 0 {
		out.Rows = append(out.Rows, ReconciliationRow{
			Position: "Rückstellungen (Abzinsung)", Basis: "§ 253 Abs. 2 HGB, § 6 Abs. 1 Nr. 3a Buchst. e EStG",
			Commercial: commercialProvisions, Tax: taxProvisions, Difference: diff,
			Explanation: "Handelsrechtlich wird mit dem laufzeitkongruenten Durchschnittszins " +
				"abgezinst, steuerlich mit 5,5 %. Der höhere steuerliche Zins ergibt den " +
				"niedrigeren Wertansatz.",
		})
		// Ein niedrigerer Rückstellungswert bedeutet ein höheres Eigenkapital:
		// die Passivposition wirkt mit umgekehrtem Vorzeichen.
		out.EquityEffect += -diff
	}
	if pendingLoss > 0 {
		out.Rows = append(out.Rows, ReconciliationRow{
			Position: "Drohverlustrückstellungen", Basis: "§ 249 Abs. 1 HGB, § 5 Abs. 4a EStG",
			Commercial: pendingLoss, Tax: 0, Difference: -pendingLoss,
			Explanation: "Handelsrechtlich Pflicht, steuerlich verboten: § 5 Abs. 4a Satz 1 EStG " +
				"lässt Rückstellungen für drohende Verluste aus schwebenden Geschäften nicht zu.",
		})
		out.EquityEffect += pendingLoss
	}
	return out, nil
}

// taxValueOf ist der steuerliche Wert einer Rückstellung: derselbe Betrag, aber
// mit 5,5 % abgezinst, wo die Restlaufzeit ein Jahr übersteigt.
func (s *TaxRegisterService) taxValueOf(p *domain.Provision, cutoff string, balance domain.Cents) domain.Cents {
	if !accounting.NeedsDiscounting(cutoff, p.ExpectedDate) {
		return balance
	}
	if p.Kind == domain.ProvisionPension {
		// Für Pensionsrückstellungen gilt nicht § 6 Abs. 1 Nr. 3a EStG, sondern
		// § 6a EStG: Teilwert nach versicherungsmathematischen Grundsätzen,
		// abgezinst mit 6 %. Diese Rechnung führt Buchfink nicht — sie käme aus
		// einem Gutachten. Mit 5,5 % gerechnet stünde in der Überleitung eine
		// Differenz, die es so nicht gibt.
		return balance
	}
	years, err := accounting.RemainingYears(cutoff, p.ExpectedDate)
	if err != nil || years <= 0 {
		return balance
	}
	// Ausgangspunkt ist der Erfüllungsbetrag und nicht der handelsrechtliche
	// Barwert: beide Wertansätze zinsen denselben Betrag ab, nur mit
	// verschiedenen Sätzen.
	//
	// Ist die Rückstellung teilweise verbraucht, gilt das nur noch für den
	// Rest. Den vollen Erfüllungsbetrag abzuzinsen ergäbe einen steuerlichen
	// Wert über dem handelsrechtlichen Bestand — die Überleitung zeigte dann
	// eine Differenz, die es nicht gibt.
	settlement := p.SettlementAmount
	if formed := p.DiscountedAmount; formed > 0 && balance < formed {
		settlement = domain.MulRound(settlement, int64(balance), int64(formed))
	}
	value := accounting.PresentValue(settlement, years, TaxDiscountPermille*1000)
	if value > balance {
		// Mehr als der handelsrechtliche Bestand kann steuerlich nicht stehen:
		// der steuerliche Zins ist der höhere, und der höhere Zins ergibt den
		// niedrigeren Wert.
		return balance
	}
	return value
}
