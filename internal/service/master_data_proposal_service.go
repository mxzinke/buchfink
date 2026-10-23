package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/calendar"
	"github.com/buchfink/buchfink/internal/domain"
)

// Vorschläge für die Stammdaten.
//
// Unternehmensdaten, Gründung und Bankkonten ändert nur der Mensch. Ein Agent,
// der sie aus einem Schreiben liest, legt einen Vorschlag ab; die App zeigt
// ihn als Vorschau gegen den aktuellen Stand, und erst das Übernehmen in der
// App schreibt — über dieselben Dienstmethoden wie die Formulare, mit ihren
// Prüfungen und ihrem Vorbehalt für den Menschen.

// MasterDataSettings sind die Unternehmensdaten, über die ein Vorschlag
// geschrieben wird.
type MasterDataSettings interface {
	GetCompanySettings(ctx context.Context) (*domain.CompanySettings, error)
	UpdateCompanySettings(ctx context.Context, settings *domain.CompanySettings) error
}

// MasterDataFoundation ist die Gründung, über die ein Vorschlag geschrieben
// wird.
type MasterDataFoundation interface {
	GetState(ctx context.Context) (*FoundationState, error)
	Save(ctx context.Context, notarizedOn string, f *domain.Foundation) (*domain.Foundation, error)
}

// MasterDataBank sind die Bankkonten, über die ein Vorschlag geschrieben wird.
type MasterDataBank interface {
	Accounts(ctx context.Context) ([]domain.BankAccount, error)
	ConfigureAccounts(ctx context.Context, accounts []domain.BankAccount) error
	SetInvoiceAccount(ctx context.Context, iban string) error
}

// MasterDataReceipts liefert die Belegnummer des Schreibens, aus dem ein
// Vorschlag stammt.
type MasterDataReceipts interface {
	Get(ctx context.Context, id uint) (*domain.Receipt, error)
}

// MasterDataProposalService führt die Vorschläge.
type MasterDataProposalService struct {
	serviceClock
	repo       domain.MasterDataProposalRepository
	auditRepo  domain.AuditRepository
	tx         domain.TxRunner
	settings   MasterDataSettings
	foundation MasterDataFoundation
	bank       MasterDataBank
	receipts   MasterDataReceipts
}

func NewMasterDataProposalService(
	repo domain.MasterDataProposalRepository, auditRepo domain.AuditRepository, tx domain.TxRunner,
	settings MasterDataSettings, foundation MasterDataFoundation, bank MasterDataBank,
) *MasterDataProposalService {
	return &MasterDataProposalService{
		repo: repo, auditRepo: auditRepo, tx: tx,
		settings: settings, foundation: foundation, bank: bank,
	}
}

// SetReceiptSource hängt die Belege an; ohne sie zeigt die Vorschau keine
// Belegnummer.
func (s *MasterDataProposalService) SetReceiptSource(src MasterDataReceipts) { s.receipts = src }

// MasterDataProposalRequest ist ein neuer Vorschlag.
type MasterDataProposalRequest struct {
	Reason    string
	ReceiptID *uint
	Changes   domain.MasterDataChanges
}

// MasterDataChangeLine ist eine Zeile der Vorschau: ein Feld mit dem Stand
// heute und dem vorgeschlagenen.
type MasterDataChangeLine struct {
	Area   string `json:"area"`
	Field  string `json:"field"`
	Before string `json:"before"`
	After  string `json:"after"`
}

// MasterDataProposalView ist ein Vorschlag mit seiner Vorschau. Fingerprint
// kennzeichnet genau diese Vorschau; übernommen wird nur, solange sie gilt.
// Problem sagt, warum sich der Vorschlag nicht übernehmen lässt.
type MasterDataProposalView struct {
	domain.MasterDataProposal
	Changes     domain.MasterDataChanges `json:"changes"`
	Lines       []MasterDataChangeLine   `json:"lines"`
	Fingerprint string                   `json:"fingerprint"`
	Problem     string                   `json:"problem,omitempty"`
	// ReceiptNumber ist die Nummer des Belegs ReceiptID.
	ReceiptNumber string `json:"receiptNumber,omitempty"`
}

