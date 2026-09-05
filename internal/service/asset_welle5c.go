package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Die Anlagenthemen der Welle 5c: die Sperren, die am Anlagekonto hängen, der
// Wertaufholungsbericht, die Einheitlichkeit des Sammelposten-Wahlrechts und die
// anschaffungsnahen Herstellungskosten.

// validateMethodForAccount hält die Abschreibungsverfahren an den Merkmalen des
// Anlagekontos fest.
//
// Ob ein Wirtschaftsgut beweglich ist, steht nicht am Anlagegut, sondern im
// Kontenkatalog — und daran hängen drei Regeln, die bisher nur zum Teil
// durchgesetzt waren: die degressive AfA des § 7 Abs. 2 EStG gilt für bewegliche
// Wirtschaftsgüter, die festen Sätze des § 7 Abs. 4 EStG für Gebäude, und die
// Staffel des § 7 Abs. 2a EStG für neue Elektrofahrzeuge. Ohne diese Prüfung
// ließ sich ein Bürogebäude degressiv abschreiben, und der Plan sah plausibel
// aus.
func validateMethodForAccount(asset *domain.FixedAsset) error {
	entry, known := accounting.LookupAssetAccount(asset.Account)

	switch asset.Method {
	case domain.DepreciationDegressive:
		if known && entry.Immovable {
			return fmt.Errorf(
				"%s (%s) trägt ein unbewegliches Wirtschaftsgut. Die degressive Abschreibung des "+
					"§ 7 Abs. 2 EStG gibt es nur für bewegliche Wirtschaftsgüter des Anlagevermögens — "+
					"für Gebäude gelten die festen Sätze des § 7 Abs. 4 EStG",
				asset.Account, entry.Name)
		}
		if asset.Class == domain.AssetClassIntangible {
			return fmt.Errorf(
				"die degressive Abschreibung des § 7 Abs. 2 EStG gilt für bewegliche Wirtschaftsgüter " +
					"des Sachanlagevermögens; ein immaterieller Vermögensgegenstand ist keines")
		}
	case domain.DepreciationBuildingLinear:
		if !known {
			return fmt.Errorf(
				"die festen Sätze des § 7 Abs. 4 EStG gelten für Gebäude. Das Konto %s steht nicht im "+
					"Katalog der Anlagekonten — wähle eines der Gebäudekonten", asset.Account)
		}
		if !entry.Immovable {
			return fmt.Errorf(
				"%s (%s) trägt kein Gebäude. Die festen Sätze des § 7 Abs. 4 EStG gelten für Gebäude; "+
					"alles andere wird linear über seine betriebsgewöhnliche Nutzungsdauer abgeschrieben",
				asset.Account, entry.Name)
		}
		if _, err := accounting.BuildingRateFor(
			entry.Residential, asset.BuildingReferenceDate); err != nil {
			return err
		}
	case domain.DepreciationElectricVehicle:
		if known && entry.Group != "Fahrzeuge" {
			return fmt.Errorf(
				"%s (%s) trägt kein Fahrzeug. § 7 Abs. 2a EStG gilt für neue, rein elektrisch "+
					"betriebene Fahrzeuge — buche sie auf ein Fahrzeugkonto",
				asset.Account, entry.Name)
		}
		if _, ok := accounting.ElectricVehicleWindowFor(asset.DepreciationStart()); !ok {
			return fmt.Errorf(
				"für eine Anschaffung am %s gibt es die Staffel des § 7 Abs. 2a EStG nicht. Sie gilt "+
					"für Anschaffungen nach dem 30.06.2025 und vor dem 01.01.2028",
				asset.DepreciationStart())
		}
	}
	return nil
}

// -------------------------------------------------------------------------
// Einheitlichkeit des Wahlrechts (§ 6 Abs. 2a Satz 5 EStG)
// -------------------------------------------------------------------------

// PoolConsistencyRow ist ein Zugang, der zur Frage der Einheitlichkeit gehört.
type PoolConsistencyRow struct {
	AssetID         uint                      `json:"assetId"`
	InventoryNumber string                    `json:"inventoryNumber"`
	Name            string                    `json:"name"`
	AcquisitionDate string                    `json:"acquisitionDate"`
	Cost            domain.Cents              `json:"cost"`
	Method          domain.DepreciationMethod `json:"method"`
}

// PoolConsistencyReport beantwortet für ein Wirtschaftsjahr, ob das Wahlrecht
// des § 6 Abs. 2a EStG einheitlich ausgeübt wurde.
type PoolConsistencyReport struct {
	FiscalYear int `json:"fiscalYear"`
	// LowerLimit und UpperLimit sind die Grenzen, zwischen denen sich die Frage
	// überhaupt stellt.
	LowerLimit domain.Cents `json:"lowerLimit"`
	UpperLimit domain.Cents `json:"upperLimit"`
	// Pooled und Immediate sind die Zugänge des Jahres in diesem Wertbereich.
	Pooled    []PoolConsistencyRow `json:"pooled"`
	Immediate []PoolConsistencyRow `json:"immediate"`
	// Consistent sagt, ob das Wahlrecht einheitlich ausgeübt ist.
	Consistent bool   `json:"consistent"`
	Note       string `json:"note"`
}

