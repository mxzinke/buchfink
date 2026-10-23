package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/calendar"
	"github.com/buchfink/buchfink/internal/declaration"
	"github.com/buchfink/buchfink/internal/domain"
)

// common sammelt, was jede Erklärung braucht: Stammdaten, Zeitraum, Stand
// der Bücher, Eingaben und gespeicherte Stände.
func (s *DeclarationService) common(ctx context.Context, kind declaration.Kind, year int) (declaration.Common, error) {
	cfg, err := s.src.Settings.GetCompanySettings(ctx)
	if err != nil {
		return declaration.Common{}, fmt.Errorf("die Unternehmensdaten ließen sich nicht lesen: %w", err)
	}
	fy, err := s.fiscalYearEndingIn(ctx, year)
	if err != nil {
		return declaration.Common{}, err
	}
	period := declaration.Period{Year: year, From: fy.StartDate, To: fy.EndDate}
	if kind == declaration.KindVatYear {
		// Besteuerungszeitraum der Umsatzsteuer ist das Kalenderjahr
		// (§ 16 Abs. 1 Satz 2 UStG).
		period.From, period.To = fmt.Sprintf("%d-01-01", year), fmt.Sprintf("%d-12-31", year)
	}
	basis := declaration.Basis{
		ClosingState: string(fy.Status), ProgramVersion: ProgramVersion(), ComputedAt: s.now().UTC(),
		JournalHash: domain.GenesisHash,
	}
	if last, err := s.src.Journal.GetLastEntry(ctx, fy.Year); err != nil {
		return declaration.Common{}, err
	} else if last != nil {
		basis.JournalHash = last.EntryHash
	}
	count, err := s.src.Journal.Count(ctx, fy.Year)
	if err != nil {
		return declaration.Common{}, err
	}
	basis.JournalEntries = int(count)

	inputs, err := s.inputs(ctx, kind, year)
	if err != nil {
		return declaration.Common{}, err
	}
	stored, err := s.repo.FindRecords(ctx, string(kind), year)
	if err != nil {
		return declaration.Common{}, err
	}
	records := make([]declaration.RecordState, 0, len(stored))
	for _, r := range stored {
		state := declaration.RecordState{Status: declaration.Status(r.Status), FiguresHash: r.FiguresHash}
		if r.Status == string(declaration.StatusSubmitted) {
			state.Submission = &declaration.Submission{At: r.SubmittedAt, TransferTicket: r.TransferTicket,
				FiguresHash: r.FiguresHash, Actor: r.Actor, OverrideReason: r.OverrideReason}
		}
		records = append(records, state)
	}
	return declaration.Common{Year: year, Period: period, Settings: *cfg, Inputs: inputs, Basis: basis, Records: records}, nil
}

// fiscalYearEndingIn ist das Wirtschaftsjahr, das im Kalenderjahr endet: sein
// Gewinn gilt in diesem Jahr als bezogen (§ 7 Abs. 4 Satz 2 KStG, § 10 Abs. 2
// GewStG).
func (s *DeclarationService) fiscalYearEndingIn(ctx context.Context, year int) (*domain.FiscalYear, error) {
	years, err := s.src.FiscalYears.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	prefix := fmt.Sprintf("%d-", year)
	for i := range years {
		if strings.HasPrefix(years[i].EndDate, prefix) {
			return &years[i], nil
		}
	}
	return nil, fmt.Errorf("kein Geschäftsjahr endet im Jahr %d; ohne Geschäftsjahr gibt es keine Erklärung", year)
}

