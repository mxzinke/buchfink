package domain

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// GenesisHash is the root anchor for the first journal entry of a fiscal year.
const GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

// Side is the debit/credit side of a journal line.
type Side string

const (
	SideDebit  Side = "S" // Soll
	SideCredit Side = "H" // Haben
)

// Opposite returns the other side: where the Gegenbuchung sits, and the side a
// correction has to use to take a line back.
func (s Side) Opposite() Side {
	if s == SideDebit {
		return SideCredit
	}
	return SideDebit
}

// EntryKind separates original bookings from corrections.
type EntryKind string

const (
	// EntryKindNormal is an original business transaction.
	EntryKindNormal EntryKind = "normal"
	// EntryKindReversal is a Generalumkehr: same accounts, same sides, negated
	// amounts. Unlike a side-swapped Storno it leaves the Verkehrszahlen
	// (turnover figures) of the affected accounts at zero, so the Summen- und
	// Saldenliste and the derived VAT figures stay correct after a correction.
	EntryKindReversal EntryKind = "reversal"
)

// EntrySource records which part of the system produced an entry. It is part of
// the Verfahrensdokumentation: an auditor must be able to see how a booking came
// into existence.
type EntrySource string

const (
	EntrySourceManual  EntrySource = "manual"
	EntrySourceReceipt EntrySource = "receipt" // Eingangsbeleg
	EntrySourceInvoice EntrySource = "invoice" // Ausgangsrechnung
	EntrySourcePayment EntrySource = "payment" // Zahlung / OP-Ausgleich
	// EntrySourceAdvance ist die Vereinnahmung einer Anzahlung. Sie ist eine
	// eigene Quelle und keine gewöhnliche Zahlung, weil mit ihr die Steuer
	// entsteht (§ 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG) — eine Zahlung tut das
	// sonst nie, und die Voranmeldung muss die beiden auseinanderhalten können.
	EntrySourceAdvance      EntrySource = "advance"
	EntrySourceOpening      EntrySource = "opening"      // Eröffnungsbilanz
	EntrySourceDepreciation EntrySource = "depreciation" // AfA
	EntrySourceClosing      EntrySource = "closing"      // Abschlussbuchung
)

// JournalLine is a single Soll- or Haben-Zeile of a booking.
//
// Amount is always in Cents and normally positive. For an EntryKindReversal
// entry the amounts are negative on the same side as the original, which is what
// makes it a Generalumkehr rather than a side swap.
type JournalLine struct {
	ID       uint  `gorm:"primaryKey" json:"id"`
	EntryID  uint  `gorm:"index;not null" json:"entryId"`
	Position int   `gorm:"not null" json:"position"`
	Side     Side  `gorm:"size:1;not null;index" json:"side"`
	Amount   Cents `gorm:"not null" json:"amount"`

	// Account is either a Sachkonto (4 digits, e.g. "6815") or a Personenkonto
	// (5 digits: 10000-69999 Debitoren, 70000-99999 Kreditoren).
	Account     string `gorm:"size:10;not null;index" json:"account"`
	AccountName string `gorm:"-" json:"accountName,omitempty"`

	// ContactID is set on Personenkonto lines and links the open item to its
	// business partner.
	ContactID *uint `gorm:"index" json:"contactId,omitempty"`

	// TaxKey names the Steuerschlüssel this line was derived from (see tax.go).
	// It is empty on lines that carry no VAT relevance.
	TaxKey string `gorm:"size:30;index" json:"taxKey,omitempty"`
	// TaxBase carries the same sign as Amount, always. A Generalumkehr negates
	// both and keeps the side; a correction that changes the base — a Skonto
	// under § 17 Abs. 1 UStG — writes both positive on the opposite side. Every
	// reader derives the direction from the line's side, so a base that
	// disagreed in sign with its amount would raise a turnover while lowering
	// the tax on it.
	//
	// TaxBase is the Bemessungsgrundlage the tax amount was computed from. Only
	// set on tax lines; needed to reproduce the UStVA figures from the journal.
	TaxBase Cents `gorm:"default:0" json:"taxBase,omitempty"`

	Text string `gorm:"size:255;serializer:encrypted" json:"text,omitempty"`
}