const (
	areaCompany    = "Unternehmensdaten"
	areaFoundation = "Gründung"
	areaBank       = "Bankkonten"
)

// Propose legt einen Vorschlag ab. Er ändert nichts an den Stammdaten.
func (s *MasterDataProposalService) Propose(
	ctx context.Context, req MasterDataProposalRequest,
) (*MasterDataProposalView, error) {
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return nil, fmt.Errorf("die Begründung fehlt: woher stammen die Angaben?")
	}
	changes := req.Changes
	if err := normalizeChanges(&changes); err != nil {
		return nil, err
	}
	if changes.Empty() {
		return nil, fmt.Errorf("der Vorschlag enthält keine Änderung")
	}
	raw, err := json.Marshal(changes)
	if err != nil {
		return nil, err
	}
	p := &domain.MasterDataProposal{
		Status: domain.ProposalOpen, Reason: reason, ReceiptID: req.ReceiptID, Changes: string(raw),
		Channel: string(actor.RecordedChannel(ctx)), Actor: actor.Actor(),
	}
	var view *MasterDataProposalView
	err = s.tx.RunInTx(ctx, func(ctx context.Context) error {
		if err := s.checkAgainstCurrent(ctx, changes); err != nil {
			return err
		}
		if err := s.repo.Create(ctx, p); err != nil {
			return err
		}
		if view, err = s.view(ctx, *p); err != nil {
			return err
		}
		if view.Problem != "" {
			return fmt.Errorf("%s", view.Problem)
		}
		return logAudit(ctx, s.auditRepo, domain.AuditActionCreate, "STAMMDATEN_VORSCHLAG", fmt.Sprint(p.ID),
			fmt.Sprintf("Vorschlag %d für die Stammdaten über %s: %s", p.ID, p.Channel, summarizeLines(view.Lines)))
	})
	if err != nil {
		return nil, err
	}
	return view, nil
}

// Pending liefert die offenen Vorschläge mit ihrer Vorschau, den ältesten
// zuerst.
func (s *MasterDataProposalService) Pending(ctx context.Context) ([]MasterDataProposalView, error) {
	open, err := s.repo.FindByStatus(ctx, domain.ProposalOpen)
	if err != nil {
		return nil, err
	}
	out := make([]MasterDataProposalView, 0, len(open))
	for _, p := range open {
		view, err := s.view(ctx, p)
		if err != nil {
			return nil, err
		}
		out = append(out, *view)
	}
	return out, nil
}

// Recent liefert die jüngsten Vorschläge ohne Vorschau, den neuesten zuerst:
// der Agent sieht daran, was übernommen oder abgelehnt ist.
func (s *MasterDataProposalService) Recent(ctx context.Context, limit int) ([]domain.MasterDataProposal, error) {
	return s.repo.FindRecent(ctx, limit)
}

// Accept übernimmt einen Vorschlag, so wie die Vorschau mit fingerprint ihn
// gezeigt hat. Nur in der App. Gründung, Bankkonten und Unternehmensdaten
// werden zusammen geschrieben oder gar nicht.
func (s *MasterDataProposalService) Accept(
	ctx context.Context, id uint, fingerprint string,
) (*domain.MasterDataProposal, error) {
	if err := requireHuman(ctx, "Das Übernehmen eines Vorschlags für die Stammdaten"); err != nil {
		return nil, err
	}
	var out *domain.MasterDataProposal
	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		p, err := s.openProposal(ctx, id)
		if err != nil {
			return err
		}
		view, err := s.view(ctx, *p)
		if err != nil {
			return err
		}
		if view.Problem != "" {
			return fmt.Errorf("%s", view.Problem)
		}
		if view.Fingerprint != fingerprint {
			return fmt.Errorf("die Stammdaten haben sich seit der Vorschau geändert; bitte die Vorschau neu prüfen")
		}
		if err := s.apply(ctx, view.Changes); err != nil {
			return err
		}
		s.decide(p, domain.ProposalAccepted, "")
		if err := s.repo.Save(ctx, p); err != nil {
			return err
		}
		out = p
		return logAudit(ctx, s.auditRepo, domain.AuditActionUpdate, "STAMMDATEN_VORSCHLAG", fmt.Sprint(p.ID),
			fmt.Sprintf("Vorschlag %d für die Stammdaten übernommen: %s", p.ID, summarizeLines(view.Lines)))
	})
	return out, err
}