// EnsureLists ersetzt nicht belegte Listen durch leere.
func (r *PoolConsistencyReport) EnsureLists() {
	if r.Pooled == nil {
		r.Pooled = make([]PoolConsistencyRow, 0)
	}
	if r.Immediate == nil {
		r.Immediate = make([]PoolConsistencyRow, 0)
	}
}

// PoolConsistency prüft die Einheitlichkeit des Wahlrechts eines Jahres.
//
// § 6 Abs. 2a Satz 5 EStG: „Die Wahlrechte nach den Sätzen 1 und 4 sind für alle
// in einem Wirtschaftsjahr angeschafften … Wirtschaftsgüter einheitlich
// auszuüben." Wer im Mai poolt, darf im Oktober kein gleichartiges Gut mehr
// sofort abziehen — und umgekehrt. Der Wertbereich, in dem sich die Frage
// stellt, liegt zwischen der Sammelposten-Untergrenze und der GWG-Grenze; unter
// ihr gibt es keinen Sammelposten, über ihr keinen Sofortabzug.
func (s *AssetService) PoolConsistency(ctx context.Context, year int) (*PoolConsistencyReport, error) {
	if year == 0 {
		year = s.fiscalYear
	}
	out := &PoolConsistencyReport{FiscalYear: year, Consistent: true}
	out.EnsureLists()

	startMonth := s.fiscalYearStartMonth(ctx)
	assets, err := s.assetRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	for i := range assets {
		a := &assets[i]
		if domain.GetFiscalYearForDate(a.AcquisitionDate, startMonth) != year {
			continue
		}
		params, err := accounting.AfAParametersFor(a.AcquisitionDate)
		if err != nil {
			continue
		}
		out.LowerLimit, out.UpperLimit = params.PoolLowerLimit, params.GWGImmediateLimit
		if a.AcquisitionCost <= params.PoolLowerLimit || a.AcquisitionCost > params.GWGImmediateLimit {
			continue
		}
		row := PoolConsistencyRow{
			AssetID: a.ID, InventoryNumber: a.InventoryNumber, Name: a.Name,
			AcquisitionDate: a.AcquisitionDate, Cost: a.AcquisitionCost, Method: a.Method,
		}
		switch a.Method {
		case domain.DepreciationPool:
			out.Pooled = append(out.Pooled, row)
		case domain.DepreciationImmediate:
			out.Immediate = append(out.Immediate, row)
		}
	}

	out.Consistent = len(out.Pooled) == 0 || len(out.Immediate) == 0
	if out.Consistent {
		out.Note = fmt.Sprintf(
			"Das Wahlrecht des § 6 Abs. 2a EStG ist im Wirtschaftsjahr %d einheitlich ausgeübt. "+
				"Es betrifft die Zugänge über %s € und bis %s € netto.",
			year, out.LowerLimit, out.UpperLimit)
		return out, nil
	}
	out.Note = fmt.Sprintf(
		"Im Wirtschaftsjahr %d stehen %d Wirtschaftsgüter im Sammelposten und %d im Sofortabzug. "+
			"§ 6 Abs. 2a Satz 5 EStG verlangt die einheitliche Ausübung für alle Wirtschaftsgüter "+
			"eines Wirtschaftsjahres zwischen %s € und %s € netto — eines der beiden ist zu ändern.",
		year, len(out.Pooled), len(out.Immediate), out.LowerLimit, out.UpperLimit)
	return out, nil
}

// checkPoolConsistency weist einen Zugang zurück, der das Wahlrecht des Jahres
// brechen würde.
func (s *AssetService) checkPoolConsistency(ctx context.Context, asset *domain.FixedAsset) error {
	if asset.Method != domain.DepreciationPool && asset.Method != domain.DepreciationImmediate {
		return nil
	}
	params, err := accounting.AfAParametersFor(asset.AcquisitionDate)
	if err != nil {
		return err
	}
	if asset.AcquisitionCost <= params.PoolLowerLimit ||
		asset.AcquisitionCost > params.GWGImmediateLimit {
		// Außerhalb des Wertbereichs gibt es kein Wahlrecht, das einheitlich
		// auszuüben wäre: unter der Untergrenze kennt § 6 Abs. 2a EStG keinen
		// Sammelposten, über der GWG-Grenze § 6 Abs. 2 EStG keinen Sofortabzug.
		return nil
	}

	year := domain.GetFiscalYearForDate(asset.AcquisitionDate, s.fiscalYearStartMonth(ctx))
	report, err := s.PoolConsistency(ctx, year)
	if err != nil {
		return err
	}
	conflicting := report.Immediate
	other := "Sofortabzug"
	if asset.Method == domain.DepreciationImmediate {
		conflicting, other = report.Pooled, "Sammelposten"
	}
	for _, row := range conflicting {
		if row.AssetID == asset.ID {
			continue
		}
		return fmt.Errorf(
			"im Wirtschaftsjahr %d steht %s (%s) bereits im %s. § 6 Abs. 2a Satz 5 EStG verlangt, "+
				"das Wahlrecht für alle Wirtschaftsgüter eines Wirtschaftsjahres zwischen %s € und "+
				"%s € netto einheitlich auszuüben — entweder alle in den Sammelposten oder alle in "+
				"den Sofortabzug",
			year, row.Name, row.InventoryNumber, other, report.LowerLimit, report.UpperLimit)
	}
	return nil
}

// -------------------------------------------------------------------------
// Wertaufholung (§ 253 Abs. 5 Satz 1 HGB)
// -------------------------------------------------------------------------

