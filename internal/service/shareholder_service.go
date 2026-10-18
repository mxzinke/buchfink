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

// ShareholderService führt die Vorgänge zwischen der GmbH und ihren
// Gesellschaftern: Darlehen in beide Richtungen mit Zinsbuchung, die Salden
// der Verrechnungs- und Darlehenskonten je Gesellschafter und die Befunde zum
// Fremdvergleich. Befunde sperren nichts; ob eine verdeckte
// Gewinnausschüttung vorliegt, ist eine Wertungsfrage (§ 8 Abs. 3 Satz 2 KStG,
// R 8.5 KStR 2022).
type ShareholderService struct {
	loans        domain.ShareholderLoanRepository
	journalRepo  domain.JournalRepository
	contactRepo  domain.ContactRepository
	settingsRepo domain.SettingsRepository
	auditRepo    domain.AuditRepository
	journalSvc   *JournalService
	posting      *PostingService
	receipts     *ReceiptService
}

// NewShareholderService verdrahtet den Dienst.
func NewShareholderService(
	loans domain.ShareholderLoanRepository,
	journalRepo domain.JournalRepository,
	contactRepo domain.ContactRepository,
	settingsRepo domain.SettingsRepository,
	auditRepo domain.AuditRepository,
	journalSvc *JournalService,
	posting *PostingService,
	receipts *ReceiptService,
) *ShareholderService {
	return &ShareholderService{
		loans: loans, journalRepo: journalRepo, contactRepo: contactRepo,
		settingsRepo: settingsRepo, auditRepo: auditRepo,
		journalSvc: journalSvc, posting: posting, receipts: receipts,
	}
}

// shareholderAccounts sind die Konten, deren Salden je Gesellschafter geführt
// werden (§ 42 Abs. 3 GmbHG). Die offene Ausschüttung (3519) fehlt: der
// Beschluss bucht sie in einer Summe.
var shareholderAccounts = []string{
	domain.AccountAusleihungenGesellschafter, "0961",
	"1250", "1251", "1255",
	domain.AccountForderungenGesellschafter, domain.AccountForderungenGesellschafterBis1,
	domain.AccountForderungenGesellschafterUeber1,
	"1330", "1331", "1335",
	"3340", "3341", "3345", "3348",
	"3510", "3511", "3514", "3517",
	domain.AccountVerbindlichkeitenGesellschafter, domain.AccountVerbGesellschafterBis1,
	domain.AccountVerbGesellschafter1Bis5, domain.AccountVerbGesellschafterUeber5,
}

func isShareholderAccount(account string) bool {
	for _, a := range shareholderAccounts {
		if a == account {
			return true
		}
	}
	return false
}

// loanAccounts sind die zulässigen Darlehenskonten je Richtung.
var loanAccounts = map[domain.LoanDirection][]string{
	domain.LoanToCompany: {
		domain.AccountVerbindlichkeitenGesellschafter, domain.AccountVerbGesellschafterBis1,
		domain.AccountVerbGesellschafter1Bis5, domain.AccountVerbGesellschafterUeber5,
	},
	domain.LoanToShareholder: {
		domain.AccountAusleihungenGesellschafter, domain.AccountForderungenGesellschafter,
		domain.AccountForderungenGesellschafterBis1, domain.AccountForderungenGesellschafterUeber1,
	},
}

// LoanRequest ist ein neuer Darlehensvertrag mit einem Gesellschafter.
type LoanRequest struct {
	ContactID      uint                 `json:"contactId"`
	Direction      domain.LoanDirection `json:"direction"`
	Principal      domain.Cents         `json:"principal"`
	InterestRateBP int                  `json:"interestRateBp"`
	DayCount       domain.DayCount      `json:"dayCount,omitempty"`
	ContractDate   string               `json:"contractDate,omitempty"`
	StartDate      string               `json:"startDate"`
	MaturityDate   string               `json:"maturityDate,omitempty"`
	// Account ist das Darlehenskonto; leer heißt nach Richtung und
	// vereinbarter Laufzeit.
	Account string `json:"account,omitempty"`
	// InterestAccount ist das Zinskonto; leer heißt 7110 für Zinsen an die
	// GmbH, sonst 7316 bzw. 7317 nach der Beteiligungsquote.
	InterestAccount   string `json:"interestAccount,omitempty"`
	ContractReceiptID uint   `json:"contractReceiptId,omitempty"`
	Collateral        string `json:"collateral,omitempty"`
	ComparisonRateBP  *int   `json:"comparisonRateBp,omitempty"`
	ComparisonSource  string `json:"comparisonSource,omitempty"`
	Subordinated      bool   `json:"subordinated,omitempty"`
	Note              string `json:"note,omitempty"`
}

// LoanView ist ein Darlehen mit seinem Stand zu einem Stichtag.
type LoanView struct {
	Loan        domain.ShareholderLoan `json:"loan"`
	ContactName string                 `json:"contactName"`
	// Outstanding ist der gebuchte Bestand zum Stichtag.
	Outstanding domain.Cents `json:"outstanding"`
	// FirstMovement ist der Tag der ersten Auszahlung im Journal.
	FirstMovement string `json:"firstMovement,omitempty"`
	// InterestBookedThrough ist das Ende der zuletzt gebuchten Zinsperiode.
	InterestBookedThrough string                `json:"interestBookedThrough,omitempty"`
	Findings              []domain.CheckFinding `json:"findings"`
}