// Reject lehnt einen Vorschlag ab. Nur in der App; die Notiz sieht der Agent.
func (s *MasterDataProposalService) Reject(ctx context.Context, id uint, note string) (*domain.MasterDataProposal, error) {
	if err := requireHuman(ctx, "Das Ablehnen eines Vorschlags für die Stammdaten"); err != nil {
		return nil, err
	}
	var out *domain.MasterDataProposal
	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		p, err := s.openProposal(ctx, id)
		if err != nil {
			return err
		}
		s.decide(p, domain.ProposalRejected, strings.TrimSpace(note))
		if err := s.repo.Save(ctx, p); err != nil {
			return err
		}
		out = p
		details := fmt.Sprintf("Vorschlag %d für die Stammdaten abgelehnt", p.ID)
		if p.DecisionNote != "" {
			details += ": " + p.DecisionNote
		}
		return logAudit(ctx, s.auditRepo, domain.AuditActionUpdate, "STAMMDATEN_VORSCHLAG", fmt.Sprint(p.ID), details)
	})
	return out, err
}

func (s *MasterDataProposalService) openProposal(ctx context.Context, id uint) (*domain.MasterDataProposal, error) {
	p, err := s.repo.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.Status != domain.ProposalOpen {
		return nil, fmt.Errorf("Vorschlag %d ist bereits entschieden (%s)", id, p.Status)
	}
	return p, nil
}

func (s *MasterDataProposalService) decide(p *domain.MasterDataProposal, status domain.MasterDataProposalStatus, note string) {
	now := s.instant()
	p.Status, p.DecidedAt, p.DecidedBy, p.DecisionNote = status, &now, actor.Actor(), note
}

// apply schreibt über die Dienste der Formulare: erst die Gründung, die das
// Gründungsdatum setzt und an der die Eintragung hängt, dann die Bankkonten,
// zuletzt die Unternehmensdaten auf dem dann aktuellen Stand.
func (s *MasterDataProposalService) apply(ctx context.Context, c domain.MasterDataChanges) error {
	if c.Foundation != nil {
		state, err := s.foundation.GetState(ctx)
		if err != nil {
			return err
		}
		if _, err := s.foundation.Save(ctx, c.Foundation.NotarizedOn, foundationOf(*c.Foundation, state)); err != nil {
			return err
		}
	}
	if len(c.BankAccounts) > 0 {
		existing, err := s.bank.Accounts(ctx)
		if err != nil {
			return err
		}
		accounts, err := resolveBankAccounts(c.BankAccounts, existing)
		if err != nil {
			return err
		}
		if err := s.bank.ConfigureAccounts(ctx, accounts); err != nil {
			return err
		}
		if target, current := invoiceAccount(accounts, existing); target != current {
			if err := s.bank.SetInvoiceAccount(ctx, target); err != nil {
				return err
			}
		}
	}
	if c.Company != nil {
		current, err := s.settings.GetCompanySettings(ctx)
		if err != nil {
			return err
		}
		c.Company.ApplyTo(current)
		if err := s.settings.UpdateCompanySettings(ctx, current); err != nil {
			return err
		}
	}
	return nil
}

