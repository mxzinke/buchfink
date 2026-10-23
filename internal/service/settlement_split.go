package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// BankSplitLine ist ein Teil eines Bankumsatzes. Genau eine Art je Zeile:
//
//   - OpenItemEntryID gleicht einen offenen Posten aus (wie Settle, mit
//     Differenz),
//   - LoanID bucht Auszahlung oder Tilgung eines Gesellschafterdarlehens,
//   - Account bucht gegen ein Konto, optional mit Kontakt.
//
// Amount (Darlehen, Konto) zählt in Richtung des Umsatzes: positiv steht die
// Zeile auf der Gegenseite der Bank, negativ auf derselben Seite. So wird ein
// Abzug geschrieben, etwa die Kapitalertragsteuer einer Zinsgutschrift.
type BankSplitLine struct {
	OpenItemEntryID  uint                  `json:"openItemEntryId,omitempty"`
	SettledAmount    domain.Cents          `json:"settledAmount,omitempty"`
	DifferenceKind   domain.DifferenceKind `json:"differenceKind,omitempty"`
	DifferenceAmount domain.Cents          `json:"differenceAmount,omitempty"`

	LoanID    uint         `json:"loanId,omitempty"`
	Account   string       `json:"account,omitempty"`
	ContactID uint         `json:"contactId,omitempty"`
	Amount    domain.Cents `json:"amount,omitempty"`
	Text      string       `json:"text,omitempty"`
}

// BankSplitRequest bucht einen Bankumsatz auf mehrere Gegenkonten.
type BankSplitRequest struct {
	BankTxID    uint            `json:"bankTxId"`
	Description string          `json:"description,omitempty"`
	Lines       []BankSplitLine `json:"lines"`
}

// LoanLineSource liefert Konto und Gesellschafter eines Darlehens für die
// Aufteilung eines Bankumsatzes.
type LoanLineSource interface {
	LoanLine(ctx context.Context, loanID uint) (account string, contactID uint, label string, err error)
}

// maxSplitLines begrenzt die Aufteilung: mehr Zeilen sind keine Aufteilung
// eines Umsatzes mehr, sondern ein Sammelauszug, der einzeln zu buchen ist.
const maxSplitLines = 20

// Split bucht einen Bankumsatz auf mehrere Gegenkonten in einer
// Buchung: Dividende oder Zinsen brutto mit Kapitalertragsteuer und
// Solidaritätszuschlag, Darlehensrate aus Tilgung und Zins, Steuerlastschrift
// mehrerer Steuerarten, Sammelüberweisung. Die Zeilen müssen den Umsatz auf
// den Cent treffen.
func (s *settlement) Split(ctx context.Context, req BankSplitRequest) (*domain.JournalEntry, error) {
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (*domain.JournalEntry, error) {
		return s.split(ctx, req)
	})
}

