package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// AllocationRequest assigns part of a payment to one open item.
type AllocationRequest struct {
	OpenItemEntryID uint `json:"openItemEntryId"`
	// SettledAmount is the amount the open item is reduced by. With a Skonto
	// that is the full remaining amount even though less cash moves.
	SettledAmount    domain.Cents          `json:"settledAmount"`
	DifferenceKind   domain.DifferenceKind `json:"differenceKind"`
	DifferenceAmount domain.Cents          `json:"differenceAmount"`
}

// cashAmount is what actually moves on the liquid account for this allocation.
// A Skonto or a rounding difference reduces it, a bank fee increases it.
func (a AllocationRequest) cashAmount() domain.Cents {
	switch a.DifferenceKind {
	case domain.DifferenceSkonto, domain.DifferenceRounding, domain.DifferenceCurrency:
		return a.SettledAmount - a.DifferenceAmount
	case domain.DifferenceBankFee:
		return a.SettledAmount + a.DifferenceAmount
	default:
		return a.SettledAmount
	}
}

// PaymentRequest settles one or more open items from a single payment.
type PaymentRequest struct {
	BankTxID       *uint               `json:"bankTxId,omitempty"`
	PaymentAccount string              `json:"paymentAccount"`
	PaymentDate    string              `json:"paymentDate"`
	ValueDate      string              `json:"valueDate,omitempty"`
	Description    string              `json:"description,omitempty"`
	Allocations    []AllocationRequest `json:"allocations"`
}

// PaymentService settles open items and books the differences.
type PaymentService struct {
	journalSvc     *JournalService
	journalRepo    domain.JournalRepository
	allocationRepo domain.PaymentAllocationRepository
	contactRepo    domain.ContactRepository
	bankRepo       domain.BankRepository
	fiscalYear     int
}

// NewPaymentService creates the payment matching service.
func NewPaymentService(
	journalSvc *JournalService,
	journalRepo domain.JournalRepository,
	allocationRepo domain.PaymentAllocationRepository,
	contactRepo domain.ContactRepository,
	bankRepo domain.BankRepository,
	fiscalYear int,
) *PaymentService {
	return &PaymentService{
		journalSvc:     journalSvc,
		journalRepo:    journalRepo,
		allocationRepo: allocationRepo,
		contactRepo:    contactRepo,
		bankRepo:       bankRepo,
		fiscalYear:     fiscalYear,
	}
}

// SetFiscalYear updates the year the open items are computed for.
func (s *PaymentService) SetFiscalYear(year int) { s.fiscalYear = year }