// view rechnet die Vorschau gegen den Stand von heute.
func (s *MasterDataProposalService) view(ctx context.Context, p domain.MasterDataProposal) (*MasterDataProposalView, error) {
	view := &MasterDataProposalView{MasterDataProposal: p, Lines: []MasterDataChangeLine{}}
	if err := json.Unmarshal([]byte(p.Changes), &view.Changes); err != nil {
		return nil, fmt.Errorf("Vorschlag %d lässt sich nicht lesen: %w", p.ID, err)
	}
	if p.ReceiptID != nil && s.receipts != nil {
		r, err := s.receipts.Get(ctx, *p.ReceiptID)
		if err != nil {
			return nil, err
		}
		view.ReceiptNumber = r.ReceiptNumber
	}
	c := view.Changes
	settings, err := s.settings.GetCompanySettings(ctx)
	if err != nil {
		return nil, err
	}
	if c.Foundation != nil {
		state, err := s.foundation.GetState(ctx)
		if err != nil {
			return nil, err
		}
		view.Lines = append(view.Lines, foundationLines(*c.Foundation, settings.FoundedOn, state)...)
	}
	if len(c.BankAccounts) > 0 {
		existing, err := s.bank.Accounts(ctx)
		if err != nil {
			return nil, err
		}
		accounts, err := resolveBankAccounts(c.BankAccounts, existing)
		if err != nil {
			view.Problem = err.Error()
		}
		view.Lines = append(view.Lines, bankLines(accounts, existing)...)
	}
	if c.Company != nil {
		before := *settings
		if c.Foundation != nil {
			// Das Stammkapital zeigt schon die Gründung; die Unternehmensdaten
			// übernehmen es von dort.
			before.ShareCapital = c.Foundation.ShareCapital
		}
		after := before
		c.Company.ApplyTo(&after)
		view.Lines = append(view.Lines, companyLines(&before, &after)...)
	}
	if len(view.Lines) == 0 && view.Problem == "" && p.Status == domain.ProposalOpen {
		view.Problem = "Der Vorschlag ändert nichts mehr: die Angaben stehen schon so in den Stammdaten."
	}
	sum := sha256.New()
	_ = json.NewEncoder(sum).Encode(struct {
		ID    uint                   `json:"id"`
		Lines []MasterDataChangeLine `json:"lines"`
	}{p.ID, view.Lines})
	view.Fingerprint = hex.EncodeToString(sum.Sum(nil))[:16]
	return view, nil
}

// checkAgainstCurrent prüft, was erst mit dem Stand des Mandanten zu prüfen
// ist: die Gründung nach den Regeln der Rechtsform, die Steuernummer nach dem
// Bundesland.
func (s *MasterDataProposalService) checkAgainstCurrent(ctx context.Context, c domain.MasterDataChanges) error {
	settings, err := s.settings.GetCompanySettings(ctx)
	if err != nil {
		return err
	}
	if c.Foundation != nil {
		if co := c.Company; co != nil && co.ShareCapital != nil && *co.ShareCapital != c.Foundation.ShareCapital {
			return fmt.Errorf("das Stammkapital der Unternehmensdaten (%s) weicht von dem der Gründung (%s) ab",
				euro(*co.ShareCapital), euro(c.Foundation.ShareCapital))
		}
		rules, err := foundationRulesOf(settings.LegalForm)
		if err != nil {
			return err
		}
		if err := validateFoundation(c.Foundation.NotarizedOn, foundationOf(*c.Foundation, nil), rules); err != nil {
			return err
		}
	}
	if c.Company != nil {
		after := *settings
		c.Company.ApplyTo(&after)
		if c.Company.TaxNumber != nil || c.Company.FederalState != nil {
			if _, err := domain.ElsterTaxNumber(after.TaxNumber, after.FederalState); err != nil {
				return err
			}
		}
	}
	return nil
}

