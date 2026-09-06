package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Eigene Konten (BEL-06 K2).
//
// Der SKR04 deckt den Regelfall ab, aber nicht jeden Betrieb: wer eine
// Aufwandsart getrennt auswerten will — „Fremdleistungen Fotografie" neben
// „Fremdleistungen Text" —, braucht ein eigenes Konto, und der Weg dahin darf
// nicht heißen, ein fremdes Konto umzubenennen. Das wäre der Weg, den ohne
// diese Funktion jeder ginge, und er zerstörte still die Zuordnung des
// Kontenrahmens.
//
// Zwei Bedingungen machen ein eigenes Konto tragfähig, und beide werden hier
// geprüft: die Nummer muss frei sein — sonst überschriebe sie ein Konto des
// Rahmens —, und die Gliederungsposition ist Pflicht, weil ein Konto ohne sie
// weder in der Bilanz noch in der GuV noch in der E-Bilanz erscheint. Es wäre
// ein Konto, auf dem Geld liegt, das im Abschluss fehlt.

// ChartInvalidator verwirft einen zwischengespeicherten Kontenplan.
//
// Der Buchungsweg hält den Kontenplan für die Laufzeit fest; ohne diesen Anruf
// wäre ein neu angelegtes Konto für ihn nicht vorhanden und ein gesperrtes
// weiterhin bebuchbar — die Kontenpflege bliebe bis zum Neustart wirkungslos.
type ChartInvalidator interface {
	InvalidateChart()
}

// AccountService legt eigene Konten an und sperrt sie.
type AccountService struct {
	accountRepo domain.AccountRepository
	auditRepo   domain.AuditRepository
	taxResolver domain.TaxResolver
	chart       ChartInvalidator
}

// SetChartInvalidator hängt den Buchungsweg an, dessen Kontenplan nach jeder
// Änderung neu zu lesen ist. Optional.
func (s *AccountService) SetChartInvalidator(c ChartInvalidator) { s.chart = c }

// invalidateChart meldet die Änderung an den Buchungsweg.
func (s *AccountService) invalidateChart() {
	if s.chart != nil {
		s.chart.InvalidateChart()
	}
}

// NewAccountService verdrahtet die Kontenpflege.
func NewAccountService(
	accountRepo domain.AccountRepository,
	auditRepo domain.AuditRepository,
) *AccountService {
	return &AccountService{
		accountRepo: accountRepo,
		auditRepo:   auditRepo,
		taxResolver: accounting.NewSKR04TaxResolver(),
	}
}

// CustomAccountRequest ist die Eingabe für ein eigenes Konto.
type CustomAccountRequest struct {
	Number string `json:"number"`
	Name   string `json:"name"`
	// HGBPosition ist die Gliederungsposition des SKR04 (position_id), unter
	// der das Konto in Bilanz oder GuV erscheint. Pflicht.
	HGBPosition string `json:"hgbPosition"`
	// TaxKeyDefault ist der Steuerschlüssel, den die Oberfläche für Buchungen
	// auf dieses Konto vorschlägt. Freiwillig und ohne Automatik: ein
	// selbst angelegtes Konto ist kein Automatikkonto, die Steuerzeile entsteht
	// weiterhin aus dem Steuerfall der Buchung.
	TaxKeyDefault string `json:"taxKeyDefault"`
	Description   string `json:"description"`
}

// CreateCustom legt ein eigenes Konto an.
func (s *AccountService) CreateCustom(ctx context.Context, req CustomAccountRequest) (*domain.Account, error) {
	number := strings.TrimSpace(req.Number)
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("ein Konto braucht eine Bezeichnung")
	}
	// Die Position wird vor der Nummer aufgelöst: ob eine Nummer der Klasse 0
	// zulässig ist, richtet sich nach ihr (siehe ensureFreeNumber).
	position, err := positionByID(strings.TrimSpace(req.HGBPosition))
	if err != nil {
		return nil, err
	}
	if err := s.ensureFreeNumber(ctx, number, position); err != nil {
		return nil, err
	}
	if !accounting.HasPositionTarget(position.ID) {
		return nil, fmt.Errorf(
			"die Position %q ist in der Gliederung von Bilanz und GuV nicht hinterlegt. Ein Konto "+
				"unter ihr erschiene weder im Abschluss noch in der E-Bilanz — wähle eine Position, "+
				"die Buchfink abbildet", position.Name)
	}

	if key := strings.TrimSpace(req.TaxKeyDefault); key != "" {
		if err := ensureKnownTaxKey(key); err != nil {
			return nil, err
		}
	}

	account := &domain.Account{
		Number:           number,
		Name:             name,
		Type:             domain.AccountType(position.AccountType),
		Category:         position.MainGroup,
		Subcategory:      position.Group,
		Kontenklasse:     position.Kontenklasse.Number,
		KontenklasseName: position.Kontenklasse.Name,
		PositionID:       position.ID,
		Posten:           position.Name,
		BalanceSide:      position.BalanceSide,
		HGBCode:          position.HGBCode,
		StatementType:    position.StatementType,
		Description:      strings.TrimSpace(req.Description),
		Zusatzfunktion:   strings.TrimSpace(req.TaxKeyDefault),
		IsCustom:         true,
		IsActive:         true,
	}
	if err := s.accountRepo.Create(ctx, account); err != nil {
		return nil, fmt.Errorf("das Konto konnte nicht angelegt werden: %w", err)
	}
	s.invalidateChart()
	s.log(ctx, domain.AuditActionCreate, account, fmt.Sprintf(
		"Eigenes Konto %s „%s\" angelegt, Gliederungsposition %s (%s)",
		account.Number, account.Name, position.ID, position.Name))
	return account, nil
}