// JournalEntry is one complete Geschäftsvorfall (Buchungssatz) with n lines.
//
// A booking is never a pair of accounts: "Aufwand und Vorsteuer an
// Verbindlichkeit" is the normal case, and a Reverse-Charge purchase produces
// four lines. Keeping head and lines together preserves the transaction as an
// atomic unit, which is what Storno, OPOS matching and the hash chain all
// depend on.
type JournalEntry struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	FiscalYear  int    `gorm:"index;not null" json:"fiscalYear"`
	EntryNumber string `gorm:"size:30;not null;uniqueIndex" json:"entryNumber"` // e.g. "2026-000001"

	// Four distinct dates. Collapsing them is the most common cause of wrong
	// periods and wrong VAT returns, so each has its own field.
	BookingDate     string `gorm:"size:10;not null;index" json:"bookingDate"` // Buchungsdatum – bestimmt die Periode
	DocumentDate    string `gorm:"size:10;not null" json:"documentDate"`      // Belegdatum (Rechnungsdatum)
	ServiceDateFrom string `gorm:"size:10;not null" json:"serviceDateFrom"`   // Leistungsdatum / -beginn; der Zeitpunkt ist Pflichtangabe nach § 14 Abs. 4 Nr. 6 UStG
	ServiceDateTo   string `gorm:"size:10;not null" json:"serviceDateTo"`     // Leistungsende; gleich From bei Zeitpunktleistung
	ValueDate       string `gorm:"size:10" json:"valueDate,omitempty"`        // Valuta, nur bei Zahlungsbuchungen

	Description string      `gorm:"size:500;not null;serializer:encrypted" json:"description"`
	Source      EntrySource `gorm:"size:20;not null;index" json:"source"`

	// Belegverweis. DocumentNumber is the Belegfeld the auditor uses to find the
	// document and the field the DATEV export carries; ReceiptHash pins the
	// Beleg's whole file list at once.
	//
	// This used to be a single file: one DocumentHash, one DocumentPath. That
	// cannot hold a ZUGFeRD invoice (PDF plus XML) or an XRechnung (XML plus a
	// generated rendering), so the reference now points at the Beleg entity.
	// ReceiptID is deliberately kept out of the hash chain — like the old path it
	// is a location, not content — while ReceiptHash and DocumentNumber together
	// identify the Beleg even if two of them happen to hold identical files.
	DocumentNumber string `gorm:"size:100;index" json:"documentNumber,omitempty"`
	ReceiptID      *uint  `gorm:"index" json:"receiptId,omitempty"`
	ReceiptHash    string `gorm:"size:64" json:"receiptHash,omitempty"`

	// TaxTreatment records which Steuerfall produced this booking.
	//
	// It used to be input only, recoverable afterwards from the accounts alone.
	// That works on the outgoing side, where every case has its own revenue
	// account, but not on the incoming one: a purchase at the Nullsteuersatz of
	// § 12 Abs. 3 UStG books to the same expense account as an exempt one and has
	// no tax line either. Without this field the two would be indistinguishable
	// once booked — and they are not the same thing, because one keeps the input
	// tax deduction and the other does not.
	TaxTreatment TaxTreatment `gorm:"size:40;index" json:"taxTreatment,omitempty"`

	ContactID *uint `gorm:"index" json:"contactId,omitempty"`
	BankTxID  *uint `gorm:"index" json:"bankTxId,omitempty"`

	// Correction handling.
	Kind           EntryKind `gorm:"size:20;not null;default:'normal';index" json:"kind"`
	ReversalOfID   *uint     `gorm:"index" json:"reversalOfId,omitempty"`
	ReversalReason string    `gorm:"size:255;serializer:encrypted" json:"reversalReason,omitempty"`

	// Foreign currency. The rate is stored as an integer in millionths so that
	// re-computing a converted amount from the journal is deterministic.
	Currency           string `gorm:"size:3;not null;default:'EUR'" json:"currency"`
	ExchangeRateMicros int64  `gorm:"not null;default:1000000" json:"exchangeRateMicros"`
	ExchangeRateSource string `gorm:"size:50" json:"exchangeRateSource,omitempty"`
	ExchangeRateDate   string `gorm:"size:10" json:"exchangeRateDate,omitempty"`

	// PostingRuleVersion records which version of the Gruppen→Konten mapping
	// produced this booking. Without it a later change to the mapping would make
	// historical bookings unexplainable.
	PostingRuleVersion string `gorm:"size:20" json:"postingRuleVersion,omitempty"`

	Lines []JournalLine `gorm:"foreignKey:EntryID;constraint:OnDelete:CASCADE" json:"lines"`

	// Entertainment carries the Aufzeichnung § 4 Abs. 5 Satz 1 Nr. 2 EStG
	// demands whenever entertainment expenses are booked. It hangs off the entry
	// rather than the Beleg because the Beleg-Hash covers only the file list: a
	// participant list stored there would be covered by no checksum at all, and a
	// record the deduction depends on must not be silently editable.
	Entertainment *EntertainmentDetail `gorm:"foreignKey:EntryID;constraint:OnDelete:CASCADE" json:"entertainment,omitempty"`

	PreviousHash string    `gorm:"size:64;not null" json:"previousHash"`
	EntryHash    string    `gorm:"size:64;not null" json:"entryHash"`
	CreatedAt    time.Time `json:"createdAt"`
}