// normalizeChanges bringt die Angaben in die Form der Formulare und weist
// ab, was keines der Formulare annähme.
func normalizeChanges(c *domain.MasterDataChanges) error {
	if co := c.Company; co != nil {
		for _, p := range []*string{
			co.CompanyName, co.ManagingDirectors, co.SupervisoryBoardChair, co.Street, co.HouseNumber,
			co.AddressAddition, co.PostalCode, co.City, co.ContactName, co.ContactPhone, co.ContactEmail,
			co.Notary, co.DeedNumber, co.Seat, co.RegisterCourt, co.RegisterNumber, co.TaxOffice,
		} {
			if p != nil {
				*p = strings.TrimSpace(*p)
			}
		}
		if co.CompanyName != nil && *co.CompanyName == "" {
			return fmt.Errorf("die Firma kann nicht leer sein")
		}
		upper := func(p *string) {
			if p != nil {
				*p = strings.ToUpper(strings.Join(strings.Fields(*p), ""))
			}
		}
		upper(co.CountryCode)
		upper(co.FederalState)
		upper(co.VatID)
		upper(co.TaxationType)
		if co.TaxNumber != nil {
			*co.TaxNumber = strings.TrimSpace(*co.TaxNumber)
		}
		if co.CountryCode != nil && len(*co.CountryCode) != 2 {
			return fmt.Errorf("das Land ist als ISO-Code mit zwei Buchstaben anzugeben, etwa DE")
		}
		if co.FederalState != nil && *co.FederalState != "" && !calendar.ValidState(*co.FederalState) {
			return fmt.Errorf("unbekanntes Bundesland %q (erwartet ein Kürzel wie HH oder BY)", *co.FederalState)
		}
		if co.VatPeriod != nil {
			if _, ok := vatPeriodLabels[*co.VatPeriod]; !ok || *co.VatPeriod == "unknown" {
				return fmt.Errorf("unbekannter Voranmeldungszeitraum %q (erwartet month, quarter, year oder none)", *co.VatPeriod)
			}
		}
		if co.TaxationType != nil && *co.TaxationType != "SOLL" && *co.TaxationType != "IST" {
			return fmt.Errorf("unbekannte Besteuerung %q (erwartet SOLL oder IST)", *co.TaxationType)
		}
		if co.RegisteredOn != nil && *co.RegisteredOn != "" {
			if _, err := calendar.Parse(*co.RegisteredOn); err != nil {
				return fmt.Errorf("das Datum der Eintragung ist kein Kalendertag (erwartet JJJJ-MM-TT)")
			}
		}
		if co.SpecialPrepayment != nil && *co.SpecialPrepayment < 0 {
			return fmt.Errorf("die Sondervorauszahlung kann nicht negativ sein")
		}
		if co.TaxAdvisor != nil {
			if err := co.TaxAdvisor.Validate(); err != nil {
				return err
			}
		}
		if co.ShareCapital != nil && *co.ShareCapital < 0 {
			return fmt.Errorf("das Stammkapital kann nicht negativ sein")
		}
		if co.Shareholders != nil {
			for i, sh := range *co.Shareholders {
				(*co.Shareholders)[i].Name = strings.TrimSpace(sh.Name)
				if (*co.Shareholders)[i].Name == "" || sh.ShareCapital <= 0 {
					return fmt.Errorf("der %d. Gesellschafter braucht Namen und Geschäftsanteil", i+1)
				}
			}
		}
		if co.Empty() {
			c.Company = nil
		}
	}
	for i := range c.BankAccounts {
		a := &c.BankAccounts[i]
		a.IBAN = domain.NormalizeIBAN(a.IBAN)
		a.BIC = strings.ToUpper(strings.Join(strings.Fields(a.BIC), ""))
		a.Name, a.BankName = strings.TrimSpace(a.Name), strings.TrimSpace(a.BankName)
		a.ClosingBalance, a.ClosingDate = 0, ""
		if !domain.ValidIBAN(a.IBAN) {
			return fmt.Errorf("ungültige IBAN: %s", a.IBAN)
		}
		if a.BIC != "" && !domain.ValidBIC(a.BIC) {
			return fmt.Errorf("ungültige BIC: %s", a.BIC)
		}
		if a.Name == "" {
			return fmt.Errorf("das Bankkonto %s braucht einen Namen", a.IBAN)
		}
		if a.LedgerAccount != "" && !isBankLedgerAccount(a.LedgerAccount) {
			return fmt.Errorf("für ein Bankkonto ist eines der Konten 1800 bis 1850 vorgesehen")
		}
		if a.Currency != "" && a.Currency != "EUR" {
			return fmt.Errorf("Buchfink führt Bankkonten in Euro")
		}
		for _, other := range c.BankAccounts[:i] {
			if other.IBAN == a.IBAN {
				return fmt.Errorf("IBAN %s ist mehrfach angegeben", a.IBAN)
			}
			if other.IsInvoiceAccount && a.IsInvoiceAccount {
				return fmt.Errorf("nur ein Konto kann das Konto für Rechnungen sein")
			}
		}
	}
	return nil
}