// SetBlocked sperrt ein eigenes Konto für neue Buchungen oder gibt es wieder
// frei.
//
// Gesperrt und nicht gelöscht, sobald das Konto bebucht ist: die Buchungen
// zeigen auf die Nummer, und ein gelöschtes Konto ließe Bilanz und Kontoblatt
// mit einer Nummer zurück, zu der es keine Bezeichnung mehr gibt. Ein Konto
// ohne jede Buchung wird ebenfalls nur gesperrt — der Unterschied zwischen
// „nie benutzt" und „gelöscht" ist für den Kontenplan keiner, und eine zweite
// Fassung der Regel wäre die, die beim nächsten Fall vergessen wird.
func (s *AccountService) SetBlocked(ctx context.Context, number string, blocked bool, reason string) (*domain.Account, error) {
	account, err := s.accountRepo.FindByNumber(ctx, strings.TrimSpace(number))
	if err != nil || account == nil {
		return nil, fmt.Errorf("das Konto %s gibt es nicht", number)
	}
	if !account.IsCustom {
		return nil, fmt.Errorf(
			"das Konto %s gehört zum SKR04 und wird nicht gesperrt. Gesperrt werden nur selbst "+
				"angelegte Konten — ein Konto des Kontenrahmens fehlte sonst dort, wo eine Auswertung "+
				"es erwartet", account.Number)
	}
	if blocked && strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("zum Sperren eines Kontos gehört eine Begründung")
	}
	if account.IsActive == !blocked {
		return account, nil
	}
	before := *account
	account.IsActive = !blocked
	if err := s.accountRepo.Update(ctx, account); err != nil {
		return nil, err
	}
	s.invalidateChart()
	action := "freigegeben"
	if blocked {
		action = "gesperrt: " + strings.TrimSpace(reason)
	}
	if s.auditRepo != nil {
		_ = s.auditRepo.LogChange(ctx, domain.AuditActionUpdate, "ACCOUNT", account.Number,
			fmt.Sprintf("Konto %s „%s\" %s", account.Number, account.Name, action), before, account)
	}
	return account, nil
}

// CustomAccounts liefert die selbst angelegten Konten.
func (s *AccountService) CustomAccounts(ctx context.Context) ([]domain.Account, error) {
	all, err := s.accountRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Account, 0, 8)
	for i := range all {
		if all[i].IsCustom {
			out = append(out, all[i])
		}
	}
	return out, nil
}

// AvailablePositions liefert die Gliederungspositionen, unter denen ein eigenes
// Konto stehen darf: die, die tatsächlich in Bilanz und GuV einfließen.
func (s *AccountService) AvailablePositions() ([]domain.StatementPositionOption, error) {
	cat, err := accounting.GetSKR04Catalog()
	if err != nil {
		return nil, err
	}
	out := make([]domain.StatementPositionOption, 0, len(cat.Positions))
	for _, p := range cat.Positions {
		if !accounting.HasPositionTarget(p.ID) {
			continue
		}
		out = append(out, domain.StatementPositionOption{
			ID: p.ID, Name: p.Name, StatementType: p.StatementType,
			BalanceSide: p.BalanceSide, HGBCode: p.HGBCode,
			AccountType: p.AccountType,
		})
	}
	return out, nil
}