// RegisterLoan legt einen Darlehensvertrag an. Konto und Zinskonto folgen aus
// Richtung, Laufzeit und Beteiligung, wenn sie nicht angegeben sind.
// Unübliche Konditionen sperren nichts; sie stehen als Befund am Ergebnis.
func (s *ShareholderService) RegisterLoan(ctx context.Context, req LoanRequest) (*LoanView, error) {
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (*LoanView, error) {
		loan, contact, cfg, err := s.loanFrom(ctx, req)
		if err != nil {
			return nil, err
		}
		if err := s.loans.Save(ctx, loan); err != nil {
			return nil, fmt.Errorf("der Darlehensvertrag ließ sich nicht speichern: %w", err)
		}
		if err := logAudit(ctx, s.auditRepo, domain.AuditActionCreate, "SHAREHOLDER_LOAN", fmt.Sprintf("%d", loan.ID),
			fmt.Sprintf("Gesellschafterdarlehen %s %s über %s € zu %s ab %s erfasst (Konto %s, Zinsen %s)",
				directionLabel(loan.Direction), contact.LegalName(), loan.Principal, domain.RateLabel(loan.InterestRateBP),
				domain.GermanDate(loan.StartDate), loan.Account, loan.InterestAccount)); err != nil {
			return nil, err
		}
		return s.view(ctx, loan, contact, cfg, loan.StartDate)
	})
}

// CorrectLoan berichtigt einen erfassten Darlehensvertrag oder hält eine
// Änderung fest, etwa eine Kündigung über die Fälligkeit. Vorher und nachher
// stehen mit dem Grund im Änderungsprotokoll. Sind Zinsen gebucht, bleiben die
// Angaben, aus denen sie gerechnet sind; sie ändern sich erst nach dem Storno
// der Zinsbuchungen.
func (s *ShareholderService) CorrectLoan(ctx context.Context, id uint, req LoanRequest, reason string) (*LoanView, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, fmt.Errorf("eine Berichtigung des Darlehensvertrags braucht einen Grund")
	}
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (*LoanView, error) {
		before, loan, contact, cfg, err := s.correctedLoan(ctx, id, req)
		if err != nil {
			return nil, err
		}
		if err := s.loans.Save(ctx, loan); err != nil {
			return nil, fmt.Errorf("der Darlehensvertrag ließ sich nicht speichern: %w", err)
		}
		if err := logAuditChange(ctx, s.auditRepo, domain.AuditActionUpdate, "SHAREHOLDER_LOAN", fmt.Sprintf("%d", loan.ID),
			fmt.Sprintf("Gesellschafterdarlehen %s %s berichtigt: %s", directionLabel(loan.Direction),
				contact.LegalName(), reason), before, loan); err != nil {
			return nil, err
		}
		return s.view(ctx, loan, contact, cfg, loan.StartDate)
	})
}

// PreviewLoan prüft einen Vertrag wie RegisterLoan oder, mit id, wie
// CorrectLoan und rechnet Konten, Stand und Befunde, ohne zu speichern.
func (s *ShareholderService) PreviewLoan(ctx context.Context, id uint, req LoanRequest) (*LoanView, error) {
	if id != 0 {
		_, loan, contact, cfg, err := s.correctedLoan(ctx, id, req)
		if err != nil {
			return nil, err
		}
		return s.view(ctx, loan, contact, cfg, loan.StartDate)
	}
	loan, contact, cfg, err := s.loanFrom(ctx, req)
	if err != nil {
		return nil, err
	}
	return s.view(ctx, loan, contact, cfg, loan.StartDate)
}

// correctedLoan ist der berichtigte Vertrag neben dem bisherigen. Sind Zinsen
// gebucht, weist es eine Änderung der Angaben ab, aus denen sie gerechnet sind.
func (s *ShareholderService) correctedLoan(
	ctx context.Context, id uint, req LoanRequest,
) (*domain.ShareholderLoan, *domain.ShareholderLoan, *domain.Contact, *domain.CompanySettings, error) {
	before, err := s.loans.FindByID(ctx, id)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("Darlehen %d wurde nicht gefunden: %w", id, err)
	}
	loan, contact, cfg, err := s.loanFrom(ctx, req)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	loan.ID, loan.CreatedAt = before.ID, before.CreatedAt
	booked, err := s.bookedInterest(ctx, id)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if len(booked) > 0 {
		through := ""
		for _, p := range booked {
			if p.PeriodTo > through {
				through = p.PeriodTo
			}
		}
		if changed := interestTermsChanged(before, loan); changed != "" {
			return nil, nil, nil, nil, fmt.Errorf("für das Darlehen sind Zinsen bis %s gebucht; %s ändert sich erst, "+
				"wenn die Zinsbuchungen storniert sind", domain.GermanDate(through), changed)
		}
		if loan.MaturityDate != "" && loan.MaturityDate < through {
			return nil, nil, nil, nil, fmt.Errorf("die Fälligkeit %s liegt vor dem Ende der gebuchten Zinsen am %s",
				domain.GermanDate(loan.MaturityDate), domain.GermanDate(through))
		}
	}
	return before, loan, contact, cfg, nil
}

// interestTermsChanged nennt die Angaben, aus denen gebuchte Zinsen gerechnet
// sind und die sich geändert haben; leer, wenn keine.
func interestTermsChanged(before, after *domain.ShareholderLoan) string {
	var changed []string
	for _, c := range []struct {
		label string
		diff  bool
	}{
		{"der Gesellschafter", before.ContactID != after.ContactID},
		{"die Richtung", before.Direction != after.Direction},
		{"der Zinssatz", before.InterestRateBP != after.InterestRateBP},
		{"die Zinsmethode", before.DayCount != after.DayCount},
		{"der Beginn", before.StartDate != after.StartDate},
		{"das Darlehenskonto", before.Account != after.Account},
		{"das Zinskonto", before.InterestAccount != after.InterestAccount},
	} {
		if c.diff {
			changed = append(changed, c.label)
		}
	}
	return strings.Join(changed, ", ")
}

