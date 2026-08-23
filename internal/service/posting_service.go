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
	ContactID       uint   `json:"contactId"`
	BookingDate     string `json:"bookingDate"`
	DocumentDate    string `json:"documentDate"`
	ServiceDateFrom string `json:"serviceDateFrom"`
	ServiceDateTo   string `json:"serviceDateTo"`

	// ReceiptID names the Beleg this booking belongs to. It replaces the former
	// pair of document hash and path: a Beleg is several files, and the
	// Belegnummer comes from the Beleg rather than being typed in freehand.
	ReceiptID uint `json:"receiptId"`

	Description    string              `json:"description"`
	TaxTreatment   domain.TaxTreatment `json:"taxTreatment"`
	Positions      []ReceiptPosition   `json:"positions"`
	Settlement     SettlementKind      `json:"settlement"`
	PaymentAccount string              `json:"paymentAccount,omitempty"`
	Currency       string              `json:"currency,omitempty"`

	// Entertainment carries the record § 4 Abs. 5 Satz 1 Nr. 2 EStG requires
	// whenever entertainment expenses are booked. Without it the deduction is
	// lost even for the deductible 70 %, so the booking is refused rather than
	// written incomplete.
	Entertainment *domain.EntertainmentDetail `json:"entertainment,omitempty"`
}

// PostingPreview is what a booking would look like, computed without writing it.
//
// It exists so the user interface never re-implements the tax rules. A second
// computation in the frontend is a second truth that drifts the moment a
// Steuerfall is added — and the one in the frontend is the one nobody tests.
type PostingPreview struct {
	Lines []domain.JournalLine `json:"lines"`
	// Net is the sum of the expense or revenue lines, Gross what is actually
	// paid or received, and Tax the difference between them. Under Reverse
	// Charge the two coincide and Tax is zero, which is exactly right: the
	// supplier is paid the net amount. The tax that arises there is visible in
	// Lines, where the two legs cancel out.
	Net      domain.Cents `json:"net"`
	Tax      domain.Cents `json:"tax"`
	Gross    domain.Cents `json:"gross"`
	Balanced bool         `json:"balanced"`

	// Warnings are notes the user should see before booking. They never block.
	// They are computed on demand rather than stored: a conserved legal
	// assessment goes stale, and every input it depends on is still there.
	Warnings []PostingWarning `json:"warnings,omitempty"`
}

// PostingService turns business documents into journal entries using the
// deterministic Gruppe → Konten mapping.
type PostingService struct {
	journalSvc  *JournalService
	contactRepo domain.ContactRepository
	taxResolver domain.TaxResolver
	receiptSvc  *ReceiptService
}

// SetReceiptService wires in the Beleg service. Without it a booking cannot
// reference a Beleg, which is what the manual journal entry path relies on.
func (s *PostingService) SetReceiptService(receiptSvc *ReceiptService) { s.receiptSvc = receiptSvc }

// NewPostingService creates the posting service.
func NewPostingService(journalSvc *JournalService, contactRepo domain.ContactRepository) *PostingService {
	return &PostingService{
		journalSvc:  journalSvc,
		contactRepo: contactRepo,
		taxResolver: journalSvc.TaxResolver(),
	}
}

// PostIncomingReceipt books an Eingangsbeleg and seals its Beleg.
//
// The order is deliberate: the Beleg is checked for bookability first, then the
// journal writes, then the Beleg is sealed. Sealing belongs *behind* the journal
// write — if the booking fails the Beleg has to stay open so it can be corrected
// and booked again, whereas the other order would leave an unchangeable Beleg
// with no booking.
func (s *PostingService) PostIncomingReceipt(ctx context.Context, req ReceiptRequest) (*domain.JournalEntry, error) {
	lines, contact, receipt, err := s.buildIncomingLines(ctx, req)
	if err != nil {
		return nil, err
	}

	description := req.Description
	if description == "" {
		description = fmt.Sprintf("Eingangsbeleg %s, %s", receipt.ReceiptNumber, contact.Name)
	}

	entry := &domain.JournalEntry{
		BookingDate:        req.BookingDate,
		DocumentDate:       req.DocumentDate,
		ServiceDateFrom:    req.ServiceDateFrom,
		ServiceDateTo:      req.ServiceDateTo,
		Description:        description,
		Source:             domain.EntrySourceReceipt,
		DocumentNumber:     receipt.ReceiptNumber,
		TaxTreatment:       req.TaxTreatment,
		ReceiptID:          &receipt.ID,
		ReceiptHash:        receipt.ReceiptHash,
		ContactID:          &contact.ID,
		Currency:           req.Currency,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
		Entertainment:      req.Entertainment,
	}

	created, err := s.journalSvc.Post(ctx, entry)
	if err != nil {
		return nil, err
	}

	if err := s.receiptSvc.Seal(ctx, receipt.ID, created.ID); err != nil {
		// The booking stands; only the seal is missing. Saying so beats
		// pretending the booking failed, and ReceiptService repairs the state
		// when the Beleg is read again.
		return created, fmt.Errorf(
			"die Buchung %s wurde geschrieben, der Beleg %s konnte aber nicht versiegelt werden: %w",
			created.EntryNumber, receipt.ReceiptNumber, err)
	}
	return created, nil
}

