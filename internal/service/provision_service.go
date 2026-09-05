package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// ProvisionService ist die Rückstellungsbuchhaltung (§ 249 HGB).
//
// Eine Rückstellung ist die einzige Bilanzposition, die vollständig aus einer
// Schätzung besteht: es gibt keinen Beleg über ihre Höhe, weil der Vorgang, den
// sie abbildet, noch nicht abgeschlossen ist. Daraus folgt alles, was dieser
// Dienst tut — jede Bewegung trägt ihre Begründung, jede Auflösung nennt den
// weggefallenen Grund, und die Abzinsung rechnet mit einem hinterlegten Satz
// statt mit einem gefundenen.
//
// Die Kartei lebt über die Jahre wie die Anlagenkartei: eine Rückstellung, die
// im Jahr 2026 gebildet und 2028 verbraucht wird, trägt drei Jahre Bewegungen,
// und der Rückstellungsspiegel eines Jahres braucht sie alle.
type ProvisionService struct {
	provisionRepo domain.ProvisionRepository
	rateRepo      domain.DiscountRateRepository
	journalRepo   domain.JournalRepository
	journalSvc    *JournalService
	settingsRepo  domain.SettingsRepository
	auditRepo     domain.AuditRepository
	closingSvc    *ClosingService
	receipts      closingReceiptFiler
	fiscalYear    int
}

// NewProvisionService wires die Rückstellungen.
func NewProvisionService(
	provisionRepo domain.ProvisionRepository,
	rateRepo domain.DiscountRateRepository,
	journalRepo domain.JournalRepository,
	journalSvc *JournalService,
	settingsRepo domain.SettingsRepository,
	auditRepo domain.AuditRepository,
	closingSvc *ClosingService,
	fiscalYear int,
) *ProvisionService {
	return &ProvisionService{
		provisionRepo: provisionRepo,
		rateRepo:      rateRepo,
		journalRepo:   journalRepo,
		journalSvc:    journalSvc,
		settingsRepo:  settingsRepo,
		auditRepo:     auditRepo,
		closingSvc:    closingSvc,
		fiscalYear:    fiscalYear,
	}
}

// SetFiscalYear schaltet das aktive Geschäftsjahr um.
func (s *ProvisionService) SetFiscalYear(year int) { s.fiscalYear = year }

// SetReceiptService gibt dem Dienst den Belegspeicher für die Eigenbelege.
func (s *ProvisionService) SetReceiptService(r closingReceiptFiler) { s.receipts = r }

// -------------------------------------------------------------------------
// Lesen
// -------------------------------------------------------------------------

// List liefert die Rückstellungen, die ein Geschäftsjahr betreffen: die in ihm
// gebildeten und die aus früheren Jahren, die noch bestehen.
func (s *ProvisionService) List(ctx context.Context, year int) ([]domain.Provision, error) {
	if year == 0 {
		year = s.fiscalYear
	}
	all, err := s.liveProvisions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Provision, 0, len(all))
	for i := range all {
		p := &all[i]
		if p.FiscalYear > year {
			continue
		}
		// Aufgenommen wird, was das Jahr betrifft: die in ihm gebildeten, die
		// mit Bestand — und die, die in ihm eine Bewegung hatten. Ohne den
		// letzten Fall fehlte eine 2026 gebildete und 2027 vollständig
		// verbrauchte Rückstellung in der Liste 2027, obwohl der Spiegel
		// desselben Jahres ihren Verbrauch zeigt.
		if p.FiscalYear == year || p.BalanceAt(year) != 0 || movedIn(p, year) {
			s.fillEntryNumbers(ctx, p.Movements)
			out = append(out, *p)
		}
	}
	sortProvisionsByDate(out)
	return out, nil
}

// movedIn meldet, ob die Rückstellung im Geschäftsjahr eine Bewegung hatte.
func movedIn(p *domain.Provision, year int) bool {
	for _, m := range p.Movements {
		if m.FiscalYear == year {
			return true
		}
	}
	return false
}

// liveProvisions sind alle Rückstellungen ohne die Bewegungen, deren Buchung per
// Generalumkehr zurückgenommen wurde.
func (s *ProvisionService) liveProvisions(ctx context.Context) ([]domain.Provision, error) {
	all, err := s.provisionRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	return liveProvisions(ctx, s.journalRepo, all)
}

// Mirror ist der Rückstellungsspiegel eines Geschäftsjahres — Bestandteil des
// Anhangs.
func (s *ProvisionService) Mirror(ctx context.Context, year int) (*accounting.ProvisionMirror, error) {
	if year == 0 {
		year = s.fiscalYear
	}
	all, err := s.liveProvisions(ctx)
	if err != nil {
		return nil, err
	}
	relevant := make([]domain.Provision, 0, len(all))
	for i := range all {
		if all[i].FiscalYear <= year {
			relevant = append(relevant, all[i])
		}
	}
	return accounting.BuildProvisionMirror(relevant, year), nil
}

// -------------------------------------------------------------------------
// Bildung
// -------------------------------------------------------------------------

// ProvisionRequest ist die Bildung oder Erhöhung einer Rückstellung.
type ProvisionRequest struct {
	// ProvisionID ist bei einer Zuführung gesetzt, bei der Bildung leer.
	ProvisionID uint                 `json:"provisionId,omitempty"`
	FiscalYear  int                  `json:"fiscalYear"`
	Kind        domain.ProvisionKind `json:"kind"`
	Text        string               `json:"text"`
	Amount      domain.Cents         `json:"amount"`
	ExpectedOn  string               `json:"expectedOn"`
	Reason      string               `json:"reason"`
	// BalanceAccount und ExpenseAccount überschreiben den Vorschlag aus der Art.
	BalanceAccount string `json:"balanceAccount,omitempty"`
	ExpenseAccount string `json:"expenseAccount,omitempty"`
	// Date ist das Buchungsdatum; leer heißt Bilanzstichtag.
	Date string `json:"date,omitempty"`
}

