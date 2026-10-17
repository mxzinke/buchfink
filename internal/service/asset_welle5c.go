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

// Die Anlagenthemen der Welle 5c: die Sperren, die sich nach dem Anlagekonto
// richten, der Wertaufholungsbericht und die Einheitlichkeit des
// Sammelposten-Wahlrechts.

// validateMethodForAccount hält die Abschreibungsverfahren an den Merkmalen des
// Anlagekontos fest.
//
// Ob ein Wirtschaftsgut beweglich ist, steht im Kontenkatalog, nicht am
// Anlagegut — und danach richten sich drei Regeln, die bisher nur zum Teil
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
				"%s (%s) ist ein unbewegliches Wirtschaftsgut. Die degressive Abschreibung des "+
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
				"%s (%s) ist kein Gebäude. Die festen Sätze des § 7 Abs. 4 EStG gelten für Gebäude; "+
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
				"%s (%s) ist kein Fahrzeug. § 7 Abs. 2a EStG gilt für neue, rein elektrisch "+
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
	// Einmal für den ganzen Bericht, nicht je Anlagegut.
	periods := s.fiscalPeriods(ctx)
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

		s.enrich(asset, year, startMonth, periods)
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
	// berichtigter Name, ein neuer Standort — das Anlagegut an, bis jemand eine
	// Begründung nachträgt. Die Begründung
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
			"und der Software dieses Schreibens; auf dem Konto stehen auch andere Wirtschaftsgüter, "+
			"und das ist der Grund für die Abweichung. Eingetragen sind %d Monate — festzuhalten ist, "+
			"worauf diese Nutzungsdauer beruht. Die Tabellen binden nur die Finanzverwaltung; eine "+
			"Abweichung ohne Begründung hält aber keiner Prüfung stand",
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