// loanFrom baut einen geprüften Vertrag aus der Anfrage.
func (s *ShareholderService) loanFrom(
	ctx context.Context, req LoanRequest,
) (*domain.ShareholderLoan, *domain.Contact, *domain.CompanySettings, error) {
	contact, err := s.contactRepo.FindByID(ctx, req.ContactID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("der Gesellschafter %d wurde nicht gefunden: %w", req.ContactID, err)
	}
	if err := ensureNotBlocked(contact); err != nil {
		return nil, nil, nil, err
	}
	cfg, err := s.companySettings(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	role := companyRoleOf(cfg, contact)
	if !role.shareholder {
		return nil, nil, nil, fmt.Errorf(
			"%s steht nicht in der Gesellschafterliste der Unternehmensdaten. Ein Gesellschafterdarlehen "+
				"setzt einen Gesellschafter voraus; ein Darlehen eines Dritten ist ein gewöhnliches Darlehen",
			contact.LegalName())
	}
	loan := &domain.ShareholderLoan{
		ContactID: contact.ID, Direction: req.Direction, Principal: req.Principal,
		InterestRateBP: req.InterestRateBP, DayCount: req.DayCount,
		ContractDate: req.ContractDate, StartDate: req.StartDate, MaturityDate: req.MaturityDate,
		Collateral: strings.TrimSpace(req.Collateral), ComparisonRateBP: req.ComparisonRateBP,
		ComparisonSource: strings.TrimSpace(req.ComparisonSource), Subordinated: req.Subordinated,
		Note: strings.TrimSpace(req.Note),
	}
	if loan.DayCount == "" {
		loan.DayCount = domain.DayCountAct365
	}
	if err := loan.Validate(); err != nil {
		return nil, nil, nil, err
	}
	if loan.Account, err = loanAccount(loan, req.Account); err != nil {
		return nil, nil, nil, err
	}
	if loan.InterestAccount, err = interestAccount(loan, req.InterestAccount, role); err != nil {
		return nil, nil, nil, err
	}
	if req.ContractReceiptID != 0 {
		receipt, err := s.receipts.Get(ctx, req.ContractReceiptID)
		if err != nil {
			return nil, nil, nil, err
		}
		if receipt.Status == domain.ReceiptStatusDiscarded {
			return nil, nil, nil, fmt.Errorf("Beleg %s wurde verworfen und kann den Vertrag nicht belegen", receipt.ReceiptNumber)
		}
		id := receipt.ID
		loan.ContractReceiptID = &id
	}
	return loan, contact, cfg, nil
}

func directionLabel(d domain.LoanDirection) string {
	if d == domain.LoanToCompany {
		return "von"
	}
	return "an"
}

// loanAccount wählt das Darlehenskonto nach vereinbarter Laufzeit: bis ein
// Jahr, bis fünf Jahre, darüber (§ 268 Abs. 4 und 5 HGB). Ohne bestimmte
// Laufzeit ist das Darlehen mit drei Monaten kündbar (§ 488 Abs. 3 BGB).
func loanAccount(loan *domain.ShareholderLoan, requested string) (string, error) {
	allowed := loanAccounts[loan.Direction]
	if requested = strings.TrimSpace(requested); requested != "" {
		for _, a := range allowed {
			if a == requested {
				return requested, nil
			}
		}
		return "", fmt.Errorf("Konto %s passt nicht zu diesem Darlehen; zulässig sind %s",
			requested, strings.Join(allowed, ", "))
	}
	months := 0
	if loan.MaturityDate != "" {
		months = monthsBetween(loan.StartDate, loan.MaturityDate)
	}
	if loan.Direction == domain.LoanToCompany {
		switch {
		case months > 60:
			return domain.AccountVerbGesellschafterUeber5, nil
		case months > 12:
			return domain.AccountVerbGesellschafter1Bis5, nil
		default:
			return domain.AccountVerbGesellschafterBis1, nil
		}
	}
	if months > 12 {
		return domain.AccountForderungenGesellschafterUeber1, nil
	}
	return domain.AccountForderungenGesellschafterBis1, nil
}

// interestAccount wählt das Zinskonto. Zinsen an einen Gesellschafter mit mehr
// als 25 % am Stammkapital führt der SKR04 auf 7317.
func interestAccount(loan *domain.ShareholderLoan, requested string, role companyRole) (string, error) {
	requested = strings.TrimSpace(requested)
	if loan.Direction == domain.LoanToShareholder {
		if requested != "" && requested != domain.AccountSonstigeZinsertraege {
			return "", fmt.Errorf("Zinsen eines Darlehens an den Gesellschafter sind Ertrag auf %s",
				domain.AccountSonstigeZinsertraege)
		}
		return domain.AccountSonstigeZinsertraege, nil
	}
	switch requested {
	case domain.AccountZinsenGesellschafterdarlehen, domain.AccountZinsenGesellschafterUeber25:
		return requested, nil
	case "":
		if role.stakePermille > 250 {
			return domain.AccountZinsenGesellschafterUeber25, nil
		}
		return domain.AccountZinsenGesellschafterdarlehen, nil
	}
	return "", fmt.Errorf("Zinsen für ein Gesellschafterdarlehen stehen auf %s oder %s",
		domain.AccountZinsenGesellschafterdarlehen, domain.AccountZinsenGesellschafterUeber25)
}

func monthsBetween(from, to string) int {
	a, errA := time.Parse(domain.DateLayout, from)
	b, errB := time.Parse(domain.DateLayout, to)
	if errA != nil || errB != nil {
		return 0
	}
	months := (b.Year()-a.Year())*12 + int(b.Month()) - int(a.Month())
	if b.Day() > a.Day() {
		months++
	}
	return months
}

// LoanLine liefert Konto und Gesellschafter eines Darlehens für die Aufteilung
// eines Bankumsatzes.
func (s *ShareholderService) LoanLine(ctx context.Context, loanID uint) (string, uint, string, error) {
	loan, err := s.loans.FindByID(ctx, loanID)
	if err != nil {
		return "", 0, "", fmt.Errorf("Darlehen %d wurde nicht gefunden: %w", loanID, err)
	}
	label := fmt.Sprintf("Gesellschafterdarlehen %d", loan.ID)
	if contact, err := s.contactRepo.FindByID(ctx, loan.ContactID); err == nil {
		label = fmt.Sprintf("Gesellschafterdarlehen %s", contact.LegalName())
	}
	return loan.Account, loan.ContactID, label, nil
}

// Loans liefert alle Darlehensverträge mit ihrem Stand und ihren Befunden zum
// Stichtag.
func (s *ShareholderService) Loans(ctx context.Context, cutoff string) ([]LoanView, error) {
	if err := domain.CheckDate("Stichtag", cutoff); err != nil {
		return nil, err
	}
	loans, err := s.loans.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	cfg, err := s.companySettings(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]LoanView, 0, len(loans))
	for i := range loans {
		contact, err := s.contactRepo.FindByID(ctx, loans[i].ContactID)
		if err != nil {
			return nil, fmt.Errorf("der Gesellschafter des Darlehens %d fehlt: %w", loans[i].ID, err)
		}
		view, err := s.view(ctx, &loans[i], contact, cfg, cutoff)
		if err != nil {
			return nil, err
		}
		out = append(out, *view)
	}
	return out, nil
}

