package accounting

import (
	"fmt"
	"sort"
	"strings"
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
// Die Werte selbst stehen nicht mehr hier, sondern in afa_rules.json — siehe
// afa_rules.go. Historische Fassungen werden dort aufgehoben und nicht
// überschrieben: ein 2020 angeschafftes Wirtschaftsgut muss mit den Grenzen
// erklärbar bleiben, die damals galten.
type AfAParameters struct {
	// ValidFrom is the first day this set applies to, as ISO date.
	ValidFrom string `json:"validFrom"`
	// Note ist der Satz aus der Ressource, der den Eintrag einordnet.
	Note string `json:"note,omitempty"`

	// GWGImmediateLimit is the ceiling for the Sofortabzug of § 6 Abs. 2 Satz 1
	// EStG — net of Vorsteuer, and only for selbständig nutzbare Güter.
	GWGImmediateLimit domain.Cents `json:"gwgImmediateLimit"`
	// GWGRecordThreshold is the amount from which a GWG has to be recorded in a
	// besonderes, laufend zu führendes Verzeichnis (§ 6 Abs. 2 Satz 4 EStG).
	// Satz 5 makes the record unnecessary where the figures are already visible
	// in the bookkeeping — which the Anlagenverzeichnis is.
	GWGRecordThreshold domain.Cents `json:"gwgRecordThreshold"`
	// PoolLowerLimit and PoolUpperLimit bound the Sammelposten of § 6 Abs. 2a
	// Satz 1 EStG.
	PoolLowerLimit domain.Cents `json:"poolLowerLimit"`
	PoolUpperLimit domain.Cents `json:"poolUpperLimit"`
	// PoolYears is the number of fiscal years a Sammelposten is dissolved over:
	// the year it is formed and the following four (§ 6 Abs. 2a Satz 2 EStG).
	PoolYears int `json:"poolYears"`
}

// afaParameterSets are ordered by ValidFrom, oldest first.
var afaParameterSets = afaRules.ParameterSets

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
	From  string `json:"from"`
	Until string `json:"until"`
	// FactorPermille is the multiple of the linear percentage the degressive
	// rate may reach, in thousandths — 3000 für das Dreifache, 2500 für das
	// Zweieinhalbfache. Der Faktor war in einer der Fassungen ein halber, ein
	// ganzzahliges Feld hätte ihn stillschweigend auf zwei oder drei gerundet.
	// MaxPermille is the absolute ceiling in permille.
	FactorPermille int64 `json:"factorPermille"`
	MaxPermille    int64 `json:"maxPermille"`
	// Source names the provision, for the message the user gets.
	Source string `json:"source"`
}

// degressiveWindows lists the periods Buchfink computes a degressive AfA for.
//
// Die Fenster sind nicht historisch interessant, sondern Rechenvoraussetzung:
// die Anlagenkartei wird über die Jahre geführt, und ein 2021 angeschafftes
// Wirtschaftsgut schreibt heute noch nach dem Satz ab, der damals galt. Ohne die
// alten Fassungen wäre sein Plan nicht mehr rechenbar — und ein Plan, den die
// Software verweigert, obwohl das Gesetz ihn eröffnet hat, ist so falsch wie
// einer, den sie erfindet.
//
// Zwischen den Fenstern liegen Lücken, und die sind gewollt: 2023, das erste
// Quartal 2024 und das erste Halbjahr 2025 kannten keine degressive AfA. Dort
// bleibt es bei der linearen.
var degressiveWindows = afaRules.DegressiveWindows

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

// degressiveWindowList renders the open periods for an error message. Es nützt
// niemandem zu erfahren, dass ein Datum außerhalb liegt, ohne zu erfahren, wo
// die Grenzen verlaufen.
func degressiveWindowList() string {
	parts := make([]string, 0, len(degressiveWindows))
	for _, w := range degressiveWindows {
		parts = append(parts, fmt.Sprintf("%s bis %s", germanDate(w.From), germanDate(w.Until)))
	}
	return strings.Join(parts, ", ")
}

