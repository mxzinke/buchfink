package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// TakeoverAsset ist ein Anlagegut aus dem Anlagenverzeichnis des Vorsystems.
type TakeoverAsset struct {
	Name string `json:"name"`
	// Account ist das Anlagekonto. Aus ihm folgen Anlagenklasse und
	// Aufwandskonto der Abschreibung.
	Account string `json:"account"`
	// AcquisitionDate und AcquisitionCost sind der Zugang im Vorsystem: der Tag
	// der Anschaffung und die historischen Anschaffungs- oder
	// Herstellungskosten, nicht der Buchwert.
	AcquisitionDate string       `json:"acquisitionDate"`
	AcquisitionCost domain.Cents `json:"acquisitionCost"`
	// Method ist das Abschreibungsverfahren; leer heißt linear, bei einem nicht
	// abnutzbaren Anlagekonto keine Abschreibung.
	Method           domain.DepreciationMethod `json:"method,omitempty"`
	UsefulLifeMonths int                       `json:"usefulLifeMonths,omitempty"`
	UsefulLifeReason string                    `json:"usefulLifeReason,omitempty"`
	// BuildingReferenceDate ist bei einem Gebäude der Stichtag für den Satz des
	// § 7 Abs. 4 EStG.
	BuildingReferenceDate string `json:"buildingReferenceDate,omitempty"`
	// AccumulatedDepreciation sind die Abschreibungen, die das Vorsystem bis zum
	// Umstieg gebucht hat. Anschaffungskosten abzüglich dieses Betrags sind der
	// Buchwert, mit dem das Anlagegut im Anlagekonto der Schlussbilanz steht.
	AccumulatedDepreciation domain.Cents `json:"accumulatedDepreciation"`
	// LegacyRef ist die Inventarnummer im Vorsystem.
	LegacyRef string `json:"legacyRef,omitempty"`
}

// TakeoverRequest ist die Schlussbilanz des Vorsystems mit ihren Nebenbüchern.
//
// Accounts enthält die Bestände je Sachkonto, wie die Summen- und Saldenliste
// sie nennt, auch die Sammelkonten der Forderungen und Verbindlichkeiten aus
// Lieferungen und Leistungen. Auf ein Sammelkonto bucht Buchfink nicht; sein
// Bestand ist die Kontrollsumme der einzeln übernommenen offenen Posten.
// Geschäftsjahr und Buchungstag setzt der Dienst aus dem festgelegten Umstieg.
type TakeoverRequest struct {
	OpeningBalanceRequest
	// Assets ist das Anlagenverzeichnis zum Umstieg. Die Buchwerte je
	// Anlagekonto ergeben den Bestand dieses Kontos in Accounts.
	Assets []TakeoverAsset `json:"assets"`
}

// TakeoverAssetPreview zeigt ein Anlagegut, wie Buchfink es fortführt.
type TakeoverAssetPreview struct {
	// AssetID und InventoryNumber sind nach der Buchung belegt.
	AssetID                 uint         `json:"assetId,omitempty"`
	InventoryNumber         string       `json:"inventoryNumber,omitempty"`
	Name                    string       `json:"name"`
	Account                 string       `json:"account"`
	AcquisitionCost         domain.Cents `json:"acquisitionCost"`
	AccumulatedDepreciation domain.Cents `json:"accumulatedDepreciation"`
	BookValue               domain.Cents `json:"bookValue"`
	// FirstYearDepreciation ist die planmäßige Abschreibung im ersten
	// Geschäftsjahr in Buchfink.
	FirstYearDepreciation domain.Cents `json:"firstYearDepreciation"`
}

// TakeoverPreview ist die Vorschau der Eröffnung oder ihr gebuchter Stand.
type TakeoverPreview struct {
	OpeningBalancePreview
	Assets []TakeoverAssetPreview `json:"assets"`
	// Differences nennt, was der Buchung entgegensteht: Aktiva und Passiva
	// gehen auseinander, oder ein Nebenbuch weicht von seinem Sachkonto ab.
	Differences []string `json:"differences"`
}

// EnsureLists ersetzt nicht belegte Listen durch leere.
func (p *TakeoverPreview) EnsureLists() {
	p.OpeningBalancePreview.EnsureLists()
	if p.Assets == nil {
		p.Assets = make([]TakeoverAssetPreview, 0)
	}
	if p.Differences == nil {
		p.Differences = make([]string, 0)
	}
}