// WriteUpCandidate ist ein Anlagegut mit außerplanmäßiger Abschreibung, für das
// sich die Frage der Zuschreibung stellt.
type WriteUpCandidate struct {
	AssetID         uint         `json:"assetId"`
	InventoryNumber string       `json:"inventoryNumber"`
	Name            string       `json:"name"`
	Account         string       `json:"account"`
	BookValue       domain.Cents `json:"bookValue"`
	// ContinuedCost ist der Buchwert, den das Anlagegut ohne die
	// außerplanmäßige Abschreibung heute hätte — die Obergrenze der Zuschreibung.
	ContinuedCost domain.Cents `json:"continuedCost"`
	// MaxWriteUp ist der Spielraum: fortgeführte Anschaffungskosten minus
	// Buchwert.
	MaxWriteUp domain.Cents `json:"maxWriteUp"`
	// Impairments sind die außerplanmäßigen Abschreibungen mit ihren Gründen.
	Impairments []WriteUpImpairment `json:"impairments"`
	// Confirmed sagt, ob für dieses Geschäftsjahr bereits bestätigt wurde, dass
	// der Grund fortbesteht.
	Confirmed     bool   `json:"confirmed"`
	ConfirmedNote string `json:"confirmedNote,omitempty"`
	Note          string `json:"note"`
}

// WriteUpImpairment ist eine frühere außerplanmäßige Abschreibung.
type WriteUpImpairment struct {
	FiscalYear int          `json:"fiscalYear"`
	Date       string       `json:"date"`
	Amount     domain.Cents `json:"amount"`
	Reason     string       `json:"reason"`
}

// WriteUpReport ist der Bericht eines Geschäftsjahres.
type WriteUpReport struct {
	FiscalYear int                `json:"fiscalYear"`
	Candidates []WriteUpCandidate `json:"candidates"`
	// Open zählt die Anlagegüter, für die weder zugeschrieben noch bestätigt
	// wurde.
	Open int    `json:"open"`
	Note string `json:"note"`
}

// EnsureLists ersetzt nicht belegte Listen durch leere.
func (r *WriteUpReport) EnsureLists() {
	if r.Candidates == nil {
		r.Candidates = make([]WriteUpCandidate, 0)
	}
}

// WriteUpReport listet die Anlagegüter, bei denen zu prüfen ist, ob der Grund
// der außerplanmäßigen Abschreibung weggefallen ist.
//
// Die Zuschreibung ist ein Gebot und kein Wahlrecht (§ 253 Abs. 5 Satz 1 HGB):
// fällt der Grund weg, ist zuzuschreiben — höchstens bis zu den fortgeführten
// Anschaffungskosten. Ob er weggefallen ist, weiß nur der Bilanzierende. Der
// Bericht stellt deshalb die Frage und rechnet den Spielraum; die Antwort gibt
// ein Mensch, und sie wird festgehalten, damit sie im nächsten Jahr erneut
// gestellt werden kann und nicht in diesem zweimal.
func (s *AssetService) WriteUpReport(ctx context.Context, year int) (*WriteUpReport, error) {
	if year == 0 {
		year = s.fiscalYear
	}
	out := &WriteUpReport{FiscalYear: year}
	out.EnsureLists()

	startMonth := s.fiscalYearStartMonth(ctx)
	assets, err := s.assetRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	for i := range assets {
		asset := &assets[i]
		if asset.IsDisposed() && disposedBefore(asset, year) {
			continue
		}
		impairments := make([]WriteUpImpairment, 0, 2)
		for _, m := range asset.Movements {
			if m.Kind == domain.AssetMovementImpairment && m.FiscalYear <= year {
				impairments = append(impairments, WriteUpImpairment{
					FiscalYear: m.FiscalYear, Date: m.Date,
					Amount: m.DepreciationAmount, Reason: m.Note,
				})
			}
		}
		if len(impairments) == 0 {
			continue
		}
		if asset.Account == accounting.GoodwillAccount {
			// § 253 Abs. 5 Satz 2 HGB verbietet die Zuschreibung auf den
			// Geschäfts- oder Firmenwert. Ihn in eine Liste zu stellen, aus der
			// nichts folgen darf, wäre eine Frage ohne mögliche Antwort.
			continue
		}

		s.enrich(asset, year, startMonth)
		ceiling, err := s.writeUpCeiling(ctx, asset, year, startMonth)
		if err != nil {
			return nil, err
		}
		if ceiling <= 0 {
			continue
		}

		candidate := WriteUpCandidate{
			AssetID: asset.ID, InventoryNumber: asset.InventoryNumber, Name: asset.Name,
			Account: asset.Account, BookValue: asset.BookValue,
			ContinuedCost: asset.BookValue + ceiling, MaxWriteUp: ceiling,
			Impairments:   impairments,
			Confirmed:     asset.ImpairmentPersistsYear == year,
			ConfirmedNote: asset.ImpairmentPersistsNote,
		}
		if candidate.Confirmed {
			candidate.Note = fmt.Sprintf(
				"Für %d ist bestätigt, dass der Grund der außerplanmäßigen Abschreibung fortbesteht: %s",
				year, asset.ImpairmentPersistsNote)
		} else {
			out.Open++
			candidate.Note = fmt.Sprintf(
				"Bis zu %s € könnten zugeschrieben werden. Prüfe, ob der Grund der außerplanmäßigen "+
					"Abschreibung weggefallen ist — dann ist die Zuschreibung nach § 253 Abs. 5 Satz 1 "+
					"HGB geboten und kein Wahlrecht.", ceiling)
		}
		out.Candidates = append(out.Candidates, candidate)
	}

	sort.SliceStable(out.Candidates, func(i, j int) bool {
		if out.Candidates[i].Confirmed != out.Candidates[j].Confirmed {
			return !out.Candidates[i].Confirmed
		}
		return out.Candidates[i].InventoryNumber < out.Candidates[j].InventoryNumber
	})
	out.Note = fmt.Sprintf(
		"%d von %d Anlagegütern mit außerplanmäßiger Abschreibung sind für %d noch nicht beantwortet. "+
			"Entweder wird zugeschrieben oder festgehalten, dass der Grund fortbesteht.",
		out.Open, len(out.Candidates), year)
	return out, nil
}

