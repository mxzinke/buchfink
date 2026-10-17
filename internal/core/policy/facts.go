package policy

import (
	"context"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// LimitsFor liest die Schwellen des Mandanten.
func LimitsFor(ctx context.Context, s *core.Services) (Limits, error) {
	settings, err := s.Settings.AgentSettings(ctx)
	if err != nil {
		return Limits{}, err
	}
	return LimitsFrom(settings), nil
}

// ClassifyReceiptRequest sammelt die Fakten einer Belegbuchung und urteilt.
// Ein Fehler heißt: die Buchung ginge auch mit Rückfrage nicht so durch
// (Beleg fehlt, Vorschau scheitert); er kommt aus dem Dienst.
func ClassifyReceiptRequest(ctx context.Context, s *core.Services, req service.ReceiptRequest, today string) (Verdict, error) {
	limits, err := LimitsFor(ctx, s)
	if err != nil {
		return Verdict{}, err
	}
	f, err := ReceiptFactsFor(ctx, s, req, today)
	if err != nil {
		return Verdict{}, err
	}
	return ClassifyReceipt(f, limits), nil
}

// ReceiptFactsFor sammelt die Fakten einer Belegbuchung.
func ReceiptFactsFor(ctx context.Context, s *core.Services, req service.ReceiptRequest, today string) (ReceiptFacts, error) {
	f := ReceiptFacts{
		Treatment: req.TaxTreatment, Currency: strings.ToUpper(strings.TrimSpace(req.Currency)),
		Settlement: string(req.Settlement), BookingDate: req.BookingDate, DocumentDate: req.DocumentDate,
		Today: today, ContactID: req.ContactID, Net: map[domain.TaxRate]domain.Cents{},
	}
	receipt, err := s.Receipts.Get(ctx, req.ReceiptID)
	if err != nil {
		return f, err
	}
	f.Open = receipt.IsOpen()
	f.Incoming = receipt.Direction == domain.DirectionIncoming
	f.Invoice = receipt.Kind == "" || receipt.Kind == domain.ReceiptKindInvoice
	_, f.Structured = receipt.FileByRole(domain.ReceiptRoleStructured)

	if findings, err := s.Receipts.Findings(ctx, receipt.ID); err == nil && findings != nil {
		f.Checked, f.Findings = findings.Checked, findings.Total
	}
	if f.Structured {
		proposal, err := s.EInvoice.Propose(ctx, receipt.ID)
		if err != nil {
			f.ProposalError = err.Error()
		} else {
			if proposal.MatchedContact {
				f.ProposalContactID = proposal.Request.ContactID
			}
			f.ProposalTreatment = proposal.Request.TaxTreatment
			f.ProposalCurrency = strings.ToUpper(strings.TrimSpace(proposal.Request.Currency))
			f.ProposalGross = proposal.GrossAmount
			f.ProposalNet = map[domain.TaxRate]domain.Cents{}
			for _, p := range proposal.Request.Positions {
				f.ProposalNet[p.TaxRate] += p.Net
			}
		}
	}

	contact, err := s.Repos.Contacts.FindByID(ctx, req.ContactID)
	if err != nil {
		return f, err
	}
	f.ContactBlocked = contact.Blocked
	if contact.DefaultPostingGroup != "" && contact.DefaultPostingGroupConfirmedAt != "" {
		f.ConfirmedGroup = contact.DefaultPostingGroup
		if f.BookingsWithGroup, err = bookingsWithGroup(ctx, s, contact.ID, f.ConfirmedGroup); err != nil {
			return f, err
		}
	}

	for _, p := range req.Positions {
		f.Net[p.TaxRate] += p.Net
		if p.PostingGroup != "" {
			f.Groups = append(f.Groups, p.PostingGroup)
		}
		f.Special = append(f.Special, positionSpecials(p)...)
	}
	f.Special = append(f.Special, requestSpecials(req)...)

	// Eine Fremdwährung ist ohnehin Rückfrage; ihre Vorschau holte einen Kurs
	// aus dem Netz, und das gehört nicht unter die Schreibsperre.
	if isEuro(f.Currency) {
		preview, err := s.Posting.PreviewIncomingReceipt(ctx, req)
		if err != nil {
			return f, err
		}
		f.Gross = preview.Gross
		if len(preview.InputTaxFindings) > 0 {
			f.Special = append(f.Special, "befund_vorsteuer")
		}
	}
	f.DuplicateOf, err = duplicateOf(ctx, s, receipt, contact)
	return f, err
}

func positionSpecials(p service.ReceiptPosition) []string {
	var out []string
	if p.Account != "" {
		out = append(out, "konto_statt_gruppe")
	}
	if p.InputTaxShare != 0 && p.InputTaxShare != 1000 {
		out = append(out, "vorsteueranteil")
	}
	if p.Gift != nil {
		out = append(out, "geschenk")
	}
	if p.PostingGroup == "" {
		return out
	}
	g, err := accounting.LookupPostingGroup(p.PostingGroup)
	if err != nil {
		return append(out, "unbekannte_gruppe")
	}
	if g.Key == "gwg" || g.RecipientRequired || g.Limit != "" || g.InputTaxExcluded ||
		g.DeductibleQuota != "" || g.NonDeductibleAccount != "" {
		out = append(out, "gruppe_mit_sonderregel")
	}
	return out
}

func requestSpecials(req service.ReceiptRequest) []string {
	var out []string
	if strings.TrimSpace(req.OverrideReason) != "" {
		out = append(out, "uebersteuerung")
	}
	if req.Entertainment != nil {
		out = append(out, "bewirtung")
	}
	if req.ProvisionID != 0 {
		out = append(out, "rueckstellung")
	}
	if req.AdvanceTarget != "" || len(req.SettledAdvanceIDs) > 0 {
		out = append(out, "anzahlung")
	}
	if req.CreditorContactID != 0 {
		out = append(out, "auslage")
	}
	if req.ForeignAmount != 0 {
		out = append(out, "fremdbetrag")
	}
	return out
}

// bookingsWithGroup zählt die nicht stornierten Belegbuchungen eines
// Lieferanten auf den Konten seiner Gruppe im Inland.
func bookingsWithGroup(ctx context.Context, s *core.Services, contactID uint, key string) (int, error) {
	group, err := accounting.LookupPostingGroup(key)
	if err != nil {
		return 0, nil
	}
	accounts := map[string]bool{group.Account: true}
	for _, rate := range []domain.TaxRate{domain.TaxRateStandard, domain.TaxRateReduced, domain.TaxRateNone} {
		accounts[group.ResolveAccount(domain.TaxTreatmentDomestic, rate)] = true
	}
	entries, err := s.Accounting.GetAllEntries(ctx)
	if err != nil {
		return 0, err
	}
	reversed := map[uint]bool{}
	for _, e := range entries {
		if e.ReversalOfID != nil {
			reversed[*e.ReversalOfID] = true
		}
	}
	n := 0
	for _, e := range entries {
		if e.ContactID == nil || *e.ContactID != contactID || e.Kind != domain.EntryKindNormal ||
			e.Source != domain.EntrySourceReceipt || reversed[e.ID] {
			continue
		}
		for _, l := range e.Lines {
			if l.Side == domain.SideDebit && accounts[l.Account] {
				n++
				break
			}
		}
	}
	return n, nil
}

// duplicateOf sucht einen anderen Eingangsbeleg desselben Ausstellers mit
// derselben Rechnungsnummer oder demselben Betrag am selben Tag.
func duplicateOf(ctx context.Context, s *core.Services, receipt *domain.Receipt, contact *domain.Contact) (string, error) {
	issuers := map[string]bool{normalizeName(receipt.IssuerName): true, normalizeName(contact.Name): true,
		normalizeName(contact.Company): true}
	delete(issuers, "")
	for _, year := range []int{receipt.FiscalYear, receipt.FiscalYear - 1} {
		if year <= 0 {
			continue
		}
		others, err := s.Receipts.List(ctx, year, "")
		if err != nil {
			return "", err
		}
		for _, o := range others {
			if o.ID == receipt.ID || o.Status == domain.ReceiptStatusDiscarded || o.Direction != domain.DirectionIncoming {
				continue
			}
			if !issuers[normalizeName(o.IssuerName)] {
				continue
			}
			sameNumber := receipt.Subject != "" && strings.EqualFold(strings.TrimSpace(o.Subject), strings.TrimSpace(receipt.Subject))
			sameAmount := receipt.GrossAmount != 0 && o.GrossAmount == receipt.GrossAmount && o.DocumentDate == receipt.DocumentDate
			if sameNumber || sameAmount {
				return o.ReceiptNumber, nil
			}
		}
	}
	return "", nil
}

// ClassifyBankRequest sammelt die Fakten einer Zuordnung und urteilt.
func ClassifyBankRequest(ctx context.Context, s *core.Services, bankTxID, entryID uint) (Verdict, error) {
	f, err := BankFactsFor(ctx, s, bankTxID, entryID)
	if err != nil {
		return Verdict{}, err
	}
	return ClassifyBankMatch(f), nil
}

// BankFactsFor sammelt die Fakten einer Zuordnung: alle offenen Posten mit
// genau dem Betrag des Umsatzes in derselben Richtung und die Nummer, die
// der Verwendungszweck von ihnen nennt.
func BankFactsFor(ctx context.Context, s *core.Services, bankTxID, entryID uint) (BankFacts, error) {
	f := BankFacts{Requested: entryID}
	tx, err := s.Bank.Transaction(ctx, bankTxID)
	if err != nil {
		return f, err
	}
	f.Open = tx.MatchStatus == "" || tx.MatchStatus == domain.MatchStatusUnmatched
	f.Currency = strings.ToUpper(strings.TrimSpace(tx.Currency))
	items, err := s.Payments.OpenItems(ctx)
	if err != nil {
		return f, err
	}
	for _, item := range items {
		if !sameDirection(tx.Amount, item) || item.OpenAmount != tx.Amount.Abs() {
			continue
		}
		numbers := []string{item.DocumentNumber, item.EntryNumber}
		if invoiceNo := supplierInvoiceNumber(ctx, s, item); invoiceNo != "" {
			numbers = append(numbers, invoiceNo)
		}
		f.Candidates = append(f.Candidates, BankCandidate{
			EntryID: item.EntryID, Reference: accounting.MatchedReference(tx.RemittanceInfo, numbers...),
		})
	}
	return f, nil
}

// SameDirection meldet, ob ein Bankumsatz einen offenen Posten ausgleichen
// kann: Geld herein für eine Forderung, Geld hinaus für eine Verbindlichkeit.
func sameDirection(amount domain.Cents, item domain.OpenItem) bool {
	if item.OpenAmount <= 0 {
		return false
	}
	switch item.ContactType {
	case domain.ContactTypeCustomer:
		return amount > 0
	case domain.ContactTypeVendor:
		return amount < 0
	}
	return false
}

// supplierInvoiceNumber ist die Rechnungsnummer des Lieferanten zu einem
// offenen Posten aus einem Eingangsbeleg; sie steht im Betreff des Belegs.
func supplierInvoiceNumber(ctx context.Context, s *core.Services, item domain.OpenItem) string {
	if item.ContactType != domain.ContactTypeVendor {
		return ""
	}
	entry, err := s.Repos.Journal.FindByID(ctx, item.EntryID)
	if err != nil || entry.ReceiptID == nil {
		return ""
	}
	receipt, err := s.Receipts.Get(ctx, *entry.ReceiptID)
	if err != nil {
		return ""
	}
	return receipt.Subject
}

func normalizeName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == 'ä' || r == 'ö' || r == 'ü' || r == 'ß' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