func (s *settlement) split(ctx context.Context, req BankSplitRequest) (*domain.JournalEntry, error) {
	if len(req.Lines) == 0 {
		return nil, fmt.Errorf("die Aufteilung hat keine Zeilen")
	}
	if len(req.Lines) > maxSplitLines {
		return nil, fmt.Errorf("die Aufteilung hat %d Zeilen, höchstens %d sind möglich", len(req.Lines), maxSplitLines)
	}
	tx, err := s.openBankTx(ctx, req.BankTxID)
	if err != nil {
		return nil, err
	}
	if tx.MatchStatus == domain.MatchStatusIgnored {
		return nil, fmt.Errorf("Bankumsatz %d ist als nicht zu buchen markiert", tx.ID)
	}
	if tx.Amount == 0 {
		return nil, fmt.Errorf("Bankumsatz %d hat den Betrag null", tx.ID)
	}
	incoming := tx.Amount > 0
	bankSide, counterSide := domain.SideDebit, domain.SideCredit
	if !incoming {
		bankSide, counterSide = domain.SideCredit, domain.SideDebit
	}

	var allocations []AllocationRequest
	var accountLines []domain.JournalLine
	var accountCash domain.Cents
	contacts := map[uint]bool{}
	for i, line := range req.Lines {
		switch {
		case line.OpenItemEntryID != 0:
			if line.LoanID != 0 || line.Account != "" || line.Amount != 0 {
				return nil, fmt.Errorf("Zeile %d gleicht einen offenen Posten aus und nennt zugleich Konto, "+
					"Darlehen oder Betrag; der Betrag eines Postens steht in settledAmount", i+1)
			}
			allocations = append(allocations, AllocationRequest{
				OpenItemEntryID: line.OpenItemEntryID, SettledAmount: line.SettledAmount,
				DifferenceKind: line.DifferenceKind, DifferenceAmount: line.DifferenceAmount,
			})
		default:
			jl, err := s.splitAccountLine(ctx, tx, line, counterSide)
			if err != nil {
				return nil, fmt.Errorf("Zeile %d: %w", i+1, err)
			}
			accountCash += line.Amount
			accountLines = append(accountLines, jl)
			if jl.ContactID != nil {
				contacts[*jl.ContactID] = true
			}
		}
	}

	parts := &allocationParts{}
	if len(allocations) > 0 {
		id := tx.ID
		parts, err = s.allocate(ctx, allocations, &id, counterSide)
		if err != nil {
			return nil, err
		}
		if parts.cash <= 0 {
			rule := "ein Zahlungsausgang gleicht eine Verbindlichkeit aus"
			if incoming {
				rule = "ein Zahlungseingang gleicht eine Forderung oder die Gutschrift eines Lieferanten aus"
			}
			return nil, fmt.Errorf("die Posten passen nicht zur Richtung dieses Umsatzes: %s; eine Gutschrift "+
				"lässt sich nur mit Posten der Gegenrichtung verrechnen", rule)
		}
		for _, a := range parts.allocations {
			contacts[a.ContactID] = true
		}
	}

	if total := parts.cash + accountCash; total != tx.Amount.Abs() {
		return nil, fmt.Errorf(
			"die Aufteilung ergibt %s €, der Bankumsatz lautet über %s € (Differenz %s €)",
			total, tx.Amount.Abs(), tx.Amount.Abs()-total)
	}

	lines := append(parts.lines, accountLines...)
	lines = append(lines, domain.JournalLine{Side: bankSide, Account: tx.LedgerAccount, Amount: tx.Amount.Abs()})

	description := strings.TrimSpace(req.Description)
	if description == "" {
		description = fmt.Sprintf("Aufteilung %s – %s", tx.CounterpartyName, tx.RemittanceInfo)
	}
	entry := &domain.JournalEntry{
		BookingDate:        tx.BookingDate,
		DocumentDate:       tx.BookingDate,
		ServiceDateFrom:    tx.BookingDate,
		ServiceDateTo:      tx.BookingDate,
		ValueDate:          tx.ValueDate,
		Description:        description,
		Source:             domain.EntrySourcePayment,
		BankTxID:           &tx.ID,
		Currency:           tx.Currency,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
	}
	// Der Kontakt am Kopf nur, wenn die Aufteilung genau einen hat.
	if len(contacts) == 1 {
		for id := range contacts {
			cid := id
			entry.ContactID = &cid
		}
	}
	return s.postSettlement(ctx, entry, parts)
}

// splitAccountLine baut die Zeile eines Gegenkontos oder Darlehens.
func (s *settlement) splitAccountLine(
	ctx context.Context, tx *domain.BankTransaction, line BankSplitLine, counterSide domain.Side,
) (domain.JournalLine, error) {
	if line.Amount == 0 {
		return domain.JournalLine{}, fmt.Errorf("der Betrag fehlt")
	}
	account, text := strings.TrimSpace(line.Account), strings.TrimSpace(line.Text)
	contactID := line.ContactID
	if line.LoanID != 0 {
		if s.loans == nil {
			return domain.JournalLine{}, fmt.Errorf("die Gesellschafterdarlehen sind nicht eingerichtet")
		}
		if account != "" || contactID != 0 {
			return domain.JournalLine{}, fmt.Errorf("eine Darlehenszeile nimmt Konto und Gesellschafter aus dem Vertrag")
		}
		loanAccount, loanContact, label, err := s.loans.LoanLine(ctx, line.LoanID)
		if err != nil {
			return domain.JournalLine{}, err
		}
		account, contactID = loanAccount, loanContact
		if text == "" {
			text = label
		}
	}
	if account == "" {
		return domain.JournalLine{}, fmt.Errorf("es fehlt das Gegenkonto")
	}
	if err := s.checkSplitAccount(ctx, tx, account); err != nil {
		return domain.JournalLine{}, err
	}
	if contactID == 0 && isShareholderAccount(account) {
		return domain.JournalLine{}, fmt.Errorf(
			"Konto %s führt Beträge gegenüber Gesellschaftern; die Zeile braucht den Gesellschafter als Kontakt", account)
	}
	side, amount := counterSide, line.Amount
	if amount < 0 {
		side, amount = counterSide.Opposite(), -amount
	}
	jl := domain.JournalLine{Side: side, Account: account, Amount: amount, Text: text}
	if contactID != 0 {
		if _, err := s.contactRepo.FindByID(ctx, contactID); err != nil {
			return domain.JournalLine{}, fmt.Errorf("Kontakt %d wurde nicht gefunden: %w", contactID, err)
		}
		cid := contactID
		jl.ContactID = &cid
	}
	return jl, nil
}