// DebitTotal sums the Soll side.
func (e *JournalEntry) DebitTotal() Cents {
	var total Cents
	for _, l := range e.Lines {
		if l.Side == SideDebit {
			total += l.Amount
		}
	}
	return total
}

// CreditTotal sums the Haben side.
func (e *JournalEntry) CreditTotal() Cents {
	var total Cents
	for _, l := range e.Lines {
		if l.Side == SideCredit {
			total += l.Amount
		}
	}
	return total
}

// GrossAmount is the absolute size of the transaction, used for display and for
// OPOS comparisons. It is the larger of the two sides in absolute terms, which
// equals either side once the entry is balanced.
func (e *JournalEntry) GrossAmount() Cents {
	return e.DebitTotal().Abs()
}

// IsBalanced reports whether Soll equals Haben. Exact, no tolerance.
func (e *JournalEntry) IsBalanced() bool {
	return e.DebitTotal() == e.CreditTotal()
}

// Validate enforces the invariants every booking must satisfy before it may
// enter the journal. These are hard rules, not warnings: a journal that can hold
// an unbalanced or account-less entry cannot produce a correct Bilanz.
func (e *JournalEntry) Validate() error {
	if len(e.Lines) < 2 {
		return fmt.Errorf("eine Buchung braucht mindestens zwei Zeilen (Soll und Haben), hat aber %d", len(e.Lines))
	}
	if e.BookingDate == "" {
		return fmt.Errorf("Buchungsdatum fehlt")
	}
	if e.DocumentDate == "" {
		return fmt.Errorf("Belegdatum fehlt")
	}
	if e.ServiceDateFrom == "" || e.ServiceDateTo == "" {
		return fmt.Errorf("Leistungsdatum fehlt (der Leistungszeitpunkt ist Pflichtangabe nach § 14 Abs. 4 Nr. 6 UStG)")
	}
	if e.ServiceDateTo < e.ServiceDateFrom {
		return fmt.Errorf("Leistungsende %s liegt vor dem Leistungsbeginn %s", e.ServiceDateTo, e.ServiceDateFrom)
	}
	if e.Description == "" {
		return fmt.Errorf("Buchungstext fehlt")
	}

	var hasDebit, hasCredit bool
	for i, l := range e.Lines {
		if l.Account == "" {
			return fmt.Errorf("Zeile %d: Konto fehlt", i+1)
		}
		if l.Side != SideDebit && l.Side != SideCredit {
			return fmt.Errorf("Zeile %d: ungültige Buchungsseite %q", i+1, l.Side)
		}
		if l.Amount == 0 {
			return fmt.Errorf("Zeile %d: Betrag darf nicht null sein", i+1)
		}
		switch e.Kind {
		case EntryKindNormal:
			if l.Amount < 0 {
				return fmt.Errorf("Zeile %d: negative Beträge sind nur bei Generalumkehr zulässig", i+1)
			}
		case EntryKindReversal:
			if l.Amount > 0 {
				return fmt.Errorf("Zeile %d: eine Generalumkehr muss negative Beträge tragen", i+1)
			}
		}
		if l.Side == SideDebit {
			hasDebit = true
		} else {
			hasCredit = true
		}
	}

	if !hasDebit || !hasCredit {
		return fmt.Errorf("eine Buchung braucht mindestens eine Soll- und eine Haben-Zeile")
	}
	if !e.IsBalanced() {
		return fmt.Errorf(
			"Buchung ist nicht ausgeglichen: Soll %s €, Haben %s € (Differenz %s €)",
			e.DebitTotal(), e.CreditTotal(), (e.DebitTotal() - e.CreditTotal()),
		)
	}
	if e.Kind == EntryKindReversal && e.ReversalOfID == nil {
		return fmt.Errorf("eine Generalumkehr muss auf die ursprüngliche Buchung verweisen")
	}
	if e.Kind != EntryKindReversal && e.ReversalOfID != nil {
		return fmt.Errorf("nur eine Generalumkehr darf auf eine ursprüngliche Buchung verweisen")
	}
	return nil
}