// view rechnet den Stand eines Darlehens zum Stichtag.
func (s *ShareholderService) view(
	ctx context.Context, loan *domain.ShareholderLoan, contact *domain.Contact, cfg *domain.CompanySettings, cutoff string,
) (*LoanView, error) {
	movements, err := s.loanMovements(ctx, loan, cutoff)
	if err != nil {
		return nil, err
	}
	view := &LoanView{Loan: *loan, ContactName: contact.LegalName(), Findings: make([]domain.CheckFinding, 0)}
	for _, m := range movements {
		view.Outstanding += m.Amount
		if view.FirstMovement == "" && m.Amount > 0 {
			view.FirstMovement = m.Date
		}
	}
	periods, err := s.bookedInterest(ctx, loan.ID)
	if err != nil {
		return nil, err
	}
	for _, p := range periods {
		if p.PeriodTo > view.InterestBookedThrough {
			view.InterestBookedThrough = p.PeriodTo
		}
	}
	view.Findings = s.loanFindings(ctx, view, movements, companyRoleOf(cfg, contact), cutoff)
	return view, nil
}

// loanMovements sind die gebuchten Bewegungen des Darlehens bis zum Stichtag,
// als Bestandsänderung positiv bei Auszahlung.
func (s *ShareholderService) loanMovements(
	ctx context.Context, loan *domain.ShareholderLoan, cutoff string,
) ([]accounting.LoanMovement, error) {
	byContact, err := s.contactMovements(ctx, []string{loan.Account}, cutoff)
	if err != nil {
		return nil, err
	}
	raw := byContact[loan.ContactID][loan.Account]
	out := make([]accounting.LoanMovement, 0, len(raw))
	for _, m := range raw {
		amount := m.Amount
		if loan.Direction == domain.LoanToCompany {
			amount = -amount // Verbindlichkeit: Haben erhöht den Bestand
		}
		out = append(out, accounting.LoanMovement{Date: m.Date, Amount: amount})
	}
	return out, nil
}

// contactMovements liest die Buchungszeilen der Konten bis zum Stichtag je
// Kontakt und Konto, als Sollsaldo vorzeichenbehaftet. Der Saldenvortrag
// bleibt außen vor: er trägt den Saldo des Kontos ohne Kontakt ins neue Jahr,
// und die Zeilen des Vorjahres zählen hier ohnehin mit.
func (s *ShareholderService) contactMovements(
	ctx context.Context, accounts []string, cutoff string,
) (map[uint]map[string][]accounting.LoanMovement, error) {
	out := map[uint]map[string][]accounting.LoanMovement{}
	wanted := map[string]bool{}
	for _, a := range accounts {
		wanted[a] = true
	}
	seen := map[uint]bool{}
	for _, account := range accounts {
		entries, err := s.journalRepo.FindByAccountRange(ctx, account, "", cutoff)
		if err != nil {
			return nil, fmt.Errorf("das Konto %s ließ sich nicht lesen: %w", account, err)
		}
		for i := range entries {
			e := &entries[i]
			if seen[e.ID] || isCarryForward(e) {
				continue
			}
			seen[e.ID] = true
			for _, l := range e.Lines {
				if !wanted[l.Account] {
					continue
				}
				var contactID uint
				if l.ContactID != nil {
					contactID = *l.ContactID
				}
				amount := l.Amount
				if l.Side == domain.SideCredit {
					amount = -amount
				}
				if out[contactID] == nil {
					out[contactID] = map[string][]accounting.LoanMovement{}
				}
				out[contactID][l.Account] = append(out[contactID][l.Account],
					accounting.LoanMovement{Date: e.BookingDate, Amount: amount})
			}
		}
	}
	for _, byAccount := range out {
		for account := range byAccount {
			list := byAccount[account]
			sort.SliceStable(list, func(i, j int) bool { return list[i].Date < list[j].Date })
		}
	}
	return out, nil
}