// germanDate turns an ISO date into the form a German reader expects.
func germanDate(iso string) string {
	if len(iso) != 10 {
		return iso
	}
	return iso[8:10] + "." + iso[5:7] + "." + iso[0:4]
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

	// BasisChangesByYear carries nachträgliche Anschaffungs- oder
	// Herstellungskosten und Anschaffungspreisminderungen, dem Geschäftsjahr
	// zugeordnet, in dem sie angefallen sind.
	//
	// Sie ändern die Vergangenheit nicht. Der Betrag wird so behandelt, als wäre
	// er zu Beginn seines Jahres angefallen (R 7.4 Abs. 9 EStR), und von da an
	// verteilt sich der neue Restbuchwert auf die Restnutzungsdauer. Den ganzen
	// Plan von vorn zu rechnen wäre der naheliegende Fehler: er behauptete
	// rückwirkend, in längst abgeschlossenen Jahren sei zu wenig abgeschrieben
	// worden.
	BasisChangesByYear map[int]domain.Cents

	// LifeExtensionsByYear verlängert die Nutzungsdauer ab dem genannten
	// Geschäftsjahr. Eine Erweiterung, die das Wirtschaftsgut länger nutzbar
	// macht, wirkt nach vorn — die bereits gebuchten Jahre bleiben, wie sie sind.
	LifeExtensionsByYear map[int]int

	// SpecialPermille ist der Satz der Sonderabschreibung nach § 7g Abs. 5 EStG
	// in Promille der Anschaffungskosten, SpecialYears die Zahl der Jahre, auf
	// die er gleichmäßig verteilt wird. Null heißt: keine Sonderabschreibung.
	SpecialPermille int
	SpecialYears    int

	// BuildingPermille ist der feste Jahressatz des § 7 Abs. 4 EStG in Promille
	// der Anschaffungskosten. Er wird vom Aufrufer aus BuildingRateFor
	// aufgelöst — die Regel hängt an Wohnzweck und Stichtag, und beides steht am
	// Anlagegut und nicht am Plan.
	BuildingPermille int64
}

// Die Sonderabschreibung des § 7g Abs. 5 EStG in Zahlen.
const (
	// SpecialMaxPermille sind die höchstens 40 Prozent der Anschaffungs- oder
	// Herstellungskosten, die § 7g Abs. 5 EStG zulässt.
	SpecialMaxPermille = 400
	// SpecialPeriodYears ist der Begünstigungszeitraum: das Jahr der Anschaffung
	// und die vier folgenden. Über ihn darf verteilt werden, und mit seinem Ende
	// beginnt die Restwertabschreibung des § 7a Abs. 9 EStG.
	SpecialPeriodYears = 5
)

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
	// SpecialAmount ist die Sonderabschreibung des Jahres (§ 7g Abs. 5 EStG).
	//
	// Sie steht neben der planmäßigen AfA und nicht an ihrer Stelle
	// (§ 7a Abs. 4 EStG) — und sie läuft im SKR04 über ein eigenes
	// Aufwandskonto. Beides ist der Grund, warum sie hier ein zweites Feld ist
	// und nicht in Amount aufgeht: eine Summe ließe sich nicht mehr auf zwei
	// Konten buchen.
	SpecialAmount    domain.Cents `json:"specialAmount,omitempty"`
	ClosingBookValue domain.Cents `json:"closingBookValue"`

	// TaxAmount ist die planmäßige AfA derselben Jahres in der *steuerlichen*
	// Rechnung, TaxClosingBookValue der steuerliche Restbuchwert danach.
	//
	// Ohne Sonderabschreibung sind beide Zahlen die handelsrechtlichen: die
	// Einheitsbilanz ist der Regelfall. Erst § 7g Abs. 5 EStG trennt die beiden
	// Rechnungen — die Sonderabschreibung mindert allein den steuerlichen
	// Wertansatz, und nach dem Begünstigungszeitraum verteilt § 7a Abs. 9 EStG
	// den dann verbliebenen steuerlichen Restwert auf die Restnutzungsdauer.
	// Der handelsrechtliche Plan läuft daneben unverändert bis auf null
	// (§ 253 HGB; § 254 HGB a. F. ist mit dem BilMoG entfallen).
	TaxAmount           domain.Cents `json:"taxAmount"`
	TaxClosingBookValue domain.Cents `json:"taxClosingBookValue"`

	Note string `json:"note,omitempty"`
}