// ProvisionPreview ist die Vorschau vor der Freigabe.
type ProvisionPreview struct {
	Provision domain.Provision     `json:"provision"`
	Lines     []domain.JournalLine `json:"lines"`
	// SettlementAmount ist der Erfüllungsbetrag, Amount der gebuchte Betrag —
	// bei einer abgezinsten Rückstellung ist das der Barwert.
	SettlementAmount domain.Cents `json:"settlementAmount"`
	Amount           domain.Cents `json:"amount"`
	Discounted       bool         `json:"discounted"`
	DiscountYears    int          `json:"discountYears"`
	DiscountRate     string       `json:"discountRate,omitempty"`
	// DiscountMonth ist der Monat der Zinstabelle, mit der gerechnet wurde.
	DiscountMonth string `json:"discountMonth,omitempty"`
	// TaxAmount ist der steuerliche Wert nach § 6 Abs. 1 Nr. 3a Buchst. e EStG
	// (5,5 %). Er wird nicht gebucht, sondern im Verzeichnis ausgewiesen.
	TaxAmount   domain.Cents `json:"taxAmount"`
	BookingDate string       `json:"bookingDate"`
	// BookingYear ist das Geschäftsjahr des Buchungsdatums — das Jahr, in dem
	// die Bewegung im Rückstellungsspiegel steht. Bei der Bildung ist es das
	// Jahr der Rückstellung, bei einer Zuführung kann es ein späteres sein.
	BookingYear int      `json:"bookingYear"`
	Explanation string   `json:"explanation"`
	Findings    []string `json:"findings"`
	// IsIncrease sagt, dass die Vorschau eine Zuführung zu einer bestehenden
	// Rückstellung rechnet und keine Bildung.
	IsIncrease bool `json:"isIncrease"`
}

// TaxDiscountPermille ist der steuerliche Abzinsungssatz von 5,5 %
// (§ 6 Abs. 1 Nr. 3a Buchst. e EStG). Er steht als Konstante und nicht in der
// Zinstabelle, weil er im Gesetz steht und sich nicht monatlich ändert.
const TaxDiscountPermille = 55

// Preview rechnet eine Bildung oder Zuführung, ohne sie zu buchen.
func (s *ProvisionService) Preview(ctx context.Context, req ProvisionRequest) (*ProvisionPreview, error) {
	year := req.FiscalYear
	if year == 0 {
		year = s.fiscalYear
	}
	fy, err := s.closingSvc.PeriodOf(ctx, year)
	if err != nil {
		return nil, err
	}
	bookingDate := req.Date
	if bookingDate == "" {
		bookingDate = fy.EndDate
	}

	// Die Zuführung ändert die Rückstellung nicht, sie erhöht sie nur: Art,
	// Text, Fälligkeit und Konten stammen aus der bestehenden Rückstellung. Aus
	// dem Antrag eine zweite Rückstellung zu rechnen — mit einem anderen
	// Erfüllungszeitpunkt und damit einem anderen Barwert — hieße, unter dem
	// Namen der Zuführung eine neue zu bilden.
	var existing *domain.Provision
	if req.ProvisionID != 0 {
		existing, err = s.load(ctx, req.ProvisionID)
		if err != nil {
			return nil, err
		}
		req.Kind = existing.Kind
		if strings.TrimSpace(req.Text) == "" {
			req.Text = existing.Text
		}
		req.ExpectedOn = existing.ExpectedDate
		req.BalanceAccount = existing.BalanceAccount
		req.ExpenseAccount = existing.ExpenseAccount
		if strings.TrimSpace(req.Reason) == "" {
			return nil, fmt.Errorf(
				"eine Zuführung ist eine geänderte Schätzung und braucht ihre eigene Begründung: " +
					"woraus folgt, dass der bisherige Betrag nicht mehr reicht")
		}
	}

	balance, expense := accounting.ProvisionAccounts(req.Kind)
	if req.BalanceAccount != "" {
		balance = req.BalanceAccount
	}
	if req.ExpenseAccount != "" {
		expense = req.ExpenseAccount
	}

	provision := domain.Provision{
		ID: req.ProvisionID, FiscalYear: year, Kind: req.Kind, Text: req.Text,
		SettlementAmount: req.Amount, ExpectedDate: req.ExpectedOn,
		DiscountedAmount: req.Amount, BalanceAccount: balance, ExpenseAccount: expense,
		Reason: req.Reason,
	}
	if err := provision.Validate(); err != nil {
		return nil, err
	}

	preview := &ProvisionPreview{
		SettlementAmount: req.Amount, Amount: req.Amount,
		BookingDate: bookingDate,
		BookingYear: domain.GetFiscalYearForDate(bookingDate, s.fiscalYearStartMonth(ctx)),
		Findings:    make([]string, 0), IsIncrease: existing != nil,
	}

	if accounting.NeedsDiscounting(fy.EndDate, req.ExpectedOn) {
		years, err := accounting.RemainingYears(fy.EndDate, req.ExpectedOn)
		if err != nil {
			return nil, err
		}
		preview.DiscountYears = years
		rates, month, err := s.ratesFor(ctx, fy.EndDate)
		if err != nil {
			return nil, err
		}
		average := accounting.DiscountAverageFor(req.Kind)
		rate, ok := accounting.DiscountRateFor(rates, years, average)
		if !ok {
			preview.Findings = append(preview.Findings, fmt.Sprintf(
				"Für eine Restlaufzeit von %d Jahren ist zum %s kein Abzinsungssatz aus dem "+
					"%d-Jahres-Durchschnitt hinterlegt. § 253 Abs. 2 HGB verlangt die Abzinsung; "+
					"Buchfink zinst nicht ab und rät auch nicht. Pflege die Sätze der Deutschen "+
					"Bundesbank ein.",
				years, fy.EndDate, average))
		} else {
			preview.Discounted = true
			preview.DiscountRate = discountRateLabel(rate.RateMicros)
			preview.DiscountMonth = month
			preview.Amount = accounting.PresentValue(req.Amount, years, rate.RateMicros)
			provision.DiscountedAmount = preview.Amount
			provision.DiscountRateMicros = rate.RateMicros
			if month != fy.EndDate[:7] {
				preview.Findings = append(preview.Findings, fmt.Sprintf(
					"Für den Stichtagsmonat %s ist keine Zinstabelle hinterlegt; gerechnet wurde mit "+
						"den Sätzen vom %s. § 253 Abs. 2 Satz 1 HGB meint den Satz zum Bilanzstichtag — "+
						"pflege die Veröffentlichung der Deutschen Bundesbank für %s nach.",
					fy.EndDate[:7], month, fy.EndDate[:7]))
			}
		}
		// Der steuerliche Wert entsteht immer, auch ohne handelsrechtlichen
		// Satz: § 6 Abs. 1 Nr. 3a Buchst. e EStG nennt die 5,5 % im Gesetz.
		preview.TaxAmount = accounting.PresentValue(req.Amount, years, TaxDiscountPermille*1000)
	} else {
		preview.TaxAmount = req.Amount
	}

	if req.Kind == domain.ProvisionPension {
		// Die Pensionsrückstellung ist die eine Art, die Buchfink erfasst, aber
		// nicht bewertet. Handelsrechtlich verlangt § 253 Abs. 1 Satz 2 HGB den
		// Erfüllungsbetrag nach versicherungsmathematischen Grundsätzen,
		// steuerlich rechnet § 6a EStG mit dem Teilwert und 6 % — beides sind
		// Gutachtenrechnungen und keine Formel, die ein Buchführungsprogramm
		// anwenden kann. Deshalb steht hier kein steuerlicher Vergleichswert:
		// die 5,5 % des § 6 Abs. 1 Nr. 3a Buchst. e EStG gelten für Pensionen
		// gerade nicht, und eine damit gerechnete Differenz ginge falsch in die
		// Überleitung ein.
		preview.TaxAmount = req.Amount
		preview.Findings = append(preview.Findings,
			"Pensionsrückstellungen erfasst Buchfink, bewertet sie aber nicht: der Erfüllungsbetrag "+
				"folgt aus einem versicherungsmathematischen Gutachten (§ 253 Abs. 1 Satz 2 HGB), "+
				"abgezinst wird mit dem Zehnjahresdurchschnitt (§ 253 Abs. 2 Satz 2 HGB). Der "+
				"steuerliche Wert nach § 6a EStG (Teilwert, 6 %) wird nicht gerechnet und erscheint "+
				"deshalb nicht in der Überleitung zur Steuerbilanz.")
	}

	if existing != nil {
		// Die Vorschau zeigt die Rückstellung, wie sie nach der Zuführung
		// dasteht: derselbe Datensatz mit gewachsenem Erfüllungsbetrag.
		merged := *existing
		merged.SettlementAmount += req.Amount
		merged.DiscountedAmount += preview.Amount
		if provision.DiscountRateMicros > 0 {
			merged.DiscountRateMicros = provision.DiscountRateMicros
		}
		provision = merged
	}
	if provision.Movements == nil {
		// Eine neu gebildete Rückstellung hat noch keine Bewegung. Als nil ginge
		// die Liste als `null` ans Frontend, und `provision.movements.map` nähme
		// im Render den ganzen Baum mit.
		provision.Movements = make([]domain.ProvisionMovement, 0)
	}
	preview.Provision = provision
	preview.Lines = []domain.JournalLine{
		{Side: domain.SideDebit, Account: expense, Amount: preview.Amount, Text: req.Text},
		{Side: domain.SideCredit, Account: balance, Amount: preview.Amount, Text: req.Text},
	}
	preview.Explanation = provisionExplanation(preview, req)
	return preview, nil
}