// isCarryForward erkennt die Buchungen des Saldenvortrags.
func isCarryForward(e *domain.JournalEntry) bool {
	return e.Source == domain.EntrySourceOpening && e.DocumentNumber == carryForwardReference(e.FiscalYear)
}

// bookedInterest sind die gebuchten Zinsperioden, deren Buchung nicht
// storniert ist.
func (s *ShareholderService) bookedInterest(ctx context.Context, loanID uint) ([]domain.ShareholderLoanInterest, error) {
	all, err := s.loans.FindInterest(ctx, loanID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ShareholderLoanInterest, 0, len(all))
	for _, p := range all {
		reversal, err := s.journalRepo.FindReversalOf(ctx, p.EntryID)
		if err != nil {
			return nil, err
		}
		if reversal == nil {
			out = append(out, p)
		}
	}
	return out, nil
}

// InterestRequest bucht die Zinsen eines Darlehens für einen Zeitraum.
type InterestRequest struct {
	LoanID uint `json:"loanId"`
	// From ist leer der Tag nach der letzten gebuchten Periode, sonst der
	// Beginn des Darlehens.
	From string `json:"from,omitempty"`
	To   string `json:"to"`
	// BookingDate ist leer das Ende der Periode.
	BookingDate string `json:"bookingDate,omitempty"`
	// ReceiptID ist eine Zinsabrechnung; ohne sie entsteht ein Eigenbeleg mit
	// der Rechnung.
	ReceiptID     uint `json:"receiptId,omitempty"`
	ClosingPeriod bool `json:"closingPeriod,omitempty"`
}

// InterestPreview ist die Zinsrechnung vor dem Buchen.
type InterestPreview struct {
	Loan        domain.ShareholderLoan        `json:"loan"`
	ContactName string                        `json:"contactName"`
	From        string                        `json:"from"`
	To          string                        `json:"to"`
	Interest    accounting.LoanInterestResult `json:"interest"`
	Lines       []domain.JournalLine          `json:"lines"`
	Explanation string                        `json:"explanation"`
}

// PreviewInterest rechnet die Zinsen eines Zeitraums, ohne zu buchen.
func (s *ShareholderService) PreviewInterest(ctx context.Context, req InterestRequest) (*InterestPreview, error) {
	loan, err := s.loans.FindByID(ctx, req.LoanID)
	if err != nil {
		return nil, fmt.Errorf("Darlehen %d wurde nicht gefunden: %w", req.LoanID, err)
	}
	contact, err := s.contactRepo.FindByID(ctx, loan.ContactID)
	if err != nil {
		return nil, err
	}
	periods, err := s.bookedInterest(ctx, loan.ID)
	if err != nil {
		return nil, err
	}
	from := req.From
	if from == "" {
		from = loan.StartDate
		for _, p := range periods {
			if next := addDays(p.PeriodTo, 1); next > from {
				from = next
			}
		}
	}
	if err := domain.CheckDate("Zinsende", req.To); err != nil {
		return nil, err
	}
	for _, p := range periods {
		if from <= p.PeriodTo && req.To >= p.PeriodFrom {
			return nil, fmt.Errorf("die Zinsen vom %s bis %s sind bereits gebucht; der Zeitraum überschneidet sich",
				domain.GermanDate(p.PeriodFrom), domain.GermanDate(p.PeriodTo))
		}
	}
	movements, err := s.loanMovements(ctx, loan, req.To)
	if err != nil {
		return nil, err
	}
	result, err := accounting.LoanInterest(movements, loan.InterestRateBP, loan.DayCount, from, req.To)
	if err != nil {
		return nil, err
	}
	preview := &InterestPreview{
		Loan: *loan, ContactName: contact.LegalName(), From: from, To: req.To, Interest: result,
	}
	cid := contact.ID
	text := fmt.Sprintf("Zinsen %s–%s", domain.GermanDate(from), domain.GermanDate(req.To))
	if loan.Direction == domain.LoanToCompany {
		preview.Lines = []domain.JournalLine{
			{Side: domain.SideDebit, Account: loan.InterestAccount, Amount: result.Amount, Text: text},
			{Side: domain.SideCredit, Account: domain.AccountVerbindlichkeitenGesellschafter, ContactID: &cid,
				Amount: result.Amount, Text: text},
		}
	} else {
		preview.Lines = []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountForderungenGesellschafter, ContactID: &cid,
				Amount: result.Amount, Text: text},
			{Side: domain.SideCredit, Account: loan.InterestAccount, Amount: result.Amount, Text: text},
		}
	}
	preview.Explanation = interestExplanation(loan, contact, result)
	return preview, nil
}

// interestExplanation ist der Inhalt des Eigenbelegs: woraus die Zinsen
// folgen.
func interestExplanation(loan *domain.ShareholderLoan, contact *domain.Contact, r accounting.LoanInterestResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Zinsen Gesellschafterdarlehen %s %s, %s, Zinsmethode %s",
		directionLabel(loan.Direction), contact.LegalName(), domain.RateLabel(loan.InterestRateBP), loan.DayCount)
	if loan.ContractDate != "" {
		fmt.Fprintf(&b, ", Vertrag vom %s", domain.GermanDate(loan.ContractDate))
	}
	b.WriteString(". ")
	for _, seg := range r.Segments {
		fmt.Fprintf(&b, "%s bis %s: %s € × %d/%d Tage; ",
			domain.GermanDate(seg.From), domain.GermanDate(seg.To), seg.Balance, seg.Days, r.YearDays)
	}
	fmt.Fprintf(&b, "Zinsen %s €.", r.Amount)
	return b.String()
}

