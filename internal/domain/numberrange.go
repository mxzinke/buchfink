package domain

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// NumberRangeKey identifies a gapless counter.
//
// GoBD requires numbering to be gapless and free of duplicates, and § 14 Abs. 4
// Nr. 4 UStG requires outgoing invoice numbers to be unique and consecutive.
// Deriving a number from a database row id does not satisfy either: ids are
// shared across fiscal years and skip on rollback. Every series therefore has
// its own counter, allocated inside the same transaction that writes the record.
type NumberRangeKey string

const (
	NumberRangeJournal  NumberRangeKey = "journal"  // Buchungsnummer, je Geschäftsjahr
	NumberRangeReceipt  NumberRangeKey = "receipt"  // Eingangsbeleg, je Geschäftsjahr
	NumberRangeInvoice  NumberRangeKey = "invoice"  // Ausgangsrechnung, je Geschäftsjahr
	NumberRangeDebitor  NumberRangeKey = "debitor"  // Debitorenkonto, jahresübergreifend
	NumberRangeCreditor NumberRangeKey = "creditor" // Kreditorenkonto, jahresübergreifend
)

// Personenkonten-Nummernkreise nach DATEV SKR04:
// Sollsalden Forderungen aus LuL 10000-69999 = Debitoren,
// Habensalden Verbindlichkeiten aus LuL 70000-99999 = Kreditoren.
const (
	DebitorRangeStart  = 10000
	DebitorRangeEnd    = 69999
	CreditorRangeStart = 70000
	CreditorRangeEnd   = 99999
)