// vatYearData sammelt die Größen der Umsatzsteuererklärung.
func (s *DeclarationService) vatYearData(ctx context.Context, year int, cfg domain.CompanySettings) (declaration.VatYearData, error) {
	annual, err := s.src.VatReturns.Draft(ctx, strconv.Itoa(year))
	if err != nil {
		return declaration.VatYearData{}, err
	}
	saved, err := s.src.VatReturns.List(ctx, year)
	if err != nil {
		return declaration.VatYearData{}, err
	}
	latest := map[string]domain.VatReturn{}
	for _, r := range saved {
		if r.Status != domain.VatReturnSubmitted || r.PeriodType == domain.VatPeriodYear {
			continue
		}
		if cur, ok := latest[r.PeriodKey]; !ok || r.ID > cur.ID {
			latest[r.PeriodKey] = r
		}
	}
	submitted := make([]domain.VatReturn, 0, len(latest))
	for _, r := range latest {
		submitted = append(submitted, r)
	}
	sort.Slice(submitted, func(i, j int) bool { return submitted[i].PeriodKey < submitted[j].PeriodKey })

	periodType := s.src.VatReturns.PeriodType(ctx)
	expected := []string{}
	if periodType == domain.VatPeriodMonth || periodType == domain.VatPeriodQuarter {
		today := calendar.Today(s.now)
		for _, p := range accounting.VatPeriodsOfYear(year, periodType) {
			if p.To >= today || (cfg.FoundedOn != "" && p.To < cfg.FoundedOn) {
				continue
			}
			expected = append(expected, p.Key)
		}
	}

	data := declaration.VatYearData{
		PeriodType: periodType, Annual: annual, Submitted: submitted, ExpectedPeriods: expected,
		Ledger: declaration.Ledger{},
	}
	if s.src.ZM != nil {
		if err := s.zmData(ctx, year, &data); err != nil {
			return declaration.VatYearData{}, err
		}
	}
	if s.src.InputTax != nil {
		corrections, err := s.src.InputTax.FindAll(ctx)
		if err != nil {
			return declaration.VatYearData{}, err
		}
		data.Corrections = corrections
	}
	ledger, err := s.ledger(ctx, year)
	if err != nil {
		return declaration.VatYearData{}, err
	}
	data.Ledger = ledger
	return data, nil
}

func (s *DeclarationService) zmData(ctx context.Context, year int, data *declaration.VatYearData) error {
	saved, err := s.src.ZM.List(ctx, year)
	if err != nil {
		return err
	}
	latest := map[string]domain.ZMReturn{}
	for _, r := range saved {
		if r.Status != domain.VatReturnSubmitted {
			continue
		}
		if cur, ok := latest[r.PeriodKey]; !ok || r.ID > cur.ID {
			latest[r.PeriodKey] = r
		}
	}
	for _, r := range latest {
		for _, l := range r.Lines {
			data.ZMServices += l.Amount
			data.ZMLines++
		}
	}
	periods, err := s.src.ZM.Periods(ctx, year)
	if err != nil {
		return err
	}
	today := calendar.Today(s.now)
	for _, p := range periods {
		if p.Total != 0 && p.Status != domain.VatReturnSubmitted && p.To < today {
			data.ZMMissing = append(data.ZMMissing, p.Key)
		}
	}
	return nil
}

// ledger summiert die Buchungen des Geschäftsjahres je Konto.
func (s *DeclarationService) ledger(ctx context.Context, fiscalYear int) (declaration.Ledger, error) {
	entries, err := s.src.Journal.FindAll(ctx, fiscalYear)
	if err != nil {
		return nil, fmt.Errorf("die Buchungen des Geschäftsjahres %d ließen sich nicht lesen: %w", fiscalYear, err)
	}
	names := map[string]string{}
	if s.src.Statement != nil {
		if accounts, err := s.src.Statement.Accounts(ctx, fiscalYear); err == nil {
			for _, a := range accounts {
				names[a.Number] = a.Name
			}
		}
	}
	l := declaration.Ledger{}
	for i := range entries {
		for _, line := range entries[i].Lines {
			l.AddLine(&entries[i], line, names[line.Account])
		}
	}
	return l, nil
}