// OpenItems lists the unsettled receivables and payables.
//
// TODO: stichtagsbezogene OP-Liste — „welche Posten waren am 31.12. offen".
// Diese hier ist die operative Sicht: was ist heute noch offen, und wogegen darf
// gebucht werden. Für den Jahresabschluss braucht es dieselbe Rechnung mit einer
// Datumsgrenze auf Buchungsdatum der Zahlung, nicht mit einer Jahreszahl. Die
// Bilanzposition selbst kommt aus den Salden der Personenkonten und ist davon
// nicht betroffen.
func (s *PaymentService) OpenItems(ctx context.Context) ([]domain.OpenItem, error) {
	// Auch die Vorjahre: eine Forderung aus dem Dezember ist im Januar noch eine
	// Forderung. Wäre die Liste auf das laufende Jahr begrenzt, ließe sich die
	// Rechnung, die den Jahreswechsel überlebt hat, überhaupt nicht mehr
	// ausgleichen — sie stünde in keiner Auswahl.
	entries, err := s.journalRepo.FindThroughFiscalYear(ctx, s.fiscalYear)
	if err != nil {
		return nil, err
	}
	settled, err := s.allocationRepo.SettledByOpenItem(ctx)
	if err != nil {
		return nil, err
	}

	contacts, err := s.contactRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[uint]domain.Contact, len(contacts))
	for _, c := range contacts {
		byID[c.ID] = c
	}

	// A Generalumkehr cancels the open item of the entry it reverses.
	reversed := map[uint]bool{}
	for i := range entries {
		if entries[i].ReversalOfID != nil {
			reversed[*entries[i].ReversalOfID] = true
		}
	}

	var items []domain.OpenItem
	for i := range entries {
		entry := &entries[i]
		if entry.Kind == domain.EntryKindReversal || reversed[entry.ID] {
			continue
		}
		// Payments themselves do not create open items.
		if entry.Source == domain.EntrySourcePayment {
			continue
		}

		line, ok := ledgerLine(entry)
		if !ok || entry.ContactID == nil {
			continue
		}
		contact, ok := byID[*entry.ContactID]
		if !ok {
			continue
		}

		gross := line.Amount
		alreadySettled := settled[entry.ID]
		open := gross - alreadySettled
		if open == 0 {
			continue
		}

		items = append(items, domain.OpenItem{
			EntryID:        entry.ID,
			EntryNumber:    entry.EntryNumber,
			ContactID:      contact.ID,
			ContactName:    contact.Name,
			ContactType:    contact.Type,
			LedgerAccount:  line.Account,
			DocumentNumber: entry.DocumentNumber,
			DocumentDate:   entry.DocumentDate,
			DueDate:        dueDate(entry, contact),
			GrossAmount:    gross,
			SettledAmount:  alreadySettled,
			OpenAmount:     open,
			TaxRate:        documentTaxRate(entry),
			TaxTreatment:   entry.TaxTreatment,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].DueDate != items[j].DueDate {
			return items[i].DueDate < items[j].DueDate
		}
		return items[i].EntryNumber < items[j].EntryNumber
	})
	return items, nil
}