// ConfirmImpairmentPersists hält fest, dass der Grund einer außerplanmäßigen
// Abschreibung im Geschäftsjahr fortbesteht.
func (s *AssetService) ConfirmImpairmentPersists(
	ctx context.Context, assetID uint, year int, note string,
) (*WriteUpReport, error) {
	if strings.TrimSpace(note) == "" {
		return nil, fmt.Errorf(
			"halte fest, warum der Grund der außerplanmäßigen Abschreibung fortbesteht. Die " +
				"Zuschreibung ist ein Gebot; sie zu unterlassen ist eine Aussage und braucht eine " +
				"Begründung")
	}
	if year == 0 {
		year = s.fiscalYear
	}
	asset, err := s.assetRepo.FindByID(ctx, assetID)
	if err != nil {
		return nil, fmt.Errorf("Anlagegut %d wurde nicht gefunden: %w", assetID, err)
	}
	asset.ImpairmentPersistsYear = year
	asset.ImpairmentPersistsNote = strings.TrimSpace(note)
	if err := s.assetRepo.Save(ctx, asset); err != nil {
		return nil, fmt.Errorf("die Bestätigung ließ sich nicht speichern: %w", err)
	}
	s.audit(ctx, domain.AuditActionUpdate, asset.ID, fmt.Sprintf(
		"Wertaufholung %d geprüft: der Grund der außerplanmäßigen Abschreibung auf %s besteht fort — %s",
		year, asset.InventoryNumber, note))
	return s.WriteUpReport(ctx, year)
}

// -------------------------------------------------------------------------
// Anschaffungsnahe Herstellungskosten (§ 6 Abs. 1 Nr. 1a EStG)
// -------------------------------------------------------------------------

// NearAcquisitionCheck ist die Prüfung des 15-%-Rahmens eines Gebäudes.
type NearAcquisitionCheck struct {
	Applicable bool `json:"applicable"`
	// PeriodEnd ist der letzte Tag des Dreijahreszeitraums.
	PeriodEnd string `json:"periodEnd,omitempty"`
	// Limit ist der Rahmen: 15 % der Gebäude-Anschaffungskosten netto.
	Limit domain.Cents `json:"limit"`
	// Spent ist der bisher angefallene Instandsetzungs- und
	// Modernisierungsaufwand, Planned der des aktuellen Vorgangs.
	Spent   domain.Cents `json:"spent"`
	Planned domain.Cents `json:"planned"`
	// Exceeded sagt, ob der Rahmen mit diesem Vorgang gerissen wird.
	Exceeded bool   `json:"exceeded"`
	Note     string `json:"note"`
}

