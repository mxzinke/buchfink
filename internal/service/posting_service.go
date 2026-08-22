// SPDX-FileCopyrightText: 2026 Maximilian Pfennig
// SPDX-License-Identifier: EUPL-1.2

package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// SettlementKind separates *when* something is paid from *how* it is paid.
//
// The concept originally called an immediately settled receipt "bar", which
// silently implied the bank account. Cash is Kasse (1600) and has its own rules;
// a card payment is a different account again. Payment timing and payment means
// are therefore two independent inputs.
type SettlementKind string

const (
	// SettlementOpen books against the partner's Personenkonto and leaves an
	// open item.
	SettlementOpen SettlementKind = "open"
	// SettlementPaid books directly against a liquid account.
	SettlementPaid SettlementKind = "paid"
)

// ReceiptPosition is one net amount of an incoming document, classified by a
// fachliche Gruppe (or a directly chosen account) and a VAT rate.
type ReceiptPosition struct {
	PostingGroup string         `json:"postingGroup"`
	Account      string         `json:"account,omitempty"` // direkte Kontowahl statt Gruppe
	Net          domain.Cents   `json:"net"`
	TaxRate      domain.TaxRate `json:"taxRate"`
	Text         string         `json:"text,omitempty"`
}

// ReceiptRequest is the complete input for booking an incoming document.
//
// These are the eight facts a booking cannot be derived without: direction (implied
// by the type), partner, Belegdatum, Leistungszeitraum, net amounts per VAT
// rate, Steuerfall, fachliche Gruppe and how it is settled.
type ReceiptRequest struct {
	ContactID       uint                `json:"contactId"`
	BookingDate     string              `json:"bookingDate"`
	DocumentDate    string              `json:"documentDate"`
	ServiceDateFrom string              `json:"serviceDateFrom"`
	ServiceDateTo   string              `json:"serviceDateTo"`
	DocumentNumber  string              `json:"documentNumber"`
	DocumentHash    string              `json:"documentHash,omitempty"`
	DocumentPath    string              `json:"documentPath,omitempty"`
	Description     string              `json:"description"`
	TaxTreatment    domain.TaxTreatment `json:"taxTreatment"`
	Positions       []ReceiptPosition   `json:"positions"`
	Settlement      SettlementKind      `json:"settlement"`
	PaymentAccount  string              `json:"paymentAccount,omitempty"`
	Currency        string              `json:"currency,omitempty"`
}

// PostingService turns business documents into journal entries using the
// deterministic Gruppe → Konten mapping.
type PostingService struct {
	journalSvc  *JournalService
	contactRepo domain.ContactRepository
	taxResolver domain.TaxResolver
}

// NewPostingService creates the posting service.
func NewPostingService(journalSvc *JournalService, contactRepo domain.ContactRepository) *PostingService {
	return &PostingService{
		journalSvc:  journalSvc,
		contactRepo: contactRepo,
		taxResolver: journalSvc.TaxResolver(),
	}
}

// PostIncomingReceipt books an Eingangsbeleg.
func (s *PostingService) PostIncomingReceipt(ctx context.Context, req ReceiptRequest) (*domain.JournalEntry, error) {
	if len(req.Positions) == 0 {
		return nil, fmt.Errorf("der Beleg hat keine Positionen")
	}
	if req.TaxTreatment == "" {
		req.TaxTreatment = domain.TaxTreatmentDomestic
	}

	contact, err := s.contactRepo.FindByID(ctx, req.ContactID)
	if err != nil {
		return nil, fmt.Errorf("Lieferant konnte nicht geladen werden: %w", err)
	}
	if contact.Type != domain.ContactTypeVendor {
		return nil, fmt.Errorf("%s ist als Kunde angelegt und kann keinen Eingangsbeleg stellen", contact.Name)
	}
	if err := validateIncomingTreatment(req.TaxTreatment, contact); err != nil {
		return nil, err
	}

	var lines []domain.JournalLine

	// 1. Aufwands- bzw. Anschaffungszeilen aus den fachlichen Gruppen.
	netByRate := map[domain.TaxRate]domain.Cents{}
	for i, p := range req.Positions {
		if p.Net <= 0 {
			return nil, fmt.Errorf("Position %d: der Nettobetrag muss größer als null sein", i+1)
		}
		account, err := s.resolveAccount(p, domain.DirectionIncoming, req.TaxTreatment)
		if err != nil {
			return nil, fmt.Errorf("Position %d: %w", i+1, err)
		}
		lines = append(lines, domain.JournalLine{
			Side: domain.SideDebit, Account: account, Amount: p.Net, Text: p.Text,
		})
		netByRate[p.TaxRate] += p.Net
	}

	// 2. Steuerzeilen, einmal je Steuersatzgruppe gerundet.
	taxLines, err := s.taxLines(domain.DirectionIncoming, req.TaxTreatment, netByRate)
	if err != nil {
		return nil, err
	}
	lines = append(lines, taxLines...)

	// 3. Gegenzeile: was tatsächlich an den Lieferanten zu zahlen ist.
	settlementLine, err := s.settlementLine(lines, req.Settlement, req.PaymentAccount, contact)
	if err != nil {
		return nil, err
	}
	lines = append(lines, settlementLine)

	description := req.Description
	if description == "" {
		description = fmt.Sprintf("Eingangsbeleg %s, %s", req.DocumentNumber, contact.Name)
	}

	entry := &domain.JournalEntry{
		BookingDate:        req.BookingDate,
		DocumentDate:       req.DocumentDate,
		ServiceDateFrom:    req.ServiceDateFrom,
		ServiceDateTo:      req.ServiceDateTo,
		Description:        description,
		Source:             domain.EntrySourceReceipt,
		DocumentNumber:     req.DocumentNumber,
		DocumentHash:       req.DocumentHash,
		DocumentPath:       req.DocumentPath,
		ContactID:          &contact.ID,
		Currency:           req.Currency,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
	}

	return s.journalSvc.Post(ctx, entry)
}