// PreviewIncomingReceipt computes the booking of an Eingangsbeleg without
// writing it.
func (s *PostingService) PreviewIncomingReceipt(ctx context.Context, req ReceiptRequest) (*PostingPreview, error) {
	lines, contact, receipt, err := s.buildIncomingLines(ctx, req)
	if err != nil {
		return nil, err
	}
	preview := s.preview(ctx, lines)
	if notice := eInvoiceNotice(contact, receipt, req.TaxTreatment, req.DocumentDate, preview.Gross); notice != nil {
		preview.Warnings = append(preview.Warnings, *notice)
	}
	return preview, nil
}

// buildIncomingLines produces the journal lines of an Eingangsbeleg. It is the
// single implementation both the booking and its preview run through, so the two
// cannot disagree.
func (s *PostingService) buildIncomingLines(ctx context.Context, req ReceiptRequest) ([]domain.JournalLine, *domain.Contact, *domain.Receipt, error) {
	if len(req.Positions) == 0 {
		return nil, nil, nil, fmt.Errorf("der Beleg hat keine Positionen")
	}
	// Kein stiller Vorgabewert: ein steuerfreier, ein nullbesteuerter und ein
	// dem Reverse-Charge unterliegender Einkauf sehen im Betrag gleich aus und
	// werden verschieden gebucht. Wo der Steuerfall fehlt — etwa weil der
	// Rechnungsdatensatz Kategorien mischt —, ist er zu wählen, nicht zu raten.
	if req.TaxTreatment == "" {
		return nil, nil, nil, fmt.Errorf(
			"der Steuerfall fehlt. Er ist anzugeben und lässt sich nicht aus den Beträgen erschließen")
	}

	contact, err := s.contactRepo.FindByID(ctx, req.ContactID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Lieferant konnte nicht geladen werden: %w", err)
	}
	if contact.Type != domain.ContactTypeVendor {
		return nil, nil, nil, fmt.Errorf("%s ist als Kunde angelegt und kann keinen Eingangsbeleg stellen", contact.Name)
	}
	if err := validateIncomingTreatment(req.TaxTreatment, contact); err != nil {
		return nil, nil, nil, err
	}

	receipt, err := s.loadBookableReceipt(ctx, req.ReceiptID)
	if err != nil {
		return nil, nil, nil, err
	}

	var lines []domain.JournalLine

	// 1. Aufwands- bzw. Anschaffungszeilen aus den fachlichen Gruppen.
	netByRate := map[domain.TaxRate]domain.Cents{}
	needsEntertainmentRecord := false
	for i, p := range req.Positions {
		if p.Net <= 0 {
			return nil, nil, nil, fmt.Errorf("Position %d: der Nettobetrag muss größer als null sein", i+1)
		}
		positionLines, quota, err := s.expenseLines(p, req.TaxTreatment, req.DocumentDate)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("Position %d: %w", i+1, err)
		}
		lines = append(lines, positionLines...)
		if quota == accounting.QuotaEntertainment {
			needsEntertainmentRecord = true
		}
		// Die Bemessungsgrundlage bleibt der volle Nettobetrag, auch wo der
		// Aufwand geteilt wird: § 15 Abs. 1a Satz 2 UStG nimmt Bewirtungs-
		// aufwendungen vom Vorsteuerausschluss ausdrücklich aus.
		netByRate[p.TaxRate] += p.Net
	}

	if needsEntertainmentRecord {
		if req.Entertainment == nil {
			return nil, nil, nil, fmt.Errorf(
				"zu einer Bewirtung gehören Ort, Tag, Teilnehmer und Anlass (§ 4 Abs. 5 Satz 1 Nr. 2 EStG). Ohne diese Aufzeichnung ist der Abzug auch für die abziehbaren 70 %% verloren")
		}
		if err := req.Entertainment.Validate(); err != nil {
			return nil, nil, nil, err
		}
	}

	// 2. Steuerzeilen, einmal je Steuersatzgruppe gerundet.
	taxLines, err := s.taxLines(domain.DirectionIncoming, req.TaxTreatment, netByRate)
	if err != nil {
		return nil, nil, nil, err
	}
	lines = append(lines, taxLines...)

	// 3. Gegenzeile: was tatsächlich an den Lieferanten zu zahlen ist.
	settlementLine, err := s.settlementLine(lines, req.Settlement, req.PaymentAccount, contact)
	if err != nil {
		return nil, nil, nil, err
	}
	lines = append(lines, settlementLine)

	return lines, contact, receipt, nil
}