func provisionExplanation(preview *ProvisionPreview, req ProvisionRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Erfüllungsbetrag %s € nach § 253 Abs. 1 Satz 2 HGB, erwartet zum %s. ",
		preview.SettlementAmount, req.ExpectedOn)
	if preview.Discounted {
		fmt.Fprintf(&b, "Die Restlaufzeit von %d Jahren übersteigt ein Jahr; abgezinst mit %s "+
			"(§ 253 Abs. 2 Satz 1 HGB) ergibt das einen Barwert von %s €. ",
			preview.DiscountYears, preview.DiscountRate, preview.Amount)
	} else if preview.DiscountYears > 0 {
		b.WriteString("Abgezinst wurde nicht, weil kein Satz hinterlegt ist. ")
	} else {
		b.WriteString("Abgezinst wird nicht: die Restlaufzeit beträgt höchstens ein Jahr. ")
	}
	if preview.TaxAmount != preview.Amount {
		fmt.Fprintf(&b, "Steuerlich wäre mit 5,5 %% abzuzinsen (§ 6 Abs. 1 Nr. 3a Buchst. e EStG); "+
			"das ergäbe %s € und damit eine Differenz von %s €, die im Verzeichnis nach "+
			"§ 5 Abs. 1 Satz 2 EStG ausgewiesen wird.", preview.TaxAmount, preview.Amount-preview.TaxAmount)
	}
	return b.String()
}

// BookFormation bildet eine Rückstellung und bucht die Zuführung.
func (s *ProvisionService) BookFormation(ctx context.Context, req ProvisionRequest) (*domain.Provision, error) {
	if req.Kind.IsTax() && req.ProvisionID == 0 {
		// Die Steuerrückstellung entsteht über ihren eigenen Baustein, der das
		// Ergebnis kennt. Von Hand gebildet trüge sie eine Zahl, die mit der
		// GuV nichts zu tun hat.
		return nil, fmt.Errorf(
			"eine Steuerrückstellung entsteht über den Abschlussbaustein „Steuerrückstellung\": " +
				"er rechnet sie aus dem Ergebnis des Jahres")
	}
	return s.bookProvisionMovement(ctx, req, domain.ProvisionFormation)
}