// corporateData sammelt die Größen der Körperschaft- und der
// Gewerbesteuererklärung.
func (s *DeclarationService) corporateData(ctx context.Context, common declaration.Common) (declaration.CorporateData, error) {
	fy, err := s.fiscalYearEndingIn(ctx, common.Year)
	if err != nil {
		return declaration.CorporateData{}, err
	}
	ledger, err := s.ledger(ctx, fy.Year)
	if err != nil {
		return declaration.CorporateData{}, err
	}
	stmt, err := s.src.Statement.Statement(ctx, fy.Year, domain.DepthFull)
	if err != nil {
		return declaration.CorporateData{}, fmt.Errorf("Bilanz und Gewinn- und Verlustrechnung %d ließen sich nicht aufstellen: %w", fy.Year, err)
	}
	cfg := common.Settings
	data := declaration.CorporateData{
		Ledger:        ledger,
		NetIncome:     statementAmount(stmt, "guv.17", false),
		BalanceResult: statementAmount(stmt, "passiva.A.V", false),
		Revenue:       statementAmount(stmt, "guv.1", false),
		Wages:         -statementAmount(stmt, "guv.6.a", false),
		Founded:       cfg.FoundedOn != "" && cfg.FoundedOn >= fy.StartDate && cfg.FoundedOn <= fy.EndDate,
		TradeTaxRate:  s.tradeTaxRate(ctx),
	}
	switch {
	case stmt.HasPrior:
		data.PriorKnown = true
		data.PriorEquity = statementAmount(stmt, "passiva.A", true)
		data.PriorSubscribed = statementAmount(stmt, "passiva.A.I", true)
	case s.openingEquity(ctx, fy.Year, ledger, &data):
	}

	if s.src.TaxRegister != nil {
		// Ohne Überleitung bleibt die Zeile der Steuerbilanz offen; die übrige
		// Erklärung lässt sich trotzdem rechnen.
		if rec, err := s.src.TaxRegister.Reconcile(ctx, fy.Year); err == nil {
			data.TaxBalanceEffect = rec.EquityEffect
			for _, row := range rec.Rows {
				data.TaxBalanceNotes = append(data.TaxBalanceNotes, fmt.Sprintf("%s: %s", row.Position, row.Explanation))
			}
			if prior, err := s.src.TaxRegister.Reconcile(ctx, fy.Year-1); err == nil {
				data.PriorTaxBalanceEffect = prior.EquityEffect
			} else if !data.Founded {
				data.TaxBalanceUnknown = fmt.Sprintf("Die Überleitung des Vorjahres ließ sich nicht rechnen: %v", err)
			}
		} else {
			data.TaxBalanceUnknown = fmt.Sprintf("Die Überleitung zur Steuerbilanz ließ sich nicht rechnen: %v", err)
		}
	}

	if s.src.Appropriations != nil {
		for _, y := range []int{fy.Year - 2, fy.Year - 1, fy.Year} {
			a, err := s.src.Appropriations.FindByYear(ctx, y)
			if err != nil {
				return declaration.CorporateData{}, fmt.Errorf("der Gewinnverwendungsbeschluss %d ließ sich nicht lesen: %w", y, err)
			}
			if a == nil || a.Distribution == 0 {
				continue
			}
			data.Distributions = append(data.Distributions, declaration.Distribution{
				ForYear: a.Year, DecisionDate: a.DecisionDate, Amount: a.Distribution})
		}
	}
	if s.src.Assets != nil {
		if err := s.financialAssets(ctx, fy, cfg, &data); err != nil {
			return declaration.CorporateData{}, err
		}
	}
	return data, nil
}

// openingEquity liest Eigen- und Nennkapital zum Schluss des Vorjahres aus
// dem Saldenvortrag, wenn das Vorjahr nicht in den Büchern steht.
func (s *DeclarationService) openingEquity(ctx context.Context, year int, l declaration.Ledger, data *declaration.CorporateData) bool {
	accounts, err := s.src.Statement.Accounts(ctx, year)
	if err != nil {
		return false
	}
	found := false
	for _, acc := range accounts {
		t, ok := l[acc.Number]
		if !ok || t.OpeningDebit == t.OpeningCredit {
			continue
		}
		probe := acc
		probe.DebitSum, probe.CreditSum = t.OpeningDebit, t.OpeningCredit
		key, ok := accounting.StatementKeyForAccount(probe)
		if !ok || !strings.HasPrefix(key, "passiva.A") {
			continue
		}
		found = true
		balance := t.OpeningCredit - t.OpeningDebit
		data.PriorEquity += balance
		if key == "passiva.A.I" {
			data.PriorSubscribed += balance
		}
	}
	data.PriorKnown = found
	return found
}