// CheckNearAcquisitionCost prüft, ob eine Instandsetzung den 15-%-Rahmen des
// § 6 Abs. 1 Nr. 1a EStG sprengt.
//
// Aufwendungen für Instandsetzung und Modernisierung, die innerhalb von drei
// Jahren nach der Anschaffung eines Gebäudes anfallen, gehören zu den
// Herstellungskosten, wenn sie ohne Umsatzsteuer 15 % der
// Gebäude-Anschaffungskosten übersteigen. Dann sind sie nicht sofort abziehbar,
// sondern zu aktivieren — und der Unterschied zwischen beidem ist bei einem
// Gebäude die Steuerlast eines ganzen Jahres.
//
// Buchfink warnt und bucht nicht um: welche Maßnahme unter die Vorschrift fällt
// und welche als Erhaltungsaufwand danebensteht, ist eine Beurteilung. Der
// Vorschlag steht daneben, und ausgeführt wird er über RecordCostAdjustment.
func (s *AssetService) CheckNearAcquisitionCost(
	ctx context.Context, assetID uint, date string, amount domain.Cents,
) (*NearAcquisitionCheck, error) {
	out := &NearAcquisitionCheck{Planned: amount}
	asset, err := s.assetRepo.FindByID(ctx, assetID)
	if err != nil {
		return nil, fmt.Errorf("Anlagegut %d wurde nicht gefunden: %w", assetID, err)
	}
	entry, known := accounting.LookupAssetAccount(asset.Account)
	if !known || !entry.Immovable {
		out.Note = "§ 6 Abs. 1 Nr. 1a EStG gilt für Gebäude; dieses Anlagegut ist keines."
		return out, nil
	}
	params, err := accounting.TaxParametersFor(asset.AcquisitionDate)
	if err != nil {
		return nil, err
	}
	periodEnd, err := yearsAfter(asset.AcquisitionDate, params.NearAcquisitionYears)
	if err != nil {
		return nil, err
	}
	out.PeriodEnd = periodEnd
	if date > periodEnd {
		out.Note = fmt.Sprintf(
			"Der Dreijahreszeitraum des § 6 Abs. 1 Nr. 1a EStG endete am %s. Danach ist "+
				"Instandsetzungsaufwand sofort abziehbar, gleich wie hoch er ist.", germanDay(periodEnd))
		return out, nil
	}

	out.Applicable = true
	out.Limit = domain.MulRound(asset.AcquisitionCost, params.NearAcquisitionPermille, 1000)
	for _, m := range asset.Movements {
		if m.Kind != domain.AssetMovementMaintenance {
			continue
		}
		if m.Date < asset.AcquisitionDate || m.Date > periodEnd {
			continue
		}
		if !m.IsModernisation {
			// Satz 2 nimmt Erweiterungen und die jährlich üblicherweise
			// anfallenden Erhaltungsarbeiten aus. Was der Anwender so
			// gekennzeichnet hat, zählt nicht gegen den Rahmen.
			continue
		}
		out.Spent += m.ExpenseAmount
	}

	total := out.Spent + amount
	out.Exceeded = total > out.Limit
	if !out.Exceeded {
		out.Note = fmt.Sprintf(
			"Innerhalb von drei Jahren nach der Anschaffung sind bisher %s € Instandsetzungs- und "+
				"Modernisierungsaufwand angefallen; mit diesem Vorgang sind es %s €. Der Rahmen des "+
				"§ 6 Abs. 1 Nr. 1a EStG liegt bei %s € (15 %% der Anschaffungskosten von %s € netto) "+
				"und ist bis zum %s zu beachten.",
			out.Spent, total, out.Limit, asset.AcquisitionCost, germanDay(periodEnd))
		return out, nil
	}
	out.Note = fmt.Sprintf(
		"Mit diesem Vorgang übersteigt der Instandsetzungs- und Modernisierungsaufwand der ersten "+
			"drei Jahre den Rahmen des § 6 Abs. 1 Nr. 1a EStG: %s € gegenüber %s € (15 %% der "+
			"Anschaffungskosten von %s € netto). Damit gehören sämtliche dieser Aufwendungen zu den "+
			"Herstellungskosten des Gebäudes und sind nicht sofort abziehbar. Die bisher als Aufwand "+
			"gebuchten Beträge sind zurückzunehmen und als nachträgliche Herstellungskosten zu "+
			"aktivieren.",
		total, out.Limit, asset.AcquisitionCost)
	return out, nil
}

// statedPermille macht aus dem Anteil eines Anlageguts eine Angabe oder keine.
//
// Am Anlagegut ist die Null der unausgefüllte Wert: die Erfassungsmaske fragt
// den Anteil nicht ab, und ein leeres Feld heißt dort „nicht angegeben" und
// nicht „null Promille". Das Verzeichnis unterscheidet beides, deshalb wird
// hier übersetzt statt durchgereicht.
func statedPermille(permille int) *int {
	if permille <= 0 {
		return nil
	}
	return &permille
}

// yearsAfter liefert den letzten Tag des Zeitraums von n Jahren ab einem Datum.
func yearsAfter(date string, years int) (string, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return "", fmt.Errorf("%q ist kein Datum im Format JJJJ-MM-TT", date)
	}
	return t.AddDate(years, 0, 0).AddDate(0, 0, -1).Format("2006-01-02"), nil
}

// checkUsefulLifeReason verlangt eine Begründung, wo die Nutzungsdauer vom
// Vorschlag des BMF-Schreibens vom 22.02.2022 abweicht.
//
// Nur dort und nicht bei jedem Konto mit einem Vorschlag: das Schreiben zur
// einjährigen Nutzungsdauer digitaler Wirtschaftsgüter ist die eine Stelle, an
// der Buchfink einen Wert vorschlägt, der von den AfA-Tabellen erheblich
// abweicht und ein Wahlrecht ist. Für einen Pkw oder Büromöbel steht in den
// Tabellen ein Erfahrungswert, kein Wahlrecht — dort eine Begründung zu
// verlangen ginge über die Vorschrift hinaus und sperrte bestehende Anlagen beim
// nächsten Speichern.
//
// Die AfA-Tabellen und das BMF-Schreiben binden ohnehin die Finanzverwaltung und
// nicht den Steuerpflichtigen. Eine begründete abweichende Nutzungsdauer ist
// zulässig — „begründet" heißt, dass die Begründung existiert. Ohne sie ist die
// Abweichung im Zweifel kein Wahlrecht, sondern ein Tippfehler, und der fällt
// erst in der Betriebsprüfung auf.
func (s *AssetService) checkUsefulLifeReason(
	ctx context.Context, asset *domain.FixedAsset,
) error {
	if asset.Method != domain.DepreciationLinear && asset.Method != domain.DepreciationDegressive {
		return nil
	}
	// Bei einem bestehenden Anlagegut greift die Pflicht nur, wo sich die
	// Nutzungsdauer ändert. Sonst hielte jede Stammdatenänderung — ein
	// berichtigter Name, ein neuer Standort — den vor dieser Welle mit 36 Monaten
	// angelegten Server an, bis jemand eine Begründung nachträgt. Die Begründung
	// gehört zur Entscheidung über die Nutzungsdauer und nicht zum Speichern.
	if asset.ID != 0 {
		existing, err := s.assetRepo.FindByID(ctx, asset.ID)
		// Der Lesefehler bleibt hier unbeantwortet: die Speicherroutine lädt den
		// Bestand gleich darauf selbst und meldet ihn mit dem Text, der die
		// Kennung nennt. Zwei Meldungen für einen Fehler wären eine zu viel.
		if err == nil && existing != nil && existing.UsefulLifeMonths == asset.UsefulLifeMonths {
			return nil
		}
	}
	entry, known := accounting.LookupAssetAccount(asset.Account)
	if !known || entry.DefaultUsefulLifeMonths <= 0 {
		return nil
	}
	// Dieselbe Kennzeichnung, die der Kontenkatalog an die Maske gibt: sie soll
	// das Feld genau dort anbieten, wo es hier verlangt wird.
	if !entry.UsefulLifeReasonRequired {
		return nil
	}
	if asset.UsefulLifeMonths == entry.DefaultUsefulLifeMonths {
		return nil
	}
	if strings.TrimSpace(asset.UsefulLifeReason) != "" {
		return nil
	}
	return fmt.Errorf(
		"für %s (%s) schlägt Buchfink %d Monate vor (%s). Der Vorschlag gilt der Computerhardware "+
			"und der Software dieses Schreibens; das Konto trägt auch anderes, und dann ist genau "+
			"das die Begründung. Du hast %d Monate eingetragen — halte fest, worauf die abweichende "+
			"Nutzungsdauer beruht. Die Tabellen binden die Finanzverwaltung und nicht dich, aber "+
			"eine Abweichung ohne Begründung hält keiner Prüfung stand",
		asset.Account, entry.Name, entry.DefaultUsefulLifeMonths, entry.UsefulLifeSource,
		asset.UsefulLifeMonths)
}

