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
	// Beide Abfragen sind bewusst nicht auf das Wirtschaftsjahr begrenzt — die
	// Begründung steht an den Schnittstellen, die sie beantworten.
	entries, err := s.journalRepo.FindOpenItemCandidates(ctx, s.fiscalYear)
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

	var items []domain.OpenItem
	for i := range entries {
		entry := &entries[i]
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

		// Steuersatz und Steuerfall werden hier bestimmt, nicht erst dort, wo ein
		// Skonto sie braucht: so gilt für jeden Leser dieselbe Regel, und ein
		// leerer Steuerfall heißt überall dasselbe — nicht bestimmbar.
		rate, rateKnown := documentTaxRate(entry)
		treatment := entry.TaxTreatment
		switch {
		case !rateKnown:
			// Eine Steuerzeile, die zu keinem bekannten Satz passt. Dann ist auch
			// der Steuerfall nicht bestimmbar.
			treatment = ""
		case treatment == "":
			// Von Hand im Journal erfasste Buchungen tragen keinen Steuerfall.
			// Trägt das Dokument eine Steuerzeile, ist es ein steuerpflichtiger
			// Inlandsumsatz; trägt es keine, gibt es nichts zu berichtigen.
			treatment = domain.TaxTreatmentDomestic
			if rate == domain.TaxRateNone {
				treatment = domain.TaxTreatmentNotTaxable
			}
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
			TaxRate:        rate,
			TaxTreatment:   treatment,
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

		cid := item.ContactID
		// Settling a payable is a debit on the Kreditorenkonto, settling a
		// receivable a credit on the Debitorenkonto — the item knows which.
		lines = append(lines, domain.JournalLine{
			Side:      item.SettleSide(),
			Account:   item.LedgerAccount,
			ContactID: &cid,
			Amount:    alloc.SettledAmount,
			Text:      item.DocumentNumber,
		})
		if contactID == nil {
			contactID = &cid
		}

		diffLines, err := s.differenceLines(alloc, item)
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
	lines = append(lines, domain.JournalLine{
		Side: lines[0].Side.Opposite(), Account: req.PaymentAccount, Amount: totalCash,
	})

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
) ([]domain.JournalLine, error) {
	if alloc.DifferenceKind == domain.DifferenceNone {
		return nil, nil
	}

	// The difference sits on the side opposite the settlement, except for a bank
	// fee, which is an additional expense on the same side as the settlement.
	opposite := item.SettleSide().Opposite()

	switch alloc.DifferenceKind {
	case domain.DifferenceSkonto:
		return s.skontoLines(alloc, item, opposite)

	case domain.DifferenceBankFee:
		return []domain.JournalLine{{
			Side: item.SettleSide(), Account: domain.AccountNebenkostenGeld,
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
	opposite domain.Side,
) ([]domain.JournalLine, error) {
	if item.TaxTreatment == "" {
		return nil, fmt.Errorf(
			"der Steuerfall von %s lässt sich nicht bestimmen; ein Skonto darauf wäre nach § 17 Abs. 1 UStG nicht sauber zu berichtigen",
			item.DocumentNumber)
	}
	direction := item.Direction()

	// Nur beim steuerpflichtigen Inlandsumsatz steckt die Steuer im offenen
	// Betrag. Bei § 13b, beim innergemeinschaftlichen Erwerb und bei jedem
	// steuerfreien Umsatz ist die Rechnung netto ausgestellt — dort ist das
	// ganze Skonto Bemessungsgrundlage, und ihm eine Steuer herauszurechnen,
	// die nie berechnet wurde, verkürzte die Minderung um 19 %. Derselbe Satz
	// entscheidet über das Skontokonto: ohne ausgewiesene Steuer ist es das
	// ohne Steuersatz.
	skontoRate := domain.TaxRateNone
	if item.TaxTreatment == domain.TaxTreatmentDomestic {
		skontoRate = item.TaxRate
	}
	net := skontoRate.NetFromGross(alloc.DifferenceAmount)

	// Der Steuersatz des Belegs bleibt der des Belegs: er bestimmt die Höhe der
	// Berichtigung, auch wo im Skonto selbst keine Steuer steckt.
	legs, err := s.journalSvc.TaxResolver().Resolve(direction, item.TaxTreatment, item.TaxRate, net)
	if err != nil {
		return nil, fmt.Errorf("die Steuerkorrektur des Skontos ließ sich nicht auflösen: %w", err)
	}
	account, err := domain.SkontoAccount(direction, skontoRate)
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
		line := taxLegLine(leg)
		line.Side = leg.Side.Opposite()
		line.TaxKey = "SKONTO_" + leg.Key
		line.Text = "Steuerkorrektur Skonto (§ 17 Abs. 1 UStG)"
		lines = append(lines, line)
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
//
// The second result separates "carried no VAT" from "carried VAT I could not
// read". Both would be TaxRateNone, and TaxRateNone means "no rate applies" —
// letting it also mean "unknown" is how a taxable supply ends up booked as a
// tax-free one.
func documentTaxRate(entry *domain.JournalEntry) (domain.TaxRate, bool) {
	for _, l := range entry.Lines {
		if l.TaxKey == "" || l.TaxBase == 0 {
			continue
		}
		// Recover the rate from base and amount rather than parsing the key.
		switch {
		case domain.TaxRateStandard.Tax(l.TaxBase) == l.Amount:
			return domain.TaxRateStandard, true
		case domain.TaxRateReduced.Tax(l.TaxBase) == l.Amount:
			return domain.TaxRateReduced, true
		}
		return domain.TaxRateNone, false
	}
	return domain.TaxRateNone, true
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