// ensureFreeNumber prüft, dass die Nummer im freien Bereich des SKR04 liegt.
func (s *AccountService) ensureFreeNumber(
	ctx context.Context, number string, position accounting.SKR04Position,
) error {
	if len(number) != 4 {
		return fmt.Errorf(
			"eine Kontonummer im SKR04 hat vier Stellen. Fünfstellige Nummern sind die Personenkonten "+
				"der Geschäftspartner (%d bis %d) und entstehen mit dem Geschäftspartner",
			domain.DebitorRangeStart, domain.CreditorRangeEnd)
	}
	value, err := strconv.Atoi(number)
	if err != nil || value <= 0 {
		return fmt.Errorf("die Kontonummer %q besteht nicht aus Ziffern", number)
	}
	// Die Klasse 9 (Vortrags- und statistische Konten) bleibt gesperrt: sie
	// enthält den Saldenvortrag und die statistischen Konten, deren Nummern die
	// Auswertungen kennen. Ein eigenes Konto dort träfe früher oder später auf
	// eine Nummer, die Buchfink selbst braucht.
	//
	// Die Klasse 0 ist dagegen die Klasse des Anlagevermögens und keine
	// Programmmechanik. Ein eigenes Anlagenkonto ist ein normaler Fall — ein
	// Betrieb, der eine Anlagenart führt, für die der SKR04 kein Konto
	// vorsieht. Zugelassen wird sie deshalb genau dann, wenn eine Position des
	// Anlagevermögens gewählt ist; sonst stünde ein Aufwandskonto mit einer
	// Anlagennummer in der Bilanz.
	class := value / 1000
	if class == 9 {
		return fmt.Errorf(
			"die Kontonummer %s liegt in der Klasse 9. Sie ist den Vortrags- und statistischen "+
				"Konten vorbehalten — lege das Konto in der Klasse der Aufwendungen, Erträge oder "+
				"Bestände an, zu der es gehört", number)
	}
	if class == 0 && position.Kontenklasse.Number != 0 {
		return fmt.Errorf(
			"die Kontonummer %s liegt in der Klasse 0 (Anlagevermögen), die gewählte Position "+
				"%q gehört aber nicht dorthin. Wähle eine Position des Anlagevermögens oder eine "+
				"Nummer aus der Klasse, zu der das Konto gehört", number, position.Name)
	}
	if s.taxResolver != nil && s.taxResolver.IsTaxAccount(number) {
		return fmt.Errorf(
			"das Konto %s ist ein Steuerkonto und wird nur über die Steuerautomatik bebucht", number)
	}
	if _, ok := domain.CollectiveAccounts()[number]; ok {
		return fmt.Errorf("das Konto %s ist ein Sammelkonto der Bilanz", number)
	}
	existing, err := s.accountRepo.FindByNumber(ctx, number)
	if err == nil && existing != nil {
		return fmt.Errorf(
			"die Kontonummer %s ist mit „%s\" belegt. Wähle eine freie Nummer — ein Konto des "+
				"Kontenrahmens umzuwidmen zerstört die Zuordnung, auf der Bilanz, GuV und E-Bilanz "+
				"beruhen", number, existing.Name)
	}
	return nil
}

// positionByID sucht die Gliederungsposition im Katalog.
func positionByID(id string) (accounting.SKR04Position, error) {
	if id == "" {
		return accounting.SKR04Position{}, fmt.Errorf(
			"ein eigenes Konto braucht seine Position in der Gliederung nach §§ 266, 275 HGB. Ohne " +
				"sie erschiene es weder in der Bilanz noch in der GuV noch in der E-Bilanz")
	}
	cat, err := accounting.GetSKR04Catalog()
	if err != nil {
		return accounting.SKR04Position{}, err
	}
	for _, p := range cat.Positions {
		if p.ID == id {
			return p, nil
		}
	}
	return accounting.SKR04Position{}, fmt.Errorf("die Gliederungsposition %q gibt es nicht", id)
}

// ensureKnownTaxKey weist einen Steuerschlüssel ab, den der Katalog nicht
// kennt: ein erfundener Schlüssel stünde in der Buchung und in keiner
// Kennziffer der Voranmeldung.
func ensureKnownTaxKey(key string) error {
	for _, info := range accounting.TaxKeyCatalog() {
		if info.Key == key {
			return nil
		}
	}
	return fmt.Errorf("den Steuerschlüssel %q gibt es nicht", key)
}

func (s *AccountService) log(ctx context.Context, action domain.AuditAction, account *domain.Account, details string) {
	if s.auditRepo == nil || account == nil {
		return
	}
	_ = s.auditRepo.Log(ctx, action, "ACCOUNT", account.Number, details)
}