// EntertainmentDetail is the record § 4 Abs. 5 Satz 1 Nr. 2 Sätze 2 und 3 EStG
// requires for entertainment expenses: place, day, participants and occasion,
// alongside the amount. For a restaurant bill the occasion and the participants
// suffice and the invoice is attached.
//
// It is not a booking line but a written record the deduction hangs on — without
// it even the deductible 70 % are lost. It is therefore part of the entry's
// canonical form and covered by the hash chain.
type EntertainmentDetail struct {
	ID      uint `gorm:"primaryKey" json:"id"`
	EntryID uint `gorm:"uniqueIndex;not null" json:"entryId"`

	Place        string `gorm:"size:255;not null;serializer:encrypted" json:"place"`
	Day          string `gorm:"size:10;not null" json:"day"`
	Participants string `gorm:"type:text;not null;serializer:encrypted" json:"participants"`
	Occasion     string `gorm:"size:500;not null;serializer:encrypted" json:"occasion"`
}

// Validate checks that the record is complete. An incomplete one is worse than
// none: it looks like compliance and is not.
func (d *EntertainmentDetail) Validate() error {
	missing := make([]string, 0, 4)
	if strings.TrimSpace(d.Place) == "" {
		missing = append(missing, "Ort")
	}
	if strings.TrimSpace(d.Day) == "" {
		missing = append(missing, "Tag")
	}
	if strings.TrimSpace(d.Participants) == "" {
		missing = append(missing, "Teilnehmer")
	}
	if strings.TrimSpace(d.Occasion) == "" {
		missing = append(missing, "Anlass")
	}
	if len(missing) > 0 {
		return fmt.Errorf(
			"zu einer Bewirtung gehören Ort, Tag, Teilnehmer und Anlass (§ 4 Abs. 5 Satz 1 Nr. 2 EStG). Es fehlt: %s",
			strings.Join(missing, ", "))
	}
	return nil
}

// AccountTurnover holds the Verkehrszahlen of one account.
type AccountTurnover struct {
	Debit  Cents `json:"debit"`
	Credit Cents `json:"credit"`
	Count  int   `json:"count"`
	// Aggregated counts the Personenkonten folded into this account. It is set
	// only on the Sammelkonten and lets the UI name where the figure comes from
	// instead of showing a balance with no visible origin.
	Aggregated int `json:"aggregated,omitempty"`
}

// EntryHashFunc links an entry to its predecessor in the hash chain.
type EntryHashFunc func(e *JournalEntry, prevHash string) string