// PostOutgoingInvoice books an Ausgangsrechnung as a receivable.
func (s *PostingService) PostOutgoingInvoice(ctx context.Context, inv *domain.Invoice, contact *domain.Contact) (*domain.JournalEntry, error) {
	group, err := accounting.LookupPostingGroup("erloese")
	if err != nil {
		return nil, err
	}

	var lines []domain.JournalLine
	netByRate := map[domain.TaxRate]domain.Cents{}

	for i := range inv.Items {
		item := &inv.Items[i]
		g := group
		if item.PostingGroup != "" {
			g, err = accounting.LookupPostingGroup(item.PostingGroup)
			if err != nil {
				return nil, fmt.Errorf("Position %d: %w", i+1, err)
			}
			if g.Direction != domain.DirectionOutgoing {
				return nil, fmt.Errorf("Position %d: %q ist keine Ertragsgruppe", i+1, g.Label)
			}
		}
		lines = append(lines, domain.JournalLine{
			Side:    domain.SideCredit,
			Account: g.ResolveAccount(inv.TaxTreatment, item.TaxRate),
			Amount:  item.TotalNet(),
			Text:    item.Description,
		})
		netByRate[item.TaxRate] += item.TotalNet()
	}

	taxLines, err := s.taxLines(domain.DirectionOutgoing, inv.TaxTreatment, netByRate)
	if err != nil {
		return nil, err
	}
	lines = append(lines, taxLines...)

	// An issued invoice is always an open item; the payment is a later,
	// separate business transaction.
	settlementLine, err := s.settlementLine(lines, SettlementOpen, "", contact)
	if err != nil {
		return nil, err
	}
	lines = append(lines, settlementLine)

	entry := &domain.JournalEntry{
		FiscalYear:         inv.FiscalYear,
		BookingDate:        inv.Date,
		DocumentDate:       inv.Date,
		ServiceDateFrom:    inv.ServiceDateFrom,
		ServiceDateTo:      inv.ServiceDateTo,
		Description:        fmt.Sprintf("Rechnung %s an %s", inv.InvoiceNumber, contact.Name),
		Source:             domain.EntrySourceInvoice,
		DocumentNumber:     inv.InvoiceNumber,
		ContactID:          &contact.ID,
		Currency:           inv.Currency,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
	}

	return s.journalSvc.Post(ctx, entry)
}

func (s *PostingService) resolveAccount(p ReceiptPosition, dir domain.Direction, treatment domain.TaxTreatment) (string, error) {
	if p.Account != "" {
		// A directly chosen account still has to pass the journal's checks; this
		// is the escape hatch for cases the group catalog does not cover.
		return p.Account, nil
	}
	if p.PostingGroup == "" {
		return "", fmt.Errorf("weder eine Buchungsgruppe noch ein Konto angegeben")
	}
	group, err := accounting.LookupPostingGroup(p.PostingGroup)
	if err != nil {
		return "", err
	}
	if group.Direction != dir {
		return "", fmt.Errorf("%q passt nicht zur Belegrichtung", group.Label)
	}
	return group.ResolveAccount(treatment, p.TaxRate), nil
}