// NumberRange persists the next free value of one counter.
type NumberRange struct {
	Key        NumberRangeKey `gorm:"primaryKey;size:30" json:"key"`
	FiscalYear int            `gorm:"primaryKey" json:"fiscalYear"` // 0 = jahresübergreifend
	Next       int64          `gorm:"not null" json:"next"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

// NumberRangeRepository allocates numbers atomically.
type NumberRangeRepository interface {
	// Allocate consumes and returns the next value of a counter.
	Allocate(ctx context.Context, key NumberRangeKey, fiscalYear int) (int64, error)
	// Peek reports the next value without consuming it.
	Peek(ctx context.Context, key NumberRangeKey, fiscalYear int) (int64, error)
}

// FormatJournalNumber renders a Buchungsnummer, e.g. "2026-000001".
func FormatJournalNumber(fiscalYear int, seq int64) string {
	return fmt.Sprintf("%d-%06d", fiscalYear, seq)
}

// FormatReceiptNumber renders an Eingangsbeleg number, e.g. "ER-2026-0001".
func FormatReceiptNumber(fiscalYear int, seq int64) string {
	return fmt.Sprintf("ER-%d-%04d", fiscalYear, seq)
}

// FormatInvoiceNumber renders an Ausgangsrechnung number, e.g. "RE-2026-0001".
func FormatInvoiceNumber(fiscalYear int, seq int64) string {
	return FormatInvoiceNumberWith(DefaultInvoiceNumberFormat, fiscalYear, seq)
}

// DefaultInvoiceNumberFormat ist die Voreinstellung des Nummernkreises.
//
// Das Format ist einstellbar (Einstellung `invoice_number_format`), weil ein
// Mandant mit vorhandener Buchhaltung seine Systematik fortführen muss: eine
// neue Software, die bei RE-2026-0001 anfängt, während die alte bei 2026-0473
// stand, produziert doppelte Nummern. Die gewählte Systematik gehört in die
// Verfahrensdokumentation — deshalb steht sie als Einstellung und nicht im Code.
const DefaultInvoiceNumberFormat = "RE-{JAHR}-{NR:4}"

// FormatInvoiceNumberWith renders a number from a configured format.
//
// Zwei Platzhalter: `{JAHR}` das Geschäftsjahr, `{NR:n}` der Zähler mit n
// führenden Nullen (`{NR}` ohne Auffüllung). Ein Format ohne `{NR}` wäre kein
// Nummernkreis — jede Rechnung hieße gleich —, deshalb wird es abgewiesen und
// nicht stillschweigend ergänzt.
func FormatInvoiceNumberWith(format string, fiscalYear int, seq int64) string {
	if err := ValidateInvoiceNumberFormat(format); err != nil {
		format = DefaultInvoiceNumberFormat
	}
	out := strings.ReplaceAll(format, "{JAHR}", fmt.Sprintf("%d", fiscalYear))
	out = numberPlaceholder.ReplaceAllStringFunc(out, func(match string) string {
		digits := numberPlaceholder.FindStringSubmatch(match)[1]
		if digits == "" {
			return fmt.Sprintf("%d", seq)
		}
		width, err := strconv.Atoi(digits)
		if err != nil || width <= 0 {
			return fmt.Sprintf("%d", seq)
		}
		return fmt.Sprintf("%0*d", width, seq)
	})
	return out
}

var numberPlaceholder = regexp.MustCompile(`\{NR(?::(\d+))?\}`)

// ValidateInvoiceNumberFormat rejects a format that could not produce a unique,
// consecutive number (§ 14 Abs. 4 Nr. 4 UStG).
func ValidateInvoiceNumberFormat(format string) error {
	if strings.TrimSpace(format) == "" {
		return fmt.Errorf("das Nummernformat ist leer")
	}
	if !numberPlaceholder.MatchString(format) {
		return fmt.Errorf(
			"im Nummernformat %q fehlt der Platzhalter {NR} für den Zähler. Ohne ihn trüge jede Rechnung dieselbe Nummer",
			format)
	}
	if len(format) > 40 {
		return fmt.Errorf("das Nummernformat ist länger als 40 Zeichen und passt nicht in das Belegfeld")
	}
	return nil
}

// ParseInvoiceSequence reads the counter value back out of a formatted number.
//
// Der Lückenbericht braucht die Umkehrung: er vergleicht die vergebenen Nummern
// mit dem Stand des Zählers, und dafür muss aus „RE-2026-0007" die 7 werden.
//
// Gelesen wird mit dem Nummernformat und nicht nach Gefühl. Vorher suchte die
// Umkehrung „die längste Ziffernfolge, die nicht das Geschäftsjahr ist"; ein
// Format ohne Trennzeichen zwischen den Platzhaltern — `{JAHR}{NR:4}` ist
// zulässig — ergibt aber „20260007", also eine einzige Ziffernfolge, die nicht
// das Jahr ist. Der Bericht las daraus den Zähler 20260007 und meldete jede
// vergebene Nummer als Lücke.
//
// Nummern aus einem früher eingestellten Format passen nicht auf den heutigen
// Ausdruck. Für sie bleibt die alte Heuristik als zweiter Weg: sie ist ungenau,
// aber eine nicht gelesene Nummer wäre im Bericht eine Lücke, die es nicht gibt
// — und die Betriebsprüfung fragt nach genau diesen Zeilen.
func ParseInvoiceSequence(number string, fiscalYear int, format string) (int64, bool) {
	if pattern := invoiceNumberPattern(format, fiscalYear); pattern != nil {
		if match := pattern.FindStringSubmatch(number); match != nil {
			if seq, err := strconv.ParseInt(match[1], 10, 64); err == nil {
				return seq, true
			}
		}
	}
	year := fmt.Sprintf("%d", fiscalYear)
	groups := digitRun.FindAllString(number, -1)
	for i := len(groups) - 1; i >= 0; i-- {
		if groups[i] == year {
			continue
		}
		seq, err := strconv.ParseInt(groups[i], 10, 64)
		if err != nil {
			continue
		}
		return seq, true
	}
	return 0, false
}

// invoiceNumberPattern baut aus dem Nummernformat den Ausdruck, der eine Nummer
// wieder zerlegt: `{JAHR}` wird zum Geschäftsjahr, `{NR:n}` zur Fangklammer um
// den Zähler, alles andere bleibt wörtlich. Der Zähler wird bewusst nicht auf
// die Stellenzahl des Formats festgelegt — er läuft über sie hinaus, sobald das
// Jahr mehr Rechnungen trägt, als die Auffüllung vorsieht.
func invoiceNumberPattern(format string, fiscalYear int) *regexp.Regexp {
	if ValidateInvoiceNumberFormat(format) != nil {
		format = DefaultInvoiceNumberFormat
	}
	var expr strings.Builder
	expr.WriteString("^")
	rest := strings.ReplaceAll(format, "{JAHR}", fmt.Sprintf("%d", fiscalYear))
	for {
		loc := numberPlaceholder.FindStringIndex(rest)
		if loc == nil {
			break
		}
		expr.WriteString(regexp.QuoteMeta(rest[:loc[0]]))
		expr.WriteString(`(\d+)`)
		rest = rest[loc[1]:]
	}
	expr.WriteString(regexp.QuoteMeta(rest))
	expr.WriteString("$")
	pattern, err := regexp.Compile(expr.String())
	if err != nil {
		return nil
	}
	// Mehr als eine Fangklammer hieße mehr als ein Zähler in einer Nummer; die
	// Umkehrung wäre dann nicht eindeutig, und geraten wird hier nicht.
	if pattern.NumSubexp() != 1 {
		return nil
	}
	return pattern
}

var digitRun = regexp.MustCompile(`\d+`)

// FormatLedgerAccount renders a Personenkonto number from a sequence value.
func FormatLedgerAccount(kind ContactType, seq int64) (string, error) {
	switch kind {
	case ContactTypeCustomer:
		n := DebitorRangeStart + seq - 1
		if n > DebitorRangeEnd {
			return "", fmt.Errorf("Debitoren-Nummernkreis %d-%d ist erschöpft", DebitorRangeStart, DebitorRangeEnd)
		}
		return fmt.Sprintf("%d", n), nil
	case ContactTypeVendor:
		n := CreditorRangeStart + seq - 1
		if n > CreditorRangeEnd {
			return "", fmt.Errorf("Kreditoren-Nummernkreis %d-%d ist erschöpft", CreditorRangeStart, CreditorRangeEnd)
		}
		return fmt.Sprintf("%d", n), nil
	default:
		return "", fmt.Errorf("unbekannter Kontakttyp %q", kind)
	}
}

// IsLedgerAccount reports whether an account number is a Personenkonto.
func IsLedgerAccount(account string) bool {
	if len(account) != 5 {
		return false
	}
	n := 0
	for _, r := range account {
		if r < '0' || r > '9' {
			return false
		}
		n = n*10 + int(r-'0')
	}
	return n >= DebitorRangeStart && n <= CreditorRangeEnd
}

// LedgerAccountKind classifies a Personenkonto as Debitor or Kreditor.
func LedgerAccountKind(account string) (ContactType, bool) {
	if !IsLedgerAccount(account) {
		return "", false
	}
	n := 0
	for _, r := range account {
		n = n*10 + int(r-'0')
	}
	if n <= DebitorRangeEnd {
		return ContactTypeCustomer, true
	}
	return ContactTypeVendor, true
}