// Settle books a payment against one or more open items.
func (s *PaymentService) Settle(ctx context.Context, req PaymentRequest) (*domain.JournalEntry, error) {
	if len(req.Allocations) == 0 {
		return nil, fmt.Errorf("es wurde kein offener Posten ausgewählt")
	}
	if req.PaymentDate == "" {
		req.PaymentDate = time.Now().Format("2006-01-02")
	}

	// A bank payment takes its account and amount from the statement, so neither
	// can be mistyped.
	var bankTx *domain.BankTransaction
	if req.BankTxID != nil {
		tx, err := s.bankRepo.FindByID(ctx, *req.BankTxID)
		if err != nil {
			return nil, fmt.Errorf("Bankumsatz %d wurde nicht gefunden: %w", *req.BankTxID, err)
		}
		if tx.MatchStatus == domain.MatchStatusMatched {
			return nil, fmt.Errorf("Bankumsatz %d ist bereits zugeordnet", tx.ID)
		}
		bankTx = tx
		req.PaymentAccount = tx.LedgerAccount
		req.PaymentDate = tx.BookingDate
		req.ValueDate = tx.ValueDate
	}
	if req.PaymentAccount == "" {
		return nil, fmt.Errorf("Zahlungsmittel fehlt (Kasse, Bank oder Kreditkarte)")
	}

	openItems, err := s.OpenItems(ctx)
	if err != nil {
		return nil, err
	}
	byEntry := make(map[uint]domain.OpenItem, len(openItems))
	for _, o := range openItems {
		byEntry[o.EntryID] = o
	}

	var lines []domain.JournalLine
	var totalCash domain.Cents
	var contactID *uint
	allocations := make([]domain.PaymentAllocation, 0, len(req.Allocations))

	for i, alloc := range req.Allocations {
		item, ok := byEntry[alloc.OpenItemEntryID]
		if !ok {
			return nil, fmt.Errorf("Zuordnung %d: die Buchung %d hat keinen offenen Posten", i+1, alloc.OpenItemEntryID)
		}
		if alloc.SettledAmount <= 0 {
			return nil, fmt.Errorf("Zuordnung %d: der Ausgleichsbetrag muss größer als null sein", i+1)
		}
		if alloc.SettledAmount > item.OpenAmount {
			return nil, fmt.Errorf(
				"Zuordnung %d: %s ist mit %s € offen, zugeordnet wurden aber %s €",
				i+1, item.DocumentNumber, item.OpenAmount, alloc.SettledAmount)
		}
		if alloc.DifferenceKind == "" {
			alloc.DifferenceKind = domain.DifferenceNone
		}
		if alloc.DifferenceKind != domain.DifferenceNone && alloc.DifferenceAmount <= 0 {
			return nil, fmt.Errorf("Zuordnung %d: für die Differenzart %q fehlt der Betrag", i+1, alloc.DifferenceKind)
		}

		// Settling a payable is a debit on the Kreditorenkonto, settling a
		// receivable a credit on the Debitorenkonto.
		settleSide := domain.SideDebit
		direction := domain.DirectionIncoming
		if item.ContactType == domain.ContactTypeCustomer {
			settleSide = domain.SideCredit
			direction = domain.DirectionOutgoing
		}

		cid := item.ContactID
		lines = append(lines, domain.JournalLine{
			Side:      settleSide,
			Account:   item.LedgerAccount,
			ContactID: &cid,
			Amount:    alloc.SettledAmount,
			Text:      item.DocumentNumber,
		})
		if contactID == nil {
			contactID = &cid
		}

		diffLines, err := s.differenceLines(alloc, item, direction, settleSide)
		if err != nil {
			return nil, fmt.Errorf("Zuordnung %d: %w", i+1, err)
		}
		lines = append(lines, diffLines...)

		cash := alloc.cashAmount()
		if cash < 0 {
			return nil, fmt.Errorf("Zuordnung %d: die Differenz ist größer als der Ausgleichsbetrag", i+1)
		}
		totalCash += cash

		allocations = append(allocations, domain.PaymentAllocation{
			OpenItemEntryID:  item.EntryID,
			BankTxID:         req.BankTxID,
			ContactID:        item.ContactID,
			SettledAmount:    alloc.SettledAmount,
			CashAmount:       cash,
			DifferenceKind:   alloc.DifferenceKind,
			DifferenceAmount: alloc.DifferenceAmount,
		})
	}

	// The cash side is what the statement says. Checking it here is what turns a
	// mis-entered Skonto into an error message instead of a silent wrong booking.
	if bankTx != nil && totalCash != bankTx.Amount.Abs() {
		return nil, fmt.Errorf(
			"die Zuordnung ergibt %s €, der Bankumsatz lautet aber über %s €",
			totalCash, bankTx.Amount.Abs())
	}
	if totalCash == 0 {
		return nil, fmt.Errorf("die Zahlung hat einen Betrag von null")
	}

	// Money leaving is a credit on the liquid account, money arriving a debit.
	cashSide := domain.SideCredit
	if lines[0].Side == domain.SideCredit {
		cashSide = domain.SideDebit
	}
	lines = append(lines, domain.JournalLine{Side: cashSide, Account: req.PaymentAccount, Amount: totalCash})

	description := req.Description
	if description == "" {
		description = paymentDescription(req, byEntry)
	}

	entry := &domain.JournalEntry{
		BookingDate:        req.PaymentDate,
		DocumentDate:       req.PaymentDate,
		ServiceDateFrom:    req.PaymentDate,
		ServiceDateTo:      req.PaymentDate,
		ValueDate:          req.ValueDate,
		Description:        description,
		Source:             domain.EntrySourcePayment,
		ContactID:          contactID,
		BankTxID:           req.BankTxID,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
	}

	created, err := s.journalSvc.Post(ctx, entry)
	if err != nil {
		return nil, err
	}

	for i := range allocations {
		allocations[i].PaymentEntryID = created.ID
	}
	if err := s.allocationRepo.Create(ctx, allocations); err != nil {
		return nil, fmt.Errorf("die Zahlungszuordnung konnte nicht gespeichert werden: %w", err)
	}

	if bankTx != nil {
		if err := s.bankRepo.SetMatchStatus(ctx, bankTx.ID, domain.MatchStatusMatched); err != nil {
			return nil, fmt.Errorf("der Bankumsatz konnte nicht als zugeordnet markiert werden: %w", err)
		}
	}

	return created, nil
}