// TotalAmount is what the fiscal year writes off altogether — handelsrechtlich
// planmäßig und steuerlich zusätzlich.
func (y AfAYear) TotalAmount() domain.Cents { return y.Amount + y.SpecialAmount }

// TaxDifference ist der Betrag, um den die steuerliche Abschreibung des Jahres
// die handelsrechtliche übersteigt. Er ist die Zeile des Verzeichnisses nach
// § 5 Abs. 1 Satz 2 EStG und kehrt sich nach dem Begünstigungszeitraum um.
func (y AfAYear) TaxDifference() domain.Cents { return y.TaxAmount + y.SpecialAmount - y.Amount }

// BuildAfASchedule computes the whole plan of an asset, year by year.
//
// Everything the Anlagenbuchhaltung shows is derived from here: the yearly
// Abschreibungslauf, the preview in the input mask and the AfA that has to be
// caught up before a disposal. One computation, one truth — a second one in the
// user interface would drift the day a rule changes.
func BuildAfASchedule(plan AfAPlan) ([]AfAYear, error) {
	// Ohne Anschaffungskosten gibt es keinen Plan — aber eine leere Liste und
	// kein nil: der Plan geht als JSON in die Maske, die ihn beim Tippen liest.
	if plan.Cost <= 0 {
		return []AfAYear{}, nil
	}
	if len(plan.AcquisitionDate) != 10 {
		return nil, fmt.Errorf("das Anschaffungsdatum fehlt oder ist unvollständig")
	}
	start := plan.FiscalYearStartMonth
	if start <= 0 || start > 12 {
		start = 1
	}
	acquisitionYear := domain.GetFiscalYearForDate(plan.AcquisitionDate, start)

	// Was im Zugangsjahr selbst hinzukommt oder wegfällt — Fracht, Montage, ein
	// Skonto —, gehört zu den Anschaffungskosten und nicht zu einer späteren
	// Änderung der Bemessungsgrundlage. Beim Sammelposten gilt das für alle
	// Zugänge: er besteht nur aus Gütern seines eigenen Wirtschaftsjahres.
	baseCost := plan.Cost
	lifeMonths := plan.UsefulLifeMonths
	later := map[int]domain.Cents{}
	for year, amount := range plan.BasisChangesByYear {
		if year <= acquisitionYear || plan.Method == domain.DepreciationPool {
			baseCost += amount
			continue
		}
		later[year] = amount
	}
	laterLife := map[int]int{}
	for year, months := range plan.LifeExtensionsByYear {
		if year <= acquisitionYear {
			lifeMonths += months
			continue
		}
		laterLife[year] = months
	}
	plan.Cost = baseCost
	plan.UsefulLifeMonths = lifeMonths

	// Vor der Methodenweiche geprüft: der Sammelposten und der Sofortabzug
	// kehren gleich danach zurück, und ein unzulässiger Satz fiele dort nie auf.
	if err := checkSpecialDepreciation(plan); err != nil {
		return nil, err
	}

	switch plan.Method {
	case domain.DepreciationNone:
		// Finanzanlagen werden nicht planmäßig abgeschrieben — die Maske zeigt
		// dann eine leere Tabelle und kein `null`.
		return []AfAYear{}, nil
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
			TaxAmount:        plan.Cost,
			Note:             "Sofortabzug im Jahr der Anschaffung (§ 6 Abs. 2 Satz 1 EStG).",
		}}, nil
	case domain.DepreciationPool:
		return poolSchedule(plan)
	case domain.DepreciationElectricVehicle:
		return electricVehicleSchedulePlan(plan, later, acquisitionYear)
	case domain.DepreciationBuildingLinear:
		return buildingSchedule(plan, later, acquisitionYear, start)
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
					"§ 7 Abs. 2 EStG hat sie nur für Anschaffungen in diesen Zeiträumen geöffnet: %s. "+
					"Außerhalb davon bleibt es bei der linearen Abschreibung",
				plan.AcquisitionDate, degressiveWindowList())
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

	// Belegt statt nil: der Plan geht als JSON in die Maske, und eine Eingabe
	// ohne Abschreibungsjahr käme dort sonst als `null` an.
	rows := make([]AfAYear, 0, 16)
	bookValue := plan.Cost
	// taxBookValue ist der Restwert der steuerlichen Rechnung. Er läuft neben
	// dem handelsrechtlichen her und weicht von ihm nur ab, wo § 7g Abs. 5 EStG
	// in Anspruch genommen wurde: die Sonderabschreibung mindert allein ihn.
	taxBookValue := plan.Cost
	remainingMonths := monthsBetween(afaStart, naturalEnd)
	switchedToLinear := false
	basisChanged := false
	var addedNote string

	// Die Sonderabschreibung des § 7g Abs. 5 EStG läuft neben dem Plan her: sie
	// ändert die planmäßige AfA nicht, sondern kommt hinzu — „neben den
	// Absetzungen für Abnutzung nach § 7 Absatz 1 oder Absatz 2". Erst mit dem
	// Ende des Begünstigungszeitraums greift § 7a Abs. 9 EStG und verteilt den
	// verbliebenen Restwert auf die Restnutzungsdauer.
	specialRemaining := domain.Cents(0)
	specialYears := plan.SpecialYears
	if plan.SpecialPermille > 0 {
		specialRemaining = domain.MulRound(plan.Cost, int64(plan.SpecialPermille), 1000)
		if specialYears <= 0 {
			specialYears = 1
		}
		if specialYears > SpecialPeriodYears {
			specialYears = SpecialPeriodYears
		}
	}
	specialLastYear := acquisitionYear + specialYears - 1
	periodEndYear := acquisitionYear + SpecialPeriodYears - 1
	restwertPhase := false

	for year := acquisitionYear; ; year++ {
		// Nachträgliche Anschaffungskosten und eine verlängerte Nutzungsdauer
		// wirken zu Beginn ihres Jahres — vor der Rechnung dieses Jahres, aber
		// ohne die bereits gebuchten davor anzurühren.
		addedNote = ""
		if added := later[year]; added != 0 {
			bookValue += added
			basisChanged = true
			if added > 0 {
				addedNote = fmt.Sprintf(
					"Nachträgliche Anschaffungskosten von %s € erhöhen die Bemessungsgrundlage; "+
						"der Restbuchwert verteilt sich ab hier auf die Restnutzungsdauer "+
						"(R 7.4 Abs. 9 EStR).", added)
			} else {
				addedNote = fmt.Sprintf(
					"Die Anschaffungskosten mindern sich um %s €; der Restbuchwert verteilt sich "+
						"ab hier auf die Restnutzungsdauer.", -added)
			}
		}
		if ext := laterLife[year]; ext > 0 {
			remainingMonths += ext
			naturalEnd = naturalEnd.AddDate(0, ext, 0)
			if !truncated {
				afaEnd = naturalEnd
			}
			basisChanged = true
			addedNote = appendNote(addedNote, fmt.Sprintf(
				"Die Nutzungsdauer verlängert sich um %d Monate.", ext))
		}
		if plan.SpecialPermille > 0 && year > periodEndYear && !restwertPhase {
			// § 7a Abs. 9 EStG: nach Ablauf des Begünstigungszeitraums bemisst
			// sich die weitere AfA nach dem Restwert und der Restnutzungsdauer.
			// Das betrifft allein die steuerliche Rechnung — handelsrechtlich
			// gab es die Sonderabschreibung nie, und der Plan läuft dort
			// unverändert mit seinem Satz weiter.
			restwertPhase = true
			addedNote = appendNote(addedNote,
				"Der Begünstigungszeitraum der Sonderabschreibung ist abgelaufen: der steuerliche "+
					"Restwert verteilt sich von hier an auf die Restnutzungsdauer (§ 7a Abs. 9 EStG). "+
					"Die handelsrechtliche Abschreibung bleibt davon unberührt.")
		}

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

		row := AfAYear{
			FiscalYear:       year,
			Months:           months,
			Method:           domain.DepreciationLinear,
			OpeningBookValue: bookValue,
			Note:             addedNote,
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
		case basisChanged:
			// Hat sich die Bemessungsgrundlage einmal geändert — durch eine
			// außerplanmäßige Abschreibung, nachträgliche Anschaffungskosten oder
			// eine verlängerte Nutzungsdauer —, verteilt sich der Restbuchwert auf
			// die Restnutzungsdauer. Weiter von den ursprünglichen
			// Anschaffungskosten zu rechnen träfe weder das Ende noch die Summe.
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

		// Die steuerliche Rechnung des Jahres. Bis zum Ende des
		// Begünstigungszeitraums ist sie die handelsrechtliche — die
		// Sonderabschreibung tritt neben die planmäßige AfA und ändert sie nicht
		// (§ 7a Abs. 4 EStG). Danach greift § 7a Abs. 9 EStG und verteilt den
		// steuerlichen Restwert auf die Restnutzungsdauer; weiter vom
		// ursprünglichen Satz zu rechnen ließe das Wirtschaftsgut steuerlich vor
		// dem Ende seiner Nutzungsdauer bei null ankommen.
		taxAmount := row.Amount
		if restwertPhase && remainingMonths > 0 {
			annual := domain.MulRound(taxBookValue, 12, int64(remainingMonths))
			taxAmount = domain.MulRound(annual, int64(months), 12)
		}
		if taxAmount > taxBookValue {
			taxAmount = taxBookValue
		}
		if taxAmount < 0 {
			taxAmount = 0
		}

		// Die Sonderabschreibung kommt oben drauf — und anders als die planmäßige
		// AfA wird sie im Anschaffungsjahr *nicht* zeitanteilig gekürzt: § 7 Abs. 1
		// Satz 4 EStG gilt für die Absetzung für Abnutzung, nicht für die
		// Sonderabschreibung. Eine im Dezember angeschaffte Maschine bekommt ihr
		// Fünftel voll.
		if specialRemaining > 0 && year >= acquisitionYear && year <= specialLastYear {
			share := domain.MulRound(
				domain.MulRound(plan.Cost, int64(plan.SpecialPermille), 1000), 1, int64(specialYears))
			if year == specialLastYear || share > specialRemaining {
				share = specialRemaining
			}
			// Der Deckel ist der steuerliche Restwert: mehr als die
			// Anschaffungskosten lässt sich auch steuerlich nicht abschreiben.
			if room := taxBookValue - taxAmount; share > room {
				share = room
			}
			if share > 0 {
				row.SpecialAmount = share
				specialRemaining -= share
				row.Note = appendNote(row.Note, fmt.Sprintf(
					"Zusätzlich Sonderabschreibung nach § 7g Abs. 5 EStG: %s €. Sie wird nicht gebucht: "+
						"sie mindert allein den steuerlichen Wertansatz, während die planmäßige AfA "+
						"handelsrechtlich unverändert weiterläuft (§ 7a Abs. 4 EStG, § 253 HGB).", share))
			}
		}

		row.TaxAmount = taxAmount
		bookValue -= row.Amount
		taxBookValue -= taxAmount + row.SpecialAmount
		if impair := plan.ImpairmentsByYear[year]; impair > 0 {
			bookValue -= impair
			if bookValue < 0 {
				bookValue = 0
			}
			// Die außerplanmäßige Abschreibung wirkt in beiden Rechnungen: sie
			// ist keine steuerliche Vergünstigung, sondern die dauernde
			// Wertminderung des § 253 Abs. 3 Satz 5 HGB.
			taxBookValue -= impair
			basisChanged = true
			row.Note = appendNote(row.Note, fmt.Sprintf(
				"Zusätzlich außerplanmäßig abgeschrieben: %s €.", impair))
		}
		if taxBookValue < 0 {
			taxBookValue = 0
		}
		row.ClosingBookValue = bookValue
		row.TaxClosingBookValue = taxBookValue
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
			FiscalYear:          first + i,
			Months:              12,
			Method:              domain.DepreciationPool,
			RateLabel:           fmt.Sprintf("1/%d", years),
			OpeningBookValue:    opening,
			Amount:              amount,
			ClosingBookValue:    bookValue,
			TaxAmount:           amount,
			TaxClosingBookValue: bookValue,
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

// electricVehicleSchedulePlan verteilt die Anschaffungskosten nach der Staffel
// des § 7 Abs. 2a EStG.
//
// Zwei Dinge unterscheiden sie von jeder anderen Methode. Die Sätze bemessen
// sich nach den Anschaffungskosten und nicht nach dem Restbuchwert — die Staffel
// endet deshalb nach sechs Jahren bei genau null und nicht asymptotisch. Und im
// Jahr der Anschaffung wird nicht zeitanteilig gekürzt: das Gesetz nennt die
// Sätze ohne den Vorbehalt des § 7 Abs. 1 Satz 4 EStG, ein im Dezember
// zugelassenes Fahrzeug bekommt seine 75 % voll.
func electricVehicleSchedulePlan(
	plan AfAPlan, later map[int]domain.Cents, acquisitionYear int,
) ([]AfAYear, error) {
	window, ok := ElectricVehicleWindowFor(plan.AcquisitionDate)
	if !ok {
		return nil, fmt.Errorf(
			"für eine Anschaffung am %s gibt es die Staffel des § 7 Abs. 2a EStG nicht. Sie gilt für "+
				"neue, rein elektrisch betriebene Fahrzeuge, die nach dem 30.06.2025 und vor dem "+
				"01.01.2028 angeschafft werden. Außerhalb davon bleibt es bei der linearen Abschreibung",
			germanDate(plan.AcquisitionDate))
	}

	disposalYear := 0
	if plan.DisposalDate != "" {
		disposalYear = domain.GetFiscalYearForDate(plan.DisposalDate, fiscalStart(plan))
	}

	rows := make([]AfAYear, 0, len(window.PermillePerYear))
	basis := plan.Cost
	bookValue := plan.Cost
	for i, permille := range window.PermillePerYear {
		year := acquisitionYear + i
		note := ""
		if added := later[year]; added != 0 {
			basis += added
			bookValue += added
			note = fmt.Sprintf(
				"Die Bemessungsgrundlage ändert sich um %s €; die Staffel läuft ab hier auf dem "+
					"neuen Wert weiter.", added)
		}
		if bookValue <= 0 {
			break
		}
		amount := domain.MulRound(basis, permille, 1000)
		// Das letzte Jahr der Staffel nimmt den Rest: sechs gerundete Sätze
		// treffen die Anschaffungskosten sonst um Cent-Beträge nicht.
		if i == len(window.PermillePerYear)-1 || amount > bookValue {
			amount = bookValue
		}
		if disposalYear > 0 && year > disposalYear {
			break
		}
		opening := bookValue
		bookValue -= amount
		if impair := plan.ImpairmentsByYear[year]; impair > 0 {
			bookValue -= impair
			if bookValue < 0 {
				bookValue = 0
			}
			note = appendNote(note, fmt.Sprintf("Zusätzlich außerplanmäßig abgeschrieben: %s €.", impair))
		}
		rows = append(rows, AfAYear{
			FiscalYear: year,
			// Zwölf Monate auch im Anschaffungsjahr: die Staffel kennt keinen
			// Zeitanteil, und eine kleinere Zahl hier läse sich als Kürzung.
			Months:              12,
			Method:              domain.DepreciationElectricVehicle,
			RateLabel:           permilleLabel(permille, 1000),
			OpeningBookValue:    opening,
			Amount:              amount,
			ClosingBookValue:    bookValue,
			TaxAmount:           amount,
			TaxClosingBookValue: bookValue,
			Note: appendNote(note, fmt.Sprintf(
				"Staffel für rein elektrisch betriebene Fahrzeuge (%s), Jahr %d von %d. Im Jahr der "+
					"Anschaffung wird nicht zeitanteilig gekürzt.",
				window.Source, i+1, len(window.PermillePerYear))),
		})
		if bookValue <= 0 {
			break
		}
	}
	return rows, nil
}

// buildingSchedule schreibt ein Gebäude mit dem festen Satz des § 7 Abs. 4 EStG
// ab.
//
// Der Satz gilt für die gesamte Nutzungsdauer und wird nicht auf einen Restwert
// umgestellt: 3 % der Anschaffungskosten, Jahr für Jahr, bis der Buchwert
// aufgebraucht ist. Nur das Anschaffungsjahr ist kürzer — für Gebäude gilt
// § 7 Abs. 1 Satz 4 EStG entsprechend (R 7.4 Abs. 2 Satz 1 EStR). Nachträgliche
// Herstellungskosten erhöhen die Bemessungsgrundlage, und derselbe Prozentsatz
// läuft auf ihr weiter.
func buildingSchedule(
	plan AfAPlan, later map[int]domain.Cents, acquisitionYear, start int,
) ([]AfAYear, error) {
	if plan.BuildingPermille <= 0 {
		return nil, fmt.Errorf(
			"für ein Gebäude fehlt der Satz des § 7 Abs. 4 EStG. Er hängt daran, ob das Gebäude " +
				"Wohnzwecken dient, und am Stichtag: Bauantrag beim Betriebsgebäude, Fertigstellung " +
				"beim Wohngebäude")
	}
	afaStart, err := monthStart(plan.AcquisitionDate)
	if err != nil {
		return nil, err
	}
	afaEnd := afaStart.AddDate(200, 0, 0) // weit genug: das Ende bestimmt der Buchwert
	truncated := false
	if plan.DisposalDate != "" {
		disposal, err := monthStart(plan.DisposalDate)
		if err != nil {
			return nil, err
		}
		afaEnd = disposal.AddDate(0, 1, 0)
		truncated = true
	}

	rows := make([]AfAYear, 0, 40)
	basis := plan.Cost
	bookValue := plan.Cost
	for year := acquisitionYear; bookValue > 0; year++ {
		note := ""
		if added := later[year]; added != 0 {
			basis += added
			bookValue += added
			note = fmt.Sprintf(
				"Nachträgliche Herstellungskosten von %s € erhöhen die Bemessungsgrundlage; der Satz "+
					"des § 7 Abs. 4 EStG läuft auf ihr weiter (R 7.4 Abs. 9 EStR).", added)
		}
		fyStart := time.Date(year, time.Month(start), 1, 0, 0, 0, 0, time.UTC)
		fyEnd := fyStart.AddDate(1, 0, 0)
		months := overlapMonths(afaStart, afaEnd, fyStart, fyEnd)
		if months <= 0 {
			if !fyStart.Before(afaEnd) {
				break
			}
			continue
		}

		annual := domain.MulRound(basis, plan.BuildingPermille, 1000)
		amount := domain.MulRound(annual, int64(months), 12)
		if amount > bookValue {
			amount = bookValue
			note = appendNote(note, "Letztes Jahr: der Restbuchwert wird vollständig abgeschrieben.")
		}
		if months < 12 {
			note = appendNote(note, fmt.Sprintf(
				"Zeitanteilig für %d von 12 Monaten (§ 7 Abs. 1 Satz 4 EStG, R 7.4 Abs. 2 EStR).", months))
		}
		opening := bookValue
		bookValue -= amount
		if impair := plan.ImpairmentsByYear[year]; impair > 0 {
			bookValue -= impair
			if bookValue < 0 {
				bookValue = 0
			}
			note = appendNote(note, fmt.Sprintf("Zusätzlich außerplanmäßig abgeschrieben: %s €.", impair))
		}
		rows = append(rows, AfAYear{
			FiscalYear:          year,
			Months:              months,
			Method:              domain.DepreciationBuildingLinear,
			RateLabel:           permilleLabel(plan.BuildingPermille, 1000),
			OpeningBookValue:    opening,
			Amount:              amount,
			ClosingBookValue:    bookValue,
			TaxAmount:           amount,
			TaxClosingBookValue: bookValue,
			Note:                note,
		})
		if truncated && !fyEnd.Before(afaEnd) {
			break
		}
	}
	return rows, nil
}

// fiscalStart liefert den Beginn des Geschäftsjahres eines Plans, notfalls den
// Januar.
func fiscalStart(plan AfAPlan) int {
	if plan.FiscalYearStartMonth <= 0 || plan.FiscalYearStartMonth > 12 {
		return 1
	}
	return plan.FiscalYearStartMonth
}

// degressiveRate returns the degressive percentage as a fraction: the multiple
// of the linear rate, capped by the statutory ceiling.
func degressiveRate(usefulLifeMonths int, window DegressiveWindow) (int64, int64) {
	// Der lineare Satz ist 12/Nutzungsdauer in Monaten; das Vielfache davon ist
	// FactorPermille * 12 / (Monate * 1000). Verglichen wird als Bruch, damit
	// kein Zwischenrunden den Deckel verschiebt.
	num := window.FactorPermille * 12
	den := int64(usefulLifeMonths) * 1000
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

// ScheduleSpecialFor returns the Sonderabschreibung of one fiscal year. Sie ist
// bewusst getrennt abzufragen: sie gehört auf ein eigenes Konto, und wer beide
// Beträge addiert bekäme, könnte sie dort nicht mehr auseinanderhalten.
func ScheduleSpecialFor(rows []AfAYear, fiscalYear int) domain.Cents {
	for _, r := range rows {
		if r.FiscalYear == fiscalYear {
			return r.SpecialAmount
		}
	}
	return 0
}

// checkSpecialDepreciation holds the two limits of § 7g Abs. 5 EStG that a plan
// can be refused for before anything is computed.
//
// Die dritte Voraussetzung — die Gewinngrenze des Vorjahres und die fast
// ausschließlich betriebliche Nutzung (§ 7g Abs. 6 EStG) — steht nicht hier:
// sie ist keine Rechnung, sondern ein Sachverhalt, den nur der Steuerpflichtige
// kennt. Er wird am Anlagegut festgehalten, nicht geraten.
func checkSpecialDepreciation(plan AfAPlan) error {
	if plan.SpecialPermille <= 0 {
		return nil
	}
	if plan.SpecialPermille > SpecialMaxPermille {
		return fmt.Errorf(
			"die Sonderabschreibung nach § 7g Abs. 5 EStG beträgt höchstens 40 %% der "+
				"Anschaffungskosten; %s sind zu viel", permilleLabel(int64(plan.SpecialPermille), 1000))
	}
	if plan.Method != domain.DepreciationLinear && plan.Method != domain.DepreciationDegressive {
		// § 7g Abs. 5 EStG lässt die Sonderabschreibung „neben den Absetzungen für
		// Abnutzung nach § 7 Absatz 1 oder Absatz 2" zu — also neben der linearen
		// wie neben der degressiven AfA. Der Sammelposten und der Sofortabzug
		// kennen dagegen keine Absetzung für Abnutzung, neben die etwas treten
		// könnte.
		return fmt.Errorf(
			"die Sonderabschreibung tritt neben die Absetzung für Abnutzung nach § 7 Abs. 1 oder "+
				"Abs. 2 EStG. Mit der Methode %q lässt sie sich nicht verbinden", plan.Method)
	}
	if plan.SpecialYears > SpecialPeriodYears {
		return fmt.Errorf(
			"der Begünstigungszeitraum umfasst das Jahr der Anschaffung und die vier folgenden "+
				"(§ 7g Abs. 5 EStG); auf %d Jahre lässt sich die Sonderabschreibung nicht verteilen",
			plan.SpecialYears)
	}
	return nil
}