// checkSplitAccount hält die Konten fern, die über einen anderen Weg gehören.
// Umsatzsteuer entsteht über Beleg oder Rechnung, nicht über den Kontoauszug;
// ein offener Posten wird über seine Zeile ausgeglichen.
func (s *settlement) checkSplitAccount(ctx context.Context, tx *domain.BankTransaction, account string) error {
	if account == tx.LedgerAccount {
		return fmt.Errorf("Konto %s ist das Bankkonto des Umsatzes selbst", account)
	}
	if domain.IsLedgerAccount(account) {
		return fmt.Errorf(
			"Konto %s ist ein Personenkonto; ein offener Posten wird mit seiner Buchung ausgeglichen, nicht über das Konto",
			account)
	}
	chart, err := s.journalSvc.Chart(ctx)
	if err != nil {
		return err
	}
	if acc, ok := chart.Lookup(account); ok && computesTax(acc.Hauptfunktion) {
		return fmt.Errorf(
			"Konto %s (%s) errechnet Umsatz- oder Vorsteuer; dieser Vorgang gehört über den Beleg bzw. die "+
				"Rechnung gebucht und die Zahlung danach dem offenen Posten zugeordnet", account, acc.Name)
	}
	return nil
}

// computesTax meldet die Automatikkonten des SKR04 (Hauptfunktion AM, AV,
// S/AM, S/AV): auf ihnen errechnet die Buchung Umsatz- oder Vorsteuer.
func computesTax(hauptfunktion string) bool {
	return strings.Contains(hauptfunktion, "AM") || strings.Contains(hauptfunktion, "AV")
}

// CreditCardSettlementRequest gleicht die Monatsabrechnung der Firmenkreditkarte
// mit ihrer Abbuchung vom Bankkonto aus.
type CreditCardSettlementRequest struct {
	BankTxID uint `json:"bankTxId"`
	// StatementDate ist der Abrechnungsstichtag der Kartenabrechnung.
	StatementDate string `json:"statementDate"`
	// Differences erklären, warum die Abbuchung vom Saldo von 3610 abweicht.
	Differences []CardDifference `json:"differences,omitempty"`
}

// CardDifference ist eine Abweichung zwischen dem Saldo von 3610 und der
// Abbuchung. Amount ist positiv, wenn weniger abgebucht wurde als auf 3610
// steht (Ertrag), negativ, wenn mehr (Aufwand).
type CardDifference struct {
	Kind   domain.DifferenceKind `json:"kind"`
	Amount domain.Cents          `json:"amount"`
}

// cardDifferenceLine bucht eine Abweichung der Kartenabrechnung: eine
// Kursdifferenz auf die Konten der Währungsumrechnung (§ 277 Abs. 5 Satz 2
// HGB), eine Rundung auf die sonstigen Posten, eine Gebühr als Nebenkosten des
// Geldverkehrs.
func cardDifferenceLine(d CardDifference) (domain.JournalLine, error) {
	if d.Amount == 0 {
		return domain.JournalLine{}, fmt.Errorf("eine Differenz ohne Betrag")
	}
	gain := d.Amount > 0
	line := domain.JournalLine{Side: domain.SideDebit, Amount: d.Amount.Abs()}
	if gain {
		line.Side = domain.SideCredit
	}
	switch d.Kind {
	case domain.DifferenceCurrency:
		line.Account, line.Text = accounting.CurrencyLossAccount, "Kursdifferenz der Kartenabrechnung"
		if gain {
			line.Account = accounting.CurrencyGainAccount
		}
	case domain.DifferenceRounding:
		line.Account, line.Text = "6300", "Rundungsdifferenz der Kartenabrechnung"
		if gain {
			line.Account = "4830"
		}
	case domain.DifferenceBankFee:
		if gain {
			return domain.JournalLine{}, fmt.Errorf("eine Gebühr erhöht die Abbuchung; ihr Betrag ist negativ")
		}
		line.Account, line.Text = domain.AccountNebenkostenGeld, "Gebühr der Kartenabrechnung"
	default:
		return domain.JournalLine{}, fmt.Errorf("Differenzart %q gibt es bei der Kartenabrechnung nicht "+
			"(waehrung, rundung oder gebuehr)", d.Kind)
	}
	return line, nil
}