// differenceLines books the difference between the settled amount and the cash
// that moved.
func (s *PaymentService) differenceLines(
	alloc AllocationRequest,
	item domain.OpenItem,
	direction domain.Direction,
	settleSide domain.Side,
) ([]domain.JournalLine, error) {
	if alloc.DifferenceKind == domain.DifferenceNone {
		return nil, nil
	}

	// The difference sits on the side opposite the settlement, except for a bank
	// fee, which is an additional expense on the same side as the settlement.
	opposite := domain.SideCredit
	if settleSide == domain.SideCredit {
		opposite = domain.SideDebit
	}

	switch alloc.DifferenceKind {
	case domain.DifferenceSkonto:
		return s.skontoLines(alloc, item, direction, opposite)

	case domain.DifferenceBankFee:
		return []domain.JournalLine{{
			Side: settleSide, Account: domain.AccountNebenkostenGeld,
			Amount: alloc.DifferenceAmount, Text: "Bankgebühr",
		}}, nil

	case domain.DifferenceRounding, domain.DifferenceCurrency:
		text := "Rundungsdifferenz"
		if alloc.DifferenceKind == domain.DifferenceCurrency {
			text = "Kursdifferenz"
		}
		// On the credit side the difference is income, on the debit side expense.
		account := "6300" // Sonstige betriebliche Aufwendungen
		if opposite == domain.SideCredit {
			account = "4830" // Sonstige betriebliche Erträge
		}
		return []domain.JournalLine{{
			Side: opposite, Account: account, Amount: alloc.DifferenceAmount, Text: text,
		}}, nil
	}

	return nil, fmt.Errorf("unbekannte Differenzart %q", alloc.DifferenceKind)
}