// BookInterest bucht die Zinsen eines Zeitraums: bei einem Darlehen des
// Gesellschafters Zinsaufwand an Verbindlichkeit gegenüber ihm (3640), bei
// einem Darlehen an ihn Forderung (1307) an Zinsertrag. Die Zahlung ordnet
// danach die Bank zu. Zinsen auf eine Kreditgewährung sind steuerfrei
// (§ 4 Nr. 8 Buchst. a UStG).
func (s *ShareholderService) BookInterest(ctx context.Context, req InterestRequest) (*domain.JournalEntry, error) {
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (*domain.JournalEntry, error) {
		preview, err := s.PreviewInterest(ctx, req)
		if err != nil {
			return nil, err
		}
		if preview.Interest.Amount <= 0 {
			return nil, fmt.Errorf("für den Zeitraum %s bis %s fallen keine Zinsen an",
				domain.GermanDate(preview.From), domain.GermanDate(preview.To))
		}
		bookingDate := req.BookingDate
		if bookingDate == "" {
			bookingDate = preview.To
		}
		treatment := domain.TaxTreatmentNotTaxable
		if preview.Loan.Direction == domain.LoanToShareholder {
			treatment = domain.TaxTreatmentExempt
		}
		cid := preview.Loan.ContactID
		manual := ManualEntryRequest{
			Entry: domain.JournalEntry{
				BookingDate:     bookingDate,
				DocumentDate:    bookingDate,
				ServiceDateFrom: preview.From,
				ServiceDateTo:   preview.To,
				Description: fmt.Sprintf("Zinsen Gesellschafterdarlehen %s %s–%s", preview.ContactName,
					domain.GermanDate(preview.From), domain.GermanDate(preview.To)),
				TaxTreatment:  treatment,
				ContactID:     &cid,
				ClosingPeriod: req.ClosingPeriod,
				Lines:         preview.Lines,
			},
		}
		if req.ReceiptID != 0 {
			manual.ReceiptID = req.ReceiptID
		} else {
			manual.SelfIssued = &SelfIssuedReceiptRequest{
				DocumentDate: bookingDate,
				GrossAmount:  preview.Interest.Amount,
				Reason:       preview.Explanation,
			}
		}
		entry, err := s.posting.PostManualEntry(ctx, manual)
		if err != nil {
			return nil, err
		}
		if err := s.loans.SaveInterest(ctx, &domain.ShareholderLoanInterest{
			LoanID: preview.Loan.ID, PeriodFrom: preview.From, PeriodTo: preview.To,
			Amount: preview.Interest.Amount, EntryID: entry.ID,
		}); err != nil {
			return nil, fmt.Errorf("die Zinsperiode ließ sich nicht vermerken: %w", err)
		}
		return entry, nil
	})
}

// ShareholderAccountBalance ist der Saldo eines Kontos für einen
// Gesellschafter, positiv im Soll.
type ShareholderAccountBalance struct {
	Account string       `json:"account"`
	Name    string       `json:"name"`
	Balance domain.Cents `json:"balance"`
}

// ShareholderBalances sind die Salden eines Gesellschafters. ContactID null
// sammelt die Zeilen ohne Kontakt; sie lassen sich keinem Gesellschafter
// zuordnen.
type ShareholderBalances struct {
	ContactID   uint                        `json:"contactId"`
	ContactName string                      `json:"contactName"`
	Accounts    []ShareholderAccountBalance `json:"accounts"`
	// Receivables und Liabilities fassen Forderungen gegen ihn und
	// Verbindlichkeiten ihm gegenüber zusammen; sie werden nicht verrechnet
	// (§ 246 Abs. 2 Satz 1 HGB).
	Receivables domain.Cents `json:"receivables"`
	Liabilities domain.Cents `json:"liabilities"`
}

// Balances liefert je Gesellschafter die Salden der Gesellschafterkonten zum
// Stichtag.
func (s *ShareholderService) Balances(ctx context.Context, cutoff string) ([]ShareholderBalances, error) {
	if err := domain.CheckDate("Stichtag", cutoff); err != nil {
		return nil, err
	}
	movements, err := s.contactMovements(ctx, shareholderAccounts, cutoff)
	if err != nil {
		return nil, err
	}
	chart, err := s.journalSvc.Chart(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ShareholderBalances, 0, len(movements))
	for contactID, byAccount := range movements {
		row := ShareholderBalances{ContactID: contactID, Accounts: make([]ShareholderAccountBalance, 0, len(byAccount))}
		if contactID == 0 {
			row.ContactName = "ohne Gesellschafter"
		} else if c, err := s.contactRepo.FindByID(ctx, contactID); err == nil {
			row.ContactName = c.LegalName()
		}
		for account, list := range byAccount {
			var balance domain.Cents
			for _, m := range list {
				balance += m.Amount
			}
			if balance == 0 {
				continue
			}
			row.Accounts = append(row.Accounts, ShareholderAccountBalance{
				Account: account, Name: chart.Name(account), Balance: balance})
			if balance > 0 {
				row.Receivables += balance
			} else {
				row.Liabilities -= balance
			}
		}
		if len(row.Accounts) == 0 {
			continue
		}
		sort.Slice(row.Accounts, func(i, j int) bool { return row.Accounts[i].Account < row.Accounts[j].Account })
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ContactName < out[j].ContactName })
	return out, nil
}