// creditCardBalance ist der offene Saldo der Kreditkartenabrechnung (3610) an
// einem Tag, als Verbindlichkeit positiv.
func (s *settlement) creditCardBalance(ctx context.Context, date string) (domain.Cents, error) {
	if err := domain.CheckDate("Stichtag", date); err != nil {
		return 0, err
	}
	turnovers, err := s.journalRepo.AccountTurnoversUntil(ctx, s.journalSvc.FiscalYearOf(ctx, date), date)
	if err != nil {
		return 0, fmt.Errorf("der Saldo der Kreditkartenabrechnung ließ sich nicht lesen: %w", err)
	}
	t := turnovers[domain.AccountKreditkartenabrechnung]
	return t.Credit - t.Debit, nil
}

// settleCreditCard bucht die Abbuchung der Kartenabrechnung gegen 3610. Sie
// muss dem Saldo von 3610 zum Abrechnungsstichtag entsprechen; die angegebenen
// Differenzen erklären eine Abweichung, etwa den Unterschied zwischen dem Kurs
// des Belegs und dem der Karte.
func (s *settlement) settleCreditCard(ctx context.Context, req CreditCardSettlementRequest) (*domain.JournalEntry, error) {
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (*domain.JournalEntry, error) {
		tx, err := s.openBankTx(ctx, req.BankTxID)
		if err != nil {
			return nil, err
		}
		if tx.Amount >= 0 {
			return nil, fmt.Errorf("Bankumsatz %d ist ein Zahlungseingang; die Kartenabrechnung wird abgebucht", tx.ID)
		}
		if req.StatementDate == "" || req.StatementDate > tx.BookingDate {
			return nil, fmt.Errorf("der Abrechnungsstichtag muss angegeben sein und darf nicht nach der Abbuchung "+
				"am %s liegen", domain.GermanDate(tx.BookingDate))
		}
		balance, err := s.creditCardBalance(ctx, req.StatementDate)
		if err != nil {
			return nil, err
		}
		paid := tx.Amount.Abs()
		var explained domain.Cents
		differenceLines := make([]domain.JournalLine, 0, len(req.Differences))
		for i, d := range req.Differences {
			line, err := cardDifferenceLine(d)
			if err != nil {
				return nil, fmt.Errorf("Differenz %d: %w", i+1, err)
			}
			explained += d.Amount
			differenceLines = append(differenceLines, line)
		}
		if balance-paid != explained {
			return nil, fmt.Errorf(
				"die Kreditkartenabrechnung (3610) steht zum %s mit %s € offen, abgebucht wurden %s € "+
					"(Differenz %s €, erklärt %s €). Fehlende Kartenbelege zuerst buchen; Kurs- und "+
					"Rundungsdifferenzen oder Gebühren als Differenzen der Kartenabrechnung angeben",
				domain.GermanDate(req.StatementDate), balance, paid, balance-paid, explained)
		}
		entry := &domain.JournalEntry{
			BookingDate:        tx.BookingDate,
			DocumentDate:       tx.BookingDate,
			ServiceDateFrom:    tx.BookingDate,
			ServiceDateTo:      tx.BookingDate,
			ValueDate:          tx.ValueDate,
			Description:        "Kreditkartenabrechnung zum " + domain.GermanDate(req.StatementDate),
			Source:             domain.EntrySourcePayment,
			BankTxID:           &tx.ID,
			PostingRuleVersion: accounting.PostingRuleVersion,
			Lines: append([]domain.JournalLine{
				{Side: domain.SideDebit, Account: domain.AccountKreditkartenabrechnung, Amount: balance},
				{Side: domain.SideCredit, Account: tx.LedgerAccount, Amount: paid},
			}, differenceLines...),
		}
		return s.postSettlement(ctx, entry, &allocationParts{})
	})
}

// creditCardWindowDays ist der Zeitraum vor der Abbuchung, in dem der
// Abrechnungsstichtag gesucht wird.
const creditCardWindowDays = 45

// creditCardStatementDate sucht den Abrechnungsstichtag zu einer Abbuchung: den
// spätesten Tag vor ihr, an dem 3610 genau mit dem abgebuchten Betrag offen
// stand. Leer, wenn es keinen gibt.
func (s *settlement) creditCardStatementDate(ctx context.Context, debit domain.Cents, debitDate string) (string, error) {
	if debit <= 0 {
		return "", nil
	}
	from := addDays(debitDate, -creditCardWindowDays)
	entries, err := s.journalRepo.FindByAccountRange(ctx, domain.AccountKreditkartenabrechnung, from, debitDate)
	if err != nil {
		return "", err
	}
	dates := map[string]bool{}
	for _, e := range entries {
		dates[e.BookingDate] = true
	}
	candidates := make([]string, 0, len(dates))
	for d := range dates {
		candidates = append(candidates, d)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(candidates)))
	for _, d := range candidates {
		balance, err := s.creditCardBalance(ctx, d)
		if err != nil {
			return "", err
		}
		if balance == debit {
			return d, nil
		}
	}
	return "", nil
}