// resolveBankAccounts vervollständigt die vorgeschlagenen Konten aus den
// vorhandenen: ein bekanntes Konto behält Buchungskonto, BIC und Bank, soweit
// der Vorschlag sie nicht nennt; ein neues bekommt das erste freie Konto von
// 1800 bis 1850.
func resolveBankAccounts(proposed, existing []domain.BankAccount) ([]domain.BankAccount, error) {
	byIBAN, used := map[string]domain.BankAccount{}, map[string]bool{}
	for _, a := range existing {
		byIBAN[a.IBAN], used[a.LedgerAccount] = a, true
	}
	out := make([]domain.BankAccount, 0, len(proposed))
	for _, a := range proposed {
		if old, ok := byIBAN[a.IBAN]; ok {
			if a.LedgerAccount == "" {
				a.LedgerAccount = old.LedgerAccount
			}
			if a.BIC == "" {
				a.BIC = old.BIC
			}
			if a.BankName == "" {
				a.BankName = old.BankName
			}
		}
		if a.LedgerAccount == "" {
			for _, n := range []string{"1800", "1810", "1820", "1830", "1840", "1850"} {
				if !used[n] {
					a.LedgerAccount = n
					break
				}
			}
			if a.LedgerAccount == "" {
				return out, fmt.Errorf("die Konten 1800 bis 1850 sind alle vergeben")
			}
		}
		used[a.LedgerAccount] = true
		if a.Currency == "" {
			a.Currency = "EUR"
		}
		out = append(out, a)
	}
	return out, nil
}

// foundationOf macht aus dem Vorschlag die Gründung; Eintragung und Kennung
// einer erfassten Gründung bleiben.
func foundationOf(p domain.FoundationProposal, state *FoundationState) *domain.Foundation {
	f := &domain.Foundation{ShareCapital: p.ShareCapital, FoundationCostCap: p.FoundationCostCap}
	if state != nil && state.Foundation != nil {
		f.ID = state.Foundation.ID
		f.RegisteredOn, f.RegisterCourt, f.RegisterNumber =
			state.Foundation.RegisteredOn, state.Foundation.RegisterCourt, state.Foundation.RegisterNumber
	}
	for _, sh := range p.Shareholders {
		f.Shareholders = append(f.Shareholders, domain.Shareholder{
			Name: strings.TrimSpace(sh.Name), ShareCapital: sh.ShareCapital, PaidIn: sh.PaidIn,
			Uncalled: sh.Uncalled, Kind: sh.Kind,
		})
	}
	return f
}

func foundationLines(p domain.FoundationProposal, foundedOn string, state *FoundationState) []MasterDataChangeLine {
	var before *domain.Foundation
	if state != nil {
		before = state.Foundation
	}
	lines := []MasterDataChangeLine{}
	add := func(field, b, a string) {
		if b != a {
			lines = append(lines, MasterDataChangeLine{Area: areaFoundation, Field: field, Before: b, After: a})
		}
	}
	var capital, costCap string
	var holders []string
	if before != nil {
		capital, costCap = euro(before.ShareCapital), euro(before.FoundationCostCap)
		for _, sh := range before.Shareholders {
			holders = append(holders, shareholderLine(sh.Name, sh.ShareCapital, sh.PaidIn, sh.Uncalled, sh.Kind))
		}
	}
	add("Beurkundung", germanDateOrEmpty(foundedOn), domain.GermanDate(p.NotarizedOn))
	add("Stammkapital", capital, euro(p.ShareCapital))
	add("Gründungsaufwand laut Satzung bis", costCap, euro(p.FoundationCostCap))
	for i, sh := range p.Shareholders {
		b := ""
		if i < len(holders) {
			b = holders[i]
		}
		add(fmt.Sprintf("Gesellschafter %d", i+1), b,
			shareholderLine(sh.Name, sh.ShareCapital, sh.PaidIn, sh.Uncalled, sh.Kind))
	}
	for i := len(p.Shareholders); i < len(holders); i++ {
		add(fmt.Sprintf("Gesellschafter %d", i+1), holders[i], "")
	}
	return lines
}