// JournalRepository defines persistence for journal entries.
type JournalRepository interface {
	FindAll(ctx context.Context, fiscalYear int) ([]JournalEntry, error)
	// FindByBookingDateRange narrows to a booking-date window inside a fiscal
	// year; empty bounds mean the whole year. The Umsatzsteuer-Voranmeldung is
	// monthly or quarterly, and filtering a whole year in Go to report one month
	// of it reads twelve times what it needs.
	FindByBookingDateRange(ctx context.Context, fiscalYear int, from, to string) ([]JournalEntry, error)
	// FindOpenItemCandidates returns the entries a Forderung or Verbindlichkeit
	// could still sit in: the given year and every earlier one, minus everything
	// a Generalumkehr has cancelled.
	//
	// Both bounds are there for the same reason. § 252 Abs. 1 Nr. 5 HGB puts the
	// Ertrag in the year of performance and the payment in the year it happens,
	// so an invoice and its settlement routinely sit in different years — a view
	// bounded to one of them sees half the story. And a Generalumkehr carries the
	// date of the correction, so it regularly sits in a *later* year than the
	// entry it cancels: whether a booking still stands cannot be answered inside
	// a year window at all.
	FindOpenItemCandidates(ctx context.Context, fiscalYear int) ([]JournalEntry, error)
	// FindOpenItemCandidatesAt beantwortet dieselbe Frage zu einem Stichtag:
	// welche Posten waren am Bilanzstichtag offen.
	//
	// Das ist nicht dieselbe Abfrage mit einem zusätzlichen Filter. Die
	// operative Sicht fragt, was heute noch offen ist, und wirft deshalb jede
	// Buchung weg, die jemals storniert wurde. Zum Stichtag zählt aber der
	// Stand von damals: eine Rechnung aus dem Dezember, die im März storniert
	// wurde, stand am 31.12. in der Bilanz und gehört in den Saldenvortrag —
	// der Storno nimmt sie im neuen Jahr wieder heraus. Deshalb ist auch die
	// Generalumkehr auf ihr Datum begrenzt.
	FindOpenItemCandidatesAt(ctx context.Context, cutoff string) ([]JournalEntry, error)
	FindByID(ctx context.Context, id uint) (*JournalEntry, error)
	FindByAccount(ctx context.Context, account string, fiscalYear int) ([]JournalEntry, error)
	// FindByAccountRange ist dasselbe Kontoblatt in einem Datumsfenster über
	// alle Geschäftsjahre hinweg. Ein Prüfer fragt nach einem Zeitraum und
	// nicht nach einem Geschäftsjahr; leere Grenzen heißen: alles.
	FindByAccountRange(ctx context.Context, account, from, to string) ([]JournalEntry, error)
	FindByContact(ctx context.Context, contactID uint, fiscalYear int) ([]JournalEntry, error)
	FindReversalOf(ctx context.Context, entryID uint) (*JournalEntry, error)
	// FindByReceipt returns the original booking that references a Beleg, or nil.
	// It is what lets an unsealed Beleg be repaired: the seal is written after
	// the journal transaction commits, so a crash in between leaves a booked
	// Beleg that still looks open.
	FindByReceipt(ctx context.Context, receiptID uint) (*JournalEntry, error)
	GetLastEntry(ctx context.Context, fiscalYear int) (*JournalEntry, error)
	// Append allocates the entry number, links the hash chain and persists the
	// entry in a single transaction. Doing all three atomically is what keeps
	// the numbering gapless: a rolled-back insert must not consume a number, and
	// two concurrent writers must not read the same chain head.
	Append(ctx context.Context, entry *JournalEntry, hash EntryHashFunc) error
	// AccountTurnovers returns Soll/Haben sums per account number for a fiscal
	// year in a single pass.
	AccountTurnovers(ctx context.Context, fiscalYear int) (map[string]AccountTurnover, error)
	// AccountTurnoversUntil sind dieselben Verkehrszahlen bis zu einem
	// Stichtag. Die Summen- und Saldenliste zum 30.06. ist keine Jahresliste
	// mit einem Filter darüber: sie muss die Buchungen nach dem Stichtag
	// weglassen, bevor summiert wird. Leerer Stichtag heißt: ganzes Jahr.
	AccountTurnoversUntil(ctx context.Context, fiscalYear int, cutoff string) (map[string]AccountTurnover, error)
	MonthlyCashflow(ctx context.Context, fiscalYear int, liquidAccounts []string) ([]CashflowDataPoint, error)
	Count(ctx context.Context, fiscalYear int) (int64, error)
	GetAvailableFiscalYears(ctx context.Context) ([]int, error)
}

// CashflowDataPoint aggregates liquid-account movements for one month.
type CashflowDataPoint struct {
	Month   string `json:"month"` // "2026-01"
	Label   string `json:"label"` // "Jan"
	Inflow  Cents  `json:"inflow"`
	Outflow Cents  `json:"outflow"`
	Net     Cents  `json:"net"`
}