// loadBookableReceipt fetches the Beleg and runs the bookability check — the
// second of the two checks, the one filing must not impose.
//
// A Beleg is required, not optional. "Keine Buchung ohne Beleg" is the rule the
// whole flow is built on; where no external document exists the user files an
// Eigenbeleg. Leaving the reference optional would reintroduce the freehand
// Belegfeld this change set removed.
func (s *PostingService) loadBookableReceipt(ctx context.Context, receiptID uint) (*domain.Receipt, error) {
	if receiptID == 0 {
		return nil, fmt.Errorf("zu jedem Eingangsbeleg gehört ein abgelegter Beleg. Liegt kein Dokument des Lieferanten vor, ist ein Eigenbeleg abzulegen")
	}
	if s.receiptSvc == nil {
		return nil, fmt.Errorf("die Belegverwaltung ist nicht eingerichtet")
	}
	receipt, err := s.receiptSvc.Get(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	if receipt.Direction != domain.DirectionIncoming {
		return nil, fmt.Errorf("Beleg %s ist ein Ausgangsbeleg und passt nicht zu einem Eingangsbeleg", receipt.ReceiptNumber)
	}
	if receipt.Status == domain.ReceiptStatusSealed {
		return nil, fmt.Errorf("Beleg %s ist bereits gebucht", receipt.ReceiptNumber)
	}
	if err := receipt.ValidateBookable(); err != nil {
		return nil, err
	}
	return receipt, nil
}

// preview turns a set of lines into the numbers a user checks against the paper
// invoice, with account names filled in for display.
func (s *PostingService) preview(ctx context.Context, lines []domain.JournalLine) *PostingPreview {
	out := make([]domain.JournalLine, len(lines))
	copy(out, lines)

	if chart, err := s.journalSvc.Chart(ctx); err == nil {
		for i := range out {
			out[i].AccountName = chart.Name(out[i].Account)
		}
	}
	for i := range out {
		out[i].Position = i + 1
	}

	var debit, credit, net, gross domain.Cents
	for i := range out {
		l := &out[i]
		if l.Side == domain.SideDebit {
			debit += l.Amount
		} else {
			credit += l.Amount
		}
	}
	// The settlement line is the last one and carries what is actually paid or
	// received; everything before it that is not a tax line is the net amount.
	if len(out) > 0 {
		gross = out[len(out)-1].Amount
		for i := 0; i < len(out)-1; i++ {
			if out[i].TaxKey == "" {
				net += out[i].Amount
			}
		}
	}

	return &PostingPreview{
		Lines:    out,
		Net:      net,
		Tax:      gross - net,
		Gross:    gross,
		Balanced: debit == credit,
	}
}

// PostOutgoingInvoice books an Ausgangsrechnung as a receivable.
func (s *PostingService) PostOutgoingInvoice(ctx context.Context, inv *domain.Invoice, contact *domain.Contact) (*domain.JournalEntry, error) {
	lines, err := s.buildOutgoingLines(inv, contact)
	if err != nil {
		return nil, err
	}

	entry := &domain.JournalEntry{
		FiscalYear:         inv.FiscalYear,
		BookingDate:        inv.Date,
		DocumentDate:       inv.Date,
		ServiceDateFrom:    inv.ServiceDateFrom,
		ServiceDateTo:      inv.ServiceDateTo,
		Description:        fmt.Sprintf("Rechnung %s an %s", inv.InvoiceNumber, contact.Name),
		Source:             domain.EntrySourceInvoice,
		DocumentNumber:     inv.InvoiceNumber,
		TaxTreatment:       inv.TaxTreatment,
		ContactID:          &contact.ID,
		Currency:           inv.Currency,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
	}

	return s.journalSvc.Post(ctx, entry)
}

// PreviewOutgoingInvoice computes the booking of an Ausgangsrechnung without
// writing it. The invoice form shows this instead of doing the arithmetic again.
func (s *PostingService) PreviewOutgoingInvoice(ctx context.Context, inv *domain.Invoice, contact *domain.Contact) (*PostingPreview, error) {
	lines, err := s.buildOutgoingLines(inv, contact)
	if err != nil {
		return nil, err
	}
	return s.preview(ctx, lines), nil
}

// buildOutgoingLines produces the journal lines of an Ausgangsrechnung, shared by
// the booking and its preview.
func (s *PostingService) buildOutgoingLines(inv *domain.Invoice, contact *domain.Contact) ([]domain.JournalLine, error) {
	group, err := accounting.LookupPostingGroup("erloese")
	if err != nil {
		return nil, err
	}
	if len(inv.Items) == 0 {
		return nil, fmt.Errorf("die Rechnung hat keine Positionen")
	}
	treatment := inv.TaxTreatment
	if treatment == "" {
		treatment = domain.TaxTreatmentDomestic
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
			Account: g.ResolveAccount(treatment, item.TaxRate),
			Amount:  item.TotalNet(),
			Text:    item.Description,
		})
		netByRate[item.TaxRate] += item.TotalNet()
	}

	taxLines, err := s.taxLines(domain.DirectionOutgoing, treatment, netByRate)
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
	return append(lines, settlementLine), nil
}