func shareholderLine(name string, share, paid, uncalled domain.Cents, kind domain.ContributionKind) string {
	if kind == "" {
		kind = domain.ContributionCash
	}
	line := fmt.Sprintf("%s: Anteil %s, eingezahlt %s", name, euro(share), euro(paid))
	if uncalled > 0 {
		line += ", nicht eingefordert " + euro(uncalled)
	}
	return line + ", " + kind.Label()
}

// invoiceAccount nennt das Konto für Rechnungen nach dem Vorschlag und das
// heutige. Ohne Angabe im Vorschlag bleibt es beim heutigen; gibt es keines,
// wird es das erste neue Konto, wie bei der Einrichtung in der App.
func invoiceAccount(accounts, existing []domain.BankAccount) (target, current string) {
	for _, a := range existing {
		if a.IsInvoiceAccount {
			current = a.IBAN
		}
	}
	target = current
	for _, a := range accounts {
		if a.IsInvoiceAccount {
			return a.IBAN, current
		}
	}
	if target == "" && len(accounts) > 0 {
		target = accounts[0].IBAN
	}
	return target, current
}

func bankLines(accounts, existing []domain.BankAccount) []MasterDataChangeLine {
	byIBAN := map[string]domain.BankAccount{}
	for _, a := range existing {
		byIBAN[a.IBAN] = a
	}
	target, current := invoiceAccount(accounts, existing)
	describe := func(a domain.BankAccount, invoice bool) string {
		parts := []string{a.Name}
		if a.BankName != "" {
			parts = append(parts, a.BankName)
		}
		if a.BIC != "" {
			parts = append(parts, "BIC "+a.BIC)
		}
		parts = append(parts, "Konto "+a.LedgerAccount)
		if invoice {
			parts = append(parts, "Konto für Rechnungen")
		}
		return strings.Join(parts, " · ")
	}
	lines := []MasterDataChangeLine{}
	add := func(iban, before, after string) {
		if before != after {
			lines = append(lines, MasterDataChangeLine{Area: areaBank, Field: formatIBAN(iban), Before: before, After: after})
		}
	}
	proposed := map[string]bool{}
	for _, a := range accounts {
		proposed[a.IBAN] = true
		before := ""
		if old, known := byIBAN[a.IBAN]; known {
			before = describe(old, old.IsInvoiceAccount)
		}
		add(a.IBAN, before, describe(a, a.IBAN == target))
	}
	if current != "" && current != target && !proposed[current] {
		old := byIBAN[current]
		add(current, describe(old, true), describe(old, false))
	}
	return lines
}