// skontoLines books a Skonto as what it is in tax law: a change of the taxable
// amount under § 17 Abs. 1 UStG.
//
// Satz 1 makes the supplier correct the tax it owes, Satz 2 makes the recipient
// correct the input tax it deducted, and Satz 8 puts both into the period in
// which the change occurred — which is why this runs with the payment and not
// as a backdated change to the invoice. Satz 5 extends all of it to § 13b and
// to the innergemeinschaftlicher Erwerb, where the recipient owes the tax and
// deducts it again: there both legs have to be corrected, even though they
// cancel out. UStAE 17.1 Abs. 3 says so in as many words — the duty to correct
// exists "auch dann, wenn sich die Berichtigung der Steuer und die Berichtigung
// des Vorsteuerabzugs im Ergebnis ausgleichen". Netting them away would leave
// two lines of the Voranmeldung overstated.
//
// The tax lines come from the Steuerautomatik rather than from a second table of
// account numbers here. That is what keeps a Skonto on a § 13b purchase from
// landing on 1406, and it is why the correction follows the Steuerfall of the
// original document instead of guessing from the rate.
func (s *PaymentService) skontoLines(
	alloc AllocationRequest,
	item domain.OpenItem,
	direction domain.Direction,
	opposite domain.Side,
) ([]domain.JournalLine, error) {
	treatment := item.TaxTreatment
	if treatment == "" {
		// Buchungen, die von Hand im Journal entstanden sind, tragen keinen
		// Steuerfall. Trägt das Dokument eine Steuerzeile, ist es ein
		// steuerpflichtiger Inlandsumsatz; trägt es keine, gibt es nichts zu
		// berichtigen.
		treatment = domain.TaxTreatmentDomestic
		if item.TaxRate == domain.TaxRateNone {
			treatment = domain.TaxTreatmentNotTaxable
		}
	}

	// Nur beim steuerpflichtigen Inlandsumsatz steckt die Steuer im offenen
	// Betrag. Bei § 13b, beim innergemeinschaftlichen Erwerb und bei jedem
	// steuerfreien Umsatz ist die Rechnung netto ausgestellt — dort ist das
	// ganze Skonto Bemessungsgrundlage, und ihm eine Steuer herauszurechnen,
	// die nie berechnet wurde, verkürzte die Minderung um 19 %.
	net := alloc.DifferenceAmount
	if treatment == domain.TaxTreatmentDomestic {
		net = item.TaxRate.NetFromGross(alloc.DifferenceAmount)
	}

	legs, err := s.journalSvc.TaxResolver().Resolve(direction, treatment, item.TaxRate, net)
	if err != nil {
		return nil, fmt.Errorf("die Steuerkorrektur des Skontos ließ sich nicht auflösen: %w", err)
	}

	// Das Aufwands- bzw. Erlöskonto richtet sich danach, ob im Skonto Steuer
	// steckt: ohne ausgewiesene Steuer ist es das Skontokonto ohne Steuersatz.
	accountRate := domain.TaxRateNone
	if treatment == domain.TaxTreatmentDomestic {
		accountRate = item.TaxRate
	}
	account, err := domain.SkontoAccount(direction, accountRate)
	if err != nil {
		return nil, err
	}

	lines := []domain.JournalLine{
		{Side: opposite, Account: account, Amount: net, Text: "Skonto"},
	}
	for _, leg := range legs {
		if leg.Amount == 0 {
			continue
		}
		// Die Minderung steht der ursprünglichen Steuerzeile gegenüber: was die
		// Rechnung im Soll gebucht hat, wird im Haben zurückgenommen.
		side := domain.SideCredit
		if leg.Side == domain.SideCredit {
			side = domain.SideDebit
		}
		lines = append(lines, domain.JournalLine{
			Side: side, Account: leg.Account, Amount: leg.Amount,
			TaxKey: "SKONTO_" + leg.Key, TaxBase: leg.Base,
			Text: "Steuerkorrektur Skonto (§ 17 Abs. 1 UStG)",
		})
	}
	return lines, nil
}

// ledgerLine returns the Personenkonto line of an entry, which is the one that
// carries the open item.
func ledgerLine(entry *domain.JournalEntry) (domain.JournalLine, bool) {
	for _, l := range entry.Lines {
		if domain.IsLedgerAccount(l.Account) {
			return l, true
		}
	}
	return domain.JournalLine{}, false
}

// documentTaxRate derives the VAT rate of a document from its tax lines. It is
// needed to correct the taxable amount when a Skonto is granted later.
func documentTaxRate(entry *domain.JournalEntry) domain.TaxRate {
	for _, l := range entry.Lines {
		if l.TaxKey == "" || l.TaxBase == 0 {
			continue
		}
		// Recover the rate from base and amount rather than parsing the key.
		for _, rate := range []domain.TaxRate{domain.TaxRateStandard, domain.TaxRateReduced} {
			if rate.Tax(l.TaxBase) == l.Amount {
				return rate
			}
		}
	}
	return domain.TaxRateNone
}

func dueDate(entry *domain.JournalEntry, contact domain.Contact) string {
	days := contact.PaymentTermsDays
	if days <= 0 {
		days = 14
	}
	t, err := time.Parse("2006-01-02", entry.DocumentDate)
	if err != nil {
		return entry.DocumentDate
	}
	return t.AddDate(0, 0, days).Format("2006-01-02")
}

func paymentDescription(req PaymentRequest, items map[uint]domain.OpenItem) string {
	if len(req.Allocations) == 1 {
		if item, ok := items[req.Allocations[0].OpenItemEntryID]; ok {
			return fmt.Sprintf("Zahlung %s – %s", item.DocumentNumber, item.ContactName)
		}
	}
	return fmt.Sprintf("Sammelzahlung über %d Belege", len(req.Allocations))
}