// BookIncrease führt einer bestehenden Rückstellung Betrag zu.
func (s *ProvisionService) BookIncrease(ctx context.Context, req ProvisionRequest) (*domain.Provision, error) {
	if req.ProvisionID == 0 {
		return nil, fmt.Errorf("zur Zuführung gehört die Rückstellung, der zugeführt wird")
	}
	return s.bookProvisionMovement(ctx, req, domain.ProvisionIncrease)
}

func (s *ProvisionService) bookProvisionMovement(
	ctx context.Context, req ProvisionRequest, kind domain.ProvisionMovementKind,
) (*domain.Provision, error) {
	preview, err := s.Preview(ctx, req)
	if err != nil {
		return nil, err
	}
	if kind == domain.ProvisionIncrease && !preview.IsIncrease {
		return nil, fmt.Errorf("die Rückstellung %d wurde nicht gefunden", req.ProvisionID)
	}
	provision := preview.Provision

	label := "Rückstellung"
	if kind == domain.ProvisionIncrease {
		label = "Zuführung zur Rückstellung"
	}
	entry := &domain.JournalEntry{
		BookingDate:        preview.BookingDate,
		DocumentDate:       preview.BookingDate,
		ServiceDateFrom:    preview.BookingDate,
		ServiceDateTo:      preview.BookingDate,
		Description:        fmt.Sprintf("%s: %s", label, provision.Text),
		Source:             domain.EntrySourceClosing,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              preview.Lines,
	}

	// Beleg und Bewegung gehören in das Geschäftsjahr des Buchungsdatums und
	// nicht in das Bildungsjahr der Rückstellung: eine Zuführung 2027 zu einer
	// Rückstellung von 2026 ist eine Bewegung des Jahres 2027. Stünde sie unter
	// 2026, ginge der Rückstellungsspiegel beider Jahre nicht mehr auf.
	receipt, err := selfIssuedVoucher(ctx, s.receipts, preview.BookingYear, closingVoucher{
		Kind: "rueckstellung", FiscalYear: preview.BookingYear, Date: preview.BookingDate,
		Description: entry.Description, Explanation: preview.Explanation,
		Calculation: preview, Lines: preview.Lines,
	})
	if err != nil {
		return nil, err
	}
	attachVoucher(entry, receipt)

	// Erst buchen, dann speichern. Andersherum bliebe eine Rückstellung ohne
	// Bewegung in der Kartei stehen, sobald die Buchung an der Periodensperre
	// oder an einem festgestellten Jahr scheitert — und der Abschlussassistent
	// meldete den Schritt als erledigt.
	created, err := postWithVoucher(ctx, s.journalSvc, s.receipts, entry, receipt)
	if err != nil {
		return nil, err
	}
	if err := s.provisionRepo.Save(ctx, &provision); err != nil {
		return nil, fmt.Errorf(
			"die Buchung %s wurde geschrieben, die Rückstellung aber nicht gespeichert: %w",
			created.EntryNumber, err)
	}
	movement := &domain.ProvisionMovement{
		ProvisionID: provision.ID, Kind: kind, Date: preview.BookingDate,
		FiscalYear: preview.BookingYear, Amount: preview.Amount,
		Reason: req.Reason, JournalEntryID: &created.ID,
	}
	if err := s.provisionRepo.AddMovement(ctx, movement); err != nil {
		return nil, fmt.Errorf(
			"die Buchung %s wurde geschrieben, die Bewegung aber nicht: %w", created.EntryNumber, err)
	}
	s.audit(ctx, domain.AuditActionCreate, provision.ID, fmt.Sprintf(
		"%s %s über %s € (%s)", label, req.Text, preview.Amount, req.Kind.Label()))
	return s.provisionRepo.FindByID(ctx, provision.ID)
}

// -------------------------------------------------------------------------
// Auflösung, Verbrauch, Aufzinsung
// -------------------------------------------------------------------------

// ProvisionChangeRequest ist eine Auflösung, ein Verbrauch oder eine Aufzinsung.
type ProvisionChangeRequest struct {
	ProvisionID uint         `json:"provisionId"`
	Amount      domain.Cents `json:"amount"`
	Date        string       `json:"date"`
	Reason      string       `json:"reason"`
	// PaymentAccount nimmt beim Verbrauch ohne Rechnung die Zahlung auf.
	PaymentAccount string `json:"paymentAccount,omitempty"`
}

// BookRelease löst eine Rückstellung ganz oder teilweise auf.
//
// Ohne Grund geht das nicht: § 249 Abs. 2 Satz 2 HGB lässt die Auflösung nur zu,
// soweit der Grund dafür entfallen ist. Eine Auflösung ohne festgehaltenen
// Grund wäre von einer Ergebnisglättung nicht zu unterscheiden.
func (s *ProvisionService) BookRelease(ctx context.Context, req ProvisionChangeRequest) (*domain.Provision, error) {
	if strings.TrimSpace(req.Reason) == "" {
		return nil, fmt.Errorf(
			"eine Rückstellung darf nur aufgelöst werden, soweit der Grund dafür entfallen ist " +
				"(§ 249 Abs. 2 Satz 2 HGB). Nenne den weggefallenen Grund")
	}
	provision, err := s.load(ctx, req.ProvisionID)
	if err != nil {
		return nil, err
	}
	amount := req.Amount
	if provision.Balance() <= 0 {
		return nil, fmt.Errorf("die Rückstellung %q steht auf null; es gibt nichts aufzulösen", provision.Text)
	}
	if amount <= 0 {
		amount = provision.Balance()
	}
	// Ein Betrag über dem Bestand wird nicht stillschweigend gekappt: die
	// Auflösung ist begründungspflichtig (§ 249 Abs. 2 Satz 2 HGB), und wer
	// 5.000 statt 500 Euro eintippt, soll das erfahren, nicht die Rückstellung
	// verlieren.
	if amount > provision.Balance() {
		return nil, fmt.Errorf("die Rückstellung %q steht mit %s € zu Buche; mehr lässt sich nicht auflösen",
			provision.Text, provision.Balance())
	}
	date := s.dateOr(ctx, req.Date)

	lines := []domain.JournalLine{
		{Side: domain.SideDebit, Account: provision.BalanceAccount, Amount: amount, Text: provision.Text},
		{Side: domain.SideCredit, Account: domain.AccountErtragAufloesungRueckstellungen,
			Amount: amount, Text: provision.Text},
	}
	entry := &domain.JournalEntry{
		BookingDate: date, DocumentDate: date, ServiceDateFrom: date, ServiceDateTo: date,
		Description:        fmt.Sprintf("Auflösung Rückstellung: %s", provision.Text),
		Source:             domain.EntrySourceClosing,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
	}
	return s.applyMovement(ctx, provision, entry, domain.ProvisionRelease, amount, date, req.Reason,
		fmt.Sprintf("Aufgelöst, weil der Grund entfallen ist: %s", req.Reason))
}