// financialAssets liest Beteiligungen und Investmentanteile aus der
// Anlagenkartei, mit den Erträgen, die über sie gebucht sind.
func (s *DeclarationService) financialAssets(ctx context.Context, fy *domain.FiscalYear, cfg domain.CompanySettings, data *declaration.CorporateData) error {
	assets, err := s.src.Assets.FindByClass(ctx, domain.AssetClassFinancial)
	if err != nil {
		return err
	}
	entries, err := s.src.Journal.FindAll(ctx, fy.Year)
	if err != nil {
		return err
	}
	byID := map[uint]*domain.JournalEntry{}
	for i := range entries {
		byID[entries[i].ID] = &entries[i]
	}
	dividendAccounts := map[string]bool{}
	for _, a := range []string{"7005", "7006", "7014", "7015", "7103", "7104"} {
		dividendAccounts[a] = true
	}
	investor, _ := cfg.InvestorTypeOrDerived()
	in := func(date string) bool { return date >= fy.StartDate && date <= fy.EndDate }

	for _, a := range assets {
		if a.DisposalDate != "" && a.DisposalDate < fy.StartDate {
			continue
		}
		fund := accounting.FundClass(a.FundClass)
		switch {
		case fund.IsFund():
			f := declaration.Fund{AssetID: a.ID, Name: a.Name, DisposedInYear: a.DisposalDate != "" && in(a.DisposalDate)}
			if ex, err := accounting.PartialExemptionFor(fund, investor); err == nil {
				f.ExemptPermille, f.Determined = ex.Permille, ex.Determined
			}
			for _, m := range a.Movements {
				switch {
				case m.Kind == domain.AssetMovementVorabpauschale && in(m.Date):
					f.Vorabpauschale += m.TaxAmount
				case m.Kind == domain.AssetMovementIncome && in(m.Date) && m.JournalEntryID != nil:
					if e := byID[*m.JournalEntryID]; e != nil {
						on, off := financialIncome(e, dividendAccounts)
						f.Income += on + off
						f.EntryIDs = append(f.EntryIDs, e.ID)
					}
				case m.Kind == domain.AssetMovementDisposal && in(m.Date):
					f.DisposedInYear = true
				}
			}
			if f.Income != 0 || f.Vorabpauschale != 0 || f.DisposedInYear {
				data.Funds = append(data.Funds, f)
			}
		case a.TaxPrivileged:
			h := declaration.Holding{AssetID: a.ID, Name: a.Name, QuotaPermille: a.HoldingPermille,
				AcquiredOn: a.AcquisitionDate, DisposedOn: a.DisposalDate}
			for _, m := range a.Movements {
				if m.Kind != domain.AssetMovementIncome || !in(m.Date) || m.JournalEntryID == nil {
					continue
				}
				if e := byID[*m.JournalEntryID]; e != nil {
					on, off := financialIncome(e, dividendAccounts)
					h.Dividends += on + off
					h.OtherAccounts += off
					h.EntryIDs = append(h.EntryIDs, e.ID)
				}
			}
			if h.Dividends != 0 {
				data.Holdings = append(data.Holdings, h)
			}
		}
	}
	return nil
}

// financialIncome ist der Finanzertrag einer Buchung: die Habenzeilen auf
// den Ertragskonten 7000 bis 7199, getrennt nach den §-8b-Konten und den
// übrigen.
func financialIncome(e *domain.JournalEntry, dividendAccounts map[string]bool) (on8b, other domain.Cents) {
	for _, l := range e.Lines {
		if l.Side != domain.SideCredit || len(l.Account) != 4 || l.Account < "7000" || l.Account > "7199" {
			continue
		}
		if dividendAccounts[l.Account] {
			on8b += l.Amount
		} else {
			other += l.Amount
		}
	}
	return on8b, other
}

func (s *DeclarationService) tradeTaxRate(ctx context.Context) int64 {
	value, err := s.src.Settings.Get(ctx, domain.SettingTradeTaxRate)
	if err != nil {
		return 0
	}
	rate, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || rate <= 0 {
		return 0
	}
	return rate
}

// statementAmount liest einen Posten der Gliederung, im Jahr oder im Vorjahr.
func statementAmount(stmt *domain.Statement, key string, prior bool) domain.Cents {
	for _, lines := range [][]domain.StatementLine{stmt.Assets, stmt.Liabilities, stmt.Income} {
		for _, l := range lines {
			if l.Key == key {
				if prior {
					return l.PriorAmount
				}
				return l.Amount
			}
		}
	}
	return 0
}