// expenseLines turns one position into its expense lines.
//
// Most positions produce exactly one. An expense that is only partly deductible
// produces two: the deductible share on the group's account and the rest on the
// non-deductible one. Booking both to one account would leave the Steuerbilanz
// wrong, and splitting it later is not possible — the information which part was
// which is gone by then.
func (s *PostingService) expenseLines(p ReceiptPosition, treatment domain.TaxTreatment, documentDate string) ([]domain.JournalLine, accounting.DeductibleQuota, error) {
	if p.Account != "" {
		// A directly chosen account still has to pass the journal's checks; this
		// is the escape hatch for cases the group catalog does not cover.
		return []domain.JournalLine{{
			Side: domain.SideDebit, Account: p.Account, Amount: p.Net, Text: p.Text,
		}}, "", nil
	}
	if p.PostingGroup == "" {
		return nil, "", fmt.Errorf("weder eine Buchungsgruppe noch ein Konto angegeben")
	}
	group, err := accounting.LookupPostingGroup(p.PostingGroup)
	if err != nil {
		return nil, "", err
	}
	if group.Direction != domain.DirectionIncoming {
		return nil, "", fmt.Errorf("%q passt nicht zur Belegrichtung", group.Label)
	}

	account := group.ResolveAccount(treatment, p.TaxRate)
	if group.NonDeductibleAccount == "" {
		return []domain.JournalLine{{
			Side: domain.SideDebit, Account: account, Amount: p.Net, Text: p.Text,
		}}, "", nil
	}

	params, err := accounting.TaxParametersFor(documentDate)
	if err != nil {
		return nil, "", err
	}
	permille := deductiblePermille(group.DeductibleQuota, params)
	if permille <= 0 || permille >= 1000 {
		return nil, "", fmt.Errorf("für %q ist kein abziehbarer Anteil hinterlegt", group.Label)
	}

	deductible := domain.MulRound(p.Net, permille, 1000)
	// Der Rest ergibt sich als Differenz und wird nicht ein zweites Mal
	// gerundet — sonst summierten sich die beiden Zeilen an einem Cent vorbei.
	nonDeductible := p.Net - deductible

	lines := []domain.JournalLine{
		{Side: domain.SideDebit, Account: account, Amount: deductible, Text: p.Text},
	}
	if nonDeductible > 0 {
		lines = append(lines, domain.JournalLine{
			Side: domain.SideDebit, Account: group.NonDeductibleAccount,
			Amount: nonDeductible, Text: p.Text,
		})
	}
	return lines, group.DeductibleQuota, nil
}

func deductiblePermille(quota accounting.DeductibleQuota, params accounting.TaxParameters) int64 {
	switch quota {
	case accounting.QuotaEntertainment:
		return params.EntertainmentDeductiblePermille
	default:
		return 0
	}
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
			lines = append(lines, taxLegLine(leg))
		}
	}
	return lines, nil
}

// taxLegLine turns a Steuerbein into the journal line that carries it.
//
// The renaming between the two types — Base to TaxBase, Key to TaxKey — is
// written once. A second copy is where a field added to TaxLeg reaches one
// booking path and silently misses the other.
func taxLegLine(leg domain.TaxLeg) domain.JournalLine {
	return domain.JournalLine{
		Side:    leg.Side,
		Account: leg.Account,
		Amount:  leg.Amount,
		TaxKey:  leg.Key,
		TaxBase: leg.Base,
	}
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