// -------------------------------------------------------------------------
// Aufnahme ins Verzeichnis nach § 15a UStG bei der Aktivierung
// -------------------------------------------------------------------------

// inputTaxRegistrar ist der Ausschnitt des Verzeichnisses nach § 15a UStG, den
// die Anlagenbuchhaltung braucht.
type inputTaxRegistrar interface {
	Register(ctx context.Context, req RegisterInputTaxRequest) (*domain.InputTaxCorrection, error)
}

// SetInputTaxRegister koppelt das Verzeichnis nach § 15a UStG an die
// Anlagenbuchhaltung.
//
// Ohne diese Kopplung landete ein Pkw mit 7.600 € Vorsteuer nur dann im
// Verzeichnis, wenn der Anwender ihn von Hand nachtrug — und ein Verzeichnis,
// das nur füllt, wer daran denkt, ist nach § 22 Abs. 4 UStG keines.
func (s *AssetService) SetInputTaxRegister(r inputTaxRegistrar) { s.inputTax = r }

// registerInputTaxCorrection nimmt ein aktiviertes Wirtschaftsgut ins
// Verzeichnis nach § 15a UStG auf.
//
// Die Schwelle ist die des § 44 Abs. 1 UStDV: ab 1.000 € Vorsteuer je
// Wirtschaftsgut. Grundstücke und Gebäude kommen unabhängig davon hinein — bei
// ihnen läuft der Berichtigungszeitraum zehn Jahre, und wer nach zehn Jahren
// nach dem ursprünglichen Verwendungsanteil gefragt wird, findet ihn nur dort.
//
// Ein Fehler beim Eintrag reißt die Aktivierung nicht mit: das Anlagegut steht
// dann in der Kartei und fehlt im Verzeichnis, und das ist der bessere der
// beiden Zustände — die Meldung sagt, was nachzutragen ist.
func (s *AssetService) registerInputTaxCorrection(ctx context.Context, asset *domain.FixedAsset) error {
	if s.inputTax == nil || asset == nil {
		return nil
	}
	// Ohne gezogene Vorsteuer gibt es nichts zu berichtigen — beim Grundstück so
	// wenig wie beim Schreibtisch. § 15a UStG berichtigt den Abzug, den es
	// gegeben hat, und das Verzeichnis weist einen Eintrag über null Euro
	// Vorsteuer folgerichtig ab. Der Regelfall wäre sonst betroffen: der
	// Grundstückserwerb ist nach § 4 Nr. 9 Buchst. a UStG steuerfrei, ohne Option
	// nach § 9 UStG steht in der Rechnung keine Steuer, und ebenso kommt ein
	// Gebäude aus dem Saldenvortrag ohne Vorsteuer in die Kartei. Die Aktivierung
	// ist zu diesem Zeitpunkt schon gespeichert; ein Fehler hier ließe den
	// Anwender den Vorgang wiederholen und legte das Anlagegut ein zweites Mal
	// an. Die Zehnjahresregel gilt dem Grundstück *mit* Vorsteuer.
	if asset.InputTaxAmount <= 0 {
		return nil
	}
	entry, known := accounting.LookupAssetAccount(asset.Account)
	immovable := known && entry.Immovable
	params, err := accounting.TaxParametersFor(asset.AcquisitionDate)
	if err != nil {
		return err
	}
	if asset.InputTaxAmount <= params.InputTaxCorrectionFloor && !immovable {
		return nil
	}
	label := asset.Name
	if asset.InventoryNumber != "" {
		label = fmt.Sprintf("%s (%s)", asset.Name, asset.InventoryNumber)
	}
	_, err = s.inputTax.Register(ctx, RegisterInputTaxRequest{
		AssetID:          asset.ID,
		Label:            label,
		Account:          asset.Account,
		AcquisitionDate:  asset.AcquisitionDate,
		NetAmount:        asset.AcquisitionCost,
		InputTaxAmount:   asset.InputTaxAmount,
		OriginalPermille: statedPermille(asset.InputTaxPermille),
		Immovable:        immovable,
		Note:             "Bei der Aktivierung aufgenommen",
	})
	if err != nil {
		return fmt.Errorf(
			"%s wurde aktiviert, der Eintrag im Verzeichnis nach § 15a UStG aber nicht angelegt. "+
				"Trage ihn nach — ohne ihn fehlt der ursprüngliche Verwendungsanteil, und der ist "+
				"später aus keiner Buchung mehr zu gewinnen: %w", asset.InventoryNumber, err)
	}
	return nil
}