// ErrTakeoverNotSet weist eine Eröffnung ab, solange der Mensch keinen Umstieg
// festgelegt hat.
var ErrTakeoverNotSet = errors.New(
	"für das Unternehmen ist kein Umstieg festgelegt. Ob es mit bestehenden Büchern beginnt und zu welchem " +
		"Tag, legt der Mensch in der App fest (Unternehmen › Stammdaten)")

// takeoverPlan ist die geprüfte Eröffnung vor dem Schreiben.
type takeoverPlan struct {
	state   *TakeoverState
	opening OpeningBalanceRequest
	// statement ist die Schlussbilanz mit den Sammelkonten.
	statement takeoverStatement
	assets    []*domain.FixedAsset
	carried   []domain.Cents
	// existing sind Anlagen einer stornierten Eröffnung; sie bleiben im
	// Verzeichnis und gehören zur erneuten Eröffnung.
	existing []domain.FixedAsset
	preview  *TakeoverPreview
}

// Preview zeigt die Eröffnung aus der Schlussbilanz des Vorsystems, ohne zu
// buchen.
func (s *TakeoverService) Preview(ctx context.Context, req TakeoverRequest) (*TakeoverPreview, error) {
	plan, err := s.plan(ctx, req)
	if err != nil {
		return nil, err
	}
	return plan.preview, nil
}