// BookConsumption verbraucht eine Rückstellung gegen ein Zahlungsmittelkonto.
//
// Der Regelfall läuft über den Belegweg — die Rechnung kommt an, wird der
// Rückstellung zugeordnet und bucht gegen sie statt gegen den Aufwand. Dieser
// Weg hier ist der zweite: die Zahlung ohne Eingangsrechnung.
func (s *ProvisionService) BookConsumption(ctx context.Context, req ProvisionChangeRequest) (*domain.Provision, error) {
	provision, err := s.load(ctx, req.ProvisionID)
	if err != nil {
		return nil, err
	}
	if req.PaymentAccount == "" {
		return nil, fmt.Errorf("zum Verbrauch gehört das Konto, von dem gezahlt wurde")
	}
	amount := req.Amount
	if amount <= 0 {
		return nil, fmt.Errorf("der verbrauchte Betrag muss größer als null sein")
	}
	balance := provision.Balance()
	date := s.dateOr(ctx, req.Date)

	lines := make([]domain.JournalLine, 0, 3)
	covered := amount
	if covered > balance {
		covered = balance
	}
	if covered > 0 {
		lines = append(lines, domain.JournalLine{
			Side: domain.SideDebit, Account: provision.BalanceAccount, Amount: covered, Text: provision.Text,
		})
	}
	// Was die Rückstellung nicht deckt, ist Aufwand des laufenden Jahres. Ihn
	// gegen die Rückstellung zu buchen hieße, sie unter null zu bringen.
	if excess := amount - covered; excess > 0 {
		lines = append(lines, domain.JournalLine{
			Side: domain.SideDebit, Account: provision.ExpenseAccount, Amount: excess,
			Text: "Mehraufwand über die Rückstellung hinaus",
		})
	}
	lines = append(lines, domain.JournalLine{
		Side: domain.SideCredit, Account: req.PaymentAccount, Amount: amount, Text: provision.Text,
	})

	entry := &domain.JournalEntry{
		BookingDate: date, DocumentDate: date, ServiceDateFrom: date, ServiceDateTo: date,
		Description:        fmt.Sprintf("Verbrauch Rückstellung: %s", provision.Text),
		Source:             domain.EntrySourceClosing,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
	}
	return s.applyMovement(ctx, provision, entry, domain.ProvisionConsumption, covered, date, req.Reason,
		fmt.Sprintf("Verbraucht mit %s €", amount))
}

// BookUnwinding bucht die Aufzinsung: der Barwert wächst, weil die Fälligkeit
// näher rückt.
func (s *ProvisionService) BookUnwinding(ctx context.Context, req ProvisionChangeRequest) (*domain.Provision, error) {
	provision, err := s.load(ctx, req.ProvisionID)
	if err != nil {
		return nil, err
	}
	date := s.dateOr(ctx, req.Date)
	amount := req.Amount
	if amount <= 0 {
		// Ohne Vorgabe rechnet Buchfink die Aufzinsung selbst: der Barwert zum
		// neuen Stichtag abzüglich des Bestands.
		years, err := accounting.RemainingYears(date, provision.ExpectedDate)
		if err != nil {
			return nil, err
		}
		// Nach einem Teilverbrauch oder einer Teilauflösung steht nur noch ein
		// Teil des Erfüllungsbetrags zu Buche. Die Aufzinsung gilt diesem
		// Rest, nicht dem ursprünglichen Betrag — sonst füllte der Zinsaufwand
		// den verbrauchten Teil wieder auf.
		settlement := provision.SettlementAmount
		if provision.DiscountedAmount > 0 && provision.Balance() < provision.DiscountedAmount {
			settlement = domain.Cents(int64(provision.SettlementAmount) * int64(provision.Balance()) / int64(provision.DiscountedAmount))
		}
		target := settlement
		if years > 0 && provision.DiscountRateMicros > 0 {
			target = accounting.PresentValue(settlement, years, provision.DiscountRateMicros)
		}
		amount = target - provision.Balance()
	}
	if amount <= 0 {
		return nil, fmt.Errorf("zum %s ist keine Aufzinsung offen", date)
	}
	lines := []domain.JournalLine{
		{Side: domain.SideDebit, Account: domain.AccountZinsaufwandAbzinsung, Amount: amount, Text: provision.Text},
		{Side: domain.SideCredit, Account: provision.BalanceAccount, Amount: amount, Text: provision.Text},
	}
	entry := &domain.JournalEntry{
		BookingDate: date, DocumentDate: date, ServiceDateFrom: date, ServiceDateTo: date,
		Description:        fmt.Sprintf("Aufzinsung Rückstellung: %s", provision.Text),
		Source:             domain.EntrySourceClosing,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
	}
	return s.applyMovement(ctx, provision, entry, domain.ProvisionUnwinding, amount, date, req.Reason,
		"Aufzinsung nach § 253 Abs. 2 HGB")
}

