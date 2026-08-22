package einvoice

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// ErrNotAnAmount is returned when a field does not hold a readable decimal.
var ErrNotAnAmount = errors.New("kein lesbarer Betrag")

// Amount is a decimal number exactly as the document wrote it.
//
// The written form is kept rather than a parsed number, for three reasons that
// all cost correctness if ignored. EN 16931 limits most amounts to two decimals
// (the BR-DEC family) — a rule that can only be checked against what was
// written, not against a value already rounded on the way in. Item net prices
// (BT-146) and VAT rates (BT-119) legitimately carry more than two decimals, so
// a cent-based type cannot hold them. And an amount that is absent means
// something different from an amount that is zero: BR-CO-13 treats a missing
// allowance total as zero, while a missing invoice total is a violation.
type Amount struct {
	raw string
}

// NewAmount takes the decimal as written. Whitespace is trimmed; nothing else
// is normalised, because normalising would destroy what BR-DEC has to look at.
func NewAmount(raw string) Amount { return Amount{raw: strings.TrimSpace(raw)} }

// Present reports whether the document stated the amount at all.
func (a Amount) Present() bool { return a.raw != "" }

// String returns the amount as written, or the empty string if absent.
func (a Amount) String() string { return a.raw }

// Decimals counts the digits after the decimal mark, as written. An amount
// without a decimal mark has none; an absent amount has none either.
func (a Amount) Decimals() int {
	dot := strings.IndexByte(a.raw, '.')
	if dot < 0 {
		return 0
	}
	return len(a.raw) - dot - 1
}

// Rat returns the exact value. The second result is false if the amount is
// absent or unreadable.
func (a Amount) Rat() (*big.Rat, bool) {
	if a.raw == "" {
		return nil, false
	}
	r, ok := new(big.Rat).SetString(a.raw)
	if !ok {
		return nil, false
	}
	return r, true
}

// Cents returns the amount in hundredths.
//
// It refuses more than two decimals rather than rounding them away: an invoice
// total with three decimals is a document defect (BR-DEC), and silently
// rounding it here would hide the very thing the check is for.
func (a Amount) Cents() (Cents, error) {
	if a.raw == "" {
		return 0, ErrNotAnAmount
	}
	if a.Decimals() > 2 {
		return 0, fmt.Errorf("%w: %q hat mehr als zwei Nachkommastellen", ErrNotAnAmount, a.raw)
	}
	r, ok := a.Rat()
	if !ok {
		return 0, fmt.Errorf("%w: %q", ErrNotAnAmount, a.raw)
	}
	hundredths := new(big.Rat).Mul(r, big.NewRat(100, 1))
	if !hundredths.IsInt() {
		return 0, fmt.Errorf("%w: %q", ErrNotAnAmount, a.raw)
	}
	if !hundredths.Num().IsInt64() {
		return 0, fmt.Errorf("%w: %q ist zu groß", ErrNotAnAmount, a.raw)
	}
	return Cents(hundredths.Num().Int64()), nil
}

// CentsOr returns the amount in hundredths, or the fallback if it is absent.
// An unreadable amount is a defect and reported as such, not defaulted.
func (a Amount) CentsOr(fallback Cents) (Cents, error) {
	if !a.Present() {
		return fallback, nil
	}
	return a.Cents()
}

// IsZero reports whether the amount is present and equals zero. It is written
// as a value comparison, so "0", "0.00" and "-0.0" all count.
func (a Amount) IsZero() bool {
	r, ok := a.Rat()
	return ok && r.Sign() == 0
}

// Sign returns -1, 0 or +1, and 0 for an absent or unreadable amount.
func (a Amount) Sign() int {
	r, ok := a.Rat()
	if !ok {
		return 0
	}
	return r.Sign()
}

// Equal compares two amounts by value, not by spelling: "19.00" equals "19".
func (a Amount) Equal(b Amount) bool {
	ra, okA := a.Rat()
	rb, okB := b.Rat()
	if !okA || !okB {
		// Unreadable values fall back to the written form, so that two equally
		// broken fields still compare equal rather than silently differing.
		return a.raw == b.raw
	}
	return ra.Cmp(rb) == 0
}

// Cents is a monetary amount in hundredths of the invoice currency.
//
// Integer arithmetic is the point: every sum rule of EN 16931 is an exact
// equality, and a float would make a correct invoice fail one time in a
// thousand for reasons nobody could reproduce.
type Cents int64

// AmountFromCents renders a cent value as a two-decimal amount, which is what
// every EN 16931 syntax expects on the wire.
func AmountFromCents(c Cents) Amount {
	sign := ""
	if c < 0 {
		sign = "-"
		c = -c
	}
	return Amount{raw: fmt.Sprintf("%s%d.%02d", sign, int64(c)/100, int64(c)%100)}
}

// Abs returns the magnitude.
func (c Cents) Abs() Cents {
	if c < 0 {
		return -c
	}
	return c
}

// String formats the amount in German notation, e.g. "1.234,56".
func (c Cents) String() string {
	sign := ""
	if c < 0 {
		sign = "-"
		c = -c
	}
	whole := fmt.Sprintf("%d", int64(c)/100)
	var grouped strings.Builder
	for i, r := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			grouped.WriteByte('.')
		}
		grouped.WriteRune(r)
	}
	return fmt.Sprintf("%s%s,%02d", sign, grouped.String(), int64(c)%100)
}

// MulPercent applies a percentage to a base and rounds commercially.
//
// The percentage arrives as an Amount rather than a number because rates are
// not always whole: 8.375 % exists, and a rate parsed into hundredths would
// turn it into 8.38 % before the multiplication — a cent or two off on every
// line, which is exactly the difference that leaves an open item that never
// closes.
func MulPercent(base Cents, percent Amount) (Cents, bool) {
	rate, ok := percent.Rat()
	if !ok {
		return 0, false
	}
	product := new(big.Rat).Mul(new(big.Rat).SetInt64(int64(base)), rate)
	product.Quo(product, big.NewRat(100, 1))
	return roundRat(product), true
}

// roundRat rounds to the nearest integer, halves away from zero — kaufmännisches
// Runden, the same rule the rest of Buchfink uses.
func roundRat(r *big.Rat) Cents {
	neg := r.Sign() < 0
	if neg {
		r = new(big.Rat).Neg(r)
	}
	half := new(big.Rat).SetFrac64(1, 2)
	r = new(big.Rat).Add(r, half)
	q := new(big.Int).Quo(r.Num(), r.Denom())
	value := Cents(q.Int64())
	if neg {
		return -value
	}
	return value
}