// Book bucht die Eröffnung: die Bestände der Sachkonten, je offenen Posten
// eine Buchung und je Anlagegut den Eintrag im Anlagenverzeichnis mit
// Anschaffungskosten und aufgelaufenen Abschreibungen.
//
// Alles oder nichts: eine Eröffnung, deren Anlagenverzeichnis fehlt, wiese
// Buchwerte aus, die niemand fortführt.
func (s *TakeoverService) Book(ctx context.Context, req TakeoverRequest) (*TakeoverPreview, error) {
	plan, err := s.plan(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(plan.preview.Differences) > 0 {
		return nil, fmt.Errorf("die Eröffnung stimmt nicht mit der Schlussbilanz des Vorsystems überein: %s",
			strings.Join(plan.preview.Differences, " "))
	}
	return atomically(ctx, s.journal, func(ctx context.Context) (*TakeoverPreview, error) {
		booked, err := s.journal.BookOpeningBalance(ctx, plan.opening)
		if err != nil {
			return nil, err
		}
		preview := plan.preview
		preview.OpeningBalancePreview = *booked
		if err := s.takeOverAssets(ctx, plan, booked.Entries); err != nil {
			return nil, err
		}
		if err := s.saveStatement(ctx, plan.statement); err != nil {
			return nil, err
		}
		if err := logAudit(ctx, s.auditRepo, domain.AuditActionCreate, "TAKEOVER", plan.state.Date, fmt.Sprintf(
			"Eröffnung zum %s aus der Schlussbilanz des Vorsystems gebucht: %d Buchungen, %d Anlagen",
			domain.GermanDate(plan.state.Date), len(booked.Entries), len(preview.Assets))); err != nil {
			return nil, err
		}
		preview.EnsureLists()
		return preview, nil
	})
}

// plan prüft die Eröffnung und rechnet ihre Vorschau.
func (s *TakeoverService) plan(ctx context.Context, req TakeoverRequest) (*takeoverPlan, error) {
	state, err := s.State(ctx)
	if err != nil {
		return nil, err
	}
	if state.Date == "" {
		return nil, ErrTakeoverNotSet
	}
	plan := &takeoverPlan{state: state}
	plan.opening = req.OpeningBalanceRequest
	plan.opening.FiscalYear, plan.opening.Date = state.FiscalYear, state.Date
	if req.ReceiptID != nil {
		plan.statement.ReceiptID = *req.ReceiptID
	}
	plan.statement.LegacySystem = req.LegacySystem
	plan.statement.Accounts = append([]OpeningBalanceLine{}, req.Accounts...)

	// Die Sammelkonten sind Kontrollsummen und werden nicht gebucht.
	control := map[domain.ContactType]domain.Cents{}
	plan.opening.Accounts = make([]OpeningBalanceLine, 0, len(req.Accounts))
	for i, line := range req.Accounts {
		kind, collective := domain.CollectiveAccounts()[line.Account]
		if !collective {
			plan.opening.Accounts = append(plan.opening.Accounts, line)
			continue
		}
		want := domain.SideDebit
		if kind == domain.ContactTypeVendor {
			want = domain.SideCredit
		}
		if line.Side != want || line.Amount <= 0 {
			return nil, fmt.Errorf(
				"Zeile %d: das Sammelkonto %s steht in der Schlussbilanz mit einem positiven Bestand im %s",
				i+1, line.Account, sideLabel(want))
		}
		control[kind] += line.Amount
	}

	preview, err := s.journal.PreviewOpeningBalance(ctx, plan.opening)
	if err != nil {
		return nil, err
	}
	plan.preview = &TakeoverPreview{OpeningBalancePreview: *preview}
	if !preview.Balanced {
		plan.preview.Differences = append(plan.preview.Differences, fmt.Sprintf(
			"Aktiva (%s €) und Passiva (%s €) der Schlussbilanz unterscheiden sich um %s €.",
			preview.DebitTotal, preview.CreditTotal, (preview.DebitTotal-preview.CreditTotal).Abs()))
	}
	plan.preview.Differences = append(plan.preview.Differences, openItemDifference(
		"Forderungen", domain.AccountForderungenLuL, req.Receivables, control[domain.ContactTypeCustomer])...)
	plan.preview.Differences = append(plan.preview.Differences, openItemDifference(
		"Verbindlichkeiten", domain.AccountVerbindlichkeitenLuL, req.Payables, control[domain.ContactTypeVendor])...)

	if err := s.planAssets(ctx, plan, req); err != nil {
		return nil, err
	}
	plan.preview.EnsureLists()
	return plan, nil
}

// openItemDifference vergleicht die einzeln übernommenen offenen Posten mit dem
// Bestand ihres Sammelkontos in der Schlussbilanz.
func openItemDifference(label, account string, items []OpenOpeningItem, control domain.Cents) []string {
	var sum domain.Cents
	for _, item := range items {
		sum += item.Amount
	}
	if sum == control {
		return nil
	}
	return []string{fmt.Sprintf(
		"Die offenen Posten der %s ergeben %s €, das Sammelkonto %s der Schlussbilanz steht auf %s € "+
			"(Differenz %s €).",
		label, sum, account, control, (sum - control).Abs())}
}

// planAssets prüft das Anlagenverzeichnis und vergleicht seine Buchwerte je
// Anlagekonto mit den Beständen der Schlussbilanz.
func (s *TakeoverService) planAssets(ctx context.Context, plan *takeoverPlan, req TakeoverRequest) error {
	existing, err := s.takenOverAssets(ctx)
	if err != nil {
		return err
	}
	if len(existing) > 0 && len(req.Assets) > 0 {
		return fmt.Errorf(
			"das Anlagenverzeichnis enthält schon %d aus dem Vorsystem übernommene Anlagen. Sie bleiben nach "+
				"einem Storno der Eröffnung bestehen und gehören zur erneuten Eröffnung; die Angabe der "+
				"Anlagen entfällt dann", len(existing))
	}
	plan.existing = existing

	startMonth := s.journal.fiscalYearStartMonth(ctx)
	bookValues := map[string]domain.Cents{}
	for i := range existing {
		asset := &existing[i]
		cost, carried := takeoverValues(asset, plan.state.FiscalYear)
		bookValues[asset.Account] += cost - carried
		plan.preview.Assets = append(plan.preview.Assets, s.assetPreview(ctx, asset, cost, carried, plan.state, startMonth))
	}
	for i := range req.Assets {
		asset, err := s.takeoverAsset(ctx, plan, &req.Assets[i], startMonth)
		if err != nil {
			return fmt.Errorf("Anlage %d (%s): %w", i+1, strings.TrimSpace(req.Assets[i].Name), err)
		}
		carried := req.Assets[i].AccumulatedDepreciation
		plan.assets = append(plan.assets, asset)
		plan.carried = append(plan.carried, carried)
		bookValues[asset.Account] += asset.AcquisitionCost - carried
		plan.preview.Assets = append(plan.preview.Assets,
			s.assetPreview(ctx, asset, asset.AcquisitionCost, carried, plan.state, startMonth))
	}

	balances := map[string]domain.Cents{}
	for _, line := range plan.opening.Accounts {
		if !isFixedAssetAccount(line.Account) {
			continue
		}
		if line.Side == domain.SideDebit {
			balances[line.Account] += line.Amount
		} else {
			balances[line.Account] -= line.Amount
		}
	}
	accounts := make([]string, 0, len(balances)+len(bookValues))
	for account := range balances {
		accounts = append(accounts, account)
	}
	for account := range bookValues {
		if _, ok := balances[account]; !ok {
			accounts = append(accounts, account)
		}
	}
	sort.Strings(accounts)
	for _, account := range accounts {
		if bookValues[account] == balances[account] {
			continue
		}
		plan.preview.Differences = append(plan.preview.Differences, fmt.Sprintf(
			"Die Anlagen auf Konto %s haben zusammen einen Buchwert von %s €, die Schlussbilanz nennt %s € "+
				"(Differenz %s €).",
			account, bookValues[account], balances[account], (bookValues[account]-balances[account]).Abs()))
	}
	return nil
}

// takeoverAsset baut das Anlagegut aus den Angaben des Vorsystems.
func (s *TakeoverService) takeoverAsset(
	ctx context.Context, plan *takeoverPlan, in *TakeoverAsset, startMonth int,
) (*domain.FixedAsset, error) {
	if _, err := time.Parse("2006-01-02", in.AcquisitionDate); err != nil || in.AcquisitionDate >= plan.state.Date {
		return nil, fmt.Errorf("das Anschaffungsdatum liegt vor dem Umstieg am %s (erwartet JJJJ-MM-TT)",
			domain.GermanDate(plan.state.Date))
	}
	if in.AccumulatedDepreciation < 0 || in.AccumulatedDepreciation > in.AcquisitionCost {
		return nil, fmt.Errorf(
			"die aufgelaufenen Abschreibungen (%s €) liegen zwischen null und den Anschaffungskosten (%s €)",
			in.AccumulatedDepreciation, in.AcquisitionCost)
	}
	account, known := accounting.LookupAssetAccount(in.Account)
	class := account.Class
	if class == "" {
		class = domain.AssetClassTangible
	}
	method := in.Method
	if method == "" {
		method = domain.DepreciationLinear
		if known && !account.Depreciable {
			method = domain.DepreciationNone
		}
	}
	if method == domain.DepreciationImmediate {
		return nil, fmt.Errorf(
			"ein sofort abgezogenes geringwertiges Wirtschaftsgut hat keinen Buchwert und wird nicht übernommen")
	}
	notes := "Aus dem Vorsystem übernommen"
	if plan.opening.LegacySystem != "" {
		notes = "Aus " + plan.opening.LegacySystem + " übernommen"
	}
	if ref := strings.TrimSpace(in.LegacyRef); ref != "" {
		notes += ", dort geführt als " + ref
	}
	asset := &domain.FixedAsset{
		Name:                  strings.TrimSpace(in.Name),
		Class:                 class,
		Account:               in.Account,
		DepreciationAccount:   account.DepreciationAccount,
		AcquisitionDate:       in.AcquisitionDate,
		AcquisitionCost:       in.AcquisitionCost,
		Method:                method,
		UsefulLifeMonths:      in.UsefulLifeMonths,
		UsefulLifeReason:      strings.TrimSpace(in.UsefulLifeReason),
		BuildingReferenceDate: in.BuildingReferenceDate,
		IsElectric:            method == domain.DepreciationElectricVehicle,
		Notes:                 notes + ".",
	}
	if method == domain.DepreciationPool {
		asset.PoolYear = domain.GetFiscalYearForDate(in.AcquisitionDate, startMonth)
	}
	if err := asset.Validate(); err != nil {
		return nil, err
	}
	if err := s.assets.validateAccounts(ctx, asset); err != nil {
		return nil, err
	}
	// Der Plan muss sich rechnen lassen, bevor gebucht wird: ein Verfahren, das
	// es für den Anschaffungstag nicht gibt, fiele sonst erst im AfA-Lauf auf.
	if _, err := accounting.BuildAfASchedule(s.takeoverSchedulePlan(ctx, asset, in.AcquisitionCost,
		in.AccumulatedDepreciation, plan.state, startMonth)); err != nil {
		return nil, err
	}
	return asset, nil
}

// takeoverSchedulePlan ist der Abschreibungsplan eines Anlageguts mit dem
// übernommenen Stand, wie ihn der AfA-Lauf nach der Buchung rechnet.
func (s *TakeoverService) takeoverSchedulePlan(
	ctx context.Context, asset *domain.FixedAsset, cost, carried domain.Cents, state *TakeoverState, startMonth int,
) accounting.AfAPlan {
	planned := *asset
	planned.Movements = []domain.AssetMovement{
		{Kind: domain.AssetMovementAcquisition, CostAmount: cost},
		{Kind: domain.AssetMovementTakeover, FiscalYear: state.FiscalYear - 1, DepreciationAmount: carried},
	}
	return s.assets.planWithPeriods(ctx, &planned, startMonth)
}

func (s *TakeoverService) assetPreview(
	ctx context.Context, asset *domain.FixedAsset, cost, carried domain.Cents, state *TakeoverState, startMonth int,
) TakeoverAssetPreview {
	preview := TakeoverAssetPreview{
		AssetID: asset.ID, InventoryNumber: asset.InventoryNumber,
		Name: asset.Name, Account: asset.Account,
		AcquisitionCost: cost, AccumulatedDepreciation: carried, BookValue: cost - carried,
	}
	if rows, err := accounting.BuildAfASchedule(
		s.takeoverSchedulePlan(ctx, asset, cost, carried, state, startMonth)); err == nil {
		preview.FirstYearDepreciation = accounting.ScheduleAmountFor(rows, state.FiscalYear)
	}
	return preview
}

// takeOverAssets legt die Anlagen an und verbindet ihre aufgelaufenen
// Abschreibungen mit der Eröffnungsbuchung der Sachkonten.
func (s *TakeoverService) takeOverAssets(
	ctx context.Context, plan *takeoverPlan, booked []*domain.JournalEntry,
) error {
	if len(plan.assets) == 0 && len(plan.existing) == 0 {
		return nil
	}
	// Die Bestände der Anlagekonten stehen in der Buchung der Sachkonten; sie
	// ist die erste der Eröffnung.
	var ledgerEntry *domain.JournalEntry
	if len(plan.opening.Accounts) > 0 && len(booked) > 0 {
		ledgerEntry = booked[0]
	}
	if ledgerEntry == nil {
		return fmt.Errorf("zu den Anlagen fehlen die Bestände der Anlagekonten")
	}
	date := addDays(plan.state.Date, -1)
	movement := func(asset *domain.FixedAsset, carried domain.Cents) error {
		return s.assetRepo.AddMovement(ctx, &domain.AssetMovement{
			AssetID:            asset.ID,
			Kind:               domain.AssetMovementTakeover,
			Account:            asset.Account,
			Date:               date,
			FiscalYear:         plan.state.FiscalYear - 1,
			DepreciationAmount: carried,
			JournalEntryID:     &ledgerEntry.ID,
			Note:               "Im Vorsystem bis zum Umstieg gebuchte Abschreibungen",
		})
	}
	for i := range plan.existing {
		asset := &plan.existing[i]
		_, carried := takeoverValues(asset, plan.state.FiscalYear)
		for _, m := range asset.Movements {
			if m.Kind == domain.AssetMovementTakeover {
				if err := s.assetRepo.DeleteMovement(ctx, m.ID); err != nil {
					return err
				}
			}
		}
		if err := movement(asset, carried); err != nil {
			return fmt.Errorf("%s: %w", asset.InventoryNumber, err)
		}
	}
	for i, asset := range plan.assets {
		saved, err := s.assets.Save(ctx, asset)
		if err != nil {
			return fmt.Errorf("Anlage %d (%s): %w", i+1, asset.Name, err)
		}
		if err := movement(saved, plan.carried[i]); err != nil {
			return fmt.Errorf("%s: %w", saved.InventoryNumber, err)
		}
		preview := &plan.preview.Assets[len(plan.existing)+i]
		preview.AssetID, preview.InventoryNumber = saved.ID, saved.InventoryNumber
	}
	return nil
}

// takenOverAssets sind die Anlagen des Verzeichnisses, die aus dem Vorsystem
// stammen.
func (s *TakeoverService) takenOverAssets(ctx context.Context) ([]domain.FixedAsset, error) {
	all, err := s.assetRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("das Anlagenverzeichnis konnte nicht gelesen werden: %w", err)
	}
	out := make([]domain.FixedAsset, 0)
	for i := range all {
		for _, m := range all[i].Movements {
			if m.Kind == domain.AssetMovementTakeover {
				out = append(out, all[i])
				break
			}
		}
	}
	return out, nil
}

// takeoverValues sind Anschaffungskosten und aufgelaufene Abschreibungen eines
// übernommenen Anlageguts zum Umstieg.
func takeoverValues(asset *domain.FixedAsset, fiscalYear int) (cost, carried domain.Cents) {
	for _, m := range asset.Movements {
		if m.FiscalYear >= fiscalYear {
			continue
		}
		cost += m.CostAmount
		carried += m.DepreciationAmount
	}
	return cost, carried
}

func sideLabel(side domain.Side) string {
	if side == domain.SideCredit {
		return "Haben"
	}
	return "Soll"
}
