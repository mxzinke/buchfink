package domain

import (
	"context"
	"fmt"
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
	EntrySourceManual       EntrySource = "manual"
	EntrySourceReceipt      EntrySource = "receipt"      // Eingangsbeleg
	EntrySourceInvoice      EntrySource = "invoice"      // Ausgangsrechnung
	EntrySourcePayment      EntrySource = "payment"      // Zahlung / OP-Ausgleich
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
	ServiceDateFrom string `gorm:"size:10;not null" json:"serviceDateFrom"`   // Leistungsdatum / -beginn (§ 14 Abs. 4 Nr. 6 UStG)
	ServiceDateTo   string `gorm:"size:10;not null" json:"serviceDateTo"`     // Leistungsende; gleich From bei Zeitpunktleistung
	ValueDate       string `gorm:"size:10" json:"valueDate,omitempty"`        // Valuta, nur bei Zahlungsbuchungen

	Description string      `gorm:"size:500;not null;serializer:encrypted" json:"description"`
	Source      EntrySource `gorm:"size:20;not null;index" json:"source"`

	// Belegverweis. DocumentNumber is the Belegfeld the auditor uses to find the
	// paper/PDF; DocumentHash pins the file content.
	DocumentNumber string `gorm:"size:100;index" json:"documentNumber,omitempty"`
	DocumentHash   string `gorm:"size:64" json:"documentHash,omitempty"`
	DocumentPath   string `gorm:"size:255;serializer:encrypted" json:"documentPath,omitempty"`

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
		return fmt.Errorf("Leistungsdatum fehlt (Pflichtangabe nach § 14 Abs. 4 Nr. 6 UStG)")
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
	FindByID(ctx context.Context, id uint) (*JournalEntry, error)
	FindByAccount(ctx context.Context, account string, fiscalYear int) ([]JournalEntry, error)
	FindByContact(ctx context.Context, contactID uint, fiscalYear int) ([]JournalEntry, error)
	FindReversalOf(ctx context.Context, entryID uint) (*JournalEntry, error)
	GetLastEntry(ctx context.Context, fiscalYear int) (*JournalEntry, error)
	// Append allocates the entry number, links the hash chain and persists the
	// entry in a single transaction. Doing all three atomically is what keeps
	// the numbering gapless: a rolled-back insert must not consume a number, and
	// two concurrent writers must not read the same chain head.
	Append(ctx context.Context, entry *JournalEntry, hash EntryHashFunc) error
	// AccountTurnovers returns Soll/Haben sums per account number for a fiscal
	// year in a single pass.
	AccountTurnovers(ctx context.Context, fiscalYear int) (map[string]AccountTurnover, error)
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

// IntegrityCheckResult is the outcome of a hash chain verification.
type IntegrityCheckResult struct {
	IsValid          bool   `json:"isValid"`
	TotalEntries     int    `json:"totalEntries"`
	CheckedEntries   int    `json:"checkedEntries"`
	FirstBrokenID    *uint  `json:"firstBrokenId,omitempty"`
	Message          string `json:"message"`
	LastVerifiedHash string `json:"lastVerifiedHash"`
	CheckedAt        string `json:"checkedAt"`
}
