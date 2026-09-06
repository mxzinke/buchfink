package accounting

import (
	"sort"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Filter über Journal und Kontoblatt (PRF-01 K3).
//
// Eine Auswertung, die sich nicht einschränken lässt, beantwortet genau eine
// Frage: „was ist gebucht?". Die Fragen einer Prüfung sind andere — alle
// Buchungen über 10.000 € auf ein Konto, alle Buchungen einer Bearbeiterin,
// alle Buchungen ohne Beleg, alle mit dem Steuerschlüssel VST19. Ohne Filter
// beantwortet man sie, indem man exportiert und in einer Tabellenkalkulation
// weitersucht; dann steht die Antwort außerhalb des Systems, aus dem sie stammt.
//
// Gefiltert wird über Zeilen und nicht über Buchungen, weil die Fragen an
// Zeilen hängen: „Konto 6300" ist eine Eigenschaft einer Zeile. Die Summenzeile
// summiert deshalb Soll und Haben der gefilterten Zeilen — nicht Bruttobeträge
// von Buchungen, die zur Hälfte herausgefallen wären.

// JournalFilter sind die Einschränkungen der Ansicht.
//
// Alle Felder sind freiwillig; ein leerer Filter liefert alles. Die Zeiger bei
// den Beträgen und beim Belegkennzeichen unterscheiden „nicht gesetzt" von
// „null" bzw. „nein" — ein Betragsfilter „von 0 €" ist etwas anderes als kein
// Betragsfilter, und „ohne Beleg" etwas anderes als „egal".
type JournalFilter struct {
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
	// Account ist das Konto der Zeile, CounterAccount ein Konto, das in
	// derselben Buchung auf der anderen Seite steht.
	Account        string `json:"account,omitempty"`
	CounterAccount string `json:"counterAccount,omitempty"`
	// AmountFrom und AmountTo grenzen den Betrag der Zeile ein.
	AmountFrom *domain.Cents `json:"amountFrom,omitempty"`
	AmountTo   *domain.Cents `json:"amountTo,omitempty"`
	TaxKey     string        `json:"taxKey,omitempty"`
	Actor      string        `json:"actor,omitempty"`
	// HasReceipt: true nur mit Beleg, false nur ohne, nil egal.
	HasReceipt *bool `json:"hasReceipt,omitempty"`
	// Text sucht in Buchungstext, Zeilentext, Belegnummer und Buchungsnummer.
	Text string `json:"text,omitempty"`
}

// IsEmpty meldet, ob der Filter nichts einschränkt.
func (f JournalFilter) IsEmpty() bool {
	return f.From == "" && f.To == "" && f.Account == "" && f.CounterAccount == "" &&
		f.AmountFrom == nil && f.AmountTo == nil && f.TaxKey == "" && f.Actor == "" &&
		f.HasReceipt == nil && strings.TrimSpace(f.Text) == ""
}

// JournalFilterRow ist eine gefilterte Journalzeile mit dem Kopf ihrer Buchung.
type JournalFilterRow struct {
	EntryID        uint             `json:"entryId"`
	EntryNumber    string           `json:"entryNumber"`
	BookingDate    string           `json:"bookingDate"`
	DocumentDate   string           `json:"documentDate"`
	DocumentNumber string           `json:"documentNumber,omitempty"`
	Description    string           `json:"description"`
	Kind           domain.EntryKind `json:"kind"`
	Actor          string           `json:"actor,omitempty"`
	ReceiptID      *uint            `json:"receiptId,omitempty"`
	Position       int              `json:"position"`
	Account        string           `json:"account"`
	AccountName    string           `json:"accountName,omitempty"`
	Side           domain.Side      `json:"side"`
	Amount         domain.Cents     `json:"amount"`
	TaxKey         string           `json:"taxKey,omitempty"`
	TaxBase        domain.Cents     `json:"taxBase,omitempty"`
	Text           string           `json:"text,omitempty"`
}

// JournalFilterResult ist die gefilterte Menge mit ihrer Summenzeile.
type JournalFilterResult struct {
	Rows []JournalFilterRow `json:"rows"`
	// RowCount und EntryCount stehen nebeneinander, weil beide gefragt werden:
	// wie viele Zeilen trifft der Filter, und über wie viele Buchungen
	// verteilen sie sich.
	RowCount    int          `json:"rowCount"`
	EntryCount  int          `json:"entryCount"`
	TotalDebit  domain.Cents `json:"totalDebit"`
	TotalCredit domain.Cents `json:"totalCredit"`
	// Balance ist Soll minus Haben der gefilterten Menge. Sie ist nicht
	// notwendig null: ein Filter schneidet Buchungen an, und das soll er.
	Balance domain.Cents `json:"balance"`
}

// FilterJournal wendet den Filter auf die Buchungen an und rechnet die Summen.
//
// accountName ist optional und liefert die Bezeichnung eines Kontos; ohne sie
// bleibt die Spalte leer, statt die Kontonummer zu wiederholen.
func FilterJournal(
	entries []domain.JournalEntry, filter JournalFilter, accountName func(string) string,
) JournalFilterResult {
	result := JournalFilterResult{Rows: make([]JournalFilterRow, 0, 32)}
	seen := map[uint]bool{}

	for i := range entries {
		entry := &entries[i]
		if !filterMatchesEntry(entry, filter) {
			continue
		}
		for _, line := range entry.Lines {
			if !filterMatchesLine(entry, line, filter) {
				continue
			}
			row := JournalFilterRow{
				EntryID: entry.ID, EntryNumber: entry.EntryNumber,
				BookingDate: entry.BookingDate, DocumentDate: entry.DocumentDate,
				DocumentNumber: entry.DocumentNumber, Description: entry.Description,
				Kind: entry.Kind, Actor: entry.Actor, ReceiptID: entry.ReceiptID,
				Position: line.Position, Account: line.Account, Side: line.Side,
				Amount: line.Amount, TaxKey: line.TaxKey, TaxBase: line.TaxBase,
				Text: line.Text,
			}
			if accountName != nil {
				row.AccountName = accountName(line.Account)
			}
			if line.Side == domain.SideDebit {
				result.TotalDebit += line.Amount
			} else {
				result.TotalCredit += line.Amount
			}
			result.Rows = append(result.Rows, row)
			seen[entry.ID] = true
		}
	}

	sort.SliceStable(result.Rows, func(i, j int) bool {
		if result.Rows[i].BookingDate != result.Rows[j].BookingDate {
			return result.Rows[i].BookingDate < result.Rows[j].BookingDate
		}
		if result.Rows[i].EntryNumber != result.Rows[j].EntryNumber {
			return result.Rows[i].EntryNumber < result.Rows[j].EntryNumber
		}
		return result.Rows[i].Position < result.Rows[j].Position
	})
	result.RowCount = len(result.Rows)
	result.EntryCount = len(seen)
	result.Balance = result.TotalDebit - result.TotalCredit
	return result
}

// filterMatchesEntry prüft die Bedingungen, die am Kopf der Buchung hängen.
func filterMatchesEntry(entry *domain.JournalEntry, f JournalFilter) bool {
	if f.From != "" && entry.BookingDate < f.From {
		return false
	}
	if f.To != "" && entry.BookingDate > f.To {
		return false
	}
	if f.Actor != "" && !strings.EqualFold(entry.Actor, f.Actor) {
		return false
	}
	if f.HasReceipt != nil {
		has := entry.ReceiptID != nil && *entry.ReceiptID != 0
		if has != *f.HasReceipt {
			return false
		}
	}
	if f.CounterAccount != "" && !hasAccount(entry, f.CounterAccount) {
		return false
	}
	if text := strings.TrimSpace(strings.ToLower(f.Text)); text != "" {
		if !entryContains(entry, text) {
			return false
		}
	}
	return true
}

// filterMatchesLine prüft die Bedingungen, die an der einzelnen Zeile hängen.
func filterMatchesLine(entry *domain.JournalEntry, l domain.JournalLine, f JournalFilter) bool {
	if f.Account != "" && l.Account != f.Account {
		return false
	}
	// Das Gegenkonto ist ein Konto derselben Buchung auf der anderen Seite. Die
	// Zeile, die es selbst trägt, gehört nicht ins Ergebnis, wenn zugleich nach
	// einem Konto gefiltert wird — sonst stünde dieselbe Buchung zweimal in der
	// Liste, einmal von jeder Seite.
	if f.Account != "" && f.CounterAccount != "" && l.Account == f.CounterAccount {
		return false
	}
	amount := l.Amount
	if amount < 0 {
		amount = -amount
	}
	if f.AmountFrom != nil && amount < *f.AmountFrom {
		return false
	}
	if f.AmountTo != nil && amount > *f.AmountTo {
		return false
	}
	if f.TaxKey != "" && l.TaxKey != f.TaxKey {
		return false
	}
	_ = entry
	return true
}

func hasAccount(entry *domain.JournalEntry, account string) bool {
	for _, l := range entry.Lines {
		if l.Account == account {
			return true
		}
	}
	return false
}

func entryContains(entry *domain.JournalEntry, needle string) bool {
	fields := []string{
		entry.Description, entry.DocumentNumber, entry.EntryNumber, entry.ReversalReason,
	}
	for _, l := range entry.Lines {
		fields = append(fields, l.Text)
	}
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), needle) {
			return true
		}
	}
	return false
}
