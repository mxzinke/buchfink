package accounting

import (
	"fmt"
	"sort"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// AfAParameters are the statutory value limits the treatment of an acquisition
// depends on. Like TaxParameters they are keyed by the day they apply to.
//
// They are not editable master data on purpose. Whether something is a GWG is
// not a choice; the limit has changed several times, and a constant hard-wired
// into the code would silently misbook a reworked prior year, while an editable
// field invites a typo to do the same.
type AfAParameters struct {
	// ValidFrom is the first day this set applies to, as ISO date.
	ValidFrom string

	// GWGImmediateLimit is the ceiling for the Sofortabzug of § 6 Abs. 2 Satz 1
	// EStG — net of Vorsteuer, and only for selbständig nutzbare Güter.
	GWGImmediateLimit domain.Cents
	// GWGRecordThreshold is the amount from which a GWG has to be recorded in a
	// besonderes, laufend zu führendes Verzeichnis (§ 6 Abs. 2 Satz 4 EStG).
	// Satz 5 makes the record unnecessary where the figures are already visible
	// in the bookkeeping — which the Anlagenverzeichnis is.
	GWGRecordThreshold domain.Cents
	// PoolLowerLimit and PoolUpperLimit bound the Sammelposten of § 6 Abs. 2a
	// Satz 1 EStG.
	PoolLowerLimit domain.Cents
	PoolUpperLimit domain.Cents
	// PoolYears is the number of fiscal years a Sammelposten is dissolved over:
	// the year it is formed and the following four (§ 6 Abs. 2a Satz 2 EStG).
	PoolYears int
}

// afaParameterSets are ordered by ValidFrom, oldest first. Historical values are
// kept rather than overwritten: an asset acquired in 2020 has to stay
// explainable with the limits that applied then.
var afaParameterSets = []AfAParameters{
	{
		// § 6 Abs. 2 und 2a EStG in der ab 2018 geltenden Fassung.
		ValidFrom:          "2018-01-01",
		GWGImmediateLimit:  80000,
		GWGRecordThreshold: 25000,
		PoolLowerLimit:     25000,
		PoolUpperLimit:     100000,
		PoolYears:          5,
	},
}

// AfAParametersFor returns the limits that applied on a date.
func AfAParametersFor(date string) (AfAParameters, error) {
	if date == "" {
		return AfAParameters{}, fmt.Errorf("ohne Datum lassen sich die Wertgrenzen nicht bestimmen")
	}
	idx := sort.Search(len(afaParameterSets), func(i int) bool {
		return afaParameterSets[i].ValidFrom > date
	})
	if idx == 0 {
		return AfAParameters{}, fmt.Errorf(
			"für den %s sind keine Wertgrenzen hinterlegt. Buchfink führt sie ab dem %s",
			date, afaParameterSets[0].ValidFrom)
	}
	return afaParameterSets[idx-1], nil
}

// DegressiveWindow is a period in which the degressive AfA of § 7 Abs. 2 EStG
// was open, together with the ceiling that applied in it.
//
// The degressive method is not a permanent part of the law. It is switched on
// and off by the legislator for defined acquisition periods, and outside such a
// window it simply does not exist. That is why this is a table of windows and
// not a flag: an asset acquired in 2024 and one acquired in 2026 do not follow
// the same rule, and both have to stay computable.
type DegressiveWindow struct {
	// From and Until bound the acquisition date, both inclusive.
	From  string
	Until string
	// FactorTimes is the multiple of the linear percentage the degressive rate
	// may reach, MaxPermille the absolute ceiling in permille.
	FactorTimes int64
	MaxPermille int64
	// Source names the provision, for the message the user gets.
	Source string
}

// degressiveWindows lists the periods Buchfink computes a degressive AfA for.
//
// Only the current window is listed. § 7 Abs. 2 EStG in its present wording
// covers acquisitions after 30.06.2025 and before 01.01.2028; the earlier
// degressive periods stood in earlier versions of the same provision and are not
// reproduced here. Refusing a degressive plan outside the window is the honest
// answer — computing one from a rule that is not in the law would be worse.
var degressiveWindows = []DegressiveWindow{
	{
		From:        "2025-07-01",
		Until:       "2027-12-31",
		FactorTimes: 3,
		MaxPermille: 300,
		Source:      "§ 7 Abs. 2 Sätze 1 und 2 EStG",
	},
}

// DegressiveWindowFor returns the window an acquisition date falls into.
func DegressiveWindowFor(acquisitionDate string) (DegressiveWindow, bool) {
	for _, w := range degressiveWindows {
		if acquisitionDate >= w.From && acquisitionDate <= w.Until {
			return w, true
		}
	}
	return DegressiveWindow{}, false
}

// DegressiveWindows exposes the table for the UI, so the hint the user reads is
// the same table the computation uses.
func DegressiveWindows() []DegressiveWindow {
	out := make([]DegressiveWindow, len(degressiveWindows))
	copy(out, degressiveWindows)
	return out
}

// AcquisitionOption is the answer to the first question of every acquisition:
// expense now, pool, or activate and depreciate.
type AcquisitionOption string

const (
	AcquisitionImmediate AcquisitionOption = "immediate"
	AcquisitionPool      AcquisitionOption = "pool"
	AcquisitionActivate  AcquisitionOption = "activate"
)

// AcquisitionAdvice explains which options an acquisition has and why.
//
// It answers rather than decides. The decisive criterion is not the amount but
// whether the item is *selbständig nutzbar* — a 300 € monitor is not, and is
// therefore no GWG however cheap it is. Buchfink cannot know that, so it asks
// and records the answer instead of guessing.
type AcquisitionAdvice struct {
	Recommended AcquisitionOption   `json:"recommended"`
	Allowed     []AcquisitionOption `json:"allowed"`
	// Reason is one paragraph in plain German, naming the provision.
	Reason string `json:"reason"`
	// PoolNote carries the catch of the Sammelposten: the Wahlrecht is exercised
	// uniformly for all acquisitions of a fiscal year.
	PoolNote string `json:"poolNote,omitempty"`
	Limits   struct {
		Immediate      domain.Cents `json:"immediate"`
		RecordFrom     domain.Cents `json:"recordFrom"`
		PoolLowerLimit domain.Cents `json:"poolLowerLimit"`
		PoolUpperLimit domain.Cents `json:"poolUpperLimit"`
	} `json:"limits"`
}

// ClassifyAcquisition applies § 6 Abs. 2 and Abs. 2a EStG to one acquisition.
//
// netCost is the Anschaffungskosten without Vorsteuer, as both provisions
// measure the limit net wherever the input tax is deductible.
func ClassifyAcquisition(netCost domain.Cents, date string, selfUsable bool) (AcquisitionAdvice, error) {
	params, err := AfAParametersFor(date)
	if err != nil {
		return AcquisitionAdvice{}, err
	}

	advice := AcquisitionAdvice{}
	advice.Limits.Immediate = params.GWGImmediateLimit
	advice.Limits.RecordFrom = params.GWGRecordThreshold
	advice.Limits.PoolLowerLimit = params.PoolLowerLimit
	advice.Limits.PoolUpperLimit = params.PoolUpperLimit

	if !selfUsable {
		advice.Recommended = AcquisitionActivate
		advice.Allowed = []AcquisitionOption{AcquisitionActivate}
		advice.Reason = "Das Wirtschaftsgut ist nicht selbständig nutzbar. Damit scheiden Sofortabzug und " +
			"Sammelposten aus, unabhängig vom Betrag: § 6 Abs. 2 Satz 1 EStG setzt beim geringwertigen " +
			"Wirtschaftsgut die selbständige Nutzbarkeit voraus. Ein Bildschirm für 300 € ist ohne Rechner " +
			"nicht nutzbar und deshalb kein GWG — er wird aktiviert und über die Nutzungsdauer abgeschrieben."
		return advice, nil
	}

	switch {
	case netCost <= params.PoolLowerLimit:
		advice.Recommended = AcquisitionImmediate
		advice.Allowed = []AcquisitionOption{AcquisitionImmediate, AcquisitionActivate}
		advice.Reason = fmt.Sprintf(
			"Bis %s € netto ist der Sofortabzug nach § 6 Abs. 2 Satz 1 EStG möglich. Unterhalb von %s € "+
				"besteht dafür nicht einmal eine Verzeichnispflicht (§ 6 Abs. 2 Satz 4 EStG).",
			params.GWGImmediateLimit, params.GWGRecordThreshold)
	case netCost <= params.GWGImmediateLimit:
		advice.Recommended = AcquisitionImmediate
		advice.Allowed = []AcquisitionOption{AcquisitionImmediate, AcquisitionPool, AcquisitionActivate}
		advice.Reason = fmt.Sprintf(
			"Zwischen %s € und %s € netto stehen alle drei Wege offen: Sofortabzug (§ 6 Abs. 2 Satz 1 EStG), "+
				"Sammelposten (§ 6 Abs. 2a EStG) oder Aktivierung mit planmäßiger AfA. Ab %s € gehört das Gut "+
				"in ein laufend geführtes Verzeichnis (§ 6 Abs. 2 Satz 4 EStG) — das Anlagenverzeichnis erfüllt das.",
			params.PoolLowerLimit, params.GWGImmediateLimit, params.GWGRecordThreshold)
	case netCost <= params.PoolUpperLimit:
		advice.Recommended = AcquisitionPool
		advice.Allowed = []AcquisitionOption{AcquisitionPool, AcquisitionActivate}
		advice.Reason = fmt.Sprintf(
			"Über %s € ist der Sofortabzug ausgeschlossen. Bis %s € netto kann das Gut in den Sammelposten "+
				"des Wirtschaftsjahres eingestellt werden (§ 6 Abs. 2a Satz 1 EStG); sonst wird es aktiviert "+
				"und über seine Nutzungsdauer abgeschrieben.",
			params.GWGImmediateLimit, params.PoolUpperLimit)
	default:
		advice.Recommended = AcquisitionActivate
		advice.Allowed = []AcquisitionOption{AcquisitionActivate}
		advice.Reason = fmt.Sprintf(
			"Über %s € netto bleibt nur die Aktivierung: das Gut kommt auf ein Anlagekonto und wird über die "+
				"betriebsgewöhnliche Nutzungsdauer abgeschrieben (§ 7 Abs. 1 EStG).",
			params.PoolUpperLimit)
	}

	for _, o := range advice.Allowed {
		if o == AcquisitionPool {
			advice.PoolNote = "Das Wahlrecht zum Sammelposten gilt einheitlich für alle Wirtschaftsgüter " +
				"eines Wirtschaftsjahres (§ 6 Abs. 2a Satz 5 EStG). Wer einmal poolt, poolt in diesem Jahr durchgehend."
			break
		}
	}
	return advice, nil
}

// AfAPlan is everything the schedule of one asset is computed from. It is a
// value, not a database row: the same computation runs for the preview of an
// asset that does not exist yet and for the yearly Abschreibungslauf.
type AfAPlan struct {
	AcquisitionDate string
	// Cost is the AfA-Bemessungsgrundlage: Anschaffungskosten plus nachträgliche
	// Anschaffungskosten minus Anschaffungspreisminderungen.
	Cost             domain.Cents
	UsefulLifeMonths int
	Method           domain.DepreciationMethod
	// FiscalYearStartMonth is 1 for a calendar fiscal year.
	FiscalYearStartMonth int
	// PoolYear is the fiscal year a Sammelposten was formed in.
	PoolYear int
	// DisposalDate truncates the plan. AfA runs up to and including the month of
	// the Abgang; what is left after that is the Restbuchwert the disposal
	// booking clears, not a further AfA.
	DisposalDate string
	// ImpairmentsByYear lowers the book value the degressive computation works
	// from. A planmäßige AfA after an außerplanmäßigen Abschreibung is computed
	// from the reduced value, not from the original one.
	ImpairmentsByYear map[int]domain.Cents
}

// AfAYear is one fiscal year of the plan.
type AfAYear struct {
	FiscalYear int `json:"fiscalYear"`
	// Months is how many months of this fiscal year the asset is depreciated in.
	// The first year is regularly short: AfA runs monatsgenau ab dem
	// Anschaffungsmonat (§ 7 Abs. 1 Satz 4 EStG).
	Months           int                       `json:"months"`
	Method           domain.DepreciationMethod `json:"method"`
	RateLabel        string                    `json:"rateLabel"`
	OpeningBookValue domain.Cents              `json:"openingBookValue"`
	Amount           domain.Cents              `json:"amount"`
	ClosingBookValue domain.Cents              `json:"closingBookValue"`
	Note             string                    `json:"note,omitempty"`
}

// BuildAfASchedule computes the whole plan of an asset, year by year.
//
// Everything the Anlagenbuchhaltung shows is derived from here: the yearly
// Abschreibungslauf, the preview in the input mask and the AfA that has to be
// caught up before a disposal. One computation, one truth — a second one in the
// user interface would drift the day a rule changes.
func BuildAfASchedule(plan AfAPlan) ([]AfAYear, error) {
	if plan.Cost <= 0 {
		return nil, nil
	}
	if len(plan.AcquisitionDate) != 10 {
		return nil, fmt.Errorf("das Anschaffungsdatum fehlt oder ist unvollständig")
	}
	start := plan.FiscalYearStartMonth
	if start <= 0 || start > 12 {
		start = 1
	}

	switch plan.Method {
	case domain.DepreciationNone:
		return nil, nil
	case domain.DepreciationImmediate:
		year := domain.GetFiscalYearForDate(plan.AcquisitionDate, start)
		return []AfAYear{{
			FiscalYear:       year,
			Months:           1,
			Method:           domain.DepreciationImmediate,
			RateLabel:        "100 %",
			OpeningBookValue: plan.Cost,
			Amount:           plan.Cost,
			ClosingBookValue: 0,
			Note:             "Sofortabzug im Jahr der Anschaffung (§ 6 Abs. 2 Satz 1 EStG).",
		}}, nil
	case domain.DepreciationPool:
		return poolSchedule(plan)
	case domain.DepreciationLinear, domain.DepreciationDegressive:
	default:
		return nil, fmt.Errorf("unbekannte Abschreibungsmethode %q", plan.Method)
	}

	if plan.UsefulLifeMonths <= 0 {
		return nil, fmt.Errorf("ohne Nutzungsdauer lässt sich keine planmäßige Abschreibung rechnen")
	}

	window := DegressiveWindow{}
	degressive := plan.Method == domain.DepreciationDegressive
	if degressive {
		w, ok := DegressiveWindowFor(plan.AcquisitionDate)
		if !ok {
			return nil, fmt.Errorf(
				"für eine Anschaffung am %s ist die degressive Abschreibung nicht zulässig. "+
					"%s öffnet sie nur für Anschaffungen zwischen dem %s und dem %s; außerhalb dieses "+
					"Zeitraums bleibt die lineare Abschreibung",
				plan.AcquisitionDate, degressiveWindows[0].Source,
				degressiveWindows[0].From, degressiveWindows[0].Until)
		}
		window = w
	}

	afaStart, err := monthStart(plan.AcquisitionDate)
	if err != nil {
		return nil, err
	}
	naturalEnd := afaStart.AddDate(0, plan.UsefulLifeMonths, 0)
	afaEnd := naturalEnd
	truncated := false
	if plan.DisposalDate != "" {
		disposal, err := monthStart(plan.DisposalDate)
		if err != nil {
			return nil, err
		}
		// Im Abgangsmonat wird noch abgeschrieben, danach nicht mehr.
		end := disposal.AddDate(0, 1, 0)
		if end.Before(afaEnd) {
			afaEnd = end
			truncated = true
		}
	}

	var rows []AfAYear
	bookValue := plan.Cost
	remainingMonths := monthsBetween(afaStart, naturalEnd)
	switchedToLinear := false
	impaired := false

	for year := domain.GetFiscalYearForDate(plan.AcquisitionDate, start); ; year++ {
		fyStart := time.Date(year, time.Month(start), 1, 0, 0, 0, 0, time.UTC)
		fyEnd := fyStart.AddDate(1, 0, 0)
		months := overlapMonths(afaStart, afaEnd, fyStart, fyEnd)
		if months <= 0 {
			if !fyStart.Before(afaEnd) {
				break
			}
			continue
		}
		if bookValue <= 0 {
			break
		}

		// Eine außerplanmäßige Abschreibung dieses Jahres mindert den Wert, von
		// dem die planmäßige AfA der Folgejahre ausgeht.
		row := AfAYear{
			FiscalYear:       year,
			Months:           months,
			Method:           domain.DepreciationLinear,
			OpeningBookValue: bookValue,
		}

		isFinal := !truncated && remainingMonths-months <= 0
		switch {
		case isFinal:
			row.Amount = bookValue
			row.RateLabel = "Restwert"
			row.Note = "Letztes Jahr der Nutzungsdauer: der Restbuchwert wird vollständig abgeschrieben."
		case degressive && !switchedToLinear:
			num, den := degressiveRate(plan.UsefulLifeMonths, window)
			annualDeg := domain.MulRound(bookValue, num, den)
			degAmount := domain.MulRound(annualDeg, int64(months), 12)

			annualLin := domain.MulRound(bookValue, 12, int64(remainingMonths))
			linAmount := domain.MulRound(annualLin, int64(months), 12)

			if linAmount > degAmount {
				// § 7 Abs. 3 EStG erlaubt den Übergang zur linearen AfA. Er lohnt
				// sich genau ab dem Jahr, in dem die Restwert-AfA höher ausfällt —
				// und danach nie wieder zurück.
				switchedToLinear = true
				row.Amount = linAmount
				row.Method = domain.DepreciationLinear
				row.RateLabel = fmt.Sprintf("linear auf %d Restmonate", remainingMonths)
				row.Note = "Übergang zur linearen Abschreibung (§ 7 Abs. 3 EStG): ab hier ist die " +
					"Restwertabschreibung höher als die degressive."
			} else {
				row.Amount = degAmount
				row.Method = domain.DepreciationDegressive
				row.RateLabel = permilleLabel(num, den)
			}
		case degressive && switchedToLinear:
			annualLin := domain.MulRound(bookValue, 12, int64(remainingMonths))
			row.Amount = domain.MulRound(annualLin, int64(months), 12)
			row.RateLabel = fmt.Sprintf("linear auf %d Restmonate", remainingMonths)
		case impaired:
			// Nach einer außerplanmäßigen Abschreibung wird die planmäßige AfA
			// vom geminderten Wert auf die Restnutzungsdauer neu verteilt. Weiter
			// von den ursprünglichen Anschaffungskosten abzuschreiben würde das
			// Anlagegut vor dem Ende seiner Nutzungsdauer auf null bringen.
			annual := domain.MulRound(bookValue, 12, int64(remainingMonths))
			row.Amount = domain.MulRound(annual, int64(months), 12)
			row.RateLabel = fmt.Sprintf("linear auf %d Restmonate", remainingMonths)
		default:
			annual := domain.MulRound(plan.Cost, 12, int64(plan.UsefulLifeMonths))
			row.Amount = domain.MulRound(annual, int64(months), 12)
			row.RateLabel = permilleLabel(12, int64(plan.UsefulLifeMonths))
		}

		if row.Amount > bookValue {
			row.Amount = bookValue
		}
		if months < 12 && !isFinal {
			row.Note = appendNote(row.Note, fmt.Sprintf(
				"Zeitanteilig für %d von 12 Monaten (§ 7 Abs. 1 Satz 4 EStG).", months))
		}

		bookValue -= row.Amount
		if impair := plan.ImpairmentsByYear[year]; impair > 0 {
			bookValue -= impair
			if bookValue < 0 {
				bookValue = 0
			}
			impaired = true
			row.Note = appendNote(row.Note, fmt.Sprintf(
				"Zusätzlich außerplanmäßig abgeschrieben: %s €.", impair))
		}
		row.ClosingBookValue = bookValue
		rows = append(rows, row)

		remainingMonths -= months
		if remainingMonths <= 0 || bookValue <= 0 || !fyEnd.Before(afaEnd) {
			break
		}
	}

	return rows, nil
}

// poolSchedule dissolves a Sammelposten by one fifth per fiscal year.
//
// Two things are deliberately different from every other method: there is no
// pro rata temporis — the fifth is due in full in the year the pool is formed —
// and the actual useful life of the items is irrelevant. Both follow from
// § 6 Abs. 2a Sätze 2 und 3 EStG, and both are the reason a pool cannot be
// modelled as an ordinary asset with a five-year life.
func poolSchedule(plan AfAPlan) ([]AfAYear, error) {
	params, err := AfAParametersFor(plan.AcquisitionDate)
	if err != nil {
		return nil, err
	}
	years := params.PoolYears
	if years <= 0 {
		years = 5
	}
	first := plan.PoolYear
	if first == 0 {
		start := plan.FiscalYearStartMonth
		if start <= 0 {
			start = 1
		}
		first = domain.GetFiscalYearForDate(plan.AcquisitionDate, start)
	}

	rows := make([]AfAYear, 0, years)
	bookValue := plan.Cost
	share := domain.MulRound(plan.Cost, 1, int64(years))
	for i := 0; i < years; i++ {
		amount := share
		if i == years-1 || amount > bookValue {
			amount = bookValue
		}
		opening := bookValue
		bookValue -= amount
		rows = append(rows, AfAYear{
			FiscalYear:       first + i,
			Months:           12,
			Method:           domain.DepreciationPool,
			RateLabel:        fmt.Sprintf("1/%d", years),
			OpeningBookValue: opening,
			Amount:           amount,
			ClosingBookValue: bookValue,
			Note: fmt.Sprintf(
				"Auflösung des Sammelpostens %d mit einem Fünftel, ohne Zeitanteil (§ 6 Abs. 2a Satz 2 EStG).",
				first),
		})
		if bookValue <= 0 {
			break
		}
	}
	return rows, nil
}

// degressiveRate returns the degressive percentage as a fraction: the multiple
// of the linear rate, capped by the statutory ceiling.
func degressiveRate(usefulLifeMonths int, window DegressiveWindow) (int64, int64) {
	// Der lineare Satz ist 12/Nutzungsdauer in Monaten; das Vielfache davon ist
	// FactorTimes * 12 / Monate. Verglichen wird als Bruch, damit kein
	// Zwischenrunden den Deckel verschiebt.
	num := window.FactorTimes * 12
	den := int64(usefulLifeMonths)
	if num*1000 > window.MaxPermille*den {
		return window.MaxPermille, 1000
	}
	return num, den
}

func permilleLabel(num, den int64) string {
	if den == 0 {
		return ""
	}
	permille := (num*1000 + den/2) / den
	whole := permille / 10
	frac := permille % 10
	if frac == 0 {
		return fmt.Sprintf("%d %%", whole)
	}
	return fmt.Sprintf("%d,%d %%", whole, frac)
}

func appendNote(existing, addition string) string {
	if existing == "" {
		return addition
	}
	return existing + " " + addition
}

func monthStart(date string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return time.Time{}, fmt.Errorf("%q ist kein Datum im Format JJJJ-MM-TT", date)
	}
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), nil
}

func monthsBetween(from, to time.Time) int {
	return int(to.Year()-from.Year())*12 + int(to.Month()) - int(from.Month())
}

// overlapMonths counts the months two half-open month ranges share.
func overlapMonths(aStart, aEnd, bStart, bEnd time.Time) int {
	start := aStart
	if bStart.After(start) {
		start = bStart
	}
	end := aEnd
	if bEnd.Before(end) {
		end = bEnd
	}
	months := monthsBetween(start, end)
	if months < 0 {
		return 0
	}
	return months
}

// ScheduleAmountFor returns the planned AfA of one fiscal year, or zero if the
// year carries none.
func ScheduleAmountFor(rows []AfAYear, fiscalYear int) domain.Cents {
	for _, r := range rows {
		if r.FiscalYear == fiscalYear {
			return r.Amount
		}
	}
	return 0
}