// clearingStandingDays ist die Dauer, ab der ein Sollsaldo auf dem
// Verrechnungskonto wie ein zinsloses Darlehen wirkt.
const clearingStandingDays = 90

// Findings liefert die Befunde zu Gesellschaftern zum Stichtag für den
// Prüflauf: Darlehen mit unüblichen Konditionen, ungebuchte Zinsen,
// Sollsalden auf dem Verrechnungskonto und Darlehenssalden ohne Vertrag.
func (s *ShareholderService) Findings(ctx context.Context, cutoff string) ([]domain.CheckFinding, error) {
	views, err := s.Loans(ctx, cutoff)
	if err != nil {
		return nil, err
	}
	out := make([]domain.CheckFinding, 0, 4)
	covered := map[string]bool{}
	for _, v := range views {
		out = append(out, v.Findings...)
		covered[fmt.Sprintf("%d/%s", v.Loan.ContactID, v.Loan.Account)] = true
	}

	movements, err := s.contactMovements(ctx, append(
		[]string{domain.AccountForderungenGesellschafter}, loanAccountList()...), cutoff)
	if err != nil {
		return nil, err
	}
	for contactID, byAccount := range movements {
		if contactID == 0 {
			continue
		}
		name := fmt.Sprintf("Kontakt %d", contactID)
		if c, err := s.contactRepo.FindByID(ctx, contactID); err == nil {
			name = c.LegalName()
		}
		if since, balance := debitStandingSince(byAccount[domain.AccountForderungenGesellschafter]); balance > 0 &&
			since != "" && addDays(since, clearingStandingDays) <= cutoff {
			out = append(out, domain.CheckFinding{
				Rule: domain.CheckRuleShareholderClearing, Severity: domain.CheckWarning,
				ObjectType: "CONTACT", ObjectID: fmt.Sprintf("%d", contactID), ObjectName: name,
				Message: fmt.Sprintf(
					"Auf dem Verrechnungskonto %s steht seit %s eine Forderung gegen %s von zuletzt %s €. "+
						"Ein stehender Sollsaldo wirkt wie ein zinsloses Darlehen an den Gesellschafter und kann eine "+
						"verdeckte Gewinnausschüttung sein; ausgleichen oder einen verzinsten Darlehensvertrag schließen",
					domain.AccountForderungenGesellschafter, domain.GermanDate(since), name, balance),
				Reference: "R 8.5 Abs. 1 KStR 2022, § 8 Abs. 3 Satz 2 KStG",
			})
		}
		for account, list := range byAccount {
			if account == domain.AccountForderungenGesellschafter || covered[fmt.Sprintf("%d/%s", contactID, account)] {
				continue
			}
			var balance domain.Cents
			for _, m := range list {
				balance += m.Amount
			}
			if balance == 0 {
				continue
			}
			out = append(out, domain.CheckFinding{
				Rule: domain.CheckRuleShareholderLoan, Severity: domain.CheckWarning,
				ObjectType: "CONTACT", ObjectID: fmt.Sprintf("%d", contactID), ObjectName: name,
				Message: fmt.Sprintf(
					"Auf dem Darlehenskonto %s steht für %s ein Saldo von %s €, ohne dass ein Darlehensvertrag erfasst "+
						"ist. Ein Darlehen zwischen GmbH und Gesellschafter braucht eine klare, im Voraus getroffene "+
						"Vereinbarung", account, name, balance.Abs()),
				Reference: "R 8.5 Abs. 2 KStR 2022",
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ObjectID != out[j].ObjectID {
			return out[i].ObjectID < out[j].ObjectID
		}
		return out[i].Message < out[j].Message
	})
	return out, nil
}

func loanAccountList() []string {
	out := []string{}
	for _, d := range []domain.LoanDirection{domain.LoanToCompany, domain.LoanToShareholder} {
		for _, a := range loanAccounts[d] {
			if a != domain.AccountForderungenGesellschafter && a != domain.AccountVerbindlichkeitenGesellschafter {
				out = append(out, a)
			}
		}
	}
	return out
}

// debitStandingSince liefert den Tag, seit dem der Saldo ununterbrochen im
// Soll steht, und den Saldo am Ende.
func debitStandingSince(list []accounting.LoanMovement) (string, domain.Cents) {
	var balance domain.Cents
	since := ""
	for i, m := range list {
		balance += m.Amount
		// Mehrere Bewegungen eines Tages zählen als eine.
		if i+1 < len(list) && list[i+1].Date == m.Date {
			continue
		}
		switch {
		case balance <= 0:
			since = ""
		case since == "":
			since = m.Date
		}
	}
	return since, balance
}

// loanFindings prüft ein Darlehen gegen den Fremdvergleich und die
// tatsächliche Durchführung.
func (s *ShareholderService) loanFindings(
	ctx context.Context, v *LoanView, movements []accounting.LoanMovement, role companyRole, cutoff string,
) []domain.CheckFinding {
	loan := &v.Loan
	out := make([]domain.CheckFinding, 0, 2)
	add := func(message, reference string) {
		out = append(out, domain.CheckFinding{
			Rule: domain.CheckRuleShareholderLoan, Severity: domain.CheckWarning,
			ObjectType: "SHAREHOLDER_LOAN", ObjectID: fmt.Sprintf("%d", loan.ID),
			ObjectName: fmt.Sprintf("Darlehen %s %s", directionLabel(loan.Direction), v.ContactName),
			Message:    message, Reference: reference,
		})
	}

	if loan.ContractReceiptID == nil {
		add("Zum Darlehen ist kein Vertrag abgelegt. Mit einem beherrschenden Gesellschafter braucht es eine "+
			"zivilrechtlich wirksame, klare und im Voraus geschlossene Vereinbarung",
			"R 8.5 Abs. 2 KStR 2022")
	}
	if loan.ContractDate != "" && v.FirstMovement != "" && loan.ContractDate > v.FirstMovement {
		add(fmt.Sprintf("Der Vertrag datiert vom %s, ausgezahlt wurde schon am %s. Eine nachträgliche "+
			"Vereinbarung wirkt nicht zurück", domain.GermanDate(loan.ContractDate), domain.GermanDate(v.FirstMovement)),
			"R 8.5 Abs. 2 KStR 2022")
	}

	rate := domain.RateLabel(loan.InterestRateBP)
	if loan.Direction == domain.LoanToShareholder {
		switch {
		case loan.InterestRateBP == 0:
			add("Das Darlehen an den Gesellschafter ist zinslos. Der entgangene Zins ist eine verhinderte "+
				"Vermögensmehrung und kann eine verdeckte Gewinnausschüttung sein",
				"R 8.5 Abs. 1 KStR 2022, § 8 Abs. 3 Satz 2 KStG")
		case loan.ComparisonRateBP != nil && loan.InterestRateBP < *loan.ComparisonRateBP:
			add(fmt.Sprintf("Der Zins von %s liegt unter dem belegten Vergleichszins von %s (%s). Der Unterschied "+
				"kann eine verdeckte Gewinnausschüttung sein", rate, domain.RateLabel(*loan.ComparisonRateBP),
				loan.ComparisonSource), "R 8.5 Abs. 1 KStR 2022, § 8 Abs. 3 Satz 2 KStG")
		}
		if loan.Collateral == "" {
			add("Für das Darlehen an den Gesellschafter ist keine Sicherheit vermerkt. Aus dem gebundenen Vermögen "+
				"darf nur ausgezahlt werden, wenn der Rückzahlungsanspruch vollwertig ist",
				"§ 30 Abs. 1 Satz 2 GmbHG")
		}
		if role.director {
			add("Der Gesellschafter ist Geschäftsführer. Ein Kredit an ihn darf nicht aus dem Vermögen gewährt "+
				"werden, das zur Erhaltung des Stammkapitals erforderlich ist", "§ 43a GmbHG")
		}
	} else if loan.ComparisonRateBP != nil && loan.InterestRateBP > *loan.ComparisonRateBP {
		add(fmt.Sprintf("Der Zins von %s liegt über dem belegten Vergleichszins von %s (%s). Der übersteigende "+
			"Teil kann eine verdeckte Gewinnausschüttung sein", rate, domain.RateLabel(*loan.ComparisonRateBP),
			loan.ComparisonSource), "§ 8 Abs. 3 Satz 2 KStG")
	}
	if loan.ComparisonRateBP == nil && loan.InterestRateBP > 0 {
		add(fmt.Sprintf("Für den Zins von %s ist kein Vergleichszins eines fremden Dritten belegt. Der "+
			"Fremdvergleich ist eine Wertungsfrage; ein Bankangebot oder eine Zinsstatistik belegt ihn", rate),
			"R 8.5 KStR 2022")
	}

	if loan.MaturityDate != "" && loan.MaturityDate < cutoff && v.Outstanding > 0 {
		add(fmt.Sprintf("Das Darlehen war am %s fällig, offen sind noch %s €. Eine Vereinbarung, die nicht "+
			"durchgeführt wird, trägt steuerlich nicht", domain.GermanDate(loan.MaturityDate), v.Outstanding),
			"R 8.5 Abs. 2 KStR 2022")
	}

	if loan.InterestRateBP > 0 {
		due := s.lastYearEnd(ctx, cutoff)
		from := loan.StartDate
		if v.InterestBookedThrough != "" {
			from = addDays(v.InterestBookedThrough, 1)
		}
		if due != "" && from <= due {
			if r, err := accounting.LoanInterest(movements, loan.InterestRateBP, loan.DayCount, from, due); err == nil &&
				r.Amount > 0 {
				add(fmt.Sprintf("Die Zinsen vom %s bis %s sind nicht gebucht (%s € nach Vertrag). Zinsen gehören in "+
					"das Jahr, auf das sie entfallen", domain.GermanDate(from), domain.GermanDate(due), r.Amount),
					"§ 252 Abs. 1 Nr. 5 HGB, R 8.5 Abs. 2 KStR 2022")
			}
		}
	}
	return out
}

// lastYearEnd ist das Ende des letzten Geschäftsjahres, das bis zum Stichtag
// abgelaufen ist; der Stichtag selbst, wenn er ein Jahresende ist.
func (s *ShareholderService) lastYearEnd(ctx context.Context, cutoff string) string {
	startMonth := 1
	if cfg, err := s.companySettings(ctx); err == nil && cfg.FiscalYearStartMonth > 0 {
		startMonth = cfg.FiscalYearStartMonth
	}
	year := domain.GetFiscalYearForDate(cutoff, startMonth)
	end := addDays(domain.FiscalYearFirstDay(year+1, startMonth), -1)
	if end == cutoff {
		return cutoff
	}
	return addDays(domain.FiscalYearFirstDay(year, startMonth), -1)
}

func (s *ShareholderService) companySettings(ctx context.Context) (*domain.CompanySettings, error) {
	if s.settingsRepo == nil {
		return nil, fmt.Errorf("die Unternehmensdaten sind nicht eingerichtet")
	}
	cfg, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("die Unternehmensdaten ließen sich nicht lesen: %w", err)
	}
	return cfg, nil
}