// applyMovement bucht und schreibt die Bewegung fort.
func (s *ProvisionService) applyMovement(
	ctx context.Context, provision *domain.Provision, entry *domain.JournalEntry,
	kind domain.ProvisionMovementKind, amount domain.Cents, date, reason, note string,
) (*domain.Provision, error) {
	bookingYear := domain.GetFiscalYearForDate(date, s.fiscalYearStartMonth(ctx))
	receipt, err := selfIssuedVoucher(ctx, s.receipts, bookingYear, closingVoucher{
		Kind: "rueckstellung-" + string(kind), FiscalYear: bookingYear, Date: date,
		Description: entry.Description, Explanation: note,
		Calculation: map[string]any{"rueckstellung": provision.Text, "betrag": amount, "grund": reason},
		Lines:       entry.Lines,
	})
	if err != nil {
		return nil, err
	}
	attachVoucher(entry, receipt)

	created, err := postWithVoucher(ctx, s.journalSvc, s.receipts, entry, receipt)
	if err != nil {
		return nil, err
	}
	movement := &domain.ProvisionMovement{
		ProvisionID: provision.ID, Kind: kind, Date: date,
		FiscalYear: bookingYear,
		Amount:     amount, Reason: reason, JournalEntryID: &created.ID,
	}
	if err := s.provisionRepo.AddMovement(ctx, movement); err != nil {
		return nil, fmt.Errorf(
			"die Buchung %s wurde geschrieben, die Bewegung aber nicht: %w", created.EntryNumber, err)
	}
	provision.Movements = append(provision.Movements, *movement)
	if provision.Balance() <= 0 && provision.SettledOn == "" {
		provision.SettledOn = date
	}
	if err := s.provisionRepo.Save(ctx, provision); err != nil {
		return nil, err
	}
	s.audit(ctx, domain.AuditActionUpdate, provision.ID, fmt.Sprintf(
		"%s der Rückstellung %q über %s €: %s", kind.Label(), provision.Text, amount, reason))
	return s.provisionRepo.FindByID(ctx, provision.ID)
}

// -------------------------------------------------------------------------
// Verbrauch aus dem Belegweg
// -------------------------------------------------------------------------

// ConsumptionSplit sagt, wie viel eine Rückstellung von einem Rechnungsbetrag
// deckt und auf welches Konto der gedeckte Teil geht.
//
// Sie ist der Ausschnitt, den der Belegweg braucht: die Eingangsrechnung, die
// einer Rückstellung zugeordnet wird, bucht gegen das Rückstellungskonto statt
// gegen den Aufwand — bis zur Höhe des Bestands, der Rest bleibt Aufwand.
func (s *ProvisionService) ConsumptionSplit(
	ctx context.Context, provisionID uint, net domain.Cents,
) (string, domain.Cents, error) {
	provision, err := s.load(ctx, provisionID)
	if err != nil {
		return "", 0, err
	}
	covered := provision.Balance()
	if covered > net {
		covered = net
	}
	if covered <= 0 {
		return "", 0, fmt.Errorf(
			"die Rückstellung %q steht auf null und kann nichts mehr aufnehmen", provision.Text)
	}
	return provision.BalanceAccount, covered, nil
}

// RecordConsumption hält den Verbrauch fest, den der Belegweg gebucht hat.
func (s *ProvisionService) RecordConsumption(
	ctx context.Context, provisionID uint, amount domain.Cents, date string, entryID uint, reason string,
) error {
	provision, err := s.load(ctx, provisionID)
	if err != nil {
		return err
	}
	movement := &domain.ProvisionMovement{
		ProvisionID: provisionID, Kind: domain.ProvisionConsumption, Date: date,
		FiscalYear: domain.GetFiscalYearForDate(date, s.fiscalYearStartMonth(ctx)),
		Amount:     amount, Reason: reason, JournalEntryID: &entryID,
	}
	if err := s.provisionRepo.AddMovement(ctx, movement); err != nil {
		return err
	}
	provision.Movements = append(provision.Movements, *movement)
	if provision.Balance() <= 0 && provision.SettledOn == "" {
		provision.SettledOn = date
	}
	if err := s.provisionRepo.Save(ctx, provision); err != nil {
		return err
	}
	// Auch der Verbrauch aus dem Belegweg gehört ins Protokoll: er mindert eine
	// Bilanzposition, und ohne Eintrag wäre später nicht zu sehen, wer sie
	// wann gemindert hat.
	s.audit(ctx, domain.AuditActionUpdate, provisionID, fmt.Sprintf(
		"Verbrauch der Rückstellung %q über %s € aus dem Belegweg: %s",
		provision.Text, amount, reason))
	return nil
}

// Settle erledigt eine Rückstellung: der Rest wird aufgelöst und die
// Rückstellung geschlossen.
//
// Der Weg fehlte bisher. Eine Rückstellung, deren Rechnung kleiner ausgefallen
// ist als geschätzt, bleibt sonst mit einem Restbetrag stehen, den niemand mehr
// anfasst — und ein Restbestand ohne Grund ist nach § 249 Abs. 2 Satz 2 HGB
// genau das, was nicht bleiben darf.
func (s *ProvisionService) Settle(
	ctx context.Context, provisionID uint, date, reason string,
) (*domain.Provision, error) {
	provision, err := s.load(ctx, provisionID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf(
			"zum Erledigen gehört der Grund: der Rest wird aufgelöst, und eine Auflösung ohne " +
				"festgehaltenen Grund ist von einer Ergebnisglättung nicht zu unterscheiden " +
				"(§ 249 Abs. 2 Satz 2 HGB)")
	}
	when := s.dateOr(ctx, date)
	if rest := provision.Balance(); rest > 0 {
		return s.BookRelease(ctx, ProvisionChangeRequest{
			ProvisionID: provisionID, Amount: rest, Date: when, Reason: reason,
		})
	}
	if provision.SettledOn == "" {
		provision.SettledOn = when
		if err := s.provisionRepo.Save(ctx, provision); err != nil {
			return nil, err
		}
		s.audit(ctx, domain.AuditActionUpdate, provisionID, fmt.Sprintf(
			"Rückstellung %q zum %s erledigt: %s", provision.Text, when, reason))
	}
	return s.provisionRepo.FindByID(ctx, provisionID)
}