// companyFields sind die Felder der Unternehmensdaten, die ein Vorschlag
// ändern kann, mit ihrer Darstellung in der Vorschau.
var companyFields = []struct {
	label string
	show  func(*domain.CompanySettings) string
}{
	{"Firma", func(s *domain.CompanySettings) string { return s.CompanyName }},
	{"Geschäftsführung", func(s *domain.CompanySettings) string { return s.ManagingDirectors }},
	{"Vorsitz des Aufsichtsrats", func(s *domain.CompanySettings) string { return s.SupervisoryBoardChair }},
	{"Straße", func(s *domain.CompanySettings) string { return s.Street }},
	{"Hausnummer", func(s *domain.CompanySettings) string { return s.HouseNumber }},
	{"Adresszusatz", func(s *domain.CompanySettings) string { return s.AddressAddition }},
	{"PLZ", func(s *domain.CompanySettings) string { return s.PostalCode }},
	{"Ort", func(s *domain.CompanySettings) string { return s.City }},
	{"Land", func(s *domain.CompanySettings) string { return s.CountryCode }},
	{"Ansprechpartner", func(s *domain.CompanySettings) string { return s.ContactName }},
	{"Telefon", func(s *domain.CompanySettings) string { return s.ContactPhone }},
	{"E-Mail", func(s *domain.CompanySettings) string { return s.ContactEmail }},
	{"Notar", func(s *domain.CompanySettings) string { return s.Notary }},
	{"Urkundenrollennummer", func(s *domain.CompanySettings) string { return s.DeedNumber }},
	{"Sitz", func(s *domain.CompanySettings) string { return s.Seat }},
	{"Registergericht", func(s *domain.CompanySettings) string { return s.RegisterCourt }},
	{"Registernummer", func(s *domain.CompanySettings) string { return s.RegisterNumber }},
	{"Eingetragen am", func(s *domain.CompanySettings) string { return germanDateOrEmpty(s.RegisteredOn) }},
	{"Steuernummer", func(s *domain.CompanySettings) string { return s.TaxNumber }},
	{"Finanzamt", func(s *domain.CompanySettings) string { return s.TaxOffice }},
	{"Bundesland", func(s *domain.CompanySettings) string { return calendar.StateName(s.FederalState) }},
	{"USt-IdNr.", func(s *domain.CompanySettings) string { return s.VatID }},
	{"Voranmeldungszeitraum", func(s *domain.CompanySettings) string { return vatPeriodLabels[s.VatPeriod] }},
	{"Besteuerung", func(s *domain.CompanySettings) string { return taxationLabels[s.TaxationType] }},
	{"Dauerfristverlängerung", func(s *domain.CompanySettings) string { return yesNo(s.PermanentExtension) }},
	{"Sondervorauszahlung", func(s *domain.CompanySettings) string { return euro(s.SpecialPrepayment) }},
	{"Steuerliche Beratung", func(s *domain.CompanySettings) string { return taxAdvisorLabel(s.TaxAdvisor) }},
	{"Stammkapital", func(s *domain.CompanySettings) string { return euro(s.ShareCapital) }},
	{"Gesellschafter", func(s *domain.CompanySettings) string {
		parts := make([]string, 0, len(s.Shareholders))
		for _, sh := range s.Shareholders {
			parts = append(parts, fmt.Sprintf("%s (%s)", sh.Name, euro(sh.ShareCapital)))
		}
		return strings.Join(parts, ", ")
	}},
}

func companyLines(before, after *domain.CompanySettings) []MasterDataChangeLine {
	lines := []MasterDataChangeLine{}
	for _, f := range companyFields {
		if b, a := f.show(before), f.show(after); b != a {
			lines = append(lines, MasterDataChangeLine{Area: areaCompany, Field: f.label, Before: b, After: a})
		}
	}
	return lines
}

var vatPeriodLabels = map[string]string{
	"month": "monatlich", "quarter": "vierteljährlich", "year": "jährlich", "none": "keine Voranmeldung",
	"unknown": "", "": "",
}

var taxationLabels = map[string]string{"SOLL": "Sollversteuerung", "IST": "Istversteuerung", "": ""}

func taxAdvisorLabel(a domain.TaxAdvisor) string {
	if !a.Engaged {
		return "keine"
	}
	label := "Steuerberater"
	if a.SinceYear > 0 {
		label += fmt.Sprintf(" ab %d", a.SinceYear)
	}
	if len(a.Declarations) > 0 {
		label += " für " + strings.Join(a.Declarations, ", ")
	}
	return label
}

func summarizeLines(lines []MasterDataChangeLine) string {
	parts := make([]string, 0, len(lines))
	for _, l := range lines {
		parts = append(parts, l.Area+" › "+l.Field)
	}
	return strings.Join(parts, ", ")
}

func euro(c domain.Cents) string { return c.String() + " €" }

func yesNo(b bool) string {
	if b {
		return "ja"
	}
	return "nein"
}

func germanDateOrEmpty(iso string) string {
	if iso == "" {
		return ""
	}
	return domain.GermanDate(iso)
}

// formatIBAN gliedert eine IBAN in Vierergruppen.
func formatIBAN(iban string) string {
	var b strings.Builder
	for i, r := range iban {
		if i > 0 && i%4 == 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}
