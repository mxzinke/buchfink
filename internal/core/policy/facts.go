package policy

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/einvoice"
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

// ClassifyProposal stuft die Buchung ein, die ein Vorschlag aus der
// E-Rechnung mit der bestätigten Standardgruppe des Lieferanten ergäbe. Ohne
// bestätigte Gruppe lässt sich die Buchung nicht durchrechnen; die übrigen
// Regeln gelten trotzdem, und der Brutto kommt aus der E-Rechnung.
func ClassifyProposal(ctx context.Context, s *core.Services, req service.ReceiptRequest, today string) Verdict {
	req.Positions = append([]service.ReceiptPosition(nil), req.Positions...)
	grouped := false
	if contact, err := s.Repos.Contacts.FindByID(ctx, req.ContactID); err == nil && contact.DefaultPostingGroupConfirmedAt != "" {
		for i := range req.Positions {
			req.Positions[i].PostingGroup = contact.DefaultPostingGroup
		}
		grouped = true
	}
	limits, err := LimitsFor(ctx, s)
	if err != nil {
		return Verdict{Level: Confirm, Reasons: []Reason{{Key: "vorschau", Text: err.Error()}}}
	}
	f, err := receiptFacts(ctx, s, req, today, grouped)
	if err != nil {
		return Verdict{Level: Confirm, Reasons: []Reason{{Key: "vorschau", Text: err.Error()}}}
	}
	return ClassifyReceipt(f, limits)
}

// ReceiptFactsFor sammelt die Fakten einer Belegbuchung.
func ReceiptFactsFor(ctx context.Context, s *core.Services, req service.ReceiptRequest, today string) (ReceiptFacts, error) {
	return receiptFacts(ctx, s, req, today, true)
}