// DiscountFindings meldet die Rückstellungen eines Jahres, bei denen die
// Abzinsung nicht mit dem Satz des Stichtagsmonats gerechnet werden konnte.
//
// Der Befund gehört in den Prüflauf und nicht nur in die Vorschau: gebildet
// wird eine Rückstellung einmal, geprüft wird das Jahr am Ende — und bis dahin
// hat niemand die Vorschau mehr vor Augen.
func (s *ProvisionService) DiscountFindings(
	ctx context.Context, year int,
) ([]domain.CheckFinding, error) {
	if year == 0 {
		year = s.fiscalYear
	}
	fy, err := s.closingSvc.PeriodOf(ctx, year)
	if err != nil {
		return nil, err
	}
	provisions, err := s.liveProvisions(ctx)
	if err != nil {
		return nil, err
	}
	rates, month, err := s.ratesFor(ctx, fy.EndDate)
	if err != nil {
		return nil, err
	}
	out := make([]domain.CheckFinding, 0)
	for i := range provisions {
		p := &provisions[i]
		if p.FiscalYear > year || p.BalanceAt(year) <= 0 {
			continue
		}
		if !accounting.NeedsDiscounting(fy.EndDate, p.ExpectedDate) {
			continue
		}
		years, err := accounting.RemainingYears(fy.EndDate, p.ExpectedDate)
		if err != nil || years <= 0 {
			continue
		}
		average := accounting.DiscountAverageFor(p.Kind)
		_, ok := accounting.DiscountRateFor(rates, years, average)
		switch {
		case !ok:
			out = append(out, domain.CheckFinding{
				Rule:       domain.CheckRuleProvisionDiscount,
				Severity:   domain.CheckWarning,
				ObjectType: "PROVISION",
				ObjectID:   fmt.Sprintf("%d", p.ID),
				ObjectName: p.Text,
				Message: fmt.Sprintf(
					"Die Rückstellung %q läuft noch %d Jahre; für diese Restlaufzeit ist zum %s kein "+
						"Abzinsungssatz aus dem %d-Jahres-Durchschnitt hinterlegt. Abgezinst wurde "+
						"deshalb nicht",
					p.Text, years, fy.EndDate, average),
				Reference: "§ 253 Abs. 2 HGB",
			})
		case month != fy.EndDate[:7]:
			out = append(out, domain.CheckFinding{
				Rule:       domain.CheckRuleProvisionDiscount,
				Severity:   domain.CheckWarning,
				ObjectType: "PROVISION",
				ObjectID:   fmt.Sprintf("%d", p.ID),
				ObjectName: p.Text,
				Message: fmt.Sprintf(
					"Für den Stichtagsmonat %s fehlt die Zinstabelle; die Rückstellung %q wurde mit "+
						"den Sätzen vom %s abgezinst", fy.EndDate[:7], p.Text, month),
				Reference: "§ 253 Abs. 2 Satz 1 HGB",
			})
		}
	}
	return out, nil
}

// -------------------------------------------------------------------------
// Zinssätze
// -------------------------------------------------------------------------

// DiscountRates liefert die Sätze eines Monats; leerer Monat heißt: die
// jüngsten, die es gibt.
func (s *ProvisionService) DiscountRates(ctx context.Context, month string) ([]domain.DiscountRate, error) {
	if month == "" {
		return s.rateRepo.FindLatestUpTo(ctx, time.Now().Format("2006-01"))
	}
	return s.rateRepo.FindByMonth(ctx, month)
}

// DiscountRateMonths nennt die Monate, für die Sätze hinterlegt sind.
func (s *ProvisionService) DiscountRateMonths(ctx context.Context) ([]string, error) {
	return s.rateRepo.Months(ctx)
}

// SaveDiscountRates schreibt gepflegte Sätze fort.
func (s *ProvisionService) SaveDiscountRates(ctx context.Context, rates []domain.DiscountRate) error {
	for i := range rates {
		if rates[i].Average == 0 {
			rates[i].Average = 7
		}
		if err := rates[i].Validate(); err != nil {
			return err
		}
	}
	if err := s.rateRepo.Save(ctx, rates); err != nil {
		return err
	}
	s.audit(ctx, domain.AuditActionUpdate, 0, fmt.Sprintf("%d Abzinsungssätze gepflegt", len(rates)))
	return nil
}