// FinancialSummary contains high-level KPIs for the dashboard.
type FinancialSummary struct {
	TotalRevenue    Cents               `json:"totalRevenue"`
	TotalExpenses   Cents               `json:"totalExpenses"`
	NetIncome       Cents               `json:"netIncome"`
	BankBalance     Cents               `json:"bankBalance"`
	OpenReceivables Cents               `json:"openReceivables"`
	OpenPayables    Cents               `json:"openPayables"`
	CashflowHistory []CashflowDataPoint `json:"cashflowHistory"`
}

// IntegrityBreakReason benennt, woran eine Kette zerbrochen ist.
type IntegrityBreakReason string

const (
	// IntegrityBreakLinkage: der Vorgängerhash der Buchung ist nicht der
	// Eigenhash ihres Vorgängers — eine Buchung wurde eingefügt oder entfernt.
	IntegrityBreakLinkage IntegrityBreakReason = "linkage"
	// IntegrityBreakContent: die Buchung hasht nicht mehr auf ihren
	// gespeicherten Eigenhash — ihre Daten wurden nachträglich verändert.
	IntegrityBreakContent IntegrityBreakReason = "content"
)

// IntegrityBreak ist ein einzelner Bruch der Kette.
//
// Er nennt erwarteten und tatsächlichen Hash, weil die Angabe „Buchung 42 ist
// gebrochen" außerhalb von Buchfink nicht nachrechenbar ist. Wer den erwarteten
// Wert kennt, kann mit der Kanonisierung aus der Feldbeschreibung selbst
// prüfen, welche Seite recht hat.
type IntegrityBreak struct {
	FiscalYear   int                  `json:"fiscalYear"`
	EntryID      uint                 `json:"entryId"`
	EntryNumber  string               `json:"entryNumber"`
	Reason       IntegrityBreakReason `json:"reason"`
	ExpectedHash string               `json:"expectedHash"`
	ActualHash   string               `json:"actualHash"`
	Message      string               `json:"message"`
}

// IntegrityCheckResult is the outcome of a hash chain verification.
//
// Geprüft wird jedes Geschäftsjahr für sich: die Kette beginnt je Jahr neu beim
// Genesis-Hash. Eine Prüfung, die nur das aktive Jahr ansieht, meldet „alles in
// Ordnung", während in einem abgeschlossenen Jahr eine Zeile verändert wurde.
type IntegrityCheckResult struct {
	IsValid      bool `json:"isValid"`
	TotalEntries int  `json:"totalEntries"`
	// CheckedEntries ist die Zahl der nachgerechneten Buchungen und damit
	// gleich TotalEntries: die Prüfung läuft nach einem Bruch weiter, statt an
	// ihm abzubrechen — sonst verdeckte die erste geänderte Buchung jede
	// spätere. Das Feld sagt also, wie viel geprüft wurde, und nicht, wo es
	// aufgehört hat; wo die Kette bricht, steht in Breaks.
	CheckedEntries   int    `json:"checkedEntries"`
	FirstBrokenID    *uint  `json:"firstBrokenId,omitempty"`
	Message          string `json:"message"`
	LastVerifiedHash string `json:"lastVerifiedHash"`
	CheckedAt        string `json:"checkedAt"`

	// FiscalYears sind die geprüften Geschäftsjahre, aufsteigend.
	FiscalYears []int `json:"fiscalYears"`
	// Breaks sind alle gefundenen Brüche, nicht nur der erste. Nach einem Bruch
	// läuft die Prüfung weiter, sonst verdeckte die erste geänderte Buchung
	// jede spätere.
	Breaks []IntegrityBreak `json:"breaks"`
}

// EnsureLists ersetzt nicht belegte Listen durch leere.
//
// Das Ergebnis geht als JSON an die Oberfläche; ein nicht belegter Slice wird
// dort zu `null`, und `breaks.length` bräche ausgerechnet im Regelfall — der
// unversehrten Buchführung.
func (r *IntegrityCheckResult) EnsureLists() {
	if r.Breaks == nil {
		r.Breaks = make([]IntegrityBreak, 0)
	}
	if r.FiscalYears == nil {
		r.FiscalYears = make([]int, 0)
	}
}