// receiptFacts sammelt die Fakten; ohne preview wird nicht durchgerechnet und
// der Brutto der E-Rechnung genommen.
func receiptFacts(ctx context.Context, s *core.Services, req service.ReceiptRequest, today string, preview bool) (ReceiptFacts, error) {
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
		f.Checked = findings.Checked
		// Ein Hinweis (etwa: Kleinbetragsrechnung, darum weniger
		// Pflichtangaben) ist kein Befund; jede Warnung und jeder Fehler ist einer.
		for _, g := range findings.Groups {
			for _, x := range g.Findings {
				if x.Severity != string(einvoice.SeverityInfo) {
					f.Findings++
				}
			}
		}
	}
	var proposal *service.EInvoiceProposal
	if f.Structured {
		proposal, err = s.EInvoice.Propose(ctx, receipt.ID)
		if err != nil {
			f.ProposalError, proposal = err.Error(), nil
		} else {
			if proposal.MatchedContact {
				f.ProposalContactID = proposal.Request.ContactID
				f.ProposalMatchedBy = proposal.MatchedBy
			}
			f.HeaderMismatch = headerMismatch(receipt, proposal)
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
	if proposal != nil {
		f.MasterDataMismatch, f.IBANMismatch = masterDataMismatch(contact, proposal)
	}
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
	switch {
	case !preview:
		f.Gross = f.ProposalGross
	case isEuro(f.Currency):
		preview, err := s.Posting.PreviewIncomingReceipt(ctx, req)
		if err != nil {
			return f, err
		}
		f.Gross = preview.Gross
		if len(preview.InputTaxFindings) > 0 {
			f.Special = append(f.Special, "befund_vorsteuer")
		}
	}
	f.DuplicateOf, err = duplicateOf(ctx, s, receipt, contact, proposal)
	return f, err
}

// headerMismatch nennt die Kopfdaten, die nicht aus der E-Rechnung stammen.
// Wer sie anders setzt, als die E-Rechnung sagt, umginge sonst die
// Dublettenprüfung oder verschöbe den Beleg in ein anderes Jahr.
func headerMismatch(r *domain.Receipt, p *service.EInvoiceProposal) []string {
	var out []string
	if r.DocumentDate != p.Request.DocumentDate {
		out = append(out, "belegdatum")
	}
	if r.GrossAmount != p.GrossAmount {
		out = append(out, "brutto")
	}
	if normalizeName(r.IssuerName) != normalizeName(p.SupplierName) {
		out = append(out, "aussteller")
	}
	if subject := strings.TrimSpace(r.Subject); subject != "" && !strings.EqualFold(subject, strings.TrimSpace(p.InvoiceNumber)) {
		out = append(out, "betreff")
	}
	return out
}

// masterDataMismatch vergleicht den Aussteller der E-Rechnung mit den
// Stammdaten des Kontakts. Was beim Kontakt fehlt, widerspricht nicht.
func masterDataMismatch(c *domain.Contact, p *service.EInvoiceProposal) ([]string, bool) {
	var out []string
	differs := func(a, b string) bool {
		a, b = domain.NormalizeTaxIdentifier(a), domain.NormalizeTaxIdentifier(b)
		return a != "" && b != "" && a != b
	}
	if differs(c.VatID, p.SupplierVatID) {
		out = append(out, "ust_id")
	}
	if differs(c.TaxID, p.SupplierTaxID) {
		out = append(out, "steuernummer")
	}
	if differs(c.PostalCode, p.SupplierPostCode) ||
		(c.City != "" && p.SupplierCity != "" && normalizeName(c.City) != normalizeName(p.SupplierCity)) {
		out = append(out, "anschrift")
	}
	iban := domain.NormalizeIBAN(c.IBAN)
	if iban == "" || len(p.PaymentAccounts) == 0 {
		return out, false
	}
	for _, a := range p.PaymentAccounts {
		if domain.NormalizeIBAN(a) == iban {
			return out, false
		}
	}
	return out, true
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
// derselben Rechnungsnummer oder demselben Betrag am selben Tag. Bei einer
// E-Rechnung zählen Nummer, Betrag und Datum aus dem strukturierten Teil,
// nicht die Kopfdaten: die lassen sich beim Ablegen frei setzen. Die Nummer
// wird über alle Geschäftsjahre gesucht.
func duplicateOf(ctx context.Context, s *core.Services, receipt *domain.Receipt, contact *domain.Contact,
	proposal *service.EInvoiceProposal) (string, error) {
	number, gross, date := receipt.Subject, receipt.GrossAmount, receipt.DocumentDate
	ids := map[string]bool{domain.NormalizeTaxIdentifier(contact.VatID): true, domain.NormalizeTaxIdentifier(contact.TaxID): true}
	issuers := map[string]bool{normalizeName(receipt.IssuerName): true, normalizeName(contact.Name): true,
		normalizeName(contact.Company): true}
	if proposal != nil {
		number, gross, date = proposal.InvoiceNumber, proposal.GrossAmount, proposal.Request.DocumentDate
		ids[domain.NormalizeTaxIdentifier(proposal.SupplierVatID)] = true
		ids[domain.NormalizeTaxIdentifier(proposal.SupplierTaxID)] = true
		issuers[normalizeName(proposal.SupplierName)] = true
	}
	delete(ids, "")
	delete(issuers, "")
	other := func(o *domain.Receipt) bool {
		return o.ID != receipt.ID && o.Status != domain.ReceiptStatusDiscarded && o.Direction == domain.DirectionIncoming
	}

	if strings.TrimSpace(number) != "" {
		same, err := s.Receipts.FindByEInvoiceNumber(ctx, number)
		if err != nil {
			return "", err
		}
		for i := range same {
			o := &same[i]
			if !other(o) {
				continue
			}
			if ids[o.EInvoiceIssuer] || (o.EInvoiceIssuer == "" && issuers[normalizeName(o.IssuerName)]) ||
				bookedTo(ctx, s, o, contact.ID) {
				return o.ReceiptNumber, nil
			}
		}
	}

	years := map[int]bool{receipt.FiscalYear: true}
	if len(date) >= 4 {
		if y, err := strconv.Atoi(date[:4]); err == nil {
			years[y-1], years[y], years[y+1] = true, true, true
		}
	}
	ordered := make([]int, 0, len(years))
	for year := range years {
		ordered = append(ordered, year)
	}
	sort.Ints(ordered)
	for _, year := range ordered {
		if year <= 0 {
			continue
		}
		others, err := s.Receipts.List(ctx, year, "")
		if err != nil {
			return "", err
		}
		for i := range others {
			o := &others[i]
			if !other(o) || (!issuers[normalizeName(o.IssuerName)] && !ids[o.EInvoiceIssuer]) {
				continue
			}
			sameNumber := strings.TrimSpace(number) != "" &&
				(strings.EqualFold(strings.TrimSpace(o.Subject), strings.TrimSpace(number)) ||
					strings.EqualFold(strings.TrimSpace(o.EInvoiceNumber), strings.TrimSpace(number)))
			sameAmount := gross != 0 && o.GrossAmount == gross && o.DocumentDate == date
			if sameNumber || sameAmount {
				return o.ReceiptNumber, nil
			}
		}
	}
	return "", nil
}

// bookedTo meldet, ob ein Beleg schon auf diesen Kontakt gebucht ist.
func bookedTo(ctx context.Context, s *core.Services, r *domain.Receipt, contactID uint) bool {
	if r.JournalEntryID == nil {
		return false
	}
	entry, err := s.Repos.Journal.FindByID(ctx, *r.JournalEntryID)
	return err == nil && entry.ContactID != nil && *entry.ContactID == contactID
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