// taxLines produces the tax legs, rounding once per rate group.
func (s *PostingService) taxLines(dir domain.Direction, treatment domain.TaxTreatment, netByRate map[domain.TaxRate]domain.Cents) ([]domain.JournalLine, error) {
	rates := make([]domain.TaxRate, 0, len(netByRate))
	for r := range netByRate {
		rates = append(rates, r)
	}
	sort.Slice(rates, func(i, j int) bool { return rates[i] < rates[j] })

	var lines []domain.JournalLine
	for _, rate := range rates {
		legs, err := s.taxResolver.Resolve(dir, treatment, rate, netByRate[rate])
		if err != nil {
			return nil, err
		}
		for _, leg := range legs {
			if leg.Amount == 0 {
				continue
			}
			lines = append(lines, domain.JournalLine{
				Side:    leg.Side,
				Account: leg.Account,
				Amount:  leg.Amount,
				TaxKey:  leg.Key,
				TaxBase: leg.Base,
			})
		}
	}
	return lines, nil
}

// settlementLine closes the entry against the partner's Personenkonto or a
// liquid account.
//
// Its amount is whatever is needed to balance the entry, which makes it correct
// for every Steuerfall without a special case: a domestic purchase settles at
// net plus input tax, a Reverse-Charge purchase settles at net, because there
// the input tax leg is cancelled out by the output tax leg.
func (s *PostingService) settlementLine(content []domain.JournalLine, kind SettlementKind, paymentAccount string, contact *domain.Contact) (domain.JournalLine, error) {
	var debit, credit domain.Cents
	for _, l := range content {
		if l.Side == domain.SideDebit {
			debit += l.Amount
		} else {
			credit += l.Amount
		}
	}

	difference := debit - credit
	if difference == 0 {
		return domain.JournalLine{}, fmt.Errorf("der Beleg hat einen Gesamtbetrag von null")
	}

	side := domain.SideCredit
	if difference < 0 {
		side = domain.SideDebit
		difference = -difference
	}

	switch kind {
	case SettlementOpen:
		return domain.JournalLine{
			Side:      side,
			Account:   contact.LedgerAccount,
			ContactID: &contact.ID,
			Amount:    difference,
		}, nil
	case SettlementPaid:
		if paymentAccount == "" {
			return domain.JournalLine{}, fmt.Errorf("bei sofortiger Zahlung muss das Zahlungsmittel angegeben werden (Kasse, Bank, Kreditkarte)")
		}
		if !isLiquidAccount(paymentAccount) {
			return domain.JournalLine{}, fmt.Errorf("Konto %s ist kein Zahlungsmittelkonto", paymentAccount)
		}
		return domain.JournalLine{Side: side, Account: paymentAccount, Amount: difference}, nil
	default:
		return domain.JournalLine{}, fmt.Errorf("unbekannte Zahlungsart %q", kind)
	}
}

func isLiquidAccount(account string) bool {
	for _, a := range domain.LiquidAccounts() {
		if a == account {
			return true
		}
	}
	return false
}

// validateIncomingTreatment blocks Steuerfälle the partner's master data does
// not support. Reverse Charge without a VAT identification number on file is the
// single most common source of a wrong Umsatzsteuer-Voranmeldung.
func validateIncomingTreatment(treatment domain.TaxTreatment, contact *domain.Contact) error {
	switch treatment {
	case domain.TaxTreatmentIntraCommunityAcquisition:
		if !contact.IsEUCounterparty() {
			return fmt.Errorf("ein innergemeinschaftlicher Erwerb setzt einen Lieferanten in einem anderen EU-Land voraus, %s ist in %q erfasst", contact.Name, contact.CountryCode)
		}
		if contact.VatID == "" {
			return fmt.Errorf("für einen innergemeinschaftlichen Erwerb braucht %s eine USt-IdNr.", contact.Name)
		}
	case domain.TaxTreatmentReverseCharge:
		// § 13b UStG hat zwei getrennte Fälle. Bei einer sonstigen Leistung eines
		// im übrigen Gemeinschaftsgebiet ansässigen Unternehmers (Abs. 1) belegt
		// die USt-IdNr. die Unternehmereigenschaft im Ausland. Bei einer
		// inländischen Bauleistung (Abs. 2 Nr. 4) hängt die Steuerschuld dagegen
		// am *Leistungsempfänger* – er muss selbst nachhaltig Bauleistungen
		// erbringen (Abs. 5). Das steht nicht in den Lieferantenstammdaten, also
		// darf eine fehlende USt-IdNr. des inländischen Lieferanten den Fall auch
		// nicht blockieren.
		if contact.IsEUCounterparty() && contact.VatID == "" {
			return fmt.Errorf(
				"für § 13b UStG braucht der ausländische Lieferant %s eine USt-IdNr., die seine Unternehmereigenschaft belegt",
				contact.Name)
		}
	}
	return nil
}