// ImportDiscountRatesCSV liest die Veröffentlichung der Deutschen Bundesbank.
//
// Erwartet werden zwei Spalten — Restlaufzeit in Jahren und Zinssatz —, mit
// Semikolon oder Komma getrennt; der Satz darf deutsch (1,49) oder englisch
// (1.49) geschrieben sein. Mehr Format zu verlangen wäre unrealistisch: die
// Datei wird von Hand aus einer Tabelle exportiert, und ihre Spaltenzahl hängt
// davon ab, wer sie exportiert hat.
func (s *ProvisionService) ImportDiscountRatesCSV(
	ctx context.Context, path, month string, average int,
) (int, error) {
	if month == "" {
		return 0, fmt.Errorf("zu den Zinssätzen gehört der Monat ihrer Veröffentlichung (JJJJ-MM)")
	}
	if average == 0 {
		average = 7
	}
	file, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("die Datei %s konnte nicht gelesen werden: %w", path, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.Comma = ';'
	records, err := reader.ReadAll()
	if err != nil {
		// Zweiter Versuch mit dem Komma als Trenner: eine Datei, die als CSV
		// im englischen Sinn exportiert wurde, ist genauso häufig.
		if _, seekErr := file.Seek(0, 0); seekErr != nil {
			return 0, fmt.Errorf("die Datei %s konnte nicht gelesen werden: %w", path, err)
		}
		reader = csv.NewReader(file)
		reader.FieldsPerRecord = -1
		records, err = reader.ReadAll()
		if err != nil {
			return 0, fmt.Errorf("die Datei %s ist keine lesbare CSV-Datei: %w", path, err)
		}
	}

	rates := make([]domain.DiscountRate, 0, len(records))
	for _, record := range records {
		if len(record) < 2 {
			continue
		}
		years, err := strconv.Atoi(strings.TrimSpace(record[0]))
		if err != nil || years < 1 || years > 50 {
			continue
		}
		micros, ok := parseRateMicros(record[1])
		if !ok {
			continue
		}
		rates = append(rates, domain.DiscountRate{
			Month: month, Years: years, RateMicros: micros, Average: average,
		})
	}
	if len(rates) == 0 {
		return 0, fmt.Errorf(
			"in %s wurde keine Zeile mit Restlaufzeit und Zinssatz gefunden. Erwartet werden zwei "+
				"Spalten: Jahre und Satz in Prozent", path)
	}
	if err := s.SaveDiscountRates(ctx, rates); err != nil {
		return 0, err
	}
	return len(rates), nil
}

// parseRateMicros liest einen Prozentsatz in Millionstel: „1,49" wird zu 14900.
func parseRateMicros(value string) (int64, bool) {
	cleaned := strings.TrimSpace(strings.ReplaceAll(value, "%", ""))
	cleaned = strings.ReplaceAll(cleaned, ",", ".")
	if cleaned == "" {
		return 0, false
	}
	percent, err := strconv.ParseFloat(cleaned, 64)
	if err != nil || percent < 0 {
		return 0, false
	}
	return int64(percent*accounting.DiscountScale/100 + 0.5), true
}

func discountRateLabel(micros int64) string {
	percent := float64(micros) * 100 / accounting.DiscountScale
	return strings.Replace(strconv.FormatFloat(percent, 'f', 2, 64), ".", ",", 1) + " %"
}

// ratesFor liefert die Zinstabelle zum Stichtag und den Monat, aus dem sie
// stammt.
//
// Gesucht wird zuerst der Monat des Stichtags — § 253 Abs. 2 Satz 1 HGB meint
// den Satz, der am Bilanzstichtag gilt. Gibt es ihn nicht, greift Buchfink auf
// den jüngsten älteren Monat zurück, nennt ihn aber: mit einem drei Monate
// alten Satz gerechnet zu haben ist etwas anderes, als mit dem des Stichtags
// gerechnet zu haben, und der Unterschied darf nicht im Stillen verschwinden.
func (s *ProvisionService) ratesFor(
	ctx context.Context, cutoff string,
) (rates []domain.DiscountRate, month string, err error) {
	if s.rateRepo == nil || len(cutoff) < 7 {
		return nil, "", nil
	}
	wanted := cutoff[:7]
	rates, err = s.rateRepo.FindByMonth(ctx, wanted)
	if err == nil && len(rates) > 0 {
		return rates, wanted, nil
	}
	rates, err = s.rateRepo.FindLatestUpTo(ctx, wanted)
	if err != nil || len(rates) == 0 {
		return nil, "", err
	}
	return rates, rates[0].Month, nil
}

// -------------------------------------------------------------------------
// Hilfen
// -------------------------------------------------------------------------

// load liest die Rückstellung so, wie sie im Journal steht.
//
// Die stornierten Bewegungen bleiben dabei außen vor: eine per Generalumkehr
// zurückgenommene Bildung ist buchhalterisch nie geschehen, und ein Bestand,
// der nur noch in der Kartei steht, dürfte weder aufgelöst noch verbraucht noch
// aufgezinst werden. Die Lesepfade — Liste, Spiegel — rechnen längst so; die
// Schreibpfade müssen es auch, sonst entstünde aus einem stornierten Bestand
// ein Ertrag aus der Auflösung von etwas, das es nicht gibt.
func (s *ProvisionService) load(ctx context.Context, id uint) (*domain.Provision, error) {
	provision, err := s.provisionRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if provision == nil {
		return nil, fmt.Errorf("die Rückstellung %d wurde nicht gefunden", id)
	}
	live, err := liveProvisions(ctx, s.journalRepo, []domain.Provision{*provision})
	if err != nil {
		return nil, err
	}
	if len(live) == 0 {
		return nil, fmt.Errorf(
			"die Rückstellung %q besteht nicht mehr: alle ihre Buchungen sind storniert. "+
				"Bilde sie neu, wenn sie weiter gebraucht wird", provision.Text)
	}
	return &live[0], nil
}

func (s *ProvisionService) dateOr(ctx context.Context, date string) string {
	if date != "" {
		return date
	}
	if fy, err := s.closingSvc.PeriodOf(ctx, s.fiscalYear); err == nil {
		return fy.EndDate
	}
	return time.Now().Format("2006-01-02")
}

func (s *ProvisionService) fiscalYearStartMonth(ctx context.Context) int {
	if s.settingsRepo == nil {
		return 1
	}
	settings, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil || settings == nil || settings.FiscalYearStartMonth <= 0 {
		return 1
	}
	return settings.FiscalYearStartMonth
}

// fillEntryNumbers trägt die Buchungsnummern nach: die Bewegung speichert die
// Kennung, und eine Kennung sagt in einer Tabelle niemandem etwas.
func (s *ProvisionService) fillEntryNumbers(ctx context.Context, movements []domain.ProvisionMovement) {
	for i := range movements {
		if movements[i].JournalEntryID == nil {
			continue
		}
		entry, err := s.journalRepo.FindByID(ctx, *movements[i].JournalEntryID)
		if err == nil && entry != nil {
			movements[i].EntryNumber = entry.EntryNumber
		}
	}
}

func (s *ProvisionService) audit(ctx context.Context, action domain.AuditAction, id uint, details string) {
	if s.auditRepo == nil {
		return
	}
	_ = s.auditRepo.Log(ctx, action, "PROVISION", fmt.Sprintf("%d", id), details)
}

// sortProvisionsByDate ordnet die Rückstellungen nach ihrem erwarteten
// Erfüllungszeitpunkt — die dringendste zuerst.
func sortProvisionsByDate(provisions []domain.Provision) {
	sort.SliceStable(provisions, func(i, j int) bool {
		return provisions[i].ExpectedDate < provisions[j].ExpectedDate
	})
}