// CapitalizeNearAcquisitionCostRequest aktiviert den Erhaltungsaufwand der
// ersten drei Jahre als nachträgliche Herstellungskosten.
type CapitalizeNearAcquisitionCostRequest struct {
	AssetID uint `json:"assetId"`
	// Date ist der Tag der Umbuchung. Leer heißt: das Ende des
	// Dreijahreszeitraums.
	Date string `json:"date,omitempty"`
	// Reason ist Pflicht. Die Umbuchung nimmt einen sofort abgezogenen Aufwand
	// zurück und verteilt ihn über die Restnutzungsdauer des Gebäudes — wer das
	// später prüft, muss lesen können, worauf die Entscheidung beruhte.
	Reason string `json:"reason"`
}

// CapitalizeNearAcquisitionCost bucht den Instandsetzungs- und
// Modernisierungsaufwand der ersten drei Jahre auf das Gebäudekonto um.
//
// § 6 Abs. 1 Nr. 1a EStG macht diesen Aufwand zu Herstellungskosten, sobald er
// 15 % der Gebäude-Anschaffungskosten übersteigt — und zwar den ganzen, nicht
// den übersteigenden Teil. Er ist dann nicht sofort abziehbar, sondern über die
// Restnutzungsdauer des Gebäudes abzuschreiben.
//
// Gebucht wird eine Umbuchung SOLL Gebäude an HABEN Aufwandskonto und keine
// Generalumkehr der ursprünglichen Belege. Das weicht bewusst von der
// Formulierung der Wellenbeschreibung ab („Umbuchung über RecordCostAdjustment
// mit Storno der Aufwandsbuchungen"); die Kartei wird über RecordCostAdjustment
// fortgeschrieben, die Aufwandsbuchungen werden aber nicht storniert. Der Unterschied ist wichtig: die
// ursprüngliche Buchung enthält neben dem Aufwand die Vorsteuer und die
// Verbindlichkeit gegenüber dem Handwerker, und die ist oft längst bezahlt. Sie
// zurückzunehmen risse die Zahlung mit, obwohl an ihr nichts falsch war — falsch
// war allein die Kontierung des Aufwands. Die Vorsteuer bleibt ebenfalls, wo sie
// ist: § 6 Abs. 1 Nr. 1a EStG ordnet einkommensteuerlich zu und ändert am
// Vorsteuerabzug nichts.
func (s *AssetService) CapitalizeNearAcquisitionCost(
	ctx context.Context, req CapitalizeNearAcquisitionCostRequest,
) (*domain.FixedAsset, error) {
	if strings.TrimSpace(req.Reason) == "" {
		return nil, fmt.Errorf(
			"zur Aktivierung als nachträgliche Herstellungskosten gehört ihr Grund. Sie nimmt einen " +
				"sofort abgezogenen Aufwand zurück und verteilt ihn über die Restnutzungsdauer — das " +
				"ist eine Beurteilung und keine Rechnung")
	}
	asset, err := s.assetRepo.FindByID(ctx, req.AssetID)
	if err != nil {
		return nil, fmt.Errorf("Anlagegut %d wurde nicht gefunden: %w", req.AssetID, err)
	}
	entry, known := accounting.LookupAssetAccount(asset.Account)
	if !known || !entry.Immovable {
		return nil, fmt.Errorf(
			"§ 6 Abs. 1 Nr. 1a EStG gilt für Gebäude; %s ist keines", asset.InventoryNumber)
	}
	params, err := accounting.TaxParametersFor(asset.AcquisitionDate)
	if err != nil {
		return nil, err
	}
	periodEnd, err := yearsAfter(asset.AcquisitionDate, params.NearAcquisitionYears)
	if err != nil {
		return nil, err
	}

	// Gesammelt wird derselbe Aufwand, den CheckNearAcquisitionCost zählt —
	// eine zweite Auswahlregel wäre eine zweite Wahrheit.
	movements := make([]domain.AssetMovement, 0, 4)
	var total domain.Cents
	byAccount := map[string]domain.Cents{}
	for _, m := range asset.Movements {
		if m.Kind != domain.AssetMovementMaintenance || !m.IsModernisation {
			continue
		}
		if m.Date < asset.AcquisitionDate || m.Date > periodEnd {
			continue
		}
		movements = append(movements, m)
		total += m.ExpenseAmount
		byAccount[maintenanceAccountOf(m, asset)] += m.ExpenseAmount
	}
	if total == 0 {
		return nil, fmt.Errorf(
			"zu %s ist im Dreijahreszeitraum kein Instandsetzungs- oder Modernisierungsaufwand "+
				"gebucht, der zu aktivieren wäre", asset.InventoryNumber)
	}
	limit := domain.MulRound(asset.AcquisitionCost, params.NearAcquisitionPermille, 1000)
	if total <= limit {
		return nil, fmt.Errorf(
			"der Instandsetzungs- und Modernisierungsaufwand der ersten drei Jahre beträgt %s € und "+
				"bleibt damit unter dem Rahmen von %s € (15 %% der Anschaffungskosten). Er ist sofort "+
				"abziehbar; eine Aktivierung wäre falsch",
			total, limit)
	}

	date := req.Date
	if date == "" {
		date = periodEnd
	}
	if len(date) != 10 {
		return nil, fmt.Errorf("das Datum der Umbuchung ist unvollständig (erwartet JJJJ-MM-TT)")
	}

	accounts := make([]string, 0, len(byAccount))
	for account := range byAccount {
		accounts = append(accounts, account)
	}
	sort.Strings(accounts)
	lines := []domain.JournalLine{{
		Side: domain.SideDebit, Account: asset.Account, Amount: total,
		Text: "Anschaffungsnahe Herstellungskosten " + asset.InventoryNumber,
	}}
	for _, account := range accounts {
		lines = append(lines, domain.JournalLine{
			Side: domain.SideCredit, Account: account, Amount: byAccount[account],
			Text: "Umbuchung Erhaltungsaufwand " + asset.InventoryNumber,
		})
	}

	posted, err := s.journalSvc.Post(ctx, &domain.JournalEntry{
		BookingDate: date, DocumentDate: date, ServiceDateFrom: date, ServiceDateTo: date,
		Description: fmt.Sprintf(
			"Anschaffungsnahe Herstellungskosten %s (§ 6 Abs. 1 Nr. 1a EStG): %s",
			asset.InventoryNumber, req.Reason),
		Source:             domain.EntrySourceManual,
		DocumentNumber:     asset.InventoryNumber,
		TaxTreatment:       domain.TaxTreatmentNotTaxable,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
	})
	if err != nil {
		return nil, err
	}

	// Die Bewegungen zählen ab jetzt nicht mehr gegen den Rahmen: sie sind kein
	// Erhaltungsaufwand mehr. Ohne diesen Vermerk zählte ein zweiter Lauf sie
	// erneut und aktivierte denselben Betrag ein zweites Mal.
	for i := range movements {
		m := movements[i]
		m.IsModernisation = false
		m.Note = appendText(m.Note, fmt.Sprintf(
			"als nachträgliche Herstellungskosten aktiviert (%s)", posted.EntryNumber))
		if err := s.assetRepo.AddMovement(ctx, &m); err != nil {
			return nil, fmt.Errorf(
				"die Umbuchung %s wurde gebucht, die Bewegung vom %s aber nicht fortgeschrieben: %w",
				posted.EntryNumber, m.Date, err)
		}
	}

	if _, err := s.RecordCostAdjustment(ctx, CostAdjustmentRequest{
		AssetID: asset.ID, Date: date, Amount: total,
		Note: fmt.Sprintf(
			"Anschaffungsnahe Herstellungskosten nach § 6 Abs. 1 Nr. 1a EStG (%s): %s",
			posted.EntryNumber, req.Reason),
		JournalEntryID: &posted.ID,
	}); err != nil {
		return nil, fmt.Errorf(
			"die Umbuchung %s wurde gebucht, die Anschaffungskosten in der Kartei aber nicht "+
				"fortgeschrieben: %w", posted.EntryNumber, err)
	}

	s.audit(ctx, domain.AuditActionUpdate, asset.ID, fmt.Sprintf(
		"Anschaffungsnahe Herstellungskosten %s: %s € von %d Erhaltungsaufwandsbuchungen auf %s "+
			"umgebucht (%s) — %s",
		asset.InventoryNumber, total, len(movements), asset.Account, posted.EntryNumber, req.Reason))
	return s.reload(ctx, asset.ID)
}

// maintenanceAccountOf liefert das Aufwandskonto, auf das eine
// Erhaltungsaufwandsbewegung gebucht wurde.
//
// Gefragt wird zuerst das Feld ExpenseAccount an der Bewegung. Bewegungen aus
// der Zeit vor diesem Feld haben es nicht; für sie bleibt der Text („1.200,00 €
// auf 6335: …") die Quelle, aus der es sich lesen lässt. Und wo auch der nichts
// hergibt, gilt das Konto, das der Kontenkatalog für dieses Anlagegut vorsieht:
// dasselbe, das BookMaintenance ohne ausdrückliche Wahl genommen hätte.
func maintenanceAccountOf(m domain.AssetMovement, asset *domain.FixedAsset) string {
	if account := strings.TrimSpace(m.ExpenseAccount); account != "" {
		return account
	}
	if _, rest, ok := strings.Cut(m.Note, " auf "); ok {
		if account, _, ok := strings.Cut(rest, ":"); ok {
			if trimmed := strings.TrimSpace(account); len(trimmed) == 4 {
				return trimmed
			}
		}
	}
	if account, err := accounting.MaintenanceAccount(asset.Class, asset.Account); err == nil {
		return account
	}
	return "6450"
}
